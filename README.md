# Dyalog API GO

<p align="center">
  <img src="static/img/dyalog.png" alt="Dyalog Internet Solutions" width="260">
</p>

<p align="center">
  <strong>API propria para WhatsApp em Go, com whatsmeow como nucleo, multi-instancia, webhooks, dashboard operacional e suporte a Docker.</strong>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go&logoColor=white">
  <img alt="Gin" src="https://img.shields.io/badge/Gin-HTTP-1995BD?style=for-the-badge">
  <img alt="Whatsmeow" src="https://img.shields.io/badge/Whatsmeow-Core-25D366?style=for-the-badge">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-ready-2496ED?style=for-the-badge&logo=docker&logoColor=white">
</p>

## Visao geral

Dyalog API GO e uma camada propria para operar instancias WhatsApp usando `go.mau.fi/whatsmeow`, sem depender da interface do WuzAPI. O projeto foi desenhado para uso em automacoes, CRMs e integracoes internas, com endpoints versionados em portugues e painel web embutido.

Principais objetivos:

- Operar multiplas instancias WhatsApp no mesmo backend.
- Conectar por QR code ou Pairing Code.
- Enviar e receber mensagens por API e webhooks.
- Persistir sessoes, instancias, tokens, webhooks e auditoria.
- Servir uma dashboard limpa para operacao e suporte.
- Rodar local, Docker, SQLite, Postgres ou Supabase.

## Recursos principais

- Multi-instancia com token individual por instancia.
- Dashboard protegida por token master ou token da instancia.
- Envio de texto, imagem, audio, documento, botoes/listas com fallback e presenca.
- Recebimento de mensagens, midias, digitando, gravando audio, recibos e chamadas via webhook.
- Download de midias recebidas por arquivo ou base64.
- Auditoria de webhooks com status, tentativas, HTTP, erro e retry.
- Retry de webhook configuravel por ate 24h.
- Recuperacao de janela offline por historico on-demand.
- Proxy global e proxy individual por instancia.
- Configuracoes avancadas por instancia.
- Monitoramento de versao do `whatsmeow`.
- Docker Hub: `aliffstefano/dyalog-api`.

## Stack

| Camada | Tecnologia |
| --- | --- |
| Linguagem | Go |
| HTTP | Gin |
| WhatsApp | `go.mau.fi/whatsmeow` |
| Banco local | SQLite |
| Banco externo | Postgres / Supabase |
| Dashboard | HTML, CSS e JS servidos pelo backend |
| Container | Docker |

## Links rapidos

- API local: `http://localhost:8080/api/v1`
- Dashboard local: `http://localhost:8080/`
- Docs da API: `http://localhost:8080/api/v1/docs`
- Documentacao detalhada: [`docs/endpoints.md`](docs/endpoints.md)
- Guia tecnico: [`docs/dyalog-api.md`](docs/dyalog-api.md)
- Exemplo de ambiente: [`.env.example`](.env.example)

## Quick start local

```bash
cd dyalog-api-go
cp .env.example .env
go mod tidy
go run ./cmd/api
```

No Windows PowerShell:

```powershell
Copy-Item .env.example .env
go mod tidy
go run ./cmd/api
```

Depois acesse:

- Dashboard: `http://localhost:8080/`
- API: `http://localhost:8080/api/v1`
- Healthcheck: `http://localhost:8080/api/v1/saude`

## Quick start Docker

```bash
docker run -d \
  --name dyalog-api \
  -p 8080:8080 \
  -e DASHBOARD_MASTER_TOKEN=troque-este-token \
  -e API_BASE_URL=http://localhost:8080 \
  -v dyalog-api-data:/app/data \
  aliffstefano/dyalog-api:latest
```

Docker Compose minimo:

```yaml
services:
  dyalog-api:
    image: aliffstefano/dyalog-api:latest
    ports:
      - "8080:8080"
    environment:
      DASHBOARD_MASTER_TOKEN: "troque-este-token"
      API_BASE_URL: "http://localhost:8080"
      DATABASE_DRIVER: "sqlite"
      DATABASE_DSN: "./data/dyalog.db"
      TZ: "America/Cuiaba"
    volumes:
      - dyalog-api-data:/app/data
    restart: unless-stopped

volumes:
  dyalog-api-data:
```

## Autenticacao

A API aceita o token no header:

