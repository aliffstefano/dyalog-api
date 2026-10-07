package service

import (
	"encoding/json"
	"testing"

	"dyalog-api-go/internal/models"
)

// Payload no formato number/contentText/buttonText/sections, usado por outras
// APIs de WhatsApp.
func TestNormalizarListaCompatFormatoSections(t *testing.T) {
	corpo := `{
		"number": "5511999999999",
		"contentText": "Selecione um item do cardapio:",
		"buttonText": "Ver opcoes",
		"sections": [{
			"title": "Pratos",
			"rows": [
				{"id": "p1", "title": "Lasanha", "description": "Bolonhesa, 4 fatias"},
				{"id": "p2", "title": "Risoto", "description": "Funghi"}
			]
		}]
	}`
	var req models.EnvioListaRequest
	if err := json.Unmarshal([]byte(corpo), &req); err != nil {
		t.Fatalf("json invalido: %v", err)
	}
	req = normalizarListaCompat(req)

	if req.Numero != "5511999999999" {
		t.Fatalf("numero = %q", req.Numero)
	}
	if req.Descricao != "Selecione um item do cardapio:" {
		t.Fatalf("descricao = %q", req.Descricao)
	}
	if req.BotaoTexto != "Ver opcoes" {
		t.Fatalf("botao_texto = %q", req.BotaoTexto)
	}
	if len(req.Secoes) != 1 || req.Secoes[0].Titulo != "Pratos" || len(req.Secoes[0].Linhas) != 2 {
		t.Fatalf("secoes = %#v", req.Secoes)
	}
	linha := req.Secoes[0].Linhas[0]
	if linha.ID != "p1" || linha.Titulo != "Lasanha" || linha.Descricao != "Bolonhesa, 4 fatias" {
		t.Fatalf("linha = %#v", linha)
	}
}

func TestVersaoMaisNova(t *testing.T) {
	casos := []struct {
		disponivel, emUso string
		esperado          bool
	}{
		{"v0.0.0-20261006124319-9399289b022b", "v0.0.0-20260929112325-8b41cfe6d9c4", true},
		{"v0.0.0-20260929112325-8b41cfe6d9c4", "v0.0.0-20261006124319-9399289b022b", false},
		{"v0.0.0-20261006124319-9399289b022b", "v0.0.0-20261006124319-9399289b022b", false},
		{"", "v0.0.0-20261006124319-9399289b022b", false},
		{"v0.1.0", "(devel)", true},
	}
	for _, caso := range casos {
		if got := versaoMaisNova(caso.disponivel, caso.emUso); got != caso.esperado {
			t.Errorf("versaoMaisNova(%q, %q) = %v", caso.disponivel, caso.emUso, got)
		}
	}
}
