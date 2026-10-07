package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestCredenciaisMetaETipoDaInstancia(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now()
	for _, inst := range []models.Instancia{
		{ID: "qr", Nome: "QR", Token: "t1", Status: "desconectada", CriadoEm: agora, AtualizadoEm: agora},
		{ID: "oficial", Nome: "Oficial", Tipo: models.TipoInstanciaMeta, Token: "t2", Status: "desconectada", CriadoEm: agora, AtualizadoEm: agora},
	} {
		if _, err := s.Criar(ctx, inst); err != nil {
			t.Fatal(err)
		}
	}
	qr, _ := s.BuscarPorID(ctx, "qr")
	oficial, _ := s.BuscarPorID(ctx, "oficial")
	if qr.Tipo != models.TipoInstanciaWhatsApp || !oficial.EhMeta() {
		t.Fatalf("tipos = %q %q", qr.Tipo, oficial.Tipo)
	}

	if _, err := s.ObterCredenciaisMeta(ctx, "oficial"); !errors.Is(err, ErrCredenciaisMetaNaoEncontradas) {
		t.Fatalf("esperado ErrCredenciaisMetaNaoEncontradas, veio %v", err)
	}
	cred := models.CredenciaisMeta{InstanciaID: "oficial", PhoneNumberID: "123", AccessToken: "tok", VerifyToken: "v"}
	if err := s.SalvarCredenciaisMeta(ctx, cred); err != nil {
		t.Fatal(err)
	}
	cred.AccessToken = "tok2"
	if err := s.SalvarCredenciaisMeta(ctx, cred); err != nil {
		t.Fatal(err)
	}
	lida, err := s.ObterCredenciaisMeta(ctx, "oficial")
	if err != nil || lida.AccessToken != "tok2" || lida.PhoneNumberID != "123" {
		t.Fatalf("credenciais lidas = %+v, %v", lida, err)
	}
	// O mesmo numero da Meta nao pode estar em duas instancias.
	if err := s.SalvarCredenciaisMeta(ctx, models.CredenciaisMeta{InstanciaID: "qr", PhoneNumberID: "123"}); err == nil {
		t.Fatal("esperado erro de phone_number_id repetido")
	}
	// Excluir a instancia apaga as credenciais junto.
	if err := s.Excluir(ctx, "oficial"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ObterCredenciaisMeta(ctx, "oficial"); !errors.Is(err, ErrCredenciaisMetaNaoEncontradas) {
		t.Fatalf("credenciais deveriam sumir com a instancia, veio %v", err)
	}
}
