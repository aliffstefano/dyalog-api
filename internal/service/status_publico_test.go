package service

import (
	"context"
	"math"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

func TestCalcularDisponibilidade(t *testing.T) {
	loc := time.UTC
	inicio := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	agora := time.Date(2026, 10, 3, 12, 0, 0, 0, loc) // 2,5 dias
	eventos := []eventoConexao{
		{time.Date(2026, 10, 2, 6, 0, 0, 0, loc), false}, // caiu as 6h do dia 2
		{time.Date(2026, 10, 2, 12, 0, 0, 0, loc), true}, // voltou ao meio-dia
	}
	dias, total := calcularDisponibilidade(eventos, true, time.Time{}, inicio, agora, 3)
	if dias[0] != 1 || math.Abs(dias[1]-0.75) > 1e-9 || dias[2] != 1 {
		t.Fatalf("dias = %v", dias)
	}
	if esperado := (60.0 - 6) / 60; math.Abs(total-esperado) > 1e-9 {
		t.Fatalf("total = %v, esperado %v", total, esperado)
	}
	// Criada no meio do dia 2: dia 1 sem dados, mede a partir da criacao.
	criada := time.Date(2026, 10, 2, 18, 0, 0, 0, loc)
	dias, _ = calcularDisponibilidade(nil, false, criada, inicio, agora, 3)
	if dias[0] != -1 || dias[1] != 0 || dias[2] != 0 {
		t.Fatalf("criada depois = %v", dias)
	}
}

func TestStatusPublicoPorLink(t *testing.T) {
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "st.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	agora := time.Now()
	for _, inst := range []models.Instancia{
		{ID: "a", Nome: "Vendas", Token: "ta", Status: "conectada", ContaID: "c"},
		{ID: "b", Nome: "Suporte", Token: "tb", Status: "desconectada", ContaID: "c"},
		{ID: "x", Nome: "Outra conta", Token: "tx", Status: "conectada", ContaID: "outra"},
	} {
		inst.CriadoEm, inst.AtualizadoEm = agora.Add(-48*time.Hour), agora
		if _, err := st.Criar(ctx, inst); err != nil {
			t.Fatal(err)
		}
		// Criar ignora o status inicial; ajusta.
		if _, err := st.AtualizarStatus(ctx, inst.ID, inst.Status); err != nil {
			t.Fatal(err)
		}
	}
	s := NovoStatusPublicoService(st, st, st, "https://painel.exemplo.com")
	if _, err := s.Publico(ctx, "st_inexistente"); err != ErrStatusPublicoNaoEncontrado {
		t.Fatalf("link invalido: %v", err)
	}
	p, err := s.Salvar(ctx, "c", "Status da Loja", true, nil, false)
	if err != nil || p.Link != "https://painel.exemplo.com/status/"+p.Token {
		t.Fatalf("p = %+v err = %v", p, err)
	}
	pub, err := s.Publico(ctx, p.Token)
	if err != nil || len(pub.Instancias) != 2 || pub.Geral != "parcial" || pub.Instancias[0].Nome != "Suporte" || pub.Instancias[0].Estado != "fora" {
		t.Fatalf("pub = %+v err = %v", pub, err)
	}
	if _, err := s.Salvar(ctx, "c", "Status da Loja", true, []string{"b"}, false); err != nil {
		t.Fatal(err)
	}
	if pub, _ := s.Publico(ctx, p.Token); len(pub.Instancias) != 1 || pub.Geral != "operando" {
		t.Fatalf("oculta = %+v", pub)
	}
	novo, _ := s.Salvar(ctx, "c", "Status da Loja", true, nil, true)
	if _, err := s.Publico(ctx, p.Token); err != ErrStatusPublicoNaoEncontrado || novo.Token == p.Token {
		t.Fatal("link antigo deveria parar de funcionar")
	}
	if _, err := s.Salvar(ctx, "c", "Status da Loja", false, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Publico(ctx, novo.Token); err != ErrStatusPublicoNaoEncontrado {
		t.Fatal("pagina desligada nao deveria abrir")
	}
}
