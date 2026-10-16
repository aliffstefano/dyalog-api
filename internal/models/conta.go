package models

import "time"

// Papeis de usuario dentro de uma conta (empresa).
const (
	PapelDono      = "dono"
	PapelAdmin     = "admin"
	PapelAtendente = "atendente"

	ContaAtiva     = "ativa"
	ContaBloqueada = "bloqueada"
)

// PapelValido diz se o papel existe.
func PapelValido(papel string) bool {
	return papel == PapelDono || papel == PapelAdmin || papel == PapelAtendente
}

// PapelGerencia diz se o papel pode configurar instancias, webhooks e equipe.
func PapelGerencia(papel string) bool { return papel == PapelDono || papel == PapelAdmin }

// Conta e a empresa cliente: dona das instancias e da equipe.
type Conta struct {
	ID      string `json:"id"`
	Nome    string `json:"nome"`
	PlanoID string `json:"plano_id,omitempty"`
	Status  string `json:"status"`
	// LimiteInstancias acima de 0 substitui o limite do plano.
	LimiteInstancias int       `json:"limite_instancias"`
	CriadoEm         time.Time `json:"criado_em"`
	AtualizadoEm     time.Time `json:"atualizado_em"`
}

// Plano define limites (e, depois, o preco) de uma conta.
type Plano struct {
	ID               string    `json:"id"`
	Nome             string    `json:"nome"`
	PrecoCentavos    int       `json:"preco_centavos"`
	LimiteInstancias int       `json:"limite_instancias"`
	Ativo            bool      `json:"ativo"`
	CriadoEm         time.Time `json:"criado_em"`
}

// Usuario entra no painel com e-mail e senha. Superadmin (o provedor) ve
// todas as contas; os demais so a propria conta.
type Usuario struct {
	ID            string    `json:"id"`
	ContaID       string    `json:"conta_id,omitempty"`
	Nome          string    `json:"nome"`
	Email         string    `json:"email"`
	Papel         string    `json:"papel"`
	Superadmin    bool      `json:"superadmin"`
	Ativo         bool      `json:"ativo"`
	SenhaHash     string    `json:"-"`
	CriadoEm      time.Time `json:"criado_em"`
	UltimoLoginEm time.Time `json:"ultimo_login_em"`
}

// Convite leva alguem a criar o usuario numa conta (ou como superadmin).
// So o hash do token fica no banco.
type Convite struct {
	ID         string    `json:"id"`
	ContaID    string    `json:"conta_id,omitempty"`
	Email      string    `json:"email"`
	Papel      string    `json:"papel"`
	Superadmin bool      `json:"superadmin"`
	CriadoPor  string    `json:"criado_por,omitempty"`
	ExpiraEm   time.Time `json:"expira_em"`
	CriadoEm   time.Time `json:"criado_em"`
	TokenHash  string    `json:"-"`
}

// ContaResumo e a conta com o uso e o plano, para a lista do superadmin.
type ContaResumo struct {
	Conta
	PlanoNome         string `json:"plano_nome,omitempty"`
	LimiteEfetivo     int    `json:"limite_efetivo"`
	Instancias        int    `json:"instancias"`
	Usuarios          int    `json:"usuarios"`
	ConvitesPendentes int    `json:"convites_pendentes"`
}
