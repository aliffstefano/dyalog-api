package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	"dyalog-api-go/internal/whatsapp"
)

type MensagemService struct {
	instanciaStore store.InstanciaStore
	gerenciador    *whatsapp.GerenciadorInstancias
}

func NovoMensagemService(instanciaStore store.InstanciaStore, gerenciador *whatsapp.GerenciadorInstancias) *MensagemService {
	return &MensagemService{instanciaStore: instanciaStore, gerenciador: gerenciador}
}

func (s *MensagemService) EnviarTexto(ctx context.Context, req models.EnvioTextoRequest) (models.ResultadoEnvio, error) {
	req = normalizarTextoCompat(req)
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoEnvio{}, ErrInstanciaNaoEncontrada
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.Mensagem) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe mensagem ou Body", ErrEntradaInvalida)
	}
	return s.gerenciador.EnviarTexto(ctx, req)
}

func normalizarTextoCompat(req models.EnvioTextoRequest) models.EnvioTextoRequest {
	if strings.TrimSpace(req.Numero) == "" {
		req.Numero = strings.TrimSpace(req.Phone)
	}
	if strings.TrimSpace(req.Mensagem) == "" {
		req.Mensagem = strings.TrimSpace(req.Body)
	}
	if strings.TrimSpace(req.MensagemID) == "" {
		req.MensagemID = strings.TrimSpace(req.ID)
	}
	if req.ContextInfo != nil {
		if strings.TrimSpace(req.RespostaMensagemID) == "" {
			req.RespostaMensagemID = strings.TrimSpace(req.ContextInfo.StanzaID)
		}
		if strings.TrimSpace(req.RespostaParticipante) == "" {
			req.RespostaParticipante = strings.TrimSpace(req.ContextInfo.Participant)
		}
	}
	return req
}

func normalizarMarcarLidaCompat(req models.MarcarLidaRequest) models.MarcarLidaRequest {
	if strings.TrimSpace(req.Numero) == "" {
		req.Numero = strings.TrimSpace(req.Phone)
	}
	if strings.TrimSpace(req.MensagemID) == "" {
		req.MensagemID = strings.TrimSpace(req.ID)
	}
	if len(req.MensagensID) == 0 && len(req.IDs) > 0 {
		req.MensagensID = append([]string(nil), req.IDs...)
	}
	if strings.TrimSpace(req.Participante) == "" {
		req.Participante = strings.TrimSpace(req.RemetenteJID)
	}
	if strings.TrimSpace(req.Participante) == "" {
		req.Participante = strings.TrimSpace(req.Participant)
	}
	if strings.TrimSpace(req.LidaEm) == "" {
		req.LidaEm = strings.TrimSpace(req.Timestamp)
	}
	return req
}

func normalizarListaMensagensID(mensagemID string, mensagensID []string) []string {
	normalizados := make([]string, 0, len(mensagensID)+1)
	if id := strings.TrimSpace(mensagemID); id != "" {
		normalizados = append(normalizados, id)
	}
	for _, id := range mensagensID {
		if id = strings.TrimSpace(id); id != "" {
			normalizados = append(normalizados, id)
		}
	}
	return deduplicarIDs(normalizados)
}

func deduplicarIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	resultado := make([]string, 0, len(ids))
	visitados := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := visitados[id]; ok {
			continue
		}
		visitados[id] = struct{}{}
		resultado = append(resultado, id)
	}
	return resultado
}

func parseLidaEm(valor string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, err := time.Parse(layout, strings.TrimSpace(valor)); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("formato invalido")
}

