package service

import (
	"context"
	"errors"
	"fmt"
	neturl "net/url"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	"dyalog-api-go/internal/whatsapp"

	"github.com/google/uuid"
)

type InstanciaService struct {
	store       store.InstanciaStore
	proxyStore  store.ProxyStore
	gerenciador *whatsapp.GerenciadorInstancias
}

func NovoInstanciaService(instanciaStore store.InstanciaStore, gerenciador *whatsapp.GerenciadorInstancias) *InstanciaService {
	proxyStore, _ := instanciaStore.(store.ProxyStore)
	return &InstanciaService{store: instanciaStore, proxyStore: proxyStore, gerenciador: gerenciador}
}

func (s *InstanciaService) RestaurarSessoes(ctx context.Context) {
	instancias, err := s.store.Listar(ctx)
	if err != nil {
		return
	}
	for _, instancia := range instancias {
		instanciaID := instancia.ID
		historicoDias := instancia.HistoricoDias
		go func() {
			s.gerenciador.ConfigurarHistorico(instanciaID, historicoDias)
			restaurada, err := s.gerenciador.RestaurarSessao(context.Background(), instanciaID)
			if err != nil {
				_, _ = s.store.AtualizarStatus(context.Background(), instanciaID, models.StatusInstanciaDesconectada)
				return
			}
			if restaurada {
				_, _ = s.store.AtualizarStatus(context.Background(), instanciaID, "sincronizando_historico")
			}
		}()
	}
}

func (s *InstanciaService) Criar(ctx context.Context, nome string) (models.Instancia, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return models.Instancia{}, ErrEntradaInvalida
	}
	agora := time.Now().UTC()
	instancia := models.Instancia{
		ID:            uuid.NewString(),
		Nome:          nome,
		Token:         uuid.NewString(),
		Status:        models.StatusInstanciaDesconectada,
		HistoricoDias: 0,
		ProxyModo:     models.ProxyModoHerdar,
		ProxyURL:      "",
		Presenca:      models.PresencaIndisponivel,
		CriadoEm:      agora,
		AtualizadoEm:  agora,
	}
	return s.store.Criar(ctx, instancia)
}

func (s *InstanciaService) Listar(ctx context.Context) ([]models.Instancia, error) {
	instancias, err := s.store.Listar(ctx)
	if err != nil {
		return nil, err
	}
	for i := range instancias {
		info, err := s.gerenciador.Info(ctx, instancias[i].ID)
		if err != nil || info.Status == "" {
			continue
		}
		if instancias[i].Status != info.Status {
			instancias[i].Status = info.Status
			instancias[i].AtualizadoEm = info.AtualizadoEm
			_, _ = s.store.AtualizarStatus(ctx, instancias[i].ID, info.Status)
		}
	}
	return instancias, nil
}

func (s *InstanciaService) Buscar(ctx context.Context, id string) (models.Instancia, error) {
	return s.store.BuscarPorID(ctx, id)
}

func (s *InstanciaService) AtualizarToken(ctx context.Context, id, token string) (models.Instancia, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return models.Instancia{}, ErrEntradaInvalida
	}
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	instancia, err := s.store.AtualizarToken(ctx, id, token)
	if err != nil {
		return models.Instancia{}, err
	}
	return instancia, nil
}

func (s *InstanciaService) AtualizarHistorico(ctx context.Context, id string, dias, maxDias int) (models.Instancia, error) {
	if dias < 0 || dias > maxDias {
		return models.Instancia{}, ErrEntradaInvalida
	}
	instancia, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	info, _ := s.gerenciador.Info(ctx, id)
	status := instancia.Status
	if info.Status != "" {
		status = info.Status
	}
	if statusBloqueiaHistorico(status) {
		return models.Instancia{}, ErrHistoricoBloqueado
	}
	instancia, err = s.store.AtualizarHistorico(ctx, id, dias)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	s.gerenciador.ConfigurarHistorico(id, dias)
	return instancia, nil
}

func (s *InstanciaService) AtualizarProxy(ctx context.Context, id, modo, proxyURL string) (models.Instancia, error) {
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	modo, proxyURL, err := normalizarProxyInstancia(modo, proxyURL)
	if err != nil {
		return models.Instancia{}, err
	}
	instancia, err := s.store.AtualizarProxy(ctx, id, modo, proxyURL)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	s.gerenciador.RecarregarInstancia(ctx, id)
	return instancia, nil
}

