package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// AssinaturaValida confere o cabecalho X-Hub-Signature-256 que a Meta manda em
// cada webhook (HMAC-SHA256 do corpo com o App Secret).
func AssinaturaValida(corpo []byte, cabecalho, appSecret string) bool {
	assinatura, ok := strings.CutPrefix(strings.TrimSpace(cabecalho), "sha256=")
	if !ok || appSecret == "" {
		return false
	}
	recebida, err := hex.DecodeString(assinatura)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(corpo)
	return hmac.Equal(recebida, mac.Sum(nil))
}

// Webhook e o corpo que a Meta manda para a URL de callback.
type Webhook struct {
	Objeto   string `json:"object"`
	Entradas []struct {
		ID       string `json:"id"`
		Mudancas []struct {
			Campo string `json:"field"`
			Valor Valor  `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

type Valor struct {
	Metadados struct {
		NumeroExibicao string `json:"display_phone_number"`
		PhoneNumberID  string `json:"phone_number_id"`
	} `json:"metadata"`
	Contatos []struct {
		Perfil struct {
			Nome string `json:"name"`
		} `json:"profile"`
		WaID string `json:"wa_id"`
	} `json:"contacts"`
	Mensagens []Mensagem `json:"messages"`
	Status    []Status   `json:"statuses"`
}

// Midia e o formato comum de imagem, audio, video, documento e figurinha.
type Midia struct {
	ID          string `json:"id"`
	MimeType    string `json:"mime_type"`
	SHA256      string `json:"sha256"`
	Legenda     string `json:"caption"`
	NomeArquivo string `json:"filename"`
	Voz         bool   `json:"voice"`
	Animada     bool   `json:"animated"`
}

type Mensagem struct {
	De        string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Tipo      string `json:"type"`
	Contexto  *struct {
		De string `json:"from"`
		ID string `json:"id"`
	} `json:"context"`
	Texto *struct {
		Corpo string `json:"body"`
	} `json:"text"`
	Imagem    *Midia `json:"image"`
	Audio     *Midia `json:"audio"`
	Video     *Midia `json:"video"`
	Documento *Midia `json:"document"`
	Figurinha *Midia `json:"sticker"`
	Local     *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Nome      string  `json:"name"`
		Endereco  string  `json:"address"`
		URL       string  `json:"url"`
	} `json:"location"`
	Contatos []map[string]any `json:"contacts"`
	Reacao   *struct {
		MensagemID string `json:"message_id"`
		Emoji      string `json:"emoji"`
	} `json:"reaction"`
	// Botao e a resposta a um botao de template.
	Botao *struct {
		Texto   string `json:"text"`
		Payload string `json:"payload"`
	} `json:"button"`
	Interativo *struct {
		Tipo          string `json:"type"`
		RespostaBotao *struct {
			ID     string `json:"id"`
			Titulo string `json:"title"`
		} `json:"button_reply"`
		RespostaLista *struct {
			ID        string `json:"id"`
			Titulo    string `json:"title"`
			Descricao string `json:"description"`
		} `json:"list_reply"`
		RespostaFlow *struct {
			Nome     string `json:"name"`
			Corpo    string `json:"body"`
			Resposta string `json:"response_json"`
		} `json:"nfm_reply"`
	} `json:"interactive"`
	Erros []map[string]any `json:"errors"`
}

type Status struct {
	ID           string           `json:"id"`
	Status       string           `json:"status"`
	Timestamp    string           `json:"timestamp"`
	Destinatario string           `json:"recipient_id"`
	Erros        []map[string]any `json:"errors"`
	Conversa     map[string]any   `json:"conversation"`
	Cobranca     map[string]any   `json:"pricing"`
}

// MidiaDe devolve a midia da mensagem e o tipo usado nos nossos webhooks.
func (m Mensagem) MidiaDe() (*Midia, string) {
	switch {
	case m.Imagem != nil:
		return m.Imagem, "imagem"
	case m.Audio != nil:
		return m.Audio, "audio"
	case m.Video != nil:
		return m.Video, "video"
	case m.Documento != nil:
		return m.Documento, "documento"
	case m.Figurinha != nil:
		return m.Figurinha, "sticker"
	default:
		return nil, ""
	}
}

func horario(timestamp string) time.Time {
	segundos, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || segundos <= 0 {
		return time.Now().UTC()
	}
	return time.Unix(segundos, 0).UTC()
}

// conteudoETipo traduz a mensagem da Meta para o par conteudo/tipo e os
// extras usados nos webhooks das instancias por QR code.
func (m Mensagem) conteudoETipo() (string, string, map[string]any, map[string]any) {
	if midia, tipo := m.MidiaDe(); midia != nil {
		extras := map[string]any{"mime_type": midia.MimeType}
		mensagem := map[string]any{"mime_type": midia.MimeType}
		if midia.NomeArquivo != "" {
			extras["nome_arquivo"] = midia.NomeArquivo
			mensagem["nome_arquivo"] = midia.NomeArquivo
		}
		conteudo := strings.TrimSpace(midia.Legenda)
		switch tipo {
		case "audio":
			conteudo = "audio recebido"
			extras["ptt"], mensagem["ptt"] = midia.Voz, midia.Voz
		case "sticker":
			conteudo = "sticker recebido"
		}
		return conteudo, tipo, extras, mensagem
	}
	switch {
	case m.Texto != nil:
		return m.Texto.Corpo, "texto", nil, nil
	case m.Local != nil:
		local := map[string]any{
			"latitude": m.Local.Latitude, "longitude": m.Local.Longitude,
			"nome": m.Local.Nome, "endereco": m.Local.Endereco, "url": m.Local.URL,
		}
		conteudo := primeiro(m.Local.Nome, m.Local.Endereco, "localizacao recebida")
		return conteudo, "localizacao", map[string]any{"localizacao": local}, map[string]any{"localizacao": local}
	case len(m.Contatos) > 0:
		nome := "contato recebido"
		if n, ok := m.Contatos[0]["name"].(map[string]any); ok {
			if f, ok := n["formatted_name"].(string); ok && f != "" {
				nome = f
			}
		}
		tipo := "contato"
		if len(m.Contatos) > 1 {
			tipo = "contatos"
		}
		return nome, tipo, map[string]any{"contatos": m.Contatos}, map[string]any{"contatos": m.Contatos}
	case m.Reacao != nil:
		conteudo := primeiro(m.Reacao.Emoji, "reacao removida")
		reacao := map[string]any{"texto": m.Reacao.Emoji, "removida": m.Reacao.Emoji == "", "mensagem_id": m.Reacao.MensagemID}
		extras := map[string]any{"reacao_texto": m.Reacao.Emoji, "reacao_removida": m.Reacao.Emoji == "", "mensagem_reagida_id": m.Reacao.MensagemID}
		return conteudo, "reacao", extras, map[string]any{"reacao": reacao}
	case m.Botao != nil:
		return botao(m.Botao.Payload, m.Botao.Texto, "template")
	case m.Interativo != nil && m.Interativo.RespostaBotao != nil:
		return botao(m.Interativo.RespostaBotao.ID, m.Interativo.RespostaBotao.Titulo, "quick_reply")
	case m.Interativo != nil && m.Interativo.RespostaLista != nil:
		r := m.Interativo.RespostaLista
		lista := map[string]any{"id": r.ID, "titulo": r.Titulo, "descricao": r.Descricao}
		extras := map[string]any{"lista_id": r.ID, "lista_titulo": r.Titulo, "lista_descricao": r.Descricao}
		return primeiro(r.Descricao, r.Titulo, r.ID), "lista", extras, map[string]any{"lista": lista}
	case m.Interativo != nil && m.Interativo.RespostaFlow != nil:
		r := m.Interativo.RespostaFlow
		var resposta any
		_ = json.Unmarshal([]byte(r.Resposta), &resposta)
		flow := map[string]any{"nome": r.Nome, "corpo": r.Corpo, "resposta": resposta}
		return primeiro(r.Corpo, "formulario respondido"), "formulario", map[string]any{"formulario": flow}, map[string]any{"formulario": flow}
	default:
		return "mensagem nao suportada (" + m.Tipo + ")", m.Tipo, map[string]any{"erros": m.Erros}, nil
	}
}

func botao(id, texto, tipo string) (string, string, map[string]any, map[string]any) {
	texto = primeiro(texto, id)
	dados := map[string]any{"id": id, "texto": texto, "tipo": tipo}
	extras := map[string]any{"botao_id": id, "botao_texto": texto, "botao_tipo": tipo}
	return texto, "botao", extras, map[string]any{"botao": dados}
}

func primeiro(valores ...string) string {
	for _, v := range valores {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// DadosMensagem monta o payload do evento "mensagens", no mesmo formato das
// instancias por QR code, para o cliente tratar os dois tipos igual.
func DadosMensagem(v Valor, m Mensagem) map[string]any {
	jid := m.De + "@s.whatsapp.net"
	nome := ""
	for _, c := range v.Contatos {
		if c.WaID == m.De {
			nome = c.Perfil.Nome
		}
	}
	recebida := horario(m.Timestamp)
	conteudo, tipo, extras, extrasMensagem := m.conteudoETipo()
	dados := map[string]any{
		"mensagem_id":      m.ID,
		"chat_jid":         jid,
		"chat_numero":      m.De,
		"grupo":            false,
		"enviado_por_mim":  false,
		"direcao":          "entrada",
		"acao":             "recebida",
		"origem":           "meta",
		"historico":        false,
		"remetente":        jid,
		"remetente_jid":    jid,
		"remetente_numero": m.De,
		"nome_remetente":   nome,
		"conteudo":         conteudo,
		"tipo":             tipo,
		"recebida_em":      recebida,
		"conversa":         map[string]any{"jid": jid, "numero": m.De, "grupo": false},
		"autor":            map[string]any{"jid": jid, "numero": m.De, "nome": nome},
	}
	mensagem := map[string]any{
		"id": m.ID, "tipo": tipo, "acao": "recebida", "conteudo": conteudo, "direcao": "entrada",
		"enviado_por_mim": false, "origem": "meta", "historico": false, "recebida_em": recebida,
	}
	if m.Contexto != nil && m.Contexto.ID != "" {
		dados["resposta_mensagem_id"] = m.Contexto.ID
		mensagem["resposta"] = map[string]any{"mensagem_id": m.Contexto.ID, "remetente_numero": m.Contexto.De}
	}
	for chave, valor := range extras {
		dados[chave] = valor
	}
	for chave, valor := range extrasMensagem {
		mensagem[chave] = valor
	}
	dados["mensagem"] = mensagem
	return dados
}

// statusRecibo traduz o status da Meta para os nomes usados no evento
// "recibos".
var statusRecibo = map[string]string{
	"sent":      "enviada",
	"delivered": "entregue",
	"read":      "lida",
	"played":    "ouvida",
	"failed":    "falhou",
	"deleted":   "apagada",
}

// DadosRecibo monta o payload do evento "recibos" a partir de um status da
// Meta (enviada, entregue, lida ou falhou).
func DadosRecibo(s Status) map[string]any {
	jid := s.Destinatario + "@s.whatsapp.net"
	status := statusRecibo[s.Status]
	if status == "" {
		status = s.Status
	}
	ocorrido := horario(s.Timestamp)
	dados := map[string]any{
		"mensagem_id":      s.ID,
		"mensagens_id":     []string{s.ID},
		"status":           status,
		"tipo_recibo":      s.Status,
		"chat_jid":         jid,
		"chat_numero":      s.Destinatario,
		"grupo":            false,
		"enviado_por_mim":  true,
		"direcao":          "saida",
		"remetente":        jid,
		"remetente_jid":    jid,
		"remetente_numero": s.Destinatario,
		"ocorrido_em":      ocorrido,
		"origem":           "meta",
		"conversa":         map[string]any{"jid": jid, "numero": s.Destinatario, "grupo": false},
		"recibo":           map[string]any{"status": status, "tipo": s.Status, "mensagens_id": []string{s.ID}, "ocorrido_em": ocorrido},
	}
	if len(s.Erros) > 0 {
		dados["erros"] = s.Erros
	}
	if s.Cobranca != nil {
		dados["cobranca"] = s.Cobranca
	}
	return dados
}
