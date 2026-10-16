package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"dyalog-api-go/internal/models"
)

// RespostasStore: respostas rapidas e etiquetas do chat, por conta.
type RespostasStore interface {
	ListarRespostasRapidas(ctx context.Context, contaID string) ([]models.RespostaRapida, error)
	SalvarRespostaRapida(ctx context.Context, r models.RespostaRapida) (models.RespostaRapida, error)
	ExcluirRespostaRapida(ctx context.Context, contaID, id string) error
	ListarEtiquetas(ctx context.Context, contaID string) ([]models.Etiqueta, error)
	SalvarEtiqueta(ctx context.Context, e models.Etiqueta) (models.Etiqueta, error)
	ExcluirEtiqueta(ctx context.Context, contaID, id string) error
	EtiquetasDoChat(ctx context.Context, instanciaID string) (map[string][]string, error)
	DefinirEtiquetasChat(ctx context.Context, instanciaID, chatJID string, etiquetas []string) error
}

// ErrDuplicado: atalho ou nome de etiqueta ja usado na conta.
var ErrDuplicado = errors.New("ja existe")

// ErrNaoEncontrado: registro inexistente nesta conta.
var ErrNaoEncontrado = errors.New("nao encontrado")

func erroUnico(err error) bool {
	texto := strings.ToLower(err.Error())
	return strings.Contains(texto, "unique") || strings.Contains(texto, "duplicate")
}

func (s *SQLStore) ListarRespostasRapidas(ctx context.Context, contaID string) ([]models.RespostaRapida, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT id, conta_id, atalho, texto, atualizado_em FROM respostas_rapidas WHERE conta_id = ? ORDER BY atalho`), contaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar respostas rapidas: %w", err)
	}
	defer linhas.Close()
	lista := []models.RespostaRapida{}
	for linhas.Next() {
		var r models.RespostaRapida
		if err := linhas.Scan(&r.ID, &r.ContaID, &r.Atalho, &r.Texto, &r.AtualizadoEm); err != nil {
			return nil, err
		}
		lista = append(lista, r)
	}
	return lista, linhas.Err()
}

// SalvarRespostaRapida cria (sem ID) ou atualiza (com ID, so da mesma conta).
func (s *SQLStore) SalvarRespostaRapida(ctx context.Context, r models.RespostaRapida) (models.RespostaRapida, error) {
	r.AtualizadoEm = time.Now().UTC()
	var err error
	if r.ID == "" {
		r.ID = uuid.NewString()
		_, err = s.db.ExecContext(ctx, s.q(`INSERT INTO respostas_rapidas (id, conta_id, atalho, texto, atualizado_em) VALUES (?, ?, ?, ?, ?)`), r.ID, r.ContaID, r.Atalho, r.Texto, r.AtualizadoEm)
	} else {
		var res interface{ RowsAffected() (int64, error) }
		res, err = s.db.ExecContext(ctx, s.q(`UPDATE respostas_rapidas SET atalho = ?, texto = ?, atualizado_em = ? WHERE id = ? AND conta_id = ?`), r.Atalho, r.Texto, r.AtualizadoEm, r.ID, r.ContaID)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				return r, ErrNaoEncontrado
			}
		}
	}
	if err != nil {
		if erroUnico(err) {
			return r, ErrDuplicado
		}
		return r, fmt.Errorf("erro ao salvar resposta rapida: %w", err)
	}
	return r, nil
}

func (s *SQLStore) ExcluirRespostaRapida(ctx context.Context, contaID, id string) error {
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM respostas_rapidas WHERE id = ? AND conta_id = ?`), id, contaID)
	if err != nil {
		return fmt.Errorf("erro ao excluir resposta rapida: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNaoEncontrado
	}
	return nil
}

func (s *SQLStore) ListarEtiquetas(ctx context.Context, contaID string) ([]models.Etiqueta, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT id, conta_id, nome, cor FROM etiquetas WHERE conta_id = ? ORDER BY nome`), contaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar etiquetas: %w", err)
	}
	defer linhas.Close()
	lista := []models.Etiqueta{}
	for linhas.Next() {
		var e models.Etiqueta
		if err := linhas.Scan(&e.ID, &e.ContaID, &e.Nome, &e.Cor); err != nil {
			return nil, err
		}
		lista = append(lista, e)
	}
	return lista, linhas.Err()
}

func (s *SQLStore) SalvarEtiqueta(ctx context.Context, e models.Etiqueta) (models.Etiqueta, error) {
	var err error
	if e.ID == "" {
		e.ID = uuid.NewString()
		_, err = s.db.ExecContext(ctx, s.q(`INSERT INTO etiquetas (id, conta_id, nome, cor) VALUES (?, ?, ?, ?)`), e.ID, e.ContaID, e.Nome, e.Cor)
	} else {
		var res interface{ RowsAffected() (int64, error) }
		res, err = s.db.ExecContext(ctx, s.q(`UPDATE etiquetas SET nome = ?, cor = ? WHERE id = ? AND conta_id = ?`), e.Nome, e.Cor, e.ID, e.ContaID)
		if err == nil {
			if n, _ := res.RowsAffected(); n == 0 {
				return e, ErrNaoEncontrado
			}
		}
	}
	if err != nil {
		if erroUnico(err) {
			return e, ErrDuplicado
		}
		return e, fmt.Errorf("erro ao salvar etiqueta: %w", err)
	}
	return e, nil
}

func (s *SQLStore) ExcluirEtiqueta(ctx context.Context, contaID, id string) error {
	if _, err := s.db.ExecContext(ctx, s.q(`DELETE FROM chat_etiquetas WHERE etiqueta_id = ? AND etiqueta_id IN (SELECT id FROM etiquetas WHERE conta_id = ?)`), id, contaID); err != nil {
		return fmt.Errorf("erro ao excluir etiqueta: %w", err)
	}
	res, err := s.db.ExecContext(ctx, s.q(`DELETE FROM etiquetas WHERE id = ? AND conta_id = ?`), id, contaID)
	if err != nil {
		return fmt.Errorf("erro ao excluir etiqueta: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNaoEncontrado
	}
	return nil
}

func (s *SQLStore) EtiquetasDoChat(ctx context.Context, instanciaID string) (map[string][]string, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT chat_jid, etiqueta_id FROM chat_etiquetas WHERE instancia_id = ?`), instanciaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar etiquetas do chat: %w", err)
	}
	defer linhas.Close()
	mapa := map[string][]string{}
	for linhas.Next() {
		var jid, etiqueta string
		if err := linhas.Scan(&jid, &etiqueta); err != nil {
			return nil, err
		}
		mapa[jid] = append(mapa[jid], etiqueta)
	}
	return mapa, linhas.Err()
}

// DefinirEtiquetasChat troca todas as etiquetas da conversa.
func (s *SQLStore) DefinirEtiquetasChat(ctx context.Context, instanciaID, chatJID string, etiquetas []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, s.q(`DELETE FROM chat_etiquetas WHERE instancia_id = ? AND chat_jid = ?`), instanciaID, chatJID); err != nil {
		return fmt.Errorf("erro ao limpar etiquetas: %w", err)
	}
	for _, id := range etiquetas {
		if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO chat_etiquetas (instancia_id, chat_jid, etiqueta_id) VALUES (?, ?, ?)`), instanciaID, chatJID, id); err != nil {
			return fmt.Errorf("erro ao gravar etiqueta: %w", err)
		}
	}
	return tx.Commit()
}
