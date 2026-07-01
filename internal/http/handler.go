package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
)

type APIHandler struct {
	cfg              *config.Config
	instanciaService *service.InstanciaService
	mensagemService  *service.MensagemService
	chamadaService   *service.ChamadaService
	midiaService     *service.MidiaService
	webhookService   *service.WebhookService
	sistemaService   *service.SistemaService
	authService      *service.AuthService
}

type criarInstanciaRequest struct {
	Nome string `json:"nome" binding:"required"`
}

type webhookRequest struct {
	Nome    string   `json:"nome" binding:"required"`
	URL     string   `json:"url" binding:"required"`
	Eventos []string `json:"eventos" binding:"required"`
	Ativo   *bool    `json:"ativo,omitempty"`
}

type loginDashboardRequest struct {
	Token string `json:"token" binding:"required"`
}

type atualizarTokenInstanciaRequest struct {
	Token string `json:"token" binding:"required"`
}

type atualizarHistoricoInstanciaRequest struct {
	Dias int `json:"dias"`
}

type atualizarProxyInstanciaRequest struct {
	Modo string `json:"modo"`
	URL  string `json:"url"`
}

type atualizarPresencaInstanciaRequest struct {
	Presenca string `json:"presenca,omitempty"`
	Acao     string `json:"acao,omitempty"`
	Type     string `json:"type,omitempty"`
}

type atualizarConfiguracaoAvancadaRequest struct {
	ManterOnline             bool   `json:"manter_online"`
	RejeitarChamadas         bool   `json:"rejeitar_chamadas"`
	MensagemRejeitarChamadas string `json:"mensagem_rejeitar_chamadas,omitempty"`
	MarcarLidaAutomatico     bool   `json:"marcar_lida_automatico"`
	IgnorarGrupos            bool   `json:"ignorar_grupos"`
	IgnorarStatus            bool   `json:"ignorar_status"`
}

type solicitarCodigoPareamentoRequest struct {
	Numero string `json:"numero" binding:"required"`
}

type atualizarProxyGlobalRequest struct {
	URL   string `json:"url"`
	Ativo *bool  `json:"ativo"`
}

func NovoAPIHandler(cfg *config.Config, instanciaService *service.InstanciaService, mensagemService *service.MensagemService, chamadaService *service.ChamadaService, midiaService *service.MidiaService, webhookService *service.WebhookService, sistemaService *service.SistemaService, authService *service.AuthService) *APIHandler {
	return &APIHandler{cfg: cfg, instanciaService: instanciaService, mensagemService: mensagemService, chamadaService: chamadaService, midiaService: midiaService, webhookService: webhookService, sistemaService: sistemaService, authService: authService}
}

func (h *APIHandler) LoginDashboard(c *gin.Context) {
	var req loginDashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe o token de acesso")
		return
	}
	acesso, err := h.authService.Autenticar(c.Request.Context(), req.Token)
	if err != nil {
		h.responderErro(c, nethttp.StatusUnauthorized, "nao_autenticado", "Token invalido")
		return
	}
	c.SetCookie(h.cfg.DashboardCookieNome, req.Token, 86400*15, "/", "", false, true)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Login realizado com sucesso", acesso))
}

func (h *APIHandler) LogoutDashboard(c *gin.Context) {
	c.SetCookie(h.cfg.DashboardCookieNome, "", -1, "/", "", false, true)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Logout realizado com sucesso", gin.H{}))
}

func (h *APIHandler) SessaoDashboard(c *gin.Context) {
	acesso := obterAcessoDashboard(c)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Sessao consultada com sucesso", acesso))
}

func (h *APIHandler) garantirMaster(c *gin.Context) bool {
	acesso := obterAcessoDashboard(c)
	if h.authService.PodeGerenciarTudo(acesso) {
		return true
	}
	h.responderErro(c, nethttp.StatusForbidden, "acesso_negado", "Acesso restrito ao token master")
	return false
}

