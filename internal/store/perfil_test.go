package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestPerfilInstanciaCompartilhadoEntreReplicas(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "perfil.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now()
	if _, err := s.Criar(ctx, models.Instancia{ID: "a", Nome: "A", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}

	perfil := models.PerfilInstancia{Numero: "5567999440667", Nome: "Aliff", Business: true, FotoURL: "https://pps.whatsapp.net/x"}
	if err := s.SalvarPerfilInstancia(ctx, "a", perfil); err != nil {
		t.Fatal(err)
	}
	perfil.NomeEmpresa = "Agencia"
	if err := s.SalvarPerfilInstancia(ctx, "a", perfil); err != nil {
		t.Fatal(err)
	}
	perfis, err := s.ListarPerfisInstancias(ctx)
	if err != nil || perfis["a"] != perfil {
		t.Fatalf("perfis = %v, err = %v", perfis, err)
	}

	if err := s.ExcluirPerfilInstancia(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if perfis, _ := s.ListarPerfisInstancias(ctx); len(perfis) != 0 {
		t.Fatalf("perfil continuou salvo: %v", perfis)
	}
}
