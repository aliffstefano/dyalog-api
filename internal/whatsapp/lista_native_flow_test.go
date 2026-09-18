package whatsapp

import (
	"encoding/json"
	"testing"

	"dyalog-api-go/internal/models"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
)

func reqListaTeste() models.EnvioListaRequest {
	return models.EnvioListaRequest{
		Titulo:     "Atendimento",
		Descricao:  "Escolha uma opcao",
		BotaoTexto: "Abrir menu",
		Rodape:     "Dyalog",
		Secoes: []models.ListaSecaoRequest{{
			Titulo: "Setores",
			Linhas: []models.ListaLinhaRequest{
				{ID: "financeiro", Titulo: "Financeiro", Descricao: "Boletos"},
				{ID: "suporte", Titulo: "Suporte"},
			},
		}},
	}
}

func TestMontarMensagemListaNativeFlowUsaSingleSelect(t *testing.T) {
	msg := montarMensagemListaNativeFlow(reqListaTeste())

	nativeFlow := msg.GetInteractiveMessage().GetNativeFlowMessage()
	if nativeFlow == nil || len(nativeFlow.GetButtons()) != 1 {
		t.Fatalf("esperado 1 botao native_flow, obtido: %#v", nativeFlow)
	}
	botao := nativeFlow.GetButtons()[0]
	if botao.GetName() != "single_select" {
		t.Fatalf("nome do botao = %q, esperado single_select", botao.GetName())
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(botao.GetButtonParamsJSON()), &params); err != nil {
		t.Fatalf("buttonParamsJSON invalido: %v", err)
	}
	if params["title"] != "Abrir menu" {
		t.Fatalf("title = %v, esperado 'Abrir menu' (texto do botao)", params["title"])
	}

	secoes, ok := params["sections"].([]interface{})
	if !ok || len(secoes) != 1 {
		t.Fatalf("sections = %#v, esperado 1 secao", params["sections"])
	}
	secao := secoes[0].(map[string]interface{})
	if secao["title"] != "Setores" {
		t.Fatalf("titulo da secao = %v", secao["title"])
	}
	linhas := secao["rows"].([]interface{})
	if len(linhas) != 2 {
		t.Fatalf("esperado 2 linhas, obtido %d", len(linhas))
	}

	primeira := linhas[0].(map[string]interface{})
	if primeira["id"] != "financeiro" || primeira["title"] != "Financeiro" || primeira["description"] != "Boletos" {
		t.Fatalf("primeira linha = %#v", primeira)
	}
	// Linha sem descricao nao deve emitir o campo vazio.
	segunda := linhas[1].(map[string]interface{})
	if _, temDescricao := segunda["description"]; temDescricao {
		t.Fatalf("linha sem descricao nao deveria ter o campo: %#v", segunda)
	}
}

func TestMontarMensagemListaNativeFlowPreencheCabecalhoERodape(t *testing.T) {
	msg := montarMensagemListaNativeFlow(reqListaTeste())
	interativo := msg.GetInteractiveMessage()

	if interativo.GetHeader().GetTitle() != "Atendimento" {
		t.Fatalf("titulo do header = %q", interativo.GetHeader().GetTitle())
	}
	if interativo.GetBody().GetText() != "Escolha uma opcao" {
		t.Fatalf("corpo = %q", interativo.GetBody().GetText())
	}
	if interativo.GetFooter().GetText() != "Dyalog" {
		t.Fatalf("rodape = %q", interativo.GetFooter().GetText())
	}
}

// O menu via native_flow usa o envelope completo, com name igual ao nome do
// botao (single_select) - mesmo padrao confirmado no fluxo de pagamento.
func TestListaNativeFlowUsaEnvelopeCompleto(t *testing.T) {
	msg := montarMensagemListaNativeFlow(reqListaTeste())
	destino := types.NewJID("5511999999999", types.DefaultUserServer)

	nos := nosInterativoNativeFlow(msg, destino)
	biz := acharNo(nos, "biz")
	if biz == nil {
		t.Fatalf("no <biz> ausente: %#v", nos)
	}
	if biz.Attrs["actual_actors"] != "2" {
		t.Fatalf("<biz> deveria ter atributos do formato completo: %#v", biz.Attrs)
	}
	filhos := biz.Content.([]waBinary.Node)
	if acharNo(filhos, "quality_control") == nil {
		t.Fatalf("<quality_control> ausente no formato completo: %#v", filhos)
	}
	flow := acharNo(acharNo(filhos, "interactive").Content.([]waBinary.Node), "native_flow")
	if flow.Attrs["name"] != "single_select" || flow.Attrs["v"] != "9" {
		t.Fatalf("native_flow = %#v, esperado name=single_select v=9", flow.Attrs)
	}
}

// ListMessage nao deve receber no <biz> nosso: o whatsmeow ja adiciona o dele
// automaticamente, e dois <biz> no mesmo stanza fazem o servidor recusar com 479.
func TestListMessageNaoRecebeBizExtra(t *testing.T) {
	destino := types.NewJID("5511999999999", types.DefaultUserServer)
	msg := montarMensagemLista(reqListaTeste())

	if nos := nosInterativoNativeFlow(msg, destino); nos != nil {
		t.Fatalf("ListMessage nao deve gerar <biz> extra (causaria duplicata/479): %#v", nos)
	}
}

// O native_flow e aceito pelo servidor sem erro mas nunca renderiza. Se
// estivesse no modo auto, a cadeia pararia nele e o destinatario receberia uma
// mensagem invisivel, em vez do fallback de texto que ele consegue ler.
func TestMontarTentativasListaAutoNaoUsaNativeFlow(t *testing.T) {
	tentativas := montarTentativasLista(models.EnvioListaRequest{})
	if len(tentativas) == 0 {
		t.Fatal("modo auto nao deveria ficar sem tentativas")
	}
	if tentativas[0].modo != "lista" {
		t.Fatalf("modo auto deveria comecar pela lista classica, obtido: %#v", tentativas)
	}
	for _, tentativa := range tentativas {
		if tentativa.modo == "native_flow" {
			t.Fatalf("native_flow nao deve entrar no auto (bloquearia o fallback de texto): %#v", tentativas)
		}
	}
}

// O modo explicito continua disponivel para experimentacao futura.
func TestModoNativeFlowExplicitoContinuaDisponivel(t *testing.T) {
	tentativas := montarTentativasLista(models.EnvioListaRequest{Modo: "native_flow"})
	if len(tentativas) != 1 || tentativas[0].modo != "native_flow" {
		t.Fatalf("modo explicito native_flow = %#v", tentativas)
	}
}
