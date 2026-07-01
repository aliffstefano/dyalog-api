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
}

type AuthService struct {
	masterToken string
	store       instanciaAuthStore
}

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
	instancia, err := s.store.BuscarPorToken(ctx, token)
	if err != nil {
		return models.AcessoDashboard{}, ErrNaoAutenticado
	}
	return models.AcessoDashboard{Tipo: "instancia", InstanciaID: instancia.ID, Nome: instancia.Nome}, nil
}

func (s *AuthService) PodeGerenciarTudo(acesso models.AcessoDashboard) bool {
	return acesso.Tipo == "master"
}

func (s *AuthService) GarantirInstancia(acesso models.AcessoDashboard, instanciaID string) error {
	if acesso.Tipo == "master" {
		return nil
	}
	if acesso.Tipo == "instancia" && acesso.InstanciaID == instanciaID {
		return nil
	}
	return ErrAcessoNegado
}
