package models

import "time"

// RespostaRapida e um texto pronto do chat, usado digitando /atalho.
type RespostaRapida struct {
	ID           string    `json:"id"`
	ContaID      string    `json:"conta_id,omitempty"`
	Atalho       string    `json:"atalho"`
	Texto        string    `json:"texto"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

// Etiqueta marca conversas do chat.
type Etiqueta struct {
	ID      string `json:"id"`
	ContaID string `json:"conta_id,omitempty"`
	Nome    string `json:"nome"`
	Cor     string `json:"cor"`
}

type DefinirEtiquetasChatRequest struct {
	ChatJID   string   `json:"chat_jid"`
	Etiquetas []string `json:"etiquetas"`
}
