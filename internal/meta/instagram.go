package meta

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// Instagram: API do Instagram com login do Instagram (graph.instagram.com).
// Os erros tem o mesmo formato da Graph API, entao reaproveita o Cliente.

// SufixoInstagram fecha o chat_jid das conversas do Instagram
// ("<IGSID>@instagram"), para nao confundir com numero do WhatsApp.
const SufixoInstagram = "@instagram"

// ClienteInstagram fala com graph.instagram.com e, no login, com
// api.instagram.com.
type ClienteInstagram struct {
	*Cliente
	oauth string
}

// NovoClienteInstagram cria o cliente. base vazio usa o endereco da Meta;
// outro valor serve para testes (vale para a Graph e para o login).
func NovoClienteInstagram(versao, base string) *ClienteInstagram {
	oauth := strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base, oauth = "https://graph.instagram.com", "https://api.instagram.com"
	}
	return &ClienteInstagram{Cliente: NovoCliente(versao, base), oauth: oauth}
}

// PerfilInstagram e a conta profissional dona do token.
type PerfilInstagram struct {
	ContaID string `json:"user_id"`
	Usuario string `json:"username"`
	Nome    string `json:"name"`
	FotoURL string `json:"profile_picture_url"`
}

// Perfil confere o token e devolve a conta. ContaID e o id que chega nos
// webhooks.
func (c *ClienteInstagram) Perfil(ctx context.Context, token string) (PerfilInstagram, error) {
	var p PerfilInstagram
	err := c.requisitar(ctx, http.MethodGet, c.url("me")+"?fields=user_id,username,name,profile_picture_url", token, nil, "", &p)
	if err == nil && p.ContaID == "" {
		err = errors.New("o Instagram nao devolveu o id da conta")
	}
	return p, err
}

// AssinarWebhooks inscreve o app nos webhooks da conta. Sem isso a URL de
// callback fica verificada mas as mensagens nao chegam.
func (c *ClienteInstagram) AssinarWebhooks(ctx context.Context, token string) error {
	campos := "messages,messaging_postbacks,messaging_seen,message_reactions"
	return c.requisitar(ctx, http.MethodPost, c.url("me/subscribed_apps")+"?subscribed_fields="+campos, token, nil, "", nil)
}

