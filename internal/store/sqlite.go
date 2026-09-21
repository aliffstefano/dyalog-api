package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dyalog-api-go/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

type InstanciaStore interface {
	Criar(ctx context.Context, instancia models.Instancia) (models.Instancia, error)
	Listar(ctx context.Context) ([]models.Instancia, error)
	BuscarPorID(ctx context.Context, id string) (models.Instancia, error)
	BuscarPorToken(ctx context.Context, token string) (models.Instancia, error)
	AtualizarStatus(ctx context.Context, id, status string) (models.Instancia, error)
	AtualizarToken(ctx context.Context, id, token string) (models.Instancia, error)
	AtualizarHistorico(ctx context.Context, id string, dias int) (models.Instancia, error)
	AtualizarProxy(ctx context.Context, id, modo, proxyURL string) (models.Instancia, error)
	AtualizarPresenca(ctx context.Context, id, presenca string) (models.Instancia, error)
	AtualizarConfiguracaoAvancada(ctx context.Context, id string, cfg models.ConfiguracaoAvancadaInstancia, presenca string) (models.Instancia, error)
	Excluir(ctx context.Context, id string) error
}

type WebhookStore interface {
	CriarWebhook(ctx context.Context, webhook models.WebhookInstancia) (models.WebhookInstancia, error)
	AtualizarWebhook(ctx context.Context, webhook models.WebhookInstancia) (models.WebhookInstancia, error)
	ListarWebhooks(ctx context.Context, instanciaID string) ([]models.WebhookInstancia, error)
	ExcluirWebhook(ctx context.Context, instanciaID, webhookID string) error
	ListarWebhooksAtivosPorEvento(ctx context.Context, instanciaID, evento string) ([]models.WebhookInstancia, error)
}

type WebhookEntregaStore interface {
	EnfileirarWebhookEntrega(ctx context.Context, entrega models.WebhookEntrega) (models.WebhookEntrega, error)
	ListarWebhookEntregas(ctx context.Context, instanciaID string, limite int) ([]models.WebhookEntrega, error)
	BuscarWebhookEntregasPendentes(ctx context.Context, limite int, agora time.Time) ([]models.WebhookEntrega, error)
	MarcarWebhookEntregaEnviando(ctx context.Context, entregaID string, agora time.Time) (bool, error)
	RegistrarResultadoWebhookEntrega(ctx context.Context, entregaID, status string, tentativas int, proximaTentativaEm *time.Time, statusHTTP int, ultimoErro string, agora time.Time) error
	LimparWebhookEntregasAntigas(ctx context.Context, antesDe time.Time) (int64, error)
	CompactarWebhookEntregas(ctx context.Context) error
}

type RuntimeStore interface {
	ObterHeartbeat(ctx context.Context, chave string) (time.Time, bool, error)
	AtualizarHeartbeat(ctx context.Context, chave string, momento time.Time) error
}

type InstanciaRuntimeLockStore interface {
	TentarAssumirInstancia(ctx context.Context, instanciaID, nodeID, endereco string, agora, expiraEm time.Time) (bool, error)
	RenovarInstancia(ctx context.Context, instanciaID, nodeID, endereco string, agora, expiraEm time.Time) (bool, error)
	LiberarInstancia(ctx context.Context, instanciaID, nodeID string) error
	ObterDonoInstancia(ctx context.Context, instanciaID string) (DonoInstancia, bool, error)
}

// DonoInstancia descreve qual container detem o lock de uma instancia.
type DonoInstancia struct {
	NodeID   string
	Endereco string
	ExpiraEm time.Time
}

type MensagemProcessadaStore interface {
	RegistrarMensagemProcessada(ctx context.Context, mensagem models.MensagemProcessada) (bool, error)
}

type SistemaStore interface {
	ObterStatusDependencia(ctx context.Context, dependencia string) (models.StatusDependencia, error)
	SalvarStatusDependencia(ctx context.Context, status models.StatusDependencia) error
}

type MidiaStore interface {
	SalvarMidiaRecebida(ctx context.Context, midia models.MidiaRecebida) (models.MidiaRecebida, error)
	BuscarMidiaRecebida(ctx context.Context, instanciaID, midiaID string) (models.MidiaRecebida, error)
}

// MidiaLimpezaStore cobre a remocao dos arquivos locais de midia que ja tem
// copia em storage externo.
type MidiaLimpezaStore interface {
	BuscarMidiasLocaisComCopiaExterna(ctx context.Context, antesDe time.Time, limite int) ([]models.MidiaArquivoLocal, error)
	EsquecerCaminhoArquivoMidia(ctx context.Context, midiaID string) error
}

type ProxyStore interface {
	ObterProxyGlobal(ctx context.Context) (models.ProxyGlobal, error)
	AtualizarProxyGlobal(ctx context.Context, proxy models.ProxyGlobal) (models.ProxyGlobal, error)
}

type ProxyConfigStore interface {
	BuscarPorID(ctx context.Context, id string) (models.Instancia, error)
	ObterProxyGlobal(ctx context.Context) (models.ProxyGlobal, error)
}

type WhatsAppDeviceStore interface {
	ObterWhatsAppDeviceJID(ctx context.Context, instanciaID string) (string, bool, error)
	SalvarWhatsAppDeviceJID(ctx context.Context, instanciaID, deviceJID string) error
	ExcluirWhatsAppDeviceJID(ctx context.Context, instanciaID string) error
}

type SQLStore struct {
	db      *sql.DB
	dialeto string
}

func (s *SQLStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

const colunasInstancia = `id, nome, token, status, historico_dias, proxy_modo, proxy_url, presenca,
       rejeitar_chamadas, mensagem_rejeitar_chamadas, marcar_lida_automatico, ignorar_grupos, ignorar_status,
       criado_em, atualizado_em`

func NovoSQLStore(driver, dsn string) (*SQLStore, error) {
	dialeto, sqlDriver, dsnNormalizado, err := normalizarBanco(driver, dsn)
	if err != nil {
		return nil, err
	}
	if dialeto == "sqlite" {
		if err := os.MkdirAll(filepath.Dir(dsnNormalizado), 0o755); err != nil {
			return nil, fmt.Errorf("erro ao criar diretorio do banco: %w", err)
		}
	}

	db, err := sql.Open(sqlDriver, dsnNormalizado)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco %s: %w", dialeto, err)
	}
	// sql.Open nao conecta de verdade (e preguicoso); sem esse Ping com timeout
	// explicito, um banco inalcancavel (rede caida, firewall, VPN desconectada)
	// trava o processo em silencio por minutos no primeiro Exec, sem logar nada,
	// ate o orquestrador matar o container por healthcheck. Com o Ping, falhamos
	// rapido e com uma mensagem de erro clara.
	pingCtx, cancelPing := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("erro ao conectar ao banco %s: %w", dialeto, err)
	}
	if dialeto == "sqlite" {
		configurarSQLite(db)
		if err := aplicarPragmasSQLite(db); err != nil {
			_ = db.Close()
			return nil, err
		}
	} else {
		configurarPostgres(db)
	}

	store := &SQLStore{db: db, dialeto: dialeto}
	if err := store.prepararSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.garantirTokensInstancias(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func normalizarBanco(driver, dsn string) (dialeto, sqlDriver, dsnNormalizado string, err error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "" {
		driver = "sqlite"
	}
	switch driver {
	case "sqlite", "sqlite3":
		if strings.TrimSpace(dsn) == "" {
			dsn = "./data/dyalog.db"
		}
		return "sqlite", "sqlite", dsn, nil
	case "postgres", "postgresql", "pgx", "supabase":
		if strings.TrimSpace(dsn) == "" {
			return "", "", "", fmt.Errorf("DATABASE_DSN e obrigatorio quando DATABASE_DRIVER=postgres")
		}
		return "postgres", "pgx", dsn, nil
	default:
		return "", "", "", fmt.Errorf("DATABASE_DRIVER invalido: use sqlite ou postgres")
	}
}

func configurarPostgres(db *sql.DB) {
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
}

func configurarSQLite(db *sql.DB) {
	// SQLite permite apenas uma escrita por vez. Um pool com varias conexoes pode
	// gerar SQLITE_BUSY quando o dashboard, webhooks e monitor gravam em paralelo.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
}

func aplicarPragmasSQLite(db *sql.DB) error {
	const pragmas = `
PRAGMA busy_timeout = 10000;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
`
	if _, err := db.Exec(pragmas); err != nil {
		return fmt.Errorf("erro ao configurar sqlite: %w", err)
	}
	return nil
}

func (s *SQLStore) q(query string) string {
	if s.dialeto != "postgres" {
		return query
	}
	indice := 0
	return placeholderSQLite.ReplaceAllStringFunc(query, func(string) string {
		indice++
		return fmt.Sprintf("$%d", indice)
	})
}

func (s *SQLStore) criarSchemaQuery() string {
	if s.dialeto == "postgres" {
		return schemaPostgres
	}
	return schemaSQLite
}

func (s *SQLStore) boolDB(valor bool) interface{} {
	if s.dialeto == "postgres" {
		return valor
	}
	return boolToInt(valor)
}

func (s *SQLStore) boolFromDB(valor interface{}) bool {
	switch v := valor.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case int64:
		return v == 1
	case []byte:
		return string(v) == "1" || strings.EqualFold(string(v), "true")
	case string:
		return v == "1" || strings.EqualFold(v, "true")
	default:
		return false
	}
}

