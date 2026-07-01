package models

import "time"

type MensagemProcessada struct {
	InstanciaID   string    `json:"instancia_id"`
	ChatJID       string    `json:"chat_jid"`
	MensagemID    string    `json:"mensagem_id"`
	RemetenteJID  string    `json:"remetente_jid"`
	EnviadaPorMim bool      `json:"enviada_por_mim"`
	Grupo         bool      `json:"grupo"`
	RecebidaEm    time.Time `json:"recebida_em"`
	Origem        string    `json:"origem"`
	ProcessadaEm  time.Time `json:"processada_em"`
}
