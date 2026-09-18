package whatsapp

import (
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func msgNativeFlow(nomes ...string) *waE2E.Message {
	botoes := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, 0, len(nomes))
	for _, nome := range nomes {
		botoes = append(botoes, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String(nome),
			ButtonParamsJSON: proto.String("{}"),
		})
	}
	return &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{Buttons: botoes},
		},
	}}
}

func acharNo(nos []waBinary.Node, tag string) *waBinary.Node {
	for i := range nos {
		if nos[i].Tag == tag {
			return &nos[i]
		}
	}
	return nil
}

// Valida o formato completo que o servidor aceita. Variacoes mais simples
// (so <interactive>, sem os atributos do <biz>, sem <quality_control>) foram
// reportadas retornando ack error 479 e nao entregando a mensagem.
func TestNosInterativoMontaFormatoCompleto(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	nos := nosInterativoNativeFlow(msgNativeFlow("quick_reply"), destino)

	biz := acharNo(nos, "biz")
	if biz == nil {
		t.Fatalf("no <biz> ausente: %#v", nos)
	}
	if biz.Attrs["actual_actors"] != "2" || biz.Attrs["host_storage"] != "2" {
		t.Fatalf("attrs de <biz> = %#v, esperado actual_actors=2 host_storage=2", biz.Attrs)
	}
	if _, ok := biz.Attrs["privacy_mode_ts"]; !ok {
		t.Fatalf("<biz> precisa de privacy_mode_ts: %#v", biz.Attrs)
	}

	filhos := biz.Content.([]waBinary.Node)
	interativo := acharNo(filhos, "interactive")
	if interativo == nil {
		t.Fatalf("no <interactive> ausente dentro de <biz>: %#v", biz)
	}
	if interativo.Attrs["type"] != "native_flow" || interativo.Attrs["v"] != "1" {
		t.Fatalf("attrs de <interactive> = %#v, esperado type=native_flow v=1", interativo.Attrs)
	}
	flow := acharNo(interativo.Content.([]waBinary.Node), "native_flow")
	if flow == nil {
		t.Fatalf("no <native_flow> ausente: %#v", interativo)
	}
	if flow.Attrs["v"] != "9" {
		t.Fatalf("v do native_flow = %v, esperado 9", flow.Attrs["v"])
	}
	if flow.Attrs["name"] != "mixed" {
		t.Fatalf("name do native_flow = %v, esperado mixed", flow.Attrs["name"])
	}

	qc := acharNo(filhos, "quality_control")
	if qc == nil {
		t.Fatalf("no <quality_control> ausente dentro de <biz>: %#v", biz)
	}
	if qc.Attrs["source_type"] != "third_party" {
		t.Fatalf("source_type = %v, esperado third_party", qc.Attrs["source_type"])
	}
	decisionID, _ := qc.Attrs["decision_id"].(string)
	if len(decisionID) != 40 { // 20 bytes em hex
		t.Fatalf("decision_id = %q (len %d), esperado 40 chars hex", decisionID, len(decisionID))
	}
	fonte := acharNo(qc.Content.([]waBinary.Node), "decision_source")
	if fonte == nil || fonte.Attrs["value"] != "df" {
		t.Fatalf("<decision_source value=\"df\"> ausente/incorreto: %#v", qc)
	}
}

// decision_id precisa ser novo a cada mensagem, nao um valor fixo.
func TestDecisionIDMudaACadaChamada(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	extrair := func() string {
		nos := nosInterativoNativeFlow(msgNativeFlow("quick_reply"), destino)
		qc := acharNo(acharNo(nos, "biz").Content.([]waBinary.Node), "quality_control")
		id, _ := qc.Attrs["decision_id"].(string)
		return id
	}
	if extrair() == extrair() {
		t.Fatal("decision_id deveria ser aleatorio por mensagem")
	}
}

