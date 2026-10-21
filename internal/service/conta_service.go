package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"dyalog-api-go/internal/email"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	duracaoSessao      = 15 * 24 * time.Hour
	validadeConvite    = 7 * 24 * time.Hour
	validadeRedefinir  = time.Hour
	prefixoSessao      = "ses_"
	tamanhoMinimoSenha = 8
)

// ErrLimiteInstancias indica que a conta ja usa todas as instancias do plano.
var ErrLimiteInstancias = errors.New("limite de instancias do plano atingido")

// ContaService cuida das contas (empresas), da equipe de cada uma, do login
// com e-mail e senha e dos planos.
type ContaService struct {
	store   store.ContaStore
	email   email.Enviador
	baseURL string
}

func NovoContaService(contaStore store.ContaStore, enviador email.Enviador, baseURL string) *ContaService {
	return &ContaService{store: contaStore, email: enviador, baseURL: strings.TrimRight(baseURL, "/")}
}

// tokenAleatorio devolve o token (vai para o usuario) e o hash (vai para o
// banco): quem ler o banco nao consegue usar sessoes nem convites.
func tokenAleatorio(prefixo string) (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := prefixo + base64.RawURLEncoding.EncodeToString(bytes)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	soma := sha256.Sum256([]byte(token))
	return hex.EncodeToString(soma[:])
}

func validarSenha(senha string) error {
	if len([]rune(senha)) < tamanhoMinimoSenha {
		return fmt.Errorf("%w: a senha precisa ter pelo menos %d caracteres", ErrEntradaInvalida, tamanhoMinimoSenha)
	}
	return nil
}

func gerarHashSenha(senha string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	return string(hash), err
}

func emailValido(valor string) bool {
	valor = strings.TrimSpace(valor)
	arroba := strings.Index(valor, "@")
	return arroba > 0 && strings.Contains(valor[arroba:], ".") && !strings.ContainsAny(valor, " <>")
}

// ── Login e sessoes ──────────────────────────────────────────────────────────

// Login confere e-mail e senha e abre uma sessao. A mesma mensagem de erro
// para e-mail e senha errados, para nao revelar quem tem conta.
func (s *ContaService) Login(ctx context.Context, emailLogin, senha string) (string, models.AcessoDashboard, error) {
	usuario, err := s.store.BuscarUsuarioPorEmail(ctx, emailLogin)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(senha)) != nil {
		return "", models.AcessoDashboard{}, ErrNaoAutenticado
	}
	acesso, err := s.acessoDoUsuario(ctx, usuario)
	if err != nil {
		return "", models.AcessoDashboard{}, err
	}
	token, err := s.abrirSessao(ctx, usuario.ID)
	if err != nil {
		return "", models.AcessoDashboard{}, err
	}
	return token, acesso, nil
}

func (s *ContaService) abrirSessao(ctx context.Context, usuarioID string) (string, error) {
	token, hash, err := tokenAleatorio(prefixoSessao)
	if err != nil {
		return "", err
	}
	if err := s.store.CriarSessao(ctx, hash, usuarioID, time.Now().Add(duracaoSessao)); err != nil {
		return "", err
	}
	_ = s.store.RegistrarLoginUsuario(ctx, usuarioID)
	return token, nil
}

// EhTokenDeSessao diz se o token e de sessao de usuario (e nao master ou de
// instancia).
func EhTokenDeSessao(token string) bool { return strings.HasPrefix(token, prefixoSessao) }

// AcessoPorSessao devolve quem esta logado na sessao.
func (s *ContaService) AcessoPorSessao(ctx context.Context, token string) (models.AcessoDashboard, error) {
	usuarioID, err := s.store.BuscarSessao(ctx, hashToken(token))
	if err != nil {
		return models.AcessoDashboard{}, ErrNaoAutenticado
	}
	usuario, err := s.store.BuscarUsuarioPorID(ctx, usuarioID)
	if err != nil {
		return models.AcessoDashboard{}, ErrNaoAutenticado
	}
	return s.acessoDoUsuario(ctx, usuario)
}

