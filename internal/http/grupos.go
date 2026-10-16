package http

import (
	"fmt"
	nethttp "net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
)

// Aba Grupos. :grupo aceita o jid (120363...@g.us) ou so o numero do grupo.

func (h *APIHandler) ListarGruposWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	grupos, err := h.instanciaService.GruposWhatsApp(c.Request.Context(), id)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Grupos listados", gin.H{"grupos": grupos}))
}

func (h *APIHandler) GrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	grupo, err := h.instanciaService.GrupoWhatsApp(c.Request.Context(), id, c.Param("grupo"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Grupo consultado", grupo))
}

func (h *APIHandler) CriarGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.CriarGrupoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	grupo, err := h.instanciaService.CriarGrupoWhatsApp(c.Request.Context(), id, req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	h.descreverAuditoria(c, "Grupo criado: "+grupo.Nome)
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Grupo criado", grupo))
}

func (h *APIHandler) AtualizarGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.AtualizarGrupoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	if err := h.instanciaService.AtualizarGrupoWhatsApp(c.Request.Context(), id, c.Param("grupo"), req); err != nil {
		h.tratarErro(c, err)
		return
	}
	h.descreverAuditoria(c, "Grupo alterado: "+c.Param("grupo"))
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Grupo atualizado", gin.H{"grupo": c.Param("grupo")}))
}

func (h *APIHandler) FotoGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	remover := c.Request.Method == nethttp.MethodDelete
	var req models.AtualizarFotoPerfilRequest
	if !remover {
		if err := c.ShouldBindJSON(&req); err != nil {
			h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
			return
		}
	}
	if err := h.instanciaService.FotoGrupoWhatsApp(c.Request.Context(), id, c.Param("grupo"), req, remover); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Foto do grupo atualizada", gin.H{"grupo": c.Param("grupo")}))
}

func (h *APIHandler) ConviteGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	redefinir := c.Query("redefinir") == "1" || strings.EqualFold(c.Query("redefinir"), "true")
	if redefinir && !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	link, err := h.instanciaService.LinkConviteGrupoWhatsApp(c.Request.Context(), id, c.Param("grupo"), redefinir)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	if redefinir {
		h.descreverAuditoria(c, "Link de convite redefinido: "+c.Param("grupo"))
		h.registrarAuditoriaLeitura(c, models.AuditoriaGrupo, "grupo.convite_redefinido")
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Link de convite", gin.H{"link": link}))
}

var rotulosAcaoParticipante = map[string]string{"adicionar": "adicionados", "remover": "removidos", "promover": "promovidos a admin", "rebaixar": "rebaixados"}

func (h *APIHandler) ParticipantesGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.ParticipantesGrupoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	resultado, err := h.instanciaService.ParticipantesGrupoWhatsApp(c.Request.Context(), id, c.Param("grupo"), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	ok := 0
	for _, r := range resultado {
		if r.Ok {
			ok++
		}
	}
	h.descreverAuditoria(c, fmt.Sprintf("Participantes %s no grupo %s: %d de %d", rotulosAcaoParticipante[strings.ToLower(req.Acao)], c.Param("grupo"), ok, len(resultado)))
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Participantes atualizados", gin.H{"resultado": resultado}))
}

func (h *APIHandler) SairGrupoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	if err := h.instanciaService.SairGrupoWhatsApp(c.Request.Context(), id, c.Param("grupo")); err != nil {
		h.tratarErro(c, err)
		return
	}
	h.descreverAuditoria(c, "Saiu do grupo "+c.Param("grupo"))
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Saiu do grupo", gin.H{"grupo": c.Param("grupo")}))
}
