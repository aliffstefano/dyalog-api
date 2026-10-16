package whatsapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	waAICommon "go.mau.fi/whatsmeow/proto/waAICommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

// Modo "lista_documento": o formato de lista que renderiza quando quem envia e
// uma conta pessoal (nao Business). Capturado em 16/10/2026 de uma lista
// enviada por outra API a partir de um numero pessoal e vista no celular:
//
//   - protobuf: o ListMessage vai DENTRO de documentWithCaptionMessage, e
//     tanto a mensagem de fora quanto a de dentro levam messageContextInfo com
//     messageSecret (32 bytes) e botMetadata.personaID;
//   - stanza: type="media", <enc mediatype="list"> (o whatsmeow desembrulha o
//     documentWithCaption e acha o ListMessage);
//   - <biz><list type="product_list" v="2"/></biz>. O whatsmeow nao poe o <biz>
//     sozinho aqui (ele nao olha dentro do documentWithCaption), entao vai em
//     AdditionalNodes.
//
// O personaID segue o mesmo formato capturado: um id fixo, o horario em
// milissegundos e 8 hex aleatorios.
const prefixoPersonaLista = "867051314767696$760019659443059"

func personaLista() string {
	aleatorio := make([]byte, 4)
	_, _ = rand.Read(aleatorio)
	return fmt.Sprintf("%s_%d_%s", prefixoPersonaLista, time.Now().UnixMilli(), hex.EncodeToString(aleatorio))
}

func montarMensagemListaDocumento(req models.EnvioListaRequest) *waE2E.Message {
	return montarListaComContexto(req, true, true)
}

// contextoListaBot devolve o gerador do messageContextInfo usado nas listas
// acima: o mesmo segredo e persona nos dois niveis. comBot=false deixa so o
// segredo (variacao de teste).
func contextoListaBot(comBot bool) func() *waE2E.MessageContextInfo {
	segredo := make([]byte, 32)
	_, _ = rand.Read(segredo)
	persona := personaLista()
	return func() *waE2E.MessageContextInfo {
		ctx := &waE2E.MessageContextInfo{MessageSecret: segredo}
		if comBot {
			ctx.BotMetadata = &waAICommon.BotMetadata{PersonaID: proto.String(persona)}
		}
		return ctx
	}
}

// montarListaComContexto monta o ListMessage com o messageContextInfo de bot,
// embrulhado ou nao em documentWithCaption.
func montarListaComContexto(req models.EnvioListaRequest, embrulhar, comBot bool) *waE2E.Message {
	contexto := contextoListaBot(comBot)
	lista := montarListMessage(req)
	// Como no capturado: sem rodape vazio.
	if strings.TrimSpace(lista.GetFooterText()) == "" {
		lista.FooterText = nil
	}
	if !embrulhar {
		return &waE2E.Message{MessageContextInfo: contexto(), ListMessage: lista}
	}
	return &waE2E.Message{
		MessageContextInfo: contexto(),
		DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				MessageContextInfo: contexto(),
				ListMessage:        lista,
			},
		},
	}
}

// montarNativeFlowComBot e o menu native_flow single_select levando o mesmo
// messageContextInfo de bot da lista que renderiza na conta pessoal.
func montarNativeFlowComBot(req models.EnvioListaRequest, embrulhar bool) *waE2E.Message {
	msg := montarMensagemListaNativeFlow(req)
	contexto := contextoListaBot(true)
	if msg.GetInteractiveMessage() == nil || !embrulhar {
		msg.MessageContextInfo = contexto()
		return msg
	}
	return &waE2E.Message{
		MessageContextInfo: contexto(),
		DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				MessageContextInfo: contexto(),
				InteractiveMessage: msg.GetInteractiveMessage(),
			},
		},
	}
}