func (s *SQLStore) agregarEventos(campo string) string {
	if s.dialeto == "postgres" {
		return fmt.Sprintf("STRING_AGG(%s, ',')", campo)
	}
	return fmt.Sprintf("GROUP_CONCAT(%s, ',')", campo)
}

var placeholderSQLite = regexp.MustCompile(`\?`)

func (s *SQLStore) prepararSchema() error {
	_, err := s.db.Exec(s.criarSchemaQuery())
	if err != nil {
		return fmt.Errorf("erro ao preparar schema: %w", err)
	}
	if err := s.garantirColunasInstancias(); err != nil {
		return err
	}
	if err := s.garantirColunasMidiasRecebidas(); err != nil {
		return err
	}
	if err := s.garantirColunasRuntimeLocks(); err != nil {
		return err
	}
	return nil
}

const schemaSQLite = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS instancias (
    id TEXT PRIMARY KEY,
    nome TEXT NOT NULL,
    token TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    historico_dias INTEGER NOT NULL DEFAULT 0,
    proxy_modo TEXT NOT NULL DEFAULT 'herdar',
    proxy_url TEXT NOT NULL DEFAULT '',
    presenca TEXT NOT NULL DEFAULT 'indisponivel',
    rejeitar_chamadas INTEGER NOT NULL DEFAULT 0,
    mensagem_rejeitar_chamadas TEXT NOT NULL DEFAULT '',
    marcar_lida_automatico INTEGER NOT NULL DEFAULT 0,
    ignorar_grupos INTEGER NOT NULL DEFAULT 0,
    ignorar_status INTEGER NOT NULL DEFAULT 0,
    criado_em DATETIME NOT NULL,
    atualizado_em DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_instancias_nome ON instancias(nome);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instancias_token ON instancias(token);

CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    instancia_id TEXT NOT NULL,
    nome TEXT NOT NULL,
    url TEXT NOT NULL,
    ativo INTEGER NOT NULL DEFAULT 1,
    criado_em DATETIME NOT NULL,
    atualizado_em DATETIME NOT NULL,
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_webhooks_instancia ON webhooks(instancia_id);

CREATE TABLE IF NOT EXISTS webhook_eventos (
    webhook_id TEXT NOT NULL,
    evento TEXT NOT NULL,
    PRIMARY KEY (webhook_id, evento),
    FOREIGN KEY(webhook_id) REFERENCES webhooks(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_webhook_eventos_evento ON webhook_eventos(evento);

CREATE TABLE IF NOT EXISTS webhook_entregas (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL,
    instancia_id TEXT NOT NULL,
    webhook_nome TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    evento TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL,
    tentativas INTEGER NOT NULL DEFAULT 0,
    max_tentativas INTEGER NOT NULL DEFAULT 5,
    proxima_tentativa_em DATETIME NOT NULL,
    ultima_tentativa_em DATETIME NULL,
    status_http INTEGER NOT NULL DEFAULT 0,
    ultimo_erro TEXT NOT NULL DEFAULT '',
    criado_em DATETIME NOT NULL,
    atualizado_em DATETIME NOT NULL,
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_webhook_entregas_instancia ON webhook_entregas(instancia_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_entregas_fila ON webhook_entregas(status, proxima_tentativa_em);

CREATE TABLE IF NOT EXISTS sistema_runtime (
    chave TEXT PRIMARY KEY,
    heartbeat_em DATETIME NOT NULL,
    atualizado_em DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS instancia_runtime_locks (
    instancia_id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL,
    endereco TEXT NOT NULL DEFAULT '',
    heartbeat_em DATETIME NOT NULL,
    expira_em DATETIME NOT NULL,
    criado_em DATETIME NOT NULL,
    atualizado_em DATETIME NOT NULL,
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_instancia_runtime_locks_expira ON instancia_runtime_locks(expira_em);

CREATE TABLE IF NOT EXISTS mensagens_processadas (
    instancia_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL,
    mensagem_id TEXT NOT NULL,
    remetente_jid TEXT NOT NULL DEFAULT '',
    enviada_por_mim INTEGER NOT NULL DEFAULT 0,
    grupo INTEGER NOT NULL DEFAULT 0,
    recebida_em DATETIME NOT NULL,
    origem TEXT NOT NULL DEFAULT '',
    processada_em DATETIME NOT NULL,
    PRIMARY KEY (instancia_id, chat_jid, mensagem_id),
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mensagens_processadas_instancia_chat ON mensagens_processadas(instancia_id, chat_jid, recebida_em DESC);

CREATE TABLE IF NOT EXISTS whatsapp_devices (
    instancia_id TEXT PRIMARY KEY,
    device_jid TEXT NOT NULL,
    atualizado_em DATETIME NOT NULL,
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_whatsapp_devices_jid ON whatsapp_devices(device_jid);

CREATE TABLE IF NOT EXISTS sistema_dependencias (
    dependencia TEXT PRIMARY KEY,
    versao_em_uso TEXT NOT NULL,
    ultima_versao_disponivel TEXT NOT NULL DEFAULT '',
    atualizacao_disponivel INTEGER NOT NULL DEFAULT 0,
    status_atualizacao TEXT NOT NULL DEFAULT 'nao_verificado',
    modo_operacao TEXT NOT NULL DEFAULT 'aviso',
    ultima_verificacao_em DATETIME NULL,
    ultima_aplicacao_em DATETIME NULL,
    artefato_preparo_caminho TEXT NOT NULL DEFAULT '',
    ultimo_erro TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS midias_recebidas (
    id TEXT PRIMARY KEY,
    instancia_id TEXT NOT NULL,
    mensagem_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL DEFAULT '',
    remetente_jid TEXT NOT NULL DEFAULT '',
    tipo TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    nome_arquivo TEXT NOT NULL DEFAULT '',
    caminho_arquivo TEXT NOT NULL,
    storage_provider TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    storage_url TEXT NOT NULL DEFAULT '',
    tamanho_bytes INTEGER NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    recebida_em DATETIME NOT NULL,
    criada_em DATETIME NOT NULL,
    UNIQUE(instancia_id, mensagem_id),
    FOREIGN KEY(instancia_id) REFERENCES instancias(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_midias_recebidas_instancia ON midias_recebidas(instancia_id, recebida_em DESC);

CREATE TABLE IF NOT EXISTS sistema_proxy (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL DEFAULT '',
    ativo INTEGER NOT NULL DEFAULT 0,
    atualizado_em DATETIME NOT NULL
);
INSERT OR IGNORE INTO sistema_proxy (id, url, ativo, atualizado_em) VALUES ('global', '', 0, CURRENT_TIMESTAMP);
`

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS instancias (
    id TEXT PRIMARY KEY,
    nome TEXT NOT NULL,
    token TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    historico_dias INTEGER NOT NULL DEFAULT 0,
    proxy_modo TEXT NOT NULL DEFAULT 'herdar',
    proxy_url TEXT NOT NULL DEFAULT '',
    presenca TEXT NOT NULL DEFAULT 'indisponivel',
    rejeitar_chamadas BOOLEAN NOT NULL DEFAULT FALSE,
    mensagem_rejeitar_chamadas TEXT NOT NULL DEFAULT '',
    marcar_lida_automatico BOOLEAN NOT NULL DEFAULT FALSE,
    ignorar_grupos BOOLEAN NOT NULL DEFAULT FALSE,
    ignorar_status BOOLEAN NOT NULL DEFAULT FALSE,
    criado_em TIMESTAMPTZ NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_instancias_nome ON instancias(nome);
CREATE UNIQUE INDEX IF NOT EXISTS idx_instancias_token ON instancias(token);

CREATE TABLE IF NOT EXISTS webhooks (
    id TEXT PRIMARY KEY,
    instancia_id TEXT NOT NULL REFERENCES instancias(id) ON DELETE CASCADE,
    nome TEXT NOT NULL,
    url TEXT NOT NULL,
    ativo BOOLEAN NOT NULL DEFAULT TRUE,
    criado_em TIMESTAMPTZ NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhooks_instancia ON webhooks(instancia_id);

CREATE TABLE IF NOT EXISTS webhook_eventos (
    webhook_id TEXT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
    evento TEXT NOT NULL,
    PRIMARY KEY (webhook_id, evento)
);
CREATE INDEX IF NOT EXISTS idx_webhook_eventos_evento ON webhook_eventos(evento);

CREATE TABLE IF NOT EXISTS webhook_entregas (
    id TEXT PRIMARY KEY,
    webhook_id TEXT NOT NULL,
    instancia_id TEXT NOT NULL REFERENCES instancias(id) ON DELETE CASCADE,
    webhook_nome TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    evento TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL,
    tentativas INTEGER NOT NULL DEFAULT 0,
    max_tentativas INTEGER NOT NULL DEFAULT 5,
    proxima_tentativa_em TIMESTAMPTZ NOT NULL,
    ultima_tentativa_em TIMESTAMPTZ NULL,
    status_http INTEGER NOT NULL DEFAULT 0,
    ultimo_erro TEXT NOT NULL DEFAULT '',
    criado_em TIMESTAMPTZ NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhook_entregas_instancia ON webhook_entregas(instancia_id, criado_em DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_entregas_fila ON webhook_entregas(status, proxima_tentativa_em);

CREATE TABLE IF NOT EXISTS sistema_runtime (
    chave TEXT PRIMARY KEY,
    heartbeat_em TIMESTAMPTZ NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS instancia_runtime_locks (
    instancia_id TEXT PRIMARY KEY REFERENCES instancias(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL,
    endereco TEXT NOT NULL DEFAULT '',
    heartbeat_em TIMESTAMPTZ NOT NULL,
    expira_em TIMESTAMPTZ NOT NULL,
    criado_em TIMESTAMPTZ NOT NULL,
    atualizado_em TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_instancia_runtime_locks_expira ON instancia_runtime_locks(expira_em);

CREATE TABLE IF NOT EXISTS mensagens_processadas (
    instancia_id TEXT NOT NULL REFERENCES instancias(id) ON DELETE CASCADE,
    chat_jid TEXT NOT NULL,
    mensagem_id TEXT NOT NULL,
    remetente_jid TEXT NOT NULL DEFAULT '',
    enviada_por_mim BOOLEAN NOT NULL DEFAULT FALSE,
    grupo BOOLEAN NOT NULL DEFAULT FALSE,
    recebida_em TIMESTAMPTZ NOT NULL,
    origem TEXT NOT NULL DEFAULT '',
    processada_em TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (instancia_id, chat_jid, mensagem_id)
);
CREATE INDEX IF NOT EXISTS idx_mensagens_processadas_instancia_chat ON mensagens_processadas(instancia_id, chat_jid, recebida_em DESC);

CREATE TABLE IF NOT EXISTS whatsapp_devices (
    instancia_id TEXT PRIMARY KEY REFERENCES instancias(id) ON DELETE CASCADE,
    device_jid TEXT NOT NULL UNIQUE,
    atualizado_em TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sistema_dependencias (
    dependencia TEXT PRIMARY KEY,
    versao_em_uso TEXT NOT NULL,
    ultima_versao_disponivel TEXT NOT NULL DEFAULT '',
    atualizacao_disponivel BOOLEAN NOT NULL DEFAULT FALSE,
    status_atualizacao TEXT NOT NULL DEFAULT 'nao_verificado',
    modo_operacao TEXT NOT NULL DEFAULT 'aviso',
    ultima_verificacao_em TIMESTAMPTZ NULL,
    ultima_aplicacao_em TIMESTAMPTZ NULL,
    artefato_preparo_caminho TEXT NOT NULL DEFAULT '',
    ultimo_erro TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS midias_recebidas (
    id TEXT PRIMARY KEY,
    instancia_id TEXT NOT NULL REFERENCES instancias(id) ON DELETE CASCADE,
    mensagem_id TEXT NOT NULL,
    chat_jid TEXT NOT NULL DEFAULT '',
    remetente_jid TEXT NOT NULL DEFAULT '',
    tipo TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    nome_arquivo TEXT NOT NULL DEFAULT '',
    caminho_arquivo TEXT NOT NULL,
    storage_provider TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL DEFAULT '',
    storage_url TEXT NOT NULL DEFAULT '',
    tamanho_bytes BIGINT NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    recebida_em TIMESTAMPTZ NOT NULL,
    criada_em TIMESTAMPTZ NOT NULL,
    UNIQUE(instancia_id, mensagem_id)
);
CREATE INDEX IF NOT EXISTS idx_midias_recebidas_instancia ON midias_recebidas(instancia_id, recebida_em DESC);

CREATE TABLE IF NOT EXISTS sistema_proxy (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL DEFAULT '',
    ativo BOOLEAN NOT NULL DEFAULT FALSE,
    atualizado_em TIMESTAMPTZ NOT NULL
);
INSERT INTO sistema_proxy (id, url, ativo, atualizado_em)
VALUES ('global', '', FALSE, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING;
`

func (s *SQLStore) Criar(ctx context.Context, instancia models.Instancia) (models.Instancia, error) {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO instancias (
    id, nome, token, status, historico_dias, proxy_modo, proxy_url, presenca,
    rejeitar_chamadas, mensagem_rejeitar_chamadas, marcar_lida_automatico, ignorar_grupos, ignorar_status,
    criado_em, atualizado_em
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		instancia.ID,
		instancia.Nome,
		instancia.Token,
		instancia.Status,
		instancia.HistoricoDias,
		instancia.ProxyModo,
		instancia.ProxyURL,
		instancia.Presenca,
		s.boolDB(instancia.RejeitarChamadas),
		instancia.MensagemRejeitarChamadas,
		s.boolDB(instancia.MarcarLidaAutomatico),
		s.boolDB(instancia.IgnorarGrupos),
		s.boolDB(instancia.IgnorarStatus),
		instancia.CriadoEm.UTC(),
		instancia.AtualizadoEm.UTC(),
	)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao criar instancia: %w", err)
	}
	return instancia, nil
}

func (s *SQLStore) Listar(ctx context.Context) ([]models.Instancia, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+colunasInstancia+` FROM instancias ORDER BY criado_em DESC`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar instancias: %w", err)
	}
	defer rows.Close()

	var instancias []models.Instancia
	for rows.Next() {
		instancia, err := s.scanInstancia(rows)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler instancia: %w", err)
		}
		instancias = append(instancias, instancia)
	}

	return instancias, rows.Err()
}

func (s *SQLStore) BuscarPorID(ctx context.Context, id string) (models.Instancia, error) {
	instancia, err := s.scanInstancia(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasInstancia+` FROM instancias WHERE id = ?`), id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao buscar instancia: %w", err)
	}
	return instancia, nil
}

func (s *SQLStore) BuscarPorToken(ctx context.Context, token string) (models.Instancia, error) {
	instancia, err := s.scanInstancia(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasInstancia+` FROM instancias WHERE token = ?`), token))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao buscar instancia por token: %w", err)
	}
	return instancia, nil
}

type scannerInstancia interface {
	Scan(dest ...interface{}) error
}

func (s *SQLStore) scanInstancia(scanner scannerInstancia) (models.Instancia, error) {
	var instancia models.Instancia
	var rejeitarChamadas, marcarLidaAutomatico, ignorarGrupos, ignorarStatus interface{}
	err := scanner.Scan(
		&instancia.ID,
		&instancia.Nome,
		&instancia.Token,
		&instancia.Status,
		&instancia.HistoricoDias,
		&instancia.ProxyModo,
		&instancia.ProxyURL,
		&instancia.Presenca,
		&rejeitarChamadas,
		&instancia.MensagemRejeitarChamadas,
		&marcarLidaAutomatico,
		&ignorarGrupos,
		&ignorarStatus,
		&instancia.CriadoEm,
		&instancia.AtualizadoEm,
	)
	if err != nil {
		return models.Instancia{}, err
	}
	instancia.RejeitarChamadas = s.boolFromDB(rejeitarChamadas)
	instancia.MarcarLidaAutomatico = s.boolFromDB(marcarLidaAutomatico)
	instancia.IgnorarGrupos = s.boolFromDB(ignorarGrupos)
	instancia.IgnorarStatus = s.boolFromDB(ignorarStatus)
	return instancia, nil
}

func (s *SQLStore) AtualizarStatus(ctx context.Context, id, status string) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET status = ?, atualizado_em = ? WHERE id = ?`), status, time.Now().UTC(), id)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao atualizar status da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) AtualizarToken(ctx context.Context, id, token string) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET token = ?, atualizado_em = ? WHERE id = ?`), token, time.Now().UTC(), id)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return models.Instancia{}, fmt.Errorf("token da instancia ja esta em uso")
		}
		return models.Instancia{}, fmt.Errorf("erro ao atualizar token da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao do token: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) AtualizarHistorico(ctx context.Context, id string, dias int) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET historico_dias = ?, atualizado_em = ? WHERE id = ?`), dias, time.Now().UTC(), id)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao atualizar historico da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao do historico: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) AtualizarProxy(ctx context.Context, id, modo, proxyURL string) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET proxy_modo = ?, proxy_url = ?, atualizado_em = ? WHERE id = ?`), modo, proxyURL, time.Now().UTC(), id)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao atualizar proxy da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao do proxy: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) AtualizarPresenca(ctx context.Context, id, presenca string) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET presenca = ?, atualizado_em = ? WHERE id = ?`), presenca, time.Now().UTC(), id)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao atualizar presenca da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao da presenca: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) AtualizarConfiguracaoAvancada(ctx context.Context, id string, cfg models.ConfiguracaoAvancadaInstancia, presenca string) (models.Instancia, error) {
	result, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET
    presenca = ?,
    rejeitar_chamadas = ?,
    mensagem_rejeitar_chamadas = ?,
    marcar_lida_automatico = ?,
    ignorar_grupos = ?,
    ignorar_status = ?,
    atualizado_em = ?
WHERE id = ?`),
		presenca,
		s.boolDB(cfg.RejeitarChamadas),
		cfg.MensagemRejeitarChamadas,
		s.boolDB(cfg.MarcarLidaAutomatico),
		s.boolDB(cfg.IgnorarGrupos),
		s.boolDB(cfg.IgnorarStatus),
		time.Now().UTC(),
		id,
	)
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao atualizar configuracao avancada da instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao validar atualizacao da configuracao avancada: %w", err)
	}
	if afetadas == 0 {
		return models.Instancia{}, ErrInstanciaNaoEncontrada
	}
	return s.BuscarPorID(ctx, id)
}

func (s *SQLStore) Excluir(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, s.q(`DELETE FROM instancias WHERE id = ?`), id)
	if err != nil {
		return fmt.Errorf("erro ao excluir instancia: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao validar exclusao: %w", err)
	}
	if afetadas == 0 {
		return ErrInstanciaNaoEncontrada
	}
	return nil
}

func (s *SQLStore) ObterWhatsAppDeviceJID(ctx context.Context, instanciaID string) (string, bool, error) {
	var deviceJID string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT device_jid FROM whatsapp_devices WHERE instancia_id = ?`), instanciaID).Scan(&deviceJID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("erro ao buscar device WhatsApp da instancia: %w", err)
	}
	return strings.TrimSpace(deviceJID), true, nil
}

