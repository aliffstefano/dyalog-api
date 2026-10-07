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
	Tipo                     string    `json:"tipo" db:"tipo"`
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
	// Perfil so vem preenchido pela replica que mantem a instancia conectada.
	Perfil *PerfilInstancia `json:"perfil,omitempty" db:"-"`
}

// EhMeta indica instancia da API oficial. Instancias antigas, sem tipo, sao
// whatsapp.
func (i Instancia) EhMeta() bool { return i.Tipo == TipoInstanciaMeta }

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

// PerfilInstancia e o proprio numero conectado na instancia, para o painel.
type PerfilInstancia struct {
	Numero      string `json:"numero,omitempty"`
	Nome        string `json:"nome,omitempty"`
	NomeEmpresa string `json:"nome_empresa,omitempty"`
	Plataforma  string `json:"plataforma,omitempty"`
	Business    bool   `json:"business"`
	FotoURL     string `json:"foto_url,omitempty"`
}
