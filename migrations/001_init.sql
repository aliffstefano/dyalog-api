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

CREATE TABLE IF NOT EXISTS sistema_proxy (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL DEFAULT '',
    ativo INTEGER NOT NULL DEFAULT 0,
    atualizado_em DATETIME NOT NULL
);

INSERT OR IGNORE INTO sistema_proxy (id, url, ativo, atualizado_em) VALUES ('global', '', 0, CURRENT_TIMESTAMP);

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
