package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/store"
)

func TestCadastroIncorporado(t *testing.T) {
	var mu sync.Mutex
	var chamadas []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		chamadas = append(chamadas, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v25.0/oauth/access_token":
			if r.URL.Query().Get("code") != "CODE" || r.URL.Query().Get("client_secret") != "segredo" {
				w.WriteHeader(400)
				_, _ = w.Write([]byte(`{"error":{"message":"code invalido","code":100}}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"tok-empresa"}`))
		case r.Header.Get("Authorization") != "Bearer tok-empresa":
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":{"message":"token","code":190}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v25.0/PN1":
			_, _ = w.Write([]byte(`{"id":"PN1","display_phone_number":"+55 67 3333-0000","verified_name":"Loja"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v25.0/WABA1/subscribed_apps":
			_, _ = w.Write([]byte(`{"success":true}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v25.0/PN1/register":
			var corpo map[string]string
			_ = json.NewDecoder(r.Body).Decode(&corpo)
			if len(corpo["pin"]) != 6 {
				w.WriteHeader(400)
				return
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			w.WriteHeader(404)
			_, _ = w.Write([]byte(`{"error":{"message":"rota","code":100}}`))
		}
	}))
	defer srv.Close()

	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := NovoInstanciaService(st, nil)
	s.UsarMeta(meta.NovoEnviador(meta.NovoCliente("v25.0", srv.URL), st), st, "https://api.exemplo.com")
	ctx := context.Background()

	if _, _, err := s.CadastrarViaMeta(ctx, CadastroMetaRequest{Nome: "Loja", Code: "CODE", WABAID: "WABA1", PhoneNumberID: "PN1"}); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("sem app configurado deveria recusar: %v", err)
	}
	s.UsarAppMeta(AppMeta{AppID: "APP", AppSecret: "segredo", ConfigID: "CFG", Versao: "v25.0"})
	if cfg := s.ConfigCadastroMeta(); cfg["habilitado"] != true || cfg["app_id"] != "APP" || cfg["config_id"] != "CFG" {
		t.Fatalf("config = %v", cfg)
	}
	if _, _, err := s.CadastrarViaMeta(ctx, CadastroMetaRequest{Nome: "Loja", Code: "CODE", WABAID: "WABA1"}); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("sem numero deveria recusar: %v", err)
	}
	if _, _, err := s.CadastrarViaMeta(ctx, CadastroMetaRequest{Nome: "Loja", Code: "VELHO", WABAID: "WABA1", PhoneNumberID: "PN1"}); err == nil || !strings.Contains(err.Error(), "code") {
		t.Fatalf("code invalido deveria recusar: %v", err)
	}

	instancia, avisos, err := s.CadastrarViaMeta(ctx, CadastroMetaRequest{Nome: "Loja", Code: "CODE", WABAID: "WABA1", PhoneNumberID: "PN1"})
	if err != nil || len(avisos) != 0 {
		t.Fatalf("cadastro: %v avisos=%v", err, avisos)
	}
	if !instancia.EhMeta() || instancia.Status != "conectada" {
		t.Fatalf("instancia = %+v", instancia)
	}
	cred, _ := st.ObterCredenciaisMeta(ctx, instancia.ID)
	if cred.AccessToken != "tok-empresa" || cred.AppSecret != "segredo" || cred.WABAID != "WABA1" || len(cred.PinRegistro) != 6 || cred.VerifyToken == "" {
		t.Fatalf("credenciais = %+v", cred)
	}
	mu.Lock()
	defer mu.Unlock()
	feitas := strings.Join(chamadas, ",")
	for _, esperada := range []string{"POST /v25.0/WABA1/subscribed_apps", "POST /v25.0/PN1/register"} {
		if !strings.Contains(feitas, esperada) {
			t.Fatalf("faltou %s em %s", esperada, feitas)
		}
	}
}