func (h *APIHandler) garantirInstancia(c *gin.Context, instanciaID string) bool {
	if err := h.authService.GarantirInstancia(obterAcessoDashboard(c), instanciaID); err == nil {
		return true
	}
	h.responderErro(c, nethttp.StatusForbidden, "acesso_negado", "Esse token nao possui acesso a esta instancia")
	return false
}

func (h *APIHandler) Saude(c *gin.Context) {
	versao, err := h.sistemaService.ObterVersao(c.Request.Context())
	if err != nil {
		h.responderErro(c, nethttp.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("API operacional", gin.H{
		"status":                 "ok",
		"api_versao":             "v1",
		"aplicacao_versao":       versao.AplicacaoVersao,
		"whatsmeow_versao":       versao.WhatsmeowVersao,
		"atualizacao_disponivel": versao.AtualizacaoDisponivel,
		"status_atualizacao":     versao.StatusAtualizacao,
	}))
}

func (h *APIHandler) Documentacao(c *gin.Context) {
	conteudo, err := os.ReadFile("docs/endpoints.md")
	if err != nil {
		h.responderErro(c, nethttp.StatusInternalServerError, "erro_documentacao", "Nao foi possivel carregar a documentacao da API")
		return
	}
	pagina := "<!DOCTYPE html><html lang=\"pt-BR\"><head><meta charset=\"UTF-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><title>Dyalog API GO - Documentacao</title><style>body{margin:0;font-family:Georgia,serif;background:#f4efe6;color:#1f2933}main{max-width:980px;margin:0 auto;padding:32px 24px 64px}h1{margin:0 0 8px;font-size:40px}p{color:#52606d}a{color:#127a6b;text-decoration:none}code{font-family:Consolas,monospace}pre{white-space:pre-wrap;word-break:break-word;background:#fff;border:1px solid #d9d2c3;border-radius:20px;padding:24px;font:14px/1.6 Consolas, monospace;overflow:auto;box-shadow:0 18px 40px rgba(31,41,51,.08)}.bar{display:flex;justify-content:space-between;align-items:center;gap:16px;margin-bottom:24px}.chip{display:inline-block;padding:10px 14px;border:1px solid #d9d2c3;border-radius:999px;background:#fff}</style></head><body><main><div class=\"bar\"><div><h1>Documentacao da API</h1><p>Referencia servida pelo proprio backend em <code>/api/v1/docs</code>.</p></div><a class=\"chip\" href=\"/\">Abrir dashboard</a></div><pre>" + html.EscapeString(string(conteudo)) + "</pre></main></body></html>"
	c.Data(nethttp.StatusOK, "text/html; charset=utf-8", []byte(pagina))
}

func (h *APIHandler) VersaoSistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	versao, err := h.sistemaService.ObterVersao(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Versao do sistema consultada com sucesso", versao))
}

func (h *APIHandler) AtualizacoesSistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	status, err := h.sistemaService.ObterAtualizacoes(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Atualizacoes consultadas com sucesso", status))
}

func (h *APIHandler) VerificarAtualizacoesSistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	status, err := h.sistemaService.VerificarAtualizacao(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Verificacao de atualizacao executada com sucesso", status))
}

func (h *APIHandler) AplicarAtualizacaoSistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	if err := h.sistemaService.AutorizarAplicacao(c.GetHeader("X-Update-Token")); err != nil {
		h.tratarErro(c, err)
		return
	}
	resultado, err := h.sistemaService.AplicarAtualizacao(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusAccepted, models.NovaRespostaSucesso("Preparo de atualizacao gerado com sucesso", resultado))
}

func (h *APIHandler) ProxySistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	proxy, err := h.instanciaService.ObterProxyGlobal(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Proxy global consultado com sucesso", proxy))
}

