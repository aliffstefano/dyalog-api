package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"

	"github.com/google/uuid"
)

type WebhookService struct {
	instanciaStore store.InstanciaStore
	webhookStore   store.WebhookStore
	entregaStore   store.WebhookEntregaStore
}

func NovoWebhookService(instanciaStore store.InstanciaStore, webhookStore store.WebhookStore, entregaStore store.WebhookEntregaStore) *WebhookService {
	return &WebhookService{instanciaStore: instanciaStore, webhookStore: webhookStore, entregaStore: entregaStore}
}

func (s *WebhookService) Listar(ctx context.Context, instanciaID string) ([]models.WebhookInstancia, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return nil, s.mapearErro(err)
	}
	webhooks, err := s.webhookStore.ListarWebhooks(ctx, instanciaID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar webhooks: %w", err)
	}
	return webhooks, nil
}

func (s *WebhookService) Criar(ctx context.Context, instanciaID, nome, webhookURL string, eventos []string, ativo bool) (models.WebhookInstancia, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return models.WebhookInstancia{}, s.mapearErro(err)
	}
	webhook, err := s.montarWebhook(models.WebhookInstancia{}, instanciaID, nome, webhookURL, eventos, ativo)
	if err != nil {
		return models.WebhookInstancia{}, err
	}
	webhook.ID = uuid.NewString()
	webhook.CriadoEm = time.Now().UTC()
	webhook.AtualizadoEm = webhook.CriadoEm
	criado, err := s.webhookStore.CriarWebhook(ctx, webhook)
	if err != nil {
		return models.WebhookInstancia{}, fmt.Errorf("erro ao criar webhook: %w", err)
	}
	return criado, nil
}

func (s *WebhookService) Atualizar(ctx context.Context, instanciaID, webhookID, nome, webhookURL string, eventos []string, ativo bool) (models.WebhookInstancia, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return models.WebhookInstancia{}, s.mapearErro(err)
	}
	lista, err := s.webhookStore.ListarWebhooks(ctx, instanciaID)
	if err != nil {
		return models.WebhookInstancia{}, fmt.Errorf("erro ao consultar webhook: %w", err)
	}
	var atual models.WebhookInstancia
	for _, item := range lista {
		if item.ID == webhookID {
			atual = item
			break
		}
	}
	if atual.ID == "" {
		return models.WebhookInstancia{}, ErrWebhookNaoEncontrado
	}
	webhook, err := s.montarWebhook(atual, instanciaID, nome, webhookURL, eventos, ativo)
	if err != nil {
		return models.WebhookInstancia{}, err
	}
	webhook.ID = webhookID
	webhook.CriadoEm = atual.CriadoEm
	webhook.AtualizadoEm = time.Now().UTC()
	atualizado, err := s.webhookStore.AtualizarWebhook(ctx, webhook)
	if err != nil {
		return models.WebhookInstancia{}, fmt.Errorf("erro ao atualizar webhook: %w", err)
	}
	return atualizado, nil
}

func (s *WebhookService) Excluir(ctx context.Context, instanciaID, webhookID string) error {
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return s.mapearErro(err)
	}
	if err := s.webhookStore.ExcluirWebhook(ctx, instanciaID, webhookID); err != nil {
		return s.mapearErro(err)
	}
	return nil
}

func (s *WebhookService) ListarEntregas(ctx context.Context, instanciaID string, limite int) ([]models.WebhookEntrega, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return nil, s.mapearErro(err)
	}
	if s.entregaStore == nil {
		return []models.WebhookEntrega{}, nil
	}
	entregas, err := s.entregaStore.ListarWebhookEntregas(ctx, instanciaID, limite)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar entregas de webhook: %w", err)
	}
	return entregas, nil
}

func (s *WebhookService) montarWebhook(base models.WebhookInstancia, instanciaID, nome, webhookURL string, eventos []string, ativo bool) (models.WebhookInstancia, error) {
	nome = strings.TrimSpace(nome)
	webhookURL = strings.TrimSpace(webhookURL)
	if nome == "" || webhookURL == "" {
		return models.WebhookInstancia{}, ErrEntradaInvalida
	}
	parsed, err := url.Parse(webhookURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return models.WebhookInstancia{}, ErrEntradaInvalida
	}
	eventos = sanitizarEventosWebhook(eventos)
	if len(eventos) == 0 {
		return models.WebhookInstancia{}, ErrEntradaInvalida
	}
	base.InstanciaID = instanciaID
	base.Nome = nome
	base.URL = webhookURL
	base.Eventos = eventos
	base.Ativo = ativo
	return base, nil
}

func (s *WebhookService) mapearErro(err error) error {
	if errors.Is(err, store.ErrInstanciaNaoEncontrada) {
		return ErrInstanciaNaoEncontrada
	}
	if errors.Is(err, store.ErrWebhookNaoEncontrado) {
		return ErrWebhookNaoEncontrado
	}
	return err
}

func sanitizarEventosWebhook(eventos []string) []string {
	unicos := make(map[string]struct{})
	for _, evento := range eventos {
		evento = strings.TrimSpace(evento)
		if _, ok := models.EventosWebhookSuportados[evento]; ok {
			unicos[evento] = struct{}{}
		}
	}
	resultado := make([]string, 0, len(unicos))
	for evento := range unicos {
		resultado = append(resultado, evento)
	}
	sort.Strings(resultado)
	return resultado
}
