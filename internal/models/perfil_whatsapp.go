package models

// PerfilWhatsApp e a identidade e a privacidade da conta conectada numa
// instancia (aba Perfil do painel).
type PerfilWhatsApp struct {
	Instancia     string              `json:"instancia"`
	Numero        string              `json:"numero"`
	JID           string              `json:"jid"`
	LID           string              `json:"lid,omitempty"`
	Nome          string              `json:"nome"`
	Recado        string              `json:"recado"`
	FotoURL       string              `json:"foto_url,omitempty"`
	Business      bool                `json:"business"`
	NomeEmpresa   string              `json:"nome_empresa,omitempty"`
	Plataforma    string              `json:"plataforma,omitempty"`
	Privacidade   PrivacidadeWhatsApp `json:"privacidade"`
	OpcoesValidas map[string][]string `json:"opcoes_validas"`
}

// PrivacidadeWhatsApp usa os valores do WhatsApp: all, contacts,
// contact_blacklist, none, match_last_seen, known.
type PrivacidadeWhatsApp struct {
	VistoPorUltimo     string `json:"visto_por_ultimo"`
	FotoPerfil         string `json:"foto_perfil"`
	Recado             string `json:"recado"`
	Online             string `json:"online"`
	Chamadas           string `json:"chamadas"`
	Grupos             string `json:"grupos"`
	ConfirmacaoLeitura string `json:"confirmacao_leitura"`
}

// AtualizarPerfilRequest troca nome e/ou recado; campo ausente fica como esta.
type AtualizarPerfilRequest struct {
	Nome   *string `json:"nome"`
	Recado *string `json:"recado"`
}

// AtualizarFotoPerfilRequest recebe a imagem em base64 (com ou sem o prefixo
// data:) ou uma URL publica.
type AtualizarFotoPerfilRequest struct {
	Imagem string `json:"imagem"`
	URL    string `json:"url"`
}

// ContatoWhatsApp e um contato salvo ou um grupo da instancia.
type ContatoWhatsApp struct {
	JID       string `json:"jid"`
	Numero    string `json:"numero,omitempty"`
	Nome      string `json:"nome"`
	NomePush  string `json:"nome_push,omitempty"`
	Empresa   string `json:"empresa,omitempty"`
	Tipo      string `json:"tipo"` // pessoal, grupo
	Salvo     bool   `json:"salvo"`
	Bloqueado bool   `json:"bloqueado"`
}

type ListaContatos struct {
	Contatos  []ContatoWhatsApp `json:"contatos"`
	Total     int               `json:"total"`
	Pagina    int               `json:"pagina"`
	PorPagina int               `json:"por_pagina"`
	Paginas   int               `json:"paginas"`
	Contagem  map[string]int    `json:"contagem"`
}

type VerificarNumerosRequest struct {
	Numeros []string `json:"numeros"`
}

type NumeroVerificado struct {
	Numero     string `json:"numero"`
	ExisteNoWA bool   `json:"existe"`
	JID        string `json:"jid,omitempty"`
	Business   bool   `json:"business"`
	Nome       string `json:"nome,omitempty"`
}

type BloqueioRequest struct {
	Numero   string `json:"numero"`
	ChatJID  string `json:"chat_jid"`
	Bloquear bool   `json:"bloquear"`
}