func (h *APIHandler) AtualizarProxySistema(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	var req atualizarProxyGlobalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe a configuracao do proxy global")
		return
	}
	ativo := strings.TrimSpace(req.URL) != ""
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	proxy, err := h.instanciaService.AtualizarProxyGlobal(c.Request.Context(), req.URL, ativo)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Proxy global atualizado com sucesso", proxy))
}

func (h *APIHandler) CriarInstancia(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	var req criarInstanciaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Nome da instancia e obrigatorio")
		return
	}
	instancia, err := h.instanciaService.Criar(c.Request.Context(), req.Nome)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Instancia criada com sucesso", instancia))
}

func (h *APIHandler) ListarInstancias(c *gin.Context) {
	instancias, err := h.instanciaService.Listar(c.Request.Context())
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	acesso := obterAcessoDashboard(c)
	if acesso.Tipo == "instancia" {
		filtradas := make([]models.Instancia, 0, 1)
		for _, instancia := range instancias {
			if instancia.ID == acesso.InstanciaID {
				filtradas = append(filtradas, instancia)
				break
			}
		}
		instancias = filtradas
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Instancias listadas com sucesso", gin.H{"instancias": instancias}))
}

func (h *APIHandler) BuscarInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	instancia, err := h.instanciaService.Buscar(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Instancia encontrada com sucesso", instancia))
}

func (h *APIHandler) ExcluirInstancia(c *gin.Context) {
	if !h.garantirMaster(c) {
		return
	}
	if err := h.instanciaService.Excluir(c.Request.Context(), c.Param("id")); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Instancia excluida com sucesso", gin.H{"id": c.Param("id")}))
}

func (h *APIHandler) AtualizarTokenInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req atualizarTokenInstanciaRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe o novo token da instancia")
		return
	}
	instancia, err := h.instanciaService.AtualizarToken(c.Request.Context(), c.Param("id"), req.Token)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	acesso := obterAcessoDashboard(c)
	if acesso.Tipo == "instancia" && acesso.InstanciaID == instancia.ID {
		c.SetCookie(h.cfg.DashboardCookieNome, instancia.Token, 86400*15, "/", "", false, true)
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Token da instancia atualizado com sucesso", instancia))
}

func (h *APIHandler) AtualizarHistoricoInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req atualizarHistoricoInstanciaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe a quantidade de dias do historico")
		return
	}
	maxDias := h.cfg.HistoricoMaxDias
	if maxDias <= 0 {
		maxDias = 90
	}
	if req.Dias < 0 || req.Dias > maxDias {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", fmt.Sprintf("Informe historico entre 0 e %d dias", maxDias))
		return
	}
	instancia, err := h.instanciaService.AtualizarHistorico(c.Request.Context(), c.Param("id"), req.Dias, maxDias)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Configuracao de historico atualizada com sucesso", gin.H{"instancia": instancia, "historico_max_dias": maxDias}))
}

func (h *APIHandler) AtualizarProxyInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req atualizarProxyInstanciaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe modo e url do proxy da instancia")
		return
	}
	instancia, err := h.instanciaService.AtualizarProxy(c.Request.Context(), c.Param("id"), req.Modo, req.URL)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Proxy da instancia atualizado com sucesso", instancia))
}

func (h *APIHandler) AtualizarPresencaInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req atualizarPresencaInstanciaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe presenca disponivel ou indisponivel")
		return
	}
	presenca := req.Presenca
	if strings.TrimSpace(presenca) == "" {
		presenca = req.Acao
	}
	if strings.TrimSpace(presenca) == "" {
		presenca = req.Type
	}
	instancia, err := h.instanciaService.AtualizarPresenca(c.Request.Context(), c.Param("id"), presenca)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Presenca da instancia atualizada com sucesso", instancia))
}

