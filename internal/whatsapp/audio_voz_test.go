package whatsapp

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"dyalog-api-go/internal/models"
)

// gerarAudio cria um tom de 2 s no formato pedido (precisa de ffmpeg).
func gerarAudio(t *testing.T, formato string, args ...string) []byte {
	t.Helper()
	if caminhoFFmpeg() == "" {
		t.Skip("ffmpeg nao instalado")
	}
	saida := filepath.Join(t.TempDir(), "audio."+formato)
	cmd := exec.Command(caminhoFFmpeg(), append(append([]string{"-hide_banner", "-loglevel", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2"}, args...), saida)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("gerar %s: %v %s", formato, err, out)
	}
	dados, err := os.ReadFile(saida)
	if err != nil {
		t.Fatal(err)
	}
	return dados
}

func TestIdentificarAudio(t *testing.T) {
	casos := map[string][]byte{
		audioMP3:       gerarAudio(t, "mp3"),
		audioMP4:       gerarAudio(t, "m4a", "-c:a", "aac"),
		audioWAV:       gerarAudio(t, "wav"),
		audioOggOpus:   gerarAudio(t, "ogg", "-c:a", "libopus"),
		audioOggVorbis: gerarAudio(t, "ogg", "-c:a", "libvorbis"),
	}
	for esperado, dados := range casos {
		if got := identificarAudio(dados); got != esperado {
			t.Errorf("esperado %s, veio %q", esperado, got)
		}
	}
}

// O caso do iPhone: MP3 (ou Vorbis) rotulado como ogg e pedido como nota de
// voz precisa sair como Ogg/Opus de verdade.
func TestAudioRotuladoErradoViraOpus(t *testing.T) {
	sim := true
	for nome, dados := range map[string][]byte{
		"mp3 dizendo ser ogg": gerarAudio(t, "mp3"),
		"ogg vorbis":          gerarAudio(t, "ogg", "-c:a", "libvorbis"),
		"m4a":                 gerarAudio(t, "m4a", "-c:a", "aac"),
	} {
		req := models.EnvioMidiaRequest{ArquivoBase64: base64.StdEncoding.EncodeToString(dados), MimeType: "audio/ogg", NomeArquivo: "voz.ogg", PTT: &sim}
		saida, mime, err := carregarAudioEnvio(context.Background(), req)
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if mime != "audio/ogg; codecs=opus" || identificarAudio(saida) != audioOggOpus {
			t.Fatalf("%s: saiu %s / %s", nome, mime, identificarAudio(saida))
		}
		onda, segundos := waveformReal(context.Background(), saida)
		if len(onda) != 64 || segundos != 2 {
			t.Fatalf("%s: onda=%d segundos=%d", nome, len(onda), segundos)
		}
	}
}

// Sem pedir nota de voz, MP3 continua MP3 (arquivo de audio), com o mime certo
// mesmo se declarado errado.
func TestMP3SemPTTMantemFormato(t *testing.T) {
	dados := gerarAudio(t, "mp3")
	req := models.EnvioMidiaRequest{ArquivoBase64: base64.StdEncoding.EncodeToString(dados), MimeType: "audio/wav"}
	saida, mime, err := carregarAudioEnvio(context.Background(), req)
	if err != nil || mime != "audio/mpeg" || len(saida) != len(dados) {
		t.Fatalf("mime=%s err=%v", mime, err)
	}
}