func (s *SQLStore) SalvarWhatsAppDeviceJID(ctx context.Context, instanciaID, deviceJID string) error {
	instanciaID = strings.TrimSpace(instanciaID)
	deviceJID = strings.TrimSpace(deviceJID)
	if instanciaID == "" || deviceJID == "" {
		return fmt.Errorf("instancia_id e device_jid sao obrigatorios")
	}
	agora := time.Now().UTC()
	query := `INSERT INTO whatsapp_devices (instancia_id, device_jid, atualizado_em)
VALUES (?, ?, ?)
ON CONFLICT(instancia_id) DO UPDATE SET
    device_jid = excluded.device_jid,
    atualizado_em = excluded.atualizado_em`
	if _, err := s.db.ExecContext(ctx, s.q(query), instanciaID, deviceJID, agora); err != nil {
		return fmt.Errorf("erro ao salvar device WhatsApp da instancia: %w", err)
	}
	return nil
}

func (s *SQLStore) ExcluirWhatsAppDeviceJID(ctx context.Context, instanciaID string) error {
	if _, err := s.db.ExecContext(ctx, s.q(`DELETE FROM whatsapp_devices WHERE instancia_id = ?`), strings.TrimSpace(instanciaID)); err != nil {
		return fmt.Errorf("erro ao excluir device WhatsApp da instancia: %w", err)
	}
	return nil
}

