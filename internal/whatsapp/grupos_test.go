package whatsapp

import (
	"errors"
	"testing"
)

func TestJIDGrupo(t *testing.T) {
	for entrada, esperado := range map[string]string{
		"120363025246125486@g.us": "120363025246125486@g.us",
		" 120363025246125486 ":    "120363025246125486@g.us",
	} {
		jid, err := JIDGrupo(entrada)
		if err != nil || jid.String() != esperado {
			t.Fatalf("%q -> %v, %v", entrada, jid, err)
		}
	}
	for _, invalido := range []string{"", "5567999999999@s.whatsapp.net"} {
		if _, err := JIDGrupo(invalido); !errors.Is(err, ErrGrupoInvalido) {
			t.Fatalf("%q deveria ser invalido: %v", invalido, err)
		}
	}
}
