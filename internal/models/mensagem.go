package models

import (
	"strings"
	"time"
)

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

type EditarTextoRequest struct {
	Instancia  string `json:"instancia,omitempty"`
	Numero     string `json:"numero,omitempty"`
	ChatJID    string `json:"chat_jid,omitempty"`
	Grupo      bool   `json:"grupo,omitempty"`
	MensagemID string `json:"mensagem_id,omitempty"`
	Mensagem   string `json:"mensagem,omitempty"`
	Phone      string `json:"Phone,omitempty"`
	Body       string `json:"Body,omitempty"`
	ID         string `json:"Id,omitempty"`
}

type ApagarMensagemRequest struct {
	Instancia    string `json:"instancia,omitempty"`
	Numero       string `json:"numero,omitempty"`
	ChatJID      string `json:"chat_jid,omitempty"`
	Grupo        bool   `json:"grupo,omitempty"`
	MensagemID   string `json:"mensagem_id,omitempty"`
	RemetenteJID string `json:"remetente_jid,omitempty"`
	Participante string `json:"participante,omitempty"`
	Phone        string `json:"Phone,omitempty"`
	ID           string `json:"Id,omitempty"`
	Participant  string `json:"Participant,omitempty"`
}

type ReagirMensagemRequest struct {
	Instancia    string `json:"instancia,omitempty"`
	Numero       string `json:"numero,omitempty"`
	ChatJID      string `json:"chat_jid,omitempty"`
	Grupo        bool   `json:"grupo,omitempty"`
	MensagemID   string `json:"mensagem_id,omitempty"`
	Emoji        string `json:"emoji,omitempty"`
	RemetenteJID string `json:"remetente_jid,omitempty"`
	Participante string `json:"participante,omitempty"`
	Phone        string `json:"Phone,omitempty"`
	ID           string `json:"Id,omitempty"`
	Reaction     string `json:"Reaction,omitempty"`
	Participant  string `json:"Participant,omitempty"`
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
	Telefone    string `json:"telefone,omitempty"`
	// Codigo e o texto copiado pelo botao do tipo copiar (cupom, Pix copia e
	// cola, codigo de rastreio).
	Codigo string `json:"codigo,omitempty"`
}

// Tipos de botao normalizados por TipoNormalizado.
const (
	BotaoResposta = "reply"
	BotaoURL      = "url"
	BotaoLigar    = "call"
	BotaoCopiar   = "copy"
)

// TipoNormalizado devolve reply, url, call ou copy. Devolve "" para tipo
// desconhecido.
func (b BotaoRequest) TipoNormalizado() string {
	tipo := strings.TrimSpace(b.Tipo)
	if tipo == "" {
		tipo = strings.TrimSpace(b.Type)
	}
	switch strings.ToLower(tipo) {
	case "", "quickreply", "quick_reply", "reply", "resposta":
		return BotaoResposta
	case "url", "link", "cta_url":
		return BotaoURL
	case "call", "ligar", "ligacao", "cta_call":
		return BotaoLigar
	case "copy", "copiar", "cta_copy":
		return BotaoCopiar
	default:
		return ""
	}
}

// Link devolve a URL do botao, aceitando URL ou Url.
func (b BotaoRequest) Link() string {
	if url := strings.TrimSpace(b.URL); url != "" {
		return url
	}
	return strings.TrimSpace(b.Url)
}

// NumeroLigar devolve o telefone do botao de ligar, aceitando telefone ou
// PhoneNumber.
func (b BotaoRequest) NumeroLigar() string {
	if telefone := strings.TrimSpace(b.Telefone); telefone != "" {
		return telefone
	}
	return strings.TrimSpace(b.PhoneNumber)
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
	// Description e o nome usado por outras APIs (formato sections/rows).
	Description string `json:"description,omitempty"`
}

type ListaSecaoRequest struct {
	Titulo string              `json:"titulo,omitempty"`
	Linhas []ListaLinhaRequest `json:"linhas,omitempty"`
	Title  string              `json:"title,omitempty"`
	Rows   []ListaLinhaRequest `json:"rows,omitempty"`
}

