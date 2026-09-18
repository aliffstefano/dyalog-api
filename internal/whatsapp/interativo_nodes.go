package whatsapp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// O whatsmeow anexa o no <biz> automaticamente para ButtonsMessage e ListMessage
// (ver getButtonTypeFromMessage em send.go), mas NAO para InteractiveMessage
// (native flow). Sem esse no, o servidor do WhatsApp aceita a mensagem e devolve
// um ID, porem descarta silenciosamente antes de entregar ao destinatario - foi
// exatamente o sintoma observado ao testar botoes e cobranca Pix.
//
// Aqui montamos manualmente os nos que o cliente oficial envia, para injeta-los
// via whatsmeow.SendRequestExtra.AdditionalNodes.
//
// AVISO: isso replica trafego do cliente oficial observado por projetos da
// comunidade (helpers do Baileys). Nao e formato documentado pela Meta e pode
// parar de funcionar quando eles mudarem a validacao do lado do servidor.

// nomeNodeFlowMisto e o valor do atributo name do no <native_flow>. Ver
// nomeNodeNativeFlow para o porque de ser sempre este valor.
const nomeNodeFlowMisto = "mixed"

// contextInfoMensagemInterativa devolve o MessageContextInfo usado nas mensagens
// de native flow. A versao 3 (em vez de 2) e sem o DeviceListMetadata vazio
// espelha o que a comunidade reporta como o formato aceito pelo servidor; era
// uma das diferencas em relacao ao que enviavamos antes.
func contextInfoMensagemInterativa() *waE2E.MessageContextInfo {
	return &waE2E.MessageContextInfo{
		DeviceListMetadataVersion: proto.Int32(3),
	}
}

// nosInterativoNativeFlow devolve os nos extras necessarios para o WhatsApp
// entregar uma InteractiveMessage de native flow. Retorna nil quando a mensagem
// nao for desse tipo (deixando o whatsmeow cuidar do caso normal).
//
// O formato abaixo replica o trafego do cliente oficial e foi validado por
// terceiros contra o servidor real. Variacoes mais simples NAO funcionam:
//
//	sem <biz>                                  -> aceito, nada entregue (silencioso)
//	<biz><buttons/></biz>                      -> ack error 405
//	<biz><interactive type="native_flow" v="1"/></biz> -> ack error 479, nao entregue
//	formato completo abaixo                    -> entregue e renderizado
//
// Ou seja, o <interactive> sozinho nao basta: <biz> precisa dos proprios
// atributos e o bloco <quality_control> precisa acompanhar.
func nosInterativoNativeFlow(msg *waE2E.Message, destino types.JID) []waBinary.Node {
	return nosInterativoNativeFlowComNome(msg, destino, "")
}

// nosInterativoNativeFlowComNome permite forcar o atributo name do <native_flow>.
// Usado onde o nome correto ainda nao e conhecido com certeza, para permitir
// testar variacoes sem recompilar.
func nosInterativoNativeFlowComNome(msg *waE2E.Message, _ types.JID, nomeForcado string) []waBinary.Node {
	nativeFlow := extrairNativeFlowMessage(msg)
	if nativeFlow == nil {
		return nil
	}
	nomeFlow := strings.TrimSpace(nomeForcado)
	if nomeFlow == "" {
		nomeFlow = nomeNodeNativeFlow(nativeFlow)
	}
	// Fluxos de pagamento usam um envelope diferente: comparando o stanza que o
	// app oficial envia com o nosso, a cobranca Pix vai com <biz> "pelado" (sem
	// atributos), native_flow sem v e sem o bloco <quality_control>. Com o
	// formato completo a mensagem e entregue e renderiza no WhatsApp Web, mas o
	// app mobile nao desenha o botao de pagamento.
	if ehFluxoPagamento(nativeFlow) {
		return []waBinary.Node{{
			Tag: "biz",
			Content: []waBinary.Node{{
				Tag: "interactive",
				Attrs: waBinary.Attrs{
					"type": "native_flow",
					"v":    "1",
				},
				Content: []waBinary.Node{{
					Tag:   "native_flow",
					Attrs: waBinary.Attrs{"name": nomeFlow},
				}},
			}},
		}}
	}
	return []waBinary.Node{{
		Tag: "biz",
		Attrs: waBinary.Attrs{
			"actual_actors":   "2",
			"host_storage":    "2",
			"privacy_mode_ts": strconv.FormatInt(time.Now().Unix(), 10),
		},
		Content: []waBinary.Node{
			{
				Tag: "interactive",
				Attrs: waBinary.Attrs{
					"type": "native_flow",
					"v":    "1",
				},
				Content: []waBinary.Node{{
					Tag: "native_flow",
					Attrs: waBinary.Attrs{
						"v":    "9",
						"name": nomeFlow,
					},
				}},
			},
			{
				Tag: "quality_control",
				Attrs: waBinary.Attrs{
					"decision_id": gerarDecisionID(),
					"source_type": "third_party",
				},
				Content: []waBinary.Node{{
					Tag:   "decision_source",
					Attrs: waBinary.Attrs{"value": "df"},
				}},
			},
		},
	}}
}

