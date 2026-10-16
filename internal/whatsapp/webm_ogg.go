package whatsapp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4/pkg/media/oggwriter"
)

// Audio gravado no navegador: o Chrome grava WebM/Opus, mas a nota de voz do
// WhatsApp (ptt) so toca Ogg/Opus. Os pacotes Opus sao os mesmos; so muda o
// "envelope". Aqui os pacotes saem do WebM e entram num Ogg, sem recodificar.

var errWebMSemOpus = errors.New("webm sem audio opus")

// IDs EBML/Matroska usados.
const (
	ebmlSegment      = 0x18538067
	ebmlCluster      = 0x1F43B675
	ebmlTracks       = 0x1654AE6B
	ebmlTrackEntry   = 0xAE
	ebmlCodecID      = 0x86
	ebmlCodecPrivate = 0x63A2
	ebmlSimpleBlock  = 0xA3
	ebmlBlockGroup   = 0xA0
	ebmlBlock        = 0xA1
)

// mestres sao os elementos com filhos que precisamos percorrer.
var mestresWebM = map[uint64]bool{ebmlSegment: true, ebmlCluster: true, ebmlTracks: true, ebmlTrackEntry: true, ebmlBlockGroup: true}

// lerVint le um inteiro de tamanho variavel EBML. marcador=true mantem o bit
// de tamanho (IDs); false remove (tamanhos).
func lerVint(dados []byte, marcador bool) (valor uint64, tamanho int, desconhecido bool, err error) {
	if len(dados) == 0 {
		return 0, 0, false, fmt.Errorf("webm truncado")
	}
	primeiro := dados[0]
	tamanho = 1
	for mascara := byte(0x80); tamanho <= 8 && primeiro&mascara == 0; mascara >>= 1 {
		tamanho++
	}
	if tamanho > 8 || len(dados) < tamanho {
		return 0, 0, false, fmt.Errorf("webm invalido")
	}
	valor = uint64(primeiro)
	if !marcador {
		valor &= uint64(0xFF >> tamanho)
	}
	todosUm := valor == uint64(0xFF>>tamanho)
	for i := 1; i < tamanho; i++ {
		valor = valor<<8 | uint64(dados[i])
		todosUm = todosUm && dados[i] == 0xFF
	}
	return valor, tamanho, !marcador && todosUm, nil
}

// pacotesOpusWebM devolve os pacotes Opus e o numero de canais.
func pacotesOpusWebM(dados []byte) ([][]byte, uint16, error) {
	var (
		pacotes     [][]byte
		canais      uint16 = 1
		opus        bool
		trilha      uint64 // numero da trilha de audio (0 = primeira que aparecer)
		trilhaAtual uint64
	)
	pos := 0
	for pos < len(dados) {
		id, tamID, _, err := lerVint(dados[pos:], true)
		if err != nil {
			return nil, 0, err
		}
		tam, tamTam, desconhecido, err := lerVint(dados[pos+tamID:], false)
		if err != nil {
			return nil, 0, err
		}
		inicio := pos + tamID + tamTam
		if mestresWebM[id] {
			pos = inicio // entra no elemento (tamanho desconhecido inclusive)
			continue
		}
		if desconhecido || inicio+int(tam) > len(dados) || int(tam) < 0 {
			// Gravacao cortada no fim: aproveita o que veio inteiro.
			break
		}
		corpo := dados[inicio : inicio+int(tam)]
		switch id {
		case 0xD7: // TrackNumber
			trilhaAtual = 0
			for _, b := range corpo {
				trilhaAtual = trilhaAtual<<8 | uint64(b)
			}
		case ebmlCodecID:
			if string(corpo) == "A_OPUS" {
				opus = true
				if trilha == 0 {
					trilha = trilhaAtual
				}
			}
		case ebmlCodecPrivate:
			// OpusHead: "OpusHead", versao, canais...
			if len(corpo) >= 10 && bytes.HasPrefix(corpo, []byte("OpusHead")) {
				canais = uint16(corpo[9])
			}
		case ebmlSimpleBlock, ebmlBlock:
			numero, tamNum, _, err := lerVint(corpo, false)
			if err != nil || len(corpo) < tamNum+3 {
				break
			}
			if trilha != 0 && numero != trilha {
				break
			}
			flags := corpo[tamNum+2]
			if flags&0x06 != 0 {
				return nil, 0, fmt.Errorf("webm com lacing nao suportado")
			}
			quadro := corpo[tamNum+3:]
			if len(quadro) > 0 {
				pacotes = append(pacotes, append([]byte(nil), quadro...))
			}
		}
		pos = inicio + int(tam)
	}
	if !opus || len(pacotes) == 0 {
		return nil, 0, errWebMSemOpus
	}
	if canais == 0 || canais > 2 {
		canais = 1
	}
	return pacotes, canais, nil
}

// WebMParaOgg converte audio WebM/Opus em Ogg/Opus.
func WebMParaOgg(dados []byte) ([]byte, error) {
	pacotes, canais, err := pacotesOpusWebM(dados)
	if err != nil {
		return nil, err
	}
	var saida bytes.Buffer
	escritor, err := oggwriter.NewWith(&saida, 48000, canais)
	if err != nil {
		return nil, err
	}
	for i, pacote := range pacotes {
		pkt := &rtp.Packet{Header: rtp.Header{SequenceNumber: uint16(i), Timestamp: uint32(i) * 960}, Payload: pacote}
		if err := escritor.WriteRTP(pkt); err != nil {
			return nil, fmt.Errorf("erro ao montar ogg: %w", err)
		}
	}
	if err := escritor.Close(); err != nil {
		return nil, err
	}
	return saida.Bytes(), nil
}

// ehWebM olha a assinatura EBML.
func ehWebM(dados []byte) bool {
	return len(dados) > 4 && binary.BigEndian.Uint32(dados[:4]) == 0x1A45DFA3
}