func (s *InstanciaService) AtualizarPresenca(ctx context.Context, id, presenca string) (models.Instancia, error) {
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	presenca = normalizarPresencaInstancia(presenca)
	if presenca == "" {
		return models.Instancia{}, ErrEntradaInvalida
	}
	instancia, err := s.store.AtualizarPresenca(ctx, id, presenca)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	_ = s.gerenciador.EnviarPresencaGlobal(ctx, id, presenca)
	return instancia, nil
}

func (s *InstanciaService) AtualizarConfiguracaoAvancada(ctx context.Context, id string, cfg models.ConfiguracaoAvancadaInstancia) (models.Instancia, error) {
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	cfg.MensagemRejeitarChamadas = strings.TrimSpace(cfg.MensagemRejeitarChamadas)
	if !cfg.RejeitarChamadas {
		cfg.MensagemRejeitarChamadas = ""
	}
	presenca := models.PresencaIndisponivel
	if cfg.ManterOnline {
		presenca = models.PresencaDisponivel
	}
	instancia, err := s.store.AtualizarConfiguracaoAvancada(ctx, id, cfg, presenca)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	_ = s.gerenciador.EnviarPresencaGlobal(ctx, id, presenca)
	return instancia, nil
}

func (s *InstanciaService) ObterProxyGlobal(ctx context.Context) (models.ProxyGlobal, error) {
	if s.proxyStore == nil {
		return models.ProxyGlobal{}, fmt.Errorf("store de proxy nao configurado")
	}
	return s.proxyStore.ObterProxyGlobal(ctx)
}

func (s *InstanciaService) AtualizarProxyGlobal(ctx context.Context, proxyURL string, ativo bool) (models.ProxyGlobal, error) {
	if s.proxyStore == nil {
		return models.ProxyGlobal{}, fmt.Errorf("store de proxy nao configurado")
	}
	proxyURL = strings.TrimSpace(proxyURL)
	if ativo {
		if err := validarProxyURL(proxyURL); err != nil {
			return models.ProxyGlobal{}, err
		}
	} else {
		proxyURL = ""
	}
	proxy, err := s.proxyStore.AtualizarProxyGlobal(ctx, models.ProxyGlobal{URL: proxyURL, Ativo: ativo})
	if err != nil {
		return models.ProxyGlobal{}, err
	}
	s.recarregarInstanciasHerdandoProxy(ctx)
	return proxy, nil
}

func (s *InstanciaService) recarregarInstanciasHerdandoProxy(ctx context.Context) {
	instancias, err := s.store.Listar(ctx)
	if err != nil {
		return
	}
	for _, instancia := range instancias {
		if modoProxyInstancia(instancia.ProxyModo) != models.ProxyModoProprio {
			s.gerenciador.RecarregarInstancia(ctx, instancia.ID)
		}
	}
}

func (s *InstanciaService) Excluir(ctx context.Context, id string) error {
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return s.mapearErro(err)
	}
	if err := s.gerenciador.ExcluirInstancia(ctx, id); err != nil {
		return fmt.Errorf("erro ao excluir instancia: %w", err)
	}
	if err := s.store.Excluir(ctx, id); err != nil {
		return s.mapearErro(err)
	}
	return nil
}

func (s *InstanciaService) Conectar(ctx context.Context, id string) (models.Instancia, string, error) {
	instanciaSalva, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return models.Instancia{}, "", s.mapearErro(err)
	}
	s.gerenciador.ConfigurarHistorico(id, instanciaSalva.HistoricoDias)
	if _, err := s.store.AtualizarStatus(ctx, id, models.StatusInstanciaConectando); err != nil {
		return models.Instancia{}, "", s.mapearErro(err)
	}
	qrCode, err := s.gerenciador.Conectar(ctx, id)
	if err != nil {
		return models.Instancia{}, "", fmt.Errorf("erro ao conectar instancia: %w", err)
	}
	info, err := s.gerenciador.Info(ctx, id)
	statusAtual := models.StatusInstanciaConectando
	if err == nil && info.Status != "" {
		statusAtual = info.Status
	}
	instancia, err := s.store.AtualizarStatus(ctx, id, statusAtual)
	if err != nil {
		return models.Instancia{}, "", s.mapearErro(err)
	}
	return instancia, qrCode, nil
}

