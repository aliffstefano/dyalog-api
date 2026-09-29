package whatsapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Simula o caminho completo: o whatsmeow loga o <message> bruto no "Recv", o
// evento decifrado chega depois e o arquivo final junta os dois.
func TestCapturaJuntaStanzaEMensagem(t *testing.T) {
	t.Setenv("CAPTURAR_MENSAGENS", "true")
	g := &GerenciadorInstancias{diretorioBase: t.TempDir()}

	stanza := &waBinary.Node{
		Tag:   "message",
		Attrs: waBinary.Attrs{"id": "ABC123", "type": "text"},
		Content: []waBinary.Node{
			{Tag: "biz", Content: []waBinary.Node{{Tag: "list", Attrs: waBinary.Attrs{"type": "product_list", "v": "2"}}}},
			{Tag: "enc", Attrs: waBinary.Attrs{"v": "2", "type": "msg"}, Content: []byte{1, 2, 3}},
		},
	}
	recv := loggerComCaptura(waLog.Noop, "inst").Sub("Recv")
	recv.Debugf("%s", stanza)

	msg := &waE2E.Message{ListMessage: &waE2E.ListMessage{Title: proto.String("Menu")}}
	g.capturarMensagem("inst", &events.Message{
		Info:       types.MessageInfo{ID: "ABC123"},
		Message:    msg,
		RawMessage: msg,
	})

	conteudo, err := os.ReadFile(filepath.Join(g.diretorioBase, "capturas", "inst_ABC123.json"))
	if err != nil {
		t.Fatalf("captura nao gravada: %v", err)
	}
	var registro map[string]interface{}
	if err := json.Unmarshal(conteudo, &registro); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	for _, campo := range []string{"stanza", "raw_message", "raw_message_base64", "info"} {
		if registro[campo] == nil {
			t.Fatalf("campo %s ausente: %s", campo, conteudo)
		}
	}
	filhos := registro["stanza"].(map[string]interface{})["content"].([]interface{})
	if filhos[0].(map[string]interface{})["tag"] != "biz" {
		t.Fatalf("<biz> nao preservado no stanza: %s", conteudo)
	}
	if retirarStanza("inst", "ABC123") != nil {
		t.Fatal("stanza deveria ser removido apos a captura")
	}
}

func TestCapturaDesligadaNaoEmbrulhaLogger(t *testing.T) {
	t.Setenv("CAPTURAR_MENSAGENS", "")
	if _, embrulhado := loggerComCaptura(waLog.Noop, "inst").(loggerCaptura); embrulhado {
		t.Fatal("sem CAPTURAR_MENSAGENS o logger nao deveria ser embrulhado")
	}
}