func (s *SQLStore) ObterProxyGlobal(ctx context.Context) (models.ProxyGlobal, error) {
	var proxy models.ProxyGlobal
	var ativo interface{}
	err := s.db.QueryRowContext(ctx, `SELECT url, ativo, atualizado_em FROM sistema_proxy WHERE id = 'global'`).Scan(&proxy.URL, &ativo, &proxy.AtualizadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ProxyGlobal{Ativo: false, AtualizadoEm: time.Now().UTC()}, nil
	}
	if err != nil {
		return models.ProxyGlobal{}, fmt.Errorf("erro ao buscar proxy global: %w", err)
	}
	proxy.Ativo = s.boolFromDB(ativo)
	return proxy, nil
}

func (s *SQLStore) AtualizarProxyGlobal(ctx context.Context, proxy models.ProxyGlobal) (models.ProxyGlobal, error) {
	proxy.AtualizadoEm = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO sistema_proxy (id, url, ativo, atualizado_em)
VALUES ('global', ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    url = excluded.url,
    ativo = excluded.ativo,
    atualizado_em = excluded.atualizado_em`), proxy.URL, s.boolDB(proxy.Ativo), proxy.AtualizadoEm)
	if err != nil {
		return models.ProxyGlobal{}, fmt.Errorf("erro ao atualizar proxy global: %w", err)
	}
	return proxy, nil
}

func (s *SQLStore) CriarWebhook(ctx context.Context, webhook models.WebhookInstancia) (models.WebhookInstancia, error) {
	if err := s.salvarWebhook(ctx, webhook, false); err != nil {
		return models.WebhookInstancia{}, err
	}
	return webhook, nil
}

func (s *SQLStore) AtualizarWebhook(ctx context.Context, webhook models.WebhookInstancia) (models.WebhookInstancia, error) {
	if err := s.salvarWebhook(ctx, webhook, true); err != nil {
		return models.WebhookInstancia{}, err
	}
	return webhook, nil
}

func (s *SQLStore) salvarWebhook(ctx context.Context, webhook models.WebhookInstancia, atualizar bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("erro ao iniciar transacao de webhook: %w", err)
	}
	defer tx.Rollback()

	if atualizar {
		result, err := tx.ExecContext(ctx, s.q(`UPDATE webhooks SET nome = ?, url = ?, ativo = ?, atualizado_em = ? WHERE id = ? AND instancia_id = ?`), webhook.Nome, webhook.URL, s.boolDB(webhook.Ativo), webhook.AtualizadoEm.UTC(), webhook.ID, webhook.InstanciaID)
		if err != nil {
			return fmt.Errorf("erro ao atualizar webhook: %w", err)
		}
		afetadas, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("erro ao validar atualizacao do webhook: %w", err)
		}
		if afetadas == 0 {
			return ErrWebhookNaoEncontrado
		}
		if _, err := tx.ExecContext(ctx, s.q(`DELETE FROM webhook_eventos WHERE webhook_id = ?`), webhook.ID); err != nil {
			return fmt.Errorf("erro ao limpar eventos do webhook: %w", err)
		}
	} else {
		_, err = tx.ExecContext(ctx, s.q(`INSERT INTO webhooks (id, instancia_id, nome, url, ativo, criado_em, atualizado_em) VALUES (?, ?, ?, ?, ?, ?, ?)`), webhook.ID, webhook.InstanciaID, webhook.Nome, webhook.URL, s.boolDB(webhook.Ativo), webhook.CriadoEm.UTC(), webhook.AtualizadoEm.UTC())
		if err != nil {
			return fmt.Errorf("erro ao criar webhook: %w", err)
		}
	}

	for _, evento := range webhook.Eventos {
		if _, err = tx.ExecContext(ctx, s.q(`INSERT INTO webhook_eventos (webhook_id, evento) VALUES (?, ?)`), webhook.ID, evento); err != nil {
			return fmt.Errorf("erro ao salvar evento do webhook: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao concluir webhook: %w", err)
	}
	return nil
}

func (s *SQLStore) ListarWebhooks(ctx context.Context, instanciaID string) ([]models.WebhookInstancia, error) {
	query := strings.ReplaceAll(`
SELECT w.id, w.instancia_id, w.nome, w.url, w.ativo, w.criado_em, w.atualizado_em, COALESCE({EVENTOS_AGG}, '')
FROM webhooks w
LEFT JOIN webhook_eventos we ON we.webhook_id = w.id
WHERE w.instancia_id = ?
GROUP BY w.id, w.instancia_id, w.nome, w.url, w.ativo, w.criado_em, w.atualizado_em
ORDER BY w.criado_em DESC`, "{EVENTOS_AGG}", s.agregarEventos("we.evento"))
	rows, err := s.db.QueryContext(ctx, s.q(query), instanciaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []models.WebhookInstancia
	for rows.Next() {
		var webhook models.WebhookInstancia
		var ativo interface{}
		var eventos string
		if err := rows.Scan(&webhook.ID, &webhook.InstanciaID, &webhook.Nome, &webhook.URL, &ativo, &webhook.CriadoEm, &webhook.AtualizadoEm, &eventos); err != nil {
			return nil, fmt.Errorf("erro ao ler webhook: %w", err)
		}
		webhook.Ativo = s.boolFromDB(ativo)
		webhook.Eventos = splitEventos(eventos)
		webhooks = append(webhooks, webhook)
	}
	return webhooks, rows.Err()
}

func (s *SQLStore) ExcluirWebhook(ctx context.Context, instanciaID, webhookID string) error {
	result, err := s.db.ExecContext(ctx, s.q(`DELETE FROM webhooks WHERE id = ? AND instancia_id = ?`), webhookID, instanciaID)
	if err != nil {
		return fmt.Errorf("erro ao excluir webhook: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("erro ao validar exclusao do webhook: %w", err)
	}
	if afetadas == 0 {
		return ErrWebhookNaoEncontrado
	}
	return nil
}

func (s *SQLStore) ListarWebhooksAtivosPorEvento(ctx context.Context, instanciaID, evento string) ([]models.WebhookInstancia, error) {
	query := strings.ReplaceAll(`
SELECT w.id, w.instancia_id, w.nome, w.url, w.ativo, w.criado_em, w.atualizado_em, COALESCE({EVENTOS_AGG}, '')
FROM webhooks w
INNER JOIN webhook_eventos we ON we.webhook_id = w.id AND we.evento = ?
LEFT JOIN webhook_eventos we2 ON we2.webhook_id = w.id
WHERE w.instancia_id = ? AND w.ativo = TRUE
GROUP BY w.id, w.instancia_id, w.nome, w.url, w.ativo, w.criado_em, w.atualizado_em`, "{EVENTOS_AGG}", s.agregarEventos("we2.evento"))
	rows, err := s.db.QueryContext(ctx, s.q(query), evento, instanciaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar webhooks por evento: %w", err)
	}
	defer rows.Close()

	var webhooks []models.WebhookInstancia
	for rows.Next() {
		var webhook models.WebhookInstancia
		var ativo interface{}
		var eventos string
		if err := rows.Scan(&webhook.ID, &webhook.InstanciaID, &webhook.Nome, &webhook.URL, &ativo, &webhook.CriadoEm, &webhook.AtualizadoEm, &eventos); err != nil {
			return nil, fmt.Errorf("erro ao ler webhook por evento: %w", err)
		}
		webhook.Ativo = s.boolFromDB(ativo)
		webhook.Eventos = splitEventos(eventos)
		webhooks = append(webhooks, webhook)
	}
	return webhooks, rows.Err()
}

func (s *SQLStore) EnfileirarWebhookEntrega(ctx context.Context, entrega models.WebhookEntrega) (models.WebhookEntrega, error) {
	if entrega.CriadoEm.IsZero() {
		entrega.CriadoEm = time.Now().UTC()
	}
	if entrega.AtualizadoEm.IsZero() {
		entrega.AtualizadoEm = entrega.CriadoEm
	}
	if entrega.ProximaTentativaEm.IsZero() {
		entrega.ProximaTentativaEm = entrega.CriadoEm
	}
	if entrega.Status == "" {
		entrega.Status = models.WebhookEntregaPendente
	}
	if entrega.MaxTentativas <= 0 {
		entrega.MaxTentativas = 5
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO webhook_entregas (
    id, webhook_id, instancia_id, webhook_nome, url, evento, payload, status,
    tentativas, max_tentativas, proxima_tentativa_em, ultima_tentativa_em,
    status_http, ultimo_erro, criado_em, atualizado_em
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		entrega.ID,
		entrega.WebhookID,
		entrega.InstanciaID,
		entrega.WebhookNome,
		entrega.URL,
		entrega.Evento,
		string(entrega.Payload),
		entrega.Status,
		entrega.Tentativas,
		entrega.MaxTentativas,
		entrega.ProximaTentativaEm.UTC(),
		timeOrNil(entrega.UltimaTentativaEm),
		entrega.StatusHTTP,
		entrega.UltimoErro,
		entrega.CriadoEm.UTC(),
		entrega.AtualizadoEm.UTC(),
	)
	if err != nil {
		return models.WebhookEntrega{}, fmt.Errorf("erro ao enfileirar entrega de webhook: %w", err)
	}
	return entrega, nil
}

func (s *SQLStore) ListarWebhookEntregas(ctx context.Context, instanciaID string, limite int) ([]models.WebhookEntrega, error) {
	if limite <= 0 || limite > 200 {
		limite = 50
	}
	rows, err := s.db.QueryContext(ctx, s.q(`
SELECT id, webhook_id, instancia_id, webhook_nome, url, evento, payload, status,
       tentativas, max_tentativas, proxima_tentativa_em, ultima_tentativa_em,
       status_http, ultimo_erro, criado_em, atualizado_em
FROM webhook_entregas
WHERE instancia_id = ?
ORDER BY criado_em DESC
LIMIT ?`), instanciaID, limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar entregas de webhook: %w", err)
	}
	defer rows.Close()
	return scanWebhookEntregas(rows)
}

func (s *SQLStore) BuscarWebhookEntregasPendentes(ctx context.Context, limite int, agora time.Time) ([]models.WebhookEntrega, error) {
	if limite <= 0 || limite > 100 {
		limite = 25
	}
	travadasAntesDe := agora.Add(-5 * time.Minute)
	rows, err := s.db.QueryContext(ctx, s.q(`
SELECT id, webhook_id, instancia_id, webhook_nome, url, evento, payload, status,
       tentativas, max_tentativas, proxima_tentativa_em, ultima_tentativa_em,
       status_http, ultimo_erro, criado_em, atualizado_em
FROM webhook_entregas
WHERE (
    status IN (?, ?) AND proxima_tentativa_em <= ?
) OR (
    status = ? AND atualizado_em <= ?
)
ORDER BY proxima_tentativa_em ASC, criado_em ASC
LIMIT ?`),
		models.WebhookEntregaPendente,
		models.WebhookEntregaFalha,
		agora.UTC(),
		models.WebhookEntregaEnviando,
		travadasAntesDe.UTC(),
		limite,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar fila de webhooks: %w", err)
	}
	defer rows.Close()
	return scanWebhookEntregas(rows)
}

func (s *SQLStore) MarcarWebhookEntregaEnviando(ctx context.Context, entregaID string, agora time.Time) (bool, error) {
	travadasAntesDe := agora.Add(-5 * time.Minute)
	result, err := s.db.ExecContext(ctx, s.q(`
UPDATE webhook_entregas
SET status = ?, ultima_tentativa_em = ?, atualizado_em = ?
WHERE id = ? AND (
    status IN (?, ?)
    OR (status = ? AND atualizado_em <= ?)
)`),
		models.WebhookEntregaEnviando,
		agora.UTC(),
		agora.UTC(),
		entregaID,
		models.WebhookEntregaPendente,
		models.WebhookEntregaFalha,
		models.WebhookEntregaEnviando,
		travadasAntesDe.UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("erro ao marcar webhook como enviando: %w", err)
	}
	afetadas, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("erro ao validar entrega de webhook: %w", err)
	}
	if afetadas == 0 {
		return false, nil
	}
	return true, nil
}

func (s *SQLStore) RegistrarResultadoWebhookEntrega(ctx context.Context, entregaID, status string, tentativas int, proximaTentativaEm *time.Time, statusHTTP int, ultimoErro string, agora time.Time) error {
	proxima := agora.UTC()
	if proximaTentativaEm != nil {
		proxima = proximaTentativaEm.UTC()
	}
	_, err := s.db.ExecContext(ctx, s.q(`
UPDATE webhook_entregas
SET status = ?, tentativas = ?, proxima_tentativa_em = ?, status_http = ?, ultimo_erro = ?, atualizado_em = ?
WHERE id = ?`),
		status,
		tentativas,
		proxima,
		statusHTTP,
		ultimoErro,
		agora.UTC(),
		entregaID,
	)
	if err != nil {
		return fmt.Errorf("erro ao registrar resultado do webhook: %w", err)
	}
	return nil
}

// LimparWebhookEntregasAntigas apaga entregas ja finalizadas (entregues ou
// esgotadas) atualizadas antes de antesDe. Entregas pendentes, em retry ("falha"
// com tentativas restantes) ou em andamento ("enviando") nunca sao apagadas aqui,
// mesmo que antigas, para nao perder itens que a fila ainda pode reprocessar.
func (s *SQLStore) LimparWebhookEntregasAntigas(ctx context.Context, antesDe time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, s.q(`
DELETE FROM webhook_entregas
WHERE status IN (?, ?) AND atualizado_em < ?`),
		models.WebhookEntregaEntregue, models.WebhookEntregaEsgotada, antesDe.UTC(),
	)
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar entregas antigas de webhook: %w", err)
	}
	afetadas, _ := result.RowsAffected()
	return afetadas, nil
}

// CompactarWebhookEntregas devolve ao sistema operacional o espaco em disco
// liberado pelos DELETEs de LimparWebhookEntregasAntigas. Um DELETE comum so
// marca as linhas como reutilizaveis internamente pelo Postgres; sem o VACUUM
// FULL o arquivo da tabela no disco nao encolhe. So se aplica ao Postgres (SQLite
// nao tem esse problema de inchaco por MVCC da mesma forma, e o VACUUM do SQLite
// reconstroi o banco inteiro, nao so uma tabela).
//
// VACUUM FULL toma um ACCESS EXCLUSIVE LOCK na tabela (bloqueia leitura e
// escrita) enquanto reescreve o arquivo inteiro. Rodar em horario de baixo
// movimento (ex: madrugada) evita impacto perceptivel.
func (s *SQLStore) CompactarWebhookEntregas(ctx context.Context) error {
	if s.dialeto != "postgres" {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `VACUUM FULL webhook_entregas`); err != nil {
		return fmt.Errorf("erro ao compactar (VACUUM FULL) webhook_entregas: %w", err)
	}
	return nil
}

func (s *SQLStore) ObterHeartbeat(ctx context.Context, chave string) (time.Time, bool, error) {
	var heartbeat time.Time
	err := s.db.QueryRowContext(ctx, s.q(`SELECT heartbeat_em FROM sistema_runtime WHERE chave = ?`), chave).Scan(&heartbeat)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, fmt.Errorf("erro ao consultar heartbeat: %w", err)
	}
	return heartbeat.UTC(), true, nil
}

func (s *SQLStore) AtualizarHeartbeat(ctx context.Context, chave string, momento time.Time) error {
	momento = momento.UTC()
	if s.dialeto == "postgres" {
		_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO sistema_runtime (chave, heartbeat_em, atualizado_em)
VALUES (?, ?, ?)
ON CONFLICT (chave) DO UPDATE SET
    heartbeat_em = excluded.heartbeat_em,
    atualizado_em = excluded.atualizado_em`), chave, momento, momento)
		if err != nil {
			return fmt.Errorf("erro ao atualizar heartbeat: %w", err)
		}
		return nil
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO sistema_runtime (chave, heartbeat_em, atualizado_em)
VALUES (?, ?, ?)
ON CONFLICT(chave) DO UPDATE SET
    heartbeat_em = excluded.heartbeat_em,
    atualizado_em = excluded.atualizado_em`), chave, momento, momento)
	if err != nil {
		return fmt.Errorf("erro ao atualizar heartbeat: %w", err)
	}
	return nil
}

func (s *SQLStore) TentarAssumirInstancia(ctx context.Context, instanciaID, nodeID, endereco string, agora, expiraEm time.Time) (bool, error) {
	instanciaID = strings.TrimSpace(instanciaID)
	nodeID = strings.TrimSpace(nodeID)
	endereco = strings.TrimSpace(endereco)
	if instanciaID == "" || nodeID == "" {
		return false, nil
	}
	agora = agora.UTC()
	expiraEm = expiraEm.UTC()
	if s.dialeto == "postgres" {
		result, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO instancia_runtime_locks (instancia_id, node_id, endereco, heartbeat_em, expira_em, criado_em, atualizado_em)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (instancia_id) DO UPDATE SET
    node_id = excluded.node_id,
    endereco = excluded.endereco,
    heartbeat_em = excluded.heartbeat_em,
    expira_em = excluded.expira_em,
    atualizado_em = excluded.atualizado_em
WHERE instancia_runtime_locks.node_id = excluded.node_id
   OR instancia_runtime_locks.expira_em <= ?`),
			instanciaID, nodeID, endereco, agora, expiraEm, agora, agora, agora)
		if err != nil {
			return false, fmt.Errorf("erro ao assumir lock da instancia: %w", err)
		}
		afetadas, _ := result.RowsAffected()
		return afetadas > 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("erro ao iniciar transacao do lock: %w", err)
	}
	defer tx.Rollback()
	var dono string
	var expiraAtual time.Time
	err = tx.QueryRowContext(ctx, s.q(`SELECT node_id, expira_em FROM instancia_runtime_locks WHERE instancia_id = ?`), instanciaID).Scan(&dono, &expiraAtual)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(ctx, s.q(`
INSERT INTO instancia_runtime_locks (instancia_id, node_id, endereco, heartbeat_em, expira_em, criado_em, atualizado_em)
VALUES (?, ?, ?, ?, ?, ?, ?)`), instanciaID, nodeID, endereco, agora, expiraEm, agora, agora)
		if err != nil {
			return false, fmt.Errorf("erro ao criar lock da instancia: %w", err)
		}
		return true, tx.Commit()
	}
	if err != nil {
		return false, fmt.Errorf("erro ao consultar lock da instancia: %w", err)
	}
	if dono != nodeID && expiraAtual.After(agora) {
		return false, nil
	}
	_, err = tx.ExecContext(ctx, s.q(`
UPDATE instancia_runtime_locks
SET node_id = ?, endereco = ?, heartbeat_em = ?, expira_em = ?, atualizado_em = ?
WHERE instancia_id = ?`), nodeID, endereco, agora, expiraEm, agora, instanciaID)
	if err != nil {
		return false, fmt.Errorf("erro ao atualizar lock da instancia: %w", err)
	}
	return true, tx.Commit()
}

