package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"dyalog-api-go/internal/buildinfo"

	"github.com/joho/godotenv"
)

type Config struct {
	Porta                         string
	BancoDriver                   string
	BancoDSN                      string
	WebhookURL                    string
	DiretorioSessoes              string
	NomeDispositivoSessao         string
	TipoClienteSessao             string
	NomePareamentoSessao          string
	Ambiente                      string
	NomeAplicacao                 string
	CaminhoArquivosTemp           string
	HTTPLogMode                   string
	WhatsAppLogLevel              string
	VersaoAplicacao               string
	CommitAplicacao               string
	DataBuildAplicacao            string
	AtualizacaoMonitoramento      bool
	AtualizacaoModo               string
	AtualizacaoJanelaInicio       string
	AtualizacaoJanelaFim          string
	AtualizacaoIntervaloMinutos   int
	AtualizacaoAplicarHabilitado  bool
	AtualizacaoAplicarToken       string
	AtualizacaoProxyURL           string
	AtualizacaoDiretorioArtefatos string
	DashboardMasterToken          string
	DashboardCookieNome           string
	BaseURL                       string
	HistoricoMaxDias              int
	WebhookMaxTentativas          int
	WebhookIntervaloBaseSegundos  int
	WebhookRetryMaxDurationHours  int
	WebhookRetryMaxIntervalMin    int
	WebhookLoteProcessamento      int
	WebhookTimeoutSegundos        int
	WebhookConcorrencia           int
	HeartbeatIntervaloSegundos    int
	RecuperacaoWebhookHabilitada  bool
	RecuperacaoMargemSegundos     int
	RecuperacaoHistoricoMensagens int
	MidiaStorageDriver            string
	MidiaStorageSupabaseURL       string
	MidiaStorageSupabaseKey       string
	MidiaStorageSupabaseBucket    string
	MidiaStoragePublicBaseURL     string
}

func Carregar() (*Config, error) {
	_ = godotenv.Load()

	bancoDriver, bancoDSN := configurarBanco()
	cfg := &Config{
		Porta:                         obter("APP_PORT", "8080"),
		BancoDriver:                   bancoDriver,
		BancoDSN:                      bancoDSN,
		WebhookURL:                    os.Getenv("WEBHOOK_URL"),
		DiretorioSessoes:              obter("SESSION_STORAGE_DIR", "./data/sessoes"),
		NomeDispositivoSessao:         obter("SESSION_DEVICE_NAME", "DyalogAPI"),
		TipoClienteSessao:             strings.ToLower(obter("SESSION_CLIENT_TYPE", "chrome")),
		NomePareamentoSessao:          obter("SESSION_PAIRING_DISPLAY_NAME", "Chrome (Windows)"),
		Ambiente:                      obter("APP_ENV", "development"),
		NomeAplicacao:                 obter("APP_NAME", "Dyalog API GO"),
		CaminhoArquivosTemp:           obter("TEMP_FILES_DIR", "./data/temp"),
		HTTPLogMode:                   strings.ToLower(obter("HTTP_LOG_MODE", "falhas")),
		WhatsAppLogLevel:              strings.ToUpper(obter("WHATSAPP_LOG_LEVEL", "ERROR")),
		VersaoAplicacao:               obter("APP_VERSION", buildinfo.Version),
		CommitAplicacao:               obter("APP_COMMIT", buildinfo.Commit),
		DataBuildAplicacao:            obter("APP_BUILD_DATE", buildinfo.BuildDate),
		AtualizacaoMonitoramento:      obterBool("UPDATE_MONITORING_ENABLED", true),
		AtualizacaoModo:               obter("UPDATE_MODE", "aviso"),
		AtualizacaoJanelaInicio:       obter("UPDATE_WINDOW_START", "01:00"),
		AtualizacaoJanelaFim:          obter("UPDATE_WINDOW_END", "02:00"),
		AtualizacaoIntervaloMinutos:   obterInt("UPDATE_INTERVAL_MINUTES", 30),
		AtualizacaoAplicarHabilitado:  obterBool("UPDATE_APPLY_ENABLED", false),
		AtualizacaoAplicarToken:       os.Getenv("UPDATE_APPLY_TOKEN"),
		AtualizacaoProxyURL:           obter("UPDATE_PROXY_URL", "https://proxy.golang.org"),
		AtualizacaoDiretorioArtefatos: obter("UPDATE_ARTIFACTS_DIR", "./data/updates"),
		DashboardMasterToken:          os.Getenv("DASHBOARD_MASTER_TOKEN"),
		DashboardCookieNome:           obter("DASHBOARD_COOKIE_NAME", "dyalog_dashboard_token"),
		BaseURL:                       strings.TrimRight(obter("API_BASE_URL", ""), "/"),
		HistoricoMaxDias:              obterInt("HISTORY_MAX_DAYS", 90),
		WebhookMaxTentativas:          obterInt("WEBHOOK_MAX_ATTEMPTS", 60),
		WebhookIntervaloBaseSegundos:  obterInt("WEBHOOK_RETRY_BASE_SECONDS", 30),
		WebhookRetryMaxDurationHours:  obterInt("WEBHOOK_RETRY_MAX_DURATION_HOURS", 24),
		WebhookRetryMaxIntervalMin:    obterInt("WEBHOOK_RETRY_MAX_INTERVAL_MINUTES", 30),
		WebhookLoteProcessamento:      obterInt("WEBHOOK_WORKER_BATCH_SIZE", 25),
		WebhookTimeoutSegundos:        obterInt("WEBHOOK_TIMEOUT_SECONDS", 5),
		WebhookConcorrencia:           obterInt("WEBHOOK_WORKER_CONCURRENCY", 5),
		HeartbeatIntervaloSegundos:    obterInt("RUNTIME_HEARTBEAT_INTERVAL_SECONDS", 30),
		RecuperacaoWebhookHabilitada:  obterBool("WEBHOOK_RECOVERY_ENABLED", true),
		RecuperacaoMargemSegundos:     obterInt("WEBHOOK_RECOVERY_MARGIN_SECONDS", 120),
		RecuperacaoHistoricoMensagens: obterInt("WEBHOOK_RECOVERY_HISTORY_COUNT", 50),
		MidiaStorageDriver:            strings.ToLower(obter("MEDIA_STORAGE_DRIVER", "local")),
		MidiaStorageSupabaseURL:       strings.TrimRight(os.Getenv("MEDIA_STORAGE_SUPABASE_URL"), "/"),
		MidiaStorageSupabaseKey:       os.Getenv("MEDIA_STORAGE_SUPABASE_KEY"),
		MidiaStorageSupabaseBucket:    os.Getenv("MEDIA_STORAGE_SUPABASE_BUCKET"),
		MidiaStoragePublicBaseURL:     strings.TrimRight(os.Getenv("MEDIA_STORAGE_PUBLIC_BASE_URL"), "/"),
	}
	if cfg.HistoricoMaxDias < 1 {
		cfg.HistoricoMaxDias = 90
	}
	if cfg.WebhookMaxTentativas < 1 {
		cfg.WebhookMaxTentativas = 5
	}
	if cfg.WebhookIntervaloBaseSegundos < 1 {
		cfg.WebhookIntervaloBaseSegundos = 30
	}
	if cfg.WebhookRetryMaxDurationHours < 1 {
		cfg.WebhookRetryMaxDurationHours = 24
	}
	if cfg.WebhookRetryMaxIntervalMin < 1 {
		cfg.WebhookRetryMaxIntervalMin = 30
	}
	if cfg.WebhookLoteProcessamento < 1 {
		cfg.WebhookLoteProcessamento = 25
	}
	if cfg.WebhookTimeoutSegundos < 1 {
		cfg.WebhookTimeoutSegundos = 5
	}
	if cfg.WebhookConcorrencia < 1 {
		cfg.WebhookConcorrencia = 5
	}
	if cfg.HeartbeatIntervaloSegundos < 5 {
		cfg.HeartbeatIntervaloSegundos = 30
	}
	if cfg.RecuperacaoMargemSegundos < 0 {
		cfg.RecuperacaoMargemSegundos = 120
	}
	if cfg.RecuperacaoHistoricoMensagens < 1 {
		cfg.RecuperacaoHistoricoMensagens = 50
	}
	if cfg.RecuperacaoHistoricoMensagens > 200 {
		cfg.RecuperacaoHistoricoMensagens = 200
	}
	switch cfg.HTTPLogMode {
	case "todos", "erros", "falhas", "desligado":
	default:
		cfg.HTTPLogMode = "falhas"
	}
	switch cfg.WhatsAppLogLevel {
	case "TRACE", "DEBUG", "INFO", "WARN", "ERROR":
	default:
		cfg.WhatsAppLogLevel = "ERROR"
	}

	return cfg, nil
}

