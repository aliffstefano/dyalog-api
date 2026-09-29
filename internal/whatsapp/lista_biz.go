package whatsapp

import (
	"strconv"
	"strings"
	"time"

	waBinary "go.mau.fi/whatsmeow/binary"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

// Modo "lista_biz": replica a lista que empresas enviam e que renderiza no
// celular e no WhatsApp Web. Capturado de duas empresas diferentes (setembro de
// 2026), o formato e:
//
//   - protobuf: ListMessage simples (description, buttonText, listType
//     SINGLE_SELECT, sections/rows), sem campos vazios, com
//     messageContextInfo.deviceListMetadataVersion = 2
//   - stanza: type="media", <enc mediatype="list"> (o whatsmeow ja faz isso)
//   - <biz actual_actors="2" host_storage="2" privacy_mode_ts="...">
//     <list type="single_select" v="1"/>
//     <quality_control decision_id="<20 bytes hex>" source_type="third_party">
//     <decision_source value="df"/></quality_control></biz>
//
// O whatsmeow sempre anexa o proprio <biz><list v="2"/></biz> em ListMessage.
// Para enviar o nosso no lugar, o build precisa do ajuste de
// tools/whatsmeow-overlay (aplicado no Dockerfile). Sem ele o stanza sai com
// dois <biz> e o servidor recusa.

// montarMensagemListaBiz monta a ListMessage omitindo campos vazios, como no
// stanza capturado.
func montarMensagemListaBiz(req models.EnvioListaRequest) *waE2E.Message {
	opcional := func(valor string) *string {
		if valor = strings.TrimSpace(valor); valor == "" {
			return nil
		}
		return proto.String(valor)
	}
	secoes := make([]*waE2E.ListMessage_Section, 0, len(req.Secoes))
	for _, secao := range req.Secoes {
		linhas := make([]*waE2E.ListMessage_Row, 0, len(secao.Linhas))
		for _, linha := range secao.Linhas {
			linhas = append(linhas, &waE2E.ListMessage_Row{
				Title:       opcional(linha.Titulo),
				Description: opcional(linha.Descricao),
				RowID:       opcional(linha.ID),
			})
		}
		secoes = append(secoes, &waE2E.ListMessage_Section{
			Title: opcional(secao.Titulo),
			Rows:  linhas,
		})
	}
	// contextInfo so vai quando e resposta a outra mensagem; vazio, a empresa
	// nao envia.
	contexto := contextInfoLista(req)
	if contexto != nil && proto.Size(contexto) == 0 {
		contexto = nil
	}
	return &waE2E.Message{
		ListMessage: &waE2E.ListMessage{
			Title:       opcional(req.Titulo),
			Description: opcional(textoMensagemListaEnvio(req)),
			ButtonText:  opcional(req.BotaoTexto),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    secoes,
			FooterText:  opcional(req.Rodape),
			ContextInfo: contexto,
		},
		MessageContextInfo: &waE2E.MessageContextInfo{
			DeviceListMetadataVersion: proto.Int32(2),
		},
	}
}

// nosBizLista devolve o <biz> identico ao capturado, com decision_id novo.
func nosBizLista() []waBinary.Node {
	return []waBinary.Node{{
		Tag: "biz",
		Attrs: waBinary.Attrs{
			"actual_actors":   "2",
			"host_storage":    "2",
			"privacy_mode_ts": strconv.FormatInt(time.Now().Unix(), 10),
		},
		Content: []waBinary.Node{
			{
				Tag:   "list",
				Attrs: waBinary.Attrs{"type": "single_select", "v": "1"},
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