func (s *SQLStore) RenovarInstancia(ctx context.Context, instanciaID, nodeID, endereco string, agora, expiraEm time.Time) (bool, error) {
	result, err := s.db.ExecContext(ctx, s.q(`
UPDATE instancia_runtime_locks
SET endereco = ?, heartbeat_em = ?, expira_em = ?, atualizado_em = ?
WHERE instancia_id = ? AND node_id = ?`),
		strings.TrimSpace(endereco), agora.UTC(), expiraEm.UTC(), agora.UTC(), strings.TrimSpace(instanciaID), strings.TrimSpace(nodeID))
	if err != nil {
		return false, fmt.Errorf("erro ao renovar lock da instancia: %w", err)
	}
	afetadas, _ := result.RowsAffected()
	return afetadas > 0, nil
}

func (s *SQLStore) LiberarInstancia(ctx context.Context, instanciaID, nodeID string) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM instancia_runtime_locks WHERE instancia_id = ? AND node_id = ?`), strings.TrimSpace(instanciaID), strings.TrimSpace(nodeID))
	if err != nil {
		return fmt.Errorf("erro ao liberar lock da instancia: %w", err)
	}
	return nil
}

func (s *SQLStore) ObterDonoInstancia(ctx context.Context, instanciaID string) (DonoInstancia, bool, error) {
	var dono DonoInstancia
	err := s.db.QueryRowContext(ctx, s.q(`SELECT node_id, endereco, expira_em FROM instancia_runtime_locks WHERE instancia_id = ?`), strings.TrimSpace(instanciaID)).Scan(&dono.NodeID, &dono.Endereco, &dono.ExpiraEm)
	if errors.Is(err, sql.ErrNoRows) {
		return DonoInstancia{}, false, nil
	}
	if err != nil {
		return DonoInstancia{}, false, fmt.Errorf("erro ao consultar dono da instancia: %w", err)
	}
	dono.ExpiraEm = dono.ExpiraEm.UTC()
	return dono, true, nil
}

func (s *SQLStore) RegistrarMensagemProcessada(ctx context.Context, mensagem models.MensagemProcessada) (bool, error) {
	if mensagem.ProcessadaEm.IsZero() {
		mensagem.ProcessadaEm = time.Now().UTC()
	}
	if mensagem.RecebidaEm.IsZero() {
		mensagem.RecebidaEm = mensagem.ProcessadaEm
	}
	if s.dialeto == "postgres" {
		result, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO mensagens_processadas (
    instancia_id, chat_jid, mensagem_id, remetente_jid, enviada_por_mim, grupo,
    recebida_em, origem, processada_em
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (instancia_id, chat_jid, mensagem_id) DO NOTHING`),
			mensagem.InstanciaID,
			mensagem.ChatJID,
			mensagem.MensagemID,
			mensagem.RemetenteJID,
			mensagem.EnviadaPorMim,
			mensagem.Grupo,
			mensagem.RecebidaEm.UTC(),
			mensagem.Origem,
			mensagem.ProcessadaEm.UTC(),
		)
		if err != nil {
			return false, fmt.Errorf("erro ao registrar mensagem processada: %w", err)
		}
		afetadas, _ := result.RowsAffected()
		return afetadas > 0, nil
	}
	result, err := s.db.ExecContext(ctx, s.q(`
INSERT OR IGNORE INTO mensagens_processadas (
    instancia_id, chat_jid, mensagem_id, remetente_jid, enviada_por_mim, grupo,
    recebida_em, origem, processada_em
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		mensagem.InstanciaID,
		mensagem.ChatJID,
		mensagem.MensagemID,
		mensagem.RemetenteJID,
		s.boolDB(mensagem.EnviadaPorMim),
		s.boolDB(mensagem.Grupo),
		mensagem.RecebidaEm.UTC(),
		mensagem.Origem,
		mensagem.ProcessadaEm.UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("erro ao registrar mensagem processada: %w", err)
	}
	afetadas, _ := result.RowsAffected()
	return afetadas > 0, nil
}

type scannerWebhookEntrega interface {
	Scan(dest ...interface{}) error
}

func scanWebhookEntregas(rows *sql.Rows) ([]models.WebhookEntrega, error) {
	var entregas []models.WebhookEntrega
	for rows.Next() {
		entrega, err := scanWebhookEntrega(rows)
		if err != nil {
			return nil, err
		}
		entregas = append(entregas, entrega)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entregas, nil
}

func scanWebhookEntrega(scanner scannerWebhookEntrega) (models.WebhookEntrega, error) {
	var entrega models.WebhookEntrega
	var payload string
	var ultimaTentativa sql.NullTime
	if err := scanner.Scan(
		&entrega.ID,
		&entrega.WebhookID,
		&entrega.InstanciaID,
		&entrega.WebhookNome,
		&entrega.URL,
		&entrega.Evento,
		&payload,
		&entrega.Status,
		&entrega.Tentativas,
		&entrega.MaxTentativas,
		&entrega.ProximaTentativaEm,
		&ultimaTentativa,
		&entrega.StatusHTTP,
		&entrega.UltimoErro,
		&entrega.CriadoEm,
		&entrega.AtualizadoEm,
	); err != nil {
		return models.WebhookEntrega{}, fmt.Errorf("erro ao ler entrega de webhook: %w", err)
	}
	entrega.Payload = []byte(payload)
	if ultimaTentativa.Valid {
		t := ultimaTentativa.Time
		entrega.UltimaTentativaEm = &t
	}
	return entrega, nil
}

func (s *SQLStore) ObterStatusDependencia(ctx context.Context, dependencia string) (models.StatusDependencia, error) {
	var status models.StatusDependencia
	var ultimaVerificacao sql.NullTime
	var ultimaAplicacao sql.NullTime
	var atualizacaoDisponivel interface{}

	err := s.db.QueryRowContext(ctx, s.q(`
SELECT dependencia, versao_em_uso, ultima_versao_disponivel, atualizacao_disponivel, status_atualizacao, modo_operacao,
       ultima_verificacao_em, ultima_aplicacao_em, artefato_preparo_caminho, ultimo_erro
FROM sistema_dependencias
WHERE dependencia = ?`), dependencia).Scan(
		&status.Dependencia,
		&status.VersaoEmUso,
		&status.UltimaVersaoDisponivel,
		&atualizacaoDisponivel,
		&status.StatusAtualizacao,
		&status.ModoOperacao,
		&ultimaVerificacao,
		&ultimaAplicacao,
		&status.ArtefatoPreparoCaminho,
		&status.UltimoErro,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.StatusDependencia{}, ErrDependenciaNaoEncontrada
	}
	if err != nil {
		return models.StatusDependencia{}, fmt.Errorf("erro ao buscar status da dependencia: %w", err)
	}

	status.AtualizacaoDisponivel = s.boolFromDB(atualizacaoDisponivel)
	if ultimaVerificacao.Valid {
		status.UltimaVerificacaoEm = &ultimaVerificacao.Time
	}
	if ultimaAplicacao.Valid {
		status.UltimaAplicacaoEm = &ultimaAplicacao.Time
	}
	return status, nil
}

func (s *SQLStore) SalvarStatusDependencia(ctx context.Context, status models.StatusDependencia) error {
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO sistema_dependencias (
    dependencia, versao_em_uso, ultima_versao_disponivel, atualizacao_disponivel, status_atualizacao, modo_operacao,
    ultima_verificacao_em, ultima_aplicacao_em, artefato_preparo_caminho, ultimo_erro
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(dependencia) DO UPDATE SET
    versao_em_uso = excluded.versao_em_uso,
    ultima_versao_disponivel = excluded.ultima_versao_disponivel,
    atualizacao_disponivel = excluded.atualizacao_disponivel,
    status_atualizacao = excluded.status_atualizacao,
    modo_operacao = excluded.modo_operacao,
    ultima_verificacao_em = excluded.ultima_verificacao_em,
    ultima_aplicacao_em = excluded.ultima_aplicacao_em,
    artefato_preparo_caminho = excluded.artefato_preparo_caminho,
    ultimo_erro = excluded.ultimo_erro`),
		status.Dependencia,
		status.VersaoEmUso,
		status.UltimaVersaoDisponivel,
		s.boolDB(status.AtualizacaoDisponivel),
		status.StatusAtualizacao,
		status.ModoOperacao,
		timeOrNil(status.UltimaVerificacaoEm),
		timeOrNil(status.UltimaAplicacaoEm),
		status.ArtefatoPreparoCaminho,
		status.UltimoErro,
	)
	if err != nil {
		return fmt.Errorf("erro ao salvar status da dependencia: %w", err)
	}
	return nil
}