```http
X-Access-Token: SEU_TOKEN
```

Tambem aceita:

```http
Authorization: Bearer SEU_TOKEN
```

Regras:

- Token master administra todas as instancias.
- Token de instancia acessa apenas a propria instancia.
- Ao usar token de instancia, o campo `instancia` pode ser omitido nos envios.
- Nao envie token no body JSON.

## Exemplo de envio de texto

```bash
curl -X POST "http://localhost:8080/api/v1/batepapo/enviar/texto" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: SEU_TOKEN_DA_INSTANCIA" \
  -d '{
    "numero": "5511999999999",
    "mensagem": "Mensagem enviada pela Dyalog API"
  }'
```

## Exemplo de webhook recebido

```json
{
  "evento": "mensagens",
  "instancia_id": "eeee80e7-9d9d-487e-bffe-509eeec7729e",
  "ocorrido_em": "2026-07-01T15:00:00Z",
  "dados": {
    "tipo": "texto",
    "conteudo": "Oi",
    "chat_numero": "5511999999999",
    "remetente_numero": "5511999999999",
    "grupo": false,
    "direcao": "entrada",
    "enviado_por_mim": false,
    "mensagem_id": "3EB0..."
  }
}
```

## Banco de dados

SQLite e o padrao para desenvolvimento:

```env
DATABASE_DRIVER=sqlite
DATABASE_DSN=./data/dyalog.db
```

Postgres/Supabase:

```env
DATABASE_DRIVER=postgres
DATABASE_DSN=postgres://usuario:senha@host:5432/postgres?sslmode=require
```

Tambem e aceito o padrao separado:

```env
DB_USER=postgres
DB_PASSWORD=senha
DB_NAME=dyalog-api
DB_HOST=postgres
DB_PORT=5432
DB_SSLMODE=disable
```

Observacoes:

- `DATABASE_DSN` tem prioridade sobre `DB_*`.
- Supabase usa `DATABASE_DRIVER=postgres`.
- Por padrao, sessoes do WhatsApp ficam em `SESSION_STORAGE_DIR`.
- Para salvar o store interno do whatsmeow no Postgres, use `WHATSAPP_STORE_DRIVER=postgres`. Se `WHATSAPP_STORE_DSN` estiver vazio, a API reutiliza o Postgres da aplicacao.
- Em Docker, mantenha volume persistente em `/app/data`.

## Atualizacao do whatsmeow

Atualizacao de dependencia Go exige rebuild e novo processo. Nao existe hot reload seguro do `whatsmeow` dentro do binario em execucao.

Fluxo local recomendado:

```bash
go get go.mau.fi/whatsmeow@latest
go mod tidy
go test ./...
go build ./cmd/api
go run ./cmd/api
```

Em producao com Docker, gere nova imagem e faca rolling update com rollback preparado.

### Tags da imagem

Cada publicacao gera duas tags no Docker Hub:

- `latest`: sempre a versao mais recente. E o padrao do `docker-compose.yml`.
- `AAAA-MM-DD`: tag imutavel da data da publicacao, preservada para rollback.

Como `latest` e sobrescrita a cada push, a tag datada e o unico caminho de volta.
Para voltar a uma versao anterior, defina `DYALOG_IMAGE_TAG` no `.env`:

```bash
DYALOG_IMAGE_TAG=2026-09-18 docker compose up -d
```

## O que ja funciona de verdade

- criar multiplas instancias
- abrir sessao real por instancia usando banco proprio em `SESSION_STORAGE_DIR/<instancia>/whatsmeow.db`
- gerar QR code real para pareamento
- gerar Pairing Code real por numero para novo pareamento
- reconectar instancias ja pareadas ao reiniciar
- enviar mensagem de texto real para privados e grupos
- responder mensagem citando a mensagem original
- enviar presenca de digitando/gravando audio
- enviar imagem real
- enviar audio real
- enviar documento real
- baixar e expor midias recebidas por webhook/API
- salvar midias recebidas localmente e, opcionalmente, enviar copia ao Supabase Storage
- receber webhooks de mensagens
- entregar webhooks por fila persistente com retry e auditoria
- receber webhooks de recibos de entrega, leitura e reproducao
- receber webhooks de digitando e gravando audio
- resolver numero real a partir de eventos com `@lid` apos o primeiro mapeamento conhecido
- cadastrar multiplos webhooks por instancia
- editar e excluir webhooks pela dashboard
- autenticacao da dashboard e da API por token
- proxy proprio por instancia e proxy global master como fallback