// TrocarCode troca o code do login do Instagram pelo token de longa duracao
// (60 dias). redirectURI precisa ser identico ao usado no login.
func (c *ClienteInstagram) TrocarCode(ctx context.Context, appID, appSecret, redirectURI, code string) (string, error) {
	form := url.Values{
		"client_id": {appID}, "client_secret": {appSecret}, "grant_type": {"authorization_code"},
		"redirect_uri": {redirectURI}, "code": {strings.TrimSuffix(code, "#_")},
	}
	var curto struct {
		AccessToken string `json:"access_token"`
		Dados       []struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := c.requisitar(ctx, http.MethodPost, c.oauth+"/oauth/access_token", "", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", &curto); err != nil {
		return "", err
	}
	token := curto.AccessToken
	if token == "" && len(curto.Dados) > 0 {
		token = curto.Dados[0].AccessToken
	}
	if token == "" {
		return "", errors.New("o Instagram nao devolveu o token de acesso")
	}
	consulta := url.Values{"grant_type": {"ig_exchange_token"}, "client_secret": {appSecret}, "access_token": {token}}
	return c.tokenLongo(ctx, c.base+"/access_token?"+consulta.Encode())
}

// RenovarToken estica o token por mais 60 dias (precisa ter mais de 24h).
func (c *ClienteInstagram) RenovarToken(ctx context.Context, token string) (string, error) {
	consulta := url.Values{"grant_type": {"ig_refresh_token"}, "access_token": {token}}
	return c.tokenLongo(ctx, c.base+"/refresh_access_token?"+consulta.Encode())
}

func (c *ClienteInstagram) tokenLongo(ctx context.Context, endereco string) (string, error) {
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := c.requisitar(ctx, http.MethodGet, endereco, "", nil, "", &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", errors.New("o Instagram nao devolveu o token de longa duracao")
	}
	return resp.AccessToken, nil
}

// NomeContato busca nome e @ de quem mandou mensagem (o webhook so traz o id).
func (c *ClienteInstagram) NomeContato(ctx context.Context, token, igsid string) (nome, usuario string) {
	var p struct {
		Nome    string `json:"name"`
		Usuario string `json:"username"`
	}
	if err := c.requisitar(ctx, http.MethodGet, c.url(igsid)+"?fields=name,username", token, nil, "", &p); err != nil {
		return "", ""
	}
	return p.Nome, p.Usuario
}

// RespostaEnvioInstagram e o retorno de POST /me/messages.
type RespostaEnvioInstagram struct {
	Destinatario string `json:"recipient_id"`
	MensagemID   string `json:"message_id"`
}

// Enviar manda a mensagem (ou acao, como mark_seen) para o contato.
func (c *ClienteInstagram) Enviar(ctx context.Context, token, para string, corpo map[string]any) (RespostaEnvioInstagram, error) {
	corpo["recipient"] = map[string]any{"id": para}
	var resp RespostaEnvioInstagram
	err := c.postarJSON(ctx, "me/messages", token, corpo, &resp)
	return resp, err
}

// BaixarURL baixa o anexo recebido (URL da CDN, sem token).
func (c *ClienteInstagram) BaixarURL(ctx context.Context, endereco string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao baixar anexo do Instagram: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("Instagram respondeu HTTP %d ao baixar o anexo", resp.StatusCode)
	}
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	return dados, resp.Header.Get("Content-Type"), err
}

// EnviadorInstagram faz, pelo Direct, os envios das rotas /batepapo/... O que
// o Instagram nao tem volta ErrNaoSuportado.
type EnviadorInstagram struct {
	cliente *ClienteInstagram
	store   store.InstagramStore
}

func NovoEnviadorInstagram(cliente *ClienteInstagram, s store.InstagramStore) *EnviadorInstagram {
	return &EnviadorInstagram{cliente: cliente, store: s}
}

func (e *EnviadorInstagram) Cliente() *ClienteInstagram { return e.cliente }

func (e *EnviadorInstagram) credenciais(ctx context.Context, instanciaID string) (models.CredenciaisInstagram, error) {
	cred, err := e.store.ObterCredenciaisInstagram(ctx, instanciaID)
	if errors.Is(err, store.ErrCredenciaisInstagramNaoEncontradas) || (err == nil && cred.AccessToken == "") {
		return models.CredenciaisInstagram{}, fmt.Errorf("%w: instancia do Instagram sem conta conectada; conecte a conta ou cadastre o access_token", ErrEntradaInvalidaInstagram)
	}
	return cred, err
}

// DestinatarioInstagram tira o IGSID do chat_jid ("<IGSID>@instagram") ou do
// campo numero.
func DestinatarioInstagram(numero, chatJID string) (string, error) {
	para := strings.TrimSpace(numero)
	if jid := strings.TrimSpace(chatJID); jid != "" {
		if !strings.HasSuffix(jid, SufixoInstagram) {
			return "", fmt.Errorf("%w: chat_jid %q nao e uma conversa do Instagram", ErrEntradaInvalidaInstagram, jid)
		}
		para = strings.TrimSuffix(jid, SufixoInstagram)
	}
	if para == "" {
		return "", fmt.Errorf("%w: informe chat_jid ou numero (id do contato no Instagram)", ErrEntradaInvalidaInstagram)
	}
	return para, nil
}

// ErrEntradaInvalidaInstagram indica destino que nao e do Instagram.
var ErrEntradaInvalidaInstagram = errors.New("destino invalido")