func (h *APIHandler) AtualizarConfiguracaoAvancadaInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req atualizarConfiguracaoAvancadaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe as configuracoes avancadas da instancia")
		return
	}
	instancia, err := h.instanciaService.AtualizarConfiguracaoAvancada(c.Request.Context(), c.Param("id"), models.ConfiguracaoAvancadaInstancia{
		ManterOnline:             req.ManterOnline,
		RejeitarChamadas:         req.RejeitarChamadas,
		MensagemRejeitarChamadas: req.MensagemRejeitarChamadas,
		MarcarLidaAutomatico:     req.MarcarLidaAutomatico,
		IgnorarGrupos:            req.IgnorarGrupos,
		IgnorarStatus:            req.IgnorarStatus,
	})
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Configuracoes avancadas atualizadas com sucesso", gin.H{
		"instancia": instancia,
		"configuracao_avancada": models.ConfiguracaoAvancadaInstancia{
			ManterOnline:             instancia.Presenca == models.PresencaDisponivel,
			RejeitarChamadas:         instancia.RejeitarChamadas,
			MensagemRejeitarChamadas: instancia.MensagemRejeitarChamadas,
			MarcarLidaAutomatico:     instancia.MarcarLidaAutomatico,
			IgnorarGrupos:            instancia.IgnorarGrupos,
			IgnorarStatus:            instancia.IgnorarStatus,
		},
	}))
}

func (h *APIHandler) ConectarInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	instancia, qrCode, err := h.instanciaService.Conectar(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Fluxo de conexao iniciado com sucesso", gin.H{"instancia": instancia, "qrcode": qrCode}))
}

func (h *APIHandler) SolicitarCodigoPareamentoInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req solicitarCodigoPareamentoRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Numero) == "" {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe o numero em formato internacional para gerar o pairing code")
		return
	}
	instancia, dados, err := h.instanciaService.SolicitarCodigoPareamento(c.Request.Context(), c.Param("id"), req.Numero)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Pairing code gerado com sucesso", gin.H{"instancia": instancia, "codigo": dados["codigo"], "numero": dados["numero"], "status": dados["status"], "metodo_pareamento": dados["metodo_pareamento"], "atualizado_em": dados["atualizado_em"]}))
}

func (h *APIHandler) DesconectarInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	instancia, err := h.instanciaService.Desconectar(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Instancia desconectada com sucesso", instancia))
}

func (h *APIHandler) StatusInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	status, err := h.instanciaService.Status(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	status["historico_max_dias"] = h.cfg.HistoricoMaxDias
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Status consultado com sucesso", status))
}

func (h *APIHandler) QRCodeInstancia(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	qrCode, err := h.instanciaService.QRCode(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("QR code consultado com sucesso", qrCode))
}

