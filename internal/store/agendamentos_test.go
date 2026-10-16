package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestAgendamentosFila(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "ag.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now().UTC().Truncate(time.Second)
	if _, err := s.Criar(ctx, models.Instancia{ID: "i", Nome: "Loja", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	vencido, _ := s.CriarAgendamento(ctx, models.Agendamento{InstanciaID: "i", ContaID: "c", Tipo: "texto", Numero: "5567", Mensagem: "oi", EnviarEm: agora.Add(-time.Minute)})
	futuro, _ := s.CriarAgendamento(ctx, models.Agendamento{InstanciaID: "i", ContaID: "c", Tipo: "documento", Numero: "5567", ArquivoBase64: "QUJD", EnviarEm: agora.Add(time.Hour)})

	fila, err := s.AgendamentosVencidos(ctx, agora, 10)
	if err != nil || len(fila) != 1 || fila[0].ID != vencido.ID || fila[0].InstanciaNome != "Loja" {
		t.Fatalf("fila = %+v %v", fila, err)
	}
	if ok, _ := s.ReservarAgendamento(ctx, vencido.ID, "r1", agora); !ok {
		t.Fatal("deveria reservar")
	}
	if ok, _ := s.ReservarAgendamento(ctx, vencido.ID, "r2", agora); ok {
		t.Fatal("segunda replica nao pode reservar o mesmo")
	}
	if n, _ := s.LiberarAgendamentosPresos(ctx, agora.Add(time.Second)); n != 1 {
		t.Fatalf("liberados = %d", n)
	}
	if err := s.AdiarAgendamento(ctx, vencido.ID, "desconectada", 1, agora.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if fila, _ := s.AgendamentosVencidos(ctx, agora, 10); len(fila) != 0 {
		t.Fatalf("adiado nao deveria estar na fila: %+v", fila)
	}
	if err := s.ConcluirAgendamento(ctx, futuro.ID, models.AgendamentoEnviado, "", "MID", 1, agora); err != nil {
		t.Fatal(err)
	}
	enviado, _ := s.BuscarAgendamento(ctx, futuro.ID)
	if enviado.Status != "enviado" || enviado.ArquivoBase64 != "" || enviado.MensagemID != "MID" || enviado.EnviadoEm.IsZero() {
		t.Fatalf("enviado = %+v", enviado)
	}
	if err := s.CancelarAgendamento(ctx, futuro.ID); !errors.Is(err, ErrAgendamentoNaoEncontrado) {
		t.Fatalf("cancelar enviado deveria falhar: %v", err)
	}
	novoTexto := "ola"
	if err := s.AtualizarAgendamentoPendente(ctx, vencido.ID, &novoTexto, nil, nil); err != nil {
		t.Fatal(err)
	}
	lista, _ := s.ListarAgendamentos(ctx, models.FiltroAgendamentos{SoContas: true, ContasIDs: []string{"c"}})
	if len(lista) != 2 || lista[0].ID != vencido.ID || lista[0].Mensagem != "ola" {
		t.Fatalf("lista = %+v", lista)
	}
	if err := s.CancelarAgendamento(ctx, vencido.ID); err != nil {
		t.Fatal(err)
	}
}