func (s *ContaService) acessoDoUsuario(ctx context.Context, usuario models.Usuario) (models.AcessoDashboard, error) {
	if !usuario.Ativo {
		return models.AcessoDashboard{}, fmt.Errorf("%w: usuario desativado", ErrAcessoNegado)
	}
	acesso := models.AcessoDashboard{
		Tipo: "usuario", UsuarioID: usuario.ID, Nome: usuario.Nome, Email: usuario.Email,
		ContaID: usuario.ContaID, Papel: usuario.Papel, Superadmin: usuario.Superadmin,
	}
	if usuario.Superadmin {
		return acesso, nil
	}
	conta, err := s.store.BuscarConta(ctx, usuario.ContaID)
	if err != nil {
		return models.AcessoDashboard{}, fmt.Errorf("%w: conta nao encontrada", ErrAcessoNegado)
	}
	if conta.Status != models.ContaAtiva {
		return models.AcessoDashboard{}, fmt.Errorf("%w: conta bloqueada, fale com o suporte", ErrAcessoNegado)
	}
	acesso.ContaNome, acesso.TiposInstancia = conta.Nome, conta.TiposInstancia
	return acesso, nil
}

func (s *ContaService) Logout(ctx context.Context, token string) {
	if EhTokenDeSessao(token) {
		_ = s.store.ExcluirSessao(ctx, hashToken(token))
	}
}

// ── Convites ─────────────────────────────────────────────────────────────────

// ResultadoConvite volta para quem convidou: o link (para copiar se o e-mail
// nao chegar) e se o e-mail foi enviado.
type ResultadoConvite struct {
	Convite      models.Convite `json:"convite"`
	Link         string         `json:"link"`
	EmailEnviado bool           `json:"email_enviado"`
	EmailErro    string         `json:"email_erro,omitempty"`
}

// podeConvidar: superadmin convida qualquer papel; dono tambem; admin convida
// admin e atendente da propria conta.
func podeConvidar(quem models.AcessoDashboard, contaID, papel string) bool {
	if quem.TudoLiberado() {
		return true
	}
	if quem.Tipo != "usuario" || quem.ContaID != contaID {
		return false
	}
	switch quem.Papel {
	case models.PapelDono:
		return true
	case models.PapelAdmin:
		return papel != models.PapelDono
	default:
		return false
	}
}

// Convidar convida alguem para a conta com um papel.
func (s *ContaService) Convidar(ctx context.Context, quem models.AcessoDashboard, contaID, emailConvidado, papel string) (ResultadoConvite, error) {
	emailConvidado = strings.ToLower(strings.TrimSpace(emailConvidado))
	if !emailValido(emailConvidado) {
		return ResultadoConvite{}, fmt.Errorf("%w: e-mail invalido", ErrEntradaInvalida)
	}
	if !models.PapelValido(papel) {
		return ResultadoConvite{}, fmt.Errorf("%w: papel deve ser dono, admin ou atendente", ErrEntradaInvalida)
	}
	if !podeConvidar(quem, contaID, papel) {
		return ResultadoConvite{}, ErrAcessoNegado
	}
	conta, err := s.store.BuscarConta(ctx, contaID)
	if err != nil {
		return ResultadoConvite{}, fmt.Errorf("%w: conta nao encontrada", ErrEntradaInvalida)
	}
	return s.criarConvite(ctx, quem, models.Convite{ContaID: contaID, Email: emailConvidado, Papel: papel},
		"Voce foi convidado para o Dyalog Connect",
		[]string{fmt.Sprintf("Voce foi convidado para a equipe de %s no Dyalog Connect como %s.", conta.Nome, papel), "Crie sua senha pelo botao abaixo. O convite vale por 7 dias."})
}

// ConvidarSuperadmin convida alguem com acesso total (so superadmin pode).
func (s *ContaService) ConvidarSuperadmin(ctx context.Context, quem models.AcessoDashboard, emailConvidado string) (ResultadoConvite, error) {
	if !quem.TudoLiberado() {
		return ResultadoConvite{}, ErrAcessoNegado
	}
	emailConvidado = strings.ToLower(strings.TrimSpace(emailConvidado))
	if !emailValido(emailConvidado) {
		return ResultadoConvite{}, fmt.Errorf("%w: e-mail invalido", ErrEntradaInvalida)
	}
	return s.criarConvite(ctx, quem, models.Convite{Email: emailConvidado, Superadmin: true},
		"Acesso de administrador ao Dyalog Connect",
		[]string{"Voce recebeu acesso de administrador do Dyalog Connect, com todas as contas e instancias.", "Crie sua senha pelo botao abaixo. O convite vale por 7 dias."})
}

