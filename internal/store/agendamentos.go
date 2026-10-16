package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dyalog-api-go/internal/models"
)

// AgendamentoStore guarda as mensagens agendadas e a fila de envio.
type AgendamentoStore interface {
	CriarAgendamento(ctx context.Context, a models.Agendamento) (models.Agendamento, error)
	BuscarAgendamento(ctx context.Context, id string) (models.Agendamento, error)
	ListarAgendamentos(ctx context.Context, f models.FiltroAgendamentos) ([]models.Agendamento, error)
	AtualizarAgendamentoPendente(ctx context.Context, id string, mensagem *string, enviarEm *time.Time, recorrencia *string) error
	CancelarAgendamento(ctx context.Context, id string) error
	AgendamentosVencidos(ctx context.Context, agora time.Time, limite int) ([]models.Agendamento, error)
	ReservarAgendamento(ctx context.Context, id, dono string, agora time.Time) (bool, error)
	ConcluirAgendamento(ctx context.Context, id, status, erro, mensagemID string, tentativas int, quando time.Time) error
	AdiarAgendamento(ctx context.Context, id, erro string, tentativas int, tentarEm time.Time) error
	LiberarAgendamentosPresos(ctx context.Context, antesDe time.Time) (int64, error)
}

// ErrAgendamentoNaoEncontrado: id inexistente ou ja fora de pendente.
var ErrAgendamentoNaoEncontrado = errors.New("agendamento nao encontrado")

const colunasAgendamento = `a.id, a.instancia_id, a.conta_id, a.tipo, a.numero, a.chat_jid, a.mensagem, a.arquivo_url, a.arquivo_base64,
a.nome_arquivo, a.mime_type, a.enviar_em, a.tentar_em, a.recorrencia, a.status, a.tentativas, a.erro, a.mensagem_id, a.criado_por,
a.criado_em, a.enviado_em, COALESCE(i.nome, '')`

func (s *SQLStore) CriarAgendamento(ctx context.Context, a models.Agendamento) (models.Agendamento, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.CriadoEm.IsZero() {
		a.CriadoEm = time.Now()
	}
	if a.Status == "" {
		a.Status = models.AgendamentoPendente
	}
	if a.TentarEm.IsZero() {
		a.TentarEm = a.EnviarEm
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO agendamentos (id, instancia_id, conta_id, tipo, numero, chat_jid, mensagem, arquivo_url, arquivo_base64, nome_arquivo,
  mime_type, enviar_em, tentar_em, recorrencia, status, criado_por, criado_em)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		a.ID, a.InstanciaID, a.ContaID, a.Tipo, a.Numero, a.ChatJID, a.Mensagem, a.ArquivoURL, a.ArquivoBase64, a.NomeArquivo,
		a.MimeType, a.EnviarEm.UTC(), a.TentarEm.UTC(), a.Recorrencia, a.Status, a.CriadoPor, a.CriadoEm.UTC())
	if err != nil {
		return models.Agendamento{}, fmt.Errorf("erro ao criar agendamento: %w", err)
	}
	a.TemArquivo = a.ArquivoBase64 != "" || a.ArquivoURL != ""
	return a, nil
}

func lerAgendamento(linha interface{ Scan(...any) error }) (models.Agendamento, error) {
	var (
		a         models.Agendamento
		enviadoEm sql.NullTime
	)
	err := linha.Scan(&a.ID, &a.InstanciaID, &a.ContaID, &a.Tipo, &a.Numero, &a.ChatJID, &a.Mensagem, &a.ArquivoURL, &a.ArquivoBase64,
		&a.NomeArquivo, &a.MimeType, &a.EnviarEm, &a.TentarEm, &a.Recorrencia, &a.Status, &a.Tentativas, &a.Erro, &a.MensagemID, &a.CriadoPor,
		&a.CriadoEm, &enviadoEm, &a.InstanciaNome)
	if err != nil {
		return a, err
	}
	if enviadoEm.Valid {
		a.EnviadoEm = enviadoEm.Time
	}
	a.TemArquivo = a.ArquivoBase64 != "" || a.ArquivoURL != ""
	return a, nil
}

func (s *SQLStore) BuscarAgendamento(ctx context.Context, id string) (models.Agendamento, error) {
	a, err := lerAgendamento(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasAgendamento+` FROM agendamentos a LEFT JOIN instancias i ON i.id = a.instancia_id WHERE a.id = ?`), id))
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrAgendamentoNaoEncontrado
	}
	return a, err
}

