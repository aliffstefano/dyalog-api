package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestRespostasEEtiquetas(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "rr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := s.Criar(ctx, models.Instancia{ID: "i", Nome: "Loja", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	r, err := s.SalvarRespostaRapida(ctx, models.RespostaRapida{ContaID: "c", Atalho: "oi", Texto: "Ola, {nome}!"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SalvarRespostaRapida(ctx, models.RespostaRapida{ContaID: "c", Atalho: "oi", Texto: "x"}); !errors.Is(err, ErrDuplicado) {
		t.Fatalf("atalho repetido: %v", err)
	}
	if _, err := s.SalvarRespostaRapida(ctx, models.RespostaRapida{ContaID: "outra", Atalho: "oi", Texto: "x"}); err != nil {
		t.Fatalf("outra conta pode usar o mesmo atalho: %v", err)
	}
	r.Texto = "Bom dia"
	if _, err := s.SalvarRespostaRapida(ctx, models.RespostaRapida{ID: r.ID, ContaID: "outra", Atalho: "oi", Texto: "invasao"}); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("outra conta nao pode editar: %v", err)
	}
	if _, err := s.SalvarRespostaRapida(ctx, r); err != nil {
		t.Fatal(err)
	}
	if lista, _ := s.ListarRespostasRapidas(ctx, "c"); len(lista) != 1 || lista[0].Texto != "Bom dia" {
		t.Fatalf("lista = %+v", lista)
	}
	e1, _ := s.SalvarEtiqueta(ctx, models.Etiqueta{ContaID: "c", Nome: "Novo cliente", Cor: "#22c55e"})
	e2, _ := s.SalvarEtiqueta(ctx, models.Etiqueta{ContaID: "c", Nome: "Pagamento", Cor: "#f59e0b"})
	if err := s.DefinirEtiquetasChat(ctx, "i", "5567@s.whatsapp.net", []string{e1.ID, e2.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.DefinirEtiquetasChat(ctx, "i", "5567@s.whatsapp.net", []string{e2.ID}); err != nil {
		t.Fatal(err)
	}
	if mapa, _ := s.EtiquetasDoChat(ctx, "i"); len(mapa["5567@s.whatsapp.net"]) != 1 || mapa["5567@s.whatsapp.net"][0] != e2.ID {
		t.Fatalf("mapa = %+v", mapa)
	}
	if err := s.ExcluirEtiqueta(ctx, "c", e2.ID); err != nil {
		t.Fatal(err)
	}
	if mapa, _ := s.EtiquetasDoChat(ctx, "i"); len(mapa) != 0 {
		t.Fatalf("etiqueta excluida continua na conversa: %+v", mapa)
	}
	if err := s.ExcluirRespostaRapida(ctx, "outra", r.ID); !errors.Is(err, ErrNaoEncontrado) {
		t.Fatalf("outra conta nao pode excluir: %v", err)
	}
}
