package models

import "time"

type EnvioTextoRequest struct {
	Instancia            string             `json:"instancia,omitempty"`
	Numero               string             `json:"numero,omitempty"`
	ChatJID              string             `json:"chat_jid,omitempty"`
	Grupo                bool               `json:"grupo,omitempty"`
	Mensagem             string             `json:"mensagem,omitempty"`
	Delay                int                `json:"delay,omitempty"`
	DelaySegundos        int                `json:"delay_segundos,omitempty"`
	DelayMS              int                `json:"delay_ms,omitempty"`
	Digitando            *bool              `json:"digitando,omitempty"`
	MensagemID           string             `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string             `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string             `json:"resposta_participante,omitempty"`
	RespostaConteudo     string             `json:"resposta_conteudo,omitempty"`
	Phone                string             `json:"Phone,omitempty"`
	Body                 string             `json:"Body,omitempty"`
	ID                   string             `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat `json:"ContextInfo,omitempty"`
}

type ContextInfoCompat struct {
	StanzaID    string `json:"StanzaId,omitempty"`
	Participant string `json:"Participant,omitempty"`
}

type EnvioMidiaRequest struct {
	Instancia       string `json:"instancia,omitempty"`
	Numero          string `json:"numero"`
	ChatJID         string `json:"chat_jid,omitempty"`
	Grupo           bool   `json:"grupo,omitempty"`
	ArquivoURL      string `json:"arquivo_url,omitempty"`
	ArquivoBase64   string `json:"arquivo_base64,omitempty"`
	CaminhoLocal    string `json:"caminho_local,omitempty"`
	Legenda         string `json:"legenda,omitempty"`
	NomeArquivo     string `json:"nome_arquivo,omitempty"`
	MimeType        string `json:"mime_type,omitempty"`
	DuracaoSegundos uint32 `json:"duracao_segundos,omitempty"`
	PTT             *bool  `json:"ptt,omitempty"`
	MensagemID      string `json:"mensagem_id,omitempty"`
}

type EnvioPresencaRequest struct {
	Instancia     string `json:"instancia,omitempty"`
	Numero        string `json:"numero,omitempty"`
	ChatJID       string `json:"chat_jid,omitempty"`
	Grupo         bool   `json:"grupo,omitempty"`
	Acao          string `json:"acao,omitempty"`
	Type          string `json:"type,omitempty"`
	Delay         int    `json:"delay,omitempty"`
	DelaySegundos int    `json:"delay_segundos,omitempty"`
	DelayMS       int    `json:"delay_ms,omitempty"`
}

type MarcarLidaRequest struct {
	Instancia     string    `json:"instancia,omitempty"`
	Numero        string    `json:"numero,omitempty"`
	ChatJID       string    `json:"chat_jid,omitempty"`
	Grupo         bool      `json:"grupo,omitempty"`
	MensagemID    string    `json:"mensagem_id,omitempty"`
	MensagensID   []string  `json:"mensagens_id,omitempty"`
	Participante  string    `json:"participante,omitempty"`
	RemetenteJID  string    `json:"remetente_jid,omitempty"`
	LidaEm        string    `json:"lida_em,omitempty"`
	Phone         string    `json:"Phone,omitempty"`
	ID            string    `json:"Id,omitempty"`
	IDs           []string  `json:"Ids,omitempty"`
	Participant   string    `json:"Participant,omitempty"`
	Timestamp     string    `json:"Timestamp,omitempty"`
	MarcadaEmTime time.Time `json:"-"`
}

type BotaoRequest struct {
	ID          string `json:"id,omitempty"`
	Texto       string `json:"texto,omitempty"`
	Tipo        string `json:"tipo,omitempty"`
	DisplayText string `json:"DisplayText,omitempty"`
	Type        string `json:"Type,omitempty"`
	URL         string `json:"URL,omitempty"`
	Url         string `json:"Url,omitempty"`
	PhoneNumber string `json:"PhoneNumber,omitempty"`
}

type EnvioBotoesRequest struct {
	Instancia            string             `json:"instancia,omitempty"`
	Numero               string             `json:"numero,omitempty"`
	ChatJID              string             `json:"chat_jid,omitempty"`
	Grupo                bool               `json:"grupo,omitempty"`
	Titulo               string             `json:"titulo,omitempty"`
	Texto                string             `json:"texto,omitempty"`
	Mensagem             string             `json:"mensagem,omitempty"`
	Rodape               string             `json:"rodape,omitempty"`
	Botoes               []BotaoRequest     `json:"botoes,omitempty"`
	Modo                 string             `json:"modo,omitempty"`
	FallbackTexto        bool               `json:"fallback_texto,omitempty"`
	MensagemID           string             `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string             `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string             `json:"resposta_participante,omitempty"`
	Phone                string             `json:"Phone,omitempty"`
	Content              string             `json:"Content,omitempty"`
	Footer               string             `json:"Footer,omitempty"`
	Buttons              []BotaoRequest     `json:"Buttons,omitempty"`
	ID                   string             `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat `json:"ContextInfo,omitempty"`
}

