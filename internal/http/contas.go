package http

import (
	nethttp "net/http"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

// UsarContas liga login com e-mail, equipe, contas e planos.
func (h *APIHandler) UsarContas(s *service.ContaService) {
	h.contas = s
	h.limiteEsqueci = novoLimitador(5)
}

func (h *APIHandler) contasDesligadas(c *gin.Context) bool {
	if h.contas != nil {
		return false
	}
	h.responderErro(c, nethttp.StatusNotFound, "contas_desligadas", "Contas de usuario nao habilitadas")
	return true
}

// ── Publicas (sem login) ─────────────────────────────────────────────────────

func (h *APIHandler) InfoConvite(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	info, err := h.contas.InfoConvite(c.Request.Context(), c.Query("token"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Convite valido", info))
}

func (h *APIHandler) AceitarConvite(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Token string `json:"token"`
		Nome  string `json:"nome"`
		Senha string `json:"senha"`
	}
	if !h.lerJSON(c, &req, nil, "Informe token, nome e senha") {
		return
	}
	token, acesso, err := h.contas.AceitarConvite(c.Request.Context(), req.Token, req.Nome, req.Senha)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	h.definirCookieSessao(c, token)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Conta criada, bem-vindo", acesso))
}

func (h *APIHandler) EsqueciSenha(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if !h.lerJSON(c, &req, nil, "Informe o e-mail") {
		return
	}
	// Limite por IP: o pedido manda e-mail, nao pode virar disparador de spam.
	if bloqueado, espera := h.limiteEsqueci.bloqueado(ipCliente(c)); bloqueado {
		responderLimite(c, "muitas_tentativas", "Muitos pedidos de redefinicao; tente mais tarde", espera)
		return
	}
	h.limiteEsqueci.registrarFalha(ipCliente(c), req.Email)
	h.contas.EsqueciSenha(c.Request.Context(), req.Email)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Se o e-mail tiver cadastro, enviamos o link para redefinir a senha", gin.H{}))
}

func (h *APIHandler) RedefinirSenha(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Token string `json:"token"`
		Senha string `json:"senha"`
	}
	if !h.lerJSON(c, &req, nil, "Informe token e senha") {
		return
	}
	if err := h.contas.RedefinirSenha(c.Request.Context(), req.Token, req.Senha); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Senha redefinida; entre com a nova senha", gin.H{}))
}

// ── Usuario logado ───────────────────────────────────────────────────────────

func (h *APIHandler) TrocarSenha(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Atual string `json:"senha_atual"`
		Nova  string `json:"senha_nova"`
	}
	if !h.lerJSON(c, &req, nil, "Informe senha_atual e senha_nova") {
		return
	}
	if err := h.contas.TrocarSenha(c.Request.Context(), obterAcessoDashboard(c), req.Atual, req.Nova); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Senha trocada; entre de novo nos outros aparelhos", gin.H{}))
}

func (h *APIHandler) Equipe(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	equipe, err := h.contas.Equipe(c.Request.Context(), obterAcessoDashboard(c), c.Query("conta_id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Equipe listada", equipe))
}

func (h *APIHandler) ConvidarEquipe(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Email   string `json:"email"`
		Papel   string `json:"papel"`
		ContaID string `json:"conta_id"`
	}
	if !h.lerJSON(c, &req, nil, "Informe email e papel") {
		return
	}
	acesso := obterAcessoDashboard(c)
	contaID := req.ContaID
	if !acesso.TudoLiberado() {
		contaID = acesso.ContaID
	}
	resultado, err := h.contas.Convidar(c.Request.Context(), acesso, contaID, req.Email, req.Papel)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Convite criado", resultado))
}

func (h *APIHandler) CancelarConvite(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	if err := h.contas.CancelarConvite(c.Request.Context(), obterAcessoDashboard(c), c.Query("conta_id"), c.Param("conviteId")); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Convite cancelado", gin.H{"id": c.Param("conviteId")}))
}

func (h *APIHandler) AtualizarMembro(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Papel string `json:"papel"`
		Ativo *bool  `json:"ativo"`
	}
	if !h.lerJSON(c, &req, nil, "Informe papel ou ativo") {
		return
	}
	usuario, err := h.contas.AtualizarMembro(c.Request.Context(), obterAcessoDashboard(c), c.Param("usuarioId"), req.Papel, req.Ativo)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Usuario atualizado", usuario))
}