## Regras de autenticacao

A API aceita:

- `Authorization: Bearer SEU_TOKEN`
- `X-Access-Token: SEU_TOKEN`

Regras:

- token master: acesso total
- token de instancia: acesso apenas a propria instancia
- com token de instancia, o campo `instancia` pode ser omitido nos envios
- o token nao deve ser enviado no body JSON

## Variaveis de ambiente

- `APP_NAME`: nome da aplicacao
- `APP_ENV`: `development` ou `production`
- `APP_PORT`: porta HTTP
- `APP_VERSION`: versao logica da aplicacao
- `APP_COMMIT`: commit do build
- `APP_BUILD_DATE`: data do build
- `HTTP_LOG_MODE`: controla logs HTTP. Aceita `falhas`, `erros`, `todos` ou `desligado`. Padrao `falhas`.
- `WHATSAPP_LOG_LEVEL`: nivel do logger do whatsmeow. Aceita `ERROR`, `WARN`, `INFO`, `DEBUG` ou `TRACE`. Padrao `ERROR`.
- `DATABASE_DRIVER`: `sqlite` ou `postgres`
- `DATABASE_DSN`: caminho SQLite ou DSN Postgres/Supabase da aplicacao
- `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_HOST`, `DB_PORT`, `DB_SSLMODE`: alternativa para montar DSN Postgres quando `DATABASE_DSN` nao estiver preenchido
- `WHATSAPP_STORE_DRIVER`: `sqlite` ou `postgres`. Padrao `sqlite`; com `postgres`, o store interno do whatsmeow usa tabela compartilhada e vinculo por instancia.
- `WHATSAPP_STORE_DSN`: DSN Postgres opcional para o whatsmeow. Se vazio e a aplicacao usa Postgres, reutiliza o mesmo DSN.
- `SESSION_STORAGE_DIR`: diretorio de sessoes do `whatsmeow`
- `SESSION_DEVICE_NAME`: nome exibido como dispositivo/sistema no pareamento do WhatsApp quando suportado pelo WhatsApp
- `SESSION_CLIENT_TYPE`: tipo/icone conhecido usado no pareamento. Aceita `chrome`, `edge`, `firefox`, `safari`, `windows`, `macos`, `android`, `opera`, `electron`; `web` e tratado como `chrome`
- `SESSION_PAIRING_DISPLAY_NAME`: nome tecnico enviado no Pairing Code. Precisa usar formato aceito pelo WhatsApp, como `Chrome (Windows)`
- `TEMP_FILES_DIR`: diretorio temporario para arquivos
- `WEBHOOK_URL`: URL opcional para eventos futuros
- `WEBHOOK_MAX_ATTEMPTS`: quantidade maxima de tentativas por entrega de webhook. Padrao `60`, usado como trava de seguranca.
- `WEBHOOK_RETRY_BASE_SECONDS`: intervalo base do retry exponencial dos webhooks
- `WEBHOOK_RETRY_MAX_DURATION_HOURS`: janela maxima de retry por entrega. Padrao `24`.
- `WEBHOOK_RETRY_MAX_INTERVAL_MINUTES`: intervalo maximo entre retries. Padrao `30`.
- `WEBHOOK_WORKER_BATCH_SIZE`: tamanho do lote processado pela fila de webhooks
- `WEBHOOK_TIMEOUT_SECONDS`: tempo maximo de espera por resposta de cada endpoint de webhook. Padrao `5`.
- `WEBHOOK_WORKER_CONCURRENCY`: quantidade de entregas de webhook processadas em paralelo. Padrao `5`.
- `RUNTIME_NODE_ID`: identificador unico do container para ownership de instancias. Se vazio, usa o hostname do container.
- `RUNTIME_LOCK_TTL_SECONDS`: tempo para outro container poder assumir uma instancia se o dono parar de renovar o lock. Padrao `90`.
- `RUNTIME_HEARTBEAT_INTERVAL_SECONDS`: frequencia do heartbeat persistente da API. Padrao `30`.
- `INSTANCE_RECONNECT_INTERVAL_SECONDS`: frequencia com que a API procura instancias caidas e reconecta as que ainda tem sessao valida. Instancias aguardando QR code, aguardando codigo de pareamento ou sem dispositivo salvo nunca sao reconectadas sozinhas. Minimo `10`, padrao `30`.
- `WEBHOOK_RECOVERY_ENABLED`: habilita deteccao de janela offline e recuperacao por historico on-demand. Cobre API desligada e queda de conexao WhatsApp detectada por disconnect/keepalive. Padrao `true`.
- `WEBHOOK_RECOVERY_MARGIN_SECONDS`: margem adicionada antes/depois da janela offline ou da queda de conexao WhatsApp para recuperar mensagens. Padrao `120`.
- `WEBHOOK_RECOVERY_HISTORY_COUNT`: quantidade de mensagens solicitadas por conversa quando uma nova mensagem servir de ancora. Padrao `50`.
- `MEDIA_STORAGE_DRIVER`: `local`, `supabase` ou `s3`
- `MEDIA_STORAGE_SUPABASE_URL`: URL do projeto Supabase quando `MEDIA_STORAGE_DRIVER=supabase`
- `MEDIA_STORAGE_SUPABASE_KEY`: chave com permissao de escrita no Supabase Storage
- `MEDIA_STORAGE_SUPABASE_BUCKET`: bucket usado para midias recebidas
- `MEDIA_STORAGE_PUBLIC_BASE_URL`: URL publica opcional para montar links diretos de midia
- `MEDIA_STORAGE_S3_ENDPOINT`: endpoint do servico S3-compatible quando `MEDIA_STORAGE_DRIVER=s3`. Aceita com ou sem esquema; sem esquema assume `https`
- `MEDIA_STORAGE_S3_ACCESS_KEY`: access key do storage S3
- `MEDIA_STORAGE_S3_SECRET_KEY`: secret key do storage S3
- `MEDIA_STORAGE_S3_BUCKET`: bucket usado para midias recebidas
- `MEDIA_STORAGE_S3_REGION`: regiao do storage S3. No Cloudflare R2 use `auto`
- `MEDIA_LOCAL_RETENTION_DAYS`: dias para apagar do disco a midia que ja tem copia no storage externo. Midia sem copia externa nunca e apagada. Padrao `0`, que desliga a limpeza
- `WEBRTC_ICE_SERVERS`: lista JSON de servidores STUN/TURN usados nas chamadas, no mesmo formato que o navegador espera
- `TURN_URLS`: URLs do TURN separadas por virgula. Usa credencial temporaria, entao exige `TURN_SECRET`
- `TURN_SECRET`: segredo compartilhado com o coturn no modo `use-auth-secret`
- `TURN_TTL_SECONDS`: validade da credencial TURN entregue ao cliente. Padrao `43200`
- `CLOUDFLARE_TURN_KEY_ID`: key ID do TURN gerenciado da Cloudflare. Dispensa rodar coturn
- `CLOUDFLARE_TURN_API_TOKEN`: token da chave de TURN da Cloudflare
- `UPDATE_MONITORING_ENABLED`: habilita monitoramento da dependencia
- `UPDATE_MODE`: `aviso` ou `preparo`
- `UPDATE_WINDOW_START`: inicio da janela noturna, ex. `01:00`
- `UPDATE_WINDOW_END`: fim da janela noturna, ex. `02:00`
- `UPDATE_INTERVAL_MINUTES`: periodicidade de verificacao dentro da janela
- `UPDATE_APPLY_ENABLED`: permite gerar artefato de preparo
- `UPDATE_APPLY_TOKEN`: token exigido em `POST /api/v1/sistema/atualizacoes/aplicar`
- `UPDATE_PROXY_URL`: proxy Go usado para consultar a ultima versao
- `UPDATE_ARTIFACTS_DIR`: diretorio dos artefatos de preparo
- `DASHBOARD_MASTER_TOKEN`: token master da dashboard e da API
- `DASHBOARD_COOKIE_NAME`: nome do cookie de sessao da dashboard
- `API_BASE_URL`: URL publica base da API para montar `midia.download_url` nos webhooks
- `HISTORY_MAX_DAYS`: limite operacional de dias para importacao de historico inicial