func (s *MensagemService) EnviarPresenca(ctx context.Context, req models.EnvioPresencaRequest) (models.ResultadoPresenca, error) {
	req = normalizarPresencaCompat(req)
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoPresenca{}, ErrInstanciaNaoEncontrada
	}
	acao := strings.ToLower(strings.TrimSpace(req.Acao))
	if acao == "" {
		return models.ResultadoPresenca{}, fmt.Errorf("%w: informe acao", ErrEntradaInvalida)
	}
	switch acao {
	case "digitando", "composing", "gravando_audio", "gravando", "audio", "pausado", "parou", "paused", "disponivel", "online", "available", "indisponivel", "offline", "unavailable":
	default:
		return models.ResultadoPresenca{}, fmt.Errorf("%w: acao deve ser digitando, gravando_audio, pausado, disponivel ou indisponivel", ErrEntradaInvalida)
	}
	if !acaoPresencaGlobal(acao) && strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoPresenca{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	resultado, err := s.gerenciador.EnviarPresenca(ctx, req)
	if err != nil {
		return models.ResultadoPresenca{}, err
	}
	if presenca := normalizarPresencaGlobalPersistida(acao); presenca != "" {
		_, _ = s.instanciaStore.AtualizarPresenca(ctx, req.Instancia, presenca)
	}
	return resultado, nil
}

func normalizarPresencaCompat(req models.EnvioPresencaRequest) models.EnvioPresencaRequest {
	if strings.TrimSpace(req.Acao) == "" {
		req.Acao = strings.TrimSpace(req.Type)
	}
	return req
}

func acaoPresencaGlobal(acao string) bool {
	switch strings.ToLower(strings.TrimSpace(acao)) {
	case "disponivel", "online", "available", "indisponivel", "offline", "unavailable":
		return true
	default:
		return false
	}
}

func normalizarPresencaGlobalPersistida(acao string) string {
	switch strings.ToLower(strings.TrimSpace(acao)) {
	case "disponivel", "online", "available":
		return models.PresencaDisponivel
	case "indisponivel", "offline", "unavailable":
		return models.PresencaIndisponivel
	default:
		return ""
	}
}

func (s *MensagemService) MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error) {
	req = normalizarMarcarLidaCompat(req)
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoMarcarLida{}, ErrInstanciaNaoEncontrada
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoMarcarLida{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(req.ChatJID)), "@g.us") {
		req.Grupo = true
	}
	req.MensagensID = normalizarListaMensagensID(req.MensagemID, req.MensagensID)
	if len(req.MensagensID) == 0 {
		return models.ResultadoMarcarLida{}, fmt.Errorf("%w: informe mensagem_id ou mensagens_id", ErrEntradaInvalida)
	}
	if req.Grupo && strings.TrimSpace(req.Participante) == "" {
		return models.ResultadoMarcarLida{}, fmt.Errorf("%w: informe participante ou remetente_jid para marcar mensagem de grupo como lida", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.LidaEm) != "" {
		marcadaEm, err := parseLidaEm(req.LidaEm)
		if err != nil {
			return models.ResultadoMarcarLida{}, fmt.Errorf("%w: lida_em deve estar em formato RFC3339", ErrEntradaInvalida)
		}
		req.MarcadaEmTime = marcadaEm
	}
	if req.MarcadaEmTime.IsZero() {
		req.MarcadaEmTime = time.Now().UTC()
	}
	return s.gerenciador.MarcarLida(ctx, req)
}

