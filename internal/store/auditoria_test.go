package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestAuditoriaRegistraFiltraEPagina(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "aud.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := s.Criar(ctx, models.Instancia{ID: "i", Nome: "Loja", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	for n, r := range []models.RegistroAuditoria{
		{InstanciaID: "i", ContaID: "c1", Tipo: models.AuditoriaConexao, Acao: "instancia.conectada", Severidade: models.SeveridadeSucesso, AtorTipo: "sistema", AtorNome: "Sistema"},
		{InstanciaID: "i", ContaID: "c1", Tipo: models.AuditoriaConexao, Acao: "instancia.desconectada", Severidade: models.SeveridadeAviso, AtorTipo: "usuario", AtorNome: "Ana Souza"},
		{InstanciaID: "i", ContaID: "c1", Tipo: models.AuditoriaWebhook, Acao: "webhook.criado", AtorTipo: "usuario", AtorNome: "Ana Souza"},
		{ContaID: "c2", Tipo: models.AuditoriaEquipe, Acao: "equipe.convite", AtorTipo: "usuario", AtorNome: "Bruno"},
	} {
		r.CriadoEm = agora.Add(time.Duration(n-10) * time.Minute)
		if err := s.RegistrarAuditoria(ctx, r); err != nil {
			t.Fatal(err)
		}
	}

	pag, err := s.ListarAuditoria(ctx, models.FiltroAuditoria{InstanciaID: "i"})
	if err != nil || len(pag.Registros) != 3 || pag.Registros[0].Acao != "webhook.criado" || pag.Registros[0].InstanciaNome != "Loja" || pag.Registros[0].Severidade != "info" {
		t.Fatalf("pag = %+v err = %v", pag, err)
	}
	if pag, _ := s.ListarAuditoria(ctx, models.FiltroAuditoria{InstanciaID: "i", Severidade: "aviso", Ator: "ana"}); len(pag.Registros) != 1 {
		t.Fatalf("filtro severidade+ator = %+v", pag)
	}
	if pag, _ := s.ListarAuditoria(ctx, models.FiltroAuditoria{SoContas: true, ContasIDs: []string{"c2"}}); len(pag.Registros) != 1 || pag.Registros[0].AtorNome != "Bruno" {
		t.Fatalf("filtro conta = %+v", pag)
	}
	if pag, _ := s.ListarAuditoria(ctx, models.FiltroAuditoria{SoContas: true}); len(pag.Registros) != 0 {
		t.Fatalf("sem contas deveria vir vazio: %+v", pag)
	}
	if pag, _ := s.ListarAuditoria(ctx, models.FiltroAuditoria{Limite: 2}); len(pag.Registros) != 2 || !pag.TemMais {
		t.Fatalf("paginacao = %+v", pag)
	}
	if pag, _ := s.ListarAuditoria(ctx, models.FiltroAuditoria{De: agora.Add(-9*time.Minute - 30*time.Second)}); len(pag.Registros) != 3 {
		t.Fatalf("filtro data = %d", len(pag.Registros))
	}
	if n, err := s.LimparAuditoriaAntiga(ctx, agora.Add(-8*time.Minute-30*time.Second)); err != nil || n != 2 {
		t.Fatalf("limpeza = %d %v", n, err)
	}
}