func (s *ContaService) criarConvite(ctx context.Context, quem models.AcessoDashboard, convite models.Convite, assunto string, texto []string) (ResultadoConvite, error) {
	if _, err := s.store.BuscarUsuarioPorEmail(ctx, convite.Email); err == nil {
		return ResultadoConvite{}, fmt.Errorf("%w: ja existe um usuario com esse e-mail", ErrEntradaInvalida)
	}
	token, hash, err := tokenAleatorio("inv_")
	if err != nil {
		return ResultadoConvite{}, err
	}
	agora := time.Now()
	convite.ID, convite.TokenHash, convite.CriadoPor = uuid.NewString(), hash, cmpOr(quem.UsuarioID, quem.Tipo)
	convite.CriadoEm, convite.ExpiraEm = agora, agora.Add(validadeConvite)
	if err := s.store.CriarConvite(ctx, convite); err != nil {
		return ResultadoConvite{}, err
	}
	resultado := ResultadoConvite{Convite: convite, Link: s.baseURL + "/convite?token=" + token}
	if err := s.email.Enviar(ctx, convite.Email, assunto, email.Modelo(assunto, texto, "Criar minha senha", resultado.Link)); err != nil {
		resultado.EmailErro = err.Error()
	} else {
		resultado.EmailEnviado = true
	}
	return resultado, nil
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// InfoConvite e o que a pagina do convite mostra antes de criar a senha.
func (s *ContaService) InfoConvite(ctx context.Context, token string) (map[string]any, error) {
	convite, err := s.store.BuscarConvitePendente(ctx, hashToken(strings.TrimSpace(token)))
	if err != nil {
		return nil, fmt.Errorf("%w: convite invalido, expirado ou ja usado", ErrEntradaInvalida)
	}
	info := map[string]any{"email": convite.Email, "papel": convite.Papel, "superadmin": convite.Superadmin, "expira_em": convite.ExpiraEm}
	if convite.ContaID != "" {
		if conta, err := s.store.BuscarConta(ctx, convite.ContaID); err == nil {
			info["conta_nome"] = conta.Nome
		}
	}
	return info, nil
}

// AceitarConvite cria o usuario com a senha escolhida e ja abre a sessao.
func (s *ContaService) AceitarConvite(ctx context.Context, token, nome, senha string) (string, models.AcessoDashboard, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "", models.AcessoDashboard{}, fmt.Errorf("%w: informe seu nome", ErrEntradaInvalida)
	}
	if err := validarSenha(senha); err != nil {
		return "", models.AcessoDashboard{}, err
	}
	convite, err := s.store.BuscarConvitePendente(ctx, hashToken(strings.TrimSpace(token)))
	if err != nil {
		return "", models.AcessoDashboard{}, fmt.Errorf("%w: convite invalido, expirado ou ja usado", ErrEntradaInvalida)
	}
	hash, err := gerarHashSenha(senha)
	if err != nil {
		return "", models.AcessoDashboard{}, err
	}
	if usado, err := s.store.MarcarConviteAceito(ctx, convite.ID); err != nil || !usado {
		return "", models.AcessoDashboard{}, fmt.Errorf("%w: convite ja usado", ErrEntradaInvalida)
	}
	usuario := models.Usuario{
		ID: uuid.NewString(), ContaID: convite.ContaID, Nome: nome, Email: convite.Email, Papel: convite.Papel,
		Superadmin: convite.Superadmin, Ativo: true, SenhaHash: hash, CriadoEm: time.Now(),
	}
	if err := s.store.CriarUsuario(ctx, usuario); err != nil {
		if errors.Is(err, store.ErrEmailEmUso) {
			return "", models.AcessoDashboard{}, fmt.Errorf("%w: ja existe um usuario com esse e-mail", ErrEntradaInvalida)
		}
		return "", models.AcessoDashboard{}, err
	}
	acesso, err := s.acessoDoUsuario(ctx, usuario)
	if err != nil {
		return "", models.AcessoDashboard{}, err
	}
	tokenSessao, err := s.abrirSessao(ctx, usuario.ID)
	return tokenSessao, acesso, err
}

// ── Senha ────────────────────────────────────────────────────────────────────