func (e *EnviadorInstagram) enviar(ctx context.Context, instanciaID, numero, chatJID, tipo string, corpo map[string]any) (models.ResultadoEnvio, error) {
	cred, err := e.credenciais(ctx, instanciaID)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	para, err := DestinatarioInstagram(numero, chatJID)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	resp, err := e.cliente.Enviar(ctx, cred.AccessToken, para, corpo)
	if err != nil {
		fmt.Printf("instagram: envio de %s para %s recusado na instancia %s: %v\n", tipo, para, instanciaID, err)
		return models.ResultadoEnvio{}, err
	}
	return models.ResultadoEnvio{
		Instancia:  instanciaID,
		Numero:     para,
		ChatJID:    cmp.Or(resp.Destinatario, para) + SufixoInstagram,
		MensagemID: resp.MensagemID,
		Status:     "enviada",
		Tipo:       tipo,
		Modo:       "instagram",
	}, nil
}

func (e *EnviadorInstagram) EnviarTexto(ctx context.Context, req models.EnvioTextoRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "texto", map[string]any{"message": map[string]any{"text": req.Mensagem}})
}

// EnviarBotoes usa respostas rapidas (ate 13 botoes de texto).
func (e *EnviadorInstagram) EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error) {
	texto := cmp.Or(strings.TrimSpace(req.Texto), strings.TrimSpace(req.Mensagem), strings.TrimSpace(req.Titulo))
	if texto == "" || len(req.Botoes) == 0 || len(req.Botoes) > 13 {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe o texto e de 1 a 13 botoes", ErrEntradaInvalidaInstagram)
	}
	rapidas := make([]map[string]any, 0, len(req.Botoes))
	for i, b := range req.Botoes {
		titulo := cmp.Or(strings.TrimSpace(b.Texto), strings.TrimSpace(b.DisplayText))
		if titulo == "" {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: botao %d sem texto", ErrEntradaInvalidaInstagram, i+1)
		}
		rapidas = append(rapidas, map[string]any{"content_type": "text", "title": titulo, "payload": cmp.Or(strings.TrimSpace(b.ID), titulo)})
	}
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "botoes", map[string]any{"message": map[string]any{"text": texto, "quick_replies": rapidas}})
}

// ReagirMensagem: o Instagram so tem o coracao; emoji vazio tira a reacao.
func (e *EnviadorInstagram) ReagirMensagem(ctx context.Context, req models.ReagirMensagemRequest) (models.ResultadoEnvio, error) {
	acao := "react"
	if strings.TrimSpace(req.Emoji) == "" {
		acao = "unreact"
	}
	resultado, err := e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "reacao", map[string]any{
		"sender_action": acao, "payload": map[string]any{"message_id": req.MensagemID, "reaction": "love"},
	})
	resultado.MensagemID = cmp.Or(resultado.MensagemID, req.MensagemID)
	return resultado, err
}

func (e *EnviadorInstagram) EnviarPresenca(ctx context.Context, req models.EnvioPresencaRequest) (models.ResultadoPresenca, error) {
	acao := "typing_on"
	if a := strings.ToLower(cmp.Or(req.Acao, req.Type)); strings.Contains(a, "paus") || strings.Contains(a, "parar") || a == "available" || a == "disponivel" {
		acao = "typing_off"
	}
	if _, err := e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "presenca", map[string]any{"sender_action": acao}); err != nil {
		return models.ResultadoPresenca{}, err
	}
	return models.ResultadoPresenca{Instancia: req.Instancia, Numero: req.Numero, ChatJID: req.ChatJID, Status: acao}, nil
}

// MarcarLida marca a conversa como vista (o Instagram marca a conversa, nao
// cada mensagem).
func (e *EnviadorInstagram) MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error) {
	if _, err := e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "read", map[string]any{"sender_action": "mark_seen"}); err != nil {
		return models.ResultadoMarcarLida{}, err
	}
	ids := req.MensagensID
	if id := strings.TrimSpace(req.MensagemID); id != "" {
		ids = append([]string{id}, ids...)
	}
	return models.ResultadoMarcarLida{Instancia: req.Instancia, Numero: req.Numero, ChatJID: req.ChatJID, MensagensID: ids, Status: "lida", Tipo: "read", LidaEm: time.Now().UTC()}, nil
}