## Base URL e autenticacao

Todas as rotas da API usam o prefixo `/api/v1`.

Base URL local/dominio:

```text
https://SEU_DOMINIO/api/v1
```

Exemplo de rota completa:

```text
POST https://SEU_DOMINIO/api/v1/batepapo/enviar/texto
```

Header recomendado para n8n e integrações:

```http
X-Access-Token: SEU_TOKEN_DA_INSTANCIA
```

Regras:

- token master: administra todas as instancias e rotas protegidas
- token da instancia: acessa apenas a propria instancia
- ao usar token da instancia, o campo `instancia` pode ser omitido no body
- nao envie o token no body JSON

## Resumo dos endpoints

Sistema:

- `GET /api/v1/saude`
- `GET /api/v1/sistema/versao`
- `GET /api/v1/sistema/atualizacoes`
- `GET /api/v1/sistema/proxy`
- `PUT /api/v1/sistema/proxy`
- `POST /api/v1/sistema/atualizacoes/verificar`
- `POST /api/v1/sistema/atualizacoes/aplicar`

Autenticacao:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/sessao`

Instancias:

- `POST /api/v1/instancias`
- `GET /api/v1/instancias`
- `GET /api/v1/instancias/:id`
- `DELETE /api/v1/instancias/:id`
- `POST /api/v1/instancias/:id/conectar`
- `POST /api/v1/instancias/:id/pairing-code`
- `POST /api/v1/instancias/:id/desconectar`
- `GET /api/v1/instancias/:id/status`
- `GET /api/v1/instancias/:id/qrcode`
- `GET /api/v1/instancias/:id/midia/:midiaId`
- `GET /api/v1/instancias/:id/webhooks`
- `POST /api/v1/instancias/:id/webhooks`
- `PUT /api/v1/instancias/:id/webhooks/:webhookId`
- `DELETE /api/v1/instancias/:id/webhooks/:webhookId`
- `GET /api/v1/instancias/:id/webhook-entregas`

Eventos de webhook suportados:

- `mensagens`: mensagens recebidas/enviadas e midias
- `recibos`: entrega, leitura e reproducao/ouvido de mensagens
- `status`: status operacional da instancia
- `digitando`: presenca de digitando
- `gravando_audio`: presenca de gravando audio
- `PUT /api/v1/instancias/:id/token`
- `PUT /api/v1/instancias/:id/historico`
- `PUT /api/v1/instancias/:id/proxy`
- `PUT /api/v1/instancias/:id/avancado`

Fluxos de pareamento:

- `POST /api/v1/instancias/:id/conectar`: inicia pareamento por QR code
- `POST /api/v1/instancias/:id/pairing-code`: gera Pairing Code para o numero informado
- o Pairing Code exige numero em formato internacional, somente com digitos
- o Pairing Code so funciona para nova conexao sem sessao ativa; se a instancia ja tiver sessao, desconecte primeiro

Bate-papo:

- `POST /api/v1/batepapo/enviar/texto`
- `POST /api/v1/batepapo/editar/texto`
- `POST /api/v1/batepapo/apagar`
- `POST /api/v1/batepapo/enviar/presenca`
- `POST /api/v1/user/presence` compatibilidade para `{"type":"unavailable"}`
- `POST /api/v1/batepapo/marcar-lida`
- `POST /api/v1/batepapo/enviar/botoes`
- `POST /api/v1/batepapo/enviar/lista`
- `POST /api/v1/batepapo/enviar/imagem`
- `POST /api/v1/batepapo/enviar/audio`
- `POST /api/v1/batepapo/enviar/documento`

Documentacao detalhada:

- [docs/endpoints.md](/D:/Sistemas/dyalog-api-go/docs/endpoints.md)
- [docs/dyalog-api.md](/D:/Sistemas/dyalog-api-go/docs/dyalog-api.md)
- [http://localhost:8080/api/v1/docs](http://localhost:8080/api/v1/docs)

## Fluxo aprovado para n8n

### 1. Webhook de mensagem recebida

Payload util para automacao:

- `body.dados.chat_numero`
- `body.dados.grupo`
- `body.dados.mensagem_id`
- `body.dados.acao` (`recebida`, `editada` ou `apagada`)
- `body.dados.remetente_jid`
- `body.dados.conteudo`

Edicoes e apagamentos tambem chegam no evento `mensagens`; filtre por `body.dados.acao` quando precisar separar o fluxo.

### 2. Resposta privada via API

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "{{ $json.body.dados.mensagem_id }}",
  "resposta_participante": "{{ $json.body.dados.remetente_jid }}",
  "resposta_conteudo": "{{ $json.body.dados.conteudo }}"
}
```

