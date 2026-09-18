# Endpoints Iniciais

## Acesso e autenticacao

Todas as rotas da API versionada usam o prefixo `/api/v1`.

Nos exemplos, `SEU_DOMINIO` representa o dominio onde a sua API esta publicada.
Troque pelo seu antes de rodar os comandos.

Base URL:

```text
https://SEU_DOMINIO/api/v1
```

Exemplo completo:

```text
POST https://SEU_DOMINIO/api/v1/batepapo/enviar/texto
```

A API aceita autenticacao por header. O header recomendado para n8n e integracoes externas e:

```http
X-Access-Token: SEU_TOKEN_DA_INSTANCIA
```

Opcoes suportadas:

- `Authorization: Bearer SEU_TOKEN`
- `X-Access-Token: SEU_TOKEN`

Regras:

- token master: pode acessar todas as instancias e rotas protegidas
- token de instancia: fica restrito a propria instancia
- quando usar token de instancia, o campo `instancia` pode ser omitido do body
- o token nao deve ser enviado no body JSON
- a documentacao tambem esta disponivel em `GET /api/v1/docs`

## Guia rapido para n8n e curl

Use este guia quando estiver configurando um node **HTTP Request** no n8n.

Configuracao padrao do node:

- `Method`: `POST`
- `Authentication`: `None`
- `Send Headers`: `true`
- `Body Content Type`: `JSON`
- `Specify Body`: `Using JSON`
- header obrigatorio: `X-Access-Token`
- o token deve ir no header, nao no body
- usando token de instancia, nao precisa enviar `instancia`
- usando token master, envie tambem `instancia`

URL base:

```text
https://SEU_DOMINIO/api/v1
```

Em ambiente local, use a porta publicada pelo container:

```text
http://localhost:8080/api/v1
```

### Texto

Endpoint:

```text
POST /batepapo/enviar/texto
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- `mensagem`

Campos opcionais mais usados:

- `delay`: segundos de digitando antes de enviar
- `grupo`: `true` quando o destino for grupo
- `resposta_mensagem_id`: ID da mensagem que sera respondida/citada
- `resposta_participante`: JID do remetente original, recomendado para reply
- `resposta_conteudo`: texto original exibido na resposta

JSON para n8n:

```json
{
  "numero": "11999999999",
  "mensagem": "Ola, tudo bem?",
  "delay": 3
}
```

cURL para importar no n8n:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/texto" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","mensagem":"Ola, tudo bem?","delay":3}'
```

Texto respondendo/citando mensagem recebida:

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "{{ $json.body.dados.mensagem.id }}",
  "resposta_participante": "{{ $json.body.dados.remetente_jid }}",
  "resposta_conteudo": "{{ $json.body.dados.conteudo }}",
  "delay": 3
}
```

Envio para grupo:

```json
{
  "numero": "120363000000000000",
  "grupo": true,
  "mensagem": "Ola grupo"
}
```

### Editar texto

Endpoint:

```text
POST /batepapo/editar/texto
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- `mensagem_id`: ID da mensagem enviada pela API/instancia
- `mensagem`: novo texto

Observacoes:

- A edicao so funciona para mensagens enviadas pela propria instancia.
- O WhatsApp limita a janela de edicao. No whatsmeow essa janela e de aproximadamente 20 minutos.
- Para grupo, envie `grupo: true` quando usar o ID numerico do grupo em `numero`.

JSON para n8n:

```json
{
  "numero": "11999999999",
  "mensagem_id": "3EB0D174A6979E7104AD8F",
  "mensagem": "Texto editado pela API"
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/editar/texto" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","mensagem_id":"3EB0D174A6979E7104AD8F","mensagem":"Texto editado pela API"}'
```

### Apagar mensagem

Endpoint:

```text
POST /batepapo/apagar
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- `mensagem_id`: ID da mensagem que deve ser apagada

Campos opcionais:

- `grupo`: use `true` quando `numero` for ID numerico de grupo
- `remetente_jid` ou `participante`: necessario para apagar mensagem de outra pessoa em grupo quando a instancia for admin

JSON:

```json
{
  "numero": "11999999999",
  "mensagem_id": "3EB0D174A6979E7104AD8F"
}
```

### Reagir mensagem

Endpoint:

```text
POST /batepapo/reagir
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- `mensagem_id`: ID da mensagem que recebera a reacao
- `emoji`: emoji da reacao. Envie vazio para remover a reacao.

Campos importantes:

- `remetente_jid`: JID de quem enviou a mensagem original. Recomendado sempre e obrigatorio para grupos.
- `grupo`: `true` quando o destino for grupo usando ID numerico.

JSON para reagir uma mensagem recebida por webhook:

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem_id": "{{ $json.body.dados.mensagem.id }}",
  "remetente_jid": "{{ $json.body.dados.remetente_jid }}",
  "emoji": "👍"
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/reagir" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","mensagem_id":"3EB0D174A6979E7104AD8F","remetente_jid":"5511999999999@s.whatsapp.net","emoji":"👍"}'
```

Remover reacao:

```json
{
  "numero": "11999999999",
  "mensagem_id": "3EB0D174A6979E7104AD8F",
  "remetente_jid": "5511999999999@s.whatsapp.net",
  "emoji": ""
}
```

### Imagem

Endpoint:

```text
POST /batepapo/enviar/imagem
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- um destes campos: `arquivo_url`, `arquivo_base64` ou `caminho_local`

Campos opcionais:

- `legenda`
- `nome_arquivo`
- `grupo`

JSON com URL:

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/imagem.jpg",
  "legenda": "Imagem enviada pela API"
}
```

JSON com base64:

```json
{
  "numero": "11999999999",
  "arquivo_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg...",
  "legenda": "Imagem enviada em base64"
}
```

cURL para importar no n8n:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/imagem" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","arquivo_url":"https://exemplo.com/imagem.jpg","legenda":"Imagem enviada pela API"}'
```

### Figurinha

Endpoint:

```text
POST /batepapo/enviar/figurinha
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- um destes campos: `arquivo_url`, `arquivo_base64` ou `caminho_local`

Regras:

- A figurinha precisa estar em `image/webp`.
- Se usar base64, prefira data URI: `data:image/webp;base64,...`
- Conversao de PNG/JPG para WebP ainda nao e feita automaticamente pela API.

JSON para n8n:

```json
{
  "numero": "11999999999",
  "arquivo_base64": "data:image/webp;base64,SEU_BASE64"
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/figurinha" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","arquivo_url":"https://exemplo.com/figurinha.webp"}'
```

### Audio

Endpoint:

```text
POST /batepapo/enviar/audio
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- um destes campos: `arquivo_url`, `arquivo_base64` ou `caminho_local`

Campos opcionais:

- `ptt`: `true` para enviar como audio gravado/voz
- `duracao_segundos`: duracao exibida no WhatsApp
- `mime_type`
- `nome_arquivo`
- `grupo`

Regras praticas:

- para audio gravado, use `ogg/opus` e `ptt: true`
- `mp3`, `m4a`, `aac` e `wav` devem ser enviados como audio comum
- para base64 sem prefixo `data:...`, envie tambem `mime_type` ou `nome_arquivo`
- se `duracao_segundos` for omitido em `ogg/opus`, a API tenta calcular automaticamente

