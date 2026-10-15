package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	webhookdispatch "dyalog-api-go/internal/webhook"
)

func assinar(corpo, segredo string) string {
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write([]byte(corpo))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func corpoWebhookMeta(phoneID, mensagemID string) string {
	return `{"object":"whatsapp_business_account","entry":[{"id":"WABA9","changes":[{"field":"messages","value":{
		"metadata":{"phone_number_id":"` + phoneID + `"},"contacts":[{"profile":{"name":"Ana"},"wa_id":"5511999990000"}],
		"messages":[{"from":"5511999990000","id":"` + mensagemID + `","timestamp":"1791396000","type":"text","text":{"body":"oi"}}]}}]}]}`
}

func TestWebhookMetaGravadoRoteadoERetomado(t *testing.T) {
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "meta.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := st.Criar(ctx, models.Instancia{ID: "oficial", Nome: "O", Tipo: models.TipoInstanciaMeta, Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	if err := st.SalvarCredenciaisMeta(ctx, models.CredenciaisMeta{InstanciaID: "oficial", PhoneNumberID: "123", AccessToken: "tok", VerifyToken: "v"}); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	recebidas := map[string]int{}
	dispatcher := webhookdispatch.NovoDispatcher(nil, nil, time.Second, time.Second, time.Hour, time.Minute, 1, 1, 1)
	dispatcher.Observar(func(_ context.Context, instanciaID, evento string, dados interface{}) {
		if evento == models.EventoWebhookMensagens {
			mu.Lock()
			recebidas[instanciaID+"/"+dados.(map[string]any)["mensagem_id"].(string)]++
			mu.Unlock()
		}
	})
	s := NovoMetaWebhookService(st, st, st, st, meta.NovoCliente("v25.0", "http://127.0.0.1:1"), dispatcher, nil)
	s.UsarApp("segredo-app", "verifica")

	if c, err := s.VerificarApp("subscribe", "verifica", "ok"); err != nil || c != "ok" {
		t.Fatalf("verificacao do app: %q %v", c, err)
	}
	corpo := corpoWebhookMeta("123", "wamid.A")
	if err := s.ReceberApp(ctx, []byte(corpo), assinar(corpo, "outro")); !errors.Is(err, ErrAssinaturaMetaInvalida) {
		t.Fatalf("assinatura errada aceita: %v", err)
	}
	// URL unica do app: vai para a instancia pelo numero; numero desconhecido e ignorado.
	if err := s.ReceberApp(ctx, []byte(corpo), assinar(corpo, "segredo-app")); err != nil {
		t.Fatal(err)
	}
	outro := corpoWebhookMeta("999", "wamid.X")
	if err := s.ReceberApp(ctx, []byte(outro), assinar(outro, "segredo-app")); err != nil {
		t.Fatal(err)
	}
	esperar := func(chave string) {
		t.Helper()
		for i := 0; i < 100; i++ {
			mu.Lock()
			n := recebidas[chave]
			mu.Unlock()
			if n > 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("evento %s nao chegou: %v", chave, recebidas)
	}
	esperar("oficial/wamid.A")

	// Container caiu depois de gravar: fica pendente e e retomado depois.
	if _, err := st.GuardarWebhookMeta(ctx, "", []byte(corpoWebhookMeta("123", "wamid.B"))); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	pendentes, err := st.ListarWebhooksMetaPendentes(ctx, time.Now().Add(time.Minute), 10)
	if err != nil || len(pendentes) != 1 {
		t.Fatalf("pendentes = %+v, err = %v", pendentes, err)
	}
	s.processarGuardado(pendentes[0])
	s.processarGuardado(pendentes[0]) // outra replica tentando o mesmo: nao repete
	esperar("oficial/wamid.B")
	mu.Lock()
	defer mu.Unlock()
	if recebidas["oficial/wamid.B"] != 1 || recebidas["oficial/wamid.A"] != 1 || len(recebidas) != 2 {
		t.Fatalf("eventos = %v", recebidas)
	}
	if restantes, _ := st.ListarWebhooksMetaPendentes(ctx, time.Now().Add(time.Minute), 10); len(restantes) != 0 {
		t.Fatalf("ficou pendente: %+v", restantes)
	}
	cred, _ := st.ObterCredenciaisMeta(ctx, "oficial")
	if cred.WABAID != "WABA9" {
		t.Fatalf("WABA nao preenchido: %+v", cred)
	}
}