func (e *EnviadorInstagram) EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "image", "imagem")
}

func (e *EnviadorInstagram) EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "audio", "audio")
}

func (e *EnviadorInstagram) EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "file", "documento")
}

// enviarMidia: o Instagram so baixa midia de URL publica.
// ponytail: base64/arquivo local recusados; subir no storage e mandar a URL
// quando alguem precisar.
func (e *EnviadorInstagram) enviarMidia(ctx context.Context, req models.EnvioMidiaRequest, tipoIG, tipo string) (models.ResultadoEnvio, error) {
	endereco := strings.TrimSpace(req.ArquivoURL)
	if !strings.HasPrefix(endereco, "http") {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: no Instagram a midia precisa vir em arquivo_url (URL publica)", ErrNaoSuportado)
	}
	resultado, err := e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, tipo, map[string]any{
		"message": map[string]any{"attachment": map[string]any{"type": tipoIG, "payload": map[string]any{"url": endereco}}},
	})
	if err == nil && strings.TrimSpace(req.Legenda) != "" {
		// Anexo nao tem legenda no Instagram: vai como texto logo depois.
		_, err = e.EnviarTexto(ctx, models.EnvioTextoRequest{Instancia: req.Instancia, Numero: req.Numero, ChatJID: req.ChatJID, Mensagem: req.Legenda})
	}
	return resultado, err
}

func instagramNaoSuportado(recurso string) error {
	return fmt.Errorf("%w: %s no Instagram", ErrNaoSuportado, recurso)
}

func (e *EnviadorInstagram) EditarTexto(context.Context, models.EditarTextoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("editar mensagem")
}

func (e *EnviadorInstagram) ApagarMensagem(context.Context, models.ApagarMensagemRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("apagar mensagem")
}

func (e *EnviadorInstagram) EnviarLista(context.Context, models.EnvioListaRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("lista; use botoes")
}

func (e *EnviadorInstagram) EnviarEnquete(context.Context, models.EnvioEnqueteRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("enquete; use botoes")
}

func (e *EnviadorInstagram) EnviarCobrancaPix(context.Context, models.EnvioCobrancaPixRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("cobranca Pix")
}

func (e *EnviadorInstagram) EnviarLocalizacao(context.Context, models.EnvioLocalizacaoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("localizacao")
}

func (e *EnviadorInstagram) EnviarContato(context.Context, models.EnvioContatoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("contato")
}

func (e *EnviadorInstagram) EnviarFigurinha(context.Context, models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("figurinha")
}

func (e *EnviadorInstagram) EnviarEvento(context.Context, models.EnvioEventoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("evento")
}

func (e *EnviadorInstagram) EnviarCarrossel(context.Context, models.EnvioCarrosselRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("carrossel")
}

func (e *EnviadorInstagram) PostarStatus(context.Context, models.EnvioStatusRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, instagramNaoSuportado("status")
}

func (e *EnviadorInstagram) CriarLinkChamada(context.Context, models.LinkChamadaRequest) (models.LinkChamadaResultado, error) {
	return models.LinkChamadaResultado{}, instagramNaoSuportado("link de chamada")
}

// ── Webhook ─────────────────────────────────────────────────────────────────

// WebhookInstagram e o corpo que o Instagram manda (object "instagram").
type WebhookInstagram struct {
	Objeto   string `json:"object"`
	Entradas []struct {
		ID      string            `json:"id"`
		Eventos []EventoInstagram `json:"messaging"`
	} `json:"entry"`
}

type idInstagram struct {
	ID string `json:"id"`
}