type EnvioListaRequest struct {
	Instancia  string              `json:"instancia,omitempty"`
	Numero     string              `json:"numero,omitempty"`
	ChatJID    string              `json:"chat_jid,omitempty"`
	Grupo      bool                `json:"grupo,omitempty"`
	Titulo     string              `json:"titulo,omitempty"`
	Descricao  string              `json:"descricao,omitempty"`
	Mensagem   string              `json:"mensagem,omitempty"`
	BotaoTexto string              `json:"botao_texto,omitempty"`
	Rodape     string              `json:"rodape,omitempty"`
	Secoes     []ListaSecaoRequest `json:"secoes,omitempty"`
	Opcoes     []ListaLinhaRequest `json:"opcoes,omitempty"`
	Modo       string              `json:"modo,omitempty"`
	// FlowName sobrescreve o atributo name do no <native_flow> quando modo for
	// native_flow. Existe para experimentacao enquanto o valor aceito pelo
	// servidor para menus nao esta confirmado.
	FlowName             string              `json:"flow_name,omitempty"`
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
	// Formato usado por outras APIs: number, contentText, buttonText,
	// footerText e sections[].rows[] com id/title/description.
	Number      string              `json:"number,omitempty"`
	ContentText string              `json:"contentText,omitempty"`
	Sections    []ListaSecaoRequest `json:"sections,omitempty"`
}

type EnvioLocalizacaoRequest struct {
	Instancia            string             `json:"instancia,omitempty"`
	Numero               string             `json:"numero,omitempty"`
	ChatJID              string             `json:"chat_jid,omitempty"`
	Grupo                bool               `json:"grupo,omitempty"`
	Latitude             float64            `json:"latitude,omitempty"`
	Longitude            float64            `json:"longitude,omitempty"`
	Nome                 string             `json:"nome,omitempty"`
	Endereco             string             `json:"endereco,omitempty"`
	MensagemID           string             `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string             `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string             `json:"resposta_participante,omitempty"`
	Phone                string             `json:"Phone,omitempty"`
	LatitudeCompat       float64            `json:"Latitude,omitempty"`
	LongitudeCompat      float64            `json:"Longitude,omitempty"`
	Name                 string             `json:"Name,omitempty"`
	Address              string             `json:"Address,omitempty"`
	ID                   string             `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat `json:"ContextInfo,omitempty"`
}

type ContatoEnvioRequest struct {
	Nome        string `json:"nome,omitempty"`
	Telefone    string `json:"telefone,omitempty"`
	Organizacao string `json:"organizacao,omitempty"`
	VCard       string `json:"vcard,omitempty"`
}

type EnvioContatoRequest struct {
	Instancia            string                `json:"instancia,omitempty"`
	Numero               string                `json:"numero,omitempty"`
	ChatJID              string                `json:"chat_jid,omitempty"`
	Grupo                bool                  `json:"grupo,omitempty"`
	Nome                 string                `json:"nome,omitempty"`
	Telefone             string                `json:"telefone,omitempty"`
	Organizacao          string                `json:"organizacao,omitempty"`
	VCard                string                `json:"vcard,omitempty"`
	Contatos             []ContatoEnvioRequest `json:"contatos,omitempty"`
	MensagemID           string                `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string                `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string                `json:"resposta_participante,omitempty"`
	Phone                string                `json:"Phone,omitempty"`
	Name                 string                `json:"Name,omitempty"`
	Vcard                string                `json:"Vcard,omitempty"`
	Contacts             []ContatoEnvioRequest `json:"Contacts,omitempty"`
	ID                   string                `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat    `json:"ContextInfo,omitempty"`
}

