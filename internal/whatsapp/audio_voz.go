package whatsapp

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"sync"
	"time"
)

// Nota de voz (ptt) no WhatsApp precisa ser Ogg com Opus de verdade. O
// formato declarado (Content-Type da URL, nome do arquivo ou mime_type) nao e
// confiavel: TTS e automacoes mandam MP3/M4A/Ogg Vorbis rotulado como ogg. O
// Android costuma tocar mesmo assim; o iPhone nao abre e mostra "Este audio
// nao esta mais disponivel". Por isso o formato e identificado pelo conteudo
// e, quando nao for Ogg/Opus, convertido com ffmpeg para o padrao das notas
// de voz (Opus mono 48 kHz).

// Formatos identificados pelo conteudo.
const (
	audioOggOpus   = "ogg_opus"
	audioOggVorbis = "ogg_vorbis"
	audioWebM      = "webm"
	audioMP3       = "mp3"
	audioMP4       = "mp4"
	audioWAV       = "wav"
	audioAAC       = "aac"
	audioAMR       = "amr"
	audioFLAC      = "flac"
	audioOutro     = ""
)

// mimeAudioReal e o mime correto de cada formato identificado.
var mimeAudioReal = map[string]string{
	audioOggOpus:   "audio/ogg; codecs=opus",
	audioOggVorbis: "audio/ogg",
	audioMP3:       "audio/mpeg",
	audioMP4:       "audio/mp4",
	audioWAV:       "audio/wav",
	audioAAC:       "audio/aac",
	audioAMR:       "audio/amr",
	audioFLAC:      "audio/flac",
}

// identificarAudio olha os primeiros bytes do arquivo.
func identificarAudio(dados []byte) string {
	switch {
	case len(dados) < 12:
		return audioOutro
	case bytes.HasPrefix(dados, []byte("OggS")):
		inicio := dados[:min(len(dados), 512)]
		switch {
		case bytes.Contains(inicio, []byte("OpusHead")):
			return audioOggOpus
		case bytes.Contains(inicio, []byte("\x01vorbis")):
			return audioOggVorbis
		}
		return audioOutro
	case ehWebM(dados):
		return audioWebM
	case bytes.HasPrefix(dados, []byte("ID3")), dados[0] == 0xFF && dados[1]&0xE0 == 0xE0 && dados[1]&0x06 != 0:
		return audioMP3
	case dados[0] == 0xFF && dados[1]&0xF6 == 0xF0: // ADTS (AAC solto)
		return audioAAC
	case string(dados[4:8]) == "ftyp":
		return audioMP4
	case string(dados[:4]) == "RIFF" && string(dados[8:12]) == "WAVE":
		return audioWAV
	case bytes.HasPrefix(dados, []byte("#!AMR")):
		return audioAMR
	case bytes.HasPrefix(dados, []byte("fLaC")):
		return audioFLAC
	}
	return audioOutro
}

var (
	ffmpegUmaVez  sync.Once
	ffmpegCaminho string
)

// caminhoFFmpeg devolve o ffmpeg do sistema ("" se nao houver). A imagem
// Docker ja traz; rodando fora dela sem ffmpeg, a conversao fica desligada.
func caminhoFFmpeg() string {
	ffmpegUmaVez.Do(func() {
		if caminho, err := exec.LookPath("ffmpeg"); err == nil {
			ffmpegCaminho = caminho
		}
	})
	return ffmpegCaminho
}

var errSemFFmpeg = errors.New("ffmpeg nao instalado")

func rodarFFmpeg(ctx context.Context, entrada []byte, args ...string) ([]byte, error) {
	caminho := caminhoFFmpeg()
	if caminho == "" {
		return nil, errSemFFmpeg
	}
	ctx, cancelar := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelar()
	cmd := exec.CommandContext(ctx, caminho, append([]string{"-hide_banner", "-loglevel", "error", "-i", "pipe:0"}, args...)...)
	cmd.Stdin = bytes.NewReader(entrada)
	var saida, erro bytes.Buffer
	cmd.Stdout, cmd.Stderr = &saida, &erro
	if err := cmd.Run(); err != nil {
		mensagem := bytes.TrimSpace(erro.Bytes())
		if len(mensagem) > 300 {
			mensagem = mensagem[:300]
		}
		return nil, fmt.Errorf("ffmpeg: %v: %s", err, mensagem)
	}
	return saida.Bytes(), nil
}

// converterParaNotaDeVoz converte qualquer audio em Ogg/Opus mono 48 kHz,
// como as notas de voz gravadas no app.
func converterParaNotaDeVoz(ctx context.Context, dados []byte) ([]byte, error) {
	ogg, err := rodarFFmpeg(ctx, dados, "-vn", "-map_metadata", "-1", "-ac", "1", "-ar", "48000",
		"-c:a", "libopus", "-b:a", "32k", "-vbr", "on", "-application", "voip", "-frame_duration", "20",
		"-f", "ogg", "pipe:1")
	if err != nil {
		return nil, err
	}
	if identificarAudio(ogg) != audioOggOpus {
		return nil, fmt.Errorf("ffmpeg nao gerou ogg/opus")
	}
	return ogg, nil
}

// waveformReal decodifica o audio e gera as 64 barras (0 a 100) que o
// WhatsApp desenha na nota de voz, mais a duracao em segundos. Sem ffmpeg,
// devolve nil e 0.
func waveformReal(ctx context.Context, dados []byte) ([]byte, uint32) {
	const taxa = 8000
	pcm, err := rodarFFmpeg(ctx, dados, "-vn", "-ac", "1", "-ar", fmt.Sprint(taxa), "-f", "s16le", "pipe:1")
	if err != nil || len(pcm) < 2 {
		return nil, 0
	}
	amostras := len(pcm) / 2
	duracao := uint32(math.Round(float64(amostras) / taxa))
	const barras = 64
	rms := make([]float64, barras)
	maior := 0.0
	por := max(1, amostras/barras)
	for i := 0; i < barras; i++ {
		inicio, fim := i*por, min((i+1)*por, amostras)
		if inicio >= fim {
			break
		}
		var soma float64
		for j := inicio; j < fim; j++ {
			v := float64(int16(binary.LittleEndian.Uint16(pcm[j*2:]))) / 32768
			soma += v * v
		}
		rms[i] = math.Sqrt(soma / float64(fim-inicio))
		maior = math.Max(maior, rms[i])
	}
	onda := make([]byte, barras)
	for i, v := range rms {
		if maior > 0 {
			onda[i] = byte(math.Round(v / maior * 100))
		}
	}
	return onda, max(duracao, 1)
}
