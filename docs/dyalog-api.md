---
id: dyalog-api-go
loadWhen: cfg.channels.dyalog_api?.enabled || cfg.channels.whatsapp_dyalog?.enabled
tokensEstimate: 5200
verifiedAt: 2026-09-16
---

# Dyalog API GO - Runbook operacional

API propria de WhatsApp baseada em Go, Gin e `go.mau.fi/whatsmeow`.
Use este documento como referencia rapida para agentes, automacoes e integracoes externas.

## Quando carregar

Carregue este contexto quando o sistema mencionar Dyalog API, WhatsApp via whatsmeow, webhooks Dyalog, n8n, envio de mensagens, midias, instancias, QR code, pairing code, recibos, presenca ou chamadas de voz (VoIP/WebRTC).

## O que este runbook garante

- Nao confundir Dyalog API com WuzAPI, WAHA ou Evolution API.
- Usar sempre o prefixo canonico `/api/v1`.
- Autenticar chamadas com `X-Access-Token`, nao com token no body.
- Separar token master de token de instancia.
- Tratar webhook como canal de entrada de eventos, nao como retorno de API.
- Lembrar que atualizacao de `whatsmeow` exige rebuild e novo deploy.
- Tratar chamadas como estado em memoria: sem WebRTC negociado nao ha audio.
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
https://SEU_DOMINIO/api/v1
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

### Chamadas (VoIP)

Rotas canonicas (a instancia vem do `X-Access-Token`):

| Metodo | Rota | Uso |
| --- | --- | --- |
| POST | `/api/v1/chamadas/iniciar` | Inicia chamada para `numero` ou `chat_jid` |
| POST | `/api/v1/chamadas/{chamadaId}/webrtc` | Negocia audio: envia `sdp_offer`, recebe `sdp_answer` |
| POST | `/api/v1/chamadas/{chamadaId}/aceitar` | Aceita chamada recebida |
| POST | `/api/v1/chamadas/{chamadaId}/rejeitar` | Rejeita chamada recebida |
| DELETE | `/api/v1/chamadas/{chamadaId}` | Encerra chamada em andamento |

Rotas equivalentes com instancia explicita na URL (aceitam token master):

| Metodo | Rota | Uso |
| --- | --- | --- |
| GET | `/api/v1/instancias/{id}/chamadas` | Lista chamadas ativas da instancia |
| POST | `/api/v1/instancias/{id}/chamadas/{chamadaId}/webrtc` | Negocia audio da chamada |
| POST | `/api/v1/instancias/{id}/chamadas/{chamadaId}/aceitar` | Aceita chamada recebida |
| POST | `/api/v1/instancias/{id}/chamadas/{chamadaId}/rejeitar` | Rejeita chamada recebida |
| DELETE | `/api/v1/instancias/{id}/chamadas/{chamadaId}` | Encerra chamada em andamento |

As duas formas caem nos mesmos handlers. Use a forma com `/instancias/{id}` quando o
chamador usa token master ou controla varias instancias; use a forma curta quando o
token ja identifica a instancia. Nao existe `POST /api/v1/instancias/{id}/chamadas/iniciar`:
para iniciar, use `/api/v1/chamadas/iniciar` com o campo `instancia` no corpo.

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
- Instancias com sessao valida que caem durante a execucao voltam sozinhas; veja Reconexao automatica.
- Para forcar novo QR, desconectar removendo a sessao/dispositivo e conectar de novo.

## Reconexao automatica

A API reconecta sozinha as instancias que caem, desde que a sessao ainda seja valida.
Duas camadas cobrem isso:

1. O auto-reconnect do proprio whatsmeow, que trata a maior parte das quedas de rede.
2. Um supervisor da API que roda a cada `INSTANCE_RECONNECT_INTERVAL_SECONDS` e pega os
   casos em que o whatsmeow desiste: sessao assumida por outra conexao (`stream replaced`),
   falha de conexao nao reconhecida e erro de stream. Nesses casos a instancia ficava
   parada ate alguem clicar em conectar, e nesse meio tempo nenhuma mensagem chegava.

