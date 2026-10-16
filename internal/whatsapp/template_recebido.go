package whatsapp

import (
	"strings"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
)

// Mensagens de modelo (template) que empresas mandam pela API oficial chegam
// como TemplateMessage: titulo, texto, rodape e botoes "hidratados" (ja com as
// variaveis preenchidas). Algumas vem com o modelo interativo dentro.

func extrairTemplateMensagem(modelo *waE2E.TemplateMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	if interativo := modelo.GetInteractiveMessageTemplate(); interativo != nil {
		conteudo, _, extras, extrasMensagem := extrairInterativoMensagem(interativo)
		if extras == nil {
			extras = map[string]interface{}{}
		}
		extras["template"] = true
		return conteudo, "template", extras, extrasMensagem
	}
	hidratado := modelo.GetHydratedTemplate()
	if hidratado == nil {
		hidratado = modelo.GetHydratedFourRowTemplate()
	}
	dados := map[string]interface{}{}
	var partes []string
	if hidratado != nil {
		titulo := strings.TrimSpace(hidratado.GetHydratedTitleText())
		cabecalho := ""
		switch {
		case hidratado.GetImageMessage() != nil:
			cabecalho = "imagem"
			titulo = primeiroTextoNaoVazio(titulo, hidratado.GetImageMessage().GetCaption())
		case hidratado.GetVideoMessage() != nil:
			cabecalho = "video"
			titulo = primeiroTextoNaoVazio(titulo, hidratado.GetVideoMessage().GetCaption())
		case hidratado.GetDocumentMessage() != nil:
			cabecalho = "documento"
			titulo = primeiroTextoNaoVazio(titulo, hidratado.GetDocumentMessage().GetTitle(), hidratado.GetDocumentMessage().GetFileName())
		case hidratado.GetLocationMessage() != nil:
			cabecalho = "localizacao"
			titulo = primeiroTextoNaoVazio(titulo, hidratado.GetLocationMessage().GetName())
		}
		texto := strings.TrimSpace(hidratado.GetHydratedContentText())
		rodape := strings.TrimSpace(hidratado.GetHydratedFooterText())
		botoes := make([]map[string]interface{}, 0, len(hidratado.GetHydratedButtons()))
		var rotulos []string
		for _, b := range hidratado.GetHydratedButtons() {
			item := map[string]interface{}{}
			switch {
			case b.GetQuickReplyButton() != nil:
				item["tipo"], item["texto"], item["id"] = "resposta", b.GetQuickReplyButton().GetDisplayText(), b.GetQuickReplyButton().GetID()
			case b.GetUrlButton() != nil:
				item["tipo"], item["texto"], item["url"] = "url", b.GetUrlButton().GetDisplayText(), b.GetUrlButton().GetURL()
			case b.GetCallButton() != nil:
				item["tipo"], item["texto"], item["telefone"] = "ligar", b.GetCallButton().GetDisplayText(), b.GetCallButton().GetPhoneNumber()
			default:
				continue
			}
			if rotulo := strings.TrimSpace(item["texto"].(string)); rotulo != "" {
				rotulos = append(rotulos, "["+rotulo+"]")
			}
			botoes = append(botoes, item)
		}
		for _, p := range []string{titulo, texto, rodape} {
			if p != "" {
				partes = append(partes, p)
			}
		}
		if len(rotulos) > 0 {
			partes = append(partes, strings.Join(rotulos, " "))
		}
		dados = map[string]interface{}{
			"titulo":    titulo,
			"texto":     texto,
			"rodape":    rodape,
			"cabecalho": cabecalho,
			"botoes":    botoes,
			"modelo_id": strings.TrimSpace(hidratado.GetTemplateID()),
		}
	}
	conteudo := strings.Join(partes, "\n\n")
	if conteudo == "" {
		conteudo = "mensagem de modelo recebida"
	}
	extras := map[string]interface{}{"template": dados}
	return conteudo, "template", extras, map[string]interface{}{"template": dados}
}