func (h *APIHandler) QRCodeInstanciaImagem(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	qrCode, err := h.instanciaService.QRCode(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	codigo, _ := qrCode["qrcode"].(string)
	if codigo == "" {
		h.responderErro(c, nethttp.StatusNotFound, "qrcode_indisponivel", "QR code indisponivel no momento")
		return
	}
	png, err := qrcode.Encode(codigo, qrcode.Medium, 320)
	if err != nil {
		h.responderErro(c, nethttp.StatusInternalServerError, "erro_qrcode", "Erro ao gerar imagem do QR code")
		return
	}
	c.Data(nethttp.StatusOK, "image/png", png)
}

func (h *APIHandler) BaixarMidiaRecebida(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	midia, err := h.midiaService.Buscar(c.Request.Context(), c.Param("id"), c.Param("midiaId"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	info, err := os.Stat(midia.CaminhoArquivo)
	if err != nil || info.IsDir() {
		h.responderErro(c, nethttp.StatusNotFound, "midia_nao_encontrada", "Arquivo de midia nao encontrado")
		return
	}
	nomeArquivo := strings.TrimSpace(midia.NomeArquivo)
	if nomeArquivo == "" {
		nomeArquivo = filepath.Base(midia.CaminhoArquivo)
	}
	if mimeType := strings.TrimSpace(midia.MimeType); mimeType != "" {
		c.Header("Content-Type", mimeType)
	}
	c.FileAttachment(midia.CaminhoArquivo, nomeArquivo)
}

func (h *APIHandler) ListarWebhooks(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	webhooks, err := h.webhookService.Listar(c.Request.Context(), c.Param("id"))
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Webhooks listados com sucesso", gin.H{"webhooks": webhooks, "eventos_suportados": []string{models.EventoWebhookMensagens, models.EventoWebhookStatus, models.EventoWebhookDigitando, models.EventoWebhookGravandoAudio, models.EventoWebhookRecibos, models.EventoWebhookChamadas}}))
}

func (h *APIHandler) CriarWebhook(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req webhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe nome, url e ao menos um evento")
		return
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	webhook, err := h.webhookService.Criar(c.Request.Context(), c.Param("id"), req.Nome, req.URL, req.Eventos, ativo)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusCreated, models.NovaRespostaSucesso("Webhook criado com sucesso", webhook))
}

func (h *APIHandler) AtualizarWebhook(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	var req webhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe nome, url e ao menos um evento")
		return
	}
	ativo := true
	if req.Ativo != nil {
		ativo = *req.Ativo
	}
	webhook, err := h.webhookService.Atualizar(c.Request.Context(), c.Param("id"), c.Param("webhookId"), req.Nome, req.URL, req.Eventos, ativo)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Webhook atualizado com sucesso", webhook))
}

func (h *APIHandler) ExcluirWebhook(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	if err := h.webhookService.Excluir(c.Request.Context(), c.Param("id"), c.Param("webhookId")); err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Webhook excluido com sucesso", gin.H{"id": c.Param("webhookId")}))
}

func (h *APIHandler) ListarEntregasWebhook(c *gin.Context) {
	if !h.garantirInstancia(c, c.Param("id")) {
		return
	}
	limite, _ := strconv.Atoi(c.DefaultQuery("limite", "50"))
	entregas, err := h.webhookService.ListarEntregas(c.Request.Context(), c.Param("id"), limite)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Entregas de webhook listadas com sucesso", gin.H{"entregas": entregas}))
}

func (h *APIHandler) EnviarTexto(c *gin.Context) {
	var req models.EnvioTextoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: mensagem e numero ou chat_jid")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.mensagemService.EnviarTexto(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Envio de texto preparado com sucesso", resultado))
}

func (h *APIHandler) EnviarPresenca(c *gin.Context) {
	var req models.EnvioPresencaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe acao ou type; numero/chat_jid so e obrigatorio para presenca de chat")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.mensagemService.EnviarPresenca(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Presenca enviada com sucesso", resultado))
}

func (h *APIHandler) MarcarMensagemLida(c *gin.Context) {
	var req models.MarcarLidaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: mensagem_id ou mensagens_id, e numero ou chat_jid")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.mensagemService.MarcarLida(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Mensagem marcada como lida com sucesso", resultado))
}

func (h *APIHandler) EnviarBotoes(c *gin.Context) {
	var req models.EnvioBotoesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: texto ou mensagem, botoes e numero ou chat_jid")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.mensagemService.EnviarBotoes(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	mensagem := "Envio de botoes preparado com sucesso"
	if resultado.Modo == "texto" {
		mensagem = "Botoes enviados como texto"
	} else if resultado.Status == "aceita_pelo_servidor" {
		mensagem = "Botoes aceitos pelo servidor do WhatsApp"
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso(mensagem, resultado))
}

func (h *APIHandler) EnviarLista(c *gin.Context) {
	var req models.EnvioListaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: descricao ou mensagem, botao_texto e opcoes/secoes, com numero ou chat_jid")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.mensagemService.EnviarLista(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	mensagem := "Envio de lista preparado com sucesso"
	if resultado.Modo == "texto" {
		mensagem = "Lista enviada como texto"
	} else if resultado.Status == "aceita_pelo_servidor" {
		mensagem = "Lista aceita pelo servidor do WhatsApp"
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso(mensagem, resultado))
}

