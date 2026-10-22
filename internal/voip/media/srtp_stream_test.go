package media

import (
	"bytes"
	"testing"

	"dyalog-api-go/internal/voip/core"
)

// Dois SSRCs com numeracao de sequencia diferente (audio e video) no mesmo
// SrtpSession: cada um precisa do seu proprio ROC/contexto.
func TestUnprotectStreamPorSSRC(t *testing.T) {
	km := core.SrtpKeyingMaterial{MasterKey: bytes.Repeat([]byte{1}, 16), MasterSalt: bytes.Repeat([]byte{2}, 14)}
	txAudio, err := NewSrtpSession(km, km, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	txVideo, _ := NewSrtpSession(km, km, 4, 4)
	rx, err := NewSrtpSession(km, km, 4, 4)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		a := &RtpPacket{Header: NewRtpHeader(120, uint16(65530+i*3), 1, 111), Payload: []byte("audio")}
		v := &RtpPacket{Header: NewRtpHeader(97, uint16(100+i), 2, 222), Payload: []byte("video")}
		pa, _ := txAudio.Protect(a)
		pv, _ := txVideo.Protect(v)
		if got, err := rx.UnprotectStream(pa); err != nil || string(got.Payload) != "audio" {
			t.Fatalf("audio %d: %v", i, err)
		}
		if got, err := rx.UnprotectStream(pv); err != nil || string(got.Payload) != "video" {
			t.Fatalf("video %d: %v", i, err)
		}
	}
}
