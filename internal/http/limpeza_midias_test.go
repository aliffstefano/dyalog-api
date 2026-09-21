package http

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

type midiaLimpezaFake struct {
	mu         sync.Mutex
	pendentes  []models.MidiaArquivoLocal
	esquecidos []string
}

func (f *midiaLimpezaFake) BuscarMidiasLocaisComCopiaExterna(_ context.Context, _ time.Time, limite int) ([]models.MidiaArquivoLocal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.pendentes) == 0 {
		return nil, nil
	}
	if limite > len(f.pendentes) {
		limite = len(f.pendentes)
	}
	lote := f.pendentes[:limite]
	f.pendentes = f.pendentes[limite:]
	return lote, nil
}

func (f *midiaLimpezaFake) EsquecerCaminhoArquivoMidia(_ context.Context, midiaID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.esquecidos = append(f.esquecidos, midiaID)
	return nil
}

// A rotina roda uma vez por dia, entao o teste chama o mesmo trabalho por dentro
// em vez de esperar o relogio.
func TestLimpezaMidiasApagaArquivoEEsqueceCaminho(t *testing.T) {
	base := t.TempDir()
	diaAntigo := filepath.Join(base, "recebidas", "instancia-a", "20260101")
	if err := os.MkdirAll(diaAntigo, 0o755); err != nil {
		t.Fatalf("erro ao preparar diretorio: %v", err)
	}
	arquivo := filepath.Join(diaAntigo, "midia.jpg")
	if err := os.WriteFile(arquivo, []byte("conteudo-qualquer"), 0o644); err != nil {
		t.Fatalf("erro ao criar arquivo: %v", err)
	}

	fake := &midiaLimpezaFake{pendentes: []models.MidiaArquivoLocal{
		{ID: "midia-1", InstanciaID: "instancia-a", CaminhoArquivo: arquivo},
	}}

	executarLimpezaMidiasLocais(context.Background(), fake, 7)

	if _, err := os.Stat(arquivo); !os.IsNotExist(err) {
		t.Fatal("o arquivo local deveria ter sido apagado")
	}
	if len(fake.esquecidos) != 1 || fake.esquecidos[0] != "midia-1" {
		t.Fatalf("caminho deveria ter sido esquecido no banco, obtido %v", fake.esquecidos)
	}
	// A pasta do dia ficou vazia e deve sair junto.
	if _, err := os.Stat(diaAntigo); !os.IsNotExist(err) {
		t.Fatal("a pasta do dia, ja vazia, deveria ter sido removida")
	}
}

func TestLimpezaMidiasDesligadaPorPadrao(t *testing.T) {
	fake := &midiaLimpezaFake{pendentes: []models.MidiaArquivoLocal{
		{ID: "midia-1", CaminhoArquivo: "/nao/deve/ser/tocado"},
	}}
	// Zero e o padrao: ninguem perde arquivo por atualizar a API sem querer.
	iniciarLimpezaMidiasLocais(context.Background(), fake, 0)
	time.Sleep(50 * time.Millisecond)
	if len(fake.esquecidos) != 0 {
		t.Fatal("com retencao zero a limpeza nao pode apagar nada")
	}
}
