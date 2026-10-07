package meta

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"dyalog-api-go/internal/models"
)

// ErrNaoSuportado indica recurso que existe nas instancias por QR code mas
// nao na API oficial.
var ErrNaoSuportado = errors.New("recurso nao disponivel na API oficial da Meta")

// Destinatario converte numero ou chat_jid no "to" da Meta (so digitos).
func Destinatario(numero, chatJID string) (string, error) {
	alvo := strings.TrimSpace(numero)
	if jid := strings.TrimSpace(chatJID); jid != "" {
		if strings.HasSuffix(jid, "@g.us") || strings.HasSuffix(jid, "@newsletter") || strings.HasSuffix(jid, "@broadcast") {
			return "", fmt.Errorf("%w: envio para grupo, canal ou lista de transmissao", ErrNaoSuportado)
		}
		if alvo == "" {
			alvo = strings.SplitN(strings.SplitN(jid, "@", 2)[0], ":", 2)[0]
		}
	}
	digitos := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, alvo)
	if len(digitos) < 8 {
		return "", fmt.Errorf("numero de destino invalido: %q", alvo)
	}
	return digitos, nil
}

func mensagemBase(para, tipo string) map[string]any {
	return map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                para,
		"type":              tipo,
	}
}

// responderA marca a mensagem como resposta a outra (citacao).
func responderA(msg map[string]any, mensagemID string) map[string]any {
	if id := strings.TrimSpace(mensagemID); id != "" {
		msg["context"] = map[string]any{"message_id": id}
	}
	return msg
}

func MontarTexto(para, texto, respostaID string) map[string]any {
	msg := mensagemBase(para, "text")
	msg["text"] = map[string]any{
		"body":        texto,
		"preview_url": strings.Contains(texto, "http://") || strings.Contains(texto, "https://"),
	}
	return responderA(msg, respostaID)
}

// MontarMidia monta imagem, video, audio, documento ou figurinha. midia e
// {"id": ...} (subida na Meta) ou {"link": ...} (URL publica).
func MontarMidia(para, tipo string, midia map[string]any, legenda, nomeArquivo string) map[string]any {
	conteudo := map[string]any{}
	for chave, valor := range midia {
		conteudo[chave] = valor
	}
	if legenda = strings.TrimSpace(legenda); legenda != "" && (tipo == "image" || tipo == "video" || tipo == "document") {
		conteudo["caption"] = legenda
	}
	if nomeArquivo = strings.TrimSpace(nomeArquivo); nomeArquivo != "" && tipo == "document" {
		conteudo["filename"] = nomeArquivo
	}
	msg := mensagemBase(para, tipo)
	msg[tipo] = conteudo
	return msg
}

func MontarLocalizacao(para string, req models.EnvioLocalizacaoRequest) map[string]any {
	local := map[string]any{"latitude": req.Latitude, "longitude": req.Longitude}
	if nome := strings.TrimSpace(req.Nome); nome != "" {
		local["name"] = nome
	}
	if endereco := strings.TrimSpace(req.Endereco); endereco != "" {
		local["address"] = endereco
	}
	msg := mensagemBase(para, "location")
	msg["location"] = local
	return responderA(msg, req.RespostaMensagemID)
}

func MontarContatos(para string, req models.EnvioContatoRequest) map[string]any {
	lista := req.Contatos
	if len(lista) == 0 {
		lista = []models.ContatoEnvioRequest{{Nome: req.Nome, Telefone: req.Telefone, Organizacao: req.Organizacao}}
	}
	contatos := make([]map[string]any, 0, len(lista))
	for _, c := range lista {
		nome := strings.TrimSpace(c.Nome)
		contato := map[string]any{
			"name": map[string]any{"formatted_name": nome, "first_name": nome},
		}
		if telefone := strings.TrimSpace(c.Telefone); telefone != "" {
			contato["phones"] = []map[string]any{{"phone": telefone, "type": "CELL"}}
		}
		if org := strings.TrimSpace(c.Organizacao); org != "" {
			contato["org"] = map[string]any{"company": org}
		}
		contatos = append(contatos, contato)
	}
	msg := mensagemBase(para, "contacts")
	msg["contacts"] = contatos
	return responderA(msg, req.RespostaMensagemID)
}

// MontarReacao reage a uma mensagem; emoji vazio remove a reacao.
func MontarReacao(para, mensagemID, emoji string) map[string]any {
	msg := mensagemBase(para, "reaction")
	msg["reaction"] = map[string]any{"message_id": mensagemID, "emoji": emoji}
	return msg
}

func MontarTemplate(para string, req models.EnvioTemplateRequest) map[string]any {
	idioma := strings.TrimSpace(req.Idioma)
	if idioma == "" {
		idioma = "pt_BR"
	}
	template := map[string]any{
		"name":     strings.TrimSpace(req.Nome),
		"language": map[string]any{"code": idioma},
	}
	switch {
	case len(req.Componentes) > 0:
		template["components"] = req.Componentes
	case len(req.Parametros) > 0:
		parametros := make([]map[string]any, 0, len(req.Parametros))
		for _, p := range req.Parametros {
			parametros = append(parametros, map[string]any{"type": "text", "text": p})
		}
		template["components"] = []map[string]any{{"type": "body", "parameters": parametros}}
	}
	msg := mensagemBase(para, "template")
	msg["template"] = template
	return msg
}