### 3. Resposta em grupo via API

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "grupo": true,
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "{{ $json.body.dados.mensagem_id }}",
  "resposta_participante": "{{ $json.body.dados.remetente_jid }}",
  "resposta_conteudo": "{{ $json.body.dados.conteudo }}"
}
```

### 4. Regra pratica de envio

- privado: use `numero`
- grupo: use `numero` + `grupo: true`
- reply citado: use `resposta_mensagem_id` + `resposta_participante`
- no n8n, prefira `Using JSON` em vez de `Using Fields Below`
- mande o token no header, nao no body
- o envio de texto tambem aceita compatibilidade WUZAPI com `Phone`, `Body`, `Id` e `ContextInfo`
- para simular digitando antes do texto, envie `delay` ou `delay_segundos` em segundos; tambem aceita `delay_ms`
- o delay envia `digitando`, aguarda o tempo informado e depois envia a mensagem; limite atual: 60 segundos

Exemplo HTTP/cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/texto" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: SEU_TOKEN_DA_INSTANCIA" \
  -d '{
    "numero": "5511999999999",
    "mensagem": "Teste pela API"
  }'
```

Exemplo com digitando por 3 segundos antes da mensagem:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/texto" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: SEU_TOKEN_DA_INSTANCIA" \
  -d '{
    "numero": "5511999999999",
    "mensagem": "Teste com digitando",
    "delay": 3
  }'
