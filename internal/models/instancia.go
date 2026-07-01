package models

import "time"

const (
	StatusInstanciaDesconectada     = "desconectada"
	StatusInstanciaConectando       = "conectando"
	StatusInstanciaAguardandoQR     = "aguardando_qrcode"
	StatusInstanciaAguardandoCodigo = "aguardando_codigo"
	StatusInstanciaConectada        = "conectada"
	StatusInstanciaDesconectando    = "desconectando"
	StatusInstanciaNaoInicializada  = "nao_inicializada"

	ProxyModoHerdar  = "herdar"
	ProxyModoProprio = "proprio"
	ProxyModoDireto  = "direto"

	PresencaDisponivel   = "disponivel"
	PresencaIndisponivel = "indisponivel"
)

type Instancia struct {
	ID                       string    `json:"id" db:"id"`
	Nome                     string    `json:"nome" db:"nome"`
	Token                    string    `json:"token,omitempty" db:"token"`
	Status                   string    `json:"status" db:"status"`
	HistoricoDias            int       `json:"historico_dias" db:"historico_dias"`
	ProxyModo                string    `json:"proxy_modo" db:"proxy_modo"`
	ProxyURL                 string    `json:"proxy_url,omitempty" db:"proxy_url"`
	Presenca                 string    `json:"presenca" db:"presenca"`
	RejeitarChamadas         bool      `json:"rejeitar_chamadas" db:"rejeitar_chamadas"`
	MensagemRejeitarChamadas string    `json:"mensagem_rejeitar_chamadas,omitempty" db:"mensagem_rejeitar_chamadas"`
	MarcarLidaAutomatico     bool      `json:"marcar_lida_automatico" db:"marcar_lida_automatico"`
	IgnorarGrupos            bool      `json:"ignorar_grupos" db:"ignorar_grupos"`
	IgnorarStatus            bool      `json:"ignorar_status" db:"ignorar_status"`
	CriadoEm                 time.Time `json:"criado_em" db:"criado_em"`
	AtualizadoEm             time.Time `json:"atualizado_em" db:"atualizado_em"`
}

type ConfiguracaoAvancadaInstancia struct {
	ManterOnline             bool   `json:"manter_online"`
	RejeitarChamadas         bool   `json:"rejeitar_chamadas"`
	MensagemRejeitarChamadas string `json:"mensagem_rejeitar_chamadas,omitempty"`
	MarcarLidaAutomatico     bool   `json:"marcar_lida_automatico"`
	IgnorarGrupos            bool   `json:"ignorar_grupos"`
	IgnorarStatus            bool   `json:"ignorar_status"`
}

type ProxyGlobal struct {
	URL          string    `json:"url,omitempty"`
	Ativo        bool      `json:"ativo"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}

type AcessoDashboard struct {
	Tipo        string `json:"tipo"`
	InstanciaID string `json:"instancia_id,omitempty"`
	Nome        string `json:"nome,omitempty"`
}
