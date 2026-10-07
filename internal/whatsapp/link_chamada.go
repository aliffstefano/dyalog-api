package whatsapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/wa"
)

const (
	prefixoLinkVideo = "https://call.whatsapp.com/video/"
	prefixoLinkVoz   = "https://call.whatsapp.com/voice/"
)

// TipoLinkChamada normaliza o tipo pedido: "video" ou "audio" (o nome que o
// servidor usa para voz). Devolve "" para tipo desconhecido.
func TipoLinkChamada(tipo string) string {
	switch strings.ToLower(strings.TrimSpace(tipo)) {
	case "video", "videochamada":
		return "video"
	case "voz", "audio", "voice", "ligacao":
		return "audio"
	default:
		return ""
	}
}

// criarLinkChamada pede ao WhatsApp um link de chamada, o mesmo do botao
// "Criar link de chamada" do app. Com inicio, o link fica preso ao horario de
// um evento. Formato copiado do Baileys (createCallLink).
func criarLinkChamada(ctx context.Context, client *whatsmeow.Client, tipo string, inicio time.Time) (string, error) {
	pedido := waBinary.Node{Tag: "link_create", Attrs: waBinary.Attrs{"media": tipo}}
	if !inicio.IsZero() {
		pedido.Content = []waBinary.Node{{
			Tag:   "event",
			Attrs: waBinary.Attrs{"start_time": strconv.FormatInt(inicio.Unix(), 10)},
		}}
	}
	resposta, err := wa.NewSocket(client).Query(ctx, waBinary.Node{
		Tag: "call",
		Attrs: waBinary.Attrs{
			"id": client.DangerousInternals().GenerateRequestID(),
			"to": types.JID{Server: "call"},
		},
		Content: []waBinary.Node{pedido},
	})
	if err != nil {
		return "", fmt.Errorf("erro ao pedir link de chamada: %w", err)
	}
	if resposta == nil {
		return "", fmt.Errorf("o WhatsApp nao respondeu ao pedido de link de chamada")
	}
	token, _ := resposta.GetChildByTag("link_create").Attrs["token"].(string)
	if token == "" {
		return "", fmt.Errorf("o WhatsApp nao devolveu o link de chamada (resposta: <%s> %v)", resposta.Tag, resposta.Attrs)
	}
	if tipo == "audio" {
		return prefixoLinkVoz + token, nil
	}
	return prefixoLinkVideo + token, nil
}

// CriarLinkChamada gera um link de chamada avulso para a instancia.
func (g *GerenciadorInstancias) CriarLinkChamada(ctx context.Context, req models.LinkChamadaRequest) (models.LinkChamadaResultado, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.LinkChamadaResultado{}, err
	}
	tipo := TipoLinkChamada(req.Tipo)
	link, err := criarLinkChamada(ctx, runtime.client, tipo, time.Time{})
	if err != nil {
		return models.LinkChamadaResultado{}, err
	}
	return models.LinkChamadaResultado{Instancia: req.Instancia, Tipo: req.Tipo, Link: link}, nil
}
