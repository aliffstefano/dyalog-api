package models

import "time"

// Resultados de um registro de envio.
const (
	EnvioResultadoEnviada  = "enviada"
	EnvioResultadoLimitada = "limitada"
)

// EnvioRegistro e uma linha do historico de envios, usado so para metricas de
// uso no painel.
type EnvioRegistro struct {
	InstanciaID string
	ChatJID     string
	Tipo        string
	Resultado   string
	CriadoEm    time.Time
}

// ResumoUso sao os numeros de uso de uma instancia mostrados no painel.
type ResumoUso struct {
	LimitePorMinuto  int          `json:"limite_por_minuto"`
	RetencaoDias     int          `json:"retencao_dias"`
	UltimoMinuto     int          `json:"ultimo_minuto"`
	UltimaHora       int          `json:"ultima_hora"`
	Hoje             UsoDia       `json:"hoje"`
	PicoMinutoHoje   int          `json:"pico_minuto_hoje"`
	PicoMinutoEm     *time.Time   `json:"pico_minuto_em,omitempty"`
	Rajadas          []UsoContato `json:"rajadas"`
	TopDestinatarios []UsoContato `json:"top_destinatarios"`
	Dias             []UsoDia     `json:"dias"`
}

// UsoDia resume um dia (no fuso do servidor).
type UsoDia struct {
	Dia           string `json:"dia"`
	Envios        int    `json:"envios"`
	ContatosNovos int    `json:"contatos_novos"`
	Limitados     int    `json:"limitados"`
}

// UsoContato e um destinatario com a quantidade de envios (total do dia, ou o
// maximo em um unico minuto no caso de rajadas).
type UsoContato struct {
	ChatJID string `json:"chat_jid"`
	Envios  int    `json:"envios"`
}