func (s *SQLStore) listarAgendamentos(ctx context.Context, where string, args []any, ordem string, limite int) ([]models.Agendamento, error) {
	args = append(args, limite)
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT `+colunasAgendamento+` FROM agendamentos a LEFT JOIN instancias i ON i.id = a.instancia_id `+where+` ORDER BY `+ordem+` LIMIT ?`), args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar agendamentos: %w", err)
	}
	defer linhas.Close()
	lista := []models.Agendamento{}
	for linhas.Next() {
		a, err := lerAgendamento(linhas)
		if err != nil {
			return nil, err
		}
		lista = append(lista, a)
	}
	return lista, linhas.Err()
}

func (s *SQLStore) ListarAgendamentos(ctx context.Context, f models.FiltroAgendamentos) ([]models.Agendamento, error) {
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
			return []models.Agendamento{}, nil
		}
		condicoes = append(condicoes, "a.conta_id IN ("+strings.TrimSuffix(strings.Repeat("?,", len(f.ContasIDs)), ",")+")")
		for _, id := range f.ContasIDs {
			args = append(args, id)
		}
	}
	if f.Status != "" {
		condicoes = append(condicoes, "a.status = ?")
		args = append(args, f.Status)
	}
	where := ""
	if len(condicoes) > 0 {
		where = "WHERE " + strings.Join(condicoes, " AND ")
	}
	limite := f.Limite
	if limite <= 0 || limite > 500 {
		limite = 200
	}
	// Pendentes primeiro (o proximo no topo); depois o historico, mais novo antes.
	return s.listarAgendamentos(ctx, where, args, `CASE WHEN a.status IN ('pendente','enviando') THEN 0 ELSE 1 END,
  CASE WHEN a.status IN ('pendente','enviando') THEN a.enviar_em END ASC, a.enviar_em DESC`, limite)
}

func (s *SQLStore) AtualizarAgendamentoPendente(ctx context.Context, id string, mensagem *string, enviarEm *time.Time, recorrencia *string) error {
	var (
		sets []string
		args []any
	)
	if mensagem != nil {
		sets = append(sets, "mensagem = ?")
		args = append(args, *mensagem)
	}
	if enviarEm != nil {
		sets = append(sets, "enviar_em = ?", "tentar_em = ?", "tentativas = 0", "erro = ''")
		args = append(args, enviarEm.UTC(), enviarEm.UTC())
	}
	if recorrencia != nil {
		sets = append(sets, "recorrencia = ?")
		args = append(args, *recorrencia)
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET `+strings.Join(sets, ", ")+` WHERE id = ? AND status = 'pendente'`), args...)
	if err != nil {
		return fmt.Errorf("erro ao atualizar agendamento: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAgendamentoNaoEncontrado
	}
	return nil
}

func (s *SQLStore) CancelarAgendamento(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET status = 'cancelado', arquivo_base64 = '' WHERE id = ? AND status = 'pendente'`), id)
	if err != nil {
		return fmt.Errorf("erro ao cancelar agendamento: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrAgendamentoNaoEncontrado
	}
	return nil
}

func (s *SQLStore) AgendamentosVencidos(ctx context.Context, agora time.Time, limite int) ([]models.Agendamento, error) {
	return s.listarAgendamentos(ctx, "WHERE a.status = 'pendente' AND a.tentar_em <= ?", []any{agora.UTC()}, "a.tentar_em ASC", limite)
}

// ReservarAgendamento marca como enviando; so uma replica consegue.
func (s *SQLStore) ReservarAgendamento(ctx context.Context, id, dono string, agora time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET status = 'enviando', dono = ?, reservado_em = ? WHERE id = ? AND status = 'pendente'`), dono, agora.UTC(), id)
	if err != nil {
		return false, fmt.Errorf("erro ao reservar agendamento: %w", err)
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// ConcluirAgendamento grava o resultado final; o arquivo sai do banco.
func (s *SQLStore) ConcluirAgendamento(ctx context.Context, id, status, erro, mensagemID string, tentativas int, quando time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET status = ?, erro = ?, mensagem_id = ?, tentativas = ?, enviado_em = ?, arquivo_base64 = '' WHERE id = ?`),
		status, erro, mensagemID, tentativas, quando.UTC(), id)
	if err != nil {
		return fmt.Errorf("erro ao concluir agendamento: %w", err)
	}
	return nil
}

func (s *SQLStore) AdiarAgendamento(ctx context.Context, id, erro string, tentativas int, tentarEm time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET status = 'pendente', erro = ?, tentativas = ?, tentar_em = ?, dono = '', reservado_em = NULL WHERE id = ?`),
		erro, tentativas, tentarEm.UTC(), id)
	if err != nil {
		return fmt.Errorf("erro ao adiar agendamento: %w", err)
	}
	return nil
}

// LiberarAgendamentosPresos devolve para a fila o que ficou "enviando" porque a
// replica caiu no meio.
func (s *SQLStore) LiberarAgendamentosPresos(ctx context.Context, antesDe time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE agendamentos SET status = 'pendente', dono = '', reservado_em = NULL WHERE status = 'enviando' AND reservado_em < ?`), antesDe.UTC())
	if err != nil {
		return 0, fmt.Errorf("erro ao liberar agendamentos: %w", err)
	}
	return res.RowsAffected()
}
