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

// VerificarWebhookInstagram e ReceberWebhookInstagram atendem a URL de
// callback do Instagram (/webhook/instagram). Os eventos vao para a instancia
// pelo id da conta.
func (h *APIHandler) VerificarWebhookInstagram(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	challenge, err := h.metaWebhook.VerificarInstagram(c.Query("hub.mode"), c.Query("hub.verify_token"), c.Query("hub.challenge"))
	if err != nil {
		c.String(nethttp.StatusForbidden, "verify_token invalido ou INSTAGRAM_WEBHOOK_VERIFY_TOKEN nao configurado")
		return
	}
	c.String(nethttp.StatusOK, challenge)
}

func (h *APIHandler) ReceberWebhookInstagram(c *gin.Context) {
	if h.metaWebhook == nil {
		c.Status(nethttp.StatusNotFound)
		return
	}
	corpo, err := io.ReadAll(io.LimitReader(c.Request.Body, 5<<20))
	if err != nil {
		c.Status(nethttp.StatusBadRequest)
		return
	}
	responderWebhookMeta(c, h.metaWebhook.ReceberInstagram(c.Request.Context(), corpo, c.GetHeader("X-Hub-Signature-256")))
}

// RetornoLoginInstagram e a pagina para onde o Instagram volta depois do
// login: entrega code e state ao painel que abriu o popup (mesma origem) e
// fecha. A troca do code pelo token e feita pelo painel, ja autenticado.
func (h *APIHandler) RetornoLoginInstagram(c *gin.Context) {
	c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'")
	c.Data(nethttp.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><meta charset="utf-8"><title>Instagram</title>
<p id="t">Conectando o Instagram...</p>
<script data-cfasync="false">
var q = new URLSearchParams(location.search);
var dados = { tipo: 'instagram_login', code: q.get('code') || '', state: q.get('state') || '', erro: q.get('error_description') || q.get('error') || '' };
// O Instagram pode cortar o window.opener (COOP): o BroadcastChannel chega ao
// painel aberto na mesma origem mesmo assim.
try { new BroadcastChannel('instagram_login').postMessage(dados); } catch (e) {}
if (window.opener) window.opener.postMessage(dados, location.origin);
document.getElementById('t').textContent = 'Pronto. Pode fechar esta janela e voltar ao painel.';
setTimeout(function () { window.close(); }, 300);
</script>`))
}

// ConfigCadastroInstagram entrega ao painel o endereco do login do Instagram.
func (h *APIHandler) ConfigCadastroInstagram(c *gin.Context) {
	if !h.garantirCriarInstancia(c) {
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Configuracao do login do Instagram", h.instanciaService.ConfigCadastroInstagram()))
}

// CadastrarViaInstagram troca o code do login e cria a instancia do Instagram.
func (h *APIHandler) CadastrarViaInstagram(c *gin.Context) {
	if !h.garantirCriarInstancia(c) {
		return
	}
	var req struct {
		Nome    string `json:"nome"`
		Code    string `json:"code"`
		ContaID string `json:"conta_id"`
	}
	if !h.lerJSON(c, &req, nil, "Informe nome e code") {
		return
	}
	contaID, ok := h.contaDaNovaInstancia(c, req.ContaID)
	if !ok {
		return
	}
	instancia, avisos, err := h.instanciaService.CadastrarViaInstagram(c.Request.Context(), req.Nome, req.Code)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	if !h.atribuirConta(c, &instancia, contaID) {
		return
	}
	marcarInstanciaAuditoria(c, instancia.ID)
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Instancia do Instagram criada pelo login", gin.H{
		"id": instancia.ID, "instancia": instancia, "avisos": avisos,
	}))
}

// ConfigurarInstagramInstancia troca o token da conta (vazio revalida).
func (h *APIHandler) ConfigurarInstagramInstancia(c *gin.Context) {
	if !h.garantirInstanciaGerenciar(c, c.Param("id")) {
		return
	}
	var req models.ConfigurarInstagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido: "+err.Error())
		return
	}
	resumo, err := h.instanciaService.ConfigurarInstagram(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Conta do Instagram validada e salva", resumo))
}
