package models

import "time"

// GrupoResumo e um grupo na lista da aba Grupos.
type GrupoResumo struct {
	JID            string    `json:"jid"`
	Nome           string    `json:"nome"`
	Descricao      string    `json:"descricao,omitempty"`
	Participantes  int       `json:"participantes"`
	SouAdmin       bool      `json:"sou_admin"`
	SoAdminsEnviam bool      `json:"so_admins_enviam"`
	SoAdminsEditam bool      `json:"so_admins_editam"`
	Comunidade     bool      `json:"comunidade"`
	CriadoEm       time.Time `json:"criado_em"`
}

// GrupoDetalhe traz os participantes.
type GrupoDetalhe struct {
	GrupoResumo
	Dono               string              `json:"dono,omitempty"`
	ListaParticipantes []ParticipanteGrupo `json:"lista_participantes"`
}

type ParticipanteGrupo struct {
	JID        string `json:"jid"`
	Numero     string `json:"numero,omitempty"`
	Nome       string `json:"nome,omitempty"`
	Admin      bool   `json:"admin"`
	SuperAdmin bool   `json:"super_admin"`
	Eu         bool   `json:"eu"`
}

type CriarGrupoRequest struct {
	Nome          string   `json:"nome"`
	Participantes []string `json:"participantes"`
}

// AtualizarGrupoRequest: campo ausente fica como esta.
type AtualizarGrupoRequest struct {
	Nome           *string `json:"nome"`
	Descricao      *string `json:"descricao"`
	SoAdminsEnviam *bool   `json:"so_admins_enviam"`
	SoAdminsEditam *bool   `json:"so_admins_editam"`
}

// ParticipantesGrupoRequest: acao adicionar, remover, promover ou rebaixar.
type ParticipantesGrupoRequest struct {
	Acao    string   `json:"acao"`
	Numeros []string `json:"numeros"`
}

type ResultadoParticipante struct {
	Numero string `json:"numero"`
	JID    string `json:"jid,omitempty"`
	Ok     bool   `json:"ok"`
	Erro   string `json:"erro,omitempty"`
}