JSON com audio gravado base64:

```json
{
  "numero": "11999999999",
  "arquivo_base64": "data:audio/ogg;base64,T2dnUwACAAAAAAAAA...",
  "ptt": true
}
```

JSON com mp3 comum:

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/audio.mp3",
  "mime_type": "audio/mpeg",
  "nome_arquivo": "audio.mp3"
}
```

cURL para importar no n8n:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/audio" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","arquivo_url":"https://exemplo.com/audio.ogg","ptt":true}'
```

### Documento

Endpoint:

```text
POST /batepapo/enviar/documento
```

Campos obrigatorios:

- `numero` ou `chat_jid`
- um destes campos: `arquivo_url`, `arquivo_base64` ou `caminho_local`

Campos opcionais:

- `nome_arquivo`
- `legenda`
- `mime_type`
- `grupo`

JSON com URL:

```json
{
  "numero": "11999999999",
  "arquivo_url": "https://exemplo.com/contrato.pdf",
  "nome_arquivo": "contrato.pdf",
  "legenda": "Segue o documento"
}
```

JSON com base64:

```json
{
  "numero": "11999999999",
  "arquivo_base64": "data:application/pdf;base64,JVBERi0xLjQKJcTl8uXr...",
  "nome_arquivo": "contrato.pdf",
  "legenda": "Segue o documento"
}
```

cURL para importar no n8n:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/documento" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","arquivo_url":"https://exemplo.com/contrato.pdf","nome_arquivo":"contrato.pdf","legenda":"Segue o documento"}'
```

### Chamadas de voz

Endpoints iniciais para chamada WhatsApp 1:1 de audio.

Importante:

- chamadas usam o mesmo token da instancia via `X-Access-Token`
- com token de instancia, `instancia` pode ser omitido
- chamada real com audio exige negociacao WebRTC pelo endpoint `/webrtc`
- grupo e video ainda nao estao homologados nesta primeira versao
- o WhatsApp pode alterar regras internas de chamada; trate como recurso experimental ate homologacao completa

Iniciar chamada:

```text
POST /chamadas/iniciar
```

JSON:

```json
{
  "numero": "11999999999"
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/chamadas/iniciar" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999"}'
```

PowerShell:

```powershell
curl.exe -X POST "http://localhost:8080/api/v1/chamadas/iniciar" `
  -H "Content-Type: application/json" `
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" `
  -d '{"numero":"11999999999"}'
```

Se preferir usar aspas duplas no PowerShell, escape com crase:

```powershell
curl.exe -X POST "http://localhost:8080/api/v1/chamadas/iniciar" `
  -H "Content-Type: application/json" `
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" `
  -d "{`"numero`":`"11999999999`"}"
