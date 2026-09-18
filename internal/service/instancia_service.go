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
		fmt.Printf("erro ao listar instancias para restaurar sessoes: %v\n", err)
		return
	}
	for _, instancia := range instancias {
		if !deveRestaurarSessaoNoStartup(instancia.Status) {
			continue
		}
		instanciaID := instancia.ID
		historicoDias := instancia.HistoricoDias
		statusAnterior := instancia.Status
		go func() {
			s.restaurarSessaoComRetry(context.Background(), instanciaID, historicoDias, statusAnterior)
		}()
	}
}

func (s *InstanciaService) restaurarSessaoComRetry(ctx context.Context, instanciaID string, historicoDias int, statusAnterior string) {
	const (
		intervaloTentativa = 10 * time.Second
		janelaTentativas   = 4 * time.Minute
		timeoutTentativa   = 45 * time.Second
	)
	deadline := time.Now().Add(janelaTentativas)
	tentativa := 1
	for {
		s.gerenciador.ConfigurarHistorico(instanciaID, historicoDias)
		tentativaCtx, cancel := context.WithTimeout(ctx, timeoutTentativa)
		restaurada, err := s.gerenciador.RestaurarSessao(tentativaCtx, instanciaID)
		cancel()
		if err == nil {
			if restaurada {
				_, _ = s.store.AtualizarStatus(ctx, instanciaID, "sincronizando_historico")
				fmt.Printf("sessao restaurada automaticamente: instancia %s\n", instanciaID)
			} else {
				_, _ = s.store.AtualizarStatus(ctx, instanciaID, models.StatusInstanciaNaoInicializada)
				fmt.Printf("sessao nao restaurada: instancia %s nao possui dispositivo salvo\n", instanciaID)
			}
			return
		}

		if errors.Is(err, whatsapp.ErrInstanciaPertenceOutroNode) && time.Now().Before(deadline) {
			if tentativa == 1 || tentativa%6 == 0 {
				fmt.Printf("restauracao aguardando lock antigo: instancia %s tentativa %d erro: %v\n", instanciaID, tentativa, err)
			}
			tentativa++
			select {
			case <-ctx.Done():
				return
			case <-time.After(intervaloTentativa):
				continue
			}
		}

		// Quando a instancia pertence a outra replica, quem manda no status e o
		// container dono: ele esta com a sessao viva e atualizando o estado. Marcar
		// "desconectada" aqui sobrescreveria um status correto por um errado.
		if errors.Is(err, whatsapp.ErrInstanciaPertenceOutroNode) {
			fmt.Printf("restauracao encerrada: instancia %s pertence a outro container e sera gerida por ele\n", instanciaID)
			return
		}

		_, _ = s.store.AtualizarStatus(ctx, instanciaID, models.StatusInstanciaDesconectada)
		fmt.Printf("erro ao restaurar sessao da instancia %s status_anterior=%s: %v\n", instanciaID, statusAnterior, err)
		return
	}
}

// SupervisionarSessoes vigia, de tempos em tempos, as instancias que cairam e
// reconecta as que ainda tem sessao valida.
//
// O auto-reconnect do whatsmeow cobre a maior parte das quedas, mas desiste em
// varios casos em que a sessao continua boa: sessao assumida por outra conexao
// (stream replaced), falha de conexao nao reconhecida e erro de stream. Nesses
// casos a instancia ficava parada ate alguem clicar em conectar, e nesse meio
// tempo nenhuma mensagem chegava.
//
// Instancias que dependem de QR code, de codigo de pareamento ou que perderam o
// dispositivo nunca entram aqui: quem decide o que fazer com elas e o usuario.
func (s *InstanciaService) SupervisionarSessoes(ctx context.Context, intervalo time.Duration) {
	if intervalo < 10*time.Second {
		intervalo = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(intervalo)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reconectarSessoesCaidas(ctx)
			}
		}
	}()
}

func (s *InstanciaService) reconectarSessoesCaidas(ctx context.Context) {
	const timeoutTentativa = 45 * time.Second
	for _, instanciaID := range s.gerenciador.InstanciasReconectaveis() {
		// O estado salvo manda: se o usuario desconectou a instancia no banco, a API
		// nao pode trazer ela de volta sozinha.
		instancia, err := s.store.BuscarPorID(ctx, instanciaID)
		if err != nil {
			continue
		}
		if !deveRestaurarSessaoNoStartup(instancia.Status) {
			continue
		}
		tentativaCtx, cancel := context.WithTimeout(ctx, timeoutTentativa)
		restaurada, err := s.gerenciador.RestaurarSessao(tentativaCtx, instanciaID)
		cancel()
		switch {
		case err != nil && errors.Is(err, whatsapp.ErrInstanciaPertenceOutroNode):
			// Outra replica e dona da sessao e vai cuidar dela.
		case err != nil:
			fmt.Printf("reconexao automatica falhou: instancia %s: %v\n", instanciaID, err)
		case restaurada:
			_, _ = s.store.AtualizarStatus(ctx, instanciaID, "sincronizando_historico")
			fmt.Printf("reconexao automatica concluida: instancia %s\n", instanciaID)
		}
	}
}

func deveRestaurarSessaoNoStartup(status string) bool {
	switch status {
	case models.StatusInstanciaNaoInicializada, models.StatusInstanciaAguardandoQR, models.StatusInstanciaAguardandoCodigo:
		return false
	default:
		return true
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
		// Sem runtime local, este container nao e dono da instancia e o estado em
		// memoria aqui nao vale nada. Quem mantem o status correto e o container dono,
		// pelo banco. Sem essa guarda, replicas nao-donas sobrescreviam o status certo
		// e o dashboard piscava entre conectada e desconectada conforme o balanceador
		// alternava qual replica respondia a listagem.
		if !s.gerenciador.PossuiRuntime(instancias[i].ID) {
			continue
		}
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
	// Mesma regra da listagem: so o container dono da instancia pode corrigir o
	// status salvo.
	if s.gerenciador.PossuiRuntime(id) && info.Status != "" && info.Status != models.StatusInstanciaNaoInicializada {
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

func (s *InstanciaService) ConsultarAvatar(ctx context.Context, req models.ConsultaAvatarRequest) (models.AvatarContato, error) {
	req.Instancia = strings.TrimSpace(req.Instancia)
	req.Numero = strings.TrimSpace(req.Numero)
	req.ChatJID = strings.TrimSpace(req.ChatJID)
	if req.Instancia == "" || (req.Numero == "" && req.ChatJID == "") {
		return models.AvatarContato{}, ErrEntradaInvalida
	}
	if _, err := s.store.BuscarPorID(ctx, req.Instancia); err != nil {
		return models.AvatarContato{}, s.mapearErro(err)
	}
	avatar, err := s.gerenciador.ConsultarAvatar(ctx, req)
	if err != nil {
		if errors.Is(err, whatsapp.ErrAvatarNaoEncontrado) {
			return models.AvatarContato{}, ErrAvatarNaoEncontrado
		}
		return models.AvatarContato{}, err
	}
	return avatar, nil
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
