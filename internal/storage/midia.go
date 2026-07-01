package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type ResultadoUpload struct {
	Provider   string
	ObjectPath string
	URL        string
}

type MidiaUploader interface {
	Enviar(ctx context.Context, objectPath, mimeType string, dados []byte) (ResultadoUpload, error)
}

func NovoMidiaUploader(driver, supabaseURL, supabaseKey, bucket, publicBaseURL string) (MidiaUploader, error) {
	driver = strings.ToLower(strings.TrimSpace(driver))
	if driver == "" || driver == "local" {
		return nil, nil
	}
	if driver != "supabase" {
		return nil, fmt.Errorf("MEDIA_STORAGE_DRIVER invalido: use local ou supabase")
	}
	supabaseURL = strings.TrimRight(strings.TrimSpace(supabaseURL), "/")
	supabaseKey = strings.TrimSpace(supabaseKey)
	bucket = strings.TrimSpace(bucket)
	if supabaseURL == "" || supabaseKey == "" || bucket == "" {
		return nil, fmt.Errorf("MEDIA_STORAGE_SUPABASE_URL, MEDIA_STORAGE_SUPABASE_KEY e MEDIA_STORAGE_SUPABASE_BUCKET sao obrigatorios para storage supabase")
	}
	return &supabaseUploader{
		baseURL:       strings.TrimRight(supabaseURL, "/"),
		key:           supabaseKey,
		bucket:        bucket,
		publicBaseURL: strings.TrimRight(strings.TrimSpace(publicBaseURL), "/"),
		client:        &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type supabaseUploader struct {
	baseURL       string
	key           string
	bucket        string
	publicBaseURL string
	client        *http.Client
}

func (s *supabaseUploader) Enviar(ctx context.Context, objectPath, mimeType string, dados []byte) (ResultadoUpload, error) {
	objectPath = limparObjectPath(objectPath)
	if objectPath == "" {
		return ResultadoUpload{}, fmt.Errorf("caminho do objeto da midia vazio")
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	endpoint := s.baseURL + "/storage/v1/object/" + pathEscape(s.bucket) + "/" + pathEscape(objectPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(dados))
	if err != nil {
		return ResultadoUpload{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.key)
	req.Header.Set("apikey", s.key)
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-upsert", "true")

	resp, err := s.client.Do(req)
	if err != nil {
		return ResultadoUpload{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ResultadoUpload{}, fmt.Errorf("supabase storage retornou status %d: %s", resp.StatusCode, strings.TrimSpace(string(corpo)))
	}
	return ResultadoUpload{
		Provider:   "supabase",
		ObjectPath: objectPath,
		URL:        s.publicURL(objectPath),
	}, nil
}

func (s *supabaseUploader) publicURL(objectPath string) string {
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + pathEscape(objectPath)
	}
	return s.baseURL + "/storage/v1/object/public/" + pathEscape(s.bucket) + "/" + pathEscape(objectPath)
}

func limparObjectPath(valor string) string {
	valor = strings.TrimSpace(strings.ReplaceAll(valor, "\\", "/"))
	valor = path.Clean("/" + valor)
	return strings.TrimPrefix(valor, "/")
}

func pathEscape(valor string) string {
	partes := strings.Split(valor, "/")
	for i, parte := range partes {
		partes[i] = url.PathEscape(parte)
	}
	return strings.Join(partes, "/")
}
