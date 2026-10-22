package transport

import "testing"

func TestIsRtcpPacket(t *testing.T) {
	casos := []struct {
		nome string
		p    []byte
		want bool
	}{
		{"SR", []byte{0x80, 200, 0, 6}, true},
		{"SDES", []byte{0x81, 202, 0, 6}, true},
		{"opus", []byte{0x80, 120, 0, 1}, false},
		{"opus com marker", []byte{0x80, 0x80 | 120, 0, 1}, false},
		{"video H264", []byte{0x80, 97, 0, 1}, false},
		{"video com marker", []byte{0x80, 0x80 | 97, 0, 1}, false},
		{"stun", []byte{0x00, 0x01}, false},
	}
	for _, c := range casos {
		if got := IsRtcpPacket(c.p); got != c.want {
			t.Errorf("%s: got %v want %v", c.nome, got, c.want)
		}
	}
}