func configurarBanco() (string, string) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("DATABASE_DRIVER")))
	dsn := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	if dsn != "" {
		if driver == "" {
			driver = inferirDriverPorDSN(dsn)
		}
		return driver, dsn
	}
	if existeConfigPostgresLegada() || driver == "postgres" || driver == "postgresql" || driver == "pgx" || driver == "supabase" {
		return "postgres", montarDSNPostgres()
	}
	if driver == "" {
		driver = "sqlite"
	}
	return driver, "./data/dyalog.db"
}

func inferirDriverPorDSN(dsn string) string {
	dsn = strings.ToLower(strings.TrimSpace(dsn))
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return "postgres"
	}
	return "sqlite"
}

func existeConfigPostgresLegada() bool {
	for _, chave := range []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT"} {
		if strings.TrimSpace(os.Getenv(chave)) != "" {
			return true
		}
	}
	return false
}

func montarDSNPostgres() string {
	host := obter("DB_HOST", "postgres")
	porta := obter("DB_PORT", "5432")
	usuario := obter("DB_USER", "postgres")
	senha := os.Getenv("DB_PASSWORD")
	banco := obter("DB_NAME", "postgres")
	sslMode := obter("DB_SSLMODE", "disable")

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(usuario, senha),
		Host:   net.JoinHostPort(host, porta),
		Path:   "/" + banco,
	}
	q := u.Query()
	q.Set("sslmode", sslMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func obter(chave, padrao string) string {
	if valor := os.Getenv(chave); valor != "" {
		return valor
	}
	return padrao
}

func obterBool(chave string, padrao bool) bool {
	valor := os.Getenv(chave)
	if valor == "" {
		return padrao
	}
	parsed, err := strconv.ParseBool(valor)
	if err != nil {
		return padrao
	}
	return parsed
}

func obterInt(chave string, padrao int) int {
	valor := os.Getenv(chave)
	if valor == "" {
		return padrao
	}
	parsed, err := strconv.Atoi(valor)
	if err != nil {
		return padrao
	}
	return parsed
}
