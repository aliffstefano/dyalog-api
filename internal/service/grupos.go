package service

import (
	"context"
	"errors"
	"fmt"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/whatsapp"
)

// Grupos da conta de WhatsApp (instancias por QR code).

func traduzirErroGrupo(err error) error {
	if errors.Is(err, whatsapp.ErrGrupoInvalido) {
		return fmt.Errorf("%w: %s", ErrEntradaInvalida, err.Error())
	}
	return traduzirErroWhatsApp(err)
}

func (s *InstanciaService) GruposWhatsApp(ctx context.Context, id string) ([]models.GrupoResumo, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return nil, err
	}
	grupos, err := s.gerenciador.GruposWhatsApp(ctx, id)
	return grupos, traduzirErroGrupo(err)
}

func (s *InstanciaService) GrupoWhatsApp(ctx context.Context, id, grupo string) (models.GrupoDetalhe, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.GrupoDetalhe{}, err
	}
	detalhe, err := s.gerenciador.GrupoWhatsApp(ctx, id, grupo)
	return detalhe, traduzirErroGrupo(err)
}

func (s *InstanciaService) CriarGrupoWhatsApp(ctx context.Context, id string, req models.CriarGrupoRequest) (models.GrupoDetalhe, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.GrupoDetalhe{}, err
	}
	detalhe, err := s.gerenciador.CriarGrupoWhatsApp(ctx, id, req)
	return detalhe, traduzirErroGrupo(err)
}

func (s *InstanciaService) AtualizarGrupoWhatsApp(ctx context.Context, id, grupo string, req models.AtualizarGrupoRequest) error {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return err
	}
	if req.Nome == nil && req.Descricao == nil && req.SoAdminsEnviam == nil && req.SoAdminsEditam == nil {
		return fmt.Errorf("%w: nada para alterar", ErrEntradaInvalida)
	}
	if req.Nome != nil && (len([]rune(*req.Nome)) == 0 || len([]rune(*req.Nome)) > 100) {
		return fmt.Errorf("%w: o nome do grupo deve ter de 1 a 100 caracteres", ErrEntradaInvalida)
	}
	return traduzirErroGrupo(s.gerenciador.AtualizarGrupoWhatsApp(ctx, id, grupo, req))
}

func (s *InstanciaService) FotoGrupoWhatsApp(ctx context.Context, id, grupo string, req models.AtualizarFotoPerfilRequest, remover bool) error {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return err
	}
	if remover {
		return traduzirErroGrupo(s.gerenciador.FotoGrupoWhatsApp(ctx, id, grupo, nil))
	}
	dados, err := lerImagemPerfil(ctx, req)
	if err != nil {
		return err
	}
	jpeg, err := whatsapp.PrepararFotoPerfil(dados)
	if err != nil {
		return traduzirErroWhatsApp(err)
	}
	return traduzirErroGrupo(s.gerenciador.FotoGrupoWhatsApp(ctx, id, grupo, jpeg))
}

func (s *InstanciaService) LinkConviteGrupoWhatsApp(ctx context.Context, id, grupo string, redefinir bool) (string, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return "", err
	}
	link, err := s.gerenciador.LinkConviteGrupoWhatsApp(ctx, id, grupo, redefinir)
	return link, traduzirErroGrupo(err)
}

func (s *InstanciaService) ParticipantesGrupoWhatsApp(ctx context.Context, id, grupo string, req models.ParticipantesGrupoRequest) ([]models.ResultadoParticipante, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return nil, err
	}
	if len(req.Numeros) == 0 || len(req.Numeros) > 100 {
		return nil, fmt.Errorf("%w: informe de 1 a 100 numeros", ErrEntradaInvalida)
	}
	resultado, err := s.gerenciador.ParticipantesGrupoWhatsApp(ctx, id, grupo, req)
	return resultado, traduzirErroGrupo(err)
}

func (s *InstanciaService) SairGrupoWhatsApp(ctx context.Context, id, grupo string) error {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return err
	}
	return traduzirErroGrupo(s.gerenciador.SairGrupoWhatsApp(ctx, id, grupo))
}