// gerarDecisionID produz o identificador de 20 bytes aleatorios (em hex) que o
// bloco <quality_control> espera, um novo por mensagem.
func gerarDecisionID() string {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand falhando e praticamente impossivel; um id derivado do
		// relogio ainda deixa a mensagem sair em vez de aborta-la.
		return hex.EncodeToString([]byte(strconv.FormatInt(time.Now().UnixNano(), 16)))
	}
	return hex.EncodeToString(buf)
}

// enviarMensagemInterativa envia a mensagem anexando os nos <biz>/<bot> quando
// ela for uma InteractiveMessage de native flow. Para qualquer outro tipo de
// mensagem, comporta-se como um SendMessage normal.
func enviarMensagemInterativa(ctx context.Context, client *whatsmeow.Client, destino types.JID, msg *waE2E.Message) (whatsmeow.SendResponse, error) {
	return enviarMensagemInterativaComNome(ctx, client, destino, msg, "")
}

// enviarMensagemInterativaComNome e a variante que permite forcar o name do
// <native_flow>, usada pelo fluxo de cobranca Pix enquanto o valor correto nao
// esta confirmado.
func enviarMensagemInterativaComNome(ctx context.Context, client *whatsmeow.Client, destino types.JID, msg *waE2E.Message, nomeFlow string) (whatsmeow.SendResponse, error) {
	nos := nosInterativoNativeFlowComNome(msg, destino, nomeFlow)
	if len(nos) == 0 {
		return client.SendMessage(ctx, destino, msg)
	}
	return client.SendMessage(ctx, destino, msg, whatsmeow.SendRequestExtra{
		AdditionalNodes: &nos,
	})
}

// extrairNativeFlowMessage encontra o NativeFlowMessage mesmo quando a
// InteractiveMessage vem embrulhada (view once, ephemeral).
func extrairNativeFlowMessage(msg *waE2E.Message) *waE2E.InteractiveMessage_NativeFlowMessage {
	if msg == nil {
		return nil
	}
	switch {
	case msg.GetViewOnceMessage() != nil:
		return extrairNativeFlowMessage(msg.GetViewOnceMessage().GetMessage())
	case msg.GetViewOnceMessageV2() != nil:
		return extrairNativeFlowMessage(msg.GetViewOnceMessageV2().GetMessage())
	case msg.GetEphemeralMessage() != nil:
		return extrairNativeFlowMessage(msg.GetEphemeralMessage().GetMessage())
	case msg.GetInteractiveMessage() != nil:
		return msg.GetInteractiveMessage().GetNativeFlowMessage()
	default:
		return nil
	}
}

// nomesFluxoPagamento sao os botoes que caracterizam uma mensagem de cobranca.
var nomesFluxoPagamento = map[string]bool{
	"payment_info":   true,
	"review_and_pay": true,
	"order_details":  true,
}

// ehFluxoPagamento indica se a mensagem interativa e uma cobranca, caso em que o
// envelope <biz> segue o formato simples do app oficial.
func ehFluxoPagamento(nativeFlow *waE2E.InteractiveMessage_NativeFlowMessage) bool {
	for _, botao := range nativeFlow.GetButtons() {
		if nomesFluxoPagamento[botao.GetName()] {
			return true
		}
	}
	return false
}

// nomesFlowProprio sao botoes cujo <native_flow name="..."> usa o proprio nome
// do botao, em vez de "mixed". Confirmado para payment_info (com "order_details"
// o servidor recusa com 473); single_select segue o mesmo padrao por analogia -
// ainda nao confirmado contra um stanza real.
var nomesFlowProprio = map[string]bool{
	"payment_info":   true,
	"review_and_pay": true,
	"order_details":  true,
	"single_select":  true,
}

// nomeNodeNativeFlow decide o valor do atributo name do no <native_flow>.
//
// "mixed" e o valor comprovado para botoes de resposta rapida. Fluxos com nome
// proprio (pagamento, menu) usam o nome do botao.
func nomeNodeNativeFlow(nativeFlow *waE2E.InteractiveMessage_NativeFlowMessage) string {
	for _, botao := range nativeFlow.GetButtons() {
		if nomesFlowProprio[botao.GetName()] {
			return botao.GetName()
		}
	}
	return nomeNodeFlowMisto
}