// EsqueciSenha manda o link de redefinicao. Responde igual exista ou nao o
// e-mail, para nao revelar quem tem conta.
func (s *ContaService) EsqueciSenha(ctx context.Context, emailUsuario string) {
	usuario, err := s.store.BuscarUsuarioPorEmail(ctx, emailUsuario)
	if err != nil || !usuario.Ativo {
		return
	}
	token, hash, err := tokenAleatorio("red_")
	if err != nil || s.store.CriarRedefinicaoSenha(ctx, hash, usuario.ID, time.Now().Add(validadeRedefinir)) != nil {
		return
	}
	link := s.baseURL + "/senha/redefinir?token=" + token
	assunto := "Redefinir sua senha do Dyalog Connect"
	if err := s.email.Enviar(ctx, usuario.Email, assunto, email.Modelo(assunto,
		[]string{"Recebemos um pedido para redefinir a sua senha.", "O link vale por 1 hora. Se nao foi voce, ignore este e-mail."},
		"Redefinir senha", link)); err != nil {
		fmt.Printf("contas: e-mail de redefinicao para %s nao enviado: %v\n", usuario.Email, err)
	}
}

// RedefinirSenha troca a senha pelo link e derruba as sessoes abertas.
func (s *ContaService) RedefinirSenha(ctx context.Context, token, senha string) error {
	if err := validarSenha(senha); err != nil {
		return err
	}
	usuarioID, err := s.store.UsarRedefinicaoSenha(ctx, hashToken(strings.TrimSpace(token)))
	if err != nil {
		return fmt.Errorf("%w: link invalido, expirado ou ja usado; peca outro", ErrEntradaInvalida)
	}
	return s.gravarSenha(ctx, usuarioID, senha)
}

// TrocarSenha e o usuario logado trocando a propria senha.
func (s *ContaService) TrocarSenha(ctx context.Context, quem models.AcessoDashboard, atual, nova string) error {
	if quem.Tipo != "usuario" {
		return fmt.Errorf("%w: so usuarios com e-mail tem senha", ErrEntradaInvalida)
	}
	usuario, err := s.store.BuscarUsuarioPorID(ctx, quem.UsuarioID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(atual)) != nil {
		return fmt.Errorf("%w: senha atual incorreta", ErrEntradaInvalida)
	}
	if err := validarSenha(nova); err != nil {
		return err
	}
	return s.gravarSenha(ctx, usuario.ID, nova)
}

func (s *ContaService) gravarSenha(ctx context.Context, usuarioID, senha string) error {
	hash, err := gerarHashSenha(senha)
	if err != nil {
		return err
	}
	if err := s.store.AtualizarSenhaUsuario(ctx, usuarioID, hash); err != nil {
		return err
	}
	return s.store.ExcluirSessoesUsuario(ctx, usuarioID)
}

// ── Equipe ───────────────────────────────────────────────────────────────────

// contaDaEquipe resolve de qual conta se fala: a do usuario, ou a escolhida
// pelo superadmin.
func contaDaEquipe(quem models.AcessoDashboard, contaID string) (string, error) {
	if quem.TudoLiberado() {
		if contaID == "" {
			return "", fmt.Errorf("%w: informe a conta", ErrEntradaInvalida)
		}
		return contaID, nil
	}
	if quem.Tipo != "usuario" || quem.ContaID == "" {
		return "", ErrAcessoNegado
	}
	return quem.ContaID, nil
}

// Equipe lista os usuarios e os convites pendentes da conta.
func (s *ContaService) Equipe(ctx context.Context, quem models.AcessoDashboard, contaID string) (map[string]any, error) {
	contaID, err := contaDaEquipe(quem, contaID)
	if err != nil {
		return nil, err
	}
	usuarios, err := s.store.ListarUsuarios(ctx, contaID, false)
	if err != nil {
		return nil, err
	}
	convites, err := s.store.ListarConvitesPendentes(ctx, contaID, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"conta_id": contaID, "usuarios": vazioSeNil(usuarios), "convites": vazioSeNil(convites)}, nil
}

func vazioSeNil[T any](lista []T) []T {
	if lista == nil {
		return []T{}
	}
	return lista
}

