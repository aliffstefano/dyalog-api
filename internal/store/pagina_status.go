package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"dyalog-api-go/internal/models"
)

// PaginaStatusStore guarda a pagina publica de status de cada conta.
type PaginaStatusStore interface {
	PaginaStatusDaConta(ctx context.Context, contaID string) (models.PaginaStatus, error)
	PaginaStatusPorToken(ctx context.Context, token string) (models.PaginaStatus, error)
	SalvarPaginaStatus(ctx context.Context, p models.PaginaStatus) error
}

func (s *SQLStore) lerPaginaStatus(linha *sql.Row) (models.PaginaStatus, error) {
	var (
		p       models.PaginaStatus
		ativa   any
		ocultas string
	)
	if err := linha.Scan(&p.ContaID, &p.Token, &p.Titulo, &ativa, &ocultas); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNaoEncontrado
		}
		return p, err
	}
	p.Ativa = s.boolFromDB(ativa)
	p.InstanciasOcultas = []string{}
	for _, id := range strings.Split(ocultas, ",") {
		if id = strings.TrimSpace(id); id != "" {
			p.InstanciasOcultas = append(p.InstanciasOcultas, id)
		}
	}
	return p, nil
}

func (s *SQLStore) PaginaStatusDaConta(ctx context.Context, contaID string) (models.PaginaStatus, error) {
	return s.lerPaginaStatus(s.db.QueryRowContext(ctx, s.q(`SELECT conta_id, token, titulo, ativa, instancias_ocultas FROM paginas_status WHERE conta_id = ?`), contaID))
}

func (s *SQLStore) PaginaStatusPorToken(ctx context.Context, token string) (models.PaginaStatus, error) {
	return s.lerPaginaStatus(s.db.QueryRowContext(ctx, s.q(`SELECT conta_id, token, titulo, ativa, instancias_ocultas FROM paginas_status WHERE token = ?`), token))
}

func (s *SQLStore) SalvarPaginaStatus(ctx context.Context, p models.PaginaStatus) error {
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO paginas_status (conta_id, token, titulo, ativa, instancias_ocultas) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(conta_id) DO UPDATE SET token = excluded.token, titulo = excluded.titulo, ativa = excluded.ativa, instancias_ocultas = excluded.instancias_ocultas`),
		p.ContaID, p.Token, p.Titulo, s.boolDB(p.Ativa), strings.Join(p.InstanciasOcultas, ","))
	if err != nil {
		return fmt.Errorf("erro ao salvar pagina de status: %w", err)
	}
	return nil
}
