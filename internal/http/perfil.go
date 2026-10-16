package http

import (
	"encoding/csv"
	"fmt"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"
)

// Perfil, privacidade e contatos da conta de WhatsApp da instancia.

func (h *APIHandler) PerfilWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	perfil, err := h.instanciaService.PerfilWhatsApp(c.Request.Context(), id)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Perfil consultado", perfil))
}

func (h *APIHandler) AtualizarPerfilWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.AtualizarPerfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	perfil, err := h.instanciaService.AtualizarPerfilWhatsApp(c.Request.Context(), id, req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	var mudou []string
	if req.Nome != nil {
		mudou = append(mudou, "nome para \""+strings.TrimSpace(*req.Nome)+"\"")
	}
	if req.Recado != nil {
		mudou = append(mudou, "recado")
	}
	h.descreverAuditoria(c, "Perfil alterado: "+strings.Join(mudou, " e "))
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Perfil atualizado", perfil))
}

func (h *APIHandler) AtualizarFotoPerfilWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.AtualizarFotoPerfilRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	if err := h.instanciaService.AtualizarFotoPerfilWhatsApp(c.Request.Context(), id, req, false); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Foto atualizada", gin.H{"instancia": id}))
}

func (h *APIHandler) RemoverFotoPerfilWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	if err := h.instanciaService.AtualizarFotoPerfilWhatsApp(c.Request.Context(), id, models.AtualizarFotoPerfilRequest{}, true); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Foto removida", gin.H{"instancia": id}))
}

func (h *APIHandler) AtualizarPrivacidadeWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var valores map[string]string
	if err := c.ShouldBindJSON(&valores); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Envie um objeto com os campos de privacidade e os valores (texto)")
		return
	}
	privacidade, err := h.instanciaService.AtualizarPrivacidadeWhatsApp(c.Request.Context(), id, valores)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	campos := make([]string, 0, len(valores))
	for campo, valor := range valores {
		campos = append(campos, campo+"="+valor)
	}
	h.descreverAuditoria(c, "Privacidade alterada: "+strings.Join(campos, ", "))
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Privacidade atualizada", privacidade))
}

func filtroContatosDaQuery(c *gin.Context) service.FiltroContatos {
	pagina, _ := strconv.Atoi(c.DefaultQuery("pagina", "1"))
	porPagina, _ := strconv.Atoi(c.DefaultQuery("por_pagina", "24"))
	if porPagina <= 0 || porPagina > 500 {
		porPagina = 24
	}
	return service.FiltroContatos{
		Busca:     c.Query("busca"),
		Tipo:      strings.ToLower(strings.TrimSpace(c.Query("tipo"))),
		Pagina:    pagina,
		PorPagina: porPagina,
		Atualizar: c.Query("atualizar") == "1" || strings.EqualFold(c.Query("atualizar"), "true"),
	}
}

func (h *APIHandler) ListarContatosWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	lista, err := h.instanciaService.ContatosWhatsApp(c.Request.Context(), id, filtroContatosDaQuery(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Contatos listados", lista))
}

// ExportarContatosWhatsApp devolve os contatos (com o mesmo filtro da tela) em CSV.
func (h *APIHandler) ExportarContatosWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	filtro := filtroContatosDaQuery(c)
	filtro.PorPagina = 0
	lista, err := h.instanciaService.ContatosWhatsApp(c.Request.Context(), id, filtro)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	nome := fmt.Sprintf("contatos-%s-%s.csv", id, time.Now().Format("20060102"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+nome+`"`)
	// BOM para o Excel abrir os acentos certo.
	_, _ = c.Writer.Write([]byte("\ufeff"))
	w := csv.NewWriter(c.Writer)
	w.Comma = ';'
	_ = w.Write([]string{"nome", "numero", "jid", "tipo", "nome_no_whatsapp", "empresa", "salvo", "bloqueado"})
	for _, contato := range lista.Contatos {
		_ = w.Write([]string{protegerCSV(contato.Nome), contato.Numero, contato.JID, contato.Tipo, protegerCSV(contato.NomePush), protegerCSV(contato.Empresa),
			strconv.FormatBool(contato.Salvo), strconv.FormatBool(contato.Bloqueado)})
	}
	w.Flush()
	h.descreverAuditoria(c, fmt.Sprintf("Contatos exportados em CSV (%d)", len(lista.Contatos)))
	h.registrarAuditoriaLeitura(c, models.AuditoriaContato, "contatos.exportados")
}

// protegerCSV evita que um nome comecando com =, +, - ou @ vire formula no Excel.
func protegerCSV(texto string) string {
	if texto != "" && strings.ContainsRune("=+-@\t\r", rune(texto[0])) {
		return "'" + texto
	}
	return texto
}

func (h *APIHandler) VerificarNumerosWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	var req models.VerificarNumerosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	resultado, err := h.instanciaService.VerificarNumerosWhatsApp(c.Request.Context(), id, req.Numeros)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Numeros verificados", resultado))
}

func (h *APIHandler) BloquearContatoWhatsApp(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstanciaGerenciar(c, id) {
		return
	}
	var req models.BloqueioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	if err := h.instanciaService.BloquearContatoWhatsApp(c.Request.Context(), id, req); err != nil {
		h.tratarErro(c, err)
		return
	}
	alvo := strings.TrimSpace(req.Numero)
	if alvo == "" {
		alvo = req.ChatJID
	}
	if req.Bloquear {
		h.descreverAuditoria(c, "Contato bloqueado: "+alvo)
	} else {
		h.descreverAuditoria(c, "Contato desbloqueado: "+alvo)
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Bloqueio atualizado", gin.H{"bloqueado": req.Bloquear}))
}
