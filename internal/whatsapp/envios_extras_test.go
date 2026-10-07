package whatsapp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

func TestBotaoNativeFlowTipos(t *testing.T) {
	casos := []struct {
		botao  models.BotaoRequest
		nome   string
		params map[string]string
	}{
		{models.BotaoRequest{Texto: "Sim", ID: "s"}, "quick_reply", map[string]string{"display_text": "Sim", "id": "s"}},
		{models.BotaoRequest{Texto: "Site", Tipo: "url", URL: "https://x.com"}, "cta_url", map[string]string{"url": "https://x.com"}},
		{models.BotaoRequest{Texto: "Ligar", Tipo: "ligar", Telefone: "+5567999999999"}, "cta_call", map[string]string{"phone_number": "+5567999999999"}},
		{models.BotaoRequest{Texto: "Copiar", Tipo: "copiar", Codigo: "CUPOM10"}, "cta_copy", map[string]string{"copy_code": "CUPOM10", "id": "CUPOM10"}},
	}
	for _, caso := range casos {
		botao, err := botaoNativeFlow(caso.botao)
		if err != nil {
			t.Fatalf("%s: %v", caso.nome, err)
		}
		if botao.GetName() != caso.nome {
			t.Fatalf("name = %q, esperado %q", botao.GetName(), caso.nome)
		}
		var params map[string]string
		if err := json.Unmarshal([]byte(botao.GetButtonParamsJSON()), &params); err != nil {
			t.Fatal(err)
		}
		for chave, valor := range caso.params {
			if params[chave] != valor {
				t.Fatalf("%s: %s = %q, esperado %q", caso.nome, chave, params[chave], valor)
			}
		}
	}
}

func TestBotoesMistosNativeFlow(t *testing.T) {
	msg, err := montarMensagemBotoesNativeFlowDireto(models.EnvioBotoesRequest{
		Texto: "Escolha",
		Botoes: []models.BotaoRequest{
			{Texto: "Sim", ID: "s"},
			{Texto: "Site", Tipo: "url", URL: "https://x.com"},
		},
	})
	if err != nil {
		t.Fatalf("botoes mistos deveriam montar: %v", err)
	}
	if n := len(msg.GetInteractiveMessage().GetNativeFlowMessage().GetButtons()); n != 2 {
		t.Fatalf("botoes = %d, esperado 2", n)
	}
}

func TestMontarMensagemEvento(t *testing.T) {
	inicio := time.Date(2026, 10, 20, 19, 0, 0, 0, time.UTC)
	msg, err := montarMensagemEvento(models.EnvioEventoRequest{
		Nome: "Reuniao", Local: "Escritorio", LinkChamada: "https://call.whatsapp.com/x",
		InicioEm: inicio, FimEm: inicio.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	evento := msg.GetEventMessage()
	if evento.GetStartTime() != inicio.Unix() || evento.GetEndTime() != inicio.Add(time.Hour).Unix() {
		t.Fatalf("horarios errados: %d %d", evento.GetStartTime(), evento.GetEndTime())
	}
	if evento.GetLocation().GetName() != "Escritorio" || evento.GetJoinLink() == "" {
		t.Fatalf("local/link ausentes")
	}
	if len(msg.GetMessageContextInfo().GetMessageSecret()) != 32 {
		t.Fatalf("evento precisa de message secret de 32 bytes")
	}
}

func TestCorARGB(t *testing.T) {
	if cor := corARGB("#FF0000", 1); cor != 0xFFFF0000 {
		t.Fatalf("cor = %x", cor)
	}
	if cor := corARGB("azul", 7); cor != 7 {
		t.Fatalf("cor invalida deveria usar o padrao, veio %x", cor)
	}
}

func TestTipoStatus(t *testing.T) {
	if tipoStatus(models.EnvioStatusRequest{Texto: "oi"}) != "texto" {
		t.Fatal("sem arquivo deveria ser texto")
	}
	if tipoStatus(models.EnvioStatusRequest{ArquivoURL: "https://x/a.jpg"}) != "imagem" {
		t.Fatal("arquivo sem mime deveria ser imagem")
	}
	if tipoStatus(models.EnvioStatusRequest{ArquivoURL: "https://x/a.mp4", MimeType: "video/mp4"}) != "video" {
		t.Fatal("mime de video deveria ser video")
	}
}

func TestCarrosselLevaEnvelopeBiz(t *testing.T) {
	cartao := &waE2E.InteractiveMessage{
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String("Produto")},
	}
	msg := montarMensagemCarrossel(models.EnvioCarrosselRequest{Texto: "Ofertas"}, []*waE2E.InteractiveMessage{cartao})
	nos := nosInterativoNativeFlow(msg, types.NewJID("5567999999999", types.DefaultUserServer))
	if len(nos) != 1 || nos[0].Tag != "biz" {
		t.Fatalf("carrossel deveria levar <biz>, veio %v", nos)
	}
}

func TestTextoFallbackCarrossel(t *testing.T) {
	texto := textoFallbackCarrossel(models.EnvioCarrosselRequest{
		Texto: "Ofertas",
		Cartoes: []models.CarrosselCartaoRequest{{
			Titulo: "Tenis", Texto: "R$ 199",
			Botoes: []models.BotaoRequest{{Texto: "Comprar", Tipo: "url", URL: "https://loja/tenis"}},
		}},
	})
	for _, esperado := range []string{"Ofertas", "*Tenis*", "R$ 199", "Comprar: https://loja/tenis"} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("texto sem %q:\n%s", esperado, texto)
		}
	}
}
