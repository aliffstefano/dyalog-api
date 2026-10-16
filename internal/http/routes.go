package http

import (
	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/dashboard"
	"dyalog-api-go/internal/service"
	"dyalog-api-go/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

func registrarRotas(engine *gin.Engine, cfg *config.Config, apiHandler *APIHandler, dashboardHandler *dashboard.Handler, authService *service.AuthService, gerenciador *whatsapp.GerenciadorInstancias) {
	engine.GET("/", dashboardHandler.PaginaInicial)
	engine.GET("/docs", dashboardHandler.Docs)
	engine.GET("/convite", dashboardHandler.PaginaConvite)
	engine.GET("/senha/redefinir", dashboardHandler.PaginaRedefinirSenha)

	auth := middlewareAutenticacaoAPI(cfg, authService, apiHandler.limiteFalhasAuth)
	proxyReplica := middlewareProxyReplica(gerenciador)

	// Webhooks da API oficial: publicos, a Meta nao manda o nosso token. A
	// seguranca vem do verify_token (GET) e da assinatura com o App Secret (POST).
	engine.GET("/webhook/meta", apiHandler.VerificarWebhookMetaApp)
	engine.POST("/webhook/meta", apiHandler.ReceberWebhookMetaApp)
	engine.GET("/webhook/meta/:id", apiHandler.VerificarWebhookMeta)
	engine.POST("/webhook/meta/:id", apiHandler.ReceberWebhookMeta)

	engine.POST("/api/v1/auth/login", apiHandler.LoginDashboard)
	engine.POST("/api/v1/auth/logout", apiHandler.LogoutDashboard)
	engine.GET("/api/v1/auth/convite", apiHandler.InfoConvite)
	engine.POST("/api/v1/auth/convite/aceitar", apiHandler.AceitarConvite)
	engine.POST("/api/v1/auth/senha/esqueci", apiHandler.EsqueciSenha)
	engine.POST("/api/v1/auth/senha/redefinir", apiHandler.RedefinirSenha)
	engine.POST("/user/presence", auth, proxyReplica, apiHandler.EnviarPresenca)

	api := engine.Group("/api/v1")
	{
		api.GET("/saude", apiHandler.Saude)
		api.GET("/docs", apiHandler.Documentacao)
	}

	apiAuth := engine.Group("/api/v1", auth, proxyReplica, apiHandler.middlewareAuditoria())
	{
		apiAuth.GET("/auth/sessao", apiHandler.SessaoDashboard)
		apiAuth.PUT("/auth/senha", apiHandler.TrocarSenha)

		apiAuth.GET("/equipe", apiHandler.Equipe)
		apiAuth.POST("/equipe/convites", apiHandler.ConvidarEquipe)
		apiAuth.DELETE("/equipe/convites/:conviteId", apiHandler.CancelarConvite)
		apiAuth.PUT("/equipe/usuarios/:usuarioId", apiHandler.AtualizarMembro)
		apiAuth.DELETE("/equipe/usuarios/:usuarioId", apiHandler.RemoverMembro)

		apiAuth.GET("/contas", apiHandler.ListarContas)
		apiAuth.POST("/contas", apiHandler.CriarConta)
		apiAuth.PUT("/contas/:contaId", apiHandler.AtualizarConta)
		apiAuth.GET("/planos", apiHandler.ListarPlanos)
		apiAuth.POST("/planos", apiHandler.SalvarPlano)
		apiAuth.PUT("/planos/:planoId", apiHandler.SalvarPlano)
		apiAuth.GET("/administradores", apiHandler.Administradores)
		apiAuth.POST("/administradores/convites", apiHandler.ConvidarAdministrador)
		apiAuth.PUT("/instancias/:id/conta", apiHandler.MoverInstanciaConta)

		apiAuth.GET("/sistema/versao", apiHandler.VersaoSistema)
		apiAuth.GET("/sistema/atualizacoes", apiHandler.AtualizacoesSistema)
		apiAuth.POST("/sistema/atualizacoes/verificar", apiHandler.VerificarAtualizacoesSistema)
		apiAuth.POST("/sistema/atualizacoes/aplicar", apiHandler.AplicarAtualizacaoSistema)
		apiAuth.GET("/sistema/proxy", apiHandler.ProxySistema)
		apiAuth.PUT("/sistema/proxy", apiHandler.AtualizarProxySistema)

		apiAuth.POST("/instancias", apiHandler.CriarInstancia)
		apiAuth.GET("/meta/cadastro", apiHandler.ConfigCadastroMeta)
		apiAuth.POST("/meta/cadastro", apiHandler.CadastrarViaMeta)
		apiAuth.GET("/instancias", apiHandler.ListarInstancias)
		apiAuth.GET("/instancias/:id", apiHandler.BuscarInstancia)
		apiAuth.DELETE("/instancias/:id", apiHandler.ExcluirInstancia)
		apiAuth.PUT("/instancias/:id/token", apiHandler.AtualizarTokenInstancia)
		apiAuth.PUT("/instancias/:id/historico", apiHandler.AtualizarHistoricoInstancia)
		apiAuth.PUT("/instancias/:id/proxy", apiHandler.AtualizarProxyInstancia)
		apiAuth.PUT("/instancias/:id/presenca", apiHandler.AtualizarPresencaInstancia)
		apiAuth.PUT("/instancias/:id/avancado", apiHandler.AtualizarConfiguracaoAvancadaInstancia)
		apiAuth.POST("/instancias/:id/conectar", apiHandler.ConectarInstancia)
		apiAuth.POST("/instancias/:id/pairing-code", apiHandler.SolicitarCodigoPareamentoInstancia)
		apiAuth.POST("/instancias/:id/desconectar", apiHandler.DesconectarInstancia)
		apiAuth.PUT("/instancias/:id/meta", apiHandler.ConfigurarMetaInstancia)
		apiAuth.GET("/instancias/:id/meta/modelos", apiHandler.ListarModelosMeta)
		apiAuth.GET("/instancias/:id/status", apiHandler.StatusInstancia)
		apiAuth.GET("/instancias/:id/qrcode", apiHandler.QRCodeInstancia)
		apiAuth.GET("/instancias/:id/qrcode/imagem", apiHandler.QRCodeInstanciaImagem)
		apiAuth.GET("/instancias/:id/contatos/avatar", apiHandler.AvatarContato)
		apiAuth.GET("/instancias/:id/midia/:midiaId", apiHandler.BaixarMidiaRecebida)
		apiAuth.GET("/instancias/:id/midias/:midiaId", apiHandler.BaixarMidiaRecebida)

		apiAuth.GET("/instancias/:id/webhooks", apiHandler.ListarWebhooks)
		apiAuth.POST("/instancias/:id/webhooks", apiHandler.CriarWebhook)
		apiAuth.PUT("/instancias/:id/webhooks/:webhookId", apiHandler.AtualizarWebhook)
		apiAuth.DELETE("/instancias/:id/webhooks/:webhookId", apiHandler.ExcluirWebhook)
		apiAuth.GET("/instancias/:id/webhook-entregas", apiHandler.ListarEntregasWebhook)
		apiAuth.GET("/instancias/:id/uso", apiHandler.UsoInstancia)
		apiAuth.GET("/uso", apiHandler.UsoGeral)

		apiAuth.GET("/instancias/:id/perfil", apiHandler.PerfilWhatsApp)
		apiAuth.PUT("/instancias/:id/perfil", apiHandler.AtualizarPerfilWhatsApp)
		apiAuth.PUT("/instancias/:id/perfil/foto", apiHandler.AtualizarFotoPerfilWhatsApp)
		apiAuth.DELETE("/instancias/:id/perfil/foto", apiHandler.RemoverFotoPerfilWhatsApp)
		apiAuth.PUT("/instancias/:id/privacidade", apiHandler.AtualizarPrivacidadeWhatsApp)
		apiAuth.GET("/instancias/:id/contatos", apiHandler.ListarContatosWhatsApp)
		apiAuth.GET("/instancias/:id/contatos/exportar", apiHandler.ExportarContatosWhatsApp)
		apiAuth.POST("/instancias/:id/contatos/verificar", apiHandler.VerificarNumerosWhatsApp)
		apiAuth.POST("/instancias/:id/contatos/bloqueio", apiHandler.BloquearContatoWhatsApp)
		apiAuth.GET("/instancias/:id/grupos", apiHandler.ListarGruposWhatsApp)
		apiAuth.POST("/instancias/:id/grupos", apiHandler.CriarGrupoWhatsApp)
		apiAuth.GET("/instancias/:id/grupos/:grupo", apiHandler.GrupoWhatsApp)
		apiAuth.PUT("/instancias/:id/grupos/:grupo", apiHandler.AtualizarGrupoWhatsApp)
		apiAuth.PUT("/instancias/:id/grupos/:grupo/foto", apiHandler.FotoGrupoWhatsApp)
		apiAuth.DELETE("/instancias/:id/grupos/:grupo/foto", apiHandler.FotoGrupoWhatsApp)
		apiAuth.GET("/instancias/:id/grupos/:grupo/convite", apiHandler.ConviteGrupoWhatsApp)
		apiAuth.POST("/instancias/:id/grupos/:grupo/participantes", apiHandler.ParticipantesGrupoWhatsApp)
		apiAuth.POST("/instancias/:id/grupos/:grupo/sair", apiHandler.SairGrupoWhatsApp)
		apiAuth.GET("/instancias/:id/auditoria", apiHandler.ListarAuditoriaInstancia)
		apiAuth.GET("/auditoria", apiHandler.ListarAuditoria)

		apiAuth.GET("/instancias/:id/chat/conversas", apiHandler.ListarConversasChat)
		apiAuth.GET("/instancias/:id/chat/mensagens", apiHandler.ListarMensagensChat)

		apiAuth.GET("/instancias/:id/chamadas", apiHandler.ListarChamadas)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/aceitar", apiHandler.AceitarChamada)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/rejeitar", apiHandler.RejeitarChamada)
		apiAuth.DELETE("/instancias/:id/chamadas/:chamadaId", apiHandler.EncerrarChamada)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/webrtc", apiHandler.SinalizarWebRTCChamada)

		apiAuth.POST("/batepapo/enviar/texto", apiHandler.EnviarTexto)
		apiAuth.POST("/batepapo/editar/texto", apiHandler.EditarTexto)
		apiAuth.POST("/batepapo/apagar", apiHandler.ApagarMensagem)
		apiAuth.POST("/batepapo/reagir", apiHandler.ReagirMensagem)
		apiAuth.POST("/batepapo/enviar/presenca", apiHandler.EnviarPresenca)
		apiAuth.POST("/user/presence", apiHandler.EnviarPresenca)
		apiAuth.POST("/batepapo/marcar-lida", apiHandler.MarcarMensagemLida)
		apiAuth.POST("/batepapo/enviar/botoes", apiHandler.EnviarBotoes)
		apiAuth.POST("/batepapo/enviar/lista", apiHandler.EnviarLista)
		apiAuth.POST("/batepapo/enviar/enquete", apiHandler.EnviarEnquete)
		apiAuth.POST("/batepapo/enviar/cobranca-pix", apiHandler.EnviarCobrancaPix)
		apiAuth.POST("/batepapo/enviar/localizacao", apiHandler.EnviarLocalizacao)
		apiAuth.POST("/batepapo/enviar/contato", apiHandler.EnviarContato)
		apiAuth.POST("/batepapo/enviar/imagem", apiHandler.EnviarImagem)
		apiAuth.POST("/batepapo/enviar/audio", apiHandler.EnviarAudio)
		apiAuth.POST("/batepapo/enviar/documento", apiHandler.EnviarDocumento)
		apiAuth.POST("/batepapo/enviar/figurinha", apiHandler.EnviarFigurinha)
		apiAuth.POST("/batepapo/enviar/evento", apiHandler.EnviarEvento)
		apiAuth.POST("/batepapo/enviar/template", apiHandler.EnviarTemplate)
		apiAuth.POST("/batepapo/enviar/carrossel", apiHandler.EnviarCarrossel)
		apiAuth.POST("/status/postar", apiHandler.PostarStatus)

		apiAuth.GET("/chamadas/ice", apiHandler.ServidoresICE)
		apiAuth.POST("/chamadas/iniciar", apiHandler.IniciarChamada)
		apiAuth.POST("/chamadas/link", apiHandler.CriarLinkChamada)
		apiAuth.POST("/chamadas/:chamadaId/aceitar", apiHandler.AceitarChamada)
		apiAuth.POST("/chamadas/:chamadaId/rejeitar", apiHandler.RejeitarChamada)
		apiAuth.DELETE("/chamadas/:chamadaId", apiHandler.EncerrarChamada)
		apiAuth.POST("/chamadas/:chamadaId/webrtc", apiHandler.SinalizarWebRTCChamada)
	}
}
