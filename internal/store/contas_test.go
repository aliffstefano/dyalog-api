package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestContasUsuariosConvitesSessoes(t *testing.T) {
	s, err := NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "contas.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	agora := time.Now()

	if err := s.SalvarPlano(ctx, models.Plano{ID: "basico", Nome: "Basico", PrecoCentavos: 9900, LimiteInstancias: 2, Ativo: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.CriarConta(ctx, models.Conta{ID: "c1", Nome: "Autoescola", PlanoID: "basico", Status: models.ContaAtiva, CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Criar(ctx, models.Instancia{ID: "i1", Nome: "I", ContaID: "c1", Token: "t", Status: "desconectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	if inst, _ := s.BuscarPorID(ctx, "i1"); inst.ContaID != "c1" {
		t.Fatalf("conta da instancia = %q", inst.ContaID)
	}

	u := models.Usuario{ID: "u1", ContaID: "c1", Nome: "Ana", Email: " Ana@Exemplo.com ", Papel: models.PapelDono, Ativo: true, SenhaHash: "h", CriadoEm: agora}
	if err := s.CriarUsuario(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := s.CriarUsuario(ctx, models.Usuario{ID: "u2", Email: "ana@exemplo.com", SenhaHash: "h", CriadoEm: agora}); !errors.Is(err, ErrEmailEmUso) {
		t.Fatalf("e-mail repetido: %v", err)
	}
	achado, err := s.BuscarUsuarioPorEmail(ctx, "ANA@exemplo.com")
	if err != nil || achado.ID != "u1" || !achado.Ativo || !achado.UltimoLoginEm.IsZero() {
		t.Fatalf("usuario = %+v, %v", achado, err)
	}
	_ = s.RegistrarLoginUsuario(ctx, "u1")
	if achado, _ = s.BuscarUsuarioPorID(ctx, "u1"); achado.UltimoLoginEm.IsZero() {
		t.Fatal("ultimo login nao gravado")
	}
	if n, _ := s.ContarDonosAtivos(ctx, "c1"); n != 1 {
		t.Fatalf("donos = %d", n)
	}

	convite := models.Convite{ID: "v1", ContaID: "c1", Email: "bia@exemplo.com", Papel: models.PapelAtendente, TokenHash: "hash1", ExpiraEm: agora.Add(time.Hour), CriadoEm: agora}
	if err := s.CriarConvite(ctx, convite); err != nil {
		t.Fatal(err)
	}
	if c, err := s.BuscarConvitePendente(ctx, "hash1"); err != nil || c.Email != "bia@exemplo.com" {
		t.Fatalf("convite = %+v, %v", c, err)
	}
	resumo, _ := s.ListarContasResumo(ctx)
	if len(resumo) != 1 || resumo[0].Instancias != 1 || resumo[0].Usuarios != 1 || resumo[0].ConvitesPendentes != 1 || resumo[0].LimiteEfetivo != 2 || resumo[0].PlanoNome != "Basico" {
		t.Fatalf("resumo = %+v", resumo)
	}
	if ok, _ := s.MarcarConviteAceito(ctx, "v1"); !ok {
		t.Fatal("convite nao aceito")
	}
	if ok, _ := s.MarcarConviteAceito(ctx, "v1"); ok {
		t.Fatal("convite usado duas vezes")
	}
	if _, err := s.BuscarConvitePendente(ctx, "hash1"); !errors.Is(err, ErrConviteInvalido) {
		t.Fatalf("convite usado ainda pendente: %v", err)
	}

	if err := s.CriarSessao(ctx, "sess", "u1", agora.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if id, err := s.BuscarSessao(ctx, "sess"); err != nil || id != "u1" {
		t.Fatalf("sessao = %q %v", id, err)
	}
	_ = s.CriarSessao(ctx, "velha", "u1", agora.Add(-time.Minute))
	if _, err := s.BuscarSessao(ctx, "velha"); !errors.Is(err, ErrSessaoInvalida) {
		t.Fatalf("sessao expirada aceita: %v", err)
	}

	_ = s.CriarRedefinicaoSenha(ctx, "red", "u1", agora.Add(time.Hour))
	if id, err := s.UsarRedefinicaoSenha(ctx, "red"); err != nil || id != "u1" {
		t.Fatalf("redefinicao = %q %v", id, err)
	}
	if _, err := s.UsarRedefinicaoSenha(ctx, "red"); !errors.Is(err, ErrRedefinicaoInvalida) {
		t.Fatalf("redefinicao reutilizada: %v", err)
	}

	if err := s.ExcluirUsuario(ctx, "u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.BuscarSessao(ctx, "sess"); !errors.Is(err, ErrSessaoInvalida) {
		t.Fatal("sessao do usuario excluido continua valida")
	}
}