```

Resposta:

```json
{
  "sucesso": true,
  "mensagem": "Chamada iniciada com sucesso",
  "dados": {
    "instancia": "ID_DA_INSTANCIA",
    "chamada_id": "CALL_ID",
    "peer_jid": "5511999999999@s.whatsapp.net",
    "numero": "5511999999999",
    "direcao": "outgoing",
    "estado": "ringing",
    "tipo": "audio"
  }
}
```

Negociar audio WebRTC:

```text
POST /chamadas/:chamada_id/webrtc
```

JSON:

```json
{
  "sdp_offer": "v=0..."
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/chamadas/CALL_ID/webrtc" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"sdp_offer":"v=0..."}'
```

Aceitar chamada recebida:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/chamadas/CALL_ID/aceitar" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Rejeitar chamada:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/chamadas/CALL_ID/rejeitar" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Encerrar chamada:

```bash
curl -X DELETE "https://SEU_DOMINIO/api/v1/chamadas/CALL_ID" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Listar chamadas ativas da instancia:

```bash
curl -X GET "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/chamadas" \
  -H "X-Access-Token: TOKEN_MASTER_OU_DA_INSTANCIA"
```

Webhook:

- marque o evento `chamadas` no webhook da instancia
- eventos enviados: `recebida`, `estado`, `encerrada`
- payload traz `chamada_id`, `peer_jid`, `direcao`, `estado`, `tipo` e `criada_em`
- o campo `api` traz os caminhos prontos para `aceitar`, `rejeitar`, `encerrar` e `webrtc`
- chamadas recebidas devem ser atendidas pela aplicacao chamando `api.aceitar`
- para audio real, a aplicacao precisa criar uma conexao WebRTC e enviar `sdp_offer` para `api.webrtc`
- modo recomendado: `MediaStreamTrack` padrao do navegador
- audio do CRM/browser para a API: track Opus negociada pelo WebRTC
- audio da API para o CRM/browser: track PCMU negociada pelo WebRTC
- fallback legado: DataChannel WebRTC chamado `pcm`, com PCM mono 16 kHz, Int16 little-endian, nos dois sentidos

Exemplo de payload de webhook de chamada:

```json
{
  "evento": "chamadas",
  "instancia_id": "ID_DA_INSTANCIA",
  "dados": {
    "acao": "recebida",
    "chamada_id": "CALL_ID",
    "peer_jid": "5511999999999@s.whatsapp.net",
    "numero": "5511999999999",
    "direcao": "incoming",
    "estado": "incoming_ringing",
    "tipo": "audio",
    "api": {
      "aceitar": "/api/v1/chamadas/CALL_ID/aceitar",
      "rejeitar": "/api/v1/chamadas/CALL_ID/rejeitar",
      "encerrar": "/api/v1/chamadas/CALL_ID",
      "webrtc": "/api/v1/chamadas/CALL_ID/webrtc"
    }
  }
}
```

Aceitar chamada recebida pelo n8n:

```bash
curl -X POST "https://SEU_DOMINIO{{ $json.body.dados.api.aceitar }}" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Rotas equivalentes com a instancia na URL:

Toda acao de chamada tambem existe no prefixo `/instancias/{id}`. Use essa forma quando o
chamador usa token master ou controla varias instancias.

| Metodo | Rota curta (token da instancia) | Rota com instancia na URL |
| --- | --- | --- |
| POST | `/chamadas/{chamadaId}/aceitar` | `/instancias/{id}/chamadas/{chamadaId}/aceitar` |
| POST | `/chamadas/{chamadaId}/rejeitar` | `/instancias/{id}/chamadas/{chamadaId}/rejeitar` |
| DELETE | `/chamadas/{chamadaId}` | `/instancias/{id}/chamadas/{chamadaId}` |
| POST | `/chamadas/{chamadaId}/webrtc` | `/instancias/{id}/chamadas/{chamadaId}/webrtc` |
| GET | (nao existe) | `/instancias/{id}/chamadas` |

Exemplo:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/chamadas/CALL_ID/aceitar" -H "X-Access-Token: TOKEN_MASTER_OU_DA_INSTANCIA"
```

Observacoes:

- `iniciar` nao tem variante com instancia na URL. Use `POST /chamadas/iniciar` e mande
  `instancia` no corpo quando o token nao identificar a instancia.
- `aceitar`, `rejeitar` e `encerrar` aceitam corpo vazio. Quando enviado, o corpo pode
  trazer `instancia`, `chamada_id` (ou `id`) e `motivo`.
- `chamada_id` e `id` sao intercambiaveis no corpo; os textos `undefined` e `null` sao
  ignorados e a API cai para o `chamadaId` da URL.

Estados possiveis em `estado`:

```text
initiating
ringing
incoming_ringing
connecting
active
on_hold
ended
```

Erros comuns em chamadas:

- `instancia nao conectada`: a instancia precisa estar conectada e logada
- `chamada nao encontrada: id=... chamadas_ativas=...`: o `chamada_id` ja encerrou ou
  nunca existiu; o proprio erro lista os ids ativos
- `informe chamada_id`: a acao foi chamada sem id na URL nem no corpo
- `informe sdp_offer`: o endpoint `/webrtc` exige o campo `sdp_offer`

### Marcar mensagem como lida

Endpoint correto:

```text
POST /batepapo/marcar-lida
```

JSON para n8n:

```json
{
  "numero": "{{ $json.body.dados.chat_numero }}",
  "mensagem_id": "{{ $json.body.dados.mensagem.id }}",
  "participante": "{{ $json.body.dados.remetente_jid }}"
}
```

cURL:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/marcar-lida" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"11999999999","mensagem_id":"3EB06F9067F80BAB89FF","participante":"225400000000000:54@lid"}'
```

### Regra para numero, grupo e chat_jid

- conversa privada: envie `numero`
- grupo: envie `numero` com o ID do grupo e `grupo: true`
- se o webhook trouxe `chat_jid`, voce pode usar `chat_jid`, mas para grupo ainda envie `grupo: true`
- numero pode vir com ou sem codigo do Brasil; a API normaliza quando possivel

### Erros comuns no n8n

- erro `Campos obrigatorios`: o body chegou sem `mensagem`, `numero` ou arquivo
- erro `token invalido`: faltou header `X-Access-Token` ou o token esta errado
- erro de JSON invalido: o campo `JSON Body` tem expressao mal fechada
- nao use `Form Data` para estes endpoints; use JSON
- nao coloque `token` no body; use header

## Padrao de resposta

Sucesso:

```json
{
  "sucesso": true,
  "mensagem": "Operacao realizada com sucesso",
  "dados": {}
}
```

Erro:

```json
{
  "sucesso": false,
  "erro": "codigo_do_erro",
  "mensagem": "Descricao do erro"
}
```

## Saude

### `GET /api/v1/saude`

Retorna status da API, versao da aplicacao, versao do `whatsmeow` em uso e estado resumido de atualizacao.

## Sistema

### `GET /api/v1/sistema/versao`

Retorna:

- versao da aplicacao
- commit e data de build
- versao do `whatsmeow` em uso
- ultima verificacao
- status da atualizacao
- indicacao de rebuild/reinicio obrigatorios

### `GET /api/v1/sistema/atualizacoes`

Retorna o estado persistido da dependencia monitorada.

### `GET /api/v1/sistema/proxy`

Consulta a configuracao de proxy global do master.

### `PUT /api/v1/sistema/proxy`

Define o proxy global usado automaticamente pelas instancias que nao tenham proxy proprio configurado.

```json
{
  "url": "http://usuario:senha@host:porta",
  "ativo": true
}
```

Observacoes:

- esquemas aceitos: `http`, `https` e `socks5`
- se `ativo=false`, a URL e ignorada e o proxy global fica desativado
- mudar proxy em Go/whatsmeow exige reconexao do cliente; a API recarrega as instancias sem proxy proprio quando possivel

### `POST /api/v1/sistema/atualizacoes/verificar`

Forca uma verificacao imediata da ultima versao publicada no proxy Go configurado.

### `POST /api/v1/sistema/atualizacoes/aplicar`

Exige header `X-Update-Token` e so funciona quando:

- `UPDATE_MODE=preparo`
- `UPDATE_APPLY_ENABLED=true`
- ambiente diferente de `production`

Fluxo manual recomendado para atualizar o `whatsmeow`:

```bash
go get go.mau.fi/whatsmeow@latest
go mod tidy
go build ./cmd/api
go run ./cmd/api
```

## Instancias

### `POST /api/v1/instancias`

Campos:

- obrigatorio: `nome`

```json
{
  "nome": "Atendimento Principal"
}
```

### `GET /api/v1/instancias`

Lista instancias cadastradas.

### `GET /api/v1/instancias/:id`

Busca uma instancia por identificador.

### `DELETE /api/v1/instancias/:id`

Exclui a instancia e remove a sessao local.

### `POST /api/v1/instancias/:id/conectar`

Inicia o fluxo de conexao e retorna QR code real quando necessario.

### `POST /api/v1/instancias/:id/pairing-code`

Gera um Pairing Code para novo pareamento por numero.

```json
{
  "numero": "5511999999999"
}
```

Observacoes:

- informe o numero completo em formato internacional, somente com digitos
- o Pairing Code so funciona quando a instancia nao possui sessao ativa
- se a instancia ja estiver conectada ou com sessao salva, desconecte primeiro
- a rota retorna `codigo`, `numero`, `status` e `metodo_pareamento`

### `POST /api/v1/instancias/:id/desconectar`

Faz logout real da sessao. No proximo conectar, a instancia volta a exigir QR code.

### `GET /api/v1/instancias/:id/status`

Consulta status consolidado da instancia. Inclui `pairing_code`, `pairing_phone`, `pairing_code_pronto`, `metodo_pareamento`, `historico_dias`, `historico_max_dias`, `historico_configurado`, `historico_bloqueado`, `historico_observacao`, `proxy_modo`, `proxy_url`, `proxy_configurado` e `proxy_observacao`.

### `GET /api/v1/instancias/:id/contatos/avatar`

Consulta a foto/avatar de um contato ou grupo usando a instancia informada.

Query params:

- `numero`: numero do contato. Ex.: `5511999999999`
- `chat_jid`: JID direto do chat, quando disponivel. Ex.: `5511999999999@s.whatsapp.net`
- `grupo`: use `true` para consultar avatar de grupo pelo ID do grupo
- `formato`: `json`, `base64` ou `arquivo`. Padrao: `json`

Resposta padrao (`formato=json`):

```json
{
  "sucesso": true,
  "mensagem": "Avatar consultado com sucesso",
  "dados": {
    "instancia": "ID_DA_INSTANCIA",
    "numero": "5511999999999",
    "chat_jid": "5511999999999@s.whatsapp.net",
    "grupo": false,
    "tem_avatar": true,
    "avatar_id": "abc123",
    "avatar_url": "https://mmg.whatsapp.net/...",
    "tipo": "image"
  }
}
```

Exemplo `curl` com URL:

```bash
curl -X GET "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/contatos/avatar?numero=5511999999999" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Exemplo `curl` retornando base64:

```bash
curl -X GET "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/contatos/avatar?numero=5511999999999&formato=base64" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

No formato `base64`, a resposta inclui:

```json
{
  "avatar_base64": "...",
  "avatar_data_uri": "data:image/jpeg;base64,...",
  "mime_type": "image/jpeg",
  "tamanho_bytes": 12345
}
```

No formato `arquivo`, a API redireciona para a URL temporaria da foto no WhatsApp:

```bash
curl -L -X GET "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/contatos/avatar?numero=5511999999999&formato=arquivo" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  --output avatar.jpg
```

### `PUT /api/v1/instancias/:id/historico`

Define quantos dias de historico inicial a API deve importar quando o WhatsApp entregar `HistorySync`. Precisa ser configurado antes de conectar. Use `0` para desativar.

```json
{
  "dias": 10
}
```

Observacoes:

- o limite operacional vem de `HISTORY_MAX_DAYS` e tambem aparece em `historico_max_dias`
- o WhatsApp/whatsmeow nao expoe um maximo garantido em dias nesse fluxo
- mensagens de historico enviadas por webhook recebem `historico=true` e `origem="historico"`

### `PUT /api/v1/instancias/:id/proxy`

Configura o proxy proprio da instancia. Quando a instancia nao tiver proxy proprio, ela usa automaticamente o proxy global do master se ele estiver ativo.

Exemplo proxy proprio:

```json
{
  "modo": "proprio",
  "url": "http://usuario:senha@host:porta"
}
```

Exemplo remover proxy proprio e voltar ao comportamento padrao:

```json
{
  "modo": "herdar"
}
```

Esquemas aceitos: `http`, `https` e `socks5`.

### `PUT /api/v1/instancias/:id/avancado`

Atualiza as configuracoes avancadas persistentes da instancia.

```json
{
  "manter_online": false,
  "rejeitar_chamadas": true,
  "mensagem_rejeitar_chamadas": "No momento nao consigo atender chamadas. Envie uma mensagem por aqui.",
  "marcar_lida_automatico": false,
  "ignorar_grupos": false,
  "ignorar_status": false
}
```

Campos:

- `manter_online`: quando `true`, a API tenta manter a presenca global como disponivel; quando `false`, mantem indisponivel. Padrao: `false`
- `rejeitar_chamadas`: quando `true`, chamadas recebidas sao rejeitadas automaticamente pelo WhatsApp. Padrao: `false`
- `mensagem_rejeitar_chamadas`: opcional; se `rejeitar_chamadas=true`, envia uma mensagem de texto apos rejeitar a chamada
- `marcar_lida_automatico`: quando `true`, mensagens recebidas sao marcadas como lidas automaticamente. Padrao: `false`
- `ignorar_grupos`: quando `true`, mensagens de grupos nao sao enviadas para webhooks. Padrao: `false`
- `ignorar_status`: quando `true`, mensagens de status do WhatsApp (`status@broadcast`) nao sao enviadas para webhooks. Padrao: `false`

Observacoes:

- o endpoint `POST /api/v1/batepapo/marcar-lida` continua disponivel mesmo quando `marcar_lida_automatico=false`
- a configuracao e persistida no banco e reaplicada quando a instancia reconectar
- chamadas de rejeicao dependem do evento entregue pelo WhatsApp/whatsmeow

### `GET /api/v1/instancias/:id/qrcode`

Consulta o QR code disponivel para pareamento. Quando o fluxo ativo for Pairing Code, a resposta tambem inclui `pairing_code` e `pairing_phone`.

### `GET /api/v1/instancias/:id/midia/:midiaId`

Baixa uma midia recebida anteriormente pela instancia. Exige token master ou token da propria instancia.

## Webhooks por instancia

### `GET /api/v1/instancias/:id/webhooks`

Lista os webhooks cadastrados para a instancia.

### `POST /api/v1/instancias/:id/webhooks`

Campos:

- obrigatorios: `nome`, `url`, `eventos`
- opcional: `ativo`

```json
{
  "nome": "Teste n8n",
  "url": "https://seu-n8n/webhook/teste",
  "eventos": ["mensagens", "recibos", "status", "digitando", "gravando_audio"],
  "ativo": true
}
```

### `PUT /api/v1/instancias/:id/webhooks/:webhookId`

Mesmo payload do `POST`, atualizando um webhook existente.

### `DELETE /api/v1/instancias/:id/webhooks/:webhookId`

Exclui o webhook.

### `GET /api/v1/instancias/:id/webhook-entregas`

Lista as entregas recentes da fila de webhooks da instancia.

Query params:

- `limite`: opcional, padrao `50`, maximo `200`

Campos principais:

- `status`: `pendente`, `enviando`, `entregue`, `falha` ou `esgotada`
- `tentativas`
- `max_tentativas`
- `status_http`
- `ultimo_erro`
- `proxima_tentativa_em`
- `ultima_tentativa_em`

Exemplo:

```bash
curl -X GET "https://SEU_DOMINIO/api/v1/instancias/ID_DA_INSTANCIA/webhook-entregas?limite=60" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA"
```

Observacoes:

- as entregas sao persistidas no banco antes do envio HTTP
- falhas temporarias entram em retry exponencial
- ao exceder o limite de tentativas, a entrega fica como `esgotada`
- a dashboard mostra esse mesmo historico na aba Auditoria

## Formato dos webhooks validados

### `mensagens`

Campos planos mantidos:

- `chat_jid`
- `chat_numero`
- `grupo`
- `mensagem_id`
- `acao`: `recebida`, `editada` ou `apagada`
- `direcao`
- `enviado_por_mim`
- `remetente_jid`
- `remetente_numero`
- `nome_remetente`
- `conteudo`
- `tipo`
- `botao_id` quando `tipo=botao`
- `botao_texto` quando `tipo=botao`
- `botao_tipo` quando `tipo=botao`
- `resposta_mensagem_id` quando a mensagem recebida cita/responde outra mensagem
- `resposta_participante` quando a mensagem recebida cita/responde outra mensagem
- `resposta_conteudo` quando a mensagem recebida cita/responde outra mensagem
- `resposta_tipo` quando a mensagem recebida cita/responde outra mensagem

Campos organizados adicionais:

- `conversa.jid`
- `conversa.numero`
- `conversa.grupo`
- `autor.jid`
- `autor.numero`
- `autor.nome`
- `mensagem.id`
- `mensagem.tipo`
- `mensagem.acao`
- `mensagem.conteudo`
- `mensagem.direcao`
- `mensagem.enviado_por_mim`
- `mensagem.duracao_segundos` quando `tipo=audio`
- `mensagem.ptt` quando `tipo=audio`
- `mensagem.botao.id`, `mensagem.botao.texto`, `mensagem.botao.tipo` quando `tipo=botao`
- `mensagem.resposta.mensagem_id`, `mensagem.resposta.participante`, `mensagem.resposta.conteudo` e `mensagem.resposta.tipo` quando a mensagem recebida cita/responde outra mensagem (o WhatsApp so envia `conteudo`/`tipo` da mensagem original quando o app do remetente os embute na citacao; se vier vazio, use `resposta_mensagem_id` para buscar a mensagem original no seu proprio historico)
- `mensagem.mime_type` quando houver midia
- `mensagem.midia.id`, `mensagem.midia.mensagem_id`, `mensagem.midia.tipo`, `mensagem.midia.mime_type`, `mensagem.midia.nome_arquivo`, `mensagem.midia.tamanho_bytes`, `mensagem.midia.sha256`, `mensagem.midia.download_path` e `mensagem.midia.download_url` quando a midia recebida for baixada com sucesso
- `mensagem.midia.storage_provider`, `mensagem.midia.storage_path` e `mensagem.midia.storage_url` quando storage externo estiver configurado

Em edicoes, o webhook continua chegando no evento `mensagens`, mas com `dados.acao="editada"` e `dados.conteudo` contendo o texto novo. Em apagamentos, chega com `dados.acao="apagada"`, `dados.tipo="apagada"` e `dados.mensagem_id` apontando para a mensagem apagada.

Campos de compatibilidade para midia recebida:

- `midia_id`
- `midia_download_path`
- `midia_download_url` quando `API_BASE_URL` estiver configurada
- `midia.mensagem_id`
- `midia.base64` com o conteudo bruto em base64
- `midia.data_uri` no formato `data:<mime>;base64,<conteudo>`
- `midia_storage_provider`, `midia_storage_path` e `midia_storage_url` quando storage externo estiver configurado
- `tamanho_bytes`
- `mime_type`
- `nome_arquivo`

### `digitando` e `gravando_audio`

Campos planos mantidos:

- `chat_jid`
- `chat_numero`
- `grupo`
- `remetente_jid`
- `remetente_numero`
- `estado`
- `estado_texto`
- `media`
- `acao_presenca`

Campos organizados adicionais:

- `conversa.jid`
- `conversa.numero`
- `conversa.grupo`
- `autor.jid`
- `autor.numero`
- `presenca.acao`
- `presenca.estado`
- `presenca.estado_texto`
- `presenca.media`

Observacao:

- em eventos com `@lid`, o numero real pode nao vir no primeiro `presence`
- depois do primeiro mapeamento conhecido por mensagem, os proximos `presence` passam a sair com `chat_numero` e `remetente_numero` corretos

### `recibos`

Evento disparado por recibos do WhatsApp para mensagens enviadas/visualizadas/reproduzidas. Use para saber quando uma mensagem foi entregue, lida ou ouvida.

Valores principais de `dados.status`:

- `entregue`: mensagem entregue ao dispositivo do contato
- `lida`: contato abriu a conversa e leu a mensagem
- `ouvida`: audio ou midia reproduzida/aberta quando o WhatsApp envia esse recibo
- `lida_por_mim`: leitura feita pelo proprio usuario em outro dispositivo
- `ouvida_por_mim`: reproducao feita pelo proprio usuario em outro dispositivo
- `retry`: mensagem chegou, mas houve falha de descriptografia no destino
- `sincronizada`: mensagem sincronizada/entregue a outro dispositivo proprio

Campos planos:

- `mensagem_id`: primeiro ID recebido no recibo
- `mensagens_id`: lista de IDs do recibo
- `status`
- `tipo_recibo`: valor bruto do WhatsApp, como `delivered`, `read`, `played`, `retry`
- `chat_jid`
- `chat_numero`
- `grupo`
- `direcao`
- `enviado_por_mim`
- `remetente_jid`
- `remetente_numero`
- `participante_jid`: usado principalmente em grupos
- `participante_numero`
- `ocorrido_em`

Exemplo:

```json
{
  "evento": "recibos",
  "instancia_id": "eeee80e7-9d9d-487e-bffe-509eeec7729e",
  "ocorrido_em": "2026-06-26T12:00:00Z",
  "dados": {
    "mensagem_id": "3EB06F9067F80BAB89FF",
    "mensagens_id": ["3EB06F9067F80BAB89FF"],
    "status": "lida",
    "tipo_recibo": "read",
    "chat_jid": "5511999999999@s.whatsapp.net",
    "chat_numero": "5511999999999",
    "grupo": false,
    "enviado_por_mim": false,
    "direcao": "entrada",
    "ocorrido_em": "2026-06-26T12:00:00Z"
  }
}
```

## Bate-papo

### `POST /api/v1/batepapo/enviar/texto`

Campos:

- obrigatorio: `mensagem`
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `mensagem_id`
- opcional: `resposta_mensagem_id`
- opcional: `resposta_participante`
- opcional: `resposta_conteudo`
- opcional: `delay` ou `delay_segundos` em segundos
- opcional: `delay_ms` em milissegundos
- opcional: `digitando`

Quando `delay`, `delay_segundos` ou `delay_ms` for maior que zero, a API envia presenca de `digitando`, aguarda o tempo informado e depois envia a mensagem. O delay e limitado a 60 segundos por seguranca operacional.

### Envio privado simples

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "mensagem": "Ola"
}
```

### Envio privado com digitando antes da mensagem

```json
{
  "numero": "5511999999999",
  "mensagem": "Ola, tudo bem?",
  "delay": 3
}
```

### Envio para grupo

Use o ID do grupo em `numero` e envie `grupo: true`.

```json
{
  "instancia": "id-da-instancia",
  "numero": "120363000000000000",
  "grupo": true,
  "mensagem": "Ola grupo"
}
```

### Resposta citada em privado

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "ID_DA_MSG_ORIGINAL",
  "resposta_participante": "5511999999999@lid",
  "resposta_conteudo": "texto original"
}
```

Formato legado compatível com WUZAPI:

```json
{
  "Phone": "5511988887777",
  "Body": "How you doin",
  "Id": "ABCDABCD1234",
  "ContextInfo": {
    "StanzaId": "3EB06F9067F80BAB89FF",
    "Participant": "5511977776666@s.whatsapp.net"
  }
}
```

Mapeamento:

- `Phone` equivale a `numero`
- `Body` equivale a `mensagem`
- `Id` equivale a `mensagem_id`
- `ContextInfo.StanzaId` equivale a `resposta_mensagem_id`
- `ContextInfo.Participant` equivale a `resposta_participante`

### Resposta citada em grupo

```json
{
  "instancia": "id-da-instancia",
  "numero": "120363000000000000",
  "grupo": true,
  "mensagem": "Resposta pela API",
  "resposta_mensagem_id": "ID_DA_MSG_ORIGINAL",
  "resposta_participante": "225400000000000:54@lid",
  "resposta_conteudo": "texto original"
}
```

### Regra pratica de envio

- privado: use `numero`
- grupo: use `numero` + `grupo: true`
- reply citado: use `resposta_mensagem_id` + `resposta_participante`
- para o n8n, prefira `Using JSON` em vez de `Using Fields Below`

### `POST /api/v1/batepapo/enviar/presenca`

### `POST /api/v1/user/presence`

Envia estado de presença para um chat, como "digitando" ou "gravando audio".

Campos:

- obrigatorio: `acao` ou `type`
- obrigatorio para presenca de chat: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `delay` ou `delay_segundos` em segundos
- opcional: `delay_ms` em milissegundos

Acoes suportadas:

- `digitando`
- `gravando_audio`
- `pausado`
- `disponivel`
- `indisponivel`
- compatibilidade: `type` com `available` ou `unavailable`

Observacoes:

- `digitando` envia `composing` com media texto
- `gravando_audio` envia `composing` com media audio
- `pausado` remove o estado de digitacao/gravacao do chat
- `disponivel` atualiza a presenca global da instancia como online
- `indisponivel` atualiza a presenca global da instancia como offline/nao disponivel
- presenca global (`available`/`unavailable`) nao usa `numero` nem `chat_jid`
- quando usar token de instancia, `instancia` pode ser omitida
- quando `acao` for `digitando` ou `gravando_audio` e houver delay, a API aguarda o tempo informado e depois envia `pausado`
- o delay e limitado a 60 segundos

Exemplo para privado:

```json
{
  "numero": "5511999999999",
  "acao": "digitando",
  "delay": 3
}
```

Exemplo para grupo:

```json
{
  "numero": "120363000000000000",
  "grupo": true,
  "acao": "gravando_audio"
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/presenca" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","acao":"digitando"}'
```

Exemplo curl para deixar a instancia indisponivel:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/presenca" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"acao":"indisponivel"}'
```

