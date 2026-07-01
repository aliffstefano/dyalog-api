package models

import "time"

type MensagemRecebida struct {
	ID            string    `json:"id"`
	InstanciaID   string    `json:"instancia_id"`
	ChatJID       string    `json:"chat_jid"`
	Remetente     string    `json:"remetente"`
	NomeRemetente string    `json:"nome_remetente"`
	Conteudo      string    `json:"conteudo"`
	Tipo          string    `json:"tipo"`
	RecebidaEm    time.Time `json:"recebida_em"`
}
