package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

type donoFalso struct{ outro string }

func (d donoFalso) PossuiRuntime(string) bool                             { return false }
func (d donoFalso) EnderecoOutroContainer(context.Context, string) string { return d.outro }

func TestProximaDataRecorrencia(t *testing.T) {
	base := time.Date(2026, 1, 31, 9, 0, 0, 0, time.UTC)
	agora := base.Add(time.Minute)
	if got := proximaData(base, "diaria", agora); !got.Equal(base.AddDate(0, 0, 1)) {
		t.Fatalf("diaria = %v", got)
	}
	if got := proximaData(base, "semanal", agora.AddDate(0, 0, 8)); !got.Equal(base.AddDate(0, 0, 14)) {
		t.Fatalf("semanal pula as que ja passaram = %v", got)
	}
	if got := proximaData(base, "mensal", agora); !got.Equal(base.AddDate(0, 1, 0)) {
		t.Fatalf("mensal = %v", got)
	}
	if got := proximaData(base, "", agora); !got.IsZero() {
		t.Fatalf("sem recorrencia = %v", got)
	}
}

func TestAgendamentoValidacaoEReplica(t *testing.T) {
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "ag.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	agora := time.Now()
	if _, err := st.Criar(ctx, models.Instancia{ID: "i", Nome: "Loja", Token: "t", Status: "conectada", ContaID: "c", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	s := NovoAgendamentoService(st, st, nil, donoFalso{outro: "http://outra:8080"}, nil, "r1")
	acesso := models.AcessoDashboard{Tipo: "usuario", ContaID: "c", Nome: "Ana"}
	for _, ruim := range []models.CriarAgendamentoRequest{
		{Instancia: "i", Numero: "5567", Mensagem: "oi"},                                    // sem data
		{Instancia: "i", Numero: "5567", Mensagem: "oi", EnviarEm: agora.Add(-time.Hour)},   // passado
		{Instancia: "i", Mensagem: "oi", EnviarEm: agora.Add(time.Hour)},                    // sem destino
		{Instancia: "i", Numero: "5567", Tipo: "documento", EnviarEm: agora.Add(time.Hour)}, // sem arquivo
		{Instancia: "i", Numero: "5567", Mensagem: "oi", EnviarEm: agora.Add(time.Hour), Recorrencia: "anual"},
	} {
		if _, err := s.Criar(ctx, acesso, ruim); err == nil {
			t.Fatalf("deveria recusar %+v", ruim)
		}
	}
	a, err := s.Criar(ctx, acesso, models.CriarAgendamentoRequest{Instancia: "i", Numero: "5567", ArquivoBase64: "data:image/png;base64,QUJD", MimeType: "image/png", EnviarEm: agora.Add(time.Second)})
	if err != nil || a.Tipo != "imagem" || a.ContaID != "c" || a.CriadoPor != "Ana" || a.ArquivoBase64 != "QUJD" {
		t.Fatalf("a = %+v err = %v", a, err)
	}
	// Outra replica detem a sessao: esta nao reserva nem envia.
	s.agora = func() time.Time { return agora.Add(time.Minute) }
	s.Processar(ctx)
	if salvo, _ := st.BuscarAgendamento(ctx, a.ID); salvo.Status != models.AgendamentoPendente {
		t.Fatalf("status = %s", salvo.Status)
	}
	if lista, _ := s.Listar(ctx, models.AcessoDashboard{Tipo: "usuario", ContaID: "outra"}, models.FiltroAgendamentos{}); len(lista) != 0 {
		t.Fatal("outra conta nao pode ver")
	}
	if _, err := s.Cancelar(ctx, models.AcessoDashboard{Tipo: "usuario", ContaID: "outra"}, a.ID); err == nil {
		t.Fatal("outra conta nao pode cancelar")
	}
	if _, err := s.Cancelar(ctx, acesso, a.ID); err != nil {
		t.Fatal(err)
	}
}