func (s *SQLStore) SalvarMidiaRecebida(ctx context.Context, midia models.MidiaRecebida) (models.MidiaRecebida, error) {
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO midias_recebidas (
    id, instancia_id, mensagem_id, chat_jid, remetente_jid, tipo, mime_type, nome_arquivo,
    caminho_arquivo, storage_provider, storage_path, storage_url, tamanho_bytes, sha256, recebida_em, criada_em
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    chat_jid = excluded.chat_jid,
    remetente_jid = excluded.remetente_jid,
    tipo = excluded.tipo,
    mime_type = excluded.mime_type,
    nome_arquivo = excluded.nome_arquivo,
    caminho_arquivo = excluded.caminho_arquivo,
    storage_provider = excluded.storage_provider,
    storage_path = excluded.storage_path,
    storage_url = excluded.storage_url,
    tamanho_bytes = excluded.tamanho_bytes,
    sha256 = excluded.sha256,
    recebida_em = excluded.recebida_em`),
		midia.ID,
		midia.InstanciaID,
		midia.MensagemID,
		midia.ChatJID,
		midia.RemetenteJID,
		midia.Tipo,
		midia.MimeType,
		midia.NomeArquivo,
		midia.CaminhoArquivo,
		midia.StorageProvider,
		midia.StoragePath,
		midia.StorageURL,
		midia.TamanhoBytes,
		midia.SHA256,
		midia.RecebidaEm.UTC(),
		midia.CriadaEm.UTC(),
	)
	if err != nil {
		return models.MidiaRecebida{}, fmt.Errorf("erro ao salvar midia recebida: %w", err)
	}
	return midia, nil
}

// BuscarMidiasLocaisComCopiaExterna lista arquivos de midia que ja subiram para o
// storage externo e podem ser apagados do disco.
//
// As duas condicoes de storage sao o ponto central: so entra aqui midia que tem
// copia fora. Midia sem copia externa fica no disco para sempre, porque apagar
// seria perder o arquivo, nao liberar espaco.
func (s *SQLStore) BuscarMidiasLocaisComCopiaExterna(ctx context.Context, antesDe time.Time, limite int) ([]models.MidiaArquivoLocal, error) {
	if limite <= 0 {
		limite = 500
	}
	linhas, err := s.db.QueryContext(ctx, s.q(`
SELECT id, instancia_id, caminho_arquivo
FROM midias_recebidas
WHERE recebida_em < ? AND caminho_arquivo <> '' AND storage_provider <> '' AND storage_url <> ''
ORDER BY recebida_em
LIMIT ?`), antesDe.UTC(), limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar midias locais para limpeza: %w", err)
	}
	defer linhas.Close()
	var arquivos []models.MidiaArquivoLocal
	for linhas.Next() {
		var arquivo models.MidiaArquivoLocal
		if err := linhas.Scan(&arquivo.ID, &arquivo.InstanciaID, &arquivo.CaminhoArquivo); err != nil {
			return nil, fmt.Errorf("erro ao ler midia local para limpeza: %w", err)
		}
		arquivos = append(arquivos, arquivo)
	}
	return arquivos, linhas.Err()
}

// EsquecerCaminhoArquivoMidia zera o caminho local depois que o arquivo foi
// apagado. O registro continua: e ele que guarda a URL no storage externo.
func (s *SQLStore) EsquecerCaminhoArquivoMidia(ctx context.Context, midiaID string) error {
	_, err := s.db.ExecContext(ctx, s.q(`
UPDATE midias_recebidas SET caminho_arquivo = '' WHERE id = ?`), midiaID)
	if err != nil {
		return fmt.Errorf("erro ao limpar caminho local da midia: %w", err)
	}
	return nil
}

func (s *SQLStore) BuscarMidiaRecebida(ctx context.Context, instanciaID, midiaID string) (models.MidiaRecebida, error) {
	var midia models.MidiaRecebida
	err := s.db.QueryRowContext(ctx, s.q(`
SELECT id, instancia_id, mensagem_id, chat_jid, remetente_jid, tipo, mime_type, nome_arquivo,
       caminho_arquivo, storage_provider, storage_path, storage_url, tamanho_bytes, sha256, recebida_em, criada_em
FROM midias_recebidas
WHERE instancia_id = ? AND id = ?`), instanciaID, midiaID).Scan(
		&midia.ID,
		&midia.InstanciaID,
		&midia.MensagemID,
		&midia.ChatJID,
		&midia.RemetenteJID,
		&midia.Tipo,
		&midia.MimeType,
		&midia.NomeArquivo,
		&midia.CaminhoArquivo,
		&midia.StorageProvider,
		&midia.StoragePath,
		&midia.StorageURL,
		&midia.TamanhoBytes,
		&midia.SHA256,
		&midia.RecebidaEm,
		&midia.CriadaEm,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.MidiaRecebida{}, ErrMidiaNaoEncontrada
	}
	if err != nil {
		return models.MidiaRecebida{}, fmt.Errorf("erro ao buscar midia recebida: %w", err)
	}
	return midia, nil
}

func (s *SQLStore) garantirColunasInstancias() error {
	colunas := []struct {
		nome  string
		query string
	}{
		{nome: "token", query: `ALTER TABLE instancias ADD COLUMN token TEXT NOT NULL DEFAULT ''`},
		{nome: "historico_dias", query: `ALTER TABLE instancias ADD COLUMN historico_dias INTEGER NOT NULL DEFAULT 0`},
		{nome: "proxy_modo", query: `ALTER TABLE instancias ADD COLUMN proxy_modo TEXT NOT NULL DEFAULT 'herdar'`},
		{nome: "proxy_url", query: `ALTER TABLE instancias ADD COLUMN proxy_url TEXT NOT NULL DEFAULT ''`},
		{nome: "presenca", query: `ALTER TABLE instancias ADD COLUMN presenca TEXT NOT NULL DEFAULT 'indisponivel'`},
		{nome: "rejeitar_chamadas", query: `ALTER TABLE instancias ADD COLUMN rejeitar_chamadas BOOLEAN NOT NULL DEFAULT FALSE`},
		{nome: "mensagem_rejeitar_chamadas", query: `ALTER TABLE instancias ADD COLUMN mensagem_rejeitar_chamadas TEXT NOT NULL DEFAULT ''`},
		{nome: "marcar_lida_automatico", query: `ALTER TABLE instancias ADD COLUMN marcar_lida_automatico BOOLEAN NOT NULL DEFAULT FALSE`},
		{nome: "ignorar_grupos", query: `ALTER TABLE instancias ADD COLUMN ignorar_grupos BOOLEAN NOT NULL DEFAULT FALSE`},
		{nome: "ignorar_status", query: `ALTER TABLE instancias ADD COLUMN ignorar_status BOOLEAN NOT NULL DEFAULT FALSE`},
	}
	for _, coluna := range colunas {
		if _, err := s.db.Exec(coluna.query); err != nil && !erroColunaDuplicada(err) {
			return fmt.Errorf("erro ao garantir coluna %s da instancia: %w", coluna.nome, err)
		}
	}
	return nil
}

func (s *SQLStore) garantirColunasMidiasRecebidas() error {
	colunas := []struct {
		nome  string
		query string
	}{
		{nome: "storage_provider", query: `ALTER TABLE midias_recebidas ADD COLUMN storage_provider TEXT NOT NULL DEFAULT ''`},
		{nome: "storage_path", query: `ALTER TABLE midias_recebidas ADD COLUMN storage_path TEXT NOT NULL DEFAULT ''`},
		{nome: "storage_url", query: `ALTER TABLE midias_recebidas ADD COLUMN storage_url TEXT NOT NULL DEFAULT ''`},
	}
	for _, coluna := range colunas {
		if _, err := s.db.Exec(coluna.query); err != nil && !erroColunaDuplicada(err) {
			return fmt.Errorf("erro ao garantir coluna %s da midia recebida: %w", coluna.nome, err)
		}
	}
	return nil
}

func (s *SQLStore) garantirColunasRuntimeLocks() error {
	colunas := []struct {
		nome  string
		query string
	}{
		{nome: "endereco", query: `ALTER TABLE instancia_runtime_locks ADD COLUMN endereco TEXT NOT NULL DEFAULT ''`},
	}
	for _, coluna := range colunas {
		if _, err := s.db.Exec(coluna.query); err != nil && !erroColunaDuplicada(err) {
			return fmt.Errorf("erro ao garantir coluna %s do lock de runtime: %w", coluna.nome, err)
		}
	}
	return nil
}

func erroColunaDuplicada(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") || strings.Contains(msg, "already exists")
}

func (s *SQLStore) garantirTokensInstancias() error {
	rows, err := s.db.Query(s.q(`SELECT id FROM instancias WHERE token = '' OR token IS NULL`))
	if err != nil {
		return fmt.Errorf("erro ao consultar tokens de instancias: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("erro ao ler instancia sem token: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("erro ao listar instancias sem token: %w", err)
	}

	for _, id := range ids {
		token, err := gerarTokenInstancia()
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(s.q(`UPDATE instancias SET token = ? WHERE id = ?`), token, id); err != nil {
			return fmt.Errorf("erro ao atualizar token da instancia: %w", err)
		}
	}
	return nil
}

func gerarTokenInstancia() (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("erro ao gerar token da instancia: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func splitEventos(valor string) []string {
	if valor == "" {
		return []string{}
	}
	partes := strings.Split(valor, ",")
	eventos := make([]string, 0, len(partes))
	for _, parte := range partes {
		parte = strings.TrimSpace(parte)
		if parte != "" {
			eventos = append(eventos, parte)
		}
	}
	return eventos
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func timeOrNil(v *time.Time) interface{} {
	if v == nil {
		return nil
	}
	return v.UTC()
}