Exemplo compatibilidade estilo WuzAPI:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/user/presence" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"type":"unavailable"}'
```

### `POST /api/v1/batepapo/marcar-lida`

Marca uma ou mais mensagens como lidas no WhatsApp.

Campos:

- obrigatorio: `mensagem_id` ou `mensagens_id`
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `participante` ou `remetente_jid`
- opcional: `lida_em`
- compatibilidade: `Phone`, `Id`, `Ids`, `Participant`, `Timestamp`

Regras:

- em conversa privada, basta `numero` ou `chat_jid` junto com `mensagem_id`
- em grupo, alem da conversa, informe tambem `participante` ou `remetente_jid`
- `lida_em` e opcional e deve estar em RFC3339; se omitido, a API usa o horario atual
- `mensagens_id` permite marcar varias mensagens de uma vez, desde que sejam do mesmo remetente
- quando usar token de instancia, `instancia` pode ser omitida

Exemplo privado:

```json
{
  "numero": "5511999999999",
  "mensagem_id": "3EB06F9067F80BAB89FF"
}
```

Exemplo grupo:

```json
{
  "chat_jid": "120363000000000000@g.us",
  "grupo": true,
  "mensagem_id": "3EB06F9067F80BAB89FF",
  "participante": "225400000000000:54@lid"
}
```

Exemplo compatibilidade:

```json
{
  "Phone": "5511999999999",
  "Id": "3EB06F9067F80BAB89FF"
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/marcar-lida" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","mensagem_id":"3EB06F9067F80BAB89FF"}'
```

Nao existe alias ativo para esta rota. Use sempre `/api/v1/batepapo/marcar-lida`.

### `POST /api/v1/batepapo/enviar/botoes`

Envia mensagem com botoes rapidos.

Campos:

- obrigatorio: `texto` ou `mensagem`
- obrigatorio: `botoes`, com 1 a 3 itens
- obrigatorio em cada botao: `id` e `texto`
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `titulo`
- opcional: `rodape`
- opcional: `modo`
- compatibilidade WUZAPI: `Phone`, `Content`, `Footer`, `Buttons`, `Id`, `ContextInfo`

Modos:

- `native_flow`: padrao, formato moderno usando `InteractiveMessage` direto
- `native_flow_view_once`: formato moderno envelopado em `ViewOnceMessage`, mantido para teste comparativo
- `template`: formato `TemplateMessage/HydratedFourRowTemplate`, usado automaticamente quando o payload vier no estilo WUZAPI `Phone/Content/Buttons`
- `texto`: fallback operacional que envia as opcoes como menu textual
- `buttons` ou `legacy`: formato legado, mantido apenas para diagnostico; pode retornar erro `405`

Observacoes:

- `texto` do botao aceita ate 20 caracteres
- `id` do botao aceita ate 256 caracteres
- no estilo WUZAPI, `DisplayText` vira `texto`, `Type` vira `tipo`, e `quickreply` sem `id` usa o proprio texto como ID
- botoes `url` precisam de `Url` ou `URL`; botoes `call` precisam de `PhoneNumber`
- quando usar token de instancia, `instancia` pode ser omitida
- se `modo` for omitido, a API usa `native_flow`
- se o payload usar campos WUZAPI (`Phone`, `Content` ou `Buttons`) e `modo` for omitido, a API usa `template`
- se o servidor do WhatsApp responder `405`, a API faz fallback automatico para texto e retorna isso no campo `observacao`
- o WhatsApp pode aceitar a mensagem e ainda assim nao renderizar botoes em alguns clientes/contas comuns; nesse caso teste `native_flow_view_once`
- se precisar garantir entrega do conteudo, envie `modo: "texto"` ou `fallback_texto: true`
- quando o usuario clicar, o webhook chega como `tipo="botao"` com `botao_id`, `botao_texto` e `mensagem.botao`

Exemplo:

```json
{
  "numero": "5511999999999",
  "mensagem": "Escolha uma opcao",
  "rodape": "Dyalog",
  "botoes": [
    { "id": "confirmar", "texto": "Confirmar" },
    { "id": "cancelar", "texto": "Cancelar" }
  ]
}
```

Exemplo compatibilidade WUZAPI template:

```json
{
  "Phone": "5511999999999",
  "Content": "Escolha uma opcao",
  "Footer": "Dyalog",
  "Buttons": [
    { "DisplayText": "Sim", "Type": "quickreply" },
    { "DisplayText": "Nao", "Type": "quickreply" },
    { "DisplayText": "Site", "Type": "url", "Url": "https://SEU_DOMINIO" }
  ]
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/botoes" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","mensagem":"Escolha uma opcao","botoes":[{"id":"sim","texto":"Sim"},{"id":"nao","texto":"Nao"}]}'
```

Exemplo curl com payload de compatibilidade WUZAPI:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/botoes" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"Phone":"5511999999999","Content":"Escolha uma opcao","Footer":"Dyalog","Buttons":[{"DisplayText":"Sim","Type":"quickreply"},{"DisplayText":"Nao","Type":"quickreply"}]}'
```

