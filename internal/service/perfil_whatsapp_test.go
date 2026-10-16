package service

import (
	"testing"

	"dyalog-api-go/internal/models"
)

func TestFiltrarContatos(t *testing.T) {
	todos := []models.ContatoWhatsApp{
		{JID: "1", Numero: "5567999990001", Nome: "Ana Souza", Tipo: "pessoal", Salvo: true},
		{JID: "2", Numero: "5567999990002", Nome: "Bruno", NomePush: "Bru", Tipo: "pessoal", Bloqueado: true},
		{JID: "3", Nome: "Equipe Vendas", Tipo: "grupo", Salvo: true},
		{JID: "4", Numero: "5511988887777", Nome: "", Tipo: "pessoal"},
	}
	if l := filtrarContatos(todos, FiltroContatos{}); l.Total != 4 || l.Contagem["grupo"] != 1 || l.Contagem["bloqueado"] != 1 || l.Contagem["pessoal"] != 3 {
		t.Fatalf("todos = %+v", l)
	}
	if l := filtrarContatos(todos, FiltroContatos{Tipo: "grupo"}); l.Total != 1 || l.Contatos[0].JID != "3" {
		t.Fatalf("grupo = %+v", l)
	}
	if l := filtrarContatos(todos, FiltroContatos{Busca: "bru"}); l.Total != 1 {
		t.Fatalf("busca nome = %+v", l)
	}
	if l := filtrarContatos(todos, FiltroContatos{Busca: "(11) 98888"}); l.Total != 1 || l.Contatos[0].JID != "4" {
		t.Fatalf("busca numero = %+v", l)
	}
	if l := filtrarContatos(todos, FiltroContatos{Tipo: "salvo"}); l.Total != 1 {
		t.Fatalf("salvo = %+v", l)
	}
	l := filtrarContatos(todos, FiltroContatos{PorPagina: 3, Pagina: 9})
	if l.Paginas != 2 || l.Pagina != 2 || len(l.Contatos) != 1 {
		t.Fatalf("paginacao = %+v", l)
	}
}