// EventoInstagram e um item de entry[].messaging: mensagem, reacao, leitura
// ou clique em botao.
type EventoInstagram struct {
	Remetente    idInstagram `json:"sender"`
	Destinatario idInstagram `json:"recipient"`
	Timestamp    int64       `json:"timestamp"`
	Mensagem     *struct {
		MID     string `json:"mid"`
		Texto   string `json:"text"`
		Eco     bool   `json:"is_echo"`
		Apagada bool   `json:"is_deleted"`
		Anexos  []struct {
			Tipo    string `json:"type"`
			Payload struct {
				URL string `json:"url"`
			} `json:"payload"`
		} `json:"attachments"`
		RespostaA *struct {
			MID   string `json:"mid"`
			Story *struct {
				URL string `json:"url"`
				ID  string `json:"id"`
			} `json:"story"`
		} `json:"reply_to"`
		RespostaRapida *struct {
			Payload string `json:"payload"`
		} `json:"quick_reply"`
	} `json:"message"`
	Lida *struct {
		MID string `json:"mid"`
	} `json:"read"`
	Reacao *struct {
		MID    string `json:"mid"`
		Acao   string `json:"action"`
		Emoji  string `json:"emoji"`
		Reacao string `json:"reaction"`
	} `json:"reaction"`
	Postback *struct {
		MID     string `json:"mid"`
		Titulo  string `json:"title"`
		Payload string `json:"payload"`
	} `json:"postback"`
}

// Contato e o id de quem esta do outro lado da conversa (no eco, quem
// recebeu).
func (e EventoInstagram) Contato() string {
	if e.Mensagem != nil && e.Mensagem.Eco {
		return e.Destinatario.ID
	}
	return e.Remetente.ID
}

func (e EventoInstagram) horario() time.Time {
	if e.Timestamp <= 0 {
		return time.Now().UTC()
	}
	return time.UnixMilli(e.Timestamp).UTC()
}

// tiposAnexoInstagram traduz para os tipos dos nossos webhooks os anexos que
// da para baixar.
var tiposAnexoInstagram = map[string]string{"image": "imagem", "video": "video", "audio": "audio", "file": "documento"}

// AnexoInstagram devolve a URL e o nosso tipo do primeiro anexo baixavel.
func (e EventoInstagram) AnexoInstagram() (string, string) {
	if e.Mensagem == nil {
		return "", ""
	}
	for _, a := range e.Mensagem.Anexos {
		if tipo := tiposAnexoInstagram[a.Tipo]; tipo != "" && a.Payload.URL != "" {
			return a.Payload.URL, tipo
		}
	}
	return "", ""
}

