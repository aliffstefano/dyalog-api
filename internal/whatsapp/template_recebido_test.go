package whatsapp

import (
	"strings"
	"testing"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestTemplateRecebidoMostraTexto(t *testing.T) {
	hidratado := &waE2E.TemplateMessage_HydratedFourRowTemplate{
		HydratedContentText: proto.String("Ola Aliff, seu pedido 123 saiu para entrega."),
		HydratedFooterText:  proto.String("Loja Exemplo"),
		Title:               &waE2E.TemplateMessage_HydratedFourRowTemplate_HydratedTitleText{HydratedTitleText: "Pedido enviado"},
		HydratedButtons: []*waE2E.HydratedTemplateButton{
			{HydratedButton: &waE2E.HydratedTemplateButton_QuickReplyButton{QuickReplyButton: &waE2E.HydratedTemplateButton_HydratedQuickReplyButton{DisplayText: proto.String("Recebi"), ID: proto.String("ok")}}},
			{HydratedButton: &waE2E.HydratedTemplateButton_UrlButton{UrlButton: &waE2E.HydratedTemplateButton_HydratedURLButton{DisplayText: proto.String("Rastrear"), URL: proto.String("https://exemplo.com/r/123")}}},
		},
	}
	// Formato da Cloud API: HydratedTemplate (e as vezes tambem Format).
	msg := &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{HydratedTemplate: hidratado}}
	conteudo, tipo, extras, _ := extrairConteudoMensagem(msg)
	if tipo != "template" {
		t.Fatalf("tipo = %q", tipo)
	}
	for _, parte := range []string{"Pedido enviado", "seu pedido 123", "Loja Exemplo", "[Recebi]", "[Rastrear]"} {
		if !strings.Contains(conteudo, parte) {
			t.Fatalf("conteudo sem %q: %q", parte, conteudo)
		}
	}
	dados := extras["template"].(map[string]interface{})
	if len(dados["botoes"].([]map[string]interface{})) != 2 {
		t.Fatalf("botoes = %v", dados["botoes"])
	}
	// So no oneof Format, sem HydratedTemplate.
	msg = &waE2E.Message{TemplateMessage: &waE2E.TemplateMessage{Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{HydratedFourRowTemplate: hidratado}}}
	if conteudo, _, _, _ := extrairConteudoMensagem(msg); !strings.Contains(conteudo, "saiu para entrega") {
		t.Fatalf("format: %q", conteudo)
	}
}
