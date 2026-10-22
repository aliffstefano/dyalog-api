package wanode

import (
	"strings"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
)

func TestDumpNodeRedactsSecrets(t *testing.T) {
	n := waBinary.Node{Tag: "offer", Attrs: waBinary.Attrs{"call-id": "ABC"}, Content: []waBinary.Node{
		{Tag: "relay", Attrs: waBinary.Attrs{"ip": "1.2.3.4", "token": "segredo"}},
		{Tag: "enc", Content: []byte{1, 2, 3}},
	}}
	got := DumpNode(&n)
	if strings.Contains(got, "segredo") || strings.Contains(got, "\x01") {
		t.Fatalf("vazou conteudo sensivel: %s", got)
	}
	for _, want := range []string{`call-id="ABC"`, `ip="1.2.3.4"`, "[3 bytes]", `token="<redacted>"`} {
		if !strings.Contains(got, want) {
			t.Fatalf("faltou %q em %s", want, got)
		}
	}
}