Fallback textual:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/botoes" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","mensagem":"Escolha uma opcao","modo":"texto","botoes":[{"id":"sim","texto":"Sim"},{"id":"nao","texto":"Nao"}]}'
```

### `POST /api/v1/batepapo/enviar/lista`

Envia mensagem com lista interativa de selecao unica.

> **Nao funciona mais como menu interativo.** O WhatsApp recusa os formatos de
> menu (`ack error 405`) quando o remetente e uma conta comum conectada via
> multi-device, que e o caso das instancias da Dyalog. So remetentes da API
> oficial do WhatsApp Business conseguem enviar. Isso foi verificado replicando
> byte a byte o stanza de uma empresa cujo menu renderiza normalmente - o
> servidor recusou do mesmo jeito, entao a restricao e no tipo de remetente, nao
> no formato da mensagem.
>
> Na pratica o endpoint continua util: ao ser recusado, ele envia as opcoes
> **como texto numerado** automaticamente, e a resposta traz `modo: "texto"` com
> uma `observacao` explicando. Para um menu que o cliente possa **clicar**, use:
>
> - [`enviar/enquete`](#post-apiv1batepapoenviarenquete) - ate 12 opcoes, e o
>   voto volta ja resolvido para o texto da opcao
> - [`enviar/botoes`](#post-apiv1batepapoenviarbotoes) - ate 3 opcoes

Campos:

- obrigatorio: `descricao` ou `mensagem`
- obrigatorio: `botao_texto`
- obrigatorio: `numero` ou `chat_jid`
- obrigatorio: `secoes`, `opcoes` ou `List`
- obrigatorio em cada linha: `id` e `titulo`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `titulo`
- opcional: `rodape`
- opcional: `modo`
- opcional: `fallback_texto`
- compatibilidade WUZAPI: `Phone`, `ButtonText`, `Desc`, `TopText`, `FooterText`, `List`, `Id`, `ContextInfo`

Modos:

- `lista`: padrao, envia `ListMessage` interativa
- `list`: alias do mesmo formato
- `texto`: envia menu textual em vez de lista interativa
- `fallback_texto`: alias operacional do mesmo fallback

Regras:

- a lista aceita de 1 a 10 linhas no total
- se usar `opcoes` ou `List`, a API cria uma secao unica automaticamente
- se usar `secoes`, cada secao pode ter titulo e varias linhas
- quando usar token de instancia, `instancia` pode ser omitida
- se `modo` for omitido, a API usa `lista`
- se o payload vier no estilo WUZAPI (`Phone`, `ButtonText`, `Desc` ou `List`) e `modo` for omitido, a API usa `lista`
- se o servidor do WhatsApp responder `405`, a API faz fallback automatico para texto e retorna isso no campo `observacao`
- o WhatsApp pode aceitar a mensagem e ainda assim nao renderizar a lista em algumas contas/clientes; nesse caso use `modo: "texto"` ou `fallback_texto: true`
- quando o usuario selecionar um item, o webhook chega como `tipo="lista"` com `lista_id`, `lista_titulo`, `lista_descricao` e `mensagem.lista`

Exemplo:

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

Exemplo com secoes:

```json
{
  "numero": "5511999999999",
  "titulo": "Atendimento",
  "descricao": "Escolha uma opcao",
  "botao_texto": "Abrir lista",
  "secoes": [
    {
      "titulo": "Setores",
      "linhas": [
        { "id": "financeiro", "titulo": "Financeiro", "descricao": "Boletos e pagamentos" },
        { "id": "suporte", "titulo": "Suporte", "descricao": "Ajuda tecnica" }
      ]
    }
  ]
}
```

Exemplo compatibilidade WUZAPI:

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

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/lista" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","titulo":"Atendimento","descricao":"Escolha uma opcao","botao_texto":"Abrir lista","opcoes":[{"id":"financeiro","titulo":"Financeiro","descricao":"Boletos e pagamentos"},{"id":"suporte","titulo":"Suporte","descricao":"Ajuda tecnica"}]}'
```

