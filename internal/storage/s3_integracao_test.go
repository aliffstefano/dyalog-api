package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestS3EnviarFazRequisicaoValida sobe um servidor falso no lugar do S3 e confere
// que o driver monta uma requisicao que um servico compativel aceitaria: PUT no
// caminho do bucket, corpo intacto, Content-Type preservado e assinatura SigV4.
// Nao substitui um teste contra R2 ou Garage de verdade, mas pega erro de fiacao
// sem depender de rede nem de container.
func TestS3EnviarFazRequisicaoValida(t *testing.T) {
	var (
		mu          sync.Mutex
		metodo      string
		caminho     string
		tipo        string
		corpo       []byte
		autoriza    string
		md5         string
		codificacao string
		visto       bool
	)

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		dados := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(dados)
		mu.Lock()
		metodo, caminho, tipo = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		autoriza, corpo, visto = r.Header.Get("Authorization"), dados, true
		md5, codificacao = r.Header.Get("Content-Md5"), r.Header.Get("Content-Encoding")
		mu.Unlock()
		w.Header().Set("ETag", `"abc123"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer servidor.Close()

	up, err := NovoMidiaUploader(Config{
		Driver:        "s3",
		S3Endpoint:    servidor.URL,
		S3AccessKey:   "chave",
		S3SecretKey:   "segredo",
		S3Bucket:      "midias",
		S3Region:      "auto",
		PublicBaseURL: "https://midia.exemplo.com",
	})
	if err != nil {
		t.Fatalf("erro ao criar uploader: %v", err)
	}

	conteudo := []byte("conteudo-de-teste-da-midia")
	resultado, err := up.Enviar(context.Background(), "instancias/abc/20260921/foto.jpg", "image/jpeg", conteudo)
	if err != nil {
		t.Fatalf("Enviar retornou erro: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if !visto {
		t.Fatal("o servidor nao recebeu o upload")
	}
	if metodo != http.MethodPut {
		t.Fatalf("metodo = %s, esperado PUT", metodo)
	}
	if !strings.Contains(caminho, "midias") || !strings.Contains(caminho, "foto.jpg") {
		t.Fatalf("caminho = %q, esperado conter bucket e objeto", caminho)
	}
	if tipo != "image/jpeg" {
		t.Fatalf("Content-Type = %q, esperado image/jpeg", tipo)
	}
	if string(corpo) != string(conteudo) {
		t.Fatalf("corpo enviado = %q, esperado %q", corpo, conteudo)
	}
	if !strings.HasPrefix(autoriza, "AWS4-HMAC-SHA256") {
		t.Fatalf("Authorization = %q, esperado assinatura SigV4", autoriza)
	}
	if resultado.Provider != "s3" {
		t.Fatalf("Provider = %q, esperado s3", resultado.Provider)
	}
	if resultado.URL != "https://midia.exemplo.com/instancias/abc/20260921/foto.jpg" {
		t.Fatalf("URL = %q", resultado.URL)
	}
	// As duas assercoes abaixo travam a decisao de compatibilidade tomada no
	// driver. Sem elas, alguem remove DisableContentSha256 no futuro, o minio-go
	// volta a assinatura streaming em endpoint HTTP e o upload quebra so em
	// producao, contra o servico do cliente.
	if strings.Contains(strings.ToLower(codificacao), "chunked") {
		t.Fatalf("Content-Encoding = %q, o corpo nao deve ir em aws-chunked", codificacao)
	}
	if md5 == "" {
		t.Fatal("Content-Md5 ausente: o servidor perde a conferencia de integridade")
	}
}
