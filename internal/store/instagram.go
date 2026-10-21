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

// ErrCredenciaisInstagramNaoEncontradas indica instancia do Instagram ainda
// sem conta conectada.
var ErrCredenciaisInstagramNaoEncontradas = errors.New("credenciais do Instagram nao cadastradas")

// InstagramStore guarda as contas conectadas nas instancias do Instagram.
type InstagramStore interface {
	ObterCredenciaisInstagram(ctx context.Context, instanciaID string) (models.CredenciaisInstagram, error)
	ObterCredenciaisInstagramPorConta(ctx context.Context, contaID string) (models.CredenciaisInstagram, error)
	SalvarCredenciaisInstagram(ctx context.Context, cred models.CredenciaisInstagram) error
	// ListarCredenciaisInstagramAntigas devolve as salvas antes de "antes", para
	// renovar o token de 60 dias.
	ListarCredenciaisInstagramAntigas(ctx context.Context, antes time.Time) ([]models.CredenciaisInstagram, error)
}

const colunasInstagram = `instancia_id, conta_id, usuario, nome, foto_url, access_token, atualizado_em`

func escanearInstagram(linha interface{ Scan(...any) error }) (models.CredenciaisInstagram, error) {
	var cred models.CredenciaisInstagram
	err := linha.Scan(&cred.InstanciaID, &cred.ContaID, &cred.Usuario, &cred.Nome, &cred.FotoURL, &cred.AccessToken, &cred.AtualizadoEm)
	return cred, err
}

func (s *SQLStore) ObterCredenciaisInstagram(ctx context.Context, instanciaID string) (models.CredenciaisInstagram, error) {
	return s.buscarCredenciaisInstagram(ctx, "instancia_id", instanciaID)
}

// ObterCredenciaisInstagramPorConta acha a instancia pelo id da conta: o
// webhook do Instagram chega com a conta, nao com a instancia.
func (s *SQLStore) ObterCredenciaisInstagramPorConta(ctx context.Context, contaID string) (models.CredenciaisInstagram, error) {
	if contaID == "" {
		return models.CredenciaisInstagram{}, ErrCredenciaisInstagramNaoEncontradas
	}
	return s.buscarCredenciaisInstagram(ctx, "conta_id", contaID)
}

func (s *SQLStore) buscarCredenciaisInstagram(ctx context.Context, coluna, valor string) (models.CredenciaisInstagram, error) {
	cred, err := escanearInstagram(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasInstagram+` FROM instancias_instagram WHERE `+coluna+` = ?`), valor))
	if errors.Is(err, sql.ErrNoRows) {
		return models.CredenciaisInstagram{}, ErrCredenciaisInstagramNaoEncontradas
	}
	if err != nil {
		return models.CredenciaisInstagram{}, fmt.Errorf("erro ao buscar credenciais do Instagram: %w", err)
	}
	return cred, nil
}

func (s *SQLStore) SalvarCredenciaisInstagram(ctx context.Context, cred models.CredenciaisInstagram) error {
	if cred.AtualizadoEm.IsZero() {
		cred.AtualizadoEm = time.Now()
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO instancias_instagram (`+colunasInstagram+`)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(instancia_id) DO UPDATE SET
    conta_id = excluded.conta_id,
    usuario = excluded.usuario,
    nome = excluded.nome,
    foto_url = excluded.foto_url,
    access_token = excluded.access_token,
    atualizado_em = excluded.atualizado_em`),
		cred.InstanciaID, cred.ContaID, cred.Usuario, cred.Nome, cred.FotoURL, cred.AccessToken, cred.AtualizadoEm.UTC(),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return fmt.Errorf("essa conta do Instagram ja esta em outra instancia")
		}
		return fmt.Errorf("erro ao salvar credenciais do Instagram: %w", err)
	}
	return nil
}

func (s *SQLStore) ListarCredenciaisInstagramAntigas(ctx context.Context, antes time.Time) ([]models.CredenciaisInstagram, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT `+colunasInstagram+` FROM instancias_instagram WHERE access_token <> '' AND atualizado_em < ?`), antes.UTC())
	if err != nil {
		return nil, fmt.Errorf("erro ao listar credenciais do Instagram: %w", err)
	}
	defer linhas.Close()
	var lista []models.CredenciaisInstagram
	for linhas.Next() {
		cred, err := escanearInstagram(linhas)
		if err != nil {
			return nil, fmt.Errorf("erro ao ler credenciais do Instagram: %w", err)
		}
		lista = append(lista, cred)
	}
	return lista, linhas.Err()
}
