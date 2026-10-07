package meta

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

const webhookExemplo = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "WABA",
    "changes": [{
      "field": "messages",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "15550001111", "phone_number_id": "123"},
        "contacts": [{"profile": {"name": "Kerry"}, "wa_id": "5567999440667"}],
        "messages": [
          {"from": "5567999440667", "id": "wamid.T", "timestamp": "1791396000", "type": "text", "text": {"body": "Oi"}},
          {"from": "5567999440667", "id": "wamid.L", "timestamp": "1791396001", "type": "interactive",
           "context": {"from": "15550001111", "id": "wamid.MENU"},
           "interactive": {"type": "list_reply", "list_reply": {"id": "p1", "title": "Lasanha", "description": "Bolonhesa"}}},
          {"from": "5567999440667", "id": "wamid.B", "timestamp": "1791396002", "type": "interactive",
           "interactive": {"type": "button_reply", "button_reply": {"id": "sim", "title": "Sim"}}},
          {"from": "5567999440667", "id": "wamid.I", "timestamp": "1791396003", "type": "image",
           "image": {"id": "MID", "mime_type": "image/jpeg", "caption": "foto"}}
        ],
        "statuses": [{"id": "wamid.ENV", "status": "read", "timestamp": "1791396004", "recipient_id": "5567999440667"}]
      }
    }]
  }]
}`

func TestDadosMensagemNoFormatoDasInstanciasQR(t *testing.T) {
	var w Webhook
	if err := json.Unmarshal([]byte(webhookExemplo), &w); err != nil {
		t.Fatal(err)
	}
	valor := w.Entradas[0].Mudancas[0].Valor
	if len(valor.Mensagens) != 4 || len(valor.Status) != 1 {
		t.Fatalf("mensagens=%d status=%d", len(valor.Mensagens), len(valor.Status))
	}

	texto := DadosMensagem(valor, valor.Mensagens[0])
	if texto["tipo"] != "texto" || texto["conteudo"] != "Oi" || texto["nome_remetente"] != "Kerry" ||
		texto["chat_jid"] != "5567999440667@s.whatsapp.net" || texto["direcao"] != "entrada" {
		t.Fatalf("texto: %v", texto)
	}

	lista := DadosMensagem(valor, valor.Mensagens[1])
	if lista["tipo"] != "lista" || lista["lista_id"] != "p1" || lista["resposta_mensagem_id"] != "wamid.MENU" {
		t.Fatalf("lista: %v", lista)
	}

	botao := DadosMensagem(valor, valor.Mensagens[2])
	if botao["tipo"] != "botao" || botao["botao_id"] != "sim" || botao["conteudo"] != "Sim" {
		t.Fatalf("botao: %v", botao)
	}

	midia, tipo := valor.Mensagens[3].MidiaDe()
	if midia == nil || tipo != "imagem" || midia.ID != "MID" {
		t.Fatalf("midia: %v %s", midia, tipo)
	}
	if imagem := DadosMensagem(valor, valor.Mensagens[3]); imagem["conteudo"] != "foto" || imagem["mime_type"] != "image/jpeg" {
		t.Fatalf("imagem: %v", imagem)
	}

	recibo := DadosRecibo(valor.Status[0])
	if recibo["status"] != "lida" || recibo["mensagem_id"] != "wamid.ENV" {
		t.Fatalf("recibo: %v", recibo)
	}
}

func TestAssinaturaValida(t *testing.T) {
	corpo := []byte(webhookExemplo)
	mac := hmac.New(sha256.New, []byte("segredo"))
	mac.Write(corpo)
	assinatura := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !AssinaturaValida(corpo, assinatura, "segredo") {
		t.Fatal("assinatura correta recusada")
	}
	if AssinaturaValida(corpo, assinatura, "outro") || AssinaturaValida(corpo, "", "segredo") || AssinaturaValida(append(corpo, ' '), assinatura, "segredo") {
		t.Fatal("assinatura errada aceita")
	}
}