Exemplo curl com payload de compatibilidade WUZAPI:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/lista" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"Phone":"5511999999999","TopText":"Atendimento","Desc":"Escolha uma opcao","ButtonText":"Abrir lista","List":[{"RowId":"financeiro","title":"Financeiro","desc":"Boletos e pagamentos"},{"RowId":"suporte","title":"Suporte","desc":"Ajuda tecnica"}]}'
```

Fallback textual:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/lista" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","descricao":"Escolha uma opcao","botao_texto":"Abrir lista","modo":"texto","opcoes":[{"id":"financeiro","titulo":"Financeiro"},{"id":"suporte","titulo":"Suporte"}]}'
```

### `POST /api/v1/batepapo/enviar/enquete`

Envia uma enquete (poll) nativa do WhatsApp. Diferente de botoes e lista, a enquete e renderizada de forma consistente em todos os clientes/contas e nao costuma ser filtrada pelo WhatsApp, por isso e uma alternativa recomendada para oferecer opcoes selecionaveis quando `enviar/botoes` ou `enviar/lista` forem rejeitados ou nao renderizarem.

Campos:

- obrigatorio: `nome` ou `pergunta`
- obrigatorio: `opcoes` (lista de textos, de 2 a 12, sem duplicatas)
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `selecao_multipla` (permite escolher mais de uma opcao; padrao e selecao unica)
- opcional: `opcoes_selecionaveis` (quantidade maxima de opcoes que o destinatario pode marcar; se omitido, usa 1 para selecao unica ou todas quando `selecao_multipla` for `true`)
- opcional: `resposta_mensagem_id`, `resposta_participante` (responder a uma mensagem especifica)
- compatibilidade WUZAPI: `Phone`, `Name`, `Options`, `Id`, `ContextInfo`

