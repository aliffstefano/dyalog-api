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

// ChatStore guarda as mensagens da tela de Chat do painel.
type ChatStore interface {
	RegistrarChatMensagem(ctx context.Context, mensagem models.ChatMensagem) error
	EditarChatMensagem(ctx context.Context, instanciaID, mensagemID, tipo, conteudo string) error
	AtualizarStatusChat(ctx context.Context, instanciaID string, mensagensID []string, status, erro string) error
	ListarConversasChat(ctx context.Context, instanciaID string, limite int) ([]models.ChatConversa, error)
	ListarMensagensChat(ctx context.Context, instanciaID, chatJID string, limite int) ([]models.ChatMensagem, error)
	LimparChatAntigo(ctx context.Context, antesDe time.Time) (int64, error)
}

// ordemStatusChat impede que um recibo atrasado volte o status (lida nao
// volta para entregue). Status fora da lista nao mudam a mensagem.
var ordemStatusChat = map[string]int{"enviada": 1, "entregue": 2, "lida": 3, "ouvida": 4, "falhou": 5}

// OrdemStatusChat diz se o status e acompanhado pelo chat (0 = ignorado).
func OrdemStatusChat(status string) int { return ordemStatusChat[status] }

const colunasChat = `instancia_id, chat_jid, mensagem_id, direcao, tipo, conteudo, nome, remetente_jid, status, erro, midia_id, mime_type, nome_arquivo, criado_em`

func (s *SQLStore) RegistrarChatMensagem(ctx context.Context, m models.ChatMensagem) error {
	if m.CriadoEm.IsZero() {
		m.CriadoEm = time.Now()
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO chat_mensagens (`+colunasChat+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(instancia_id, mensagem_id) DO NOTHING`),
		m.InstanciaID, m.ChatJID, m.MensagemID, m.Direcao, m.Tipo, m.Conteudo, m.Nome, m.RemetenteJID,
		m.Status, m.Erro, m.MidiaID, m.MimeType, m.NomeArquivo, m.CriadoEm.UTC(),
	)
	if err != nil {
		return fmt.Errorf("erro ao registrar mensagem do chat: %w", err)
	}
	return nil
}

func (s *SQLStore) EditarChatMensagem(ctx context.Context, instanciaID, mensagemID, tipo, conteudo string) error {
	_, err := s.db.ExecContext(ctx, s.q(`
UPDATE chat_mensagens SET tipo = CASE WHEN ? = '' THEN tipo ELSE ? END, conteudo = ?
WHERE instancia_id = ? AND mensagem_id = ?`), tipo, tipo, conteudo, instanciaID, mensagemID)
	if err != nil {
		return fmt.Errorf("erro ao editar mensagem do chat: %w", err)
	}
	return nil
}

func (s *SQLStore) AtualizarStatusChat(ctx context.Context, instanciaID string, mensagensID []string, status, erro string) error {
	ordem := ordemStatusChat[status]
	if ordem == 0 || len(mensagensID) == 0 {
		return nil
	}
	marcadores := strings.TrimSuffix(strings.Repeat("?,", len(mensagensID)), ",")
	args := []any{status, erro, instanciaID}
	for _, id := range mensagensID {
		args = append(args, id)
	}
	args = append(args, ordem)
	_, err := s.db.ExecContext(ctx, s.q(`
UPDATE chat_mensagens SET status = ?, erro = ?
WHERE instancia_id = ? AND direcao = 'saida' AND mensagem_id IN (`+marcadores+`)
  AND (CASE status WHEN 'enviada' THEN 1 WHEN 'entregue' THEN 2 WHEN 'lida' THEN 3 WHEN 'ouvida' THEN 4 WHEN 'falhou' THEN 5 ELSE 0 END) < ?`), args...)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status no chat: %w", err)
	}
	return nil
}

func (s *SQLStore) ListarConversasChat(ctx context.Context, instanciaID string, limite int) ([]models.ChatConversa, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`
SELECT chat_jid FROM chat_mensagens WHERE instancia_id = ?
GROUP BY chat_jid ORDER BY MAX(criado_em) DESC LIMIT ?`), instanciaID, limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar conversas: %w", err)
	}
	var chats []string
	for linhas.Next() {
		var jid string
		if err := linhas.Scan(&jid); err != nil {
			linhas.Close()
			return nil, err
		}
		chats = append(chats, jid)
	}
	linhas.Close()
	if err := linhas.Err(); err != nil {
		return nil, err
	}

	conversas := make([]models.ChatConversa, 0, len(chats))
	for _, jid := range chats {
		ultimas, err := s.ListarMensagensChat(ctx, instanciaID, jid, 1)
		if err != nil {
			return nil, err
		}
		if len(ultimas) == 0 {
			continue
		}
		conversa := models.ChatConversa{ChatJID: jid, Grupo: strings.HasSuffix(jid, "@g.us"), Ultima: ultimas[0]}
		// Nome do contato: o ultimo nome visto numa mensagem recebida (em grupo
		// seria o de quem escreveu, entao fica sem).
		if !conversa.Grupo {
			err := s.db.QueryRowContext(ctx, s.q(`
SELECT nome FROM chat_mensagens
WHERE instancia_id = ? AND chat_jid = ? AND direcao = 'entrada' AND nome <> ''
ORDER BY criado_em DESC LIMIT 1`), instanciaID, jid).Scan(&conversa.Nome)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		conversas = append(conversas, conversa)
	}
	return conversas, nil
}

// ListarMensagensChat devolve as ultimas mensagens da conversa, da mais antiga
// para a mais nova.
func (s *SQLStore) ListarMensagensChat(ctx context.Context, instanciaID, chatJID string, limite int) ([]models.ChatMensagem, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`
SELECT `+colunasChat+` FROM chat_mensagens
WHERE instancia_id = ? AND chat_jid = ?
ORDER BY criado_em DESC LIMIT ?`), instanciaID, chatJID, limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar mensagens do chat: %w", err)
	}
	defer linhas.Close()
	var mensagens []models.ChatMensagem
	for linhas.Next() {
		var m models.ChatMensagem
		if err := linhas.Scan(&m.InstanciaID, &m.ChatJID, &m.MensagemID, &m.Direcao, &m.Tipo, &m.Conteudo, &m.Nome,
			&m.RemetenteJID, &m.Status, &m.Erro, &m.MidiaID, &m.MimeType, &m.NomeArquivo, &m.CriadoEm); err != nil {
			return nil, fmt.Errorf("erro ao ler mensagem do chat: %w", err)
		}
		mensagens = append(mensagens, m)
	}
	if err := linhas.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(mensagens)-1; i < j; i, j = i+1, j-1 {
		mensagens[i], mensagens[j] = mensagens[j], mensagens[i]
	}
	return mensagens, nil
}

func (s *SQLStore) LimparChatAntigo(ctx context.Context, antesDe time.Time) (int64, error) {
	resultado, err := s.db.ExecContext(ctx, s.q(`DELETE FROM chat_mensagens WHERE criado_em < ?`), antesDe.UTC())
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar mensagens antigas do chat: %w", err)
	}
	return resultado.RowsAffected()
}