func (h *APIHandler) EnviarImagem(c *gin.Context)    { h.enviarMidia(c, "imagem") }
func (h *APIHandler) EnviarAudio(c *gin.Context)     { h.enviarMidia(c, "audio") }
func (h *APIHandler) EnviarDocumento(c *gin.Context) { h.enviarMidia(c, "documento") }

func (h *APIHandler) ListarChamadas(c *gin.Context) {
	instanciaID := c.Param("id")
	if !h.preencherInstanciaDaRequisicao(c, &instanciaID) {
		return
	}
	chamadas, err := h.chamadaService.Listar(c.Request.Context(), instanciaID)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Chamadas listadas com sucesso", gin.H{"chamadas": chamadas}))
}

func (h *APIHandler) IniciarChamada(c *gin.Context) {
	var req models.IniciarChamadaRequest
	if err := decodificarJSONTolerante(c, &req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: numero ou chat_jid")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	resultado, err := h.chamadaService.Iniciar(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Chamada iniciada com sucesso", resultado))
}

func (h *APIHandler) AceitarChamada(c *gin.Context) {
	h.acaoChamada(c, "aceitar")
}

func (h *APIHandler) RejeitarChamada(c *gin.Context) {
	h.acaoChamada(c, "rejeitar")
}

func (h *APIHandler) EncerrarChamada(c *gin.Context) {
	h.acaoChamada(c, "encerrar")
}

func (h *APIHandler) acaoChamada(c *gin.Context, acao string) {
	req := models.AcaoChamadaRequest{
		Instancia: c.Param("id"),
		ChamadaID: c.Param("chamadaId"),
	}
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		_ = decodificarJSONTolerante(c, &req)
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	if strings.TrimSpace(req.ChamadaID) == "" {
		req.ChamadaID = c.Param("chamadaId")
	}
	var (
		resultado models.ResultadoChamada
		err       error
	)
	switch acao {
	case "aceitar":
		resultado, err = h.chamadaService.Aceitar(c.Request.Context(), req)
	case "rejeitar":
		resultado, err = h.chamadaService.Rejeitar(c.Request.Context(), req)
	default:
		resultado, err = h.chamadaService.Encerrar(c.Request.Context(), req)
	}
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Acao de chamada executada com sucesso", resultado))
}

func (h *APIHandler) SinalizarWebRTCChamada(c *gin.Context) {
	req := models.SinalizacaoWebRTCRequest{
		Instancia: c.Param("id"),
		ChamadaID: c.Param("chamadaId"),
	}
	if err := decodificarJSONTolerante(c, &req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: sdp_offer")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}
	if strings.TrimSpace(req.ChamadaID) == "" {
		req.ChamadaID = c.Param("chamadaId")
	}
	resultado, err := h.chamadaService.SinalizarWebRTC(c.Request.Context(), req)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Sinalizacao WebRTC preparada com sucesso", resultado))
}

func (h *APIHandler) enviarMidia(c *gin.Context, tipo string) {
	var req models.EnvioMidiaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Campos obrigatorios: numero ou chat_jid, e arquivo_url, arquivo_base64 ou caminho_local; use grupo=true para grupo")
		return
	}
	if !h.preencherInstanciaDaRequisicao(c, &req.Instancia) {
		return
	}

	var (
		resultado models.ResultadoEnvio
		err       error
	)

	switch tipo {
	case "imagem":
		resultado, err = h.mensagemService.EnviarImagem(c.Request.Context(), req)
	case "audio":
		resultado, err = h.mensagemService.EnviarAudio(c.Request.Context(), req)
	case "documento":
		resultado, err = h.mensagemService.EnviarDocumento(c.Request.Context(), req)
	}

	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Envio preparado com sucesso", resultado))
}

