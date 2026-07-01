package service

import (
	"context"
	"errors"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

type MidiaService struct {
	store store.MidiaStore
}

func NovoMidiaService(store store.MidiaStore) *MidiaService {
	return &MidiaService{store: store}
}

func (s *MidiaService) Buscar(ctx context.Context, instanciaID, midiaID string) (models.MidiaRecebida, error) {
	midia, err := s.store.BuscarMidiaRecebida(ctx, instanciaID, midiaID)
	if errors.Is(err, store.ErrMidiaNaoEncontrada) {
		return models.MidiaRecebida{}, ErrMidiaNaoEncontrada
	}
	return midia, err
}
