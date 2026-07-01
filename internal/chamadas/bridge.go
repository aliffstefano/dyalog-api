package chamadas

import (
	"io"
	"log/slog"
	"math"
	"sync/atomic"

	"dyalog-api-go/internal/voip/media"

	"github.com/pion/opus"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

const canalPCM = "pcm"
const (
	pcmuSampleRate       = 8000
	pcmuSamplesPorPacote = 160 // 20 ms
)

type Bridge struct {
	pc         *webrtc.PeerConnection
	dc         atomic.Pointer[webrtc.DataChannel]
	audioTrack *webrtc.TrackLocalStaticRTP
	log        *slog.Logger

	pcmuSeq       uint16
	pcmuTimestamp uint32
	pcmuBuffer    []byte

	OnBrowserPCM  func(pcm []float32)
	OnTerminalICE func()
}

func NovoBridge(sdpOferta string, log *slog.Logger) (*Bridge, string, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		return nil, "", err
	}
	br := &Bridge{pc: pc, log: log}

	audioTrack, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
		MimeType:  webrtc.MimeTypePCMU,
		ClockRate: pcmuSampleRate,
		Channels:  1,
	}, "dyalog-audio", "dyalog-call")
	if err != nil {
		_ = pc.Close()
		return nil, "", err
	}
	br.audioTrack = audioTrack
	if _, err := pc.AddTrack(audioTrack); err != nil {
		_ = pc.Close()
		return nil, "", err
	}

	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		if track.Kind() != webrtc.RTPCodecTypeAudio {
			return
		}
		go br.consumirTrackAudio(track)
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		if dc.Label() != canalPCM {
			return
		}
		br.dc.Store(dc)
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			if br.OnBrowserPCM != nil && len(msg.Data) > 0 {
				br.OnBrowserPCM(media.PCMInt16LEToFloat32(msg.Data))
			}
		})
	})

	pc.OnICEConnectionStateChange(func(estado webrtc.ICEConnectionState) {
		if log != nil {
			log.Debug("estado ice da chamada", "estado", estado.String())
		}
		if estado == webrtc.ICEConnectionStateFailed || estado == webrtc.ICEConnectionStateClosed {
			if br.OnTerminalICE != nil {
				br.OnTerminalICE()
			}
		}
	})

	if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdpOferta}); err != nil {
		_ = pc.Close()
		return nil, "", err
	}
	resposta, err := pc.CreateAnswer(nil)
	if err != nil {
		_ = pc.Close()
		return nil, "", err
	}
	coletaCompleta := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(resposta); err != nil {
		_ = pc.Close()
		return nil, "", err
	}
	<-coletaCompleta

	return br, pc.LocalDescription().SDP, nil
}

func (b *Bridge) WritePCM(pcm []float32) error {
	if b.audioTrack != nil && len(pcm) > 0 {
		if err := b.writePCMComoPCMU(pcm); err != nil && b.log != nil {
			b.log.Debug("erro ao escrever audio track pcmu", "err", err)
		}
	}
	dc := b.dc.Load()
	if dc == nil || len(pcm) == 0 {
		return nil
	}
	return dc.Send(media.PCMFloat32ToInt16LE(pcm))
}

func (b *Bridge) Close() {
	if b.pc != nil {
		_ = b.pc.Close()
	}
}

func (b *Bridge) consumirTrackAudio(track *webrtc.TrackRemote) {
	switch track.Codec().MimeType {
	case webrtc.MimeTypeOpus:
		b.consumirOpus(track)
	case webrtc.MimeTypePCMU:
		b.consumirPCMU(track)
	default:
		if b.log != nil {
			b.log.Debug("codec de audio do browser nao suportado", "mime", track.Codec().MimeType)
		}
	}
}