O supervisor so reconecta instancia que tem dispositivo salvo no store, ou seja, que volta
sem pedir QR code.

Nunca sao reconectadas sozinhas:

- `aguardando_qrcode` e `aguardando_codigo`: dependem de alguem ler o QR ou digitar o codigo.
- `nao_inicializada`: sem dispositivo salvo. Cai aqui quem foi desconectado manualmente pela
  API ou deslogado pelo celular (`events.LoggedOut`), inclusive quando o WhatsApp exige login novo.
- instancia em banimento temporario, ate o prazo informado pelo WhatsApp expirar.
- instancia com cliente desatualizado (erro 405), por 1 hora. O que resolve e atualizar o whatsmeow.
- instancia cujo status salvo no banco e `nao_inicializada`, `aguardando_qrcode` ou `aguardando_codigo`.
- instancia que pertence a outra replica: quem reconecta e o container dono.

O status salvo no banco tem a palavra final. Desconectar pela API faz logout, apaga o
dispositivo e marca `nao_inicializada`, entao a instancia nao volta sozinha.

## Webhooks

Eventos configuraveis:

```text
mensagens
status
digitando
gravando_audio
recibos
chamadas
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
  "chat_jid": "5511999999999@s.whatsapp.net",
  "chat_numero": "5511999999999",
  "remetente_jid": "5511999999999:54@s.whatsapp.net",
  "remetente_numero": "5511999999999",
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
  "chat_jid": "5511999999999@s.whatsapp.net",
  "chat_numero": "5511999999999",
  "remetente_jid": "5511999999999@s.whatsapp.net",
  "remetente_numero": "5511999999999",
  "grupo": false
}
```

### Evento chamadas

Disparado quando a instancia esta inscrita no evento `chamadas`.

Acoes possiveis em `dados.acao`:

```text
recebida
estado
encerrada
```

Campos esperados:

```json
{
  "acao": "recebida",
  "id": "CALL_ID",
  "chamada_id": "CALL_ID",
  "peer_jid": "5511999999999@s.whatsapp.net",
  "peer_numero": "5511999999999",
  "numero": "5511999999999",
  "caller_pn": "5511999999999",
  "call_creator": "5511999999999@s.whatsapp.net",
  "direcao": "incoming",
  "estado": "incoming_ringing",
  "tipo": "audio",
  "criada_em": "2026-06-26T10:00:00Z",
  "api": {
    "aceitar": "/api/v1/chamadas/CALL_ID/aceitar",
    "rejeitar": "/api/v1/chamadas/CALL_ID/rejeitar",
    "encerrar": "/api/v1/chamadas/CALL_ID",
    "webrtc": "/api/v1/chamadas/CALL_ID/webrtc"
  }
}
```

Regras:

- `api` traz caminhos relativos ja com o `chamada_id` preenchido; prefixe com `API_BASE_URL`.
- `id` e `chamada_id` sao o mesmo valor, mantidos por compatibilidade.
- `numero` vem de `caller_pn` quando disponivel; senao e derivado do `peer_jid`.
- Nenhum audio trafega no webhook; ele so avisa. O audio exige negociar WebRTC.

## Payloads de envio

### Texto

```json
{
  "numero": "11999999999",
  "mensagem": "Resposta pela API",
  "delay": 3
}
```

Resposta citando mensagem:

```json
{
  "numero": "11999999999",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "3EB0...",
  "resposta_participante": "5511999999999:54@s.whatsapp.net",
  "resposta_conteudo": "Mensagem original"
}
```

Grupo:

```json
{
  "numero": "120363000000000000",
  "grupo": true,
  "mensagem": "Mensagem no grupo"
}
```

### Imagem

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/imagem.jpg",
  "legenda": "Legenda"
}
```

Base64:

```json
{
  "numero": "11999999999",
  "arquivo_base64": "data:image/png;base64,iVBORw0KGgo...",
  "legenda": "Imagem por base64"
}
```

### Documento

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/arquivo.pdf",
  "nome_arquivo": "contrato.pdf",
  "mime_type": "application/pdf"
}
```

