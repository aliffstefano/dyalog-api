package service

import (
	"context"
	"errors"
	"strings"

	"dyalog-api-go/internal/models"
)

var ErrNaoAutenticado = errors.New("nao autenticado")
var ErrAcessoNegado = errors.New("acesso negado")

type instanciaAuthStore interface {
	BuscarPorToken(ctx context.Context, token string) (models.Instancia, error)
	BuscarPorID(ctx context.Context, id string) (models.Instancia, error)
}

type AuthService struct {
	masterToken string
	store       instanciaAuthStore
	contas      *ContaService
}

// UsarContas liga o login com e-mail e senha (sessoes de usuario).
func (s *AuthService) UsarContas(c *ContaService) { s.contas = c }

func NovoAuthService(masterToken string, store instanciaAuthStore) *AuthService {
	return &AuthService{masterToken: strings.TrimSpace(masterToken), store: store}
}

func (s *AuthService) Autenticar(ctx context.Context, token string) (models.AcessoDashboard, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return models.AcessoDashboard{}, ErrNaoAutenticado
	}
	if s.masterToken != "" && token == s.masterToken {
		return models.AcessoDashboard{Tipo: "master", Nome: "Administrador"}, nil
	}
	if EhTokenDeSessao(token) {
		if s.contas == nil {
			return models.AcessoDashboard{}, ErrNaoAutenticado
		}
		return s.contas.AcessoPorSessao(ctx, token)
	}
	instancia, err := s.store.BuscarPorToken(ctx, token)
	if err != nil {
		return models.AcessoDashboard{}, ErrNaoAutenticado
	}
	return models.AcessoDashboard{Tipo: "instancia", InstanciaID: instancia.ID, Nome: instancia.Nome}, nil
}

// PodeGerenciarTudo: token master ou superadmin.
func (s *AuthService) PodeGerenciarTudo(acesso models.AcessoDashboard) bool {
	return acesso.TudoLiberado()
}

// GarantirInstancia confere o acesso a instancia. gerenciar=true e para
// configurar (conectar, webhooks, token, credenciais...): na conta, so dono e
// admin; atendente so usa (chat, envio, status). O token da propria instancia
// pode tudo nela.
func (s *AuthService) GarantirInstancia(ctx context.Context, acesso models.AcessoDashboard, instanciaID string, gerenciar bool) error {
	switch {
	case acesso.TudoLiberado():
		return nil
	case acesso.Tipo == "instancia":
		if acesso.InstanciaID == instanciaID {
			return nil
		}
	case acesso.Tipo == "usuario" && acesso.ContaID != "":
		instancia, err := s.store.BuscarPorID(ctx, instanciaID)
		if err != nil || instancia.ContaID != acesso.ContaID {
			return ErrAcessoNegado
		}
		if gerenciar && !models.PapelGerencia(acesso.Papel) {
			return ErrAcessoNegado
		}
		return nil
	}
	return ErrAcessoNegado
}

// PodeCriarInstancias: superadmin, ou dono/admin de uma conta.
func (s *AuthService) PodeCriarInstancias(acesso models.AcessoDashboard) bool {
	return acesso.TudoLiberado() || (acesso.Tipo == "usuario" && acesso.ContaID != "" && models.PapelGerencia(acesso.Papel))
}