// AtualizarMembro troca papel ou ativa/desativa alguem da equipe.
func (s *ContaService) AtualizarMembro(ctx context.Context, quem models.AcessoDashboard, usuarioID, papel string, ativo *bool) (models.Usuario, error) {
	alvo, err := s.membroGerenciavel(ctx, quem, usuarioID)
	if err != nil {
		return models.Usuario{}, err
	}
	novo := alvo
	if papel = strings.TrimSpace(papel); papel != "" {
		if !models.PapelValido(papel) {
			return models.Usuario{}, fmt.Errorf("%w: papel deve ser dono, admin ou atendente", ErrEntradaInvalida)
		}
		if papel == models.PapelDono && !podeConvidar(quem, alvo.ContaID, models.PapelDono) {
			return models.Usuario{}, ErrAcessoNegado
		}
		novo.Papel = papel
	}
	if ativo != nil {
		novo.Ativo = *ativo
	}
	if alvo.Papel == models.PapelDono && alvo.Ativo && (novo.Papel != models.PapelDono || !novo.Ativo) {
		if err := s.garantirOutroDono(ctx, alvo.ContaID); err != nil {
			return models.Usuario{}, err
		}
	}
	if err := s.store.AtualizarUsuario(ctx, novo); err != nil {
		return models.Usuario{}, err
	}
	if !novo.Ativo {
		_ = s.store.ExcluirSessoesUsuario(ctx, novo.ID)
	}
	return novo, nil
}

// RemoverMembro tira alguem da equipe.
func (s *ContaService) RemoverMembro(ctx context.Context, quem models.AcessoDashboard, usuarioID string) error {
	alvo, err := s.membroGerenciavel(ctx, quem, usuarioID)
	if err != nil {
		return err
	}
	if alvo.Papel == models.PapelDono && alvo.Ativo {
		if err := s.garantirOutroDono(ctx, alvo.ContaID); err != nil {
			return err
		}
	}
	return s.store.ExcluirUsuario(ctx, alvo.ID)
}

func (s *ContaService) membroGerenciavel(ctx context.Context, quem models.AcessoDashboard, usuarioID string) (models.Usuario, error) {
	alvo, err := s.store.BuscarUsuarioPorID(ctx, usuarioID)
	if err != nil || alvo.Superadmin {
		return models.Usuario{}, fmt.Errorf("%w: usuario nao encontrado", ErrEntradaInvalida)
	}
	if quem.UsuarioID == alvo.ID {
		return models.Usuario{}, fmt.Errorf("%w: voce nao pode alterar o proprio acesso", ErrEntradaInvalida)
	}
	if quem.TudoLiberado() {
		return alvo, nil
	}
	if quem.Tipo != "usuario" || quem.ContaID != alvo.ContaID || !models.PapelGerencia(quem.Papel) {
		return models.Usuario{}, ErrAcessoNegado
	}
	// Admin nao mexe em dono.
	if quem.Papel == models.PapelAdmin && alvo.Papel == models.PapelDono {
		return models.Usuario{}, ErrAcessoNegado
	}
	return alvo, nil
}

func (s *ContaService) garantirOutroDono(ctx context.Context, contaID string) error {
	donos, err := s.store.ContarDonosAtivos(ctx, contaID)
	if err != nil {
		return err
	}
	if donos <= 1 {
		return fmt.Errorf("%w: a conta precisa de pelo menos um dono ativo", ErrEntradaInvalida)
	}
	return nil
}

// CancelarConvite apaga um convite ainda nao usado.
func (s *ContaService) CancelarConvite(ctx context.Context, quem models.AcessoDashboard, contaID, conviteID string) error {
	contaID, err := contaDaEquipe(quem, contaID)
	if err != nil {
		return err
	}
	if !quem.TudoLiberado() && !models.PapelGerencia(quem.Papel) {
		return ErrAcessoNegado
	}
	if err := s.store.ExcluirConvite(ctx, conviteID, contaID); err != nil {
		return fmt.Errorf("%w: convite nao encontrado", ErrEntradaInvalida)
	}
	return nil
}

// ── Contas e planos (superadmin) ─────────────────────────────────────────────

func (s *ContaService) ListarContas(ctx context.Context, quem models.AcessoDashboard) ([]models.ContaResumo, error) {
	if !quem.TudoLiberado() {
		return nil, ErrAcessoNegado
	}
	contas, err := s.store.ListarContasResumo(ctx)
	return vazioSeNil(contas), err
}

