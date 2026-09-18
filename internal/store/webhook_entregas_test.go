package store

import (
	"context"
	"testing"
	"time"

	"dyalog-api-go/internal/models"

	"github.com/google/uuid"
)

func novoStoreTeste(t *testing.T) *SQLStore {
	t.Helper()
	s, err := NovoSQLStore("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("erro ao criar store de teste: %v", err)
	}
	t.Cleanup(func() { _ = s.db.Close() })
	return s
}

func inserirEntregaTeste(t *testing.T, s *SQLStore, status string, atualizadoEm time.Time) string {
	t.Helper()
	ctx := context.Background()
	agora := atualizadoEm
	instanciaID := uuid.NewString()
	// webhook_entregas.instancia_id tem FOREIGN KEY para instancias(id); precisamos
	// de uma instancia existente para o insert nao falhar.
	_, err := s.Criar(ctx, models.Instancia{
		ID:           instanciaID,
		Nome:         "instancia-teste",
		Token:        uuid.NewString(),
		Status:       "desconectada",
		CriadoEm:     agora,
		AtualizadoEm: agora,
	})
	if err != nil {
		t.Fatalf("erro ao criar instancia de teste: %v", err)
	}
	entrega := models.WebhookEntrega{
		ID:                 uuid.NewString(),
		WebhookID:          uuid.NewString(),
		InstanciaID:        instanciaID,
		URL:                "https://exemplo.com/webhook",
		Evento:             "mensagens",
		Payload:            []byte(`{"teste":true}`),
		Status:             models.WebhookEntregaPendente,
		MaxTentativas:      5,
		ProximaTentativaEm: agora,
		CriadoEm:           agora,
		AtualizadoEm:       agora,
	}
	if _, err := s.EnfileirarWebhookEntrega(ctx, entrega); err != nil {
		t.Fatalf("erro ao enfileirar entrega de teste: %v", err)
	}
	// EnfileirarWebhookEntrega sempre grava com status/atualizado_em do momento da
	// insercao; forcamos aqui o status e o atualizado_em desejados para simular
	// entregas antigas/finalizadas sem depender do fluxo real de retry.
	if _, err := s.db.ExecContext(ctx, s.q(`UPDATE webhook_entregas SET status = ?, atualizado_em = ? WHERE id = ?`), status, agora.UTC(), entrega.ID); err != nil {
		t.Fatalf("erro ao ajustar entrega de teste: %v", err)
	}
	return entrega.ID
}

func contarEntregas(t *testing.T, s *SQLStore) int {
	t.Helper()
	var total int
	if err := s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM webhook_entregas`).Scan(&total); err != nil {
		t.Fatalf("erro ao contar entregas: %v", err)
	}
	return total
}

func TestLimparWebhookEntregasAntigasApagaFinalizadasAntigas(t *testing.T) {
	s := novoStoreTeste(t)
	antiga := time.Now().UTC().Add(-40 * 24 * time.Hour)

	inserirEntregaTeste(t, s, models.WebhookEntregaEntregue, antiga)
	inserirEntregaTeste(t, s, models.WebhookEntregaEsgotada, antiga)

	apagadas, err := s.LimparWebhookEntregasAntigas(context.Background(), time.Now().UTC().Add(-30*24*time.Hour))
	if err != nil {
		t.Fatalf("erro ao limpar entregas: %v", err)
	}
	if apagadas != 2 {
		t.Fatalf("apagadas = %d, esperado 2", apagadas)
	}
	if total := contarEntregas(t, s); total != 0 {
		t.Fatalf("total apos limpeza = %d, esperado 0", total)
	}
}

func TestLimparWebhookEntregasAntigasPreservaRecentesEEmAndamento(t *testing.T) {
	s := novoStoreTeste(t)
	antiga := time.Now().UTC().Add(-40 * 24 * time.Hour)
	recente := time.Now().UTC().Add(-1 * time.Hour)
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)

	// Finalizada mas recente: deve sobreviver.
	inserirEntregaTeste(t, s, models.WebhookEntregaEntregue, recente)
	// Antiga mas ainda pendente/em retry/enviando: nunca deve ser apagada por essa
	// rotina, mesmo que velha, pois a fila ainda pode reprocessa-la.
	inserirEntregaTeste(t, s, models.WebhookEntregaPendente, antiga)
	inserirEntregaTeste(t, s, models.WebhookEntregaFalha, antiga)
	inserirEntregaTeste(t, s, models.WebhookEntregaEnviando, antiga)

	apagadas, err := s.LimparWebhookEntregasAntigas(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("erro ao limpar entregas: %v", err)
	}
	if apagadas != 0 {
		t.Fatalf("apagadas = %d, esperado 0", apagadas)
	}
	if total := contarEntregas(t, s); total != 4 {
		t.Fatalf("total apos limpeza = %d, esperado 4 (nenhuma deveria ser removida)", total)
	}
}
