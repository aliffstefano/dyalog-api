package models

import "time"

// Tipos e severidades dos registros de auditoria.
const (
	AuditoriaConexao = "conexao"
	AuditoriaAcao    = "acao"
	AuditoriaConfig  = "configuracao"
	AuditoriaWebhook = "webhook"
	AuditoriaPerfil  = "perfil"
	AuditoriaContato = "contato"
	AuditoriaAcesso  = "acesso"
	AuditoriaEquipe  = "equipe"

	SeveridadeInfo    = "info"
	SeveridadeSucesso = "sucesso"
	SeveridadeAviso   = "aviso"
	SeveridadeErro    = "erro"
)

// RegistroAuditoria e uma linha do historico "quem fez o que".
type RegistroAuditoria struct {
	ID            string    `json:"id"`
	InstanciaID   string    `json:"instancia_id,omitempty"`
	InstanciaNome string    `json:"instancia_nome,omitempty"`
	ContaID       string    `json:"conta_id,omitempty"`
	Tipo          string    `json:"tipo"`
	Acao          string    `json:"acao"`
	Descricao     string    `json:"descricao"`
	Severidade    string    `json:"severidade"`
	AtorTipo      string    `json:"ator_tipo"` // usuario, master, token, sistema
	AtorID        string    `json:"ator_id,omitempty"`
	AtorNome      string    `json:"ator_nome"`
	IP            string    `json:"ip,omitempty"`
	Detalhes      string    `json:"detalhes,omitempty"`
	CriadoEm      time.Time `json:"criado_em"`
}

// FiltroAuditoria: campos vazios nao filtram. ContasIDs limita a contas
// (vazio = todas, para quem enxerga tudo).
type FiltroAuditoria struct {
	InstanciaID  string
	ContasIDs    []string
	SoContas     bool
	Tipo         string
	Severidade   string
	Ator         string
	De           time.Time
	Ate          time.Time
	Limite       int
	Deslocamento int
}

type PaginaAuditoria struct {
	Registros []RegistroAuditoria `json:"registros"`
	TemMais   bool                `json:"tem_mais"`
}