// CriarConta cria a empresa e convida o dono.
func (s *ContaService) CriarConta(ctx context.Context, quem models.AcessoDashboard, nome, planoID, emailDono string) (models.Conta, ResultadoConvite, error) {
	if !quem.TudoLiberado() {
		return models.Conta{}, ResultadoConvite{}, ErrAcessoNegado
	}
	nome = strings.TrimSpace(nome)
	if nome == "" || !emailValido(emailDono) {
		return models.Conta{}, ResultadoConvite{}, fmt.Errorf("%w: informe o nome da empresa e o e-mail do dono", ErrEntradaInvalida)
	}
	if planoID != "" {
		if _, err := s.store.BuscarPlano(ctx, planoID); err != nil {
			return models.Conta{}, ResultadoConvite{}, fmt.Errorf("%w: plano nao encontrado", ErrEntradaInvalida)
		}
	}
	if _, err := s.store.BuscarUsuarioPorEmail(ctx, emailDono); err == nil {
		return models.Conta{}, ResultadoConvite{}, fmt.Errorf("%w: ja existe um usuario com esse e-mail", ErrEntradaInvalida)
	}
	agora := time.Now()
	conta := models.Conta{ID: uuid.NewString(), Nome: nome, PlanoID: planoID, Status: models.ContaAtiva, CriadoEm: agora, AtualizadoEm: agora}
	if err := s.store.CriarConta(ctx, conta); err != nil {
		return models.Conta{}, ResultadoConvite{}, err
	}
	convite, err := s.Convidar(ctx, quem, conta.ID, emailDono, models.PapelDono)
	return conta, convite, err
}

// AtualizarConta muda nome, plano, status (ativa/bloqueada) ou limite.
func (s *ContaService) AtualizarConta(ctx context.Context, quem models.AcessoDashboard, conta models.Conta) (models.Conta, error) {
	if !quem.TudoLiberado() {
		return models.Conta{}, ErrAcessoNegado
	}
	atual, err := s.store.BuscarConta(ctx, conta.ID)
	if err != nil {
		return models.Conta{}, fmt.Errorf("%w: conta nao encontrada", ErrEntradaInvalida)
	}
	if nome := strings.TrimSpace(conta.Nome); nome != "" {
		atual.Nome = nome
	}
	if conta.Status != "" {
		if conta.Status != models.ContaAtiva && conta.Status != models.ContaBloqueada {
			return models.Conta{}, fmt.Errorf("%w: status deve ser ativa ou bloqueada", ErrEntradaInvalida)
		}
		atual.Status = conta.Status
	}
	if conta.PlanoID != atual.PlanoID {
		if conta.PlanoID != "" {
			if _, err := s.store.BuscarPlano(ctx, conta.PlanoID); err != nil {
				return models.Conta{}, fmt.Errorf("%w: plano nao encontrado", ErrEntradaInvalida)
			}
		}
		atual.PlanoID = conta.PlanoID
	}
	if conta.LimiteInstancias >= 0 {
		atual.LimiteInstancias = conta.LimiteInstancias
	}
	tipos, err := normalizarTiposInstancia(conta.TiposInstancia)
	if err != nil {
		return models.Conta{}, err
	}
	atual.TiposInstancia = tipos
	if err := s.store.AtualizarConta(ctx, atual); err != nil {
		return models.Conta{}, err
	}
	return atual, nil
}

// MoverInstancia coloca uma instancia numa conta ("" volta para o
// superadmin).
func (s *ContaService) MoverInstancia(ctx context.Context, quem models.AcessoDashboard, instanciaID, contaID string) error {
	if !quem.TudoLiberado() {
		return ErrAcessoNegado
	}
	if contaID != "" {
		if _, err := s.store.BuscarConta(ctx, contaID); err != nil {
			return fmt.Errorf("%w: conta nao encontrada", ErrEntradaInvalida)
		}
	}
	if err := s.store.AtualizarContaInstancia(ctx, instanciaID, contaID); err != nil {
		if errors.Is(err, store.ErrInstanciaNaoEncontrada) {
			return ErrInstanciaNaoEncontrada
		}
		return err
	}
	return nil
}

// LimiteEfetivo e o limite de instancias da conta (0 = sem limite).
func (s *ContaService) LimiteEfetivo(ctx context.Context, contaID string) (int, error) {
	conta, err := s.store.BuscarConta(ctx, contaID)
	if err != nil {
		return 0, err
	}
	if conta.LimiteInstancias > 0 {
		return conta.LimiteInstancias, nil
	}
	if conta.PlanoID == "" {
		return 0, nil
	}
	plano, err := s.store.BuscarPlano(ctx, conta.PlanoID)
	if err != nil {
		return 0, nil
	}
	return plano.LimiteInstancias, nil
}