func (s *MensagemService) EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error) {
	req = normalizarBotoesCompat(req)
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoEnvio{}, ErrInstanciaNaoEncontrada
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.Texto) == "" && strings.TrimSpace(req.Mensagem) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe texto ou mensagem", ErrEntradaInvalida)
	}
	if len(req.Botoes) == 0 || len(req.Botoes) > 3 {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe de 1 a 3 botoes", ErrEntradaInvalida)
	}
	for _, botao := range req.Botoes {
		tipo := tipoBotao(botao)
		if strings.TrimSpace(botao.Texto) == "" {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: cada botao precisa de texto ou DisplayText", ErrEntradaInvalida)
		}
		if len([]rune(strings.TrimSpace(botao.Texto))) > 20 {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: texto de botao deve ter no maximo 20 caracteres", ErrEntradaInvalida)
		}
		switch tipo {
		case "", "quickreply", "quick_reply", "reply":
			if strings.TrimSpace(botao.ID) == "" {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: botao quickreply precisa de id", ErrEntradaInvalida)
			}
			if len(strings.TrimSpace(botao.ID)) > 256 {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: id de botao deve ter no maximo 256 caracteres", ErrEntradaInvalida)
			}
		case "url":
			if strings.TrimSpace(urlBotao(botao)) == "" {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: botao url precisa de URL ou Url", ErrEntradaInvalida)
			}
		case "call":
			if strings.TrimSpace(botao.PhoneNumber) == "" {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: botao call precisa de PhoneNumber", ErrEntradaInvalida)
			}
		default:
			return models.ResultadoEnvio{}, fmt.Errorf("%w: tipo de botao deve ser quickreply, url ou call", ErrEntradaInvalida)
		}
	}
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "", "auto", "buttons", "legacy", "native_flow", "native_flow_direct", "direct", "native_flow_view_once", "view_once", "viewonce", "template", "wuzapi_template", "hydrated_template", "texto", "text", "fallback_texto":
	default:
		return models.ResultadoEnvio{}, fmt.Errorf("%w: modo deve ser native_flow, native_flow_view_once, template, texto, buttons ou auto", ErrEntradaInvalida)
	}
	return s.gerenciador.EnviarBotoes(ctx, req)
}

func (s *MensagemService) EnviarLista(ctx context.Context, req models.EnvioListaRequest) (models.ResultadoEnvio, error) {
	req = normalizarListaCompat(req)
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoEnvio{}, ErrInstanciaNaoEncontrada
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if strings.TrimSpace(textoMensagemLista(req)) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe descricao, mensagem ou Desc", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.BotaoTexto) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe botao_texto ou ButtonText", ErrEntradaInvalida)
	}
	totalLinhas := 0
	if len(req.Secoes) == 0 {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe opcoes, secoes ou List", ErrEntradaInvalida)
	}
	for _, secao := range req.Secoes {
		if len(secao.Linhas) == 0 {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: cada secao precisa de pelo menos uma linha", ErrEntradaInvalida)
		}
		totalLinhas += len(secao.Linhas)
		for _, linha := range secao.Linhas {
			if strings.TrimSpace(linha.ID) == "" {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: cada linha precisa de id ou RowId", ErrEntradaInvalida)
			}
			if strings.TrimSpace(linha.Titulo) == "" {
				return models.ResultadoEnvio{}, fmt.Errorf("%w: cada linha precisa de titulo ou title", ErrEntradaInvalida)
			}
		}
	}
	if totalLinhas == 0 || totalLinhas > 10 {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: a lista deve ter entre 1 e 10 linhas no total", ErrEntradaInvalida)
	}
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "", "lista", "list", "texto", "text", "fallback_texto":
	default:
		return models.ResultadoEnvio{}, fmt.Errorf("%w: modo deve ser lista, texto ou auto", ErrEntradaInvalida)
	}
	return s.gerenciador.EnviarLista(ctx, req)
}

