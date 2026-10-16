package http

import (
	"errors"
	"io"
	nethttp "net/http"

	"dyalog-api-go/internal/models"
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

// ReceberWebhookMeta recebe mensagens e status da Meta. Grava no banco,
// responde 200 e processa em segundo plano; erro de assinatura volta 401 para
// a Meta nao considerar entregue.
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
	responderWebhookMeta(c, h.metaWebhook.Receber(c.Request.Context(), c.Param("id"), corpo, c.GetHeader("X-Hub-Signature-256")))
}

// ConfigCadastroMeta entrega ao painel o app_id e o config_id para abrir o
// login da Meta (cadastro incorporado).
func (h *APIHandler) ConfigCadastroMeta(c *gin.Context) {
	if !h.garantirCriarInstancia(c) {
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Configuracao do cadastro incorporado", h.instanciaService.ConfigCadastroMeta()))
}

// CadastrarViaMeta conclui o cadastro incorporado e cria a instancia oficial.
func (h *APIHandler) CadastrarViaMeta(c *gin.Context) {
	if !h.garantirCriarInstancia(c) {
		return
	}
	var req struct {
		service.CadastroMetaRequest
		ContaID string `json:"conta_id"`
	}
	if !h.lerJSON(c, &req, nil, "Informe nome, code, waba_id e phone_number_id") {
		return
	}
	contaID, ok := h.contaDaNovaInstancia(c, req.ContaID)
	if !ok {
		return
	}
	instancia, avisos, err := h.instanciaService.CadastrarViaMeta(c.Request.Context(), req.CadastroMetaRequest)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	if !h.atribuirConta(c, &instancia, contaID) {
		return
	}
	marcarInstanciaAuditoria(c, instancia.ID)
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Instancia oficial criada pelo cadastro da Meta", gin.H{
		"id": instancia.ID, "instancia": instancia, "avisos": avisos,
	}))
}

// VerificarWebhookMetaApp e ReceberWebhookMetaApp atendem a URL de callback
// unica do app (provedor): /webhook/meta. Os eventos vao para a instancia pelo
// phone_number_id.
func (h *APIHandler) VerificarWebhookMetaApp(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	challenge, err := h.metaWebhook.VerificarApp(c.Query("hub.mode"), c.Query("hub.verify_token"), c.Query("hub.challenge"))
	if err != nil {
		c.String(nethttp.StatusForbidden, "verify_token invalido ou META_WEBHOOK_VERIFY_TOKEN nao configurado")
		return
	}
	c.String(nethttp.StatusOK, challenge)
}

func (h *APIHandler) ReceberWebhookMetaApp(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	corpo, err := io.ReadAll(io.LimitReader(c.Request.Body, 5<<20))
	if err != nil {
		c.Status(nethttp.StatusBadRequest)
		return
	}
	responderWebhookMeta(c, h.metaWebhook.ReceberApp(c.Request.Context(), corpo, c.GetHeader("X-Hub-Signature-256")))
}

// responderWebhookMeta: 200 so quando o webhook ja esta gravado. Falha ao
// gravar volta 500 para a Meta reenviar depois.
func responderWebhookMeta(c *gin.Context, err error) {
	switch {
	case err == nil:
		c.Status(nethttp.StatusOK)
	case errors.Is(err, service.ErrAssinaturaMetaInvalida):
		c.Status(nethttp.StatusUnauthorized)
	case errors.Is(err, service.ErrInstanciaNaoEncontrada):
		c.Status(nethttp.StatusNotFound)
	case errors.Is(err, service.ErrEntradaInvalida):
		c.Status(nethttp.StatusBadRequest)
	default:
		c.Status(nethttp.StatusInternalServerError)
	}
}
