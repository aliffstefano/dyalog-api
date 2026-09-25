package chamadas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const respostaCloudflareExemplo = `{
  "iceServers": [
    {"urls": ["stun:stun.cloudflare.com:3478"]},
    {
      "urls": [
        "turn:turn.cloudflare.com:3478?transport=udp",
        "turns:turn.cloudflare.com:443?transport=tcp"
      ],
      "username": "usuario-gerado",
      "credential": "senha-gerada"
    }
  ]
}`

func servidorCloudflareFalso(t *testing.T, chamadas *int32, corpo string, status int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(chamadas, 1)
		if r.Method != http.MethodPost {
			t.Errorf("metodo = %s, esperado POST", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/credentials/generate-ice-servers") {
			t.Errorf("caminho inesperado: %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer token-secreto" {
			t.Errorf("Authorization = %q", auth)
		}
		dados, _ := io.ReadAll(r.Body)
		var pedido map[string]int
		if err := json.Unmarshal(dados, &pedido); err != nil || pedido["ttl"] == 0 {
			t.Errorf("corpo do pedido inesperado: %s", dados)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(corpo))
	}))
}

func TestCloudflareTURNDevolveServidores(t *testing.T) {
	var chamadas int32
	srv := servidorCloudflareFalso(t, &chamadas, respostaCloudflareExemplo, http.StatusCreated)
	defer srv.Close()

	cli := novoClienteCloudflareTURN("chave-1", "token-secreto", srv.URL, time.Hour)
	servidores, err := cli.Servidores(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(servidores) != 2 {
		t.Fatalf("esperado STUN e TURN, obtido %d", len(servidores))
	}
	if servidores[1].Username != "usuario-gerado" || servidores[1].Credential != "senha-gerada" {
		t.Fatalf("credencial nao repassada: %+v", servidores[1])
	}
	// A porta 443 e o que atravessa rede que bloqueia UDP; se sumir, o caso
	// dificil volta a falhar.
	if !strings.Contains(strings.Join(servidores[1].URLs, " "), ":443") {
		t.Fatal("a URL na porta 443 deveria vir na lista")
	}
}

// Sem cache, todo inicio de chamada viraria uma ida a rede.
func TestCloudflareTURNUsaCache(t *testing.T) {
	var chamadas int32
	srv := servidorCloudflareFalso(t, &chamadas, respostaCloudflareExemplo, http.StatusCreated)
	defer srv.Close()

	cli := novoClienteCloudflareTURN("chave-1", "token-secreto", srv.URL, time.Hour)
	base := time.Now()
	for i := 0; i < 5; i++ {
		if _, err := cli.Servidores(context.Background(), base); err != nil {
			t.Fatalf("erro na chamada %d: %v", i, err)
		}
	}
	if n := atomic.LoadInt32(&chamadas); n != 1 {
		t.Fatalf("a API foi consultada %d vezes, esperado 1", n)
	}
	// Passados 80% da validade, renova.
	if _, err := cli.Servidores(context.Background(), base.Add(55*time.Minute)); err != nil {
		t.Fatalf("erro ao renovar: %v", err)
	}
	if n := atomic.LoadInt32(&chamadas); n != 2 {
		t.Fatalf("apos a validade a API deveria ser consultada de novo, obtido %d", n)
	}
}

// Se a Cloudflare falhar, e melhor entregar credencial vencendo do que nada:
// sem servidor ICE a chamada fica sem audio em rede restritiva.
func TestCloudflareTURNCaiParaCacheAntigoEmErro(t *testing.T) {
	var chamadas int32
	var falhar atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		if falhar.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"erro":"indisponivel"}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(respostaCloudflareExemplo))
	}))
	defer srv.Close()

	cli := novoClienteCloudflareTURN("chave-1", "token-secreto", srv.URL, time.Hour)
	base := time.Now()
	if _, err := cli.Servidores(context.Background(), base); err != nil {
		t.Fatalf("primeira busca falhou: %v", err)
	}
	falhar.Store(true)
	servidores, err := cli.Servidores(context.Background(), base.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("com cache disponivel nao deveria propagar o erro: %v", err)
	}
	if len(servidores) != 2 {
		t.Fatalf("o cache antigo deveria ter sido devolvido, obtido %d", len(servidores))
	}
}

func TestCloudflareTURNErroSemCache(t *testing.T) {
	var chamadas int32
	srv := servidorCloudflareFalso(t, &chamadas, `{"erro":"chave invalida"}`, http.StatusUnauthorized)
	defer srv.Close()

	cli := novoClienteCloudflareTURN("chave-1", "token-secreto", srv.URL, time.Hour)
	if _, err := cli.Servidores(context.Background(), time.Now()); err == nil {
		t.Fatal("sem cache, o erro precisa aparecer")
	}
}

// A API ja documentou iceServers como objeto unico e como lista.
func TestCloudflareTURNAceitaObjetoUnico(t *testing.T) {
	var chamadas int32
	corpo := `{"iceServers":{"urls":["turn:turn.cloudflare.com:3478"],"username":"u","credential":"c"}}`
	srv := servidorCloudflareFalso(t, &chamadas, corpo, http.StatusCreated)
	defer srv.Close()

	cli := novoClienteCloudflareTURN("chave-1", "token-secreto", srv.URL, time.Hour)
	servidores, err := cli.Servidores(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("formato de objeto unico deveria ser aceito: %v", err)
	}
	if len(servidores) != 1 || servidores[0].Username != "u" {
		t.Fatalf("servidores = %+v", servidores)
	}
}
