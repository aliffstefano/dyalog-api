package http

import (
	"errors"
	"io"
	nethttp "net/http"

	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

// UsarMetaWebhook liga as rotas publicas que recebem os webhooks da API
// oficial.
func (h *APIHandler) UsarMetaWebhook(s *service.MetaWebhookService) { h.metaWebhook = s }

// VerificarWebhookMeta responde ao GET que a Meta faz ao cadastrar a Callback
// URL: devolve hub.challenge em texto puro quando hub.verify_token confere.
func (h *APIHandler) VerificarWebhookMeta(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	challenge, err := h.metaWebhook.Verificar(c.Request.Context(), c.Param("id"), c.Query("hub.mode"), c.Query("hub.verify_token"), c.Query("hub.challenge"))
	if err != nil {
		c.String(nethttp.StatusForbidden, "verify_token invalido")
		return
	}
	c.String(nethttp.StatusOK, challenge)
}

// ReceberWebhookMeta recebe mensagens e status da Meta. Responde 200 logo e
// processa em segundo plano; erro de assinatura volta 401 para a Meta nao
// considerar entregue.
func (h *APIHandler) ReceberWebhookMeta(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	corpo, err := io.ReadAll(io.LimitReader(c.Request.Body, 5<<20))
	if err != nil {
		c.Status(nethttp.StatusBadRequest)
		return
	}
	err = h.metaWebhook.Receber(c.Request.Context(), c.Param("id"), corpo, c.GetHeader("X-Hub-Signature-256"))
	switch {
	case err == nil:
		c.Status(nethttp.StatusOK)
	case errors.Is(err, service.ErrAssinaturaMetaInvalida):
		c.Status(nethttp.StatusUnauthorized)
	case errors.Is(err, service.ErrInstanciaNaoEncontrada):
		c.Status(nethttp.StatusNotFound)
	default:
		c.Status(nethttp.StatusBadRequest)
	}
}
