---
id: dyalog-api-go
loadWhen: cfg.channels.dyalog_api?.enabled || cfg.channels.whatsapp_dyalog?.enabled
tokensEstimate: 3500
verifiedAt: 2026-06-26
---

# Dyalog API GO - Runbook operacional

API propria de WhatsApp baseada em Go, Gin e `go.mau.fi/whatsmeow`.
Use este documento como referencia rapida para agentes, automacoes e integracoes externas.

## Quando carregar

Carregue este contexto quando o sistema mencionar Dyalog API, WhatsApp via whatsmeow, webhooks Dyalog, n8n, envio de mensagens, midias, instancias, QR code, pairing code, recibos ou presenca.

## O que este runbook garante

- Nao confundir Dyalog API com WuzAPI, WAHA ou Evolution API.
- Usar sempre o prefixo canonico `/api/v1`.
- Autenticar chamadas com `X-Access-Token`, nao com token no body.
- Separar token master de token de instancia.
- Tratar webhook como canal de entrada de eventos, nao como retorno de API.
- Lembrar que atualizacao de `whatsmeow` exige rebuild e novo deploy.
- Evitar prometer hot reload, envio garantido de botoes interativos ou historico ilimitado.

## Arquitetura resumida

- HTTP: Gin em `internal/http`.
- Regras de negocio: `internal/service`.
- WhatsApp/whatsmeow: `internal/whatsapp`.
- Persistencia: `internal/store`.
- Webhooks: `internal/webhook`.
- Dashboard: `internal/dashboard`.
- Documentacao principal de endpoints: `docs/endpoints.md`.

Fluxo recomendado:

```text
handler -> service -> whatsapp/store/webhook
```

Handlers nao devem chamar `whatsmeow` diretamente.

## Autenticacao

Header recomendado para API:

```http
X-Access-Token: SEU_TOKEN_DA_INSTANCIA
```

Tambem e aceito:

```http
Authorization: Bearer SEU_TOKEN
```

Regras:

- Token master administra todas as instancias.
- Token de instancia acessa somente a instancia vinculada.
- Com token de instancia, o campo `instancia` no body pode ser omitido.
- Com token master, informe `instancia` quando a rota depender de uma instancia.
- Nao envie `token` dentro do JSON body.

Base local comum:

```text
http://localhost:8080/api/v1
```

Exemplo em producao/local com dominio:

```text
https://apilocal.dyalog.com.br/api/v1
```

## Endpoints canonicos

### Sistema

| Metodo | Rota | Uso |
| --- | --- | --- |
| GET | `/api/v1/saude` | Healthcheck publico |
| GET | `/api/v1/docs` | Documentacao Markdown dos endpoints |
| GET | `/api/v1/sistema/versao` | Versao da app e whatsmeow |
| GET | `/api/v1/sistema/atualizacoes` | Status de atualizacao do whatsmeow |
| POST | `/api/v1/sistema/atualizacoes/verificar` | Verifica nova versao |
| POST | `/api/v1/sistema/atualizacoes/aplicar` | Stub/protegido para preparo controlado |
| GET | `/api/v1/sistema/proxy` | Proxy global |
| PUT | `/api/v1/sistema/proxy` | Atualiza proxy global |

### Autenticacao do dashboard

| Metodo | Rota | Uso |
| --- | --- | --- |
| POST | `/api/v1/auth/login` | Login com token master ou token de instancia |
| POST | `/api/v1/auth/logout` | Encerra sessao |
| GET | `/api/v1/auth/sessao` | Dados da sessao atual |

### Instancias

| Metodo | Rota | Uso |
| --- | --- | --- |
| POST | `/api/v1/instancias` | Cria instancia |
| GET | `/api/v1/instancias` | Lista instancias visiveis para o token |
| GET | `/api/v1/instancias/{id}` | Busca instancia |
| DELETE | `/api/v1/instancias/{id}` | Remove instancia |
| PUT | `/api/v1/instancias/{id}/token` | Troca token da instancia |
| PUT | `/api/v1/instancias/{id}/historico` | Configura dias de historico antes de conectar |
| PUT | `/api/v1/instancias/{id}/proxy` | Configura proxy proprio da instancia |
| PUT | `/api/v1/instancias/{id}/presenca` | Persiste presenca disponivel/indisponivel |
| PUT | `/api/v1/instancias/{id}/avancado` | Configuracoes avancadas |
| POST | `/api/v1/instancias/{id}/conectar` | Inicia conexao via QR |
| POST | `/api/v1/instancias/{id}/pairing-code` | Gera codigo de pareamento |
| POST | `/api/v1/instancias/{id}/desconectar` | Desconecta/remove sessao |
| GET | `/api/v1/instancias/{id}/status` | Status da instancia |
| GET | `/api/v1/instancias/{id}/qrcode` | Dados do QR code |
| GET | `/api/v1/instancias/{id}/qrcode/imagem` | Imagem PNG do QR code |
| GET | `/api/v1/instancias/{id}/midia/{midiaId}` | Download de midia recebida |

