package config

import (
	"fmt"
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
	WhatsAppStoreDriver           string
	WhatsAppStoreDSN              string
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
	WebhookEntregaRetencaoDias    int
	RuntimeNodeID                 string
	RuntimeNodeEndereco           string
	RuntimeLockTTLSeconds         int
	HeartbeatIntervaloSegundos    int
	ReconexaoIntervaloSegundos    int
	RecuperacaoWebhookHabilitada  bool
	RecuperacaoMargemSegundos     int
	RecuperacaoHistoricoMensagens int
	MidiaStorageDriver            string
	MidiaStorageSupabaseURL       string
	MidiaStorageSupabaseKey       string
	MidiaStorageSupabaseBucket    string
	MidiaStoragePublicBaseURL     string
	MidiaStorageS3Endpoint        string
	MidiaStorageS3AccessKey       string
	MidiaStorageS3SecretKey       string
	MidiaStorageS3Bucket          string
	MidiaStorageS3Region          string
}

func Carregar() (*Config, error) {
	_ = godotenv.Load()

	bancoDriver, bancoDSN := configurarBanco()
	whatsAppStoreDriver, whatsAppStoreDSN, err := configurarStoreWhatsApp(bancoDriver, bancoDSN)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		Porta:                         obter("APP_PORT", "8080"),
		BancoDriver:                   bancoDriver,
		BancoDSN:                      bancoDSN,
		WhatsAppStoreDriver:           whatsAppStoreDriver,
		WhatsAppStoreDSN:              whatsAppStoreDSN,
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
		WebhookEntregaRetencaoDias:    obterInt("WEBHOOK_ENTREGA_RETENTION_DAYS", 30),
		RuntimeNodeID:                 obterRuntimeNodeID(),
		RuntimeNodeEndereco:           obterRuntimeNodeEndereco(obterInt("APP_PORT", 8080)),
		RuntimeLockTTLSeconds:         obterInt("RUNTIME_LOCK_TTL_SECONDS", 90),
		HeartbeatIntervaloSegundos:    obterInt("RUNTIME_HEARTBEAT_INTERVAL_SECONDS", 30),
		ReconexaoIntervaloSegundos:    obterInt("INSTANCE_RECONNECT_INTERVAL_SECONDS", 30),
		RecuperacaoWebhookHabilitada:  obterBool("WEBHOOK_RECOVERY_ENABLED", true),
		RecuperacaoMargemSegundos:     obterInt("WEBHOOK_RECOVERY_MARGIN_SECONDS", 120),
		RecuperacaoHistoricoMensagens: obterInt("WEBHOOK_RECOVERY_HISTORY_COUNT", 50),
		MidiaStorageDriver:            strings.ToLower(obter("MEDIA_STORAGE_DRIVER", "local")),
		MidiaStorageSupabaseURL:       strings.TrimRight(os.Getenv("MEDIA_STORAGE_SUPABASE_URL"), "/"),
		MidiaStorageSupabaseKey:       os.Getenv("MEDIA_STORAGE_SUPABASE_KEY"),
		MidiaStorageSupabaseBucket:    os.Getenv("MEDIA_STORAGE_SUPABASE_BUCKET"),
		MidiaStoragePublicBaseURL:     strings.TrimRight(os.Getenv("MEDIA_STORAGE_PUBLIC_BASE_URL"), "/"),
		MidiaStorageS3Endpoint:        strings.TrimSpace(os.Getenv("MEDIA_STORAGE_S3_ENDPOINT")),
		MidiaStorageS3AccessKey:       strings.TrimSpace(os.Getenv("MEDIA_STORAGE_S3_ACCESS_KEY")),
		MidiaStorageS3SecretKey:       strings.TrimSpace(os.Getenv("MEDIA_STORAGE_S3_SECRET_KEY")),
		MidiaStorageS3Bucket:          strings.TrimSpace(os.Getenv("MEDIA_STORAGE_S3_BUCKET")),
		MidiaStorageS3Region:          strings.TrimSpace(os.Getenv("MEDIA_STORAGE_S3_REGION")),
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
	if cfg.WebhookEntregaRetencaoDias < 1 {
		cfg.WebhookEntregaRetencaoDias = 30
	}
	if cfg.RuntimeLockTTLSeconds < 15 {
		cfg.RuntimeLockTTLSeconds = 90
	}
	if cfg.HeartbeatIntervaloSegundos < 5 {
		cfg.HeartbeatIntervaloSegundos = 30
	}
	if cfg.ReconexaoIntervaloSegundos < 10 {
		cfg.ReconexaoIntervaloSegundos = 30
	}
	if cfg.RuntimeLockTTLSeconds <= cfg.HeartbeatIntervaloSegundos {
		cfg.RuntimeLockTTLSeconds = cfg.HeartbeatIntervaloSegundos * 3
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

func obterRuntimeNodeID() string {
	if valor := strings.TrimSpace(os.Getenv("RUNTIME_NODE_ID")); valor != "" {
		return valor
	}
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		return strings.TrimSpace(hostname)
	}
	return "node-" + buildinfo.Commit
}

// obterRuntimeNodeEndereco descobre a URL pela qual as outras replicas conseguem
// falar com este container. Em Docker Swarm cada replica tem um IP proprio na rede
// overlay; e esse IP (e nao o VIP do servico, que balancearia de volta) que precisa
// ser registrado no lock para o encaminhamento entre replicas funcionar.
func obterRuntimeNodeEndereco(porta int) string {
	if valor := strings.TrimSpace(os.Getenv("RUNTIME_NODE_ADDRESS")); valor != "" {
		return strings.TrimRight(valor, "/")
	}
	if porta <= 0 {
		porta = 8080
	}
	if ip := detectarIPLocal(); ip != "" {
		return fmt.Sprintf("http://%s:%d", ip, porta)
	}
	return ""
}

func detectarIPLocal() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	var fallback string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		enderecos, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, endereco := range enderecos {
			rede, ok := endereco.(*net.IPNet)
			if !ok {
				continue
			}
			ip := rede.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			// Redes overlay do Swarm usam 10.0.0.0/8; a docker_gwbridge usa 172.x.
			// Preferimos a overlay porque e a que conecta as replicas entre si.
			if ip[0] == 10 {
				return ip.String()
			}
			if fallback == "" {
				fallback = ip.String()
			}
		}
	}
	return fallback
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

func configurarStoreWhatsApp(bancoDriver, bancoDSN string) (string, string, error) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("WHATSAPP_STORE_DRIVER")))
	dsn := strings.TrimSpace(os.Getenv("WHATSAPP_STORE_DSN"))
	if driver == "" && dsn != "" {
		driver = inferirDriverPorDSN(dsn)
	}
	if driver == "" {
		driver = "sqlite"
	}
	switch driver {
	case "sqlite", "sqlite3":
		return "sqlite", "", nil
	case "postgres", "postgresql", "pgx", "supabase":
		if dsn == "" && bancoDriver == "postgres" {
			dsn = bancoDSN
		}
		if dsn == "" {
			return "", "", fmt.Errorf("WHATSAPP_STORE_DSN e obrigatorio quando WHATSAPP_STORE_DRIVER=postgres sem DATABASE_DRIVER=postgres")
		}
		return "postgres", dsn, nil
	default:
		return "", "", fmt.Errorf("WHATSAPP_STORE_DRIVER invalido: use sqlite ou postgres")
	}
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
