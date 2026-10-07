package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
)

// ErrCredenciaisMetaNaoEncontradas indica instancia da API oficial ainda sem
// credenciais cadastradas.
var ErrCredenciaisMetaNaoEncontradas = errors.New("credenciais da Meta nao cadastradas")

// MetaStore guarda as credenciais das instancias da API oficial (Cloud API).
type MetaStore interface {
	ObterCredenciaisMeta(ctx context.Context, instanciaID string) (models.CredenciaisMeta, error)
	SalvarCredenciaisMeta(ctx context.Context, cred models.CredenciaisMeta) error
}

func (s *SQLStore) ObterCredenciaisMeta(ctx context.Context, instanciaID string) (models.CredenciaisMeta, error) {
	cred := models.CredenciaisMeta{InstanciaID: instanciaID}
	err := s.db.QueryRowContext(ctx, s.q(`
SELECT phone_number_id, waba_id, access_token, app_secret, verify_token, numero_exibicao, nome_verificado, atualizado_em
FROM instancias_meta WHERE instancia_id = ?`), instanciaID).Scan(
		&cred.PhoneNumberID, &cred.WABAID, &cred.AccessToken, &cred.AppSecret, &cred.VerifyToken,
		&cred.NumeroExibicao, &cred.NomeVerificado, &cred.AtualizadoEm,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CredenciaisMeta{}, ErrCredenciaisMetaNaoEncontradas
	}
	if err != nil {
		return models.CredenciaisMeta{}, fmt.Errorf("erro ao buscar credenciais da Meta: %w", err)
	}
	return cred, nil
}

func (s *SQLStore) SalvarCredenciaisMeta(ctx context.Context, cred models.CredenciaisMeta) error {
	if cred.AtualizadoEm.IsZero() {
		cred.AtualizadoEm = time.Now()
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO instancias_meta (instancia_id, phone_number_id, waba_id, access_token, app_secret, verify_token, numero_exibicao, nome_verificado, atualizado_em)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(instancia_id) DO UPDATE SET
    phone_number_id = excluded.phone_number_id,
    waba_id = excluded.waba_id,
    access_token = excluded.access_token,
    app_secret = excluded.app_secret,
    verify_token = excluded.verify_token,
    numero_exibicao = excluded.numero_exibicao,
    nome_verificado = excluded.nome_verificado,
    atualizado_em = excluded.atualizado_em`),
		cred.InstanciaID, cred.PhoneNumberID, cred.WABAID, cred.AccessToken, cred.AppSecret, cred.VerifyToken,
		cred.NumeroExibicao, cred.NomeVerificado, cred.AtualizadoEm.UTC(),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fmt.Errorf("esse phone_number_id ja esta em outra instancia")
		}
		return fmt.Errorf("erro ao salvar credenciais da Meta: %w", err)
	}
	return nil
}
