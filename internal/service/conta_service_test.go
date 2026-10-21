package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dyalog-api-go/internal/email"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

type caixaEmails struct {
	mu     sync.Mutex
	links  map[string]string
	server *httptest.Server
}

func novaCaixa(t *testing.T) *caixaEmails {
	c := &caixaEmails{links: map[string]string{}}
	c.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var corpo struct {
			To   []string `json:"to"`
			HTML string   `json:"html"`
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		if r.Header.Get("Authorization") != "Bearer re_teste" {
			w.WriteHeader(401)
			return
		}
		i := strings.Index(corpo.HTML, `href="`)
		j := strings.Index(corpo.HTML[i+6:], `"`)
		c.mu.Lock()
		c.links[corpo.To[0]] = corpo.HTML[i+6 : i+6+j]
		c.mu.Unlock()
		_, _ = w.Write([]byte(`{"id":"e1"}`))
	}))
	t.Cleanup(c.server.Close)
	return c
}

func (c *caixaEmails) token(t *testing.T, para string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	link := c.links[para]
	if link == "" {
		t.Fatalf("nenhum e-mail para %s", para)
	}
	return link[strings.Index(link, "token=")+6:]
}

func TestContasConviteLoginPapeisELimite(t *testing.T) {
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	caixa := novaCaixa(t)
	contas := NovoContaService(st, email.NovoResend("re_teste", "Dyalog <oi@dyalog.com.br>", caixa.server.URL), "https://painel.exemplo.com")
	auth := NovoAuthService("master", st)
	auth.UsarContas(contas)
	master, _ := auth.Autenticar(ctx, "master")

	plano, err := contas.SalvarPlano(ctx, master, models.Plano{Nome: "Basico", PrecoCentavos: 9900, LimiteInstancias: 1, Ativo: true})
	if err != nil {
		t.Fatal(err)
	}
	conta, convite, err := contas.CriarConta(ctx, master, "Autoescola", plano.ID, "dono@exemplo.com")
	if err != nil || !convite.EmailEnviado || !strings.HasPrefix(convite.Link, "https://painel.exemplo.com/convite?token=") {
		t.Fatalf("conta: %v %+v", err, convite)
	}

	// Dono aceita o convite e ja sai logado.
	tokenDono := caixa.token(t, "dono@exemplo.com")
	if info, err := contas.InfoConvite(ctx, tokenDono); err != nil || info["conta_nome"] != "Autoescola" {
		t.Fatalf("info convite: %v %v", info, err)
	}
	if _, _, err := contas.AceitarConvite(ctx, tokenDono, "Dono", "curta"); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("senha curta aceita: %v", err)
	}
	sessaoDono, dono, err := contas.AceitarConvite(ctx, tokenDono, "Dono", "senha-forte-1")
	if err != nil || dono.Papel != models.PapelDono || dono.ContaID != conta.ID {
		t.Fatalf("aceitar: %v %+v", err, dono)
	}
	if _, _, err := contas.AceitarConvite(ctx, tokenDono, "Outro", "senha-forte-1"); err == nil {
		t.Fatal("convite usado duas vezes")
	}
	if a, err := auth.Autenticar(ctx, sessaoDono); err != nil || a.UsuarioID != dono.UsuarioID {
		t.Fatalf("sessao do dono: %v %+v", err, a)
	}

	// Dono convida atendente; atendente nao convida ninguem.
	if _, err := contas.Convidar(ctx, dono, conta.ID, "atende@exemplo.com", models.PapelAtendente); err != nil {
		t.Fatal(err)
	}
	_, atendente, err := contas.AceitarConvite(ctx, caixa.token(t, "atende@exemplo.com"), "Atendente", "senha-forte-2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contas.Convidar(ctx, atendente, conta.ID, "x@exemplo.com", models.PapelAtendente); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("atendente convidou: %v", err)
	}
	if _, err := contas.Convidar(ctx, dono, "outra-conta", "x@exemplo.com", models.PapelAtendente); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("dono convidou em outra conta: %v", err)
	}

	// Login: senha errada e certa.
	if _, _, err := contas.Login(ctx, "dono@exemplo.com", "errada"); !errors.Is(err, ErrNaoAutenticado) {
		t.Fatalf("senha errada: %v", err)
	}
	if _, a, err := contas.Login(ctx, "DONO@exemplo.com", "senha-forte-1"); err != nil || a.ContaNome != "Autoescola" {
		t.Fatalf("login: %v %+v", err, a)
	}

	// Instancias: so da propria conta; atendente usa mas nao configura.
	agora := time.Now()
	for _, inst := range []models.Instancia{{ID: "da-conta", ContaID: conta.ID}, {ID: "de-fora", ContaID: "outra"}} {
		inst.Nome, inst.Token, inst.Status, inst.CriadoEm, inst.AtualizadoEm = inst.ID, "tok-"+inst.ID, "desconectada", agora, agora
		if _, err := st.Criar(ctx, inst); err != nil {
			t.Fatal(err)
		}
	}
	if auth.GarantirInstancia(ctx, dono, "da-conta", true) != nil || auth.GarantirInstancia(ctx, atendente, "da-conta", false) != nil {
		t.Fatal("acesso a propria conta negado")
	}
	if auth.GarantirInstancia(ctx, atendente, "da-conta", true) == nil {
		t.Fatal("atendente configurou instancia")
	}
	if auth.GarantirInstancia(ctx, dono, "de-fora", false) == nil {
		t.Fatal("dono acessou instancia de outra conta")
	}
	if !auth.PodeCriarInstancias(dono) || auth.PodeCriarInstancias(atendente) {
		t.Fatal("permissao de criar instancia errada")
	}

	// Limite do plano: 1 instancia, ja usada.
	if err := contas.GarantirLimite(ctx, conta.ID); !errors.Is(err, ErrLimiteInstancias) {
		t.Fatalf("limite nao aplicado: %v", err)
	}
	if _, err := contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, LimiteInstancias: 3}); err != nil {
		t.Fatal(err)
	}
	if err := contas.GarantirLimite(ctx, conta.ID); err != nil {
		t.Fatalf("limite proprio da conta ignorado: %v", err)
	}

	// Tipos liberados: so a API oficial; todos marcados volta a "todos".
	if _, err := contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, LimiteInstancias: -1, TiposInstancia: " meta,meta "}); err != nil {
		t.Fatal(err)
	}
	if err := contas.GarantirTipo(ctx, conta.ID, ""); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("QR code liberado numa conta so da Meta: %v", err)
	}
	if err := contas.GarantirTipo(ctx, conta.ID, "meta"); err != nil {
		t.Fatalf("Meta recusada: %v", err)
	}
	if _, err := contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, LimiteInstancias: -1, TiposInstancia: "telegram"}); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("tipo invalido aceito: %v", err)
	}
	if c, err := contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, LimiteInstancias: -1, TiposInstancia: "instagram,whatsapp,meta"}); err != nil || c.TiposInstancia != "" {
		t.Fatalf("todos os tipos deveriam virar vazio: %q %v", c.TiposInstancia, err)
	}

	// Ultimo dono nao pode ser rebaixado; atendente desativado perde a sessao.
	if _, err := contas.AtualizarMembro(ctx, master, dono.UsuarioID, models.PapelAdmin, nil); !errors.Is(err, ErrEntradaInvalida) {
		t.Fatalf("rebaixou o ultimo dono: %v", err)
	}
	sessaoAtendente, _, _ := contas.Login(ctx, "atende@exemplo.com", "senha-forte-2")
	desativar := false
	if _, err := contas.AtualizarMembro(ctx, dono, atendente.UsuarioID, "", &desativar); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Autenticar(ctx, sessaoAtendente); err == nil {
		t.Fatal("sessao de usuario desativado continua valida")
	}

	// Conta bloqueada nao entra.
	if _, err := contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, Status: models.ContaBloqueada, LimiteInstancias: -1}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := contas.Login(ctx, "dono@exemplo.com", "senha-forte-1"); !errors.Is(err, ErrAcessoNegado) {
		t.Fatalf("conta bloqueada entrou: %v", err)
	}
	_, _ = contas.AtualizarConta(ctx, master, models.Conta{ID: conta.ID, PlanoID: plano.ID, Status: models.ContaAtiva, LimiteInstancias: -1})

	// Esqueci a senha: link por e-mail, uma vez so, derruba sessoes.
	contas.EsqueciSenha(ctx, "nao-existe@exemplo.com")
	contas.EsqueciSenha(ctx, "dono@exemplo.com")
	tokenRedefinir := caixa.token(t, "dono@exemplo.com")
	if !strings.HasPrefix(tokenRedefinir, "red_") {
		t.Fatalf("link de redefinicao errado: %s", tokenRedefinir)
	}
	if err := contas.RedefinirSenha(ctx, tokenRedefinir, "senha-nova-123"); err != nil {
		t.Fatal(err)
	}
	if err := contas.RedefinirSenha(ctx, tokenRedefinir, "outra-senha-123"); err == nil {
		t.Fatal("link de redefinicao usado duas vezes")
	}
	if _, err := auth.Autenticar(ctx, sessaoDono); err == nil {
		t.Fatal("sessao antiga continua valida depois de trocar a senha")
	}
	if _, _, err := contas.Login(ctx, "dono@exemplo.com", "senha-nova-123"); err != nil {
		t.Fatalf("login com a nova senha: %v", err)
	}
}
