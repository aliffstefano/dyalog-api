package http

import (
	nethttp "net/http"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

// UsarChat liga as rotas da tela de Chat (nil = chat desligado).
func (h *APIHandler) UsarChat(s *service.ChatService) { h.chat = s }

func (h *APIHandler) chatDesligado(c *gin.Context) bool {
	if h.chat != nil {
		return false
	}
	h.responderErro(c, nethttp.StatusNotFound, "chat_desligado", "O chat do painel esta desligado (CHAT_RETENTION_DAYS=0)")
	return true
}

func (h *APIHandler) ListarConversasChat(c *gin.Context) {
	instanciaID := c.Param("id")
	if h.chatDesligado(c) || !h.preencherInstanciaDaRequisicao(c, &instanciaID) {
		return
	}
	conversas, err := h.chat.ListarConversas(c.Request.Context(), instanciaID)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Conversas listadas com sucesso", gin.H{"conversas": conversas}))
}

func (h *APIHandler) ListarMensagensChat(c *gin.Context) {
	instanciaID := c.Param("id")
	if h.chatDesligado(c) || !h.preencherInstanciaDaRequisicao(c, &instanciaID) {
		return
	}
	mensagens, err := h.chat.ListarMensagens(c.Request.Context(), instanciaID, c.Query("chat_jid"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Mensagens listadas com sucesso", gin.H{"mensagens": mensagens}))
}
