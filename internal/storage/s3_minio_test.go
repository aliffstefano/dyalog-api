package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestS3ContraServidorReal sobe o driver contra um servidor S3 de verdade e faz o
// ciclo completo: cria bucket, envia, le de volta e confere os bytes.
//
// Fica desligado por padrao. Para rodar, suba um MinIO e aponte as variaveis:
//
//	docker run -d --name minio-teste -p 19000:9000 \
//	  -e MINIO_ROOT_USER=testekey -e MINIO_ROOT_PASSWORD=testesegredo123 \
//	  quay.io/minio/minio:latest server /data
//
//	MINIO_TESTE_ENDPOINT=http://127.0.0.1:19000 go test ./internal/storage/ -run Real -v
//
// Serve tambem para validar contra R2 ou Garage: e so trocar as variaveis.
func TestS3ContraServidorReal(t *testing.T) {
	endpoint := os.Getenv("MINIO_TESTE_ENDPOINT")
	if endpoint == "" {
		t.Skip("MINIO_TESTE_ENDPOINT nao definido; pulando teste contra servidor real")
	}
	accessKey := valorOuPadrao("MINIO_TESTE_ACCESS_KEY", "testekey")
	secretKey := valorOuPadrao("MINIO_TESTE_SECRET_KEY", "testesegredo123")
	bucket := valorOuPadrao("MINIO_TESTE_BUCKET", "dyalog-midias")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	host, seguro, err := separarEndpointS3(endpoint)
	if err != nil {
		t.Fatalf("endpoint invalido: %v", err)
	}
	admin, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: seguro,
	})
	if err != nil {
		t.Fatalf("erro ao conectar no servidor: %v", err)
	}
	existe, err := admin.BucketExists(ctx, bucket)
	if err != nil {
		t.Fatalf("erro ao consultar bucket: %v", err)
	}
	if !existe {
		if err := admin.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("erro ao criar bucket: %v", err)
		}
	}

	up, err := NovoMidiaUploader(Config{
		Driver:      "s3",
		S3Endpoint:  endpoint,
		S3AccessKey: accessKey,
		S3SecretKey: secretKey,
		S3Bucket:    bucket,
		S3Region:    "us-east-1",
	})
	if err != nil {
		t.Fatalf("erro ao criar uploader: %v", err)
	}

	// Conteudo binario de proposito: base64 ou texto esconderiam corrupcao.
	conteudo := make([]byte, 4096)
	for i := range conteudo {
		conteudo[i] = byte(i % 251)
	}
	objectPath := "instancias/teste/20260921/arquivo binario.bin"

	resultado, err := up.Enviar(ctx, objectPath, "application/octet-stream", conteudo)
	if err != nil {
		t.Fatalf("Enviar falhou contra servidor real: %v", err)
	}
	if resultado.Provider != "s3" || resultado.ObjectPath != objectPath {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}

	obj, err := admin.GetObject(ctx, bucket, objectPath, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("erro ao ler objeto: %v", err)
	}
	defer obj.Close()
	lido, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("erro ao baixar objeto: %v", err)
	}
	if !bytes.Equal(lido, conteudo) {
		t.Fatalf("conteudo baixado difere: %d bytes contra %d enviados", len(lido), len(conteudo))
	}

	info, err := admin.StatObject(ctx, bucket, objectPath, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("erro no stat: %v", err)
	}
	if info.ContentType != "application/octet-stream" {
		t.Fatalf("ContentType = %q, esperado application/octet-stream", info.ContentType)
	}

	// Reenviar o mesmo caminho deve sobrescrever, nao duplicar nem falhar:
	// a API reenvia midia quando um retry de webhook reprocessa a mensagem.
	if _, err := up.Enviar(ctx, objectPath, "application/octet-stream", conteudo); err != nil {
		t.Fatalf("reenvio do mesmo objeto falhou: %v", err)
	}

	t.Logf("upload real validado: %s -> %s (%d bytes)", objectPath, resultado.URL, len(conteudo))
}

func valorOuPadrao(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