func (s *InstanciaService) SolicitarCodigoPareamento(ctx context.Context, id, numero string) (models.Instancia, map[string]interface{}, error) {
	instanciaSalva, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return models.Instancia{}, nil, s.mapearErro(err)
	}
	s.gerenciador.ConfigurarHistorico(id, instanciaSalva.HistoricoDias)
	if _, err := s.store.AtualizarStatus(ctx, id, models.StatusInstanciaConectando); err != nil {
		return models.Instancia{}, nil, s.mapearErro(err)
	}
	codigo, numeroNormalizado, err := s.gerenciador.SolicitarCodigoPareamento(ctx, id, numero)
	if err != nil {
		return models.Instancia{}, nil, fmt.Errorf("erro ao gerar pairing code: %w", err)
	}
	info, err := s.gerenciador.Info(ctx, id)
	statusAtual := models.StatusInstanciaConectando
	if err == nil && info.Status != "" {
		statusAtual = info.Status
	}
	instancia, err := s.store.AtualizarStatus(ctx, id, statusAtual)
	if err != nil {
		return models.Instancia{}, nil, s.mapearErro(err)
	}
	return instancia, map[string]interface{}{
		"codigo":            codigo,
		"numero":            numeroNormalizado,
		"status":            statusAtual,
		"pairing_code":      info.PairingCode,
		"pairing_phone":     info.PairingPhone,
		"metodo_pareamento": info.MetodoPareamento,
		"atualizado_em":     info.AtualizadoEm,
	}, nil
}

func (s *InstanciaService) Desconectar(ctx context.Context, id string) (models.Instancia, error) {
	if _, err := s.store.BuscarPorID(ctx, id); err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	if err := s.gerenciador.Desconectar(ctx, id); err != nil {
		return models.Instancia{}, fmt.Errorf("erro ao desconectar instancia: %w", err)
	}
	return s.store.AtualizarStatus(ctx, id, models.StatusInstanciaNaoInicializada)
}

func (s *InstanciaService) Status(ctx context.Context, id string) (map[string]interface{}, error) {
	instancia, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return nil, s.mapearErro(err)
	}
	info, err := s.gerenciador.Info(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar status: %w", err)
	}
	status := instancia.Status
	if info.Status != "" && info.Status != models.StatusInstanciaNaoInicializada {
		status = info.Status
		_, _ = s.store.AtualizarStatus(ctx, id, status)
	}
	return map[string]interface{}{
		"id":                    instancia.ID,
		"nome":                  instancia.Nome,
		"token":                 instancia.Token,
		"status":                status,
		"erro":                  info.UltimoErro,
		"atualizado_em":         info.AtualizadoEm,
		"conectado":             status == models.StatusInstanciaConectada,
		"pairing_code":          info.PairingCode,
		"pairing_phone":         info.PairingPhone,
		"pairing_code_pronto":   strings.TrimSpace(info.PairingCode) != "",
		"metodo_pareamento":     info.MetodoPareamento,
		"historico_dias":        instancia.HistoricoDias,
		"historico_configurado": instancia.HistoricoDias > 0,
		"historico_bloqueado":   statusBloqueiaHistorico(status),
		"historico_observacao":  observacaoHistorico(instancia.HistoricoDias),
		"proxy_modo":            modoProxyInstancia(instancia.ProxyModo),
		"proxy_url":             instancia.ProxyURL,
		"proxy_configurado":     modoProxyInstancia(instancia.ProxyModo) == models.ProxyModoProprio && strings.TrimSpace(instancia.ProxyURL) != "",
		"proxy_observacao":      observacaoProxy(instancia.ProxyModo),
		"presenca":              normalizarPresencaInstancia(instancia.Presenca),
		"presenca_observacao":   observacaoPresenca(instancia.Presenca),
		"configuracao_avancada": configuracaoAvancada(instancia),
	}, nil
}

