package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
)

func (s *MensagemService) EnviarEvento(ctx context.Context, req models.EnvioEventoRequest) (models.ResultadoEnvio, error) {
	if err := s.validarDestino(ctx, req.Instancia, req.Numero, req.ChatJID); err != nil {
		return models.ResultadoEnvio{}, err
	}
	if strings.TrimSpace(req.Nome) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe nome do evento", ErrEntradaInvalida)
	}
	if len([]rune(strings.TrimSpace(req.Nome))) > 256 {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: nome do evento deve ter no maximo 256 caracteres", ErrEntradaInvalida)
	}
	inicio, err := lerDataEvento(req.Inicio)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: inicio %v", ErrEntradaInvalida, err)
	}
	if inicio.Before(time.Now().Add(-5 * time.Minute)) {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: inicio do evento ja passou", ErrEntradaInvalida)
	}
	req.InicioEm = inicio
	if strings.TrimSpace(req.Fim) != "" {
		fim, err := lerDataEvento(req.Fim)
		if err != nil {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: fim %v", ErrEntradaInvalida, err)
		}
		if !fim.After(inicio) {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: fim do evento deve ser depois do inicio", ErrEntradaInvalida)
		}
		req.FimEm = fim
	}
	return s.registrarEnvio(req.Instancia)(s.gerenciador.EnviarEvento(ctx, req))
}

// lerDataEvento aceita RFC3339 ou "2006-01-02 15:04" no fuso do servidor.
func lerDataEvento(valor string) (time.Time, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return time.Time{}, fmt.Errorf("e obrigatorio")
	}
	if data, err := time.Parse(time.RFC3339, valor); err == nil {
		return data, nil
	}
	for _, formato := range []string{"2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02 15:04:05"} {
		if data, err := time.ParseInLocation(formato, valor, time.Local); err == nil {
			return data, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalido: use 2026-10-20T19:00:00-04:00 ou 2026-10-20 19:00")
}

func (s *MensagemService) PostarStatus(ctx context.Context, req models.EnvioStatusRequest) (models.ResultadoEnvio, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoEnvio{}, ErrInstanciaNaoEncontrada
	}
	temArquivo := strings.TrimSpace(req.ArquivoURL) != "" || strings.TrimSpace(req.ArquivoBase64) != "" || strings.TrimSpace(req.CaminhoLocal) != ""
	tipo := strings.ToLower(strings.TrimSpace(req.Tipo))
	if tipo == "" && !temArquivo {
		tipo = "texto"
	}
	switch tipo {
	case "":
		// Sem tipo e com arquivo: imagem ou video, decidido no envio.
	case "texto", "text":
		if temArquivo {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: status de texto nao leva arquivo; use tipo imagem ou video", ErrEntradaInvalida)
		}
		texto := strings.TrimSpace(req.Texto)
		if texto == "" {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: informe texto, ou arquivo_url/arquivo_base64 para imagem e video", ErrEntradaInvalida)
		}
		if len([]rune(texto)) > 700 {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: status de texto deve ter no maximo 700 caracteres", ErrEntradaInvalida)
		}
	case "imagem", "image", "foto", "video":
		if !temArquivo {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: informe arquivo_url, arquivo_base64 ou caminho_local", ErrEntradaInvalida)
		}
	default:
		return models.ResultadoEnvio{}, fmt.Errorf("%w: tipo deve ser texto, imagem ou video", ErrEntradaInvalida)
	}
	return s.registrarEnvio(req.Instancia)(s.gerenciador.PostarStatus(ctx, req))
}

const maxCartoesCarrossel = 10

func (s *MensagemService) EnviarCarrossel(ctx context.Context, req models.EnvioCarrosselRequest) (models.ResultadoEnvio, error) {
	if err := s.validarDestino(ctx, req.Instancia, req.Numero, req.ChatJID); err != nil {
		return models.ResultadoEnvio{}, err
	}
	if len(req.Cartoes) == 0 || len(req.Cartoes) > maxCartoesCarrossel {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe de 1 a %d cartoes", ErrEntradaInvalida, maxCartoesCarrossel)
	}
	for i := range req.Cartoes {
		cartao := &req.Cartoes[i]
		if strings.TrimSpace(cartao.ImagemURL) == "" && strings.TrimSpace(cartao.ImagemBase64) == "" {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: cartao %d precisa de imagem_url ou imagem_base64", ErrEntradaInvalida, i+1)
		}
		if strings.TrimSpace(cartao.Texto) == "" {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: cartao %d precisa de texto", ErrEntradaInvalida, i+1)
		}
		if len(cartao.Botoes) == 0 || len(cartao.Botoes) > 3 {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: cartao %d precisa de 1 a 3 botoes", ErrEntradaInvalida, i+1)
		}
		for j := range cartao.Botoes {
			cartao.Botoes[j] = normalizarBotaoCompat(j, cartao.Botoes[j])
		}
		if err := validarBotoes(cartao.Botoes); err != nil {
			return models.ResultadoEnvio{}, fmt.Errorf("cartao %d: %w", i+1, err)
		}
	}
	return s.registrarEnvio(req.Instancia)(s.gerenciador.EnviarCarrossel(ctx, req))
}