// GarantirLimite recusa criar instancia alem do plano.
func (s *ContaService) GarantirLimite(ctx context.Context, contaID string) error {
	if contaID == "" {
		return nil
	}
	limite, err := s.LimiteEfetivo(ctx, contaID)
	if err != nil || limite == 0 {
		return err
	}
	usadas, err := s.store.ContarInstanciasConta(ctx, contaID)
	if err != nil {
		return err
	}
	if usadas >= limite {
		return fmt.Errorf("%w: o plano permite %d instancia(s); fale com o suporte para aumentar", ErrLimiteInstancias, limite)
	}
	return nil
}

// normalizarTiposInstancia confere a lista ("meta, instagram") e a devolve
// sem repetidos; todos marcados vira vazio (= todos, inclusive tipos novos).
func normalizarTiposInstancia(lista string) (string, error) {
	validos := []string{models.TipoInstanciaWhatsApp, models.TipoInstanciaMeta, models.TipoInstanciaInstagram}
	var tipos []string
	for _, t := range strings.Split(lista, ",") {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || slices.Contains(tipos, t) {
			continue
		}
		if !slices.Contains(validos, t) {
			return "", fmt.Errorf("%w: tipo de instancia invalido: %s", ErrEntradaInvalida, t)
		}
		tipos = append(tipos, t)
	}
	if len(tipos) == len(validos) {
		return "", nil
	}
	return strings.Join(tipos, ","), nil
}

// GarantirTipo recusa criar instancia de tipo que a conta nao usa.
func (s *ContaService) GarantirTipo(ctx context.Context, contaID, tipo string) error {
	if contaID == "" {
		return nil
	}
	conta, err := s.store.BuscarConta(ctx, contaID)
	if err != nil {
		return err
	}
	if tipo = strings.ToLower(strings.TrimSpace(tipo)); tipo == "" {
		tipo = models.TipoInstanciaWhatsApp
	}
	if !conta.PermiteTipo(tipo) {
		return fmt.Errorf("%w: esta conta nao pode criar instancias do tipo %s; fale com o suporte", ErrAcessoNegado, tipo)
	}
	return nil
}

func (s *ContaService) ListarPlanos(ctx context.Context) ([]models.Plano, error) {
	planos, err := s.store.ListarPlanos(ctx)
	return vazioSeNil(planos), err
}

// SalvarPlano cria (sem id) ou atualiza um plano.
func (s *ContaService) SalvarPlano(ctx context.Context, quem models.AcessoDashboard, plano models.Plano) (models.Plano, error) {
	if !quem.TudoLiberado() {
		return models.Plano{}, ErrAcessoNegado
	}
	plano.Nome = strings.TrimSpace(plano.Nome)
	if plano.Nome == "" || plano.PrecoCentavos < 0 || plano.LimiteInstancias < 0 {
		return models.Plano{}, fmt.Errorf("%w: informe nome, preco e limite validos", ErrEntradaInvalida)
	}
	if plano.ID == "" {
		plano.ID = uuid.NewString()
	} else if atual, err := s.store.BuscarPlano(ctx, plano.ID); err == nil {
		plano.CriadoEm = atual.CriadoEm
	}
	if err := s.store.SalvarPlano(ctx, plano); err != nil {
		return models.Plano{}, err
	}
	return plano, nil
}

// Administradores lista os superadmins e os convites de superadmin pendentes.
func (s *ContaService) Administradores(ctx context.Context, quem models.AcessoDashboard) (map[string]any, error) {
	if !quem.TudoLiberado() {
		return nil, ErrAcessoNegado
	}
	usuarios, err := s.store.ListarUsuarios(ctx, "", true)
	if err != nil {
		return nil, err
	}
	convites, err := s.store.ListarConvitesPendentes(ctx, "", true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"usuarios": vazioSeNil(usuarios), "convites": vazioSeNil(convites)}, nil
}

// AtribuirInstanciaNova coloca na conta a instancia que acabou de ser criada
// por um usuario dela (a permissao ja foi conferida na criacao).
func (s *ContaService) AtribuirInstanciaNova(ctx context.Context, instanciaID, contaID string) error {
	if contaID == "" {
		return nil
	}
	return s.store.AtualizarContaInstancia(ctx, instanciaID, contaID)
}