### Audio

Audio comum:

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/audio.mp3",
  "mime_type": "audio/mpeg"
}
```

Audio como gravado/PTT:

```json
{
  "numero": "11999999999",
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
  "numero": "11999999999",
  "acao": "digitando",
  "delay": 3
}
```

Gravando audio:

```json
{
  "numero": "11999999999",
  "acao": "gravando_audio",
  "delay": 3
}
```

Pausado:

```json
{
  "numero": "11999999999",
  "acao": "pausado"
}
```

Compatibilidade:

```json
{
  "numero": "11999999999",
  "type": "unavailable"
}
```

### Marcar como lida

```json
{
  "numero": "11999999999",
  "mensagem_id": "3EB0...",
  "participante": "5511999999999:54@s.whatsapp.net"
}
```

Compatibilidade:

```json
{
  "Phone": "11999999999",
  "Id": "3EB0...",
  "Participant": "5511999999999:54@s.whatsapp.net"
}
```

## Chamadas (VoIP)

Chamadas de audio WhatsApp sao suportadas de ponta a ponta: sinalizacao via whatsmeow e
audio via WebRTC entre a API e a aplicacao (CRM/navegador).

### Estados da chamada

```text
initiating
ringing
incoming_ringing
connecting
active
on_hold
ended
```

Outros campos:

- `direcao`: `outgoing` ou `incoming`.
- `tipo`: `audio` ou `video`.
- motivos de encerramento internos: `user_ended`, `declined`, `timeout`, `busy`,
  `cancelled`, `failed`, `do_not_disturb`, `unknown`.

### Fluxo de chamada de saida

1. `POST /api/v1/chamadas/iniciar` com `{"numero":"11999999999"}` ou `{"chat_jid":"...@s.whatsapp.net"}`.
   Campo opcional `video: true`. A resposta traz `chamada_id` e `estado: ringing`.
2. A aplicacao cria uma `RTCPeerConnection` com uma track de audio e gera o offer.
3. `POST /api/v1/chamadas/{chamadaId}/webrtc` com `{"sdp_offer":"v=0..."}`.
   A resposta traz `sdp_answer`, que a aplicacao aplica como remote description.
4. Quando o outro lado atende, o webhook envia `acao: estado` com `estado: active`.
5. `DELETE /api/v1/chamadas/{chamadaId}` encerra.

### Fluxo de chamada recebida

1. Webhook `chamadas` chega com `acao: recebida` e `estado: incoming_ringing`.
2. A aplicacao chama `api.aceitar` (ou `api.rejeitar`).
3. A aplicacao negocia o audio em `api.webrtc`, igual ao passo 3 do fluxo de saida.
4. `api.encerrar` (DELETE) finaliza.

Chamadas recebidas nao sao atendidas sozinhas: sem chamar `aceitar`, a chamada toca ate expirar.

### Audio

Resposta de `/webrtc`:

```json
{
  "instancia": "ID_DA_INSTANCIA",
  "id": "CALL_ID",
  "chamada_id": "CALL_ID",
  "sdp_answer": "v=0...",
  "transporte": "media_track",
  "audio_envio": "opus_track_browser_para_api",
  "audio_retorno": "pcmu_track_api_para_browser"
}
```

- Modo recomendado: `MediaStreamTrack` padrao do navegador.
- Aplicacao para API: track Opus.
- API para aplicacao: track PCMU.
- Fallback legado: DataChannel WebRTC chamado `pcm`, PCM mono 16 kHz, Int16 little-endian, nos dois sentidos.

### Rejeicao automatica

Em `PUT /api/v1/instancias/{id}/avancado`:

- `rejeitar_chamadas`: padrao `false`; quando `true`, toda chamada recebida e rejeitada automaticamente.
- `mensagem_rejeitar_chamadas`: opcional; enviada ao contato apos a rejeicao.

Com `rejeitar_chamadas` ligado nao existe chamada ativa para aceitar, e o webhook
`chamadas` nao recebe `acao: recebida` para essas chamadas.

### Limites e falhas conhecidas

- A instancia precisa estar conectada e logada; senao a resposta e `instancia nao conectada`.
- `chamada_id` so existe enquanto a chamada esta ativa em memoria; apos encerrar,
  qualquer acao retorna `chamada nao encontrada` com a lista de ids ativos no erro.
- Chamadas sao estado em memoria do container dono da instancia. Em multi-container o
  middleware de proxy encaminha a requisicao para a replica dona (pelo `:id` da URL, pelo
  campo `instancia` do corpo ou pelo token), entao as rotas de chamada funcionam em
  qualquer replica; a midia WebRTC, porem, e negociada com a replica dona, que precisa
  estar alcancavel pela aplicacao.
- Reiniciar a API derruba as chamadas ativas.
- `video: true` e aceito na sinalizacao, mas a ponte de midia trata audio.

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

## Recuperacao de janela offline

`WEBHOOK_RECOVERY_ENABLED=true` cobre duas falhas:

- API desligada: detectada pelo heartbeat persistente `sistema_runtime`.
- Internet/WhatsApp fora com API ligada: detectada por `Disconnected` ou timeout de keepalive do whatsmeow.

Ao voltar, a API registra uma janela com `WEBHOOK_RECOVERY_MARGIN_SECONDS` antes/depois. A recuperacao e on-demand: a proxima mensagem de cada chat vira ancora para solicitar `WEBHOOK_RECOVERY_HISTORY_COUNT` mensagens daquele chat via `HistorySync`, e a API filtra somente o periodo da janela.

## Store WhatsApp em Postgres

Por padrao, o store interno do whatsmeow usa SQLite por instancia em `SESSION_STORAGE_DIR/<instancia>/whatsmeow.db`.

Para centralizar esse store no Postgres:

```env
WHATSAPP_STORE_DRIVER=postgres
```

Se `WHATSAPP_STORE_DSN` estiver vazio, a API reutiliza o Postgres da aplicacao (`DATABASE_DSN` ou `DB_*`). O vinculo entre instancia e device fica em `whatsapp_devices`, evitando que uma instancia carregue o device de outra em um store compartilhado.

## Storage de midia

Midia recebida sempre grava no disco do container. O `MEDIA_STORAGE_DRIVER` define
se, alem disso, ela sobe para um storage externo.

| Driver | Uso |
| --- | --- |
| `local` | so disco do container. Padrao |
| `supabase` | Supabase Storage, via REST proprio do Supabase |
| `s3` | qualquer servico compativel com S3: Cloudflare R2, Garage, MinIO, Backblaze B2, Wasabi |

Com varias replicas, `local` e problematico: o arquivo fica no disco da replica que
recebeu a mensagem, e um download que caia em outra replica nao acha. Storage externo
resolve isso, porque todas as replicas escrevem e leem do mesmo lugar.

### Driver s3

Variaveis: `MEDIA_STORAGE_S3_ENDPOINT`, `MEDIA_STORAGE_S3_ACCESS_KEY`,
`MEDIA_STORAGE_S3_SECRET_KEY`, `MEDIA_STORAGE_S3_BUCKET` e `MEDIA_STORAGE_S3_REGION`.

O endpoint aceita com ou sem esquema; sem esquema, assume `https`. Trocar de um
servico compativel para outro e so mudar as variaveis, sem rebuild.

Cloudflare R2:

```text
MEDIA_STORAGE_S3_ENDPOINT=https://SEU_ACCOUNT_ID.r2.cloudflarestorage.com
MEDIA_STORAGE_S3_REGION=auto
```

Garage ou MinIO:

```text
MEDIA_STORAGE_S3_ENDPOINT=http://garage.interno:3900
MEDIA_STORAGE_S3_REGION=garage
```

### URL publica

`MEDIA_STORAGE_PUBLIC_BASE_URL` e o endereco usado para montar o link salvo em
`storage_url`. Sem ela, o link sai apontando direto para o endpoint com o bucket no
caminho, o que so funciona se o bucket for publico.

- No R2, use um dominio custom ligado ao bucket. O endereco `r2.dev` e so para
  desenvolvimento: tem limite de requisicoes e a Cloudflare nao recomenda em producao.
- No Garage, use o modo website do bucket.

### Um driver por vez

So um driver fica ativo. Nao da para gravar em dois storages ao mesmo tempo: cada
midia guarda um unico `storage_provider`, `storage_path` e `storage_url`.

Trocar de driver vale so para midia nova. As antigas continuam onde foram gravadas,
e os links ja salvos no banco continuam apontando para la. Ou seja, o storage antigo
precisa continuar de pe enquanto esses links importarem, ou a midia antiga precisa
ser migrada e os links reescritos.

### Limpeza da copia local

Midia recebida sempre grava no disco, mesmo com storage externo ligado. Nada
apagava esses arquivos, entao a pasta crescia para sempre.

`MEDIA_LOCAL_RETENTION_DAYS` liga a limpeza: uma vez por dia, as 5h no fuso do
container, arquivos mais antigos que N dias sao removidos do disco.

Duas garantias importantes:

- So entra na limpeza midia que **ja tem copia no storage externo**, ou seja, com
  `storage_provider` e `storage_url` preenchidos. Midia sem copia fica no disco
  para sempre: apagar seria perder o arquivo, nao liberar espaco.
- O padrao e `0`, que desliga tudo. Ninguem perde arquivo por atualizar a API.

O registro no banco continua; so o `caminho_arquivo` e zerado. E o endpoint
`GET /api/v1/instancias/{id}/midia/{midiaId}` passa a redirecionar para a
`storage_url` quando o arquivo local ja saiu, entao o download continua funcionando.

Pastas de dia que ficam vazias sao removidas junto.

### Compatibilidade

O driver usa `minio-go`, nao o SDK oficial da AWS: versoes recentes do
`aws-sdk-go-v2` mandam headers de checksum por padrao que varios servicos
compativeis rejeitam, o R2 entre eles, e o sintoma aparece como erro de assinatura.

O upload tambem desliga a assinatura streaming do `minio-go`, que quebraria o corpo
em blocos `aws-chunked` em endpoint sem TLS e nao e aceita por todo servico
compativel. No lugar vai `Content-MD5`, entao o servidor continua conferindo
integridade.

## Multi-container

Multi-container e seguro quando o Postgres esta ativo e o ownership por instancia esta habilitado. A API usa a tabela `instancia_runtime_locks` para garantir que uma instancia WhatsApp tenha somente um container dono por vez.

Variaveis:

- `RUNTIME_NODE_ID`: identificador unico do container. Se vazio, usa hostname.
- `RUNTIME_LOCK_TTL_SECONDS`: tempo para failover quando o dono para de renovar. Padrao `90`.
- `RUNTIME_HEARTBEAT_INTERVAL_SECONDS`: intervalo de renovacao. Deve ser menor que o TTL.
- `INSTANCE_RECONNECT_INTERVAL_SECONDS`: intervalo da reconexao automatica de instancias. Padrao `30`.

Comportamento:

- so o dono conecta o WebSocket da instancia
- se o dono cair, outro container pode assumir apos o TTL
- entregas de webhook usam claim atomico para evitar duplicidade
- se uma chamada de envio cair em uma replica que nao e dona da instancia, a API retorna `409 instancia_em_outro_container`

Para escala horizontal completa, adicione roteamento interno para encaminhar comandos WhatsApp ao container dono. Sem isso, multiplas replicas aumentam disponibilidade/failover e capacidade HTTP geral, mas nao garantem que todo request de envio caia no owner correto.

Ao trocar de SQLite para Postgres, sessoes antigas em `whatsmeow.db` nao sao migradas automaticamente; as instancias precisam ser pareadas novamente ou migradas por rotina propria.

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
