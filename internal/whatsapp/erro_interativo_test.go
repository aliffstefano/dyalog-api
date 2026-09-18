package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
)

func TestErroInterativoNaoPermitidoReconheceCodigosDeRecusa(t *testing.T) {
	casos := map[string]bool{
		"server returned error 405": true,
		"server returned error 473": true,
		"server returned error 479": true,
		"not-allowed":               true,
		// Erros que nao sao recusa de mensagem interativa nao devem acionar o
		// fallback para texto: sao falhas reais que precisam subir para o chamador.
		"server returned error 500": false,
		"context deadline exceeded": false,
		"connection reset by peer":  false,
	}
	for mensagem, esperado := range casos {
		if obtido := erroInterativoNaoPermitido(errors.New(mensagem)); obtido != esperado {
			t.Fatalf("erroInterativoNaoPermitido(%q) = %v, esperado %v", mensagem, obtido, esperado)
		}
	}
}

func TestErroInterativoNaoPermitidoComErroTipadoDoWhatsmeow(t *testing.T) {
	if !erroInterativoNaoPermitido(whatsmeow.ErrIQNotAllowed) {
		t.Fatal("ErrIQNotAllowed deveria ser reconhecido")
	}
}

func TestErroInterativoNaoPermitidoIgnoraNil(t *testing.T) {
	if erroInterativoNaoPermitido(nil) {
		t.Fatal("erro nil nao deveria acionar fallback")
	}
}