Regras:

- quando usar token de instancia, `instancia` pode ser omitida
- a API guarda em memoria o texto de cada opcao para poder resolver o voto do destinatario; se a instancia reiniciar entre o envio da enquete e o voto, o webhook do voto volta apenas com os dados criptografados
- quando o destinatario vota, o webhook chega como `tipo="enquete_voto"` com o `conteudo` preenchido com o texto da(s) opcao(oes) escolhida(s), `extras.enquete_voto_opcoes` (lista de textos) e `mensagem.enquete_voto.opcoes_selecionadas`

Exemplo:

```json
{
  "numero": "5511999999999",
  "nome": "Qual plano voce prefere?",
  "opcoes": ["Basico", "Pro", "Enterprise"]
}
```

Exemplo com selecao multipla:

```json
{
  "numero": "5511999999999",
  "nome": "Quais recursos voce usa?",
  "opcoes": ["Chat", "Chamadas", "Automacoes"],
  "selecao_multipla": true
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/enquete" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","nome":"Qual plano voce prefere?","opcoes":["Basico","Pro","Enterprise"]}'
```

### `POST /api/v1/batepapo/enviar/cobranca-pix`

Envia o botao nativo de pagamento do WhatsApp ("Cobrar via Pix"), o mesmo recurso que o app oficial usa quando a conta tem Pagamentos habilitado pela Meta. **Isso e diferente de botao comum**: e um recurso de pagamento real controlado pela Meta, normalmente atrelado a contas com esse recurso ja habilitado (o mesmo numero precisa conseguir usar "Cobrar" no app pra essa API ter chance de funcionar). O formato do payload foi construido a partir de uma cobranca real capturada via webhook desta propria API.

Campos:

- obrigatorio: `chave_pix`
- obrigatorio: `tipo_chave` (`telefone`, `email`, `cpf`, `cnpj` ou `aleatoria`)
- obrigatorio: `nome_beneficiario`
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `valor` (em reais; se omitido ou 0, a cobranca e enviada "sem valor definido", igual ao exemplo real que validamos)
- opcional: `referencia` (gerada automaticamente se omitida)
- opcional: `descricao` (vira o texto/corpo da mensagem e o nome do item cobrado)
- opcional: `fallback_texto` (força enviar como texto simples com os dados da chave em vez do botao nativo)
- opcional: `resposta_mensagem_id`, `resposta_participante`
- compatibilidade WUZAPI: `Phone`, `Id`, `ContextInfo`

Regras:

