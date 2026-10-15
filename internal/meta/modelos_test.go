package meta

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"dyalog-api-go/internal/models"
)

func TestListarModelosComPaginacao(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" || r.URL.Path != "/v25.0/WABA/message_templates" {
			http.Error(w, `{"error":{"message":"rota","code":100}}`, 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{"data":[{"id":"1","name":"pedido_confirmado","language":"pt_BR","status":"APPROVED","category":"UTILITY",
				"components":[{"type":"HEADER","format":"TEXT","text":"Pedido {{1}}"},
				{"type":"BODY","text":"Ola {{1}}, seu pedido {{2}} saiu. Obrigado, {{1}}!"},
				{"type":"FOOTER","text":"Loja"},
				{"type":"BUTTONS","buttons":[{"type":"QUICK_REPLY","text":"Rastrear"}]}]}],
				"paging":{"next":"` + srv.URL + `/v25.0/WABA/message_templates?after=X"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"2","name":"boas_vindas","language":"pt_BR","status":"PENDING","category":"MARKETING","parameter_format":"NAMED",
			"components":[{"type":"HEADER","format":"IMAGE"},{"type":"BODY","text":"Oi {{nome}}!"}]}]}`))
	}))
	defer srv.Close()

	c := NovoCliente("v25.0", srv.URL)
	if _, err := c.ListarModelos(context.Background(), models.CredenciaisMeta{AccessToken: "tok"}); !errors.Is(err, ErrSemWABA) {
		t.Fatalf("sem WABA deveria dar ErrSemWABA, veio %v", err)
	}
	modelos, err := c.ListarModelos(context.Background(), models.CredenciaisMeta{AccessToken: "tok", WABAID: "WABA"})
	if err != nil || len(modelos) != 2 {
		t.Fatalf("modelos = %+v, err = %v", modelos, err)
	}
	m := modelos[0]
	if !m.Aprovado() || m.Cabecalho != "Pedido {{1}}" || len(m.VariaveisCabecalho) != 1 || len(m.VariaveisCorpo) != 2 ||
		m.Rodape != "Loja" || len(m.Botoes) != 1 || m.FormatoParametros != "positional" {
		t.Fatalf("modelo mal resumido: %+v", m)
	}
	if texto := m.Renderizar(ValoresModelo(models.EnvioTemplateRequest{Parametros: []string{"Ana", "#42"}})); texto != "Ola Ana, seu pedido #42 saiu. Obrigado, Ana!" {
		t.Fatalf("texto = %q", texto)
	}
	n := modelos[1]
	if n.Aprovado() || n.CabecalhoFormato != "IMAGE" || n.FormatoParametros != "named" || len(n.VariaveisCorpo) != 1 || n.VariaveisCorpo[0] != "nome" {
		t.Fatalf("modelo nomeado mal resumido: %+v", n)
	}
	valores := ValoresModelo(models.EnvioTemplateRequest{Componentes: []map[string]interface{}{
		{"type": "body", "parameters": []interface{}{map[string]interface{}{"type": "text", "parameter_name": "nome", "text": "Bia"}}},
	}})
	if texto := n.Renderizar(valores); texto != "Oi Bia!" {
		t.Fatalf("texto nomeado = %q", texto)
	}
}
