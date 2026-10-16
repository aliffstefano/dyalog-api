package models

import "time"

// Status dos agendamentos.
const (
	AgendamentoPendente  = "pendente"
	AgendamentoEnviando  = "enviando"
	AgendamentoEnviado   = "enviado"
	AgendamentoFalhou    = "falhou"
	AgendamentoCancelado = "cancelado"
)

// Agendamento e uma mensagem para enviar mais tarde (texto ou arquivo).
type Agendamento struct {
	ID            string    `json:"id"`
	InstanciaID   string    `json:"instancia_id"`
	InstanciaNome string    `json:"instancia_nome,omitempty"`
	ContaID       string    `json:"conta_id,omitempty"`
	Tipo          string    `json:"tipo"` // texto, imagem, documento, audio
	Numero        string    `json:"numero,omitempty"`
	ChatJID       string    `json:"chat_jid,omitempty"`
	Mensagem      string    `json:"mensagem,omitempty"`
	ArquivoURL    string    `json:"arquivo_url,omitempty"`
	ArquivoBase64 string    `json:"-"`
	TemArquivo    bool      `json:"tem_arquivo"`
	NomeArquivo   string    `json:"nome_arquivo,omitempty"`
	MimeType      string    `json:"mime_type,omitempty"`
	EnviarEm      time.Time `json:"enviar_em"`
	TentarEm      time.Time `json:"-"`
	Recorrencia   string    `json:"recorrencia,omitempty"` // diaria, semanal, mensal
	Status        string    `json:"status"`
	Tentativas    int       `json:"tentativas"`
	Erro          string    `json:"erro,omitempty"`
	MensagemID    string    `json:"mensagem_id,omitempty"`
	CriadoPor     string    `json:"criado_por,omitempty"`
	CriadoEm      time.Time `json:"criado_em"`
	EnviadoEm     time.Time `json:"enviado_em,omitempty"`
}

type CriarAgendamentoRequest struct {
	Instancia     string    `json:"instancia"`
	Numero        string    `json:"numero"`
	ChatJID       string    `json:"chat_jid"`
	Tipo          string    `json:"tipo"`
	Mensagem      string    `json:"mensagem"`
	ArquivoURL    string    `json:"arquivo_url"`
	ArquivoBase64 string    `json:"arquivo_base64"`
	NomeArquivo   string    `json:"nome_arquivo"`
	MimeType      string    `json:"mime_type"`
	EnviarEm      time.Time `json:"enviar_em"`
	Recorrencia   string    `json:"recorrencia"`
}

// AtualizarAgendamentoRequest so vale enquanto pendente.
type AtualizarAgendamentoRequest struct {
	Mensagem    *string    `json:"mensagem"`
	EnviarEm    *time.Time `json:"enviar_em"`
	Recorrencia *string    `json:"recorrencia"`
}

type FiltroAgendamentos struct {
	InstanciaID string
	ContasIDs   []string
	SoContas    bool
	Status      string
	Limite      int
}
