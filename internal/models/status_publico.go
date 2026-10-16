package models

import "time"

// PaginaStatus e a configuracao da pagina publica de status de uma conta.
type PaginaStatus struct {
	ContaID           string   `json:"conta_id,omitempty"`
	Token             string   `json:"token,omitempty"`
	Titulo            string   `json:"titulo"`
	Ativa             bool     `json:"ativa"`
	InstanciasOcultas []string `json:"instancias_ocultas"`
	Link              string   `json:"link,omitempty"`
}

// StatusPublico e o que a pagina mostra (sem numero, token ou id).
type StatusPublico struct {
	Titulo       string              `json:"titulo"`
	Geral        string              `json:"geral"` // operando, parcial, fora
	Instancias   []StatusPublicoItem `json:"instancias"`
	AtualizadoEm time.Time           `json:"atualizado_em"`
}

type StatusPublicoItem struct {
	Nome            string    `json:"nome"`
	Estado          string    `json:"estado"` // operando, reconectando, fora
	Desde           time.Time `json:"desde"`
	Disponibilidade float64   `json:"disponibilidade_30d"`
	Dias            []float64 `json:"dias"` // 30 valores de 0 a 1 (ou -1 sem dados), do mais antigo ao de hoje
}
