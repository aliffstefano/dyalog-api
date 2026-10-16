package whatsapp

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// gravacao_chrome.webm: 1,5 s gravados pelo MediaRecorder do Chromium
// (audio/webm;codecs=opus), o mesmo formato do botao de microfone do chat.
func TestWebMParaOggGravacaoDoChrome(t *testing.T) {
	webm, err := os.ReadFile("testdata/gravacao_chrome.webm")
	if err != nil {
		t.Fatal(err)
	}
	if !ehWebM(webm) {
		t.Fatal("amostra nao reconhecida como webm")
	}
	pacotes, canais, err := pacotesOpusWebM(webm)
	if err != nil {
		t.Fatal(err)
	}
	// O Chrome usa quadros de 20 a 60 ms: 1,5 s da 25 a 75 pacotes.
	if len(pacotes) < 20 || len(pacotes) > 90 || canais < 1 {
		t.Fatalf("pacotes = %d canais = %d", len(pacotes), canais)
	}
	ogg, err := WebMParaOgg(webm)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(ogg, []byte("OggS")) || !bytes.Contains(ogg, []byte("OpusHead")) || !bytes.Contains(ogg, []byte("OpusTags")) {
		t.Fatal("saida nao e ogg/opus")
	}
	// Granule da ultima pagina = amostras a 48 kHz: deve dar ~1,5 s.
	ultima := bytes.LastIndex(ogg, []byte("OggS"))
	granule := binary.LittleEndian.Uint64(ogg[ultima+6 : ultima+14])
	if segundos := float64(granule) / 48000; segundos < 1.2 || segundos > 1.8 {
		t.Fatalf("duracao = %.2fs", segundos)
	}
	if mime := detectarMimeType("", "", ogg); mime != "audio/ogg" && mime != "application/ogg" {
		t.Fatalf("mime detectado = %q", mime)
	}
}

func TestWebMParaOggRecusaLixo(t *testing.T) {
	if _, err := WebMParaOgg([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x80}); err == nil {
		t.Fatal("deveria falhar sem audio")
	}
}
