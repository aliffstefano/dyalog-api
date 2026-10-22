package call

import (
	"dyalog-api-go/internal/voip/core"
	"dyalog-api-go/internal/voip/media"
	"dyalog-api-go/internal/voip/transport"
	"dyalog-api-go/internal/voip/wanode"
	"fmt"
	"time"
)

func (m *CallManager) initCodec() {
	if m.codec != nil {
		return
	}
	codec, err := media.NewMLowCodec(media.DefaultCodecOptions)
	if err != nil {
		m.log.Warn("MLow codec unavailable — call will run signaling-only (no audio)", "err", err)
		return
	}
	m.codec = codec
}

func (m *CallManager) FeedCapturedPCM(data []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.codec == nil || m.rtpSession == nil || m.srtpSession == nil || !m.relay.HasConnection() {
		return
	}
	m.lastCaptureAt = time.Now()
	frameSize := m.codec.FrameSize()
	if m.encodeBuf == nil {
		m.encodeBuf = make([]float32, frameSize)
		m.encodeBufPos = 0
	}

	offset := 0
	for offset < len(data) {
		toCopy := min(len(data)-offset, frameSize-m.encodeBufPos)
		copy(m.encodeBuf[m.encodeBufPos:], data[offset:offset+toCopy])
		m.encodeBufPos += toCopy
		offset += toCopy
		if m.encodeBufPos < frameSize {
			break
		}
		frame := make([]float32, frameSize)
		copy(frame, m.encodeBuf)
		m.encodeBufPos = 0

		opus, err := m.codec.Encode(frame)
		if err != nil {
			m.log.Debug("encode error", "err", err)
			continue
		}
		m.sendOpusFrameLocked(opus)
	}
}

func (m *CallManager) sendOpusFrameLocked(opus []byte) {
	if m.rtpSession == nil || m.srtpSession == nil {
		return
	}
	marker := !m.firstPacketSent
	pkt := m.rtpSession.CreatePacketWithDuration(opus, m.codec.FrameSize(), marker)
	if m.debeEnabled {
		pkt.Header.Extension = true
		pkt.Header.ExtensionProfile = 0xbede
		pkt.Header.ExtensionData = nil
	}
	m.firstPacketSent = true

	srtp, err := m.srtpSession.Protect(pkt)
	if err != nil {
		m.log.Debug("srtp protect error", "err", err)
		return
	}
	m.relay.Broadcast(srtp)
}

func (m *CallManager) startSilenceKeepaliveLocked() {
	if m.keepaliveStop != nil || m.codec == nil {
		return
	}
	stop := make(chan struct{})
	m.keepaliveStop = stop
	frameSize := m.codec.FrameSize()
	go func() {
		ticker := time.NewTicker(60 * time.Millisecond)
		defer ticker.Stop()
		silence := make([]float32, frameSize)
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				m.mu.Lock()
				ready := m.codec != nil && m.rtpSession != nil && m.srtpSession != nil && m.relay.HasConnection()
				idle := time.Since(m.lastCaptureAt) > 120*time.Millisecond
				if ready && idle {
					if opus, err := m.codec.Encode(silence); err == nil {
						m.sendOpusFrameLocked(opus)
					}
				}
				m.mu.Unlock()
			}
		}
	}()
}

// onRelayVideo descriptografa o video do peer (mesmas chaves do audio, contexto por
// SSRC) e entrega o RTP H.264 a quem estiver consumindo.
func (m *CallManager) onRelayVideo(data []byte) {
	m.mu.Lock()
	srtp := m.srtpSession
	m.mu.Unlock()
	if srtp == nil || m.OnPeerVideo == nil {
		return
	}
	pkt, err := srtp.UnprotectStream(data)
	if err != nil {
		if wanode.DebugXMLEnabled() && m.videoLogados.Add(1) <= 20 {
			m.log.Info("video srtp unprotect falhou", "err", err, "len", len(data))
		}
		return
	}
	if len(pkt.Payload) == 0 {
		return
	}
	// Confirma o empacotamento: RFC 6184 comeca com o header de NAL (tipo em &0x1f).
	if wanode.DebugXMLEnabled() && m.videoLogados.Add(1) <= 20 {
		n := min(len(pkt.Payload), 8)
		m.log.Info("video rtp descriptografado", "seq", pkt.Header.SequenceNumber, "ts", pkt.Header.Timestamp, "marker", pkt.Header.Marker, "payload_ini", fmt.Sprintf("%x", pkt.Payload[:n]))
	}
	m.OnPeerVideo(pkt)
}

func (m *CallManager) onRelayData(data []byte) {
	if transport.IsStunPacket(data) {
		return
	}
	if !transport.IsRtpPacket(data) {
		return
	}
	if len(data) < 12 {
		return
	}
	// RTCP (SR/SDES) passa pelo filtro de RTP; o PT dele nao e de midia.
	if transport.IsRtcpPacket(data) {
		return
	}
	pt := data[1] & 0x7f
	if pt == core.PayloadTypeWhatsAppVideoH264 {
		m.onRelayVideo(data)
		return
	}
	if pt != core.PayloadTypeWhatsAppOpus {
		// Diagnostico de video (VOIP_DEBUG_XML=1): payload type e ssrc do que o
		// relay entrega e nao e audio. So os primeiros 40 por chamada.
		if !wanode.DebugXMLEnabled() || m.naoAudioLogados.Add(1) > 40 {
			return
		}
		m.log.Info("rtp nao-audio descartado","pt", pt, "ssrc", uint32(data[8])<<24|uint32(data[9])<<16|uint32(data[10])<<8|uint32(data[11]), "len", len(data), "marker", data[1]&0x80 != 0)
		return
	}

	m.mu.Lock()
	if m.srtpSession == nil || m.codec == nil {
		m.mu.Unlock()
		return
	}
	ssrc := uint32(data[8])<<24 | uint32(data[9])<<16 | uint32(data[10])<<8 | uint32(data[11])
	if ssrc == m.selfSsrc {
		m.mu.Unlock()
		return
	}
	if !m.actualPeerSet {
		m.actualPeerSet = true
		if !containsSsrc(m.peerSsrcs, ssrc) {
			m.peerSsrcs = []uint32{ssrc}
			m.relay.SetSubscriptionSsrc(ssrc)
			go m.relay.ResendSubscriptions()
		}
	}
	srtp := m.srtpSession
	codec := m.codec
	m.mu.Unlock()

	pkt, err := srtp.Unprotect(data)
	if err != nil {
		m.log.Debug("srtp unprotect error", "err", err)
		return
	}
	if len(pkt.Payload) == 0 {
		return
	}
	pcm, err := codec.Decode(pkt.Payload)
	if err != nil {
		return
	}
	pcm = media.NormalizeFrame(pcm, codec.FrameSize())
	if m.OnPeerAudio != nil {
		m.OnPeerAudio(pcm)
	}
}
