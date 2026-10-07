package meta

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dyalog-api-go/internal/models"
)

func TestDestinatario(t *testing.T) {
	casos := map[[2]string]string{
		{"+55 (67) 99944-0667", ""}:             "5567999440667",
		{"", "5567999440667@s.whatsapp.net"}:    "5567999440667",
		{"", "5567999440667:12@s.whatsapp.net"}: "5567999440667",
	}
	for entrada, esperado := range casos {
		if got, err := Destinatario(entrada[0], entrada[1]); err != nil || got != esperado {
			t.Fatalf("Destinatario(%q,%q) = %q, %v", entrada[0], entrada[1], got, err)
		}
	}
	if _, err := Destinatario("", "120363@g.us"); !errors.Is(err, ErrNaoSuportado) {
		t.Fatalf("grupo deveria dar ErrNaoSuportado, veio %v", err)
	}
}

func TestMontarBotoes(t *testing.T) {
	resposta := models.EnvioBotoesRequest{Titulo: "Oi", Rodape: "Dyalog", Botoes: []models.BotaoRequest{{ID: "s", Texto: "Sim"}, {ID: "n", Texto: "Nao"}}}
	msg, err := MontarBotoes("55", resposta, "Confirma?")
	if err != nil {
		t.Fatal(err)
	}
	inter := msg["interactive"].(map[string]any)
	if inter["type"] != "button" || len(inter["action"].(map[string]any)["buttons"].([]map[string]any)) != 2 {
		t.Fatalf("botoes de resposta mal montados: %v", inter)
	}

	link := models.EnvioBotoesRequest{Botoes: []models.BotaoRequest{{Tipo: "url", Texto: "Site", URL: "https://x.com"}}}
	msg, err = MontarBotoes("55", link, "Veja")
	if err != nil || msg["interactive"].(map[string]any)["type"] != "cta_url" {
		t.Fatalf("botao de link deveria virar cta_url: %v %v", msg, err)
	}

	misto := models.EnvioBotoesRequest{Botoes: []models.BotaoRequest{{ID: "s", Texto: "Sim"}, {Tipo: "url", Texto: "Site", URL: "https://x.com"}}}
	if _, err := MontarBotoes("55", misto, "x"); !errors.Is(err, ErrNaoSuportado) {
		t.Fatalf("misturar resposta e link deveria dar ErrNaoSuportado, veio %v", err)
	}
	copiar := models.EnvioBotoesRequest{Botoes: []models.BotaoRequest{{Tipo: "copy", Texto: "Copiar", Codigo: "X"}}}
	if _, err := MontarBotoes("55", copiar, "x"); !errors.Is(err, ErrNaoSuportado) {
		t.Fatalf("botao copiar deveria dar ErrNaoSuportado, veio %v", err)
	}
}

func TestMontarListaLimites(t *testing.T) {
	linhas := func(n int) []models.ListaLinhaRequest {
		l := make([]models.ListaLinhaRequest, n)
		for i := range l {
			l[i] = models.ListaLinhaRequest{ID: "i", Titulo: "Item"}
		}
		return l
	}
	req := models.EnvioListaRequest{BotaoTexto: "Ver", Secoes: []models.ListaSecaoRequest{{Titulo: "A", Linhas: linhas(6)}, {Titulo: "B", Linhas: linhas(4)}}}
	msg, err := MontarLista("55", req, "Escolha")
	if err != nil {
		t.Fatal(err)
	}
	if msg["interactive"].(map[string]any)["type"] != "list" {
		t.Fatalf("lista mal montada: %v", msg)
	}
	req.Secoes[1].Linhas = linhas(5)
	if _, err := MontarLista("55", req, "Escolha"); err == nil {
		t.Fatal("11 linhas deveria passar do limite da Meta")
	}
}

func TestMontarTemplateComParametros(t *testing.T) {
	msg := MontarTemplate("55", models.EnvioTemplateRequest{Nome: "pedido", Idioma: "en_US", Parametros: []string{"John", "123"}})
	dados, _ := json.Marshal(msg)
	texto := string(dados)
	for _, trecho := range []string{`"name":"pedido"`, `"code":"en_US"`, `"type":"body"`, `"text":"John"`} {
		if !strings.Contains(texto, trecho) {
			t.Fatalf("template sem %s: %s", trecho, texto)
		}
	}
}

func TestClienteEnviarEErro(t *testing.T) {
	var recebido map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("token nao enviado")
		}
		corpo, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(corpo, &recebido)
		if strings.Contains(string(corpo), "fora_janela") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Re-engagement message","type":"OAuthException","code":131047,"error_data":{"details":"Message failed to send because more than 24 hours have passed"}}}`))
			return
		}
		if r.URL.Path != "/v25.0/123/messages" {
			t.Errorf("caminho = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","contacts":[{"input":"5567","wa_id":"556799440667"}],"messages":[{"id":"wamid.X"}]}`))
	}))
	defer srv.Close()
	c := NovoCliente("", srv.URL)
	cred := models.CredenciaisMeta{PhoneNumberID: "123", AccessToken: "tok"}

	resp, err := c.Enviar(context.Background(), cred, MontarTexto("5567999440667", "oi", ""))
	if err != nil || resp.MensagemID() != "wamid.X" || resp.WaID() != "556799440667" {
		t.Fatalf("envio: %+v %v", resp, err)
	}
	if recebido["to"] != "5567999440667" || recebido["type"] != "text" {
		t.Fatalf("payload enviado: %v", recebido)
	}

	_, err = c.Enviar(context.Background(), cred, MontarTexto("55", "fora_janela", ""))
	if !ErroJanela24h(err) || !strings.Contains(err.Error(), "template") {
		t.Fatalf("erro de janela 24h nao reconhecido: %v", err)
	}
}