### Webhooks

| Metodo | Rota | Uso |
| --- | --- | --- |
| GET | `/api/v1/instancias/{id}/webhooks` | Lista webhooks |
| POST | `/api/v1/instancias/{id}/webhooks` | Cria webhook |
| PUT | `/api/v1/instancias/{id}/webhooks/{webhookId}` | Edita webhook |
| DELETE | `/api/v1/instancias/{id}/webhooks/{webhookId}` | Remove webhook |
| GET | `/api/v1/instancias/{id}/webhook-entregas` | Auditoria de entregas e retries |

### Bate-papo

| Metodo | Rota | Uso |
| --- | --- | --- |
| POST | `/api/v1/batepapo/enviar/texto` | Envia texto |
| POST | `/api/v1/batepapo/enviar/imagem` | Envia imagem |
| POST | `/api/v1/batepapo/enviar/audio` | Envia audio |
| POST | `/api/v1/batepapo/enviar/documento` | Envia documento |
| POST | `/api/v1/batepapo/enviar/botoes` | Tenta enviar botoes ou fallback texto |
| POST | `/api/v1/batepapo/enviar/lista` | Tenta enviar lista ou fallback texto |
| POST | `/api/v1/batepapo/enviar/presenca` | Envia digitando/gravando/pausado/disponivel |
| POST | `/api/v1/user/presence` | Compatibilidade para presenca |
| POST | `/api/v1/batepapo/marcar-lida` | Marca mensagem como lida |

## Estados de instancia

Estados conhecidos:

```text
desconectada
conectando
aguardando_qrcode
aguardando_codigo
conectada
desconectando
nao_inicializada
```

Observacoes:

- `aguardando_qrcode` deve exibir `/qrcode/imagem`.
- `aguardando_codigo` deve exibir o pairing code.
- Ao reiniciar a API, instancias com sessao valida devem autenticar novamente sem novo QR.
- Para forcar novo QR, desconectar removendo a sessao/dispositivo e conectar de novo.

## Webhooks

Eventos configuraveis:

```text
mensagens
status
digitando
gravando_audio
recibos
```

Regras importantes:

- Webhook deve receber `POST`.
- No n8n em modo teste, o node precisa estar escutando para receber.
- Eventos so devem ser enviados se o webhook estiver ativo e inscrito no evento.
- Entregas sao enfileiradas, persistidas e reenviadas com retry antes de serem marcadas como esgotadas.
- Auditoria por instancia: `GET /api/v1/instancias/{id}/webhook-entregas?limite=60`.
- Status/newsletter nao deve cair como mensagem comum.
- Token da instancia nao deve ser enviado no webhook por seguranca.

Payload base:

```json
{
  "evento": "mensagens",
  "instancia_id": "uuid",
  "ocorrido_em": "2026-06-26T10:00:00Z",
  "dados": {}
}
```

### Evento mensagens

Campos comuns em `dados`:

```json
{
  "mensagem_id": "3EB0...",
  "tipo": "texto",
  "conteudo": "Mensagem recebida",
  "direcao": "entrada",
  "enviado_por_mim": false,
  "grupo": false,
  "chat_jid": "556799440667@s.whatsapp.net",
  "chat_numero": "556799440667",
  "remetente_jid": "556799440667:54@s.whatsapp.net",
  "remetente_numero": "556799440667",
  "nome_remetente": "Contato",
  "recebida_em": "2026-06-26T10:00:00Z",
  "historico": false,
  "origem": "tempo_real"
}
```

Para midias, use `dados.mensagem.midia` ou `dados.midia`:

