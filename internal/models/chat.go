package models

import "time"

// ChatMensagem e uma mensagem guardada para a tela de Chat do painel:
// recebidas (pelos mesmos eventos dos webhooks) e enviadas pela API.
type ChatMensagem struct {
	InstanciaID  string    `json:"instancia_id"`
	ChatJID      string    `json:"chat_jid"`
	MensagemID   string    `json:"mensagem_id"`
	Direcao      string    `json:"direcao"`
	Tipo         string    `json:"tipo"`
	Conteudo     string    `json:"conteudo"`
	Nome         string    `json:"nome,omitempty"`
	RemetenteJID string    `json:"remetente_jid,omitempty"`
	Status       string    `json:"status,omitempty"`
	Erro         string    `json:"erro,omitempty"`
	MidiaID      string    `json:"midia_id,omitempty"`
	MimeType     string    `json:"mime_type,omitempty"`
	NomeArquivo  string    `json:"nome_arquivo,omitempty"`
	CriadoEm     time.Time `json:"criado_em"`
}

// ChatConversa resume uma conversa: o contato e a ultima mensagem.
type ChatConversa struct {
	ChatJID string       `json:"chat_jid"`
	Nome    string       `json:"nome,omitempty"`
	Grupo   bool         `json:"grupo"`
	Ultima  ChatMensagem `json:"ultima"`
}