type EnvioCobrancaPixRequest struct {
	Instancia        string  `json:"instancia,omitempty"`
	Numero           string  `json:"numero,omitempty"`
	ChatJID          string  `json:"chat_jid,omitempty"`
	Grupo            bool    `json:"grupo,omitempty"`
	ChavePix         string  `json:"chave_pix,omitempty"`
	TipoChave        string  `json:"tipo_chave,omitempty"`
	NomeBeneficiario string  `json:"nome_beneficiario,omitempty"`
	Valor            float64 `json:"valor,omitempty"`
	Referencia       string  `json:"referencia,omitempty"`
	Descricao        string  `json:"descricao,omitempty"`
	// FlowName sobrescreve o atributo name do no <native_flow> no stanza. Existe
	// para experimentacao: o WhatsApp valida esse valor e o formato aceito para
	// pagamento ainda nao esta confirmado. Quando vazio, a API decide sozinha.
	FlowName             string             `json:"flow_name,omitempty"`
	FallbackTexto        bool               `json:"fallback_texto,omitempty"`
	MensagemID           string             `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string             `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string             `json:"resposta_participante,omitempty"`
	Phone                string             `json:"Phone,omitempty"`
	ID                   string             `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat `json:"ContextInfo,omitempty"`
}

type EnvioEnqueteRequest struct {
	Instancia            string             `json:"instancia,omitempty"`
	Numero               string             `json:"numero,omitempty"`
	ChatJID              string             `json:"chat_jid,omitempty"`
	Grupo                bool               `json:"grupo,omitempty"`
	Nome                 string             `json:"nome,omitempty"`
	Pergunta             string             `json:"pergunta,omitempty"`
	Opcoes               []string           `json:"opcoes,omitempty"`
	SelecaoMultipla      bool               `json:"selecao_multipla,omitempty"`
	OpcoesSelecionaveis  int                `json:"opcoes_selecionaveis,omitempty"`
	MensagemID           string             `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string             `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string             `json:"resposta_participante,omitempty"`
	Phone                string             `json:"Phone,omitempty"`
	Name                 string             `json:"Name,omitempty"`
	Options              []string           `json:"Options,omitempty"`
	ID                   string             `json:"Id,omitempty"`
	ContextInfo          *ContextInfoCompat `json:"ContextInfo,omitempty"`
}

type EnvioEventoRequest struct {
	Instancia string `json:"instancia,omitempty"`
	Numero    string `json:"numero,omitempty"`
	ChatJID   string `json:"chat_jid,omitempty"`
	Grupo     bool   `json:"grupo,omitempty"`
	Nome      string `json:"nome,omitempty"`
	Descricao string `json:"descricao,omitempty"`
	// Inicio e Fim aceitam RFC3339 (2026-10-20T19:00:00-04:00) ou
	// "2026-10-20 19:00" no fuso do servidor.
	Inicio string `json:"inicio,omitempty"`
	Fim    string `json:"fim,omitempty"`
	// Local e opcional: nome e endereco aparecem no cartao; latitude e
	// longitude abrem o mapa.
	Local     string  `json:"local,omitempty"`
	Endereco  string  `json:"endereco,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	// LinkChamada vira o botao "Entrar" do evento (link de chamada do WhatsApp,
	// Meet, Zoom...).
	LinkChamada           string `json:"link_chamada,omitempty"`
	PermitirAcompanhantes bool   `json:"permitir_acompanhantes,omitempty"`
	// Lembrete avisa os participantes LembreteMinutos antes do inicio (15 se
	// nao informado).
	Lembrete        bool `json:"lembrete,omitempty"`
	LembreteMinutos int  `json:"lembrete_minutos,omitempty"`
	// Chamada ("video" ou "voz") gera um link de chamada do WhatsApp preso ao
	// horario do evento, como no app. Ignorado se LinkChamada vier preenchido.
	Chamada string `json:"chamada,omitempty"`
	// ChamadaAgendada marca o evento como chamada agendada do WhatsApp.
	ChamadaAgendada      bool   `json:"chamada_agendada,omitempty"`
	MensagemID           string `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string `json:"resposta_participante,omitempty"`

	// Formato de outras APIs (number, name, startAt, location{}...).
	Number            string              `json:"number,omitempty"`
	Name              string              `json:"name,omitempty"`
	Description       string              `json:"description,omitempty"`
	StartAt           string              `json:"startAt,omitempty"`
	EndAt             string              `json:"endAt,omitempty"`
	Location          *EventoLocalRequest `json:"location,omitempty"`
	JoinLink          string              `json:"joinLink,omitempty"`
	HasReminder       bool                `json:"hasReminder,omitempty"`
	ReminderOffsetSec int64               `json:"reminderOffsetSec,omitempty"`
	IsScheduleCall    bool                `json:"isScheduleCall,omitempty"`
	Call              string              `json:"call,omitempty"`
	ExtraGuests       bool                `json:"extraGuestsAllowed,omitempty"`

	InicioEm         time.Time `json:"-"`
	FimEm            time.Time `json:"-"`
	LembreteSegundos int64     `json:"-"`
}

type LinkChamadaRequest struct {
	Instancia string `json:"instancia,omitempty"`
	// Tipo e "video" ou "voz".
	Tipo string `json:"tipo,omitempty"`
}

type LinkChamadaResultado struct {
	Instancia string `json:"instancia"`
	Tipo      string `json:"tipo"`
	Link      string `json:"link"`
}

type EventoLocalRequest struct {
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

type EnvioStatusRequest struct {
	Instancia string `json:"instancia,omitempty"`
	// Tipo e texto, imagem ou video. Vazio decide pelo que foi enviado: com
	// arquivo vira imagem/video, sem arquivo vira texto.
	Tipo  string `json:"tipo,omitempty"`
	Texto string `json:"texto,omitempty"`
	// CorFundo e a cor do status de texto, em #RRGGBB.
	CorFundo string `json:"cor_fundo,omitempty"`
	// Fonte e o estilo de letra do status de texto (0 a 9).
	Fonte         int    `json:"fonte,omitempty"`
	ArquivoURL    string `json:"arquivo_url,omitempty"`
	ArquivoBase64 string `json:"arquivo_base64,omitempty"`
	CaminhoLocal  string `json:"caminho_local,omitempty"`
	Legenda       string `json:"legenda,omitempty"`
	MimeType      string `json:"mime_type,omitempty"`
	MensagemID    string `json:"mensagem_id,omitempty"`
}

type CarrosselCartaoRequest struct {
	Titulo       string         `json:"titulo,omitempty"`
	Texto        string         `json:"texto,omitempty"`
	Rodape       string         `json:"rodape,omitempty"`
	ImagemURL    string         `json:"imagem_url,omitempty"`
	ImagemBase64 string         `json:"imagem_base64,omitempty"`
	Botoes       []BotaoRequest `json:"botoes,omitempty"`
}

type EnvioCarrosselRequest struct {
	Instancia            string                   `json:"instancia,omitempty"`
	Numero               string                   `json:"numero,omitempty"`
	ChatJID              string                   `json:"chat_jid,omitempty"`
	Grupo                bool                     `json:"grupo,omitempty"`
	Texto                string                   `json:"texto,omitempty"`
	Rodape               string                   `json:"rodape,omitempty"`
	Cartoes              []CarrosselCartaoRequest `json:"cartoes,omitempty"`
	FallbackTexto        bool                     `json:"fallback_texto,omitempty"`
	MensagemID           string                   `json:"mensagem_id,omitempty"`
	RespostaMensagemID   string                   `json:"resposta_mensagem_id,omitempty"`
	RespostaParticipante string                   `json:"resposta_participante,omitempty"`
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

type ConsultaAvatarRequest struct {
	Instancia string `json:"instancia,omitempty"`
	Numero    string `json:"numero,omitempty"`
	ChatJID   string `json:"chat_jid,omitempty"`
	Grupo     bool   `json:"grupo,omitempty"`
}

type AvatarContato struct {
	Instancia string `json:"instancia"`
	Numero    string `json:"numero,omitempty"`
	ChatJID   string `json:"chat_jid,omitempty"`
	Grupo     bool   `json:"grupo"`
	TemAvatar bool   `json:"tem_avatar"`
	AvatarID  string `json:"avatar_id,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Tipo      string `json:"tipo,omitempty"`
}
