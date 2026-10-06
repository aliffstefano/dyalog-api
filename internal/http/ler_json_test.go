package http

import (
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dyalog-api-go/internal/models"

	"github.com/gin-gonic/gin"
)

// O n8n costuma mandar booleanos como texto; o erro tem que dizer qual campo.
func TestLerJSONMostraCampoComTipoErrado(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gravador := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(gravador)
	c.Request = httptest.NewRequest(nethttp.MethodPost, "/", strings.NewReader(`{"numero":"120363000000000000","grupo":"true","mensagem":"oi"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var req models.EnvioTextoRequest
	if (&APIHandler{}).lerJSON(c, &req, nil, "Campos obrigatorios: mensagem e numero ou chat_jid") {
		t.Fatal("deveria recusar grupo como texto")
	}
	if gravador.Code != nethttp.StatusBadRequest || !strings.Contains(gravador.Body.String(), "grupo") {
		t.Fatalf("status=%d corpo=%s", gravador.Code, gravador.Body.String())
	}
}
