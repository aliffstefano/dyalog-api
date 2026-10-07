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

// Cliente esquecido repetindo o mesmo token errado nao bloqueia o IP; tokens
// diferentes (tentativa de adivinhar) bloqueiam.
func TestRegistrarFalhaContaSoTokensDiferentes(t *testing.T) {
	l := novoLimitador(3)
	for i := 0; i < 50; i++ {
		l.registrarFalha("1.1.1.1", "token-antigo")
		l.registrarFalha("1.1.1.1", "")
	}
	if bloqueado, _ := l.bloqueado("1.1.1.1"); bloqueado {
		t.Fatal("o mesmo token repetido nao deveria bloquear")
	}
	for _, token := range []string{"a", "b", "c"} {
		l.registrarFalha("2.2.2.2", token)
	}
	if bloqueado, _ := l.bloqueado("2.2.2.2"); !bloqueado {
		t.Fatal("3 tokens diferentes deveriam bloquear com limite 3")
	}
}

func TestIpClienteCloudflare(t *testing.T) {
	gin.SetMode(gin.TestMode)
	motor := gin.New()
	if err := motor.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nome, remoto, xff, cf, esperado string
	}{
		{"via cloudflare e traefik", "10.0.0.5:1234", "172.70.1.1", "200.1.2.3", "200.1.2.3"},
		{"direto no servidor forjando cabecalho", "10.0.0.5:1234", "45.6.7.8", "200.1.2.3", "45.6.7.8"},
		{"sem cloudflare", "10.0.0.5:1234", "45.6.7.8", "", "45.6.7.8"},
	}
	for _, caso := range casos {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(nethttp.MethodGet, "/", nil)
		c.Request.RemoteAddr = caso.remoto
		c.Request.Header.Set("X-Forwarded-For", caso.xff)
		if caso.cf != "" {
			c.Request.Header.Set("CF-Connecting-IP", caso.cf)
		}
		motor.HandleContext(c)
		if ip := ipCliente(c); ip != caso.esperado {
			t.Errorf("%s: ip = %s, esperado %s", caso.nome, ip, caso.esperado)
		}
	}
}