func normalizarBotoesCompat(req models.EnvioBotoesRequest) models.EnvioBotoesRequest {
	usouCompat := false
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.Phone) != "" {
		req.Numero = strings.TrimSpace(req.Phone)
		usouCompat = true
	}
	if strings.TrimSpace(req.Mensagem) == "" && strings.TrimSpace(req.Texto) == "" && strings.TrimSpace(req.Content) != "" {
		req.Mensagem = strings.TrimSpace(req.Content)
		usouCompat = true
	}
	if strings.TrimSpace(req.Rodape) == "" && strings.TrimSpace(req.Footer) != "" {
		req.Rodape = strings.TrimSpace(req.Footer)
		usouCompat = true
	}
	if strings.TrimSpace(req.MensagemID) == "" && strings.TrimSpace(req.ID) != "" {
		req.MensagemID = strings.TrimSpace(req.ID)
		usouCompat = true
	}
	if len(req.Botoes) == 0 && len(req.Buttons) > 0 {
		req.Botoes = req.Buttons
		usouCompat = true
	}
	if req.ContextInfo != nil {
		if strings.TrimSpace(req.RespostaMensagemID) == "" {
			req.RespostaMensagemID = strings.TrimSpace(req.ContextInfo.StanzaID)
		}
		if strings.TrimSpace(req.RespostaParticipante) == "" {
			req.RespostaParticipante = strings.TrimSpace(req.ContextInfo.Participant)
		}
		usouCompat = true
	}
	for i := range req.Botoes {
		req.Botoes[i] = normalizarBotaoCompat(i, req.Botoes[i])
	}
	if usouCompat && strings.TrimSpace(req.Modo) == "" {
		req.Modo = "template"
	}
	return req
}

func normalizarBotaoCompat(indice int, botao models.BotaoRequest) models.BotaoRequest {
	if strings.TrimSpace(botao.Texto) == "" {
		botao.Texto = strings.TrimSpace(botao.DisplayText)
	}
	if strings.TrimSpace(botao.Tipo) == "" {
		botao.Tipo = strings.TrimSpace(botao.Type)
	}
	if strings.TrimSpace(botao.ID) == "" && ehQuickReply(botao) {
		if texto := strings.TrimSpace(botao.Texto); texto != "" {
			botao.ID = texto
		} else {
			botao.ID = fmt.Sprintf("botao_%d", indice+1)
		}
	}
	return botao
}

func tipoBotao(botao models.BotaoRequest) string {
	tipo := strings.TrimSpace(botao.Tipo)
	if tipo == "" {
		tipo = strings.TrimSpace(botao.Type)
	}
	return strings.ToLower(tipo)
}

func ehQuickReply(botao models.BotaoRequest) bool {
	switch tipoBotao(botao) {
	case "", "quickreply", "quick_reply", "reply":
		return true
	default:
		return false
	}
}

func urlBotao(botao models.BotaoRequest) string {
	if url := strings.TrimSpace(botao.URL); url != "" {
		return url
	}
	return strings.TrimSpace(botao.Url)
}

func normalizarListaCompat(req models.EnvioListaRequest) models.EnvioListaRequest {
	usouCompat := false
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.Phone) != "" {
		req.Numero = strings.TrimSpace(req.Phone)
		usouCompat = true
	}
	if strings.TrimSpace(req.Titulo) == "" && strings.TrimSpace(req.TopText) != "" {
		req.Titulo = strings.TrimSpace(req.TopText)
		usouCompat = true
	}
	if strings.TrimSpace(req.Descricao) == "" && strings.TrimSpace(req.Mensagem) == "" && strings.TrimSpace(req.Desc) != "" {
		req.Descricao = strings.TrimSpace(req.Desc)
		usouCompat = true
	}
	if strings.TrimSpace(req.BotaoTexto) == "" && strings.TrimSpace(req.ButtonText) != "" {
		req.BotaoTexto = strings.TrimSpace(req.ButtonText)
		usouCompat = true
	}
	if strings.TrimSpace(req.Rodape) == "" && strings.TrimSpace(req.FooterText) != "" {
		req.Rodape = strings.TrimSpace(req.FooterText)
		usouCompat = true
	}
	if strings.TrimSpace(req.MensagemID) == "" && strings.TrimSpace(req.ID) != "" {
		req.MensagemID = strings.TrimSpace(req.ID)
		usouCompat = true
	}
	if len(req.Opcoes) == 0 && len(req.List) > 0 {
		req.Opcoes = req.List
		usouCompat = true
	}
	if len(req.Secoes) == 0 && len(req.Opcoes) > 0 {
		linhas := make([]models.ListaLinhaRequest, 0, len(req.Opcoes))
		for i, linha := range req.Opcoes {
			linhas = append(linhas, normalizarLinhaListaCompat(i, linha))
		}
		req.Secoes = []models.ListaSecaoRequest{{
			Titulo: "Opcoes",
			Linhas: linhas,
		}}
	}
	for i := range req.Secoes {
		req.Secoes[i] = normalizarSecaoListaCompat(req.Secoes[i])
	}
	if req.ContextInfo != nil {
		if strings.TrimSpace(req.RespostaMensagemID) == "" {
			req.RespostaMensagemID = strings.TrimSpace(req.ContextInfo.StanzaID)
		}
		if strings.TrimSpace(req.RespostaParticipante) == "" {
			req.RespostaParticipante = strings.TrimSpace(req.ContextInfo.Participant)
		}
		usouCompat = true
	}
	if usouCompat && strings.TrimSpace(req.Modo) == "" {
		req.Modo = "lista"
	}
	return req
}