func (h *APIHandler) RemoverMembro(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	if err := h.contas.RemoverMembro(c.Request.Context(), obterAcessoDashboard(c), c.Param("usuarioId")); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Usuario removido", gin.H{"id": c.Param("usuarioId")}))
}

// ── Superadmin: contas, planos, administradores ──────────────────────────────

func (h *APIHandler) ListarContas(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	contas, err := h.contas.ListarContas(c.Request.Context(), obterAcessoDashboard(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Contas listadas", gin.H{"contas": contas}))
}

func (h *APIHandler) CriarConta(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Nome      string `json:"nome"`
		PlanoID   string `json:"plano_id"`
		EmailDono string `json:"email_dono"`
	}
	if !h.lerJSON(c, &req, nil, "Informe nome e email_dono") {
		return
	}
	conta, convite, err := h.contas.CriarConta(c.Request.Context(), obterAcessoDashboard(c), req.Nome, req.PlanoID, req.EmailDono)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Conta criada e dono convidado", gin.H{"conta": conta, "convite": convite}))
}

func (h *APIHandler) AtualizarConta(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Nome             string  `json:"nome"`
		PlanoID          *string `json:"plano_id"`
		Status           string  `json:"status"`
		LimiteInstancias *int    `json:"limite_instancias"`
		TiposInstancia   *string `json:"tipos_instancia"`
	}
	if !h.lerJSON(c, &req, nil, "Informe os campos da conta") {
		return
	}
	acesso := obterAcessoDashboard(c)
	conta := models.Conta{ID: c.Param("contaId"), Nome: req.Nome, Status: req.Status, LimiteInstancias: -1}
	if req.LimiteInstancias != nil {
		conta.LimiteInstancias = *req.LimiteInstancias
	}
	if req.PlanoID != nil {
		conta.PlanoID = *req.PlanoID
	}
	if req.TiposInstancia != nil {
		conta.TiposInstancia = *req.TiposInstancia
	}
	if req.PlanoID == nil || req.TiposInstancia == nil {
		if atual, err := h.contas.ListarContas(c.Request.Context(), acesso); err == nil {
			for _, a := range atual {
				if a.ID != conta.ID {
					continue
				}
				if req.PlanoID == nil {
					conta.PlanoID = a.PlanoID
				}
				if req.TiposInstancia == nil {
					conta.TiposInstancia = a.TiposInstancia
				}
			}
		}
	}
	atualizada, err := h.contas.AtualizarConta(c.Request.Context(), acesso, conta)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Conta atualizada", atualizada))
}

func (h *APIHandler) MoverInstanciaConta(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		ContaID string `json:"conta_id"`
	}
	if !h.lerJSON(c, &req, nil, "Informe conta_id (vazio tira de qualquer conta)") {
		return
	}
	if err := h.contas.MoverInstancia(c.Request.Context(), obterAcessoDashboard(c), c.Param("id"), req.ContaID); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Instancia movida", gin.H{"id": c.Param("id"), "conta_id": req.ContaID}))
}

func (h *APIHandler) ListarPlanos(c *gin.Context) {
	if h.contasDesligadas(c) || !h.garantirMaster(c) {
		return
	}
	planos, err := h.contas.ListarPlanos(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Planos listados", gin.H{"planos": planos}))
}

func (h *APIHandler) SalvarPlano(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var plano models.Plano
	if !h.lerJSON(c, &plano, nil, "Informe nome, preco_centavos e limite_instancias") {
		return
	}
	if id := c.Param("planoId"); id != "" {
		plano.ID = id
	}
	salvo, err := h.contas.SalvarPlano(c.Request.Context(), obterAcessoDashboard(c), plano)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Plano salvo", salvo))
}

func (h *APIHandler) Administradores(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	lista, err := h.contas.Administradores(c.Request.Context(), obterAcessoDashboard(c))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Administradores listados", lista))
}

func (h *APIHandler) ConvidarAdministrador(c *gin.Context) {
	if h.contasDesligadas(c) {
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if !h.lerJSON(c, &req, nil, "Informe o e-mail") {
		return
	}
	resultado, err := h.contas.ConvidarSuperadmin(c.Request.Context(), obterAcessoDashboard(c), req.Email)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Convite criado", resultado))
}
