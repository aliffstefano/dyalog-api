package whatsapp

import (
	"encoding/base64"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

// raw_message_base64 de uma lista recebida de uma empresa (Pamcard), que
// renderiza no celular e no WhatsApp Web.
const listaEmpresaCapturada = "mgIWChJCCn67w1GeD4ScjuhI7sTv1QYQAqICvAESRkFudGVzIGRlIFNhaXIsIG1lIGRpeiAgbyBwb3JxdcOqIHZvY8OqIGVzdMOhIGVuY2VycmFuZG8gbyBhdGVuZGltZW50bz8aBE1lbnUgASpqEicKElByb2JsZW1hIHJlc29sdmlkbxoRUHJvYmxlbWFyZXNvbHZpZG8SKwoUQXRlbmRpbWVudG8gZGVtb3JhZG8aE0F0ZW5kaW1lbnRvZGVtb3JhZG8SEgoHRGVzaXN0aRoHRGVzaXN0aQ=="

// Com os mesmos dados, a ListMessage montada tem que ser identica a da empresa.
func TestListaBizIgualACapturada(t *testing.T) {
	binario, err := base64.StdEncoding.DecodeString(listaEmpresaCapturada)
	if err != nil {
		t.Fatal(err)
	}
	var capturada waE2E.Message
	if err := proto.Unmarshal(binario, &capturada); err != nil {
		t.Fatal(err)
	}

	msg := montarMensagemListaBiz(models.EnvioListaRequest{
		Descricao:  "Antes de Sair, me diz  o porquê você está encerrando o atendimento?",
		BotaoTexto: "Menu",
		Secoes: []models.ListaSecaoRequest{{Linhas: []models.ListaLinhaRequest{
			{ID: "Problemaresolvido", Titulo: "Problema resolvido"},
			{ID: "Atendimentodemorado", Titulo: "Atendimento demorado"},
			{ID: "Desisti", Titulo: "Desisti"},
		}}},
	})

	if !proto.Equal(msg.GetListMessage(), capturada.GetListMessage()) {
		t.Fatalf("ListMessage difere da capturada:\nnossa: %v\nempresa: %v", msg.GetListMessage(), capturada.GetListMessage())
	}
	// deviceListMetadata e preenchido pelo cliente do destinatario; a versao e
	// o que o remetente envia.
	if msg.GetMessageContextInfo().GetDeviceListMetadataVersion() != capturada.GetMessageContextInfo().GetDeviceListMetadataVersion() {
		t.Fatalf("deviceListMetadataVersion = %d, empresa = %d", msg.GetMessageContextInfo().GetDeviceListMetadataVersion(), capturada.GetMessageContextInfo().GetDeviceListMetadataVersion())
	}
}

func TestNosBizListaIgualAoCapturado(t *testing.T) {
	nos := nosBizLista()
	if len(nos) != 1 || nos[0].Tag != "biz" {
		t.Fatalf("esperado um unico <biz>: %#v", nos)
	}
	biz := nos[0]
	if biz.Attrs["actual_actors"] != "2" || biz.Attrs["host_storage"] != "2" || biz.Attrs["privacy_mode_ts"] == "" {
		t.Fatalf("atributos do <biz> = %#v", biz.Attrs)
	}
	filhos := biz.Content.([]waBinary.Node)
	lista := acharNo(filhos, "list")
	if lista == nil || lista.Attrs["type"] != "single_select" || lista.Attrs["v"] != "1" {
		t.Fatalf("<list> = %#v", lista)
	}
	qc := acharNo(filhos, "quality_control")
	if qc == nil || qc.Attrs["source_type"] != "third_party" || len(qc.Attrs["decision_id"].(string)) != 40 {
		t.Fatalf("<quality_control> = %#v", qc)
	}
	if nosBizLista()[0].Content.([]waBinary.Node)[1].Attrs["decision_id"] == qc.Attrs["decision_id"] {
		t.Fatal("decision_id deveria mudar a cada envio")
	}
}

func TestModoListaBizUsaEnvelopeProprio(t *testing.T) {
	tentativas := montarTentativasLista(models.EnvioListaRequest{Modo: "lista_biz"})
	if len(tentativas) != 1 || tentativas[0].modo != "lista_biz" || tentativas[0].nosBiz == nil {
		t.Fatalf("tentativas = %#v", tentativas)
	}
}