// Fluxos que nao sao de pagamento usam "mixed": nomes especificos foram
// recusados pelo servidor com erro 473.
func TestNomeNodeNativeFlowUsaMixedForaDePagamento(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	casos := [][]string{
		{"quick_reply"},
		{"quick_reply", "cta_url"},
	}
	for _, botoes := range casos {
		nos := nosInterativoNativeFlow(msgNativeFlow(botoes...), destino)
		biz := acharNo(nos, "biz")
		interativo := acharNo(biz.Content.([]waBinary.Node), "interactive")
		flow := acharNo(interativo.Content.([]waBinary.Node), "native_flow")
		if flow.Attrs["name"] != "mixed" {
			t.Fatalf("botoes %v -> name = %v, esperado mixed", botoes, flow.Attrs["name"])
		}
	}
}

// Cobranca usa o envelope simples do app oficial: <biz> sem atributos,
// native_flow com o nome do botao e sem v, e sem <quality_control>. Com o
// formato completo a mensagem chega mas o app mobile nao renderiza o botao.
func TestFluxoPagamentoUsaEnvelopeSimplesDoAppOficial(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	nos := nosInterativoNativeFlow(msgNativeFlow("payment_info"), destino)

	biz := acharNo(nos, "biz")
	if biz == nil {
		t.Fatalf("no <biz> ausente: %#v", nos)
	}
	if len(biz.Attrs) != 0 {
		t.Fatalf("<biz> de pagamento deve ir sem atributos, obtido: %#v", biz.Attrs)
	}

	filhos := biz.Content.([]waBinary.Node)
	if acharNo(filhos, "quality_control") != nil {
		t.Fatalf("<quality_control> nao deve acompanhar cobranca: %#v", filhos)
	}

	interativo := acharNo(filhos, "interactive")
	if interativo == nil {
		t.Fatalf("no <interactive> ausente: %#v", biz)
	}
	if interativo.Attrs["type"] != "native_flow" || interativo.Attrs["v"] != "1" {
		t.Fatalf("attrs de <interactive> = %#v, esperado type=native_flow v=1", interativo.Attrs)
	}

	flow := acharNo(interativo.Content.([]waBinary.Node), "native_flow")
	if flow.Attrs["name"] != "payment_info" {
		t.Fatalf("name = %v, esperado payment_info", flow.Attrs["name"])
	}
	if _, temV := flow.Attrs["v"]; temV {
		t.Fatalf("<native_flow> de pagamento nao deve ter atributo v: %#v", flow.Attrs)
	}
}

// O override continua disponivel para experimentacao.
func TestNomeNodeNativeFlowAceitaOverride(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	nos := nosInterativoNativeFlowComNome(msgNativeFlow("payment_info"), destino, "order_details")

	biz := acharNo(nos, "biz")
	interativo := acharNo(biz.Content.([]waBinary.Node), "interactive")
	flow := acharNo(interativo.Content.([]waBinary.Node), "native_flow")
	if flow.Attrs["name"] != "order_details" {
		t.Fatalf("override ignorado: name = %v", flow.Attrs["name"])
	}
}

// Mensagens comuns (texto puro) nao devem ganhar nos extras.
func TestNosInterativoIgnoraMensagensComuns(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)

	texto := &waE2E.Message{Conversation: proto.String("ola")}
	if nos := nosInterativoNativeFlow(texto, destino); nos != nil {
		t.Fatalf("texto puro nao deveria gerar nos extras: %#v", nos)
	}
}

func TestNosInterativoEncontraNativeFlowEmViewOnce(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	interno := msgNativeFlow("quick_reply")
	viewOnce := &waE2E.Message{ViewOnceMessage: &waE2E.FutureProofMessage{Message: interno}}

	if nos := nosInterativoNativeFlow(viewOnce, destino); acharNo(nos, "biz") == nil {
		t.Fatalf("deveria encontrar native flow dentro de ViewOnceMessage: %#v", nos)
	}
}
