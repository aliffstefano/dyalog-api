package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dyalog-api-go/internal/models"
)

// PerfilStore guarda o ultimo perfil (numero, nome, foto) visto pela replica
// dona de cada instancia, para as outras replicas mostrarem o mesmo no painel.
type PerfilStore interface {
	SalvarPerfilInstancia(ctx context.Context, instanciaID string, perfil models.PerfilInstancia) error
	ListarPerfisInstancias(ctx context.Context) (map[string]models.PerfilInstancia, error)
	ExcluirPerfilInstancia(ctx context.Context, instanciaID string) error
}

func (s *SQLStore) SalvarPerfilInstancia(ctx context.Context, instanciaID string, perfil models.PerfilInstancia) error {
	dados, err := json.Marshal(perfil)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, s.q(`
INSERT INTO instancias_perfil (instancia_id, dados, atualizado_em)
VALUES (?, ?, ?)
ON CONFLICT(instancia_id) DO UPDATE SET
    dados = excluded.dados,
    atualizado_em = excluded.atualizado_em`),
		instanciaID, string(dados), time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("erro ao salvar perfil da instancia: %w", err)
	}
	return nil
}

func (s *SQLStore) ListarPerfisInstancias(ctx context.Context) (map[string]models.PerfilInstancia, error) {
	linhas, err := s.db.QueryContext(ctx, `SELECT instancia_id, dados FROM instancias_perfil`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar perfis das instancias: %w", err)
	}
	defer linhas.Close()
	perfis := make(map[string]models.PerfilInstancia)
	for linhas.Next() {
		var id, dados string
		if err := linhas.Scan(&id, &dados); err != nil {
			return nil, fmt.Errorf("erro ao ler perfil da instancia: %w", err)
		}
		var perfil models.PerfilInstancia
		if json.Unmarshal([]byte(dados), &perfil) == nil && perfil.Numero != "" {
			perfis[id] = perfil
		}
	}
	return perfis, linhas.Err()
}

func (s *SQLStore) ExcluirPerfilInstancia(ctx context.Context, instanciaID string) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM instancias_perfil WHERE instancia_id = ?`), instanciaID)
	if err != nil {
		return fmt.Errorf("erro ao excluir perfil da instancia: %w", err)
	}
	return nil
}