- se `valor` for informado (maior que 0), a cobranca e criada com o valor especificado (`order_type: "ORDER"`); esse caminho ainda nao foi validado contra um exemplo real do app oficial, so o caminho "sem valor" foi confirmado byte-a-byte
- se o servidor do WhatsApp responder `405`, a API faz fallback automatico para texto (chave, beneficiario e valor) e retorna isso no campo `observacao`
- mesmo quando o WhatsApp aceita a mensagem, so a conta ter Pagamentos habilitado pela Meta garante que o botao realmente renderize e funcione — teste antes de depender disso em producao

Exemplo (sem valor definido):

```json
{
  "numero": "5511999999999",
  "chave_pix": "11999998888",
  "tipo_chave": "telefone",
  "nome_beneficiario": "Nome do Beneficiario"
}
```

Exemplo (com valor):

```json
{
  "numero": "5511999999999",
  "chave_pix": "financeiro@empresa.com",
  "tipo_chave": "email",
  "nome_beneficiario": "Empresa LTDA",
  "valor": 149.90,
  "descricao": "Mensalidade agosto"
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/cobranca-pix" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","chave_pix":"11999998888","tipo_chave":"telefone","nome_beneficiario":"Nome do Beneficiario"}'
```

### `POST /api/v1/batepapo/enviar/localizacao`

Envia uma localizacao estatica (pino no mapa).

Campos:

- obrigatorio: `latitude` (-90 a 90)
- obrigatorio: `longitude` (-180 a 180)
- obrigatorio: `numero` ou `chat_jid`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `nome` (nome do local)
- opcional: `endereco`
- opcional: `resposta_mensagem_id`, `resposta_participante`
- compatibilidade WUZAPI: `Phone`, `Latitude`, `Longitude`, `Name`, `Address`, `Id`, `ContextInfo`

Exemplo:

```json
{
  "numero": "5511999999999",
  "latitude": -23.55052,
  "longitude": -46.633308,
  "nome": "Av. Paulista",
  "endereco": "Sao Paulo - SP"
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/localizacao" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","latitude":-23.55052,"longitude":-46.633308,"nome":"Av. Paulista"}'
```

### `POST /api/v1/batepapo/enviar/contato`

Envia um cartao de contato (vCard). Pode enviar um unico contato ou uma lista.

Campos:

- obrigatorio: `numero` ou `chat_jid`
- obrigatorio (contato unico): `nome` e `telefone`, ou `vcard` pronto
- obrigatorio (varios contatos): `contatos` (lista de `{ "nome", "telefone", "organizacao", "vcard" }`, cada item precisa de `nome`+`telefone` ou `vcard`)
- opcional: `instancia`
- opcional: `grupo`
- opcional: `organizacao`
- opcional: `resposta_mensagem_id`, `resposta_participante`
- compatibilidade WUZAPI: `Phone`, `Name`, `Vcard`, `Contacts`, `Id`, `ContextInfo`

Regras:

- se `vcard` nao for informado, a API monta um vCard 3.0 basico a partir de `nome`/`telefone`/`organizacao`
- quando enviar mais de um contato, o webhook de recebimento no destinatario aparece como `tipo="contatos"`; um unico contato aparece como `tipo="contato"`

Exemplo (contato unico):

```json
{
  "numero": "5511999999999",
  "nome": "Suporte Dyalog",
  "telefone": "5511988887777"
}
```

Exemplo (varios contatos):

```json
{
  "numero": "5511999999999",
  "contatos": [
    { "nome": "Financeiro", "telefone": "5511988887777" },
    { "nome": "Suporte", "telefone": "5511977776666" }
  ]
}
```

Exemplo curl:

```bash
curl -X POST "https://SEU_DOMINIO/api/v1/batepapo/enviar/contato" \
  -H "Content-Type: application/json" \
  -H "X-Access-Token: TOKEN_DA_INSTANCIA" \
  -d '{"numero":"5511999999999","nome":"Suporte Dyalog","telefone":"5511988887777"}'
```

### `POST /api/v1/batepapo/enviar/imagem`

Campos:

- obrigatorio: `numero` ou `chat_jid`
- obrigatorio: `arquivo_url`, `arquivo_base64` ou `caminho_local`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `legenda`
- opcional: `nome_arquivo`

Exemplo com arquivo local:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "caminho_local": "D:/midias/foto.png",
  "legenda": "Imagem enviada pela API"
}
```

Exemplo com URL:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_url": "https://exemplo.com/imagem.jpg",
  "legenda": "Imagem enviada pela API"
}
```

Exemplo com base64:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_base64": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg...",
  "legenda": "Imagem enviada em base64"
}
```

### `POST /api/v1/batepapo/enviar/audio`

Campos:

- obrigatorio: `numero` ou `chat_jid`
- obrigatorio: `arquivo_url`, `arquivo_base64` ou `caminho_local`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `nome_arquivo`
- opcional: `mime_type`
- opcional: `duracao_segundos`
- opcional: `ptt`

Regras do audio:

- se `ptt` for omitido e o audio for `ogg/opus`, a API envia como gravado por padrao
- se quiser forcar explicitamente voz/gravado, envie `ptt: true`
- se `ptt=true`, o audio precisa ser `ogg/opus`
- `mp3`, `m4a`, `aac`, `wav` e similares devem ser enviados como audio comum
- para audio gravado, nao basta mandar `mp3` com `ptt=true`; o formato precisa ser `ogg/opus`
- `duracao_segundos` e opcional; para `ogg/opus` a API tenta calcular automaticamente
- para base64 puro sem prefixo `data:...`, envie `mime_type` ou `nome_arquivo`

Exemplo com arquivo local:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "caminho_local": "D:/midias/audio.ogg"
}
```

Exemplo com URL:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_url": "https://exemplo.com/audio.ogg"
}
```

Exemplo com base64 gravado:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_base64": "data:audio/ogg;base64,T2dnUwACAAAAAAAAA...",
  "ptt": true,
  "duracao_segundos": 12
}
```

Exemplo com base64 mp3:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_base64": "data:audio/mpeg;base64,SUQzBAAAAAAA...",
  "mime_type": "audio/mpeg",
  "nome_arquivo": "audio.mp3"
}
```

### `POST /api/v1/batepapo/enviar/documento`

Campos:

- obrigatorio: `numero` ou `chat_jid`
- obrigatorio: `arquivo_url`, `arquivo_base64` ou `caminho_local`
- opcional: `instancia`
- opcional: `grupo`
- opcional: `nome_arquivo`
- opcional: `legenda`

Exemplo com arquivo local:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "caminho_local": "D:/midias/contrato.pdf",
  "nome_arquivo": "contrato.pdf",
  "legenda": "Segue o documento"
}
```

Exemplo com URL:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_url": "https://exemplo.com/contrato.pdf",
  "nome_arquivo": "contrato.pdf",
  "legenda": "Segue o documento"
}
```

Exemplo com base64:

```json
{
  "instancia": "id-da-instancia",
  "numero": "5511999999999",
  "arquivo_base64": "data:application/pdf;base64,JVBERi0xLjQKJcTl8uXr...",
  "nome_arquivo": "contrato.pdf",
  "legenda": "Segue o documento"
}
```