```

Tambem e possivel controlar apenas a presenca:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/presenca" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: SEU_TOKEN_DA_INSTANCIA" \
  -d '{
    "numero": "5511999999999",
    "acao": "digitando",
    "delay": 3
  }'
```

### 5. Envio de botoes rapidos

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem": "Escolha uma opcao",
  "rodape": "Dyalog",
  "botoes": [
    { "id": "sim", "texto": "Sim" },
    { "id": "nao", "texto": "Nao" }
  ]
}
```

Por padrao a API usa `modo: "native_flow"` com `InteractiveMessage` direto. Se precisar comparar envelopes, use `modo: "native_flow_view_once"`. O modo `buttons`/`legacy` fica disponivel apenas para diagnostico, porque servidores recentes do WhatsApp podem recusar esse formato com erro `405`. Se isso acontecer, a API faz fallback automatico para texto e informa isso no JSON de resposta. Se quiser forcar esse comportamento desde o inicio, use `modo: "texto"` ou `fallback_texto: true`.

Tambem ha compatibilidade com o formato WUZAPI `MessageTemplate`. Quando usar `Phone`, `Content`, `Footer` e `Buttons`, a API usa `modo: "template"` automaticamente:

```json
{
  "Phone": "5511999999999",
  "Content": "Escolha uma opcao",
  "Footer": "Dyalog",
  "Buttons": [
    { "DisplayText": "Sim", "Type": "quickreply" },
    { "DisplayText": "Nao", "Type": "quickreply" }
  ]
}
```

Esse mesmo payload deve ser enviado em `/api/v1/batepapo/enviar/botoes`. Nao existe alias ativo `/api/v1/chat/send/template` nesta versao.

### 6. Marcar mensagem como lida

Privado:

```json
{
  "numero": "5511999999999",
  "mensagem_id": "3EB06F9067F80BAB89FF"
}
```

Grupo:

```json
{
  "chat_jid": "120363000000000000@g.us",
  "grupo": true,
  "mensagem_id": "3EB06F9067F80BAB89FF",
  "participante": "225400000000000:54@lid"
}
```

Compatibilidade:

```json
{
  "Phone": "5511999999999",
  "Id": "3EB06F9067F80BAB89FF"
}
```

Esse payload deve ser enviado em `/api/v1/batepapo/marcar-lida`.

### 7. Envio de lista interativa

```json
{
  "numero": "5511999999999",
  "titulo": "Atendimento",
  "descricao": "Escolha uma opcao",
  "botao_texto": "Abrir lista",
  "rodape": "Dyalog",
  "opcoes": [
    { "id": "financeiro", "titulo": "Financeiro", "descricao": "Boletos e pagamentos" },
    { "id": "suporte", "titulo": "Suporte", "descricao": "Ajuda tecnica" }
  ]
}
```

Regras operacionais:

- a lista aceita de 1 a 10 itens no total
- `opcoes` cria uma secao unica automaticamente
- se precisar separar por grupos, use `secoes`
- por padrao a API usa `modo: "lista"` com `ListMessage`
- se o cliente nao renderizar a lista interativa, use `modo: "texto"` ou `fallback_texto: true`
- se o WhatsApp devolver `405`, a API faz fallback automatico para texto e responde isso no JSON

Tambem ha compatibilidade com o payload WUZAPI de lista. Quando usar `Phone`, `ButtonText`, `Desc`, `TopText`, `FooterText` e `List`, envie para `/api/v1/batepapo/enviar/lista` e a API converte automaticamente:

```json
{
  "Phone": "5511999999999",
  "TopText": "Atendimento",
  "Desc": "Escolha uma opcao",
  "ButtonText": "Abrir lista",
  "FooterText": "Dyalog",
  "List": [
    { "RowId": "financeiro", "title": "Financeiro", "desc": "Boletos e pagamentos" },
    { "RowId": "suporte", "title": "Suporte", "desc": "Ajuda tecnica" }
  ]
}
```

Esse payload deve ser enviado em `/api/v1/batepapo/enviar/lista`. Nao existe alias ativo `/api/v1/chat/send/list` nesta versao.

Quando o usuario clicar em um item da lista, o webhook chega como `tipo="lista"` com os campos `lista_id`, `lista_titulo`, `lista_descricao` e tambem dentro de `mensagem.lista`.

## Audio: regra importante

- `ogg/opus`: pode sair como gravado
- se `ptt` for omitido e o audio for `ogg/opus`, a API envia como gravado por padrao
- se quiser forcar explicitamente voz/gravado, envie `ptt: true`
- `ptt=true` so e valido com `ogg/opus`
- `mp3`, `m4a`, `aac`, `wav` e similares devem ser enviados como audio comum
- para audio gravado, nao basta mandar `mp3` com `ptt=true`; o formato precisa ser `ogg/opus`
- `duracao_segundos` e opcional; para `ogg/opus` a API tenta calcular automaticamente
- para base64 puro sem prefixo `data:...`, envie `mime_type` ou `nome_arquivo`

## Midias recebidas

Quando chega uma midia recebida, a API tenta baixar automaticamente o arquivo com o `whatsmeow`, salvar localmente e enriquecer o webhook de `mensagens` com:

- `midia.id`
- `midia.tipo`
- `midia.mime_type`
- `midia.nome_arquivo`
- `midia.tamanho_bytes`
- `midia.sha256`
- `midia.download_path`
- `midia.download_url` quando `API_BASE_URL` estiver configurada

Tambem ficam disponiveis campos planos de compatibilidade:

- `midia_id`
- `midia_download_path`
- `midia_download_url`
- `tamanho_bytes`
- `mime_type`
- `nome_arquivo`

Download protegido:

- `GET /api/v1/instancias/:id/midia/:midiaId`
- exige token master ou token da propria instancia


## Historico inicial

A quantidade de dias de historico deve ser definida antes de conectar a instancia. Use `0` para nao importar historico.

- endpoint: `PUT /api/v1/instancias/:id/historico`
- body: `{ "dias": 10 }`
- limite exibido pela API: `historico_max_dias`, configurado por `HISTORY_MAX_DAYS`
- o `whatsmeow` recebe o `HistorySync` do WhatsApp e a API filtra as mensagens pelo periodo configurado
- o WhatsApp/whatsmeow nao expoe um maximo garantido em dias nesse fluxo; o limite do projeto e operacional para evitar carga excessiva
- mensagens de historico enviadas ao webhook de `mensagens` chegam com `historico=true` e `origem="historico"`

## Recuperacao de janela offline

Quando `WEBHOOK_RECOVERY_ENABLED=true`, a API agenda uma janela de recuperacao em dois cenarios:

- o processo ficou desligado e o heartbeat persistente parou de atualizar
- a instancia perdeu conexao com o WhatsApp enquanto o processo continuou ligado, detectado por `Disconnected` ou timeout de keepalive do proprio whatsmeow

Ao reconectar, a API nao varre todos os chats imediatamente. Ela espera a proxima mensagem de cada conversa servir como ancora e solicita ate `WEBHOOK_RECOVERY_HISTORY_COUNT` mensagens daquele chat, filtrando pela janela com margem `WEBHOOK_RECOVERY_MARGIN_SECONDS`. Isso evita ping extra e reduz carga em producao.

## Multi-container

Para rodar mais de uma replica com seguranca, use Postgres para a aplicacao e para o store do WhatsApp:

```env
DATABASE_DRIVER=postgres
WHATSAPP_STORE_DRIVER=postgres
RUNTIME_LOCK_TTL_SECONDS=90
```

Cada instancia WhatsApp tem ownership em `instancia_runtime_locks`. Apenas o container dono pode manter o WebSocket daquela instancia; se ele parar de renovar o lock, outra replica pode assumir apos o TTL. A fila de webhooks tambem faz claim atomico das entregas para evitar envio duplicado.

Sem roteamento para o container dono, uma requisicao de envio pode cair em uma replica que nao possui aquela instancia e retornar `409 instancia_em_outro_container`. Isso e intencional para preservar a sessao; a etapa seguinte para escala plena e adicionar roteamento/proxy interno por owner.

## Proxy por instancia e proxy global

Cada instancia segue esta regra:

- sem proxy proprio: usa automaticamente o proxy global do master quando ele estiver ativo
- com proxy proprio: usa a URL configurada na propria instancia e deixa de usar o global

Endpoints:

- global master: `GET /api/v1/sistema/proxy` e `PUT /api/v1/sistema/proxy`
- instancia: `PUT /api/v1/instancias/:id/proxy`

Esquemas aceitos: `http`, `https` e `socks5`. Alterar proxy no `whatsmeow` exige reconectar o cliente; a API recarrega as instancias sem proxy proprio quando o proxy global muda e recarrega a propria instancia quando voce altera o proxy individual.

## Configuracoes avancadas da instancia

Endpoint: `PUT /api/v1/instancias/:id/avancado`

Campos persistentes:

- `manter_online`: padrao `false`; quando marcado, tenta manter a presenca global como disponivel
- `rejeitar_chamadas`: padrao `false`; quando marcado, rejeita chamadas recebidas automaticamente
- `mensagem_rejeitar_chamadas`: opcional; enviada apos rejeitar a chamada quando configurada
- `marcar_lida_automatico`: padrao `false`; marca mensagens recebidas como lidas automaticamente
- `ignorar_grupos`: padrao `false`; bloqueia envio de mensagens de grupo aos webhooks
- `ignorar_status`: padrao `false`; bloqueia envio de status do WhatsApp aos webhooks

## Nome e icone do dispositivo

O WhatsApp nao aceita icone customizado da Dyalog no dispositivo conectado. O icone exibido e escolhido pelo proprio WhatsApp a partir do tipo de cliente informado no pareamento.

Variaveis:

```env
SESSION_DEVICE_NAME=DyalogAPI
SESSION_CLIENT_TYPE=chrome
SESSION_PAIRING_DISPLAY_NAME=Chrome (Windows)
```

Regras:

- `SESSION_DEVICE_NAME` tenta alterar o nome/sistema exibido no pareamento por QR code
- `SESSION_CLIENT_TYPE` controla o tipo conhecido usado no QR/pareamento, por exemplo `chrome`, `edge`, `firefox`, `windows`, `macos`
- `SESSION_PAIRING_DISPLAY_NAME` e usado no Pairing Code e deve estar no formato `Navegador (Sistema)`, como `Chrome (Windows)`
- evite `SESSION_CLIENT_TYPE=other`, pois o WhatsApp pode exibir como `Outro dispositivo`
- o WhatsApp valida esse texto; nomes livres como `DyalogAPI` podem ser ignorados ou recusados no Pairing Code
- mudancas so afetam novo pareamento; sessoes ja conectadas precisam ser desconectadas/removidas e pareadas novamente

## Atualizacao segura do whatsmeow

O projeto monitora `go.mau.fi/whatsmeow` e grava o estado da verificacao no banco. O monitor roda apenas dentro da janela configurada e exibe aviso na dashboard.

Regras operacionais:

- nao existe hot reload do `whatsmeow` em binario Go ja em execucao
- atualizar dependencia exige `go get go.mau.fi/whatsmeow@latest`, novo build e novo processo
- `UPDATE_APPLY_ENABLED=false` por padrao
- producao fica bloqueada por padrao para aplicacao automatizada
- o caminho seguro e rebuild + rollout gradual + rollback rapido
