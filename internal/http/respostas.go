package http

import (
	"errors"
	nethttp "net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// Respostas rapidas e etiquetas do chat: pertencem a conta de quem acessa
// (token master e superadmin sem conta usam a lista geral).

func (h *APIHandler) UsarRespostas(s store.RespostasStore) { h.respostas = s }

var (
	padraoAtalho = regexp.MustCompile(`^[a-z0-9_-]{1,30}$`)
	padraoCor    = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
)

// contaDoAcesso: a conta dona das respostas/etiquetas para este acesso.
func (h *APIHandler) contaDoAcesso(c *gin.Context) string {
	acesso := obterAcessoDashboard(c)
	switch acesso.Tipo {
	case "usuario":
		return acesso.ContaID
	case "instancia":
		if inst, err := h.instanciaService.Buscar(c.Request.Context(), acesso.InstanciaID); err == nil {
			return inst.ContaID
		}
	}
	return ""
}

func (h *APIHandler) respostasDesligadas(c *gin.Context) bool {
	if h.respostas != nil {
		return false
	}
	h.responderErro(c, nethttp.StatusNotFound, "indisponivel", "Respostas rapidas indisponiveis")
	return true
}

func (h *APIHandler) erroRespostas(c *gin.Context, err error, oQue string) {
	switch {
	case errors.Is(err, store.ErrDuplicado):
		h.responderErro(c, nethttp.StatusConflict, "duplicado", "Ja existe "+oQue+" com esse nome")
	case errors.Is(err, store.ErrNaoEncontrado):
		h.responderErro(c, nethttp.StatusNotFound, "nao_encontrado", strings.ToUpper(oQue[:1])+oQue[1:]+" nao encontrada")
	default:
		h.tratarErro(c, err)
	}
}

func (h *APIHandler) ListarRespostasRapidas(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	lista, err := h.respostas.ListarRespostasRapidas(c.Request.Context(), h.contaDoAcesso(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Respostas rapidas", gin.H{"respostas": lista}))
}

func (h *APIHandler) SalvarRespostaRapida(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	var req models.RespostaRapida
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	req.ID = c.Param("respostaId")
	req.ContaID = h.contaDoAcesso(c)
	req.Atalho = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(req.Atalho), "/"))
	req.Texto = strings.TrimSpace(req.Texto)
	if !padraoAtalho.MatchString(req.Atalho) {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Atalho: de 1 a 30 letras minusculas, numeros, - ou _ (sem espacos)")
		return
	}
	if req.Texto == "" || len([]rune(req.Texto)) > 4096 {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Texto: de 1 a 4096 caracteres")
		return
	}
	salva, err := h.respostas.SalvarRespostaRapida(c.Request.Context(), req)
	if err != nil {
		h.erroRespostas(c, err, "uma resposta")
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Resposta rapida salva", salva))
}

func (h *APIHandler) ExcluirRespostaRapida(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	if err := h.respostas.ExcluirRespostaRapida(c.Request.Context(), h.contaDoAcesso(c), c.Param("respostaId")); err != nil {
		h.erroRespostas(c, err, "resposta")
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Resposta rapida excluida", gin.H{}))
}

func (h *APIHandler) ListarEtiquetas(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	lista, err := h.respostas.ListarEtiquetas(c.Request.Context(), h.contaDoAcesso(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Etiquetas", gin.H{"etiquetas": lista}))
}

func (h *APIHandler) SalvarEtiqueta(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	var req models.Etiqueta
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	req.ID = c.Param("etiquetaId")
	req.ContaID = h.contaDoAcesso(c)
	req.Nome = strings.TrimSpace(req.Nome)
	if req.Nome == "" || len([]rune(req.Nome)) > 30 {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Nome da etiqueta: de 1 a 30 caracteres")
		return
	}
	if req.Cor == "" {
		req.Cor = "#1995bd"
	}
	if !padraoCor.MatchString(req.Cor) {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Cor no formato #RRGGBB")
		return
	}
	salva, err := h.respostas.SalvarEtiqueta(c.Request.Context(), req)
	if err != nil {
		h.erroRespostas(c, err, "uma etiqueta")
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Etiqueta salva", salva))
}

func (h *APIHandler) ExcluirEtiqueta(c *gin.Context) {
	if h.respostasDesligadas(c) {
		return
	}
	if err := h.respostas.ExcluirEtiqueta(c.Request.Context(), h.contaDoAcesso(c), c.Param("etiquetaId")); err != nil {
		h.erroRespostas(c, err, "etiqueta")
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Etiqueta excluida", gin.H{}))
}

func (h *APIHandler) EtiquetasDoChat(c *gin.Context) {
	id := c.Param("id")
	if h.respostasDesligadas(c) || !h.garantirInstancia(c, id) {
		return
	}
	mapa, err := h.respostas.EtiquetasDoChat(c.Request.Context(), id)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Etiquetas das conversas", gin.H{"conversas": mapa}))
}

func (h *APIHandler) DefinirEtiquetasChat(c *gin.Context) {
	id := c.Param("id")
	if h.respostasDesligadas(c) || !h.garantirInstancia(c, id) {
		return
	}
	var req models.DefinirEtiquetasChatRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.ChatJID) == "" {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe chat_jid e etiquetas")
		return
	}
	// So etiquetas da propria conta.
	validas, err := h.respostas.ListarEtiquetas(c.Request.Context(), h.contaDoAcesso(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	permitidas := map[string]bool{}
	for _, e := range validas {
		permitidas[e.ID] = true
	}
	escolhidas := []string{}
	for _, e := range req.Etiquetas {
		if permitidas[e] {
			escolhidas = append(escolhidas, e)
		}
	}
	if err := h.respostas.DefinirEtiquetasChat(c.Request.Context(), id, strings.TrimSpace(req.ChatJID), escolhidas); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Etiquetas da conversa salvas", gin.H{"etiquetas": escolhidas}))
}