```json
{
  "id": "midia-id",
  "tipo": "imagem",
  "mime_type": "image/jpeg",
  "nome_arquivo": "3EB0_imagem.jpg",
  "tamanho_bytes": 318359,
  "download_path": "/api/v1/instancias/{id}/midia/{midiaId}",
  "storage_provider": "supabase",
  "storage_path": "instancias/{id}/20260629/midia-id.jpg",
  "storage_url": "https://..."
}
```

Campos `storage_*` aparecem somente quando `MEDIA_STORAGE_DRIVER=supabase` estiver configurado. A API continua salvando copia local para download protegido.

### Evento recibos

Usado para entregue, lida, reproduzida/ouvida e eventos similares quando o whatsmeow disponibilizar recibo.

Campos esperados:

```json
{
  "mensagem_id": "3EB0...",
  "status": "lida",
  "chat_jid": "556799440667@s.whatsapp.net",
  "chat_numero": "556799440667",
  "remetente_jid": "556799440667@s.whatsapp.net",
  "remetente_numero": "556799440667",
  "grupo": false
}
```

## Payloads de envio

### Texto

```json
{
  "numero": "6799440667",
  "mensagem": "Resposta pela API",
  "delay": 3
}
```

Resposta citando mensagem:

```json
{
  "numero": "6799440667",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "3EB0...",
  "resposta_participante": "556799440667:54@s.whatsapp.net",
  "resposta_conteudo": "Mensagem original"
}
```

Grupo:

```json
{
  "numero": "120363409010682790",
  "grupo": true,
  "mensagem": "Mensagem no grupo"
}
```

### Imagem

```json
{
  "numero": "6799440667",
  "arquivo_url": "https://exemplo.com/imagem.jpg",
  "legenda": "Legenda"
}
```

Base64:

```json
{
  "numero": "6799440667",
  "arquivo_base64": "data:image/png;base64,iVBORw0KGgo...",
  "legenda": "Imagem por base64"
}
```

### Documento

```json
{
  "numero": "6799440667",
  "arquivo_url": "https://exemplo.com/arquivo.pdf",
  "nome_arquivo": "contrato.pdf",
  "mime_type": "application/pdf"
}
```

### Audio

Audio comum:

```json
{
  "numero": "6799440667",
  "arquivo_url": "https://exemplo.com/audio.mp3",
  "mime_type": "audio/mpeg"
}
```

Audio como gravado/PTT:

```json
{
  "numero": "6799440667",
  "arquivo_base64": "data:audio/ogg;base64,T2dnUwAC...",
  "mime_type": "audio/ogg; codecs=opus",
  "ptt": true
}
```

Observacoes:

- Para parecer audio gravado, prefira OGG/Opus com `ptt: true`.
- `duracao_segundos` e opcional; a API tenta inferir quando possivel.

### Presenca

Digitando:

```json
{
  "numero": "6799440667",
  "acao": "digitando",
  "delay": 3
}
```

Gravando audio:

```json
{
  "numero": "6799440667",
  "acao": "gravando_audio",
  "delay": 3
}
```

Pausado:

```json
{
  "numero": "6799440667",
  "acao": "pausado"
}
```

Compatibilidade:

```json
{
  "numero": "6799440667",
  "type": "unavailable"
}
```

### Marcar como lida

```json
{
  "numero": "6799440667",
  "mensagem_id": "3EB0...",
  "participante": "556799440667:54@s.whatsapp.net"
}
```

Compatibilidade:

```json
{
  "Phone": "6799440667",
  "Id": "3EB0...",
  "Participant": "556799440667:54@s.whatsapp.net"
}
```

## Regras para n8n

Configuracao recomendada do node HTTP Request:

- Method: `POST`.
- Authentication: `None`.
- Send Headers: `true`.
- Header: `X-Access-Token = token_da_instancia`.
- Body Content Type: `JSON`.
- Specify Body: `Using JSON`.
- URL sempre com `/api/v1`.

