package chamadas

import "testing"

func TestH264IniciaKeyframe(t *testing.T) {
	casos := []struct {
		nome string
		p    []byte
		want bool
	}{
		{"vazio", nil, false},
		{"slice P", []byte{0x41, 1, 2}, false},
		{"IDR", []byte{0x65, 1, 2}, true},
		{"SPS", []byte{0x67, 1, 2}, true},
		{"STAP-A com SPS", []byte{0x18, 0, 2, 0x67, 1, 0, 1, 0x68}, true},
		{"STAP-A so com P", []byte{0x18, 0, 2, 0x41, 1}, false},
		{"FU-A inicio de IDR", []byte{0x7c, 0x85, 1}, true},
		{"FU-A meio de IDR", []byte{0x7c, 0x05, 1}, false},
		{"FU-A inicio de P", []byte{0x5c, 0x81, 1}, false},
	}
	for _, c := range casos {
		if got := h264IniciaKeyframe(c.p); got != c.want {
			t.Errorf("%s: got %v want %v", c.nome, got, c.want)
		}
	}
}
