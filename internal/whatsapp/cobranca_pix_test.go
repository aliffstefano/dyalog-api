package whatsapp

import (
	"encoding/json"
	"testing"

	"dyalog-api-go/internal/models"
)

func TestMontarMensagemCobrancaPixSemValorReplicaFormatoReal(t *testing.T) {
	req := models.EnvioCobrancaPixRequest{
		ChavePix:         "67999440667",
		TipoChave:        "telefone",
		NomeBeneficiario: "Aliff Stefano",
	}

	msg, err := montarMensagemCobrancaPix(req)
	if err != nil {
		t.Fatalf("erro ao montar mensagem: %v", err)
	}
	interactive := msg.GetInteractiveMessage()
	if interactive == nil {
		t.Fatal("InteractiveMessage ausente")
	}
	nativeFlow := interactive.GetNativeFlowMessage()
	if nativeFlow == nil || len(nativeFlow.GetButtons()) != 1 {
		t.Fatalf("esperado exatamente 1 botao native_flow, obtido: %#v", nativeFlow)
	}
	botao := nativeFlow.GetButtons()[0]
	if botao.GetName() != "payment_info" {
		t.Fatalf("nome do botao = %q, esperado payment_info", botao.GetName())
	}

	// A cobranca real capturada do app oficial veio com corpo vazio e
	// MessageVersion 0. Preencher esses campos foi uma das diferencas
	// identificadas entre o nosso envio e o do cliente oficial.
	if interactive.GetBody().GetText() != "" {
		t.Fatalf("corpo = %q, esperado vazio quando nao ha descricao", interactive.GetBody().GetText())
	}
	if nativeFlow.GetMessageVersion() != 0 {
		t.Fatalf("MessageVersion = %d, esperado 0 (nao definido)", nativeFlow.GetMessageVersion())
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(botao.GetButtonParamsJSON()), &params); err != nil {
		t.Fatalf("params nao e JSON valido: %v", err)
	}

	// Confere contra a estrutura exata observada no webhook real (cobranca sem
	// valor definido: order_type ORDER_WITHOUT_AMOUNT, value 0, offset 1).
	if params["currency"] != "BRL" {
		t.Fatalf("currency = %v, esperado BRL", params["currency"])
	}
	pagamento := params["payment_settings"].([]interface{})
	pixSettings := pagamento[0].(map[string]interface{})
	if pixSettings["type"] != "pix_static_code" {
		t.Fatalf("payment_settings[0].type = %v, esperado pix_static_code", pixSettings["type"])
	}
	pix := pixSettings["pix_static_code"].(map[string]interface{})
	if pix["key"] != "+5567999440667" {
		t.Fatalf("chave pix formatada = %v, esperado +5567999440667", pix["key"])
	}
	// Sem flow_type=APPSWITCH o app mobile entrega a mensagem mas nao renderiza
	// o botao de pagamento (so o WhatsApp Web renderiza).
	if pix["flow_type"] != "APPSWITCH" {
		t.Fatalf("flow_type = %v, esperado APPSWITCH", pix["flow_type"])
	}
	if pix["key_type"] != "PHONE" {
		t.Fatalf("key_type = %v, esperado PHONE", pix["key_type"])
	}
	if pix["merchant_name"] != "Aliff Stefano" {
		t.Fatalf("merchant_name = %v, esperado Aliff Stefano", pix["merchant_name"])
	}

	order := params["order"].(map[string]interface{})
	if order["order_type"] != "ORDER_WITHOUT_AMOUNT" {
		t.Fatalf("order_type = %v, esperado ORDER_WITHOUT_AMOUNT", order["order_type"])
	}
	total := params["total_amount"].(map[string]interface{})
	if total["value"] != float64(0) || total["offset"] != float64(1) {
		t.Fatalf("total_amount = %#v, esperado value=0 offset=1", total)
	}
}

func TestMontarMensagemCobrancaPixComValorUsaCentavos(t *testing.T) {
	req := models.EnvioCobrancaPixRequest{
		ChavePix:         "chave@exemplo.com",
		TipoChave:        "email",
		NomeBeneficiario: "Loja Exemplo",
		Valor:            49.9,
	}

	msg, err := montarMensagemCobrancaPix(req)
	if err != nil {
		t.Fatalf("erro ao montar mensagem: %v", err)
	}
	botao := msg.GetInteractiveMessage().GetNativeFlowMessage().GetButtons()[0]
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(botao.GetButtonParamsJSON()), &params); err != nil {
		t.Fatalf("params nao e JSON valido: %v", err)
	}

	order := params["order"].(map[string]interface{})
	if order["order_type"] != "ORDER" {
		t.Fatalf("order_type = %v, esperado ORDER quando ha valor", order["order_type"])
	}
	total := params["total_amount"].(map[string]interface{})
	if total["value"] != float64(4990) || total["offset"] != float64(100) {
		t.Fatalf("total_amount = %#v, esperado value=4990 offset=100 (R$49,90)", total)
	}

	pix := params["payment_settings"].([]interface{})[0].(map[string]interface{})["pix_static_code"].(map[string]interface{})
	if pix["key"] != "chave@exemplo.com" {
		t.Fatalf("chave email nao deveria ser reformatada, obtido: %v", pix["key"])
	}
	if pix["key_type"] != "EMAIL" {
		t.Fatalf("key_type = %v, esperado EMAIL", pix["key_type"])
	}
}

func TestNormalizarChavePixTelefoneComOnzeDigitos(t *testing.T) {
	casos := map[string]string{
		"67999440667":    "+5567999440667",
		"+5567999440667": "+5567999440667",
		"5567999440667":  "+5567999440667",
	}
	for entrada, esperado := range casos {
		obtido := normalizarChavePix("PHONE", entrada)
		if obtido != esperado {
			t.Fatalf("normalizarChavePix(%q) = %q, esperado %q", entrada, obtido, esperado)
		}
	}
}

func TestNormalizarTipoChavePixInvalido(t *testing.T) {
	if _, err := normalizarTipoChavePixInterno("bitcoin"); err == nil {
		t.Fatal("esperado erro para tipo_chave invalido")
	}
}