// variacoesListaTeste sao os formatos enviados de uma vez pelo modo
// "variacoes", para descobrir no celular qual renderiza (principalmente a
// partir de conta Business). A ordem e a numeracao aparecem no texto de cada
// mensagem.
func variacoesListaTeste(req models.EnvioListaRequest) []tentativaLista {
	return []tentativaLista{
		{modo: "lista_documento", montar: func() *waE2E.Message { return montarListaComContexto(req, true, true) }, nosBiz: nosBizListaDocumento},
		{modo: "lista_documento_single", montar: func() *waE2E.Message { return montarListaComContexto(req, true, true) }, nosBiz: nosBizListaTipo("single_select")},
		{modo: "lista_documento_sem_biz", montar: func() *waE2E.Message { return montarListaComContexto(req, true, true) }, nosBiz: semNosBiz},
		{modo: "lista_documento_sem_bot", montar: func() *waE2E.Message { return montarListaComContexto(req, true, false) }, nosBiz: nosBizListaDocumento},
		{modo: "lista_bot", montar: func() *waE2E.Message { return montarListaComContexto(req, false, true) }, nosBiz: nosBizListaDocumento},
		{modo: "lista_bot_single", montar: func() *waE2E.Message { return montarListaComContexto(req, false, true) }, nosBiz: nosBizListaTipo("single_select")},
		{modo: "native_flow_bot", montar: func() *waE2E.Message { return montarNativeFlowComBot(req, false) }},
		{modo: "native_flow_documento", montar: func() *waE2E.Message { return montarNativeFlowComBot(req, true) }},
	}
}

func modoListaVariacoes(req models.EnvioListaRequest) bool {
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "variacoes", "variações", "testar":
		return true
	}
	return false
}

// enviarVariacoesLista manda cada variacao como uma mensagem separada, com
// "[n modo]" no comeco do texto, e diz na observacao quais o servidor aceitou.
// Aceitar nao quer dizer renderizar: quem diz qual funcionou e o celular.
func (g *GerenciadorInstancias) enviarVariacoesLista(ctx context.Context, client *whatsmeow.Client, req models.EnvioListaRequest, jids []types.JID) (models.ResultadoEnvio, error) {
	var (
		resultados []string
		jidUsado   types.JID
		ultimoID   string
	)
	variacoes := variacoesListaTeste(req)
	for i, base := range variacoes {
		copia := req
		copia.Descricao = fmt.Sprintf("[%d %s] %s", i+1, base.modo, textoMensagemListaEnvio(req))
		tentativa, _ := tentativaListaPorNome(copia, base.modo)
		msg := tentativa.montar()
		resp, jid, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
			if tentativa.nosBiz != nil {
				nos := tentativa.nosBiz()
				return client.SendMessage(ctx, jid, msg, whatsmeow.SendRequestExtra{AdditionalNodes: &nos})
			}
			return enviarMensagemInterativaComNome(ctx, client, jid, msg, req.FlowName)
		})
		if err != nil {
			resultados = append(resultados, fmt.Sprintf("%d %s: recusada (%v)", i+1, base.modo, err))
			continue
		}
		jidUsado, ultimoID = jid, string(resp.ID)
		resultados = append(resultados, fmt.Sprintf("%d %s: aceita (%s)", i+1, base.modo, resp.ID))
		// Um respiro entre as mensagens para chegarem na ordem.
		select {
		case <-ctx.Done():
			return models.ResultadoEnvio{}, ctx.Err()
		case <-time.After(700 * time.Millisecond):
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("nenhuma variacao de lista foi aceita: %s", strings.Join(resultados, "; "))
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: ultimoID,
		Status:     "aceita_pelo_servidor",
		Tipo:       "lista",
		Modo:       "variacoes",
		Observacao: strings.Join(resultados, "; "),
	}, nil
}

// tentativaListaPorNome acha uma das variacoes acima pelo modo.
func tentativaListaPorNome(req models.EnvioListaRequest, modo string) (tentativaLista, bool) {
	for _, v := range variacoesListaTeste(req) {
		if v.modo == modo {
			return v, true
		}
	}
	return tentativaLista{}, false
}

func nosBizListaTipo(tipo string) func() []waBinary.Node {
	return func() []waBinary.Node {
		return []waBinary.Node{{
			Tag:     "biz",
			Content: []waBinary.Node{{Tag: "list", Attrs: waBinary.Attrs{"type": tipo, "v": "2"}}},
		}}
	}
}

// semNosBiz manda a mensagem sem <biz> nenhum (nem o que o whatsmeow poria).
func semNosBiz() []waBinary.Node { return []waBinary.Node{} }

func nosBizListaDocumento() []waBinary.Node {
	return []waBinary.Node{{
		Tag: "biz",
		Content: []waBinary.Node{{
			Tag:   "list",
			Attrs: waBinary.Attrs{"type": "product_list", "v": "2"},
		}},
	}}
}

// contaBusiness diz se a instancia esta conectada num WhatsApp Business: a
// lista acima so renderiza quando quem envia e conta pessoal.
func contaBusiness(client *whatsmeow.Client) bool {
	if client == nil || client.Store == nil {
		return false
	}
	return strings.HasPrefix(client.Store.Platform, "smb") || client.Store.BusinessName != ""
}