// DadosMensagemInstagram monta o payload do evento "mensagens" no mesmo
// formato do WhatsApp, com origem "instagram" e chat_jid "<IGSID>@instagram".
// Devolve nil para evento que nao vira mensagem (leitura).
func DadosMensagemInstagram(e EventoInstagram, nome, usuario string) map[string]any {
	contato := e.Contato()
	jid := contato + SufixoInstagram
	eco := e.Mensagem != nil && e.Mensagem.Eco
	id, conteudo, tipo, acao := "", "", "", "recebida"
	extras := map[string]any{}
	switch {
	case e.Mensagem != nil && e.Mensagem.Apagada:
		id, tipo, acao = e.Mensagem.MID, "texto", "apagada"
	case e.Mensagem != nil:
		m := e.Mensagem
		id, conteudo, tipo = m.MID, m.Texto, "texto"
		if eco {
			acao = "enviada"
		}
		if m.RespostaRapida != nil {
			tipo = "botao"
			extras["botao_id"], extras["botao_texto"], extras["botao_tipo"] = m.RespostaRapida.Payload, m.Texto, "quick_reply"
		}
		if len(m.Anexos) > 0 {
			a := m.Anexos[0]
			tipo = cmp.Or(tiposAnexoInstagram[a.Tipo], a.Tipo)
			extras["anexo_url"] = a.Payload.URL
			if conteudo == "" && tiposAnexoInstagram[a.Tipo] == "" {
				conteudo = a.Tipo + " recebido"
			}
		}
		if m.RespostaA != nil {
			extras["resposta_mensagem_id"] = m.RespostaA.MID
			if m.RespostaA.Story != nil {
				extras["story"] = map[string]any{"id": m.RespostaA.Story.ID, "url": m.RespostaA.Story.URL}
			}
		}
	case e.Reacao != nil:
		emoji := cmp.Or(e.Reacao.Emoji, map[string]string{"love": "❤️"}[e.Reacao.Reacao], e.Reacao.Reacao)
		removida := e.Reacao.Acao == "unreact"
		if removida {
			emoji = ""
		}
		id = fmt.Sprintf("reacao-%s-%d", e.Reacao.MID, e.Timestamp)
		conteudo, tipo = primeiro(emoji, "reacao removida"), "reacao"
		extras["reacao_texto"], extras["reacao_removida"], extras["mensagem_reagida_id"] = emoji, removida, e.Reacao.MID
	case e.Postback != nil:
		id = cmp.Or(e.Postback.MID, fmt.Sprintf("postback-%s-%d", contato, e.Timestamp))
		conteudo, tipo = primeiro(e.Postback.Titulo, e.Postback.Payload), "botao"
		extras["botao_id"], extras["botao_texto"], extras["botao_tipo"] = e.Postback.Payload, conteudo, "postback"
	default:
		return nil
	}
	nome = cmp.Or(nome, usuario)
	recebida := e.horario()
	direcao := "entrada"
	if eco {
		direcao = "saida"
	}
	dados := map[string]any{
		"mensagem_id":       id,
		"chat_jid":          jid,
		"chat_numero":       contato,
		"grupo":             false,
		"enviado_por_mim":   eco,
		"direcao":           direcao,
		"acao":              acao,
		"origem":            "instagram",
		"historico":         false,
		"remetente":         jid,
		"remetente_jid":     jid,
		"remetente_numero":  contato,
		"nome_remetente":    nome,
		"usuario_instagram": usuario,
		"conteudo":          conteudo,
		"tipo":              tipo,
		"recebida_em":       recebida,
		"conversa":          map[string]any{"jid": jid, "numero": contato, "grupo": false},
		"autor":             map[string]any{"jid": jid, "numero": contato, "nome": nome},
	}
	for chave, valor := range extras {
		dados[chave] = valor
	}
	dados["mensagem"] = map[string]any{
		"id": id, "tipo": tipo, "acao": acao, "conteudo": conteudo, "direcao": direcao,
		"enviado_por_mim": eco, "origem": "instagram", "historico": false, "recebida_em": recebida,
	}
	return dados
}

// DadosReciboInstagram monta o evento "recibos" de uma leitura (o Instagram
// so avisa quando o contato viu).
func DadosReciboInstagram(e EventoInstagram) map[string]any {
	contato := e.Remetente.ID
	jid := contato + SufixoInstagram
	ocorrido := e.horario()
	ids := []string{e.Lida.MID}
	return map[string]any{
		"mensagem_id":      e.Lida.MID,
		"mensagens_id":     ids,
		"status":           "lida",
		"tipo_recibo":      "read",
		"chat_jid":         jid,
		"chat_numero":      contato,
		"grupo":            false,
		"enviado_por_mim":  true,
		"direcao":          "saida",
		"remetente":        jid,
		"remetente_jid":    jid,
		"remetente_numero": contato,
		"ocorrido_em":      ocorrido,
		"origem":           "instagram",
		"conversa":         map[string]any{"jid": jid, "numero": contato, "grupo": false},
		"recibo":           map[string]any{"status": "lida", "tipo": "read", "mensagens_id": ids, "ocorrido_em": ocorrido},
	}
}
