package http

import (
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLimitadorBloqueiaDepoisDoLimite(t *testing.T) {
	l := novoLimitador(3)
	for i := 0; i < 3; i++ {
		if ok, _ := l.consumir("inst"); !ok {
			t.Fatalf("envio %d deveria passar", i+1)
		}
	}
	ok, espera := l.consumir("inst")
	if ok || espera <= 0 || espera > 25e9 {
		t.Fatalf("4o envio deveria ser barrado com espera ate ~20s: ok=%v espera=%v", ok, espera)
	}
	// Outra chave tem o proprio balde.
	if ok, _ := l.consumir("outra"); !ok {
		t.Fatal("outra instancia nao deveria ser afetada")
	}
}

func TestLimitadorBloqueadoNaoGastaFicha(t *testing.T) {
	l := novoLimitador(2)
	for i := 0; i < 5; i++ {
		if bloqueado, _ := l.bloqueado("ip"); bloqueado {
			t.Fatal("consultar nao deveria gastar ficha")
		}
	}
	l.consumir("ip")
	l.consumir("ip")
	if bloqueado, espera := l.bloqueado("ip"); !bloqueado || espera <= 0 {
		t.Fatalf("deveria estar bloqueado: espera=%v", espera)
	}
}

func TestLimitadorDesligado(t *testing.T) {
	var l *limitador = novoLimitador(0)
	if ok, _ := l.consumir("x"); !ok {
		t.Fatal("limite 0 deveria desligar")
	}
	if bloqueado, _ := l.bloqueado("x"); bloqueado {
		t.Fatal("limite 0 deveria desligar")
	}
}

func TestResponderLimiteDevolve429ComRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gravador := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(gravador)
	responderLimite(c, "limite_envios", "Limite atingido", 1500e6)
	if gravador.Code != nethttp.StatusTooManyRequests || gravador.Header().Get("Retry-After") != "2" {
		t.Fatalf("status=%d retry-after=%q", gravador.Code, gravador.Header().Get("Retry-After"))
	}
}
