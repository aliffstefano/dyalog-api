package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestChatConversasMensagensEStatus(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "chat.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := s.Criar(ctx, models.Instancia{ID: "i", Nome: "I", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	a, b := "5567999990001@s.whatsapp.net", "5567999990002@s.whatsapp.net"
	for _, m := range []models.ChatMensagem{
		{InstanciaID: "i", ChatJID: a, MensagemID: "a1", Direcao: "entrada", Tipo: "texto", Conteudo: "oi", Nome: "Ana", CriadoEm: agora.Add(-3 * time.Minute)},
		{InstanciaID: "i", ChatJID: b, MensagemID: "b1", Direcao: "entrada", Tipo: "texto", Conteudo: "ola", Nome: "Bia", CriadoEm: agora.Add(-2 * time.Minute)},
		{InstanciaID: "i", ChatJID: a, MensagemID: "a2", Direcao: "saida", Tipo: "texto", Conteudo: "tudo bem?", Status: "enviada", CriadoEm: agora.Add(-time.Minute)},
		{InstanciaID: "i", ChatJID: a, MensagemID: "a2", Direcao: "saida", Tipo: "texto", Conteudo: "duplicada", CriadoEm: agora},
	} {
		if err := s.RegistrarChatMensagem(ctx, m); err != nil {
			t.Fatal(err)
		}
	}

	conversas, err := s.ListarConversasChat(ctx, "i", 10)
	if err != nil || len(conversas) != 2 || conversas[0].ChatJID != a || conversas[0].Nome != "Ana" || conversas[0].Ultima.MensagemID != "a2" {
		t.Fatalf("conversas = %+v, err = %v", conversas, err)
	}

	_ = s.AtualizarStatusChat(ctx, "i", []string{"a2"}, "lida", "")
	_ = s.AtualizarStatusChat(ctx, "i", []string{"a2"}, "entregue", "") // atrasado: nao volta
	_ = s.AtualizarStatusChat(ctx, "i", []string{"a1"}, "lida", "")     // recebida: nao muda
	_ = s.EditarChatMensagem(ctx, "i", "a1", "", "oi, editado")
	mensagens, err := s.ListarMensagensChat(ctx, "i", a, 50)
	if err != nil || len(mensagens) != 2 {
		t.Fatalf("mensagens = %+v, err = %v", mensagens, err)
	}
	if mensagens[0].Conteudo != "oi, editado" || mensagens[0].Status != "" || mensagens[1].Conteudo != "tudo bem?" || mensagens[1].Status != "lida" {
		t.Fatalf("mensagens = %+v", mensagens)
	}

	if apagadas, err := s.LimparChatAntigo(ctx, agora.Add(-90*time.Second)); err != nil || apagadas != 2 {
		t.Fatalf("apagadas = %d, err = %v", apagadas, err)
	}
}
