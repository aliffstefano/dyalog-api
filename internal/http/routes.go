package http

import (
	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/dashboard"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

func registrarRotas(engine *gin.Engine, cfg *config.Config, apiHandler *APIHandler, dashboardHandler *dashboard.Handler, authService *service.AuthService) {
	engine.GET("/", dashboardHandler.PaginaInicial)
	engine.GET("/docs", dashboardHandler.Docs)

	auth := middlewareAutenticacaoAPI(cfg, authService)

	engine.POST("/api/v1/auth/login", apiHandler.LoginDashboard)
	engine.POST("/api/v1/auth/logout", apiHandler.LogoutDashboard)
	engine.POST("/user/presence", auth, apiHandler.EnviarPresenca)

	api := engine.Group("/api/v1")
	{
		api.GET("/saude", apiHandler.Saude)
		api.GET("/docs", apiHandler.Documentacao)
	}

	apiAuth := engine.Group("/api/v1", auth)
	{
		apiAuth.GET("/auth/sessao", apiHandler.SessaoDashboard)

		apiAuth.GET("/sistema/versao", apiHandler.VersaoSistema)
		apiAuth.GET("/sistema/atualizacoes", apiHandler.AtualizacoesSistema)
		apiAuth.POST("/sistema/atualizacoes/verificar", apiHandler.VerificarAtualizacoesSistema)
		apiAuth.POST("/sistema/atualizacoes/aplicar", apiHandler.AplicarAtualizacaoSistema)
		apiAuth.GET("/sistema/proxy", apiHandler.ProxySistema)
		apiAuth.PUT("/sistema/proxy", apiHandler.AtualizarProxySistema)

		apiAuth.POST("/instancias", apiHandler.CriarInstancia)
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
		apiAuth.GET("/instancias/:id/status", apiHandler.StatusInstancia)
		apiAuth.GET("/instancias/:id/qrcode", apiHandler.QRCodeInstancia)
		apiAuth.GET("/instancias/:id/qrcode/imagem", apiHandler.QRCodeInstanciaImagem)
		apiAuth.GET("/instancias/:id/midia/:midiaId", apiHandler.BaixarMidiaRecebida)
		apiAuth.GET("/instancias/:id/midias/:midiaId", apiHandler.BaixarMidiaRecebida)

		apiAuth.GET("/instancias/:id/webhooks", apiHandler.ListarWebhooks)
		apiAuth.POST("/instancias/:id/webhooks", apiHandler.CriarWebhook)
		apiAuth.PUT("/instancias/:id/webhooks/:webhookId", apiHandler.AtualizarWebhook)
		apiAuth.DELETE("/instancias/:id/webhooks/:webhookId", apiHandler.ExcluirWebhook)
		apiAuth.GET("/instancias/:id/webhook-entregas", apiHandler.ListarEntregasWebhook)

		apiAuth.GET("/instancias/:id/chamadas", apiHandler.ListarChamadas)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/aceitar", apiHandler.AceitarChamada)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/rejeitar", apiHandler.RejeitarChamada)
		apiAuth.DELETE("/instancias/:id/chamadas/:chamadaId", apiHandler.EncerrarChamada)
		apiAuth.POST("/instancias/:id/chamadas/:chamadaId/webrtc", apiHandler.SinalizarWebRTCChamada)

		apiAuth.POST("/batepapo/enviar/texto", apiHandler.EnviarTexto)
		apiAuth.POST("/batepapo/enviar/presenca", apiHandler.EnviarPresenca)
		apiAuth.POST("/user/presence", apiHandler.EnviarPresenca)
		apiAuth.POST("/batepapo/marcar-lida", apiHandler.MarcarMensagemLida)
		apiAuth.POST("/batepapo/enviar/botoes", apiHandler.EnviarBotoes)
		apiAuth.POST("/batepapo/enviar/lista", apiHandler.EnviarLista)
		apiAuth.POST("/batepapo/enviar/imagem", apiHandler.EnviarImagem)
		apiAuth.POST("/batepapo/enviar/audio", apiHandler.EnviarAudio)
		apiAuth.POST("/batepapo/enviar/documento", apiHandler.EnviarDocumento)

		apiAuth.POST("/chamadas/iniciar", apiHandler.IniciarChamada)
		apiAuth.POST("/chamadas/:chamadaId/aceitar", apiHandler.AceitarChamada)
		apiAuth.POST("/chamadas/:chamadaId/rejeitar", apiHandler.RejeitarChamada)
		apiAuth.DELETE("/chamadas/:chamadaId", apiHandler.EncerrarChamada)
		apiAuth.POST("/chamadas/:chamadaId/webrtc", apiHandler.SinalizarWebRTCChamada)
	}
}
