package http

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/dashboard"
	"dyalog-api-go/internal/service"
	mediastorage "dyalog-api-go/internal/storage"
	"dyalog-api-go/internal/store"
	webhookdispatch "dyalog-api-go/internal/webhook"
	"dyalog-api-go/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

type Servidor struct{ engine *gin.Engine }

func NovoServidor(cfg *config.Config) (*Servidor, error) {
	if err := os.MkdirAll(cfg.DiretorioSessoes, 0o755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretorio de sessoes: %w", err)
	}
	if err := os.MkdirAll(cfg.CaminhoArquivosTemp, 0o755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretorio temporario: %w", err)
	}
	if err := os.MkdirAll(cfg.AtualizacaoDiretorioArtefatos, 0o755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretorio de atualizacoes: %w", err)
	}
	storeSQL, err := store.NovoSQLStore(cfg.BancoDriver, cfg.BancoDSN)
	if err != nil {
		return nil, err
	}
	dispatcher := webhookdispatch.NovoDispatcher(
		storeSQL,
		storeSQL,
		time.Duration(cfg.WebhookTimeoutSegundos)*time.Second,
		time.Duration(cfg.WebhookIntervaloBaseSegundos)*time.Second,
		time.Duration(cfg.WebhookRetryMaxDurationHours)*time.Hour,
		time.Duration(cfg.WebhookRetryMaxIntervalMin)*time.Minute,
		cfg.WebhookMaxTentativas,
		cfg.WebhookConcorrencia,
		cfg.WebhookLoteProcessamento,
	)
	midiaUploader, err := mediastorage.NovoMidiaUploader(
		cfg.MidiaStorageDriver,
		cfg.MidiaStorageSupabaseURL,
		cfg.MidiaStorageSupabaseKey,
		cfg.MidiaStorageSupabaseBucket,
		cfg.MidiaStoragePublicBaseURL,
	)
	if err != nil {
		return nil, err
	}
	gerenciador := whatsapp.NovoGerenciadorInstancias(
		cfg.DiretorioSessoes,
		cfg.CaminhoArquivosTemp,
		cfg.BaseURL,
		cfg.NomeDispositivoSessao,
		cfg.TipoClienteSessao,
		cfg.NomePareamentoSessao,
		cfg.WhatsAppLogLevel,
		dispatcher,
		storeSQL,
		midiaUploader,
		storeSQL,
		storeSQL,
	)
	instanciaService := service.NovoInstanciaService(storeSQL, gerenciador)
	mensagemService := service.NovoMensagemService(storeSQL, gerenciador)
	chamadaService := service.NovoChamadaService(storeSQL, gerenciador)
	midiaService := service.NovoMidiaService(storeSQL)
	webhookService := service.NovoWebhookService(storeSQL, storeSQL, storeSQL)
	sistemaService := service.NovoSistemaService(cfg, storeSQL)
	authService := service.NovoAuthService(cfg.DashboardMasterToken, storeSQL)
	dispatcher.Iniciar(context.Background())
	registrarRecuperacaoWebhook(context.Background(), cfg, storeSQL, gerenciador)
	iniciarHeartbeatRuntime(context.Background(), storeSQL, time.Duration(cfg.HeartbeatIntervaloSegundos)*time.Second)
	sistemaService.IniciarMonitoramento(context.Background())
	instanciaService.RestaurarSessoes(context.Background())
	if cfg.Ambiente == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	aplicarLoggerHTTP(engine, cfg.HTTPLogMode)
	engine.Use(gin.Recovery())
	engine.Static("/static", "./static")
	apiHandler := NovoAPIHandler(cfg, instanciaService, mensagemService, chamadaService, midiaService, webhookService, sistemaService, authService)
	dashboardHandler := dashboard.NovoHandler(cfg, authService)
	registrarRotas(engine, cfg, apiHandler, dashboardHandler, authService)
	return &Servidor{engine: engine}, nil
}

func (s *Servidor) Engine() *gin.Engine { return s.engine }

func aplicarLoggerHTTP(engine *gin.Engine, modo string) {
	switch modo {
	case "todos":
		engine.Use(gin.Logger())
	case "erros":
		engine.Use(loggerHTTPResumido(400, 0))
	case "desligado":
		return
	default:
		engine.Use(loggerHTTPResumido(500, 5*time.Second))
	}
}

func loggerHTTPResumido(statusMinimo int, lentidaoMinima time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		inicio := time.Now()
		c.Next()
		latencia := time.Since(inicio)
		status := c.Writer.Status()
		if status < statusMinimo && (lentidaoMinima <= 0 || latencia < lentidaoMinima) {
			return
		}
		log.Printf("HTTP %d %s %s %s em %s", status, c.Request.Method, c.FullPath(), c.ClientIP(), latencia.Round(time.Millisecond))
	}
}

func iniciarHeartbeatRuntime(ctx context.Context, runtimeStore store.RuntimeStore, intervalo time.Duration) {
	if runtimeStore == nil {
		return
	}
	if intervalo < 5*time.Second {
		intervalo = 30 * time.Second
	}
	_ = runtimeStore.AtualizarHeartbeat(ctx, "api", time.Now().UTC())
	go func() {
		ticker := time.NewTicker(intervalo)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case agora := <-ticker.C:
				if err := runtimeStore.AtualizarHeartbeat(context.Background(), "api", agora.UTC()); err != nil {
					fmt.Printf("erro ao atualizar heartbeat da API: %v\n", err)
				}
			}
		}
	}()
}

func registrarRecuperacaoWebhook(ctx context.Context, cfg *config.Config, runtimeStore store.RuntimeStore, gerenciador *whatsapp.GerenciadorInstancias) {
	if cfg == nil || runtimeStore == nil || gerenciador == nil || !cfg.RecuperacaoWebhookHabilitada {
		return
	}
	ultimoHeartbeat, encontrado, err := runtimeStore.ObterHeartbeat(ctx, "api")
	if err != nil {
		fmt.Printf("erro ao consultar heartbeat anterior da API: %v\n", err)
		return
	}
	if !encontrado || ultimoHeartbeat.IsZero() {
		return
	}
	agora := time.Now().UTC()
	intervalo := time.Duration(cfg.HeartbeatIntervaloSegundos) * time.Second
	if intervalo < 5*time.Second {
		intervalo = 30 * time.Second
	}
	if agora.Sub(ultimoHeartbeat) <= intervalo+15*time.Second {
		return
	}
	margem := time.Duration(cfg.RecuperacaoMargemSegundos) * time.Second
	inicio := ultimoHeartbeat.Add(-margem)
	fim := agora.Add(margem)
	instancias, err := runtimeStore.(store.InstanciaStore).Listar(ctx)
	if err != nil {
		fmt.Printf("erro ao listar instancias para recuperacao de webhook: %v\n", err)
		return
	}
	for _, instancia := range instancias {
		gerenciador.RegistrarJanelaRecuperacao(instancia.ID, inicio, fim, cfg.RecuperacaoHistoricoMensagens)
	}
	fmt.Printf("recuperacao de webhook agendada: API ficou sem heartbeat de %s a %s; janela aplicada em %d instancias\n", inicio.Format(time.RFC3339), fim.Format(time.RFC3339), len(instancias))
}