func (b *Bridge) consumirOpus(track *webrtc.TrackRemote) {
	decoder, err := opus.NewDecoderWithOutput(16000, 1)
	if err != nil {
		if b.log != nil {
			b.log.Debug("erro ao criar decoder opus", "err", err)
		}
		return
	}
	buf := make([]float32, 1920)
	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF && b.log != nil {
				b.log.Debug("erro lendo rtp opus", "err", err)
			}
			return
		}
		n, err := decoder.DecodeToFloat32(pkt.Payload, buf)
		if err != nil || n == 0 {
			continue
		}
		if b.OnBrowserPCM != nil {
			frame := make([]float32, n)
			copy(frame, buf[:n])
			b.OnBrowserPCM(frame)
		}
	}
}

func (b *Bridge) consumirPCMU(track *webrtc.TrackRemote) {
	for {
		pkt, _, err := track.ReadRTP()
		if err != nil {
			if err != io.EOF && b.log != nil {
				b.log.Debug("erro lendo rtp pcmu", "err", err)
			}
			return
		}
		if len(pkt.Payload) == 0 || b.OnBrowserPCM == nil {
			continue
		}
		pcm8k := make([]float32, len(pkt.Payload))
		for i, sample := range pkt.Payload {
			pcm8k[i] = muLawDecode(sample)
		}
		b.OnBrowserPCM(resampleLinear(pcm8k, pcmuSampleRate, 16000))
	}
}

func (b *Bridge) writePCMComoPCMU(pcm16k []float32) error {
	pcm8k := resampleLinear(pcm16k, 16000, pcmuSampleRate)
	for _, sample := range pcm8k {
		b.pcmuBuffer = append(b.pcmuBuffer, muLawEncode(sample))
	}
	for len(b.pcmuBuffer) >= pcmuSamplesPorPacote {
		payload := make([]byte, pcmuSamplesPorPacote)
		copy(payload, b.pcmuBuffer[:pcmuSamplesPorPacote])
		b.pcmuBuffer = b.pcmuBuffer[pcmuSamplesPorPacote:]
		pkt := &rtp.Packet{
			Header: rtp.Header{
				Version:        2,
				PayloadType:    0,
				SequenceNumber: b.pcmuSeq,
				Timestamp:      b.pcmuTimestamp,
			},
			Payload: payload,
		}
		b.pcmuSeq++
		b.pcmuTimestamp += pcmuSamplesPorPacote
		if err := b.audioTrack.WriteRTP(pkt); err != nil {
			return err
		}
	}
	return nil
}

func resampleLinear(in []float32, fromRate, toRate int) []float32 {
	if len(in) == 0 || fromRate <= 0 || toRate <= 0 {
		return nil
	}
	if fromRate == toRate {
		out := make([]float32, len(in))
		copy(out, in)
		return out
	}
	outLen := int(math.Round(float64(len(in)) * float64(toRate) / float64(fromRate)))
	if outLen <= 0 {
		return nil
	}
	out := make([]float32, outLen)
	ratio := float64(fromRate) / float64(toRate)
	for i := range out {
		pos := float64(i) * ratio
		idx := int(pos)
		frac := float32(pos - float64(idx))
		if idx >= len(in)-1 {
			out[i] = in[len(in)-1]
			continue
		}
		out[i] = in[idx]*(1-frac) + in[idx+1]*frac
	}
	return out
}

func muLawEncode(sample float32) byte {
	if sample > 1 {
		sample = 1
	} else if sample < -1 {
		sample = -1
	}
	pcm := int(sample * 32767)
	const bias = 0x84
	sign := 0
	if pcm < 0 {
		sign = 0x80
		pcm = -pcm
	}
	if pcm > 32635 {
		pcm = 32635
	}
	pcm += bias
	exponent := 7
	for mask := 0x4000; exponent > 0 && (pcm&mask) == 0; exponent-- {
		mask >>= 1
	}
	mantissa := (pcm >> (exponent + 3)) & 0x0f
	return byte(^(sign | (exponent << 4) | mantissa))
}

func muLawDecode(sample byte) float32 {
	u := ^sample
	sign := u & 0x80
	exponent := (u >> 4) & 0x07
	mantissa := u & 0x0f
	pcm := ((int(mantissa) << 3) + 0x84) << exponent
	pcm -= 0x84
	if sign != 0 {
		pcm = -pcm
	}
	return float32(pcm) / 32768.0
}
