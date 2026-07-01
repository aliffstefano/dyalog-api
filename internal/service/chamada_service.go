package service

import (
	"context"
	"fmt"
	"strings"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	"dyalog-api-go/internal/whatsapp"
)

type ChamadaService struct {
	instanciaStore store.InstanciaStore
	gerenciador    *whatsapp.GerenciadorInstancias
}

func NovoChamadaService(instanciaStore store.InstanciaStore, gerenciador *whatsapp.GerenciadorInstancias) *ChamadaService {
	return &ChamadaService{instanciaStore: instanciaStore, gerenciador: gerenciador}
}

func (s *ChamadaService) Iniciar(ctx context.Context, req models.IniciarChamadaRequest) (models.ResultadoChamada, error) {
	if err := s.validarInstancia(ctx, req.Instancia); err != nil {
		return models.ResultadoChamada{}, err
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoChamada{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	return s.gerenciador.IniciarChamada(ctx, req)
}

func (s *ChamadaService) Aceitar(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	if err := s.validarAcao(ctx, req); err != nil {
		return models.ResultadoChamada{}, err
	}
	return s.gerenciador.AceitarChamada(ctx, req)
}

func (s *ChamadaService) Rejeitar(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	if err := s.validarAcao(ctx, req); err != nil {
		return models.ResultadoChamada{}, err
	}
	return s.gerenciador.RejeitarChamada(ctx, req)
}

func (s *ChamadaService) Encerrar(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	if err := s.validarAcao(ctx, req); err != nil {
		return models.ResultadoChamada{}, err
	}
	return s.gerenciador.EncerrarChamada(ctx, req)
}

func (s *ChamadaService) SinalizarWebRTC(ctx context.Context, req models.SinalizacaoWebRTCRequest) (models.ResultadoWebRTC, error) {
	if err := s.validarInstancia(ctx, req.Instancia); err != nil {
		return models.ResultadoWebRTC{}, err
	}
	if strings.TrimSpace(req.ChamadaID) == "" {
		return models.ResultadoWebRTC{}, fmt.Errorf("%w: informe chamada_id", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.SDPOffer) == "" {
		return models.ResultadoWebRTC{}, fmt.Errorf("%w: informe sdp_offer", ErrEntradaInvalida)
	}
	return s.gerenciador.SinalizarWebRTC(ctx, req)
}

func (s *ChamadaService) Listar(ctx context.Context, instanciaID string) ([]models.ResultadoChamada, error) {
	if err := s.validarInstancia(ctx, instanciaID); err != nil {
		return nil, err
	}
	return s.gerenciador.ListarChamadas(ctx, instanciaID)
}

func (s *ChamadaService) validarAcao(ctx context.Context, req models.AcaoChamadaRequest) error {
	if err := s.validarInstancia(ctx, req.Instancia); err != nil {
		return err
	}
	if strings.TrimSpace(req.ChamadaID) == "" {
		return fmt.Errorf("%w: informe chamada_id", ErrEntradaInvalida)
	}
	return nil
}

func (s *ChamadaService) validarInstancia(ctx context.Context, instanciaID string) error {
	if strings.TrimSpace(instanciaID) == "" {
		return fmt.Errorf("%w: informe instancia", ErrEntradaInvalida)
	}
	if _, err := s.instanciaStore.BuscarPorID(ctx, instanciaID); err != nil {
		return ErrInstanciaNaoEncontrada
	}
	return nil
}
