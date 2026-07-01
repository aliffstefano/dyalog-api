package models

import "time"

type IniciarChamadaRequest struct {
	Instancia string `json:"instancia,omitempty"`
	Numero    string `json:"numero,omitempty"`
	ChatJID   string `json:"chat_jid,omitempty"`
	Video     bool   `json:"video,omitempty"`
}

type SinalizacaoWebRTCRequest struct {
	Instancia string `json:"instancia,omitempty"`
	ChamadaID string `json:"chamada_id,omitempty"`
	SDPOffer  string `json:"sdp_offer" binding:"required"`
}

type AcaoChamadaRequest struct {
	Instancia string `json:"instancia,omitempty"`
	ChamadaID string `json:"chamada_id,omitempty"`
	Motivo    string `json:"motivo,omitempty"`
}

type ResultadoChamada struct {
	Instancia string    `json:"instancia"`
	ChamadaID string    `json:"chamada_id"`
	PeerJID   string    `json:"peer_jid,omitempty"`
	Numero    string    `json:"numero,omitempty"`
	Direcao   string    `json:"direcao,omitempty"`
	Estado    string    `json:"estado"`
	Tipo      string    `json:"tipo"`
	CriadaEm  time.Time `json:"criada_em"`
}

type ResultadoWebRTC struct {
	Instancia    string `json:"instancia"`
	ChamadaID    string `json:"chamada_id"`
	SDPAnswer    string `json:"sdp_answer"`
	Transporte   string `json:"transporte,omitempty"`
	AudioEnvio   string `json:"audio_envio,omitempty"`
	AudioRetorno string `json:"audio_retorno,omitempty"`
}