func decodificarJSONTolerante(c *gin.Context, destino interface{}) error {
	corpo, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	texto := strings.TrimSpace(string(corpo))
	if len(texto) >= 2 && texto[0] == '\'' && texto[len(texto)-1] == '\'' {
		texto = strings.TrimSpace(texto[1 : len(texto)-1])
	}
	if texto == "" {
		return io.EOF
	}
	return json.Unmarshal([]byte(texto), destino)
}

func (h *APIHandler) preencherInstanciaDaRequisicao(c *gin.Context, instanciaID *string) bool {
	acesso := obterAcessoDashboard(c)
	if strings.TrimSpace(*instanciaID) == "" {
		if acesso.Tipo == "instancia" && acesso.InstanciaID != "" {
			*instanciaID = acesso.InstanciaID
			return true
		}
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Informe a instancia ou utilize um token de instancia")
		return false
	}
	if err := h.authService.GarantirInstancia(acesso, *instanciaID); err == nil {
		return true
	}
	h.responderErro(c, nethttp.StatusForbidden, "acesso_negado", "Esse token nao possui acesso a esta instancia")
	return false
}

func (h *APIHandler) tratarErro(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInstanciaNaoEncontrada):
		h.responderErro(c, nethttp.StatusNotFound, "instancia_nao_encontrada", "Instancia nao encontrada")
	case errors.Is(err, service.ErrWebhookNaoEncontrado):
		h.responderErro(c, nethttp.StatusNotFound, "webhook_nao_encontrado", "Webhook nao encontrado")
	case errors.Is(err, service.ErrMidiaNaoEncontrada):
		h.responderErro(c, nethttp.StatusNotFound, "midia_nao_encontrada", "Midia nao encontrada")
	case errors.Is(err, service.ErrEntradaInvalida):
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", mensagemEntradaInvalida(err))
	case errors.Is(err, service.ErrDependenciaNaoEncontrada):
		h.responderErro(c, nethttp.StatusNotFound, "dependencia_nao_encontrada", "Dependencia monitorada nao encontrada")
	case errors.Is(err, service.ErrAutorizacaoAtualizacao):
		h.responderErro(c, nethttp.StatusForbidden, "nao_autorizado", "Aplicacao de atualizacao exige token valido")
	case errors.Is(err, service.ErrHistoricoBloqueado):
		h.responderErro(c, nethttp.StatusConflict, "historico_bloqueado", "Defina a importacao de historico antes de conectar a instancia. Desconecte e gere nova conexao para mudar essa configuracao.")
	case errors.Is(err, service.ErrAtualizacaoBloqueada):
		h.responderErro(c, nethttp.StatusForbidden, "atualizacao_bloqueada", "Atualizacao bloqueada por configuracao de seguranca")
	case errors.Is(err, service.ErrNenhumaAtualizacao):
		h.responderErro(c, nethttp.StatusConflict, "nenhuma_atualizacao", "Nao existe atualizacao disponivel para aplicar")
	case errors.Is(err, service.ErrModoAtualizacaoInvalido):
		h.responderErro(c, nethttp.StatusConflict, "modo_atualizacao_invalido", "O sistema esta em modo aviso; troque para modo preparo para gerar artefato")
	default:
		h.responderErro(c, nethttp.StatusInternalServerError, "erro_interno", err.Error())
	}
}

func (h *APIHandler) responderErro(c *gin.Context, status int, codigo, mensagem string) {
	c.JSON(status, models.NovaRespostaErro(codigo, mensagem))
}

func mensagemEntradaInvalida(err error) string {
	mensagem := strings.TrimSpace(err.Error())
	prefixo := service.ErrEntradaInvalida.Error()
	if mensagem == "" || mensagem == prefixo {
		return "Dados informados sao invalidos"
	}
	mensagem = strings.TrimSpace(strings.TrimPrefix(mensagem, prefixo+":"))
	if mensagem == "" {
		return "Dados informados sao invalidos"
	}
	return mensagem
}