func normalizarSecaoListaCompat(secao models.ListaSecaoRequest) models.ListaSecaoRequest {
	if strings.TrimSpace(secao.Titulo) == "" {
		secao.Titulo = strings.TrimSpace(secao.Title)
	}
	if len(secao.Linhas) == 0 && len(secao.Rows) > 0 {
		secao.Linhas = secao.Rows
	}
	for i := range secao.Linhas {
		secao.Linhas[i] = normalizarLinhaListaCompat(i, secao.Linhas[i])
	}
	return secao
}

func normalizarLinhaListaCompat(indice int, linha models.ListaLinhaRequest) models.ListaLinhaRequest {
	if strings.TrimSpace(linha.ID) == "" {
		linha.ID = strings.TrimSpace(linha.RowID)
	}
	if strings.TrimSpace(linha.Titulo) == "" {
		linha.Titulo = strings.TrimSpace(linha.Title)
	}
	if strings.TrimSpace(linha.Descricao) == "" {
		linha.Descricao = strings.TrimSpace(linha.Desc)
	}
	if strings.TrimSpace(linha.ID) == "" && strings.TrimSpace(linha.Titulo) != "" {
		linha.ID = fmt.Sprintf("linha_%d", indice+1)
	}
	return linha
}

func textoMensagemLista(req models.EnvioListaRequest) string {
	if texto := strings.TrimSpace(req.Descricao); texto != "" {
		return texto
	}
	return strings.TrimSpace(req.Mensagem)
}

func (s *MensagemService) EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return s.enviarMidia(ctx, req, s.gerenciador.EnviarImagem)
}

func (s *MensagemService) EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return s.enviarMidia(ctx, req, s.gerenciador.EnviarAudio)
}

func (s *MensagemService) EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return s.enviarMidia(ctx, req, s.gerenciador.EnviarDocumento)
}

func (s *MensagemService) enviarMidia(ctx context.Context, req models.EnvioMidiaRequest, fn func(context.Context, models.EnvioMidiaRequest) (models.ResultadoEnvio, error)) (models.ResultadoEnvio, error) {
	if _, err := s.instanciaStore.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.ResultadoEnvio{}, ErrInstanciaNaoEncontrada
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if req.ArquivoURL == "" && req.CaminhoLocal == "" && req.ArquivoBase64 == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("%w: informe arquivo_url, arquivo_base64 ou caminho_local", ErrEntradaInvalida)
	}
	resultado, err := fn(ctx, req)
	if err != nil {
		if errors.Is(err, ErrEntradaInvalida) || errors.Is(err, whatsapp.ErrMidiaInvalida) {
			return models.ResultadoEnvio{}, fmt.Errorf("%w: %v", ErrEntradaInvalida, err)
		}
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao preparar envio de midia: %w", err)
	}
	return resultado, nil
}
