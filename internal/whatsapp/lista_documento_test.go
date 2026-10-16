package whatsapp

import (
	"encoding/base64"
	"strings"
	"testing"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

// Lista capturada (16/10/2026) que renderizou a partir de numero pessoal; o
// conteudo e o mesmo do requisicao abaixo.
const listaCapturadaBase64 = "mgJcGiBEQ0odqgQDc+xSX7w5RhyUxLShvjqU8iBRW0ZCqOG1+zo4EjY4NjcwNTEzMTQ3Njc2OTYkNzYwMDE5NjU5NDQzMDU5XzE3OTIxMTQ3NDE5NzVfNGY4ZDYwZDWqA94BCtsBmgJcGiBEQ0odqgQDc+xSX7w5RhyUxLShvjqU8iBRW0ZCqOG1+zo4EjY4NjcwNTEzMTQ3Njc2OTYkNzYwMDE5NjU5NDQzMDU5XzE3OTIxMTQ3NDE5NzVfNGY4ZDYwZDWiAnkKABIfU2VsZWNpb25lIHVtIGl0ZW0gZG8gY2FyZMOhcGlvOhoMVmVyIG9ww6fDtWVzIAEqQgoGUHJhdG9zEiIKB0xhc2FuaGESE0JvbG9uaGVzYSwgNCBmYXRpYXMaAnAxEhQKBlJpc290bxIGRnVuZ2hpGgJwMkIA"

func TestListaDocumentoIgualACapturada(t *testing.T) {
	bruto, _ := base64.StdEncoding.DecodeString(listaCapturadaBase64)
	var capturada waE2E.Message
	if err := proto.Unmarshal(bruto, &capturada); err != nil {
		t.Fatal(err)
	}
	req := models.EnvioListaRequest{
		Descricao: "Selecione um item do cardápio:", BotaoTexto: "Ver opções",
		Secoes: []models.ListaSecaoRequest{{Titulo: "Pratos", Linhas: []models.ListaLinhaRequest{
			{ID: "p1", Titulo: "Lasanha", Descricao: "Bolonhesa, 4 fatias"},
			{ID: "p2", Titulo: "Risoto", Descricao: "Funghi"},
		}}},
	}
	nossa := montarMensagemListaDocumento(req)

	// Mesma estrutura: lista dentro do documentWithCaption, os dois niveis com
	// segredo e persona; a lista em si identica.
	for nome, msg := range map[string]*waE2E.Message{"capturada": &capturada, "nossa": nossa} {
		dentro := msg.GetDocumentWithCaptionMessage().GetMessage()
		if dentro.GetListMessage() == nil || len(msg.GetMessageContextInfo().GetMessageSecret()) != 32 ||
			len(dentro.GetMessageContextInfo().GetMessageSecret()) != 32 ||
			!strings.HasPrefix(dentro.GetMessageContextInfo().GetBotMetadata().GetPersonaID(), prefixoPersonaLista+"_") {
			t.Fatalf("%s fora do formato: %v", nome, msg)
		}
	}
	if !proto.Equal(capturada.GetDocumentWithCaptionMessage().GetMessage().GetListMessage(), nossa.GetDocumentWithCaptionMessage().GetMessage().GetListMessage()) {
		t.Fatalf("lista diferente:\ncapturada %v\nnossa     %v", capturada.GetDocumentWithCaptionMessage().GetMessage().GetListMessage(), nossa.GetDocumentWithCaptionMessage().GetMessage().GetListMessage())
	}
	if biz := nosBizListaDocumento()[0]; biz.Tag != "biz" {
		t.Fatal("sem <biz>")
	} else if lista := biz.GetChildren()[0]; lista.Attrs["type"] != "product_list" || lista.Attrs["v"] != "2" {
		t.Fatalf("<biz> diferente do capturado: %v", lista.Attrs)
	}
	if getTipo := getMediaTypeLista(nossa); getTipo != "list" {
		t.Fatalf("mediatype = %q", getTipo)
	}
}

func TestListaAutoContaPessoalUsaDocumento(t *testing.T) {
	pessoal := montarTentativasLista(models.EnvioListaRequest{}, false)
	if len(pessoal) == 0 || pessoal[0].modo != "lista_documento" || pessoal[0].nosBiz == nil {
		t.Fatalf("conta pessoal deveria comecar pela lista_documento: %#v", pessoal)
	}
	if business := montarTentativasLista(models.EnvioListaRequest{}, true); business[0].modo == "lista_documento" {
		t.Fatal("conta Business nao deveria usar lista_documento no auto")
	}
}

// getMediaTypeLista repete a regra do whatsmeow (desembrulha o
// documentWithCaption) para garantir o <enc mediatype="list">.
func getMediaTypeLista(msg *waE2E.Message) string {
	if dentro := msg.GetDocumentWithCaptionMessage().GetMessage(); dentro != nil {
		msg = dentro
	}
	if msg.GetListMessage() != nil {
		return "list"
	}
	return ""
}
