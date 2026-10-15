package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// WebhookMetaPendente e um webhook da Meta gravado e ainda nao processado.
type WebhookMetaPendente struct {
	ID          string
	InstanciaID string
	Corpo       []byte
	Tentativas  int
}

// MetaWebhookStore guarda os webhooks da Meta antes de responder 200: a Meta
// so reenvia o que nao confirmamos, entao o que ja confirmamos nao pode se
// perder se o container cair no meio do processamento.
type MetaWebhookStore interface {
	GuardarWebhookMeta(ctx context.Context, instanciaID string, corpo []byte) (string, error)
	ReservarWebhookMeta(ctx context.Context, id, dono string, expiracao time.Duration) (bool, error)
	ConcluirWebhookMeta(ctx context.Context, id, erro string) error
	ListarWebhooksMetaPendentes(ctx context.Context, recebidosAntes time.Time, limite int) ([]WebhookMetaPendente, error)
	LimparWebhooksMetaAntigos(ctx context.Context, antesDe time.Time) (int64, error)
}

func (s *SQLStore) GuardarWebhookMeta(ctx context.Context, instanciaID string, corpo []byte) (string, error) {
	id := uuid.NewString()
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO meta_webhooks (id, instancia_id, corpo, recebido_em) VALUES (?, ?, ?, ?)`),
		id, instanciaID, string(corpo), time.Now().UTC())
	if err != nil {
		return "", fmt.Errorf("erro ao guardar webhook da Meta: %w", err)
	}
	return id, nil
}

// ReservarWebhookMeta marca o webhook como em processamento por este
// container. Uma reserva antiga (container que caiu) expira e pode ser
// retomada; devolve false se outro ja esta processando ou ja concluiu.
func (s *SQLStore) ReservarWebhookMeta(ctx context.Context, id, dono string, expiracao time.Duration) (bool, error) {
	agora := time.Now().UTC()
	resultado, err := s.db.ExecContext(ctx, s.q(`
UPDATE meta_webhooks SET reservado_em = ?, dono = ?, tentativas = tentativas + 1
WHERE id = ? AND processado_em IS NULL AND (reservado_em IS NULL OR reservado_em < ?)`),
		agora, dono, id, agora.Add(-expiracao))
	if err != nil {
		return false, fmt.Errorf("erro ao reservar webhook da Meta: %w", err)
	}
	linhas, err := resultado.RowsAffected()
	return linhas > 0, err
}

func (s *SQLStore) ConcluirWebhookMeta(ctx context.Context, id, erro string) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE meta_webhooks SET processado_em = ?, erro = ? WHERE id = ?`), time.Now().UTC(), erro, id)
	if err != nil {
		return fmt.Errorf("erro ao concluir webhook da Meta: %w", err)
	}
	return nil
}

func (s *SQLStore) ListarWebhooksMetaPendentes(ctx context.Context, recebidosAntes time.Time, limite int) ([]WebhookMetaPendente, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`
SELECT id, instancia_id, corpo, tentativas FROM meta_webhooks
WHERE processado_em IS NULL AND recebido_em < ?
ORDER BY recebido_em LIMIT ?`), recebidosAntes.UTC(), limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar webhooks da Meta pendentes: %w", err)
	}
	defer linhas.Close()
	var pendentes []WebhookMetaPendente
	for linhas.Next() {
		var p WebhookMetaPendente
		var corpo string
		if err := linhas.Scan(&p.ID, &p.InstanciaID, &corpo, &p.Tentativas); err != nil {
			return nil, err
		}
		p.Corpo = []byte(corpo)
		pendentes = append(pendentes, p)
	}
	return pendentes, linhas.Err()
}

func (s *SQLStore) LimparWebhooksMetaAntigos(ctx context.Context, antesDe time.Time) (int64, error) {
	resultado, err := s.db.ExecContext(ctx, s.q(`DELETE FROM meta_webhooks WHERE processado_em IS NOT NULL AND recebido_em < ?`), antesDe.UTC())
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar webhooks da Meta: %w", err)
	}
	return resultado.RowsAffected()
}
