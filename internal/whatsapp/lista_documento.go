package whatsapp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	waAICommon "go.mau.fi/whatsmeow/proto/waAICommon"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
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
	segredo := make([]byte, 32)
	_, _ = rand.Read(segredo)
	persona := personaLista()
	contexto := func() *waE2E.MessageContextInfo {
		return &waE2E.MessageContextInfo{
			MessageSecret: segredo,
			BotMetadata:   &waAICommon.BotMetadata{PersonaID: proto.String(persona)},
		}
	}
	lista := montarListMessage(req)
	// Como no capturado: sem rodape vazio.
	if strings.TrimSpace(lista.GetFooterText()) == "" {
		lista.FooterText = nil
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