func (s *InstanciaService) QRCode(ctx context.Context, id string) (map[string]interface{}, error) {
	instancia, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return nil, s.mapearErro(err)
	}
	info, err := s.gerenciador.Info(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter qrcode: %w", err)
	}
	status := instancia.Status
	if info.Status != "" {
		status = info.Status
	}
	return map[string]interface{}{"id": instancia.ID, "nome": instancia.Nome, "token": instancia.Token, "qrcode": info.QRCode, "pairing_code": info.PairingCode, "pairing_phone": info.PairingPhone, "status": status, "erro": info.UltimoErro, "atualizado_em": info.AtualizadoEm}, nil
}

func statusBloqueiaHistorico(status string) bool {
	switch status {
	case models.StatusInstanciaConectando, models.StatusInstanciaAguardandoQR, models.StatusInstanciaAguardandoCodigo, models.StatusInstanciaConectada, models.StatusInstanciaDesconectando, "pareada", "autenticando", "sincronizando_historico":
		return true
	default:
		return false
	}
}

func observacaoHistorico(dias int) string {
	if dias <= 0 {
		return "Historico inicial desativado. Defina a quantidade de dias antes de conectar a instancia."
	}
	return fmt.Sprintf("A API vai encaminhar historico recebido do WhatsApp dos ultimos %d dias.", dias)
}

func normalizarProxyInstancia(modo, proxyURL string) (string, string, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL != "" {
		if err := validarProxyURL(proxyURL); err != nil {
			return "", "", err
		}
		return models.ProxyModoProprio, proxyURL, nil
	}
	modo = modoProxyInstancia(modo)
	switch modo {
	case models.ProxyModoProprio:
		return "", "", ErrEntradaInvalida
	case models.ProxyModoHerdar:
		return models.ProxyModoHerdar, "", nil
	default:
		return "", "", ErrEntradaInvalida
	}
}

func modoProxyInstancia(modo string) string {
	switch strings.TrimSpace(strings.ToLower(modo)) {
	case models.ProxyModoProprio:
		return models.ProxyModoProprio
	default:
		return models.ProxyModoHerdar
	}
}

func validarProxyURL(proxyURL string) error {
	if proxyURL == "" {
		return ErrEntradaInvalida
	}
	u, err := neturl.Parse(proxyURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrEntradaInvalida
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks5":
		return nil
	default:
		return ErrEntradaInvalida
	}
}

func observacaoProxy(modo string) string {
	switch modoProxyInstancia(modo) {
	case models.ProxyModoProprio:
		return "Esta instancia usa proxy proprio e nao herda o proxy global do master."
	default:
		return "Esta instancia nao tem proxy proprio. Se o proxy global do master estiver ativo, ele sera aplicado automaticamente."
	}
}

func normalizarPresencaInstancia(presenca string) string {
	switch strings.ToLower(strings.TrimSpace(presenca)) {
	case "", models.PresencaDisponivel, "available", "online", "ativo", "ativa":
		return models.PresencaDisponivel
	case models.PresencaIndisponivel, "unavailable", "offline", "inativo", "inativa":
		return models.PresencaIndisponivel
	default:
		return ""
	}
}

func observacaoPresenca(presenca string) string {
	switch normalizarPresencaInstancia(presenca) {
	case models.PresencaIndisponivel:
		return "A instancia vai tentar manter a presenca global como indisponivel apos conectar."
	default:
		return "A instancia vai tentar manter a presenca global como disponivel apos conectar."
	}
}

func configuracaoAvancada(instancia models.Instancia) models.ConfiguracaoAvancadaInstancia {
	return models.ConfiguracaoAvancadaInstancia{
		ManterOnline:             normalizarPresencaInstancia(instancia.Presenca) == models.PresencaDisponivel,
		RejeitarChamadas:         instancia.RejeitarChamadas,
		MensagemRejeitarChamadas: instancia.MensagemRejeitarChamadas,
		MarcarLidaAutomatico:     instancia.MarcarLidaAutomatico,
		IgnorarGrupos:            instancia.IgnorarGrupos,
		IgnorarStatus:            instancia.IgnorarStatus,
	}
}

func (s *InstanciaService) mapearErro(err error) error {
	if errors.Is(err, store.ErrInstanciaNaoEncontrada) {
		return ErrInstanciaNaoEncontrada
	}
	return err
}