Exemplo texto:

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "{{ $json.body.dados.mensagem_id }}",
  "resposta_participante": "{{ $json.body.dados.remetente_jid }}",
  "resposta_conteudo": "{{ $json.body.dados.conteudo }}"
}
```

Delay dinamico no n8n deve ser calculado fora do JSON ou em node Code/Set.
Nao cole expressao JavaScript crua dentro do JSON se ela quebrar a validade do body.

Expressao segura em campo proprio do n8n:

```text
={{ Math.min(45, Math.max(2, Math.ceil(String($json.messages || '').length / 25))) }}
```

## Falhas conhecidas e diagnostico

| Sintoma | Causa comum | Acao |
| --- | --- | --- |
| `401 nao_autenticado` | Header ausente/invalido | Enviar `X-Access-Token` |
| `403 Acesso restrito ao token master` | Token de instancia acessando rota master | Usar token master ou rota permitida |
| `Campos obrigatorios: mensagem e numero ou chat_jid` | Body incompleto ou JSON invalido | Conferir body e usar `Using JSON` |
| QR aparece como texto | Usou rota de dados do QR | Usar `/qrcode/imagem` |
| Grupo da erro `no LID found` | Enviou ID de grupo como contato | Usar `grupo: true` |
| Botoes aceitos mas nao renderizam | Filtro/limitacao do WhatsApp | Usar fallback texto/lista |
| Webhook chega vazio no n8n | Node em modo incorreto ou URL teste expirada | Ativar teste ou usar URL de producao |
| Status/newsletter chegando como mensagem | Build antigo ou filtro faltando | Atualizar imagem e validar filtro |
| `database is locked` | Processo duplicado/SQLite concorrente | Parar duplicado ou usar Postgres |
| Aparece "Outro dispositivo" | Tipo/nome do device ja pareado | Ajustar env e parear de novo |

## Atualizacao do whatsmeow

Atualizacao de dependencia Go exige novo build e novo processo. Nao existe hot reload do `whatsmeow` dentro do binario em execucao.

Fluxo local:

```powershell
go mod tidy
go get go.mau.fi/whatsmeow@latest
go mod tidy
go build ./cmd/api
go run ./cmd/api
```

Fluxo Docker:

```powershell
docker build -t aliffstefano/dyalog-api:latest -t aliffstefano/dyalog-api:v1 .
docker push aliffstefano/dyalog-api:latest
docker push aliffstefano/dyalog-api:v1
```

Em Portainer, atualizacao real exige pull/redeploy da imagem nova.

## Docker e persistencia

Postgres/Supabase pode ser usado para persistencia da aplicacao.
Sessoes/dispositivos do WhatsApp ainda dependem do diretorio configurado em `SESSION_STORAGE_DIR` e devem ficar em volume persistente.

Variaveis comuns:

```env
DASHBOARD_MASTER_TOKEN=troque-este-token
DATABASE_URL=postgres://usuario:senha@host:5432/dyalog_api?sslmode=disable
SESSION_STORAGE_DIR=/app/data/sessoes
SESSION_DEVICE_NAME=DyalogAPI
SESSION_CLIENT_TYPE=chrome
TZ=America/Cuiaba
```

Para mudar como o dispositivo aparece no WhatsApp, remova o dispositivo pareado no celular e conecte novamente.

## Contrato minimo de adapter

Um adapter externo deve implementar pelo menos:

```ts
type DyalogAdapter = {
  listarInstancias(): Promise<Instancia[]>
  obterStatus(instanciaId: string): Promise<StatusInstancia>
  conectar(instanciaId: string): Promise<void>
  gerarPairingCode(instanciaId: string, telefone: string): Promise<string>
  configurarWebhooks(instanciaId: string, webhooks: WebhookConfig[]): Promise<void>
  enviarTexto(input: EnviarTexto): Promise<EnvioResposta>
  enviarMidia(input: EnviarMidia): Promise<EnvioResposta>
  enviarPresenca(input: EnviarPresenca): Promise<void>
  marcarComoLida(input: MarcarLida): Promise<void>
  parseWebhook(payload: unknown): EventoDyalog
}
```

## Checks antes de dizer pronto

- `go test ./...` executa sem erro.
- Dashboard abre e autentica com token master e token de instancia.
- Token de instancia seleciona automaticamente sua instancia.
- Webhook edita, salva e respeita eventos marcados.
- Envio de texto, imagem, audio e documento retorna `sucesso: true`.
- Recebimento de texto e midia chega no webhook.
- Status/newsletter nao vaza como mensagem comum.
- Recibos so chegam se evento `recibos` estiver marcado.
- Imagem Docker foi rebuildada e redeployada quando dependencia mudar.

## Fonte

- Codigo-fonte local do projeto Dyalog API GO.
- `docs/endpoints.md` para documentacao detalhada de API.
- `go.mau.fi/whatsmeow` como nucleo WhatsApp.
