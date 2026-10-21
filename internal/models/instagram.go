package models

import "time"

// TipoInstanciaInstagram usa a API do Instagram com login do Instagram (Direct
// da conta profissional).
const TipoInstanciaInstagram = "instagram"

// CredenciaisInstagram sao os dados de uma instancia do Instagram. ContaID e o
// id da conta profissional (user_id da Graph API do Instagram): e o id que
// chega nos webhooks (entry.id e recipient.id).
type CredenciaisInstagram struct {
	InstanciaID string `json:"instancia_id"`
	ContaID     string `json:"conta_id"`
	Usuario     string `json:"usuario,omitempty"`
	Nome        string `json:"nome,omitempty"`
	FotoURL     string `json:"foto_url,omitempty"`
	// AccessToken e o token de longa duracao (60 dias), renovado sozinho.
	AccessToken  string    `json:"-"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

// ConfigurarInstagramRequest cadastra ou troca o token da conta do Instagram.
type ConfigurarInstagramRequest struct {
	AccessToken string `json:"access_token,omitempty"`
}
