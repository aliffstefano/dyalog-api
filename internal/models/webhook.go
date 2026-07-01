package models

import (
	"encoding/json"
	"time"
)

const (
	EventoWebhookMensagens     = "mensagens"
	EventoWebhookStatus        = "status"
	EventoWebhookDigitando     = "digitando"
	EventoWebhookGravandoAudio = "gravando_audio"
	EventoWebhookRecibos       = "recibos"
	EventoWebhookChamadas      = "chamadas"
)

var EventosWebhookSuportados = map[string]struct{}{
	EventoWebhookMensagens:     {},
	EventoWebhookStatus:        {},
	EventoWebhookDigitando:     {},
	EventoWebhookGravandoAudio: {},
	EventoWebhookRecibos:       {},
	EventoWebhookChamadas:      {},
}

type WebhookInstancia struct {
	ID           string    `json:"id"`
	InstanciaID  string    `json:"instancia_id"`
	Nome         string    `json:"nome"`
	URL          string    `json:"url"`
	Eventos      []string  `json:"eventos"`
	Ativo        bool      `json:"ativo"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

type EventoWebhook struct {
	Evento      string      `json:"evento"`
	InstanciaID string      `json:"instancia_id"`
	OcorridoEm  time.Time   `json:"ocorrido_em"`
	Dados       interface{} `json:"dados"`
}

const (
	WebhookEntregaPendente = "pendente"
	WebhookEntregaEnviando = "enviando"
	WebhookEntregaEntregue = "entregue"
	WebhookEntregaFalha    = "falha"
	WebhookEntregaEsgotada = "esgotada"
)

type WebhookEntrega struct {
	ID                 string          `json:"id"`
	WebhookID          string          `json:"webhook_id"`
	InstanciaID        string          `json:"instancia_id"`
	WebhookNome        string          `json:"webhook_nome"`
	URL                string          `json:"url"`
	Evento             string          `json:"evento"`
	Payload            json.RawMessage `json:"payload,omitempty"`
	Status             string          `json:"status"`
	Tentativas         int             `json:"tentativas"`
	MaxTentativas      int             `json:"max_tentativas"`
	ProximaTentativaEm time.Time       `json:"proxima_tentativa_em"`
	UltimaTentativaEm  *time.Time      `json:"ultima_tentativa_em,omitempty"`
	StatusHTTP         int             `json:"status_http,omitempty"`
	UltimoErro         string          `json:"ultimo_erro,omitempty"`
	CriadoEm           time.Time       `json:"criado_em"`
	AtualizadoEm       time.Time       `json:"atualizado_em"`
}
