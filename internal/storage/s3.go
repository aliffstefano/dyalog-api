package storage

import (
	"bytes"
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// s3Uploader fala o protocolo S3 e serve qualquer servico compativel: Cloudflare
// R2, Garage, MinIO, Backblaze B2 e afins. Trocar de um para outro e so mudar as
// variaveis de ambiente, sem rebuild.
//
// Usamos minio-go em vez do SDK oficial da AWS de proposito: versoes recentes do
// aws-sdk-go-v2 mandam headers de checksum por padrao que varios servicos
// compativeis rejeitam, o R2 entre eles, e o sintoma aparece como erro de
// assinatura, que e chato de diagnosticar.
type s3Uploader struct {
	client        *minio.Client
	bucket        string
	endpoint      string
	publicBaseURL string
	seguro        bool
}

func novoS3Uploader(cfg Config, publicBaseURL string) (MidiaUploader, error) {
	endpoint := strings.TrimSpace(cfg.S3Endpoint)
	accessKey := strings.TrimSpace(cfg.S3AccessKey)
	secretKey := strings.TrimSpace(cfg.S3SecretKey)
	bucket := strings.TrimSpace(cfg.S3Bucket)
	regiao := strings.TrimSpace(cfg.S3Region)

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("MEDIA_STORAGE_S3_ENDPOINT, MEDIA_STORAGE_S3_ACCESS_KEY, MEDIA_STORAGE_S3_SECRET_KEY e MEDIA_STORAGE_S3_BUCKET sao obrigatorios para storage s3")
	}

	host, seguro, err := separarEndpointS3(endpoint)
	if err != nil {
		return nil, err
	}

	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: seguro,
		Region: regiao,
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao configurar storage s3: %w", err)
	}

	return &s3Uploader{
		client:        client,
		bucket:        bucket,
		endpoint:      host,
		publicBaseURL: publicBaseURL,
		seguro:        seguro,
	}, nil
}

// separarEndpointS3 aceita o endpoint com ou sem esquema e devolve o host puro,
// que e o formato que o minio-go espera, mais o flag de TLS. Sem esquema, assume
// https: endpoint de producao sem TLS e erro mais provavel que intencao.
func separarEndpointS3(endpoint string) (string, bool, error) {
	if !strings.Contains(endpoint, "://") {
		return strings.TrimRight(endpoint, "/"), true, nil
	}
	parsed, err := neturl.Parse(endpoint)
	if err != nil {
		return "", false, fmt.Errorf("MEDIA_STORAGE_S3_ENDPOINT invalido: %w", err)
	}
	if parsed.Host == "" {
		return "", false, fmt.Errorf("MEDIA_STORAGE_S3_ENDPOINT invalido: informe o host")
	}
	switch parsed.Scheme {
	case "https":
		return parsed.Host, true, nil
	case "http":
		return parsed.Host, false, nil
	default:
		return "", false, fmt.Errorf("MEDIA_STORAGE_S3_ENDPOINT invalido: esquema %q nao suportado, use http ou https", parsed.Scheme)
	}
}

func (s *s3Uploader) Enviar(ctx context.Context, objectPath, mimeType string, dados []byte) (ResultadoUpload, error) {
	objectPath = limparObjectPath(objectPath)
	if objectPath == "" {
		return ResultadoUpload{}, fmt.Errorf("caminho do objeto da midia vazio")
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	// DisableContentSha256 evita a assinatura streaming do minio-go, que quebra o
	// corpo em blocos com cabecalho aws-chunked. O minio-go so recorre a ela em
	// conexao sem TLS, mas varios servicos S3-compatible rejeitam esse formato, e o
	// sintoma aparece como erro de assinatura. Como troca, mandamos Content-MD5:
	// o servidor continua conferindo integridade, sem depender do chunked.
	_, err := s.client.PutObject(ctx, s.bucket, objectPath, bytes.NewReader(dados), int64(len(dados)), minio.PutObjectOptions{
		ContentType:          mimeType,
		DisableContentSha256: true,
		SendContentMd5:       true,
	})
	if err != nil {
		return ResultadoUpload{}, fmt.Errorf("erro ao enviar midia para storage s3: %w", err)
	}
	return ResultadoUpload{
		Provider:   "s3",
		ObjectPath: objectPath,
		URL:        s.publicURL(objectPath),
	}, nil
}

// publicURL monta o endereco de leitura. O caminho normal e configurar
// MEDIA_STORAGE_PUBLIC_BASE_URL com o dominio publico do bucket: no R2 e o
// dominio custom, no Garage e o modo website. Sem isso, sobra o endereco direto
// do endpoint, que so funciona se o bucket for publico e o servico aceitar
// leitura por path.
func (s *s3Uploader) publicURL(objectPath string) string {
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + pathEscape(objectPath)
	}
	esquema := "https"
	if !s.seguro {
		esquema = "http"
	}
	return esquema + "://" + s.endpoint + "/" + pathEscape(s.bucket) + "/" + pathEscape(objectPath)
}
