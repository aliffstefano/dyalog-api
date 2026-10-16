package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dyalog-api-go/internal/models"
)

// AuditoriaStore guarda o historico de acoes e eventos por instancia/conta.
type AuditoriaStore interface {
	RegistrarAuditoria(ctx context.Context, r models.RegistroAuditoria) error
	ListarAuditoria(ctx context.Context, f models.FiltroAuditoria) (models.PaginaAuditoria, error)
	LimparAuditoriaAntiga(ctx context.Context, antesDe time.Time) (int64, error)
}

const colunasAuditoria = `id, instancia_id, conta_id, tipo, acao, descricao, severidade, ator_tipo, ator_id, ator_nome, ip, detalhes, criado_em`

func (s *SQLStore) RegistrarAuditoria(ctx context.Context, r models.RegistroAuditoria) error {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CriadoEm.IsZero() {
		r.CriadoEm = time.Now()
	}
	if r.Severidade == "" {
		r.Severidade = models.SeveridadeInfo
	}
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO auditoria (`+colunasAuditoria+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		r.ID, r.InstanciaID, r.ContaID, r.Tipo, r.Acao, r.Descricao, r.Severidade, r.AtorTipo, r.AtorID, r.AtorNome, r.IP, r.Detalhes, r.CriadoEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao registrar auditoria: %w", err)
	}
	return nil
}

// ListarAuditoria devolve do mais novo para o mais antigo.
func (s *SQLStore) ListarAuditoria(ctx context.Context, f models.FiltroAuditoria) (models.PaginaAuditoria, error) {
	var (
		condicoes []string
		args      []any
	)
	if f.InstanciaID != "" {
		condicoes = append(condicoes, "a.instancia_id = ?")
		args = append(args, f.InstanciaID)
	}
	if f.SoContas {
		if len(f.ContasIDs) == 0 {
			return models.PaginaAuditoria{Registros: []models.RegistroAuditoria{}}, nil
		}
		condicoes = append(condicoes, "a.conta_id IN ("+strings.TrimSuffix(strings.Repeat("?,", len(f.ContasIDs)), ",")+")")
		for _, id := range f.ContasIDs {
			args = append(args, id)
		}
	}
	if f.Tipo != "" {
		condicoes = append(condicoes, "a.tipo = ?")
		args = append(args, f.Tipo)
	}
	if f.Severidade != "" {
		condicoes = append(condicoes, "a.severidade = ?")
		args = append(args, f.Severidade)
	}
	if ator := strings.TrimSpace(f.Ator); ator != "" {
		condicoes = append(condicoes, "LOWER(a.ator_nome) LIKE ?")
		args = append(args, "%"+strings.ToLower(ator)+"%")
	}
	if !f.De.IsZero() {
		condicoes = append(condicoes, "a.criado_em >= ?")
		args = append(args, f.De.UTC())
	}
	if !f.Ate.IsZero() {
		condicoes = append(condicoes, "a.criado_em < ?")
		args = append(args, f.Ate.UTC())
	}
	where := ""
	if len(condicoes) > 0 {
		where = "WHERE " + strings.Join(condicoes, " AND ")
	}
	limite := f.Limite
	if limite <= 0 || limite > 500 {
		limite = 50
	}
	args = append(args, limite+1, max(f.Deslocamento, 0))
	colunas := "a." + strings.ReplaceAll(colunasAuditoria, ", ", ", a.")
	linhas, err := s.db.QueryContext(ctx, s.q(`
SELECT `+colunas+`, COALESCE(i.nome, '') FROM auditoria a
LEFT JOIN instancias i ON i.id = a.instancia_id
`+where+`
ORDER BY a.criado_em DESC, a.id DESC LIMIT ? OFFSET ?`), args...)
	if err != nil {
		return models.PaginaAuditoria{}, fmt.Errorf("erro ao listar auditoria: %w", err)
	}
	defer linhas.Close()
	pagina := models.PaginaAuditoria{Registros: []models.RegistroAuditoria{}}
	for linhas.Next() {
		var r models.RegistroAuditoria
		if err := linhas.Scan(&r.ID, &r.InstanciaID, &r.ContaID, &r.Tipo, &r.Acao, &r.Descricao, &r.Severidade,
			&r.AtorTipo, &r.AtorID, &r.AtorNome, &r.IP, &r.Detalhes, &r.CriadoEm, &r.InstanciaNome); err != nil {
			return models.PaginaAuditoria{}, err
		}
		pagina.Registros = append(pagina.Registros, r)
	}
	if err := linhas.Err(); err != nil {
		return models.PaginaAuditoria{}, err
	}
	if len(pagina.Registros) > limite {
		pagina.Registros = pagina.Registros[:limite]
		pagina.TemMais = true
	}
	return pagina, nil
}

func (s *SQLStore) LimparAuditoriaAntiga(ctx context.Context, antesDe time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM auditoria WHERE criado_em < ?`), antesDe.UTC())
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar auditoria: %w", err)
	}
	return res.RowsAffected()
}
