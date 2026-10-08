package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

func TestChatRegistraEventosDaMetaEDoQR(t *testing.T) {
	s, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := s.Criar(ctx, models.Instancia{ID: "i", Nome: "I", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	chat := NovoChatService(s)

	// Recebida pela API oficial, enviada pela API e recibo de falha da Meta.
	var valor meta.Valor
	_ = json.Unmarshal([]byte(`{"contacts":[{"profile":{"name":"Aliff"},"wa_id":"556799440667"}],
		"messages":[{"from":"556799440667","id":"wamid.IN","timestamp":"1791396000","type":"text","text":{"body":"oi"}}]}`), &valor)
	chat.RegistrarEvento(ctx, "i", models.EventoWebhookMensagens, meta.DadosMensagem(valor, valor.Mensagens[0]))
	chat.store.RegistrarChatMensagem(ctx, models.ChatMensagem{InstanciaID: "i", ChatJID: jidConversa("556799440667@s.whatsapp.net", ""), MensagemID: "wamid.OUT", Direcao: "saida", Tipo: "texto", Conteudo: "ola", Status: "enviada"})
	var status meta.Status
	_ = json.Unmarshal([]byte(`{"id":"wamid.OUT","status":"failed","recipient_id":"556799440667","errors":[{"code":130497,"title":"Business account is restricted"}]}`), &status)
	chat.RegistrarEvento(ctx, "i", models.EventoWebhookRecibos, meta.DadosRecibo(status))

	// QR code: chega como @lid, mas com o numero resolvido vai para a mesma conversa.
	chat.RegistrarEvento(ctx, "i", models.EventoWebhookMensagens, map[string]any{
		"mensagem_id": "QR1", "chat_jid": "123456@lid", "chat_numero": "556799440667", "direcao": "entrada",
		"acao": "recebida", "tipo": "texto", "conteudo": "pelo qr", "recebida_em": time.Now().UTC(),
	})
	// Historico e status nao entram.
	chat.RegistrarEvento(ctx, "i", models.EventoWebhookMensagens, map[string]any{"mensagem_id": "H", "chat_jid": "1@s.whatsapp.net", "historico": true, "acao": "recebida"})
	chat.RegistrarEvento(ctx, "i", models.EventoWebhookMensagens, map[string]any{"mensagem_id": "S", "chat_jid": "status@broadcast", "acao": "recebida"})

	conversas, _ := chat.ListarConversas(ctx, "i")
	if len(conversas) != 1 || conversas[0].ChatJID != "556799440667@s.whatsapp.net" || conversas[0].Nome != "Aliff" {
		t.Fatalf("conversas = %+v", conversas)
	}
	mensagens, _ := chat.ListarMensagens(ctx, "i", conversas[0].ChatJID)
	if len(mensagens) != 3 {
		t.Fatalf("mensagens = %+v", mensagens)
	}
	for _, m := range mensagens {
		if m.MensagemID == "wamid.OUT" && (m.Status != "falhou" || m.Erro != "130497: Business account is restricted") {
			t.Fatalf("recibo de falha nao aplicado: %+v", m)
		}
	}
}