type ListaLinhaRequest struct {
	ID        string `json:"id,omitempty"`
	Titulo    string `json:"titulo,omitempty"`
	Descricao string `json:"descricao,omitempty"`
	RowID     string `json:"RowId,omitempty"`
	Title     string `json:"title,omitempty"`
	Desc      string `json:"desc,omitempty"`
}

type ListaSecaoRequest struct {
	Titulo string              `json:"titulo,omitempty"`
	Linhas []ListaLinhaRequest `json:"linhas,omitempty"`
	Title  string              `json:"title,omitempty"`
	Rows   []ListaLinhaRequest `json:"rows,omitempty"`
}

type EnvioListaRequest struct {
	Instancia            string              `json:"instancia,omitempty"`
	Numero               string              `json:"numero,omitempty"`
	ChatJID              string              `json:"chat_jid,omitempty"`
	Grupo                bool                `json:"grupo,omitempty"`
	Titulo               string              `json:"titulo,omitempty"`
	Descricao            string              `json:"descricao,omitempty"`
	Mensagem             string              `json:"mensagem,omitempty"`
	BotaoTexto           string              `json:"botao_texto,omitempty"`
	Rodape               string              `json:"rodape,omitempty"`
	Secoes               []ListaSecaoRequest `json:"secoes,omitempty"`
	Opcoes               []ListaLinhaRequest `json:"opcoes,omitempty"`
	Modo                 string              `json:"modo,omitempty"`
	FallbackTexto        bool                `json:"fallback_texto,omitempty"`
	MensagemID           string              `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string              `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string              `json:"resposta_participante,omitempty"`
	Phone                string              `json:"Phone,omitempty"`
	ButtonText           string              `json:"ButtonText,omitempty"`
	Desc                 string              `json:"Desc,omitempty"`
	TopText              string              `json:"TopText,omitempty"`
	FooterText           string              `json:"FooterText,omitempty"`
	List                 []ListaLinhaRequest `json:"List,omitempty"`
	ID                   string              `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat  `json:"ContextInfo,omitempty"`
}

type ResultadoEnvio struct {
	Instancia     string `json:"instancia"`
	Numero        string `json:"numero,omitempty"`
	ChatJID       string `json:"chat_jid,omitempty"`
	MensagemID    string `json:"mensagem_id,omitempty"`
	Status        string `json:"status"`
	Tipo          string `json:"tipo"`
	Modo          string `json:"modo,omitempty"`
	DelaySegundos int    `json:"delay_segundos,omitempty"`
	PresencaAntes string `json:"presenca_antes,omitempty"`
	Observacao    string `json:"observacao,omitempty"`
}

type ResultadoPresenca struct {
	Instancia            string `json:"instancia"`
	Numero               string `json:"numero,omitempty"`
	ChatJID              string `json:"chat_jid,omitempty"`
	Status               string `json:"status"`
	Tipo                 string `json:"tipo"`
	Acao                 string `json:"acao"`
	DelaySegundos        int    `json:"delay_segundos,omitempty"`
	FinalizadaComPausado bool   `json:"finalizada_com_pausado,omitempty"`
}

type ResultadoMarcarLida struct {
	Instancia    string    `json:"instancia"`
	Numero       string    `json:"numero,omitempty"`
	ChatJID      string    `json:"chat_jid,omitempty"`
	MensagensID  []string  `json:"mensagens_id"`
	Participante string    `json:"participante,omitempty"`
	Status       string    `json:"status"`
	Tipo         string    `json:"tipo"`
	LidaEm       time.Time `json:"lida_em"`
}