// interativa monta o envelope de mensagem interativa com titulo e rodape
// opcionais.
func interativa(para, tipo, titulo, corpo, rodape string, acao map[string]any, respostaID string) map[string]any {
	conteudo := map[string]any{
		"type":   tipo,
		"body":   map[string]any{"text": corpo},
		"action": acao,
	}
	if titulo = strings.TrimSpace(titulo); titulo != "" {
		conteudo["header"] = map[string]any{"type": "text", "text": titulo}
	}
	if rodape = strings.TrimSpace(rodape); rodape != "" {
		conteudo["footer"] = map[string]any{"text": rodape}
	}
	msg := mensagemBase(para, "interactive")
	msg["interactive"] = conteudo
	return responderA(msg, respostaID)
}

// MontarBotoes aceita ate 3 botoes de resposta, ou 1 botao de link. A API
// oficial nao permite misturar os dois, nem botao de ligar/copiar fora de
// template.
func MontarBotoes(para string, req models.EnvioBotoesRequest, corpo string) (map[string]any, error) {
	var resposta, links []models.BotaoRequest
	for _, botao := range req.Botoes {
		switch botao.TipoNormalizado() {
		case models.BotaoResposta:
			resposta = append(resposta, botao)
		case models.BotaoURL:
			links = append(links, botao)
		default:
			return nil, fmt.Errorf("%w: botao de ligar ou copiar so existe em template aprovado na Meta", ErrNaoSuportado)
		}
	}
	textoBotao := func(b models.BotaoRequest) string { return strings.TrimSpace(b.Texto) }
	switch {
	case len(links) == 0:
		botoes := make([]map[string]any, 0, len(resposta))
		for _, b := range resposta {
			botoes = append(botoes, map[string]any{
				"type":  "reply",
				"reply": map[string]any{"id": strings.TrimSpace(b.ID), "title": textoBotao(b)},
			})
		}
		return interativa(para, "button", req.Titulo, corpo, req.Rodape, map[string]any{"buttons": botoes}, req.RespostaMensagemID), nil
	case len(links) == 1 && len(resposta) == 0:
		acao := map[string]any{
			"name":       "cta_url",
			"parameters": map[string]any{"display_text": textoBotao(links[0]), "url": links[0].Link()},
		}
		return interativa(para, "cta_url", req.Titulo, corpo, req.Rodape, acao, req.RespostaMensagemID), nil
	default:
		return nil, fmt.Errorf("%w: a API oficial aceita ate 3 botoes de resposta ou 1 botao de link, sem misturar", ErrNaoSuportado)
	}
}

// Limites da lista na API oficial.
const (
	maxLinhasLista     = 10
	maxTituloLinha     = 24
	maxDescricaoLinha  = 72
	maxTextoBotaoLista = 20
)

func MontarLista(para string, req models.EnvioListaRequest, corpo string) (map[string]any, error) {
	total := 0
	secoes := make([]map[string]any, 0, len(req.Secoes))
	for _, secao := range req.Secoes {
		linhas := make([]map[string]any, 0, len(secao.Linhas))
		for _, linha := range secao.Linhas {
			titulo := strings.TrimSpace(linha.Titulo)
			if len([]rune(titulo)) > maxTituloLinha {
				return nil, fmt.Errorf("titulo da linha %q passa de %d caracteres (limite da Meta)", titulo, maxTituloLinha)
			}
			item := map[string]any{"id": strings.TrimSpace(linha.ID), "title": titulo}
			if descricao := strings.TrimSpace(linha.Descricao); descricao != "" {
				if len([]rune(descricao)) > maxDescricaoLinha {
					return nil, fmt.Errorf("descricao da linha %q passa de %d caracteres (limite da Meta)", titulo, maxDescricaoLinha)
				}
				item["description"] = descricao
			}
			linhas = append(linhas, item)
		}
		total += len(linhas)
		item := map[string]any{"rows": linhas}
		if titulo := strings.TrimSpace(secao.Titulo); titulo != "" {
			item["title"] = titulo
		}
		secoes = append(secoes, item)
	}
	if total > maxLinhasLista {
		return nil, fmt.Errorf("a lista da API oficial aceita no maximo %d linhas no total", maxLinhasLista)
	}
	if len([]rune(strings.TrimSpace(req.BotaoTexto))) > maxTextoBotaoLista {
		return nil, fmt.Errorf("botao_texto passa de %d caracteres (limite da Meta)", maxTextoBotaoLista)
	}
	acao := map[string]any{"button": strings.TrimSpace(req.BotaoTexto), "sections": secoes}
	return interativa(para, "list", req.Titulo, corpo, req.Rodape, acao, req.RespostaMensagemID), nil
}
