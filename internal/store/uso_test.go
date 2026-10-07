package store

import (
	"context"
	"os"
	"testing"
	"time"

	"dyalog-api-go/internal/models"

	"github.com/google/uuid"
)

// Roda no SQLite e, com DYALOG_TEST_POSTGRES_DSN definido, tambem no Postgres.
func TestResumoUso(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) { testarResumoUso(t, novoStoreTeste(t)) })
	dsn := os.Getenv("DYALOG_TEST_POSTGRES_DSN")
	if dsn == "" {
		return
	}
	t.Run("postgres", func(t *testing.T) {
		s, err := NovoSQLStore("postgres", dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.db.Close() })
		testarResumoUso(t, s)
	})
}

func testarResumoUso(t *testing.T, s *SQLStore) {
	ctx := context.Background()
	agora := time.Now()
	instancia := uuid.NewString()
	if _, err := s.Criar(ctx, models.Instancia{ID: instancia, Nome: "uso", Token: uuid.NewString(), Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	// Cliente antigo: ja mandou mensagem para a instancia.
	if _, err := s.RegistrarMensagemProcessada(ctx, models.MensagemProcessada{InstanciaID: instancia, ChatJID: "antigo@s.whatsapp.net", MensagemID: "m1"}); err != nil {
		t.Fatal(err)
	}
	enviar := func(chat, resultado string, em time.Time) {
		t.Helper()
		if err := s.RegistrarEnvio(ctx, models.EnvioRegistro{InstanciaID: instancia, ChatJID: chat, Tipo: "texto", Resultado: resultado, CriadoEm: em}); err != nil {
			t.Fatal(err)
		}
	}
	enviar("antigo@s.whatsapp.net", models.EnvioResultadoEnviada, agora)
	enviar("novo@s.whatsapp.net", models.EnvioResultadoEnviada, agora)
	enviar("novo@s.whatsapp.net", models.EnvioResultadoEnviada, agora) // segundo envio: ja nao e novo
	enviar("123@g.us", models.EnvioResultadoEnviada, agora)            // grupo nunca e contato novo
	for i := 0; i < 6; i++ {                                           // rajada
		enviar("rajada@s.whatsapp.net", models.EnvioResultadoEnviada, agora)
	}
	enviar("", models.EnvioResultadoLimitada, agora)
	enviar("ontem@s.whatsapp.net", models.EnvioResultadoEnviada, agora.AddDate(0, 0, -1))

	resumo, err := s.ResumoUso(ctx, instancia, agora, 7)
	if err != nil {
		t.Fatal(err)
	}
	if resumo.Hoje.Envios != 10 || resumo.Hoje.ContatosNovos != 2 || resumo.Hoje.Limitados != 1 {
		t.Fatalf("hoje = %+v", resumo.Hoje)
	}
	if resumo.UltimoMinuto != 10 || resumo.UltimaHora != 10 || resumo.PicoMinutoHoje != 10 {
		t.Fatalf("recentes: minuto=%d hora=%d pico=%d", resumo.UltimoMinuto, resumo.UltimaHora, resumo.PicoMinutoHoje)
	}
	if len(resumo.Rajadas) != 1 || resumo.Rajadas[0].ChatJID != "rajada@s.whatsapp.net" || resumo.Rajadas[0].Envios != 6 {
		t.Fatalf("rajadas = %+v", resumo.Rajadas)
	}
	if len(resumo.TopDestinatarios) == 0 || resumo.TopDestinatarios[0].ChatJID != "rajada@s.whatsapp.net" {
		t.Fatalf("top = %+v", resumo.TopDestinatarios)
	}
	if len(resumo.Dias) != 7 || resumo.Dias[5].Envios != 1 || resumo.Dias[5].ContatosNovos != 1 {
		t.Fatalf("dias = %+v", resumo.Dias)
	}

	porInstancia, err := s.UsoHojePorInstancia(ctx, agora)
	if err != nil {
		t.Fatal(err)
	}
	if len(porInstancia) != 1 || porInstancia[0].InstanciaID != instancia || porInstancia[0].Envios != 10 ||
		porInstancia[0].ContatosNovos != 2 || porInstancia[0].Limitados != 1 || porInstancia[0].Rajadas != 1 {
		t.Fatalf("uso por instancia = %+v", porInstancia)
	}

	apagadas, err := s.LimparEnviosAntigos(ctx, agora.Add(-time.Hour))
	if err != nil || apagadas != 1 {
		t.Fatalf("limpeza: apagadas=%d err=%v", apagadas, err)
	}
}
