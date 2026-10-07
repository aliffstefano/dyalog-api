package models

import "time"

// Tipos de instancia: "whatsapp" conecta por QR code/codigo (whatsmeow) e
// "meta" usa a API oficial (WhatsApp Cloud API).
const (
	TipoInstanciaWhatsApp = "whatsapp"
	TipoInstanciaMeta     = "meta"
)

// CredenciaisMeta sao os dados de uma instancia da API oficial. AccessToken e
// AppSecret nunca saem na resposta da API (json:"-").
type CredenciaisMeta struct {
	InstanciaID   string `json:"instancia_id"`
	PhoneNumberID string `json:"phone_number_id"`
	WABAID        string `json:"waba_id,omitempty"`
	AccessToken   string `json:"-"`
	AppSecret     string `json:"-"`
	// VerifyToken e o texto que a Meta manda na verificacao do webhook.
	VerifyToken string `json:"verify_token"`
	// NumeroExibicao e NomeVerificado vem da propria Meta ao validar.
	NumeroExibicao string    `json:"numero_exibicao,omitempty"`
	NomeVerificado string    `json:"nome_verificado,omitempty"`
	AtualizadoEm   time.Time `json:"atualizado_em"`
}

// ConfigurarMetaRequest cadastra ou troca as credenciais da API oficial.
// Campos vazios mantem o valor salvo (para trocar so o token, por exemplo).
type ConfigurarMetaRequest struct {
	PhoneNumberID string `json:"phone_number_id,omitempty"`
	WABAID        string `json:"waba_id,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`
	AppSecret     string `json:"app_secret,omitempty"`
	VerifyToken   string `json:"verify_token,omitempty"`
}

// EnvioTemplateRequest envia um modelo aprovado na Meta, o unico tipo de
// mensagem permitido fora da janela de 24h.
type EnvioTemplateRequest struct {
	Instancia string `json:"instancia,omitempty"`
	Numero    string `json:"numero,omitempty"`
	ChatJID   string `json:"chat_jid,omitempty"`
	Nome      string `json:"nome,omitempty"`
	Idioma    string `json:"idioma,omitempty"`
	// Parametros preenche as variaveis {{1}}, {{2}}... do corpo, em ordem.
	Parametros []string `json:"parametros,omitempty"`
	// Componentes e o formato cru da Meta (header, body, button), para modelos
	// com midia ou botoes. Quando informado, Parametros e ignorado.
	Componentes []map[string]interface{} `json:"componentes,omitempty"`
	MensagemID  string                   `json:"mensagem_id,omitempty"`
}
