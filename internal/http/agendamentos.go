package http

import (
	nethttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"
)

func (h *APIHandler) UsarAgendamentos(s *service.AgendamentoService) { h.agendamentos = s }

func (h *APIHandler) agendamentosDesligados(c *gin.Context) bool {
	if h.agendamentos != nil {
		return false
	}
	h.responderErro(c, nethttp.StatusNotFound, "agendamentos_desligados", "Agendamentos indisponiveis")
	return true
}

func (h *APIHandler) ListarAgendamentos(c *gin.Context) {
	if h.agendamentosDesligados(c) {
		return
	}
	filtro := models.FiltroAgendamentos{InstanciaID: strings.TrimSpace(c.Query("instancia_id")), Status: strings.TrimSpace(c.Query("status"))}
	if filtro.InstanciaID != "" && !h.garantirInstancia(c, filtro.InstanciaID) {
		return
	}
	lista, err := h.agendamentos.Listar(c.Request.Context(), obterAcessoDashboard(c), filtro)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Agendamentos listados", gin.H{"agendamentos": lista}))
}

func (h *APIHandler) CriarAgendamento(c *gin.Context) {
	if h.agendamentosDesligados(c) {
		return
	}
	var req models.CriarAgendamentoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido (enviar_em no formato 2026-10-20T09:00:00-04:00)")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	a, err := h.agendamentos.Criar(c.Request.Context(), obterAcessoDashboard(c), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	marcarInstanciaAuditoria(c, a.InstanciaID)
	h.descreverAuditoria(c, "Mensagem agendada para "+strings.TrimSpace(a.Numero+" "+a.ChatJID)+" em "+a.EnviarEm.Local().Format("02/01/2006 15:04"))
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Mensagem agendada", a))
}

func (h *APIHandler) AtualizarAgendamento(c *gin.Context) {
	if h.agendamentosDesligados(c) {
		return
	}
	var req models.AtualizarAgendamentoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	a, err := h.agendamentos.Atualizar(c.Request.Context(), obterAcessoDashboard(c), c.Param("agendamentoId"), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	marcarInstanciaAuditoria(c, a.InstanciaID)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Agendamento atualizado", a))
}

func (h *APIHandler) CancelarAgendamento(c *gin.Context) {
	if h.agendamentosDesligados(c) {
		return
	}
	a, err := h.agendamentos.Cancelar(c.Request.Context(), obterAcessoDashboard(c), c.Param("agendamentoId"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	marcarInstanciaAuditoria(c, a.InstanciaID)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Agendamento cancelado", gin.H{"id": a.ID}))
}
