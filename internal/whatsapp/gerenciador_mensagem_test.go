package whatsapp

import (
	"testing"

	waCommon "go.mau.fi/whatsmeow/proto/waCommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestExtrairConteudoMensagemComCitacao(t *testing.T) {
	msg := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text: proto.String("Sim, confirmado"),
		ContextInfo: &waE2E.ContextInfo{
			StanzaID:    proto.String("msg-citada-id"),
			Participant: proto.String("5511999999999@s.whatsapp.net"),
			QuotedMessage: &waE2E.Message{
				Conversation: proto.String("Voce confirma o pedido?"),
			},
		},
	}}

	conteudo, tipo, extras, extrasMensagem := extrairConteudoMensagem(msg)

	if conteudo != "Sim, confirmado" {
		t.Fatalf("conteudo = %q, esperado 'Sim, confirmado'", conteudo)
	}
	if tipo != "texto" {
		t.Fatalf("tipo = %q, esperado texto", tipo)
	}
	if extras["resposta_mensagem_id"] != "msg-citada-id" {
		t.Fatalf("resposta_mensagem_id = %v, esperado msg-citada-id", extras["resposta_mensagem_id"])
	}
	if extras["resposta_participante"] != "5511999999999@s.whatsapp.net" {
		t.Fatalf("resposta_participante = %v, esperado 5511999999999@s.whatsapp.net", extras["resposta_participante"])
	}
	if extras["resposta_conteudo"] != "Voce confirma o pedido?" {
		t.Fatalf("resposta_conteudo = %v, esperado 'Voce confirma o pedido?'", extras["resposta_conteudo"])
	}
	if extras["resposta_tipo"] != "texto" {
		t.Fatalf("resposta_tipo = %v, esperado texto", extras["resposta_tipo"])
	}
	resposta, ok := extrasMensagem["resposta"].(map[string]interface{})
	if !ok {
		t.Fatalf("extrasMensagem[resposta] ausente: %#v", extrasMensagem)
	}
	if resposta["mensagem_id"] != "msg-citada-id" {
		t.Fatalf("resposta.mensagem_id = %v, esperado msg-citada-id", resposta["mensagem_id"])
	}
}

func TestExtrairConteudoMensagemSemCitacao(t *testing.T) {
	msg := &waE2E.Message{Conversation: proto.String("mensagem normal")}

	_, _, extras, extrasMensagem := extrairConteudoMensagem(msg)

	if _, ok := extras["resposta_mensagem_id"]; ok {
		t.Fatalf("nao deveria haver resposta_mensagem_id: %#v", extras)
	}
	if _, ok := extrasMensagem["resposta"]; ok {
		t.Fatalf("nao deveria haver extrasMensagem[resposta]: %#v", extrasMensagem)
	}
}

func TestExtrairConteudoMensagemReacao(t *testing.T) {
	msg := &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{
		Text: proto.String("ok"),
		Key: &waCommon.MessageKey{
			ID:          proto.String("msg-original"),
			RemoteJID:   proto.String("5511999999999@s.whatsapp.net"),
			FromMe:      proto.Bool(false),
			Participant: proto.String("5511888888888@s.whatsapp.net"),
		},
		SenderTimestampMS: proto.Int64(1234),
	}}

	conteudo, tipo, extras, extrasMensagem := extrairConteudoMensagem(msg)

	if conteudo != "ok" {
		t.Fatalf("conteudo = %q, esperado ok", conteudo)
	}
	if tipo != "reacao" {
		t.Fatalf("tipo = %q, esperado reacao", tipo)
	}
	if extras["mensagem_reagida_id"] != "msg-original" {
		t.Fatalf("mensagem_reagida_id = %v, esperado msg-original", extras["mensagem_reagida_id"])
	}
	reacao, ok := extrasMensagem["reacao"].(map[string]interface{})
	if !ok {
		t.Fatalf("extrasMensagem[reacao] ausente: %#v", extrasMensagem)
	}
	if reacao["mensagem_id"] != "msg-original" {
		t.Fatalf("reacao.mensagem_id = %v, esperado msg-original", reacao["mensagem_id"])
	}
}

func TestExtrairConteudoMensagemEnquete(t *testing.T) {
	msg := &waE2E.Message{PollCreationMessageV3: &waE2E.PollCreationMessage{
		Name:                   proto.String("Qual plano?"),
		SelectableOptionsCount: proto.Uint32(1),
		Options: []*waE2E.PollCreationMessage_Option{
			{OptionName: proto.String("Basico")},
			{OptionName: proto.String("Pro")},
		},
	}}

	conteudo, tipo, extras, extrasMensagem := extrairConteudoMensagem(msg)

	if conteudo != "Qual plano?" {
		t.Fatalf("conteudo = %q, esperado Qual plano?", conteudo)
	}
	if tipo != "enquete" {
		t.Fatalf("tipo = %q, esperado enquete", tipo)
	}
	if extras["enquete_quantidade_opcoes"] != 2 {
		t.Fatalf("enquete_quantidade_opcoes = %v, esperado 2", extras["enquete_quantidade_opcoes"])
	}
	enquete, ok := extrasMensagem["enquete"].(map[string]interface{})
	if !ok {
		t.Fatalf("extrasMensagem[enquete] ausente: %#v", extrasMensagem)
	}
	if enquete["nome"] != "Qual plano?" {
		t.Fatalf("enquete.nome = %v, esperado Qual plano?", enquete["nome"])
	}
}

func TestExtrairConteudoMensagemDocumentWithCaption(t *testing.T) {
	msg := &waE2E.Message{DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
		Message: &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
			Caption:  proto.String("Contrato"),
			FileName: proto.String("contrato.pdf"),
			Mimetype: proto.String("application/pdf"),
		}},
	}}

	conteudo, tipo, extras, _ := extrairConteudoMensagem(msg)

	if conteudo != "Contrato" {
		t.Fatalf("conteudo = %q, esperado Contrato", conteudo)
	}
	if tipo != "documento" {
		t.Fatalf("tipo = %q, esperado documento", tipo)
	}
	if extras["nome_arquivo"] != "contrato.pdf" {
		t.Fatalf("nome_arquivo = %v, esperado contrato.pdf", extras["nome_arquivo"])
	}
}

func TestExtrairConteudoMensagemPTV(t *testing.T) {
	msg := &waE2E.Message{PtvMessage: &waE2E.VideoMessage{
		Caption:  proto.String("Video circular"),
		Mimetype: proto.String("video/mp4"),
	}}

	conteudo, tipo, extras, _ := extrairConteudoMensagem(msg)

	if conteudo != "Video circular" {
		t.Fatalf("conteudo = %q, esperado Video circular", conteudo)
	}
	if tipo != "video" {
		t.Fatalf("tipo = %q, esperado video", tipo)
	}
	if extras["ptv"] != true {
		t.Fatalf("ptv = %v, esperado true", extras["ptv"])
	}
}

func TestExtrairConteudoMensagemTipoNaoMapeado(t *testing.T) {
	msg := &waE2E.Message{EncReactionMessage: &waE2E.EncReactionMessage{}}

	conteudo, tipo, extras, _ := extrairConteudoMensagem(msg)

	if conteudo != "mensagem enc_reaction recebida" {
		t.Fatalf("conteudo = %q, esperado fallback identificado", conteudo)
	}
	if tipo != "enc_reaction" {
		t.Fatalf("tipo = %q, esperado enc_reaction", tipo)
	}
	if extras["tipo_original"] != "encReactionMessage" {
		t.Fatalf("tipo_original = %v, esperado encReactionMessage", extras["tipo_original"])
	}
}
