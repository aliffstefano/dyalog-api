package http

import (
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"
)

func (h *APIHandler) UsarAuditoria(s *service.AuditoriaService) { h.auditoria = s }

// acaoAuditada descreve o que gravar quando a rota termina com sucesso.
type acaoAuditada struct {
	tipo, acao, descricao, severidade string
}

// acoesAuditadas: rotas que mudam algo e entram na auditoria. Envio de
// mensagem fica de fora (volume alto; o chat e o uso ja mostram).
var acoesAuditadas = map[string]acaoAuditada{
	"POST /api/v1/instancias":                                 {models.AuditoriaAcao, "instancia.criada", "Instancia criada", models.SeveridadeSucesso},
	"POST /api/v1/meta/cadastro":                              {models.AuditoriaAcao, "instancia.criada_meta", "Instancia criada pelo cadastro da Meta", models.SeveridadeSucesso},
	"DELETE /api/v1/instancias/:id":                           {models.AuditoriaAcao, "instancia.excluida", "Instancia excluida", models.SeveridadeAviso},
	"PUT /api/v1/instancias/:id/token":                        {models.AuditoriaConfig, "instancia.token", "Token de acesso trocado", models.SeveridadeAviso},
	"PUT /api/v1/instancias/:id/historico":                    {models.AuditoriaConfig, "instancia.historico", "Historico ao conectar alterado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/proxy":                        {models.AuditoriaConfig, "instancia.proxy", "Proxy alterado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/presenca":                     {models.AuditoriaConfig, "instancia.presenca", "Presenca alterada", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/avancado":                     {models.AuditoriaConfig, "instancia.comportamento", "Comportamento alterado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/meta":                         {models.AuditoriaConfig, "instancia.meta", "Credenciais da Meta atualizadas", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/conta":                        {models.AuditoriaConfig, "instancia.conta", "Instancia movida de conta", models.SeveridadeAviso},
	"POST /api/v1/instancias/:id/conectar":                    {models.AuditoriaConexao, "instancia.conectar", "Tentando conectar", models.SeveridadeInfo},
	"POST /api/v1/instancias/:id/pairing-code":                {models.AuditoriaConexao, "instancia.codigo_pareamento", "Codigo de pareamento solicitado", models.SeveridadeInfo},
	"POST /api/v1/instancias/:id/desconectar":                 {models.AuditoriaConexao, "instancia.desconectar", "Desconectada pelo painel/API", models.SeveridadeAviso},
	"POST /api/v1/instancias/:id/webhooks":                    {models.AuditoriaWebhook, "webhook.criado", "Webhook adicionado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/webhooks/:webhookId":          {models.AuditoriaWebhook, "webhook.alterado", "Webhook alterado", models.SeveridadeInfo},
	"DELETE /api/v1/instancias/:id/webhooks/:webhookId":       {models.AuditoriaWebhook, "webhook.excluido", "Webhook excluido", models.SeveridadeAviso},
	"PUT /api/v1/instancias/:id/perfil":                       {models.AuditoriaPerfil, "perfil.alterado", "Perfil alterado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/perfil/foto":                  {models.AuditoriaPerfil, "perfil.foto", "Foto do perfil trocada", models.SeveridadeInfo},
	"DELETE /api/v1/instancias/:id/perfil/foto":               {models.AuditoriaPerfil, "perfil.foto_removida", "Foto do perfil removida", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/privacidade":                  {models.AuditoriaPerfil, "perfil.privacidade", "Privacidade alterada", models.SeveridadeInfo},
	"POST /api/v1/instancias/:id/contatos/bloqueio":           {models.AuditoriaContato, "contato.bloqueio", "Bloqueio de contato alterado", models.SeveridadeAviso},
	"POST /api/v1/equipe/convites":                            {models.AuditoriaEquipe, "equipe.convite", "Convite enviado", models.SeveridadeInfo},
	"DELETE /api/v1/equipe/convites/:conviteId":               {models.AuditoriaEquipe, "equipe.convite_cancelado", "Convite cancelado", models.SeveridadeInfo},
	"PUT /api/v1/equipe/usuarios/:usuarioId":                  {models.AuditoriaEquipe, "equipe.membro_alterado", "Papel ou acesso de um membro alterado", models.SeveridadeInfo},
	"DELETE /api/v1/equipe/usuarios/:usuarioId":               {models.AuditoriaEquipe, "equipe.membro_removido", "Membro removido da equipe", models.SeveridadeAviso},
	"POST /api/v1/instancias/:id/grupos":                      {models.AuditoriaGrupo, "grupo.criado", "Grupo criado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/grupos/:grupo":                {models.AuditoriaGrupo, "grupo.alterado", "Grupo alterado", models.SeveridadeInfo},
	"PUT /api/v1/instancias/:id/grupos/:grupo/foto":           {models.AuditoriaGrupo, "grupo.foto", "Foto do grupo trocada", models.SeveridadeInfo},
	"DELETE /api/v1/instancias/:id/grupos/:grupo/foto":        {models.AuditoriaGrupo, "grupo.foto_removida", "Foto do grupo removida", models.SeveridadeInfo},
	"POST /api/v1/instancias/:id/grupos/:grupo/participantes": {models.AuditoriaGrupo, "grupo.participantes", "Participantes do grupo alterados", models.SeveridadeInfo},
	"POST /api/v1/instancias/:id/grupos/:grupo/sair":          {models.AuditoriaGrupo, "grupo.saiu", "Saiu do grupo", models.SeveridadeAviso},
	"PUT /api/v1/auth/senha":                                  {models.AuditoriaAcesso, "acesso.senha", "Senha trocada", models.SeveridadeInfo},
	"PUT /api/v1/contas/:contaId":                             {models.AuditoriaConfig, "conta.alterada", "Conta alterada (plano, limite ou status)", models.SeveridadeInfo},
}

const (
	chaveAuditoriaDescricao = "auditoria_descricao"
	chaveAuditoriaInstancia = "auditoria_instancia"
	chaveAuditoriaRota      = "auditoria_rota"
)

// descreverAuditoria troca a descricao padrao da rota por uma mais especifica.
func (h *APIHandler) descreverAuditoria(c *gin.Context, descricao string) {
	c.Set(chaveAuditoriaDescricao, descricao)
}

// marcarInstanciaAuditoria informa a instancia quando a rota nao tem :id
// (ex.: criacao).
func marcarInstanciaAuditoria(c *gin.Context, instanciaID string) {
	c.Set(chaveAuditoriaInstancia, instanciaID)
}

// registrarAuditoriaLeitura faz uma rota GET entrar na auditoria (exportacoes).
func (h *APIHandler) registrarAuditoriaLeitura(c *gin.Context, tipo, acao string) {
	c.Set(chaveAuditoriaRota, acaoAuditada{tipo: tipo, acao: acao, severidade: models.SeveridadeInfo})
}

// middlewareAuditoria grava as acoes da tabela acima que terminaram com
// sucesso. Roda so na replica que executou o handler (o proxy de replicas
// aborta antes quando encaminha).
func (h *APIHandler) middlewareAuditoria() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if h.auditoria == nil || c.IsAborted() || c.Writer.Status() >= 400 {
			return
		}
		acao, ok := acoesAuditadas[c.Request.Method+" "+c.FullPath()]
		if valor, existe := c.Get(chaveAuditoriaRota); existe {
			acao, ok = valor.(acaoAuditada), true
		}
		if !ok {
			return
		}
		descricao := acao.descricao
		if valor := c.GetString(chaveAuditoriaDescricao); valor != "" {
			descricao = valor
		}
		instanciaID := c.Param("id")
		if valor := c.GetString(chaveAuditoriaInstancia); valor != "" {
			instanciaID = valor
		}
		acesso := obterAcessoDashboard(c)
		if contaID := c.Param("contaId"); contaID != "" && instanciaID == "" {
			acesso.ContaID = contaID
		}
		h.auditoria.RegistrarAcao(c.Request.Context(), acesso, c.ClientIP(), instanciaID, acao.tipo, acao.acao, descricao, acao.severidade, "")
	}
}

// filtroAuditoriaDaQuery le tipo, severidade, ator, de/ate (AAAA-MM-DD, ate
// inclusivo), limite e deslocamento.
func filtroAuditoriaDaQuery(c *gin.Context) (models.FiltroAuditoria, bool) {
	filtro := models.FiltroAuditoria{
		Tipo:       strings.TrimSpace(c.Query("tipo")),
		Severidade: strings.TrimSpace(c.Query("severidade")),
		Ator:       c.Query("ator"),
	}
	filtro.Limite, _ = strconv.Atoi(c.DefaultQuery("limite", "50"))
	filtro.Deslocamento, _ = strconv.Atoi(c.DefaultQuery("deslocamento", "0"))
	local := time.Local
	if de := strings.TrimSpace(c.Query("de")); de != "" {
		t, err := time.ParseInLocation("2006-01-02", de, local)
		if err != nil {
			return filtro, false
		}
		filtro.De = t
	}
	if ate := strings.TrimSpace(c.Query("ate")); ate != "" {
		t, err := time.ParseInLocation("2006-01-02", ate, local)
		if err != nil {
			return filtro, false
		}
		filtro.Ate = t.AddDate(0, 0, 1)
	}
	return filtro, true
}

func (h *APIHandler) ListarAuditoriaInstancia(c *gin.Context) {
	id := c.Param("id")
	if !h.garantirInstancia(c, id) {
		return
	}
	filtro, ok := filtroAuditoriaDaQuery(c)
	if !ok {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Datas no formato AAAA-MM-DD")
		return
	}
	filtro.InstanciaID = id
	h.responderAuditoria(c, filtro)
}

// ListarAuditoria e o historico da conta inteira (ou de tudo, para o
// superadmin). Aceita instancia_id para filtrar.
func (h *APIHandler) ListarAuditoria(c *gin.Context) {
	filtro, ok := filtroAuditoriaDaQuery(c)
	if !ok {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "Datas no formato AAAA-MM-DD")
		return
	}
	if id := strings.TrimSpace(c.Query("instancia_id")); id != "" {
		if !h.garantirInstancia(c, id) {
			return
		}
		filtro.InstanciaID = id
	}
	h.responderAuditoria(c, filtro)
}

func (h *APIHandler) responderAuditoria(c *gin.Context, filtro models.FiltroAuditoria) {
	if h.auditoria == nil {
		c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Auditoria", models.PaginaAuditoria{Registros: []models.RegistroAuditoria{}}))
		return
	}
	pagina, err := h.auditoria.Listar(c.Request.Context(), obterAcessoDashboard(c), filtro)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Auditoria", pagina))
}
