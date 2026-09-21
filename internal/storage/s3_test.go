package storage

import "testing"

func TestSepararEndpointS3(t *testing.T) {
	casos := []struct {
		entrada    string
		host       string
		seguro     bool
		esperaErro bool
	}{
		// Endpoint tipico do R2 e do Garage, com e sem esquema.
		{"https://abc123.r2.cloudflarestorage.com", "abc123.r2.cloudflarestorage.com", true, false},
		{"abc123.r2.cloudflarestorage.com", "abc123.r2.cloudflarestorage.com", true, false},
		{"http://garage.interno:3900", "garage.interno:3900", false, false},
		{"https://garage.interno:3900/", "garage.interno:3900", true, false},
		// Sem esquema assume https: endpoint de producao sem TLS e erro provavel.
		{"minio.local:9000", "minio.local:9000", true, false},
		// Esquemas que nao fazem sentido para S3 devem falhar cedo, na subida.
		{"ftp://algum.host", "", false, true},
		{"https://", "", false, true},
	}
	for _, caso := range casos {
		host, seguro, err := separarEndpointS3(caso.entrada)
		if caso.esperaErro {
			if err == nil {
				t.Fatalf("separarEndpointS3(%q) deveria falhar", caso.entrada)
			}
			continue
		}
		if err != nil {
			t.Fatalf("separarEndpointS3(%q) erro inesperado: %v", caso.entrada, err)
		}
		if host != caso.host || seguro != caso.seguro {
			t.Fatalf("separarEndpointS3(%q) = (%q, %v), esperado (%q, %v)", caso.entrada, host, seguro, caso.host, caso.seguro)
		}
	}
}

func TestNovoMidiaUploaderDrivers(t *testing.T) {
	// local e vazio nao usam storage externo: uploader nulo e o esperado.
	for _, driver := range []string{"", "local", "LOCAL", " local "} {
		up, err := NovoMidiaUploader(Config{Driver: driver})
		if err != nil || up != nil {
			t.Fatalf("driver %q = (%v, %v), esperado (nil, nil)", driver, up, err)
		}
	}
	if _, err := NovoMidiaUploader(Config{Driver: "inexistente"}); err == nil {
		t.Fatal("driver desconhecido deveria falhar")
	}
	// s3 sem credencial precisa falhar na subida, nao no primeiro upload.
	if _, err := NovoMidiaUploader(Config{Driver: "s3", S3Endpoint: "https://exemplo"}); err == nil {
		t.Fatal("driver s3 sem credenciais deveria falhar")
	}
	up, err := NovoMidiaUploader(Config{
		Driver:      "s3",
		S3Endpoint:  "https://abc123.r2.cloudflarestorage.com",
		S3AccessKey: "chave",
		S3SecretKey: "segredo",
		S3Bucket:    "midias",
		S3Region:    "auto",
	})
	if err != nil || up == nil {
		t.Fatalf("driver s3 completo = (%v, %v), esperado uploader valido", up, err)
	}
}

func TestS3PublicURL(t *testing.T) {
	// Com dominio publico configurado, a URL sai por ele.
	comDominio := &s3Uploader{bucket: "midias", endpoint: "abc.r2.cloudflarestorage.com", publicBaseURL: "https://midia.exemplo.com", seguro: true}
	if obtido := comDominio.publicURL("instancias/a b/foto.jpg"); obtido != "https://midia.exemplo.com/instancias/a%20b/foto.jpg" {
		t.Fatalf("publicURL com dominio = %q", obtido)
	}
	// Sem dominio, cai para o endpoint direto, incluindo o bucket no caminho.
	semDominio := &s3Uploader{bucket: "midias", endpoint: "garage.interno:3900", seguro: false}
	if obtido := semDominio.publicURL("foto.jpg"); obtido != "http://garage.interno:3900/midias/foto.jpg" {
		t.Fatalf("publicURL sem dominio = %q", obtido)
	}
}
