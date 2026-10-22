package chamadas

import (
	"dyalog-api-go/internal/voip/media"

	"github.com/pion/rtp"
)

// WriteVideoRTP repassa o RTP H.264 do WhatsApp para a track do navegador, sem
// transcodificar. PT e SSRC sao reescritos pela propria track; sequencia e
// timestamp seguem os do WhatsApp (continuos por SSRC).
func (b *Bridge) WriteVideoRTP(pkt *media.RtpPacket) {
	if b.videoTrack == nil || pkt == nil || pkt.Header == nil {
		return
	}
	b.videoMu.Lock()
	if b.aguardandoIDR {
		if !h264IniciaKeyframe(pkt.Payload) {
			b.videoMu.Unlock()
			return
		}
		b.aguardandoIDR = false
	}
	b.videoMu.Unlock()

	err := b.videoTrack.WriteRTP(&rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			Marker:         pkt.Header.Marker,
			SequenceNumber: pkt.Header.SequenceNumber,
			Timestamp:      pkt.Header.Timestamp,
		},
		Payload: pkt.Payload,
	})
	if err != nil && b.log != nil {
		b.log.Debug("erro ao escrever video track", "err", err)
	}
}

// h264IniciaKeyframe diz se o payload RTP (RFC 6184) abre um keyframe: SPS (7) ou
// IDR (5), solto, dentro de STAP-A (24) ou no inicio de um FU-A (28).
func h264IniciaKeyframe(p []byte) bool {
	if len(p) == 0 {
		return false
	}
	ehKey := func(t byte) bool { return t == 5 || t == 7 }
	switch t := p[0] & 0x1f; {
	case t == 24:
		for i := 1; i+2 < len(p); {
			n := int(p[i])<<8 | int(p[i+1])
			i += 2
			if i < len(p) && ehKey(p[i]&0x1f) {
				return true
			}
			i += n
		}
		return false
	case t == 28:
		return len(p) > 1 && p[1]&0x80 != 0 && ehKey(p[1]&0x1f)
	default:
		return ehKey(t)
	}
}
