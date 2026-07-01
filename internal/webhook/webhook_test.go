package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

func TestDispatcherEntregaWebhookComRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := novoStoreTeste(t)
	instancia := criarInstanciaTeste(t, st)

	var chamadas atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chamada := chamadas.Add(1)
		if r.Header.Get("X-Dyalog-Evento-Tipo") != models.EventoWebhookMensagens {
			t.Errorf("header X-Dyalog-Evento-Tipo inesperado: %s", r.Header.Get("X-Dyalog-Evento-Tipo"))
		}
		var payload models.EventoWebhook
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("payload invalido: %v", err)
		}
		if chamada == 1 {
			http.Error(w, "falha temporaria", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	webhookCriado := models.WebhookInstancia{
		ID:           "webhook-1",
		InstanciaID:  instancia.ID,
		Nome:         "Teste",
		URL:          server.URL,
		Eventos:      []string{models.EventoWebhookMensagens},
		Ativo:        true,
		CriadoEm:     time.Now().UTC(),
		AtualizadoEm: time.Now().UTC(),
	}
	if _, err := st.CriarWebhook(ctx, webhookCriado); err != nil {
		t.Fatalf("erro ao criar webhook: %v", err)
	}

	dispatcher := NovoDispatcher(st, st, 2*time.Second, 10*time.Millisecond, time.Hour, time.Minute, 3, 2, 10)
	dispatcher.Iniciar(ctx)
	dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookMensagens, map[string]string{"mensagem": "teste"})

	waitUntil(t, 3*time.Second, func() bool {
		entregas, err := st.ListarWebhookEntregas(ctx, instancia.ID, 10)
		if err != nil || len(entregas) != 1 {
			return false
		}
		return entregas[0].Status == models.WebhookEntregaEntregue && entregas[0].Tentativas == 2 && entregas[0].StatusHTTP == http.StatusNoContent
	})
	if chamadas.Load() != 2 {
		t.Fatalf("esperava 2 chamadas HTTP, recebeu %d", chamadas.Load())
	}
}

func TestDispatcherMarcaEntregaEsgotada(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := novoStoreTeste(t)
	instancia := criarInstanciaTeste(t, st)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fora do ar", http.StatusBadGateway)
	}))
	defer server.Close()

	webhookCriado := models.WebhookInstancia{
		ID:           "webhook-2",
		InstanciaID:  instancia.ID,
		Nome:         "Falha",
		URL:          server.URL,
		Eventos:      []string{models.EventoWebhookMensagens},
		Ativo:        true,
		CriadoEm:     time.Now().UTC(),
		AtualizadoEm: time.Now().UTC(),
	}
	if _, err := st.CriarWebhook(ctx, webhookCriado); err != nil {
		t.Fatalf("erro ao criar webhook: %v", err)
	}

	dispatcher := NovoDispatcher(st, st, time.Second, 5*time.Millisecond, time.Hour, time.Minute, 2, 2, 10)
	dispatcher.Iniciar(ctx)
	dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookMensagens, map[string]string{"mensagem": "teste"})

	waitUntil(t, 3*time.Second, func() bool {
		entregas, err := st.ListarWebhookEntregas(ctx, instancia.ID, 10)
		if err != nil || len(entregas) != 1 {
			return false
		}
		return entregas[0].Status == models.WebhookEntregaEsgotada && entregas[0].Tentativas == 2 && entregas[0].StatusHTTP == http.StatusBadGateway
	})
}

func novoStoreTeste(t *testing.T) *store.SQLStore {
	t.Helper()
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "dyalog.db"))
	if err != nil {
		t.Fatalf("erro ao preparar store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

func criarInstanciaTeste(t *testing.T, st *store.SQLStore) models.Instancia {
	t.Helper()
	agora := time.Now().UTC()
	instancia := models.Instancia{
		ID:           "instancia-teste",
		Nome:         "Instancia teste",
		Token:        "token-teste",
		Status:       models.StatusInstanciaConectada,
		ProxyModo:    models.ProxyModoHerdar,
		Presenca:     models.PresencaIndisponivel,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}
	criada, err := st.Criar(context.Background(), instancia)
	if err != nil {
		t.Fatalf("erro ao criar instancia: %v", err)
	}
	return criada
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condicao nao atendida em %s", timeout)
}
