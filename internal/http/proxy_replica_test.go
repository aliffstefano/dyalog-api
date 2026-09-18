package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dyalog-api-go/internal/models"

	"github.com/gin-gonic/gin"
)

func novoContextoJSON(corpo string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batepapo/enviar/texto", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	return c
}

func TestInstanciaDoCorpoPreservaCorpoIntegral(t *testing.T) {
	corpo := `{"instancia":"abc-123","numero":"5511999999999","mensagem":"ola"}`
	c := novoContextoJSON(corpo)

	if id := instanciaDoCorpo(c); id != "abc-123" {
		t.Fatalf("instanciaDoCorpo = %q, esperado abc-123", id)
	}

	restante, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("erro ao reler corpo: %v", err)
	}
	if string(restante) != corpo {
		t.Fatalf("corpo apos inspecao = %q, esperado %q", string(restante), corpo)
	}
}

func TestInstanciaDoCorpoSemCampoInstancia(t *testing.T) {
	corpo := `{"numero":"5511999999999","mensagem":"ola"}`
	c := novoContextoJSON(corpo)

	if id := instanciaDoCorpo(c); id != "" {
		t.Fatalf("instanciaDoCorpo = %q, esperado vazio", id)
	}

	restante, _ := io.ReadAll(c.Request.Body)
	if string(restante) != corpo {
		t.Fatalf("corpo apos inspecao = %q, esperado %q", string(restante), corpo)
	}
}

func TestInstanciaDoCorpoIgnoraNaoJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	corpo := "instancia=abc-123"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batepapo/enviar/texto", strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	if id := instanciaDoCorpo(c); id != "" {
		t.Fatalf("instanciaDoCorpo = %q, esperado vazio para corpo nao-JSON", id)
	}

	restante, _ := io.ReadAll(c.Request.Body)
	if string(restante) != corpo {
		t.Fatalf("corpo nao-JSON foi alterado: %q", string(restante))
	}
}

func TestResolverInstanciaAlvoPrioridades(t *testing.T) {
	t.Run("param da rota tem prioridade", func(t *testing.T) {
		c := novoContextoJSON(`{"instancia":"do-corpo"}`)
		c.Params = gin.Params{{Key: "id", Value: "do-param"}}
		if id := resolverInstanciaAlvo(c); id != "do-param" {
			t.Fatalf("resolverInstanciaAlvo = %q, esperado do-param", id)
		}
	})

	t.Run("corpo vem antes do token", func(t *testing.T) {
		c := novoContextoJSON(`{"instancia":"do-corpo"}`)
		c.Set(contextoAcessoDashboard, models.AcessoDashboard{Tipo: "instancia", InstanciaID: "do-token"})
		if id := resolverInstanciaAlvo(c); id != "do-corpo" {
			t.Fatalf("resolverInstanciaAlvo = %q, esperado do-corpo", id)
		}
	})

	t.Run("token de instancia como fallback", func(t *testing.T) {
		c := novoContextoJSON(`{"numero":"5511999999999"}`)
		c.Set(contextoAcessoDashboard, models.AcessoDashboard{Tipo: "instancia", InstanciaID: "do-token"})
		if id := resolverInstanciaAlvo(c); id != "do-token" {
			t.Fatalf("resolverInstanciaAlvo = %q, esperado do-token", id)
		}
	})

	t.Run("token master sem instancia no corpo nao resolve", func(t *testing.T) {
		c := novoContextoJSON(`{"numero":"5511999999999"}`)
		c.Set(contextoAcessoDashboard, models.AcessoDashboard{Tipo: "master"})
		if id := resolverInstanciaAlvo(c); id != "" {
			t.Fatalf("resolverInstanciaAlvo = %q, esperado vazio", id)
		}
	})
}

// Garante a trava anti-laco: uma requisicao ja encaminhada por outra replica nao
// pode ser encaminhada de novo, senao dois containers ficariam se empurrando a
// requisicao indefinidamente.
func TestMiddlewareProxyNaoReencaminha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gravador := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(gravador)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/batepapo/enviar/texto", strings.NewReader(`{"instancia":"abc"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(cabecalhoEncaminhado, "1")
	c.Request = req

	// gerenciador nil tambem exercita o caminho de "sem encaminhamento".
	middleware := middlewareProxyReplica(nil)
	middleware(c)

	if c.IsAborted() {
		t.Fatal("requisicao ja encaminhada nao deveria ser abortada pelo proxy")
	}
}
