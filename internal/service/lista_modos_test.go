package service

import "testing"

func TestModoListaValido(t *testing.T) {
	for _, modo := range []string{"", "auto", "lista_documento", "variacoes", "lista_documento_single", "lista_documento_sem_biz",
		"lista_documento_sem_bot", "lista_bot", "lista_bot_single", "native_flow_bot", "native_flow_documento", "lista_biz", "texto"} {
		if !modoListaValido(modo) {
			t.Errorf("modo %q deveria ser aceito", modo)
		}
	}
	if modoListaValido("qualquer") {
		t.Error("modo desconhecido aceito")
	}
}
