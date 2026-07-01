package whatsapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	nethttp "net/http"
	urlpkg "net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"dyalog-api-go/internal/models"
	mediastorage "dyalog-api-go/internal/storage"
	"dyalog-api-go/internal/store"
	webhookdispatch "dyalog-api-go/internal/webhook"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waCompanionReg "go.mau.fi/whatsmeow/proto/waCompanionReg"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	waStore "go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type Cliente interface {
	Conectar(ctx context.Context, instanciaID string) error
	SolicitarCodigoPareamento(ctx context.Context, instanciaID, numero string) (string, string, error)
	Desconectar(ctx context.Context, instanciaID string) error
	Status(ctx context.Context, instanciaID string) (string, error)
	QRCode(ctx context.Context, instanciaID string) (string, error)
	EnviarTexto(ctx context.Context, req models.EnvioTextoRequest) (models.ResultadoEnvio, error)
	EnviarPresenca(ctx context.Context, req models.EnvioPresencaRequest) (models.ResultadoPresenca, error)
	MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error)
	EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error)
	EnviarLista(ctx context.Context, req models.EnvioListaRequest) (models.ResultadoEnvio, error)
	EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
	EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
	EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
}

var ErrMidiaInvalida = errors.New("midia invalida")

const (
	metodoPareamentoQR     = "qr"
	metodoPareamentoCodigo = "codigo"
)

type estadoRuntime struct {
	status           string
	qrCode           string
	pairingCode      string
	pairingPhone     string
	metodoPareamento string
	ultimoErro       string
	atualizadoEm     time.Time
}

type InfoRuntime struct {
	Status           string
	QRCode           string
	PairingCode      string
	PairingPhone     string
	MetodoPareamento string
	UltimoErro       string
	AtualizadoEm     time.Time
}

type runtimeInstancia struct {
	client    *whatsmeow.Client
	container *sqlstore.Container
	qrCancel  context.CancelFunc
	chamadas  map[string]*chamadaAtiva
}

func (r *runtimeInstancia) cancelarFluxoQR() {
	if r.qrCancel != nil {
		r.qrCancel()
		r.qrCancel = nil
	}
}

type GerenciadorInstancias struct {
	mu              sync.RWMutex
	estados         map[string]estadoRuntime
	runtimes        map[string]*runtimeInstancia
	diretorioBase   string
	diretorioMidias string
	baseURL         string
	nomeDispositivo string
	tipoCliente     whatsmeow.PairClientType
	nomePareamento  string
	nivelLog        string
	dispatcher      *webhookdispatch.Dispatcher
	midiaStore      store.MidiaStore
	midiaUploader   mediastorage.MidiaUploader
	proxyStore      store.ProxyConfigStore
	mensagemStore   store.MensagemProcessadaStore
	aliasesNumero   map[string]string
	historicoDias   map[string]int
	recuperacoes    map[string]*janelaRecuperacao
}

type janelaRecuperacao struct {
	Inicio           time.Time
	Fim              time.Time
	Quantidade       int
	ChatsSolicitados map[string]bool
}

func NovoGerenciadorInstancias(diretorioBase, diretorioMidias, baseURL, nomeDispositivo, tipoCliente, nomePareamento, nivelLog string, dispatcher *webhookdispatch.Dispatcher, midiaStore store.MidiaStore, midiaUploader mediastorage.MidiaUploader, proxyStore store.ProxyConfigStore, mensagemStore store.MensagemProcessadaStore) *GerenciadorInstancias {
	nomeDispositivo = strings.TrimSpace(nomeDispositivo)
	if nomeDispositivo == "" {
		nomeDispositivo = "DyalogAPI"
	}
	nomePareamento = normalizarNomePareamento(nomePareamento)
	nivelLog = strings.ToUpper(strings.TrimSpace(nivelLog))
	if nivelLog == "" {
		nivelLog = "ERROR"
	}
	tipoClientePareamento := normalizarTipoClientePareamento(tipoCliente)
	configurarIdentidadeDispositivo(nomeDispositivo, tipoClientePareamento)
	return &GerenciadorInstancias{
		estados:         make(map[string]estadoRuntime),
		runtimes:        make(map[string]*runtimeInstancia),
		diretorioBase:   diretorioBase,
		diretorioMidias: diretorioMidias,
		baseURL:         strings.TrimRight(baseURL, "/"),
		nomeDispositivo: nomeDispositivo,
		tipoCliente:     tipoClientePareamento,
		nomePareamento:  nomePareamento,
		nivelLog:        nivelLog,
		dispatcher:      dispatcher,
		midiaStore:      midiaStore,
		midiaUploader:   midiaUploader,
		proxyStore:      proxyStore,
		mensagemStore:   mensagemStore,
		aliasesNumero:   make(map[string]string),
		historicoDias:   make(map[string]int),
		recuperacoes:    make(map[string]*janelaRecuperacao),
	}
}

func configurarIdentidadeDispositivo(nome string, tipoCliente whatsmeow.PairClientType) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		nome = "DyalogAPI"
	}

	waStore.SetOSInfo(nome, [3]uint32{1, 0, 0})
	waStore.DeviceProps.PlatformType = plataformaPorTipoCliente(tipoCliente).Enum()
	if waStore.BaseClientPayload != nil && waStore.BaseClientPayload.UserAgent != nil {
		waStore.BaseClientPayload.UserAgent.Device = proto.String(nome)
		waStore.BaseClientPayload.UserAgent.Manufacturer = proto.String("Dyalog")
	}
}

func plataformaPorTipoCliente(tipo whatsmeow.PairClientType) waCompanionReg.DeviceProps_PlatformType {
	switch tipo {
	case whatsmeow.PairClientEdge:
		return waCompanionReg.DeviceProps_EDGE
	case whatsmeow.PairClientFirefox:
		return waCompanionReg.DeviceProps_FIREFOX
	case whatsmeow.PairClientIE:
		return waCompanionReg.DeviceProps_IE
	case whatsmeow.PairClientOpera:
		return waCompanionReg.DeviceProps_OPERA
	case whatsmeow.PairClientSafari:
		return waCompanionReg.DeviceProps_SAFARI
	case whatsmeow.PairClientElectron, whatsmeow.PairClientMacOS:
		return waCompanionReg.DeviceProps_DESKTOP
	case whatsmeow.PairClientUWP:
		return waCompanionReg.DeviceProps_UWP
	case whatsmeow.PairClientAndroid:
		return waCompanionReg.DeviceProps_ANDROID_PHONE
	case whatsmeow.PairClientOtherWebClient:
		return waCompanionReg.DeviceProps_DESKTOP
	default:
		return waCompanionReg.DeviceProps_CHROME
	}
}

func normalizarNomePareamento(nome string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "Chrome (Windows)"
	}
	if strings.Contains(nome, "(") && strings.Contains(nome, ")") {
		return nome
	}
	return "Chrome (Windows)"
}

func normalizarTipoClientePareamento(tipo string) whatsmeow.PairClientType {
	switch strings.ToLower(strings.TrimSpace(tipo)) {
	case "edge":
		return whatsmeow.PairClientEdge
	case "firefox":
		return whatsmeow.PairClientFirefox
	case "ie", "internet-explorer":
		return whatsmeow.PairClientIE
	case "opera":
		return whatsmeow.PairClientOpera
	case "safari":
		return whatsmeow.PairClientSafari
	case "electron":
		return whatsmeow.PairClientElectron
	case "uwp", "windows", "windows-app":
		return whatsmeow.PairClientUWP
	case "macos", "mac":
		return whatsmeow.PairClientMacOS
	case "android":
		return whatsmeow.PairClientAndroid
	case "web":
		return whatsmeow.PairClientChrome
	case "other":
		return whatsmeow.PairClientOtherWebClient
	default:
		return whatsmeow.PairClientChrome
	}
}

func (g *GerenciadorInstancias) ConfigurarHistorico(instanciaID string, dias int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if dias <= 0 {
		delete(g.historicoDias, instanciaID)
		return
	}
	g.historicoDias[instanciaID] = dias
}

func (g *GerenciadorInstancias) obterHistoricoDias(instanciaID string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.historicoDias[instanciaID]
}

func (g *GerenciadorInstancias) RegistrarJanelaRecuperacao(instanciaID string, inicio, fim time.Time, quantidade int) {
	if instanciaID == "" || inicio.IsZero() || fim.IsZero() || !fim.After(inicio) {
		return
	}
	if quantidade <= 0 {
		quantidade = 50
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recuperacoes[instanciaID] = &janelaRecuperacao{
		Inicio:           inicio.UTC(),
		Fim:              fim.UTC(),
		Quantidade:       quantidade,
		ChatsSolicitados: make(map[string]bool),
	}
}

func (g *GerenciadorInstancias) janelaRecuperacao(instanciaID string) *janelaRecuperacao {
	g.mu.RLock()
	defer g.mu.RUnlock()
	janela := g.recuperacoes[instanciaID]
	if janela == nil {
		return nil
	}
	copia := *janela
	return &copia
}

func (g *GerenciadorInstancias) marcarChatRecuperacaoSolicitado(instanciaID, chat string) bool {
	if instanciaID == "" || chat == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	janela := g.recuperacoes[instanciaID]
	if janela == nil {
		return false
	}
	if janela.ChatsSolicitados[chat] {
		return false
	}
	janela.ChatsSolicitados[chat] = true
	return true
}

func (g *GerenciadorInstancias) EnviarPresencaGlobal(ctx context.Context, instanciaID, presenca string) error {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		return err
	}
	return g.aplicarPresencaGlobal(ctx, instanciaID, runtime, presenca)
}

func (g *GerenciadorInstancias) aplicarPresencaPersistente(ctx context.Context, instanciaID string, runtime *runtimeInstancia) error {
	presenca := models.PresencaIndisponivel
	if g.proxyStore != nil {
		if instancia, err := g.proxyStore.BuscarPorID(ctx, instanciaID); err == nil {
			presenca = instancia.Presenca
		}
	}
	return g.aplicarPresencaGlobal(ctx, instanciaID, runtime, presenca)
}

func (g *GerenciadorInstancias) aplicarPresencaGlobal(ctx context.Context, instanciaID string, runtime *runtimeInstancia, presenca string) error {
	if runtime == nil || runtime.client == nil || !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return nil
	}
	estado, err := estadoPresencaGlobal(presenca)
	if err != nil {
		return err
	}
	if err := runtime.client.SendPresence(ctx, estado); err != nil {
		g.definirEstado(instanciaID, g.obterEstado(instanciaID).status, "", err.Error())
		return err
	}
	return nil
}

func estadoPresencaGlobal(presenca string) (types.Presence, error) {
	switch strings.ToLower(strings.TrimSpace(presenca)) {
	case "", models.PresencaDisponivel, "available", "online", "ativo", "ativa":
		return types.PresenceAvailable, nil
	case models.PresencaIndisponivel, "unavailable", "offline", "inativo", "inativa":
		return types.PresenceUnavailable, nil
	default:
		return "", fmt.Errorf("presenca invalida: use disponivel ou indisponivel")
	}
}

func (g *GerenciadorInstancias) RestaurarSessao(ctx context.Context, instanciaID string) (bool, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return false, err
	}
	if runtime.client.Store.ID == nil {
		g.definirEstado(instanciaID, models.StatusInstanciaNaoInicializada, "", "")
		return false, nil
	}
	if runtime.client.IsConnected() && runtime.client.IsLoggedIn() {
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		_ = g.aplicarPresencaPersistente(ctx, instanciaID, runtime)
		return true, nil
	}
	g.definirEstado(instanciaID, models.StatusInstanciaConectando, "", "")
	if err := runtime.client.Connect(); err != nil {
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return false, fmt.Errorf("erro ao restaurar sessao da instancia: %w", err)
	}
	g.definirEstado(instanciaID, "sincronizando_historico", "", "")
	return true, nil
}

func (g *GerenciadorInstancias) Conectar(ctx context.Context, instanciaID string) (string, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return "", err
	}
	if runtime.client.IsConnected() && runtime.client.IsLoggedIn() {
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		_ = g.aplicarPresencaPersistente(ctx, instanciaID, runtime)
		return "", nil
	}
	estadoAtual := g.obterEstado(instanciaID)
	if estadoAtual.status == models.StatusInstanciaAguardandoQR && estadoAtual.qrCode != "" {
		return estadoAtual.qrCode, nil
	}
	novoPareamento := runtime.client.Store.ID == nil
	var qrChan <-chan whatsmeow.QRChannelItem
	if novoPareamento {
		runtime.cancelarFluxoQR()
		qrCtx, qrCancel := context.WithCancel(context.Background())
		runtime.qrCancel = qrCancel
		qrChan, err = runtime.client.GetQRChannel(qrCtx)
		if err != nil {
			runtime.cancelarFluxoQR()
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
			return "", fmt.Errorf("erro ao criar canal de qrcode: %w", err)
		}
		go g.consumirQRCode(instanciaID, runtime, qrChan)
	}
	g.definirEstado(instanciaID, models.StatusInstanciaConectando, "", "")
	if err := runtime.client.Connect(); err != nil {
		runtime.cancelarFluxoQR()
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return "", fmt.Errorf("erro ao conectar cliente whatsmeow: %w", err)
	}
	if !novoPareamento {
		g.definirEstado(instanciaID, "sincronizando_historico", "", "")
		return "", nil
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		estado := g.obterEstado(instanciaID)
		if estado.qrCode != "" || estado.status == models.StatusInstanciaConectada || estado.ultimoErro != "" {
			return estado.qrCode, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return g.obterEstado(instanciaID).qrCode, nil
}

func (g *GerenciadorInstancias) SolicitarCodigoPareamento(ctx context.Context, instanciaID, numero string) (string, string, error) {
	numero = apenasDigitos.ReplaceAllString(strings.TrimSpace(numero), "")
	if len(numero) <= 6 {
		return "", "", fmt.Errorf("informe o numero em formato internacional, somente com digitos")
	}
	if strings.HasPrefix(numero, "0") {
		return "", "", fmt.Errorf("o numero do pairing code deve estar em formato internacional, sem zero inicial")
	}

	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return "", numero, err
	}
	if runtime.client.Store != nil && runtime.client.Store.ID != nil {
		return "", numero, fmt.Errorf("a instancia ja possui sessao salva; desconecte a instancia antes de gerar um novo pairing code")
	}
	if runtime.client.IsConnected() && runtime.client.IsLoggedIn() {
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		return "", numero, fmt.Errorf("a instancia ja esta conectada ao WhatsApp")
	}

	estadoAtual := g.obterEstado(instanciaID)
	if estadoAtual.status == models.StatusInstanciaAguardandoCodigo && estadoAtual.pairingCode != "" {
		return estadoAtual.pairingCode, estadoAtual.pairingPhone, nil
	}

	if runtime.client.IsConnected() && estadoAtual.status == models.StatusInstanciaAguardandoQR {
		codigo, err := runtime.client.PairPhone(ctx, numero, false, g.tipoCliente, g.nomePareamento)
		if err != nil {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
			return "", numero, fmt.Errorf("erro ao solicitar pairing code: %w", err)
		}
		g.definirEstadoCodigoPareamento(instanciaID, models.StatusInstanciaAguardandoCodigo, codigo, numero, "")
		return codigo, numero, nil
	}

	if runtime.client.IsConnected() {
		runtime.cancelarFluxoQR()
		runtime.client.Disconnect()
	}

	runtime.cancelarFluxoQR()
	qrCtx, qrCancel := context.WithCancel(context.Background())
	runtime.qrCancel = qrCancel
	qrChan, err := runtime.client.GetQRChannel(qrCtx)
	if err != nil {
		runtime.cancelarFluxoQR()
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return "", numero, fmt.Errorf("erro ao preparar canal de pareamento: %w", err)
	}

	g.definirEstado(instanciaID, models.StatusInstanciaConectando, "", "")
	if err := runtime.client.Connect(); err != nil {
		runtime.cancelarFluxoQR()
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		return "", numero, fmt.Errorf("erro ao conectar cliente whatsmeow: %w", err)
	}

	espera := time.NewTimer(25 * time.Second)
	defer espera.Stop()
	for {
		select {
		case <-ctx.Done():
			runtime.cancelarFluxoQR()
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", ctx.Err().Error())
			return "", numero, ctx.Err()
		case <-espera.C:
			runtime.cancelarFluxoQR()
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "tempo esgotado para gerar pairing code")
			return "", numero, fmt.Errorf("tempo esgotado para gerar pairing code")
		case item, ok := <-qrChan:
			if !ok {
				runtime.cancelarFluxoQR()
				g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "canal de pareamento foi encerrado antes do pairing code")
				return "", numero, fmt.Errorf("canal de pareamento foi encerrado antes do pairing code")
			}
			if item.Event != whatsmeow.QRChannelEventCode {
				if err := g.tratarEventoPareamentoFalho(instanciaID, item); err != nil {
					return "", numero, err
				}
				continue
			}

			codigo, err := runtime.client.PairPhone(ctx, numero, false, g.tipoCliente, g.nomePareamento)
			if err != nil {
				runtime.cancelarFluxoQR()
				g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
				return "", numero, fmt.Errorf("erro ao solicitar pairing code: %w", err)
			}
			g.definirEstadoCodigoPareamento(instanciaID, models.StatusInstanciaAguardandoCodigo, codigo, numero, "")
			go g.consumirQRCode(instanciaID, runtime, qrChan)
			return codigo, numero, nil
		}
	}
}

func (g *GerenciadorInstancias) Desconectar(ctx context.Context, instanciaID string) error {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		return err
	}
	runtime.cancelarFluxoQR()
	if runtime.client.Store != nil && runtime.client.Store.ID != nil {
		if err := runtime.client.Logout(ctx); err != nil {
			if runtime.client.IsConnected() {
				runtime.client.Disconnect()
			}
			_ = runtime.client.Store.Delete(ctx)
		}
	} else if runtime.client.IsConnected() {
		runtime.client.Disconnect()
	}
	g.limparRuntime(instanciaID)
	g.definirEstado(instanciaID, models.StatusInstanciaNaoInicializada, "", "")
	return nil
}

func (g *GerenciadorInstancias) limparRuntime(instanciaID string) {
	g.mu.Lock()
	delete(g.runtimes, instanciaID)
	g.mu.Unlock()
}

func (g *GerenciadorInstancias) RecarregarInstancia(ctx context.Context, instanciaID string) {
	_ = ctx
	g.mu.Lock()
	runtime := g.runtimes[instanciaID]
	delete(g.runtimes, instanciaID)
	g.mu.Unlock()

	if runtime == nil {
		return
	}
	runtime.cancelarFluxoQR()
	deveRestaurar := runtime.client.Store != nil && runtime.client.Store.ID != nil
	if runtime.client.IsConnected() {
		runtime.client.Disconnect()
	}
	if deveRestaurar {
		go func() {
			_, _ = g.RestaurarSessao(context.Background(), instanciaID)
		}()
	}
}

func (g *GerenciadorInstancias) ExcluirInstancia(ctx context.Context, instanciaID string) error {
	g.mu.Lock()
	runtime := g.runtimes[instanciaID]
	delete(g.runtimes, instanciaID)
	delete(g.estados, instanciaID)
	delete(g.historicoDias, instanciaID)
	g.mu.Unlock()

	if runtime != nil {
		runtime.cancelarFluxoQR()
		if runtime.client.IsConnected() {
			runtime.client.Disconnect()
		}
		if runtime.client.Store != nil && runtime.client.Store.ID != nil {
			_ = runtime.client.Store.Delete(ctx)
		}
	}
	if err := os.RemoveAll(filepath.Join(g.diretorioBase, instanciaID)); err != nil {
		return fmt.Errorf("erro ao remover diretorio da instancia: %w", err)
	}
	return nil
}

func (g *GerenciadorInstancias) Status(ctx context.Context, instanciaID string) (string, error) {
	info, err := g.Info(ctx, instanciaID)
	if err != nil {
		return models.StatusInstanciaDesconectada, nil
	}
	if info.Status == "" {
		return models.StatusInstanciaNaoInicializada, nil
	}
	return info.Status, nil
}

func (g *GerenciadorInstancias) QRCode(ctx context.Context, instanciaID string) (string, error) {
	_, _ = g.obterOuCriarRuntime(ctx, instanciaID)
	return g.obterEstado(instanciaID).qrCode, nil
}

func (g *GerenciadorInstancias) Info(ctx context.Context, instanciaID string) (InfoRuntime, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		return InfoRuntime{}, err
	}
	estado := g.obterEstado(instanciaID)
	if runtime.client.IsConnected() && runtime.client.IsLoggedIn() && estado.status != models.StatusInstanciaConectada {
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		estado = g.obterEstado(instanciaID)
	} else if !runtime.client.IsConnected() && statusTransitorioPersistente(estado.status) {
		if runtime.client.Store != nil && runtime.client.Store.ID != nil {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", estado.ultimoErro)
		} else {
			g.definirEstado(instanciaID, models.StatusInstanciaNaoInicializada, "", estado.ultimoErro)
		}
		estado = g.obterEstado(instanciaID)
	}
	return InfoRuntime{
		Status:           estado.status,
		QRCode:           estado.qrCode,
		PairingCode:      estado.pairingCode,
		PairingPhone:     estado.pairingPhone,
		MetodoPareamento: estado.metodoPareamento,
		UltimoErro:       estado.ultimoErro,
		AtualizadoEm:     estado.atualizadoEm,
	}, nil
}

func statusTransitorioPersistente(status string) bool {
	switch status {
	case models.StatusInstanciaConectando, models.StatusInstanciaDesconectando, "pareada", "autenticando", "sincronizando_historico":
		return true
	default:
		return false
	}
}

func (g *GerenciadorInstancias) EnviarTexto(ctx context.Context, req models.EnvioTextoRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	delay := atrasoEnvio(req.DelaySegundos, req.Delay, req.DelayMS)
	presencaAntes := delay > 0 || (req.Digitando != nil && *req.Digitando)
	if presencaAntes {
		_ = runtime.client.SendPresence(ctx, types.PresenceAvailable)
		for _, jid := range jids {
			_ = runtime.client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)
		}
		defer pausarPresencaTexto(runtime.client, jids)
		if delay > 0 {
			if err := aguardarDelay(ctx, delay); err != nil {
				return models.ResultadoEnvio{}, err
			}
		}
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		msg := montarMensagemTexto(req, jid)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar mensagem no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	resultado := models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: "enviada", Tipo: "texto"}
	if presencaAntes {
		resultado.PresencaAntes = "digitando"
		resultado.DelaySegundos = segundosDelay(delay)
	}
	return resultado, nil
}

func (g *GerenciadorInstancias) EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	if usarFallbackTextoBotoes(req) {
		return g.enviarBotoesComoTexto(ctx, runtime.client, req, jids)
	}

	msg, modo, err := montarMensagemBotoes(req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		if erroInterativoNaoPermitido(ultimoErro) {
			resultadoFallback, err := g.enviarBotoesComoTexto(ctx, runtime.client, req, jids)
			if err == nil {
				resultadoFallback.Observacao = "Botoes interativos rejeitados pelo servidor do WhatsApp com erro 405; opcoes enviadas automaticamente como texto."
				return resultadoFallback, nil
			}
		}
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar botoes no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "aceita_pelo_servidor",
		Tipo:       "botoes",
		Modo:       modo,
		Observacao: "O WhatsApp retornou ID para a mensagem interativa, mas pode filtrar ou nao renderizar botoes em algumas contas/clientes.",
	}, nil
}

func (g *GerenciadorInstancias) enviarBotoesComoTexto(ctx context.Context, client *whatsmeow.Client, req models.EnvioBotoesRequest, jids []types.JID) (models.ResultadoEnvio, error) {
	msg := &waE2E.Message{Conversation: proto.String(textoFallbackBotoes(req))}
	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	for _, jid := range jids {
		resp, ultimoErro = client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar botoes como texto no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "enviada",
		Tipo:       "botoes",
		Modo:       "texto",
		Observacao: "Botoes enviados como texto para evitar filtro/renderizacao inconsistente de mensagens interativas.",
	}, nil
}

func (g *GerenciadorInstancias) EnviarLista(ctx context.Context, req models.EnvioListaRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	if usarFallbackTextoLista(req) {
		return g.enviarListaComoTexto(ctx, runtime.client, req, jids)
	}

	msg := montarMensagemLista(req)
	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		if erroInterativoNaoPermitido(ultimoErro) {
			resultadoFallback, err := g.enviarListaComoTexto(ctx, runtime.client, req, jids)
			if err == nil {
				resultadoFallback.Observacao = "Lista interativa rejeitada pelo servidor do WhatsApp com erro 405; opcoes enviadas automaticamente como texto."
				return resultadoFallback, nil
			}
		}
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar lista no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "aceita_pelo_servidor",
		Tipo:       "lista",
		Modo:       "lista",
		Observacao: "O WhatsApp retornou ID para a lista interativa, mas o cliente ainda pode filtrar ou nao renderizar a lista.",
	}, nil
}

func (g *GerenciadorInstancias) enviarListaComoTexto(ctx context.Context, client *whatsmeow.Client, req models.EnvioListaRequest, jids []types.JID) (models.ResultadoEnvio, error) {
	msg := &waE2E.Message{Conversation: proto.String(textoFallbackLista(req))}
	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	for _, jid := range jids {
		resp, ultimoErro = client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar lista como texto no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "enviada",
		Tipo:       "lista",
		Modo:       "texto",
		Observacao: "Lista enviada como texto para garantir entrega quando a renderizacao interativa nao estiver disponivel.",
	}, nil
}

func (g *GerenciadorInstancias) EnviarPresenca(ctx context.Context, req models.EnvioPresencaRequest) (models.ResultadoPresenca, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoPresenca{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoPresenca{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	acao, estado, media, presencaGlobal, estadoGlobal, err := mapearAcaoPresencaEnvio(req.Acao)
	if err != nil {
		return models.ResultadoPresenca{}, err
	}

	if presencaGlobal {
		_ = estadoGlobal
		if err := g.aplicarPresencaGlobal(ctx, req.Instancia, runtime, acao); err != nil {
			return models.ResultadoPresenca{}, fmt.Errorf("erro ao enviar presenca global no WhatsApp: %w", err)
		}
		return models.ResultadoPresenca{Instancia: req.Instancia, Numero: req.Numero, ChatJID: strings.TrimSpace(req.ChatJID), Status: "enviada", Tipo: "presenca", Acao: acao}, nil
	}

	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoPresenca{}, err
	}
	if len(jids) == 0 {
		return models.ResultadoPresenca{}, fmt.Errorf("nenhum destino encontrado para presenca")
	}

	_ = runtime.client.SendPresence(ctx, types.PresenceAvailable)

	var (
		ultimoErro error
		jidUsado   types.JID
	)
	for _, jid := range jids {
		ultimoErro = runtime.client.SendChatPresence(ctx, jid, estado, media)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		if ultimoErro == nil {
			ultimoErro = errors.New("nenhum destino aceitou a presenca")
		}
		return models.ResultadoPresenca{}, fmt.Errorf("erro ao enviar presenca no WhatsApp: %w", ultimoErro)
	}

	delay := atrasoEnvio(req.DelaySegundos, req.Delay, req.DelayMS)
	finalizadaComPausado := false
	if delay > 0 && (acao == "digitando" || acao == "gravando_audio") {
		if err := aguardarDelay(ctx, delay); err != nil {
			return models.ResultadoPresenca{}, err
		}
		if err := runtime.client.SendChatPresence(ctx, jidUsado, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
			return models.ResultadoPresenca{}, fmt.Errorf("erro ao finalizar presenca no WhatsApp: %w", err)
		}
		finalizadaComPausado = true
	}
	return models.ResultadoPresenca{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), Status: "enviada", Tipo: "presenca", Acao: acao, DelaySegundos: segundosDelay(delay), FinalizadaComPausado: finalizadaComPausado}, nil
}

func atrasoEnvio(delaySegundos, delay, delayMS int) time.Duration {
	if delayMS > 0 {
		return limitarDelay(time.Duration(delayMS) * time.Millisecond)
	}
	segundos := delaySegundos
	if segundos <= 0 {
		segundos = delay
	}
	if segundos <= 0 {
		return 0
	}
	return limitarDelay(time.Duration(segundos) * time.Second)
}

func limitarDelay(delay time.Duration) time.Duration {
	const maxDelay = 60 * time.Second
	if delay < 0 {
		return 0
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func segundosDelay(delay time.Duration) int {
	if delay <= 0 {
		return 0
	}
	return int((delay + time.Second - 1) / time.Second)
}

func aguardarDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func pausarPresencaTexto(client *whatsmeow.Client, jids []types.JID) func() {
	return func() {
		for _, jid := range jids {
			_ = client.SendChatPresence(context.Background(), jid, types.ChatPresencePaused, types.ChatPresenceMediaText)
		}
	}
}

func (g *GerenciadorInstancias) MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoMarcarLida{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoMarcarLida{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoMarcarLida{}, err
	}
	if len(jids) == 0 {
		return models.ResultadoMarcarLida{}, fmt.Errorf("nenhum destino encontrado para marcar leitura")
	}

	chat := jids[0]
	sender := types.EmptyJID
	if req.Grupo {
		sender, err = types.ParseJID(strings.TrimSpace(req.Participante))
		if err != nil {
			return models.ResultadoMarcarLida{}, fmt.Errorf("participante invalido")
		}
	}

	ids := make([]types.MessageID, 0, len(req.MensagensID))
	for _, id := range req.MensagensID {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, types.MessageID(id))
		}
	}
	if len(ids) == 0 {
		return models.ResultadoMarcarLida{}, fmt.Errorf("nenhuma mensagem valida informada para marcar leitura")
	}

	if err := runtime.client.MarkRead(ctx, ids, req.MarcadaEmTime, chat, sender); err != nil {
		return models.ResultadoMarcarLida{}, fmt.Errorf("erro ao marcar mensagem como lida no WhatsApp: %w", err)
	}

	return models.ResultadoMarcarLida{
		Instancia:    req.Instancia,
		Numero:       req.Numero,
		ChatJID:      chat.String(),
		MensagensID:  req.MensagensID,
		Participante: strings.TrimSpace(req.Participante),
		Status:       "marcada_como_lida",
		Tipo:         "leitura",
		LidaEm:       req.MarcadaEmTime.UTC(),
	}, nil
}

func mapearAcaoPresencaEnvio(acao string) (string, types.ChatPresence, types.ChatPresenceMedia, bool, types.Presence, error) {
	switch strings.ToLower(strings.TrimSpace(acao)) {
	case "digitando", "composing":
		return "digitando", types.ChatPresenceComposing, types.ChatPresenceMediaText, false, "", nil
	case "gravando_audio", "gravando", "audio":
		return "gravando_audio", types.ChatPresenceComposing, types.ChatPresenceMediaAudio, false, "", nil
	case "pausado", "parou", "paused":
		return "pausado", types.ChatPresencePaused, types.ChatPresenceMediaText, false, "", nil
	case "disponivel", "online", "available":
		return "disponivel", types.ChatPresencePaused, types.ChatPresenceMediaText, true, types.PresenceAvailable, nil
	case "indisponivel", "offline", "unavailable":
		return "indisponivel", types.ChatPresencePaused, types.ChatPresenceMediaText, true, types.PresenceUnavailable, nil
	default:
		return "", "", "", false, "", fmt.Errorf("acao de presenca invalida: use digitando, gravando_audio, pausado, disponivel ou indisponivel")
	}
}

func (g *GerenciadorInstancias) EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	dados, mimeType, err := carregarImagemEnvio(ctx, req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	largura, altura := dimensoesImagem(dados)
	upload, err := runtime.client.Upload(ctx, dados, whatsmeow.MediaImage)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao fazer upload da imagem: %w", err)
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		msg := montarMensagemImagem(req, mimeType, upload, largura, altura)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar imagem no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: "enviada", Tipo: "imagem"}, nil
}

func (g *GerenciadorInstancias) EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	dados, mimeType, err := carregarAudioEnvio(ctx, req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	reqEnvio := req
	if reqEnvio.DuracaoSegundos == 0 {
		reqEnvio.DuracaoSegundos = resolverDuracaoAudio(reqEnvio, mimeType, dados)
	}
	upload, err := runtime.client.Upload(ctx, dados, whatsmeow.MediaAudio)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao fazer upload do audio: %w", err)
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		msg := montarMensagemAudio(reqEnvio, mimeType, upload)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar audio no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: "enviada", Tipo: "audio"}, nil
}

func (g *GerenciadorInstancias) EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	dados, nomeArquivo, mimeType, err := carregarDocumentoEnvio(ctx, req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	upload, err := runtime.client.Upload(ctx, dados, whatsmeow.MediaDocument)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao fazer upload do documento: %w", err)
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)

	for _, jid := range jids {
		msg := montarMensagemDocumento(req, nomeArquivo, mimeType, upload)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar documento no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: "enviada", Tipo: "documento"}, nil
}

func carregarImagemEnvio(ctx context.Context, req models.EnvioMidiaRequest) ([]byte, string, error) {
	dados, nomeArquivo, mimeCabecalho, err := carregarConteudoMidia(ctx, req)
	if err != nil {
		return nil, "", err
	}
	mimeType := detectarMimeType(nomeArquivo, mimeCabecalho, dados)
	if !strings.HasPrefix(mimeType, "image/") || mimeType == "image/svg+xml" {
		return nil, "", fmt.Errorf("%w: arquivo informado nao e uma imagem suportada", ErrMidiaInvalida)
	}
	return dados, mimeType, nil
}

func carregarDocumentoEnvio(ctx context.Context, req models.EnvioMidiaRequest) ([]byte, string, string, error) {
	dados, nomeArquivo, mimeCabecalho, err := carregarConteudoMidia(ctx, req)
	if err != nil {
		return nil, "", "", err
	}
	mimeType := detectarMimeType(nomeArquivo, mimeCabecalho, dados)
	if nomeArquivo == "" {
		nomeArquivo = nomeArquivoBaseadoNoMime(mimeType)
	}
	return dados, nomeArquivo, mimeType, nil
}

func carregarAudioEnvio(ctx context.Context, req models.EnvioMidiaRequest) ([]byte, string, error) {
	dados, nomeArquivo, mimeCabecalho, err := carregarConteudoMidia(ctx, req)
	if err != nil {
		return nil, "", err
	}
	mimeBase := detectarMimeType(nomeArquivo, primeiroValorNaoVazio(req.MimeType, mimeCabecalho), dados)
	mimeType, err := normalizarMimeAudio(mimeBase, nomeArquivo)
	if err != nil {
		return nil, "", err
	}
	if req.PTT != nil && *req.PTT && mimeType != "audio/ogg; codecs=opus" {
		return nil, "", fmt.Errorf("%w: ptt exige audio ogg/opus", ErrMidiaInvalida)
	}
	return dados, mimeType, nil
}

func normalizarMimeAudio(mimeType, nomeArquivo string) (string, error) {
	mimeType = limparMimeType(mimeType)
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(nomeArquivo)))
	switch {
	case mimeType == "audio/ogg" || mimeType == "application/ogg" || mimeType == "audio/opus" || ext == ".ogg" || ext == ".opus":
		return "audio/ogg; codecs=opus", nil
	case mimeType == "audio/mpeg" || ext == ".mp3":
		return "audio/mpeg", nil
	case mimeType == "audio/mp4" || mimeType == "audio/x-m4a" || ext == ".m4a" || ext == ".mp4":
		return "audio/mp4", nil
	case mimeType == "audio/aac" || ext == ".aac":
		return "audio/aac", nil
	case mimeType == "audio/amr" || ext == ".amr":
		return "audio/amr", nil
	case mimeType == "audio/wav" || mimeType == "audio/x-wav" || ext == ".wav":
		return "audio/wav", nil
	case strings.HasPrefix(mimeType, "audio/"):
		return mimeType, nil
	case mimeType == "application/octet-stream" && ext != "":
		return normalizarMimeAudio(extensaoParaMimeAudio(ext), nomeArquivo)
	default:
		return "", fmt.Errorf("%w: nao foi possivel identificar o formato do audio; informe mime_type ou nome_arquivo", ErrMidiaInvalida)
	}
}

func extensaoParaMimeAudio(ext string) string {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".ogg", ".opus":
		return "audio/ogg"
	case ".mp3":
		return "audio/mpeg"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".amr":
		return "audio/amr"
	case ".wav":
		return "audio/wav"
	default:
		return ""
	}
}

func primeiroValorNaoVazio(valores ...string) string {
	for _, valor := range valores {
		if strings.TrimSpace(valor) != "" {
			return valor
		}
	}
	return ""
}

func carregarConteudoMidia(ctx context.Context, req models.EnvioMidiaRequest) ([]byte, string, string, error) {
	if caminho := strings.TrimSpace(req.CaminhoLocal); caminho != "" {
		dados, err := os.ReadFile(caminho)
		if err != nil {
			return nil, "", "", fmt.Errorf("%w: caminho_local invalido ou inacessivel", ErrMidiaInvalida)
		}
		if len(dados) == 0 {
			return nil, "", "", fmt.Errorf("%w: arquivo local vazio", ErrMidiaInvalida)
		}
		nomeArquivo := strings.TrimSpace(req.NomeArquivo)
		if nomeArquivo == "" {
			nomeArquivo = filepath.Base(caminho)
		}
		return dados, nomeArquivo, "", nil
	}

	if arquivoBase64 := strings.TrimSpace(req.ArquivoBase64); arquivoBase64 != "" {
		return decodificarMidiaBase64(arquivoBase64, req.NomeArquivo)
	}

	arquivoURL := strings.TrimSpace(req.ArquivoURL)
	if arquivoURL == "" {
		return nil, "", "", fmt.Errorf("%w: informe arquivo_url, arquivo_base64 ou caminho_local", ErrMidiaInvalida)
	}
	parsedURL, err := urlpkg.Parse(arquivoURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, "", "", fmt.Errorf("%w: arquivo_url invalida", ErrMidiaInvalida)
	}
	request, err := nethttp.NewRequestWithContext(ctx, nethttp.MethodGet, arquivoURL, nil)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: arquivo_url invalida", ErrMidiaInvalida)
	}
	cliente := &nethttp.Client{Timeout: 45 * time.Second}
	response, err := cliente.Do(request)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: nao foi possivel baixar arquivo_url", ErrMidiaInvalida)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", "", fmt.Errorf("%w: arquivo_url retornou status %d", ErrMidiaInvalida, response.StatusCode)
	}
	dados, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: erro ao ler arquivo_url", ErrMidiaInvalida)
	}
	if len(dados) == 0 {
		return nil, "", "", fmt.Errorf("%w: arquivo_url retornou conteudo vazio", ErrMidiaInvalida)
	}
	nomeArquivo := strings.TrimSpace(req.NomeArquivo)
	if nomeArquivo == "" {
		nomeArquivo = nomeArquivoDaURL(parsedURL)
	}
	return dados, nomeArquivo, response.Header.Get("Content-Type"), nil
}

func decodificarMidiaBase64(valor, nomeArquivo string) ([]byte, string, string, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return nil, "", "", fmt.Errorf("%w: arquivo_base64 vazio", ErrMidiaInvalida)
	}

	mimeCabecalho := ""
	valorNormalizado := valor
	valorLower := strings.ToLower(valor)
	if strings.HasPrefix(valorLower, "data:") {
		separador := strings.Index(valor, ",")
		if separador < 0 {
			return nil, "", "", fmt.Errorf("%w: arquivo_base64 em data URI invalido", ErrMidiaInvalida)
		}
		cabecalho := valor[5:separador]
		if !strings.Contains(strings.ToLower(cabecalho), ";base64") {
			return nil, "", "", fmt.Errorf("%w: data URI precisa estar em base64", ErrMidiaInvalida)
		}
		mimeCabecalho = cabecalho
		if idx := strings.Index(strings.ToLower(mimeCabecalho), ";base64"); idx >= 0 {
			mimeCabecalho = mimeCabecalho[:idx]
		}
		valorNormalizado = valor[separador+1:]
	}

	valorNormalizado = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t', ' ':
			return -1
		default:
			return r
		}
	}, valorNormalizado)
	if valorNormalizado == "" {
		return nil, "", "", fmt.Errorf("%w: arquivo_base64 vazio", ErrMidiaInvalida)
	}

	var dados []byte
	var err error
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		dados, err = encoding.DecodeString(valorNormalizado)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("%w: arquivo_base64 invalido", ErrMidiaInvalida)
	}
	if len(dados) == 0 {
		return nil, "", "", fmt.Errorf("%w: arquivo_base64 sem conteudo", ErrMidiaInvalida)
	}

	nomeArquivo = strings.TrimSpace(nomeArquivo)
	if nomeArquivo == "" {
		nomeArquivo = nomeArquivoBaseadoNoMime(limparMimeType(mimeCabecalho))
	}
	return dados, nomeArquivo, mimeCabecalho, nil
}

func nomeArquivoBaseadoNoMime(mimeType string) string {
	mimeType = limparMimeType(mimeType)
	if mimeType == "" {
		return "arquivo"
	}
	extensoes, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(extensoes) > 0 {
		return "arquivo" + extensoes[0]
	}
	return "arquivo"
}

func detectarMimeType(nomeArquivo, mimeCabecalho string, dados []byte) string {
	if mimeType := limparMimeType(mimeCabecalho); mimeType != "" {
		return mimeType
	}
	if nomeArquivo != "" {
		if mimeType := limparMimeType(mime.TypeByExtension(strings.ToLower(filepath.Ext(nomeArquivo)))); mimeType != "" {
			return mimeType
		}
	}
	if len(dados) == 0 {
		return "application/octet-stream"
	}
	amostra := dados
	if len(amostra) > 512 {
		amostra = amostra[:512]
	}
	return limparMimeType(nethttp.DetectContentType(amostra))
}

func limparMimeType(valor string) string {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return ""
	}
	if idx := strings.Index(valor, ";"); idx >= 0 {
		valor = valor[:idx]
	}
	return strings.TrimSpace(strings.ToLower(valor))
}

func nomeArquivoDaURL(parsedURL *urlpkg.URL) string {
	if parsedURL == nil {
		return "imagem"
	}
	nomeArquivo := path.Base(parsedURL.Path)
	if nomeArquivo == "." || nomeArquivo == "/" || nomeArquivo == "" {
		return "imagem"
	}
	return nomeArquivo
}

func dimensoesImagem(dados []byte) (uint32, uint32) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(dados))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return 0, 0
	}
	return uint32(cfg.Width), uint32(cfg.Height)
}

func montarMensagemImagem(req models.EnvioMidiaRequest, mimeType string, upload whatsmeow.UploadResponse, largura, altura uint32) *waE2E.Message {
	imageMsg := &waE2E.ImageMessage{
		Mimetype:      proto.String(mimeType),
		URL:           &upload.URL,
		DirectPath:    &upload.DirectPath,
		MediaKey:      upload.MediaKey,
		FileEncSHA256: upload.FileEncSHA256,
		FileSHA256:    upload.FileSHA256,
		FileLength:    &upload.FileLength,
	}
	if legenda := strings.TrimSpace(req.Legenda); legenda != "" {
		imageMsg.Caption = proto.String(legenda)
	}
	if largura > 0 {
		imageMsg.Width = &largura
	}
	if altura > 0 {
		imageMsg.Height = &altura
	}
	return &waE2E.Message{ImageMessage: imageMsg}
}

func montarMensagemDocumento(req models.EnvioMidiaRequest, nomeArquivo, mimeType string, upload whatsmeow.UploadResponse) *waE2E.Message {
	docMsg := &waE2E.DocumentMessage{
		Mimetype:      proto.String(mimeType),
		URL:           &upload.URL,
		DirectPath:    &upload.DirectPath,
		MediaKey:      upload.MediaKey,
		FileEncSHA256: upload.FileEncSHA256,
		FileSHA256:    upload.FileSHA256,
		FileLength:    &upload.FileLength,
	}
	nomeArquivo = strings.TrimSpace(nomeArquivo)
	if nomeArquivo != "" {
		docMsg.FileName = proto.String(nomeArquivo)
		titulo := strings.TrimSpace(strings.TrimSuffix(nomeArquivo, filepath.Ext(nomeArquivo)))
		if titulo == "" {
			titulo = nomeArquivo
		}
		docMsg.Title = proto.String(titulo)
	}
	if legenda := strings.TrimSpace(req.Legenda); legenda != "" {
		docMsg.Caption = proto.String(legenda)
	}
	return &waE2E.Message{DocumentMessage: docMsg}
}

func montarMensagemAudio(req models.EnvioMidiaRequest, mimeType string, upload whatsmeow.UploadResponse) *waE2E.Message {
	ptt := resolverPTTAudio(req, mimeType)
	mediaKeyTimestamp := time.Now().Unix()
	audioMsg := &waE2E.AudioMessage{
		Mimetype:          proto.String(mimeType),
		URL:               &upload.URL,
		DirectPath:        &upload.DirectPath,
		MediaKey:          upload.MediaKey,
		FileEncSHA256:     upload.FileEncSHA256,
		FileSHA256:        upload.FileSHA256,
		FileLength:        &upload.FileLength,
		PTT:               &ptt,
		MediaKeyTimestamp: &mediaKeyTimestamp,
	}
	if req.DuracaoSegundos > 0 {
		duracao := req.DuracaoSegundos
		audioMsg.Seconds = &duracao
	}
	return &waE2E.Message{AudioMessage: audioMsg}
}

func resolverPTTAudio(req models.EnvioMidiaRequest, mimeType string) bool {
	if req.PTT != nil {
		return *req.PTT
	}
	return mimeType == "audio/ogg; codecs=opus"
}

func (g *GerenciadorInstancias) obterOuCriarRuntime(ctx context.Context, instanciaID string) (*runtimeInstancia, error) {
	g.mu.RLock()
	runtime, ok := g.runtimes[instanciaID]
	g.mu.RUnlock()
	if ok {
		return runtime, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if runtime, ok = g.runtimes[instanciaID]; ok {
		return runtime, nil
	}
	dirInstancia := filepath.Join(g.diretorioBase, instanciaID)
	if err := os.MkdirAll(dirInstancia, 0o755); err != nil {
		return nil, fmt.Errorf("erro ao criar diretorio da instancia: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on", filepath.ToSlash(filepath.Join(dirInstancia, "whatsmeow.db")))
	container, err := sqlstore.New(context.Background(), "sqlite3", dsn, waLog.Stdout("WA-DB-"+instanciaID, g.nivelLog, false))
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir store do whatsmeow: %w", err)
	}
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("erro ao obter device store: %w", err)
	}
	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("WA-"+instanciaID, g.nivelLog, false))
	client.EnableAutoReconnect = true
	client.QRClientType = g.tipoCliente
	if err := g.aplicarProxyCliente(ctx, instanciaID, client); err != nil {
		return nil, err
	}
	runtime = &runtimeInstancia{client: client, container: container, chamadas: make(map[string]*chamadaAtiva)}
	client.AddEventHandler(func(evt interface{}) { g.tratarEvento(instanciaID, runtime, evt) })
	g.runtimes[instanciaID] = runtime
	if client.Store.ID != nil {
		g.estados[instanciaID] = estadoRuntime{status: models.StatusInstanciaDesconectada, atualizadoEm: time.Now().UTC()}
	} else {
		g.estados[instanciaID] = estadoRuntime{status: models.StatusInstanciaNaoInicializada, atualizadoEm: time.Now().UTC()}
	}
	return runtime, nil
}

func (g *GerenciadorInstancias) aplicarProxyCliente(ctx context.Context, instanciaID string, client *whatsmeow.Client) error {
	proxyURL, err := g.resolverProxyURL(ctx, instanciaID)
	if err != nil {
		return err
	}
	if err := client.SetProxyAddress(proxyURL); err != nil {
		return fmt.Errorf("proxy configurado para a instancia e invalido: %w", err)
	}
	return nil
}

func (g *GerenciadorInstancias) resolverProxyURL(ctx context.Context, instanciaID string) (string, error) {
	if g.proxyStore == nil {
		return "", nil
	}
	instancia, err := g.proxyStore.BuscarPorID(ctx, instanciaID)
	if err != nil {
		return "", err
	}
	switch instancia.ProxyModo {
	case models.ProxyModoProprio:
		return strings.TrimSpace(instancia.ProxyURL), nil
	}
	global, err := g.proxyStore.ObterProxyGlobal(ctx)
	if err != nil {
		return "", err
	}
	if global.Ativo {
		return strings.TrimSpace(global.URL), nil
	}
	return "", nil
}

func (g *GerenciadorInstancias) tratarEvento(instanciaID string, runtime *runtimeInstancia, evt interface{}) {
	if g.tratarEventoChamada(instanciaID, runtime, evt) {
		return
	}
	switch evento := evt.(type) {
	case *events.Connected:
		runtime.cancelarFluxoQR()
		go func() {
			_ = g.aplicarPresencaPersistente(context.Background(), instanciaID, runtime)
		}()
		estado := g.obterEstado(instanciaID)
		if estado.status == models.StatusInstanciaAguardandoCodigo && estado.pairingCode != "" {
			g.definirEstadoCodigoPareamento(instanciaID, models.StatusInstanciaAguardandoCodigo, estado.pairingCode, estado.pairingPhone, "")
		} else if estado.status == "pareada" || estado.status == "autenticando" || estado.status == "sincronizando_historico" || estado.status == models.StatusInstanciaConectando {
			g.definirEstado(instanciaID, "sincronizando_historico", "", "")
		} else {
			g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		}
	case *events.Disconnected:
		estado := g.obterEstado(instanciaID)
		if runtime.client.IsLoggedIn() {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "")
		} else if estado.status == models.StatusInstanciaAguardandoCodigo && estado.pairingCode != "" {
			g.definirEstadoCodigoPareamento(instanciaID, models.StatusInstanciaAguardandoCodigo, estado.pairingCode, estado.pairingPhone, "")
		} else {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, estado.qrCode, "")
		}
	case *events.LoggedOut:
		runtime.cancelarFluxoQR()
		_ = runtime.client.Store.Delete(context.Background())
		g.definirEstado(instanciaID, models.StatusInstanciaNaoInicializada, "", "dispositivo desconectado externamente")
	case *events.PairSuccess:
		g.definirEstado(instanciaID, "pareada", "", "")
	case *events.PairError:
		runtime.cancelarFluxoQR()
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", evento.Error.Error())
	case *events.HistorySync:
		go func() {
			g.processarHistorico(instanciaID, runtime.client, evento)
			_ = g.aplicarPresencaPersistente(context.Background(), instanciaID, runtime)
		}()
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
	case *events.Message:
		configuracao := g.obterConfiguracaoInstancia(context.Background(), instanciaID)
		if g.deveIgnorarMensagem(configuracao, evento) {
			return
		}
		g.solicitarHistoricoRecuperacao(instanciaID, runtime.client, evento)
		if configuracao.MarcarLidaAutomatico && !evento.Info.IsFromMe {
			go g.marcarMensagemRecebidaComoLida(instanciaID, runtime.client, evento)
		}
		go g.dispararEventoMensagem(instanciaID, runtime.client, evento, "tempo_real", false)
		g.prepararAcompanhamentoPresenca(instanciaID, runtime.client, evento)
	case *events.ChatPresence:
		g.dispararEventoPresenca(instanciaID, evento)
	case *events.Receipt:
		configuracao := g.obterConfiguracaoInstancia(context.Background(), instanciaID)
		if g.deveIgnorarRecibo(configuracao, evento) {
			return
		}
		go g.dispararEventoRecibo(instanciaID, evento)
	}
}

func (g *GerenciadorInstancias) obterConfiguracaoInstancia(ctx context.Context, instanciaID string) models.Instancia {
	instancia := models.Instancia{Presenca: models.PresencaIndisponivel}
	if g.proxyStore == nil {
		return instancia
	}
	salva, err := g.proxyStore.BuscarPorID(ctx, instanciaID)
	if err != nil {
		return instancia
	}
	return salva
}

func (g *GerenciadorInstancias) deveIgnorarMensagem(instancia models.Instancia, evento *events.Message) bool {
	if evento == nil {
		return true
	}
	if mensagemEhStatusOuCanal(evento) {
		return true
	}
	if instancia.IgnorarGrupos && evento.Info.IsGroup {
		return true
	}
	return false
}

func (g *GerenciadorInstancias) deveIgnorarRecibo(instancia models.Instancia, evento *events.Receipt) bool {
	if evento == nil {
		return true
	}
	if reciboEhStatusOuCanal(evento) {
		return true
	}
	return instancia.IgnorarGrupos && evento.IsGroup
}

func mensagemEhStatusOuCanal(evento *events.Message) bool {
	if evento == nil {
		return false
	}
	return evento.NewsletterMeta != nil ||
		jidEhStatusOuCanal(evento.Info.Chat) ||
		jidEhStatusOuCanal(evento.Info.Sender) ||
		jidEhStatusOuCanal(evento.Info.RecipientAlt) ||
		jidEhStatusOuCanal(evento.Info.SenderAlt)
}

func reciboEhStatusOuCanal(evento *events.Receipt) bool {
	if evento == nil {
		return false
	}
	return jidEhStatusOuCanal(evento.Chat) ||
		jidEhStatusOuCanal(evento.Sender) ||
		jidEhStatusOuCanal(evento.RecipientAlt) ||
		jidEhStatusOuCanal(evento.SenderAlt) ||
		jidEhStatusOuCanal(evento.MessageSender)
}

func jidEhStatusOuCanal(jid types.JID) bool {
	return jid == types.StatusBroadcastJID ||
		jid.Server == types.BroadcastServer ||
		jid.Server == types.NewsletterServer
}

func (g *GerenciadorInstancias) marcarMensagemRecebidaComoLida(instanciaID string, client *whatsmeow.Client, evento *events.Message) {
	if client == nil || evento == nil || evento.Info.ID == "" {
		return
	}
	sender := types.EmptyJID
	if evento.Info.IsGroup {
		sender = evento.Info.Sender
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.MarkRead(ctx, []types.MessageID{evento.Info.ID}, time.Now(), evento.Info.Chat, sender); err != nil {
		fmt.Printf("erro ao marcar mensagem como lida na instancia %s: %v\n", instanciaID, err)
	}
}

func (g *GerenciadorInstancias) rejeitarChamadaSeConfigurado(instanciaID string, client *whatsmeow.Client, meta types.BasicCallMeta) {
	if client == nil || strings.TrimSpace(meta.CallID) == "" {
		return
	}
	instancia := g.obterConfiguracaoInstancia(context.Background(), instanciaID)
	if !instancia.RejeitarChamadas {
		return
	}
	destino := meta.CallCreator
	if destino.IsEmpty() {
		destino = meta.From
	}
	if destino.IsEmpty() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.RejectCall(ctx, destino, meta.CallID); err != nil {
		fmt.Printf("erro ao rejeitar chamada na instancia %s: %v\n", instanciaID, err)
	}
	mensagem := strings.TrimSpace(instancia.MensagemRejeitarChamadas)
	if mensagem == "" {
		return
	}
	_, err := g.EnviarTexto(context.Background(), models.EnvioTextoRequest{
		Instancia: instanciaID,
		ChatJID:   destino.String(),
		Mensagem:  mensagem,
	})
	if err != nil {
		fmt.Printf("erro ao enviar mensagem de rejeicao de chamada na instancia %s: %v\n", instanciaID, err)
	}
}

func (g *GerenciadorInstancias) solicitarHistoricoRecuperacao(instanciaID string, client *whatsmeow.Client, evento *events.Message) {
	if client == nil || evento == nil || evento.Info.ID == "" {
		return
	}
	janela := g.janelaRecuperacao(instanciaID)
	if janela == nil {
		return
	}
	if evento.Info.Timestamp.IsZero() || evento.Info.Timestamp.UTC().Before(janela.Fim) {
		return
	}
	chat := evento.Info.Chat.String()
	if !g.marcarChatRecuperacaoSolicitado(instanciaID, chat) {
		return
	}
	info := evento.Info
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_, err := client.SendPeerMessage(ctx, client.BuildHistorySyncRequest(&info, janela.Quantidade))
		if err != nil {
			fmt.Printf("erro ao solicitar historico de recuperacao na instancia %s chat %s: %v\n", instanciaID, chat, err)
		}
	}()
}

func (g *GerenciadorInstancias) processarHistorico(instanciaID string, client *whatsmeow.Client, evento *events.HistorySync) {
	if client == nil || evento == nil || evento.Data == nil {
		return
	}
	dias := g.obterHistoricoDias(instanciaID)
	janela := g.janelaRecuperacao(instanciaID)
	if dias <= 0 && janela == nil {
		return
	}
	configuracao := g.obterConfiguracaoInstancia(context.Background(), instanciaID)
	var limite time.Time
	if dias > 0 {
		limite = time.Now().UTC().AddDate(0, 0, -dias)
	}
	for _, conversa := range evento.Data.GetConversations() {
		chatJID, err := types.ParseJID(conversa.GetID())
		if err != nil {
			continue
		}
		for _, item := range conversa.GetMessages() {
			webMsg := item.GetMessage()
			if webMsg == nil {
				continue
			}
			msgEvt, err := client.ParseWebMessage(chatJID, webMsg)
			if err != nil || msgEvt == nil {
				continue
			}
			timestamp := msgEvt.Info.Timestamp.UTC()
			if !limite.IsZero() && !timestamp.IsZero() && timestamp.Before(limite) {
				continue
			}
			origem := "historico"
			if janela != nil {
				inicio := janela.Inicio
				fim := janela.Fim
				if !timestamp.IsZero() && (timestamp.Before(inicio) || timestamp.After(fim)) {
					if dias <= 0 {
						continue
					}
				} else {
					origem = "recuperacao"
				}
			}
			if g.deveIgnorarMensagem(configuracao, msgEvt) {
				continue
			}
			g.dispararEventoMensagem(instanciaID, client, msgEvt, origem, true)
		}
	}
}

func (g *GerenciadorInstancias) dispararEventoMensagem(instanciaID string, client *whatsmeow.Client, evento *events.Message, origem string, historico bool) {
	if g.dispatcher == nil || evento == nil {
		return
	}
	if origem == "" {
		origem = "tempo_real"
	}
	if g.mensagemStore != nil && evento.Info.ID != "" {
		inserida, err := g.mensagemStore.RegistrarMensagemProcessada(context.Background(), models.MensagemProcessada{
			InstanciaID:   instanciaID,
			ChatJID:       evento.Info.Chat.String(),
			MensagemID:    string(evento.Info.ID),
			RemetenteJID:  evento.Info.Sender.String(),
			EnviadaPorMim: evento.Info.IsFromMe,
			Grupo:         evento.Info.IsGroup,
			RecebidaEm:    evento.Info.Timestamp.UTC(),
			Origem:        origem,
			ProcessadaEm:  time.Now().UTC(),
		})
		if err != nil {
			fmt.Printf("erro ao registrar mensagem processada na instancia %s: %v\n", instanciaID, err)
		} else if !inserida {
			return
		}
	}
	g.registrarAliasNumero(evento.Info.Chat, evento.Info.RecipientAlt)
	g.registrarAliasNumero(evento.Info.Sender, evento.Info.SenderAlt)
	chatNumero := g.resolverNumeroPreferencial(evento.Info.Chat, evento.Info.RecipientAlt)
	remetenteNumero := g.resolverNumeroPreferencial(evento.Info.Sender, evento.Info.SenderAlt)
	direcao := direcaoMensagem(evento.Info.IsFromMe)
	conteudo, tipo, extras, extrasMensagem := extrairConteudoMensagem(evento.Message)
	if tipo == "ignorada" {
		return
	}
	dados := map[string]interface{}{
		"mensagem_id":      string(evento.Info.ID),
		"chat_jid":         evento.Info.Chat.String(),
		"chat_numero":      chatNumero,
		"grupo":            evento.Info.IsGroup,
		"enviado_por_mim":  evento.Info.IsFromMe,
		"direcao":          direcao,
		"origem":           origem,
		"historico":        historico,
		"remetente":        evento.Info.Sender.String(),
		"remetente_jid":    evento.Info.Sender.String(),
		"remetente_numero": remetenteNumero,
		"nome_remetente":   evento.Info.PushName,
		"conteudo":         conteudo,
		"tipo":             tipo,
		"recebida_em":      evento.Info.Timestamp.UTC(),
		"conversa": map[string]interface{}{
			"jid":    evento.Info.Chat.String(),
			"numero": chatNumero,
			"grupo":  evento.Info.IsGroup,
		},
		"autor": map[string]interface{}{
			"jid":    evento.Info.Sender.String(),
			"numero": remetenteNumero,
			"nome":   evento.Info.PushName,
		},
		"mensagem": map[string]interface{}{
			"id":              string(evento.Info.ID),
			"tipo":            tipo,
			"conteudo":        conteudo,
			"direcao":         direcao,
			"enviado_por_mim": evento.Info.IsFromMe,
			"origem":          origem,
			"historico":       historico,
			"recebida_em":     evento.Info.Timestamp.UTC(),
		},
	}
	for chave, valor := range extras {
		dados[chave] = valor
	}
	mensagemDados, _ := dados["mensagem"].(map[string]interface{})
	for chave, valor := range extrasMensagem {
		mensagemDados[chave] = valor
	}
	midia, err := g.processarMidiaRecebida(instanciaID, client, evento)
	if err != nil {
		dados["midia_erro"] = err.Error()
		mensagemDados["midia_erro"] = err.Error()
	} else if midia != nil {
		g.anexarMidiaRecebidaAoPayload(dados, *midia)
	}
	g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookMensagens, dados)
}

func (g *GerenciadorInstancias) dispararEventoRecibo(instanciaID string, evento *events.Receipt) {
	if g.dispatcher == nil || evento == nil || len(evento.MessageIDs) == 0 {
		return
	}
	g.registrarAliasNumero(evento.Chat, evento.RecipientAlt)
	g.registrarAliasNumero(evento.Sender, evento.SenderAlt)
	chatNumero := g.resolverNumeroPreferencial(evento.Chat, evento.RecipientAlt)
	remetenteNumero := g.resolverNumeroPreferencial(evento.Sender, evento.SenderAlt)
	participanteNumero := g.resolverNumeroPreferencial(evento.MessageSender, types.EmptyJID)
	mensagensID := idsRecibo(evento.MessageIDs)
	status, tipo := statusRecibo(evento.Type)
	ocorridoEm := evento.Timestamp.UTC()
	if ocorridoEm.IsZero() {
		ocorridoEm = time.Now().UTC()
	}
	dados := map[string]interface{}{
		"mensagem_id":         mensagensID[0],
		"mensagens_id":        mensagensID,
		"status":              status,
		"tipo_recibo":         tipo,
		"chat_jid":            evento.Chat.String(),
		"chat_numero":         chatNumero,
		"grupo":               evento.IsGroup,
		"enviado_por_mim":     evento.IsFromMe,
		"direcao":             direcaoMensagem(evento.IsFromMe),
		"remetente":           evento.Sender.String(),
		"remetente_jid":       evento.Sender.String(),
		"remetente_numero":    remetenteNumero,
		"participante":        evento.MessageSender.String(),
		"participante_jid":    evento.MessageSender.String(),
		"participante_numero": participanteNumero,
		"ocorrido_em":         ocorridoEm,
		"conversa": map[string]interface{}{
			"jid":    evento.Chat.String(),
			"numero": chatNumero,
			"grupo":  evento.IsGroup,
		},
		"recibo": map[string]interface{}{
			"status":       status,
			"tipo":         tipo,
			"mensagens_id": mensagensID,
			"ocorrido_em":  ocorridoEm,
		},
	}
	if evento.MessageSender.IsEmpty() {
		dados["participante"] = ""
		dados["participante_jid"] = ""
		dados["participante_numero"] = ""
	}
	g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookRecibos, dados)
}

func idsRecibo(ids []types.MessageID) []string {
	resultado := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		resultado = append(resultado, string(id))
	}
	if len(resultado) == 0 {
		return []string{""}
	}
	return resultado
}

func statusRecibo(tipo types.ReceiptType) (string, string) {
	tipoTexto := string(tipo)
	switch tipo {
	case types.ReceiptTypeDelivered:
		return "entregue", "delivered"
	case types.ReceiptTypeSender:
		return "sincronizada", tipoTexto
	case types.ReceiptTypeRetry:
		return "retry", tipoTexto
	case types.ReceiptTypeRead:
		return "lida", tipoTexto
	case types.ReceiptTypeReadSelf:
		return "lida_por_mim", tipoTexto
	case types.ReceiptTypePlayed:
		return "ouvida", tipoTexto
	case types.ReceiptTypePlayedSelf:
		return "ouvida_por_mim", tipoTexto
	case types.ReceiptTypeServerError:
		return "erro_servidor", tipoTexto
	case types.ReceiptTypeInactive:
		return "inativo", tipoTexto
	case types.ReceiptTypePeerMsg:
		return "peer_msg", tipoTexto
	case types.ReceiptTypeHistorySync:
		return "historico", tipoTexto
	default:
		if tipoTexto == "" {
			return "entregue", "delivered"
		}
		return tipoTexto, tipoTexto
	}
}

func (g *GerenciadorInstancias) anexarMidiaRecebidaAoPayload(dados map[string]interface{}, midia models.MidiaRecebida) {
	downloadPath := midia.DownloadPath()
	downloadURL := g.montarDownloadURLMidia(midia)
	payloadResumo := map[string]interface{}{
		"id":            midia.ID,
		"mensagem_id":   midia.MensagemID,
		"tipo":          midia.Tipo,
		"mime_type":     midia.MimeType,
		"nome_arquivo":  midia.NomeArquivo,
		"tamanho_bytes": midia.TamanhoBytes,
		"sha256":        midia.SHA256,
		"download_path": downloadPath,
	}
	if downloadURL != "" {
		payloadResumo["download_url"] = downloadURL
	}
	if strings.TrimSpace(midia.StorageProvider) != "" {
		payloadResumo["storage_provider"] = midia.StorageProvider
		payloadResumo["storage_path"] = midia.StoragePath
		payloadResumo["storage_url"] = midia.StorageURL
	}
	payloadCompleto := copiarMapaPayload(payloadResumo)
	if strings.TrimSpace(midia.Base64) != "" {
		payloadCompleto["base64"] = midia.Base64
	}
	if strings.TrimSpace(midia.DataURI) != "" {
		payloadCompleto["data_uri"] = midia.DataURI
	}
	dados["midia"] = payloadCompleto
	dados["midia_id"] = midia.ID
	dados["midia_download_path"] = downloadPath
	dados["tamanho_bytes"] = midia.TamanhoBytes
	if downloadURL != "" {
		dados["midia_download_url"] = downloadURL
	}
	if strings.TrimSpace(midia.StorageURL) != "" {
		dados["midia_storage_provider"] = midia.StorageProvider
		dados["midia_storage_path"] = midia.StoragePath
		dados["midia_storage_url"] = midia.StorageURL
	}
	if strings.TrimSpace(midia.NomeArquivo) != "" {
		dados["nome_arquivo"] = midia.NomeArquivo
	}
	if strings.TrimSpace(midia.MimeType) != "" {
		dados["mime_type"] = midia.MimeType
	}
	if mensagemDados, ok := dados["mensagem"].(map[string]interface{}); ok {
		mensagemDados["midia"] = payloadResumo
	}
}

func copiarMapaPayload(origem map[string]interface{}) map[string]interface{} {
	copia := make(map[string]interface{}, len(origem))
	for chave, valor := range origem {
		copia[chave] = valor
	}
	return copia
}

func (g *GerenciadorInstancias) montarDownloadURLMidia(midia models.MidiaRecebida) string {
	if strings.TrimSpace(g.baseURL) == "" {
		return ""
	}
	return g.baseURL + midia.DownloadPath()
}

type midiaMensagemRecebida struct {
	Tipo         string
	MimeType     string
	NomeArquivo  string
	Downloadable whatsmeow.DownloadableMessage
}

func (g *GerenciadorInstancias) processarMidiaRecebida(instanciaID string, client *whatsmeow.Client, evento *events.Message) (*models.MidiaRecebida, error) {
	if client == nil || g.midiaStore == nil || evento == nil {
		return nil, nil
	}
	info := identificarMidiaMensagem(evento.Message)
	if info == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	dados, err := client.Download(ctx, info.Downloadable)
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar midia recebida: %w", err)
	}
	midia, err := g.salvarMidiaRecebida(instanciaID, evento, *info, dados)
	if err != nil {
		return nil, err
	}
	return &midia, nil
}

func (g *GerenciadorInstancias) salvarMidiaRecebida(instanciaID string, evento *events.Message, info midiaMensagemRecebida, dados []byte) (models.MidiaRecebida, error) {
	if len(dados) == 0 {
		return models.MidiaRecebida{}, fmt.Errorf("midia recebida vazia")
	}
	agora := time.Now().UTC()
	mensagemID := string(evento.Info.ID)
	identificador := gerarIDMidiaRecebida(instanciaID, mensagemID, info.Tipo)
	nomeBase := normalizarNomeArquivoMidiaRecebida(info.NomeArquivo, info.Tipo, info.MimeType)
	nomeArquivo := nomeArquivoUnicoMidiaRecebida(nomeBase, mensagemID)
	extensao := strings.ToLower(filepath.Ext(nomeArquivo))
	if extensao == "" {
		extensao = extensaoPorMime(info.MimeType)
	}
	if extensao == "" {
		extensao = ".bin"
	}
	diretorio := filepath.Join(g.diretorioMidias, "recebidas", instanciaID, agora.Format("20060102"))
	if err := os.MkdirAll(diretorio, 0o755); err != nil {
		return models.MidiaRecebida{}, fmt.Errorf("erro ao criar diretorio da midia recebida: %w", err)
	}
	caminhoArquivo := filepath.Join(diretorio, identificador+extensao)
	if err := os.WriteFile(caminhoArquivo, dados, 0o644); err != nil {
		return models.MidiaRecebida{}, fmt.Errorf("erro ao salvar midia recebida: %w", err)
	}
	if absoluto, err := filepath.Abs(caminhoArquivo); err == nil {
		caminhoArquivo = absoluto
	}
	soma := sha256.Sum256(dados)
	mimeType := limparMimeType(info.MimeType)
	arquivoBase64 := base64.StdEncoding.EncodeToString(dados)
	dataURI := ""
	if mimeType != "" {
		dataURI = "data:" + mimeType + ";base64," + arquivoBase64
	}
	midia := models.MidiaRecebida{
		ID:             identificador,
		InstanciaID:    instanciaID,
		MensagemID:     mensagemID,
		ChatJID:        evento.Info.Chat.String(),
		RemetenteJID:   evento.Info.Sender.String(),
		Tipo:           info.Tipo,
		MimeType:       mimeType,
		NomeArquivo:    nomeArquivo,
		CaminhoArquivo: caminhoArquivo,
		TamanhoBytes:   int64(len(dados)),
		SHA256:         hex.EncodeToString(soma[:]),
		Base64:         arquivoBase64,
		DataURI:        dataURI,
		RecebidaEm:     evento.Info.Timestamp.UTC(),
		CriadaEm:       agora,
	}
	if g.midiaUploader != nil {
		uploadCtx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
		defer cancel()
		objectPath := path.Join("instancias", instanciaID, agora.Format("20060102"), identificador+extensao)
		resultado, err := g.midiaUploader.Enviar(uploadCtx, objectPath, mimeType, dados)
		if err != nil {
			fmt.Printf("erro ao enviar midia %s para storage externo: %v\n", identificador, err)
		} else {
			midia.StorageProvider = resultado.Provider
			midia.StoragePath = resultado.ObjectPath
			midia.StorageURL = resultado.URL
		}
	}
	return g.midiaStore.SalvarMidiaRecebida(context.Background(), midia)
}

func identificarMidiaMensagem(msg *waE2E.Message) *midiaMensagemRecebida {
	msg = normalizarMensagemConteudo(msg)
	if msg == nil {
		return nil
	}
	if imagem := msg.GetImageMessage(); imagem != nil {
		return &midiaMensagemRecebida{Tipo: "imagem", MimeType: imagem.GetMimetype(), NomeArquivo: normalizarNomeArquivoMidiaRecebida("", "imagem", imagem.GetMimetype()), Downloadable: imagem}
	}
	if documento := msg.GetDocumentMessage(); documento != nil {
		return &midiaMensagemRecebida{Tipo: "documento", MimeType: documento.GetMimetype(), NomeArquivo: documento.GetFileName(), Downloadable: documento}
	}
	if audio := msg.GetAudioMessage(); audio != nil {
		return &midiaMensagemRecebida{Tipo: "audio", MimeType: audio.GetMimetype(), NomeArquivo: normalizarNomeArquivoMidiaRecebida("", "audio", audio.GetMimetype()), Downloadable: audio}
	}
	if video := msg.GetVideoMessage(); video != nil {
		return &midiaMensagemRecebida{Tipo: "video", MimeType: video.GetMimetype(), NomeArquivo: normalizarNomeArquivoMidiaRecebida("", "video", video.GetMimetype()), Downloadable: video}
	}
	if sticker := msg.GetStickerMessage(); sticker != nil {
		return &midiaMensagemRecebida{Tipo: "sticker", MimeType: sticker.GetMimetype(), NomeArquivo: normalizarNomeArquivoMidiaRecebida("", "sticker", sticker.GetMimetype()), Downloadable: sticker}
	}
	return nil
}

func normalizarNomeArquivoMidiaRecebida(nomeArquivo, tipo, mimeType string) string {
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)
	if nomeArquivo != "" {
		return nomeArquivo
	}
	prefixo := tipo
	if prefixo == "" {
		prefixo = "midia"
	}
	extensao := extensaoPorMime(mimeType)
	if extensao == "" {
		extensao = ".bin"
	}
	return prefixo + extensao
}

func nomeArquivoUnicoMidiaRecebida(nomeArquivo, mensagemID string) string {
	nomeArquivo = sanitizarNomeArquivo(nomeArquivo)
	mensagemID = sanitizarNomeArquivo(mensagemID)
	if nomeArquivo == "" {
		nomeArquivo = "midia.bin"
	}
	if mensagemID == "" {
		return nomeArquivo
	}
	extensao := strings.ToLower(filepath.Ext(nomeArquivo))
	base := strings.TrimSuffix(nomeArquivo, filepath.Ext(nomeArquivo))
	base = strings.TrimSpace(base)
	if base == "" {
		base = "midia"
	}
	return mensagemID + "_" + base + extensao
}
func sanitizarNomeArquivo(valor string) string {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return ""
	}
	substitutos := []string{"<", ">", ":", "\"", "/", "\\", "|", "?", "*"}
	for _, alvo := range substitutos {
		valor = strings.ReplaceAll(valor, alvo, "_")
	}
	valor = strings.ReplaceAll(valor, "\r", " ")
	valor = strings.ReplaceAll(valor, "\n", " ")
	valor = strings.TrimSpace(valor)
	if valor == "." || valor == ".." {
		return ""
	}
	return valor
}

func extensaoPorMime(mimeType string) string {
	mimeType = limparMimeType(mimeType)
	switch mimeType {
	case "audio/ogg":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".m4a"
	case "image/jpeg":
		return ".jpg"
	case "application/pdf":
		return ".pdf"
	}
	extensoes, err := mime.ExtensionsByType(mimeType)
	if err == nil && len(extensoes) > 0 {
		return extensoes[0]
	}
	return ""
}

func gerarIDMidiaRecebida(instanciaID, mensagemID, tipo string) string {
	soma := sha256.Sum256([]byte(instanciaID + ":" + mensagemID + ":" + tipo))
	return hex.EncodeToString(soma[:12])
}

func (g *GerenciadorInstancias) dispararEventoPresenca(instanciaID string, evento *events.ChatPresence) {
	if g.dispatcher == nil || evento == nil {
		return
	}
	g.registrarAliasNumero(evento.MessageSource.Chat, evento.MessageSource.RecipientAlt)
	g.registrarAliasNumero(evento.MessageSource.Sender, evento.MessageSource.SenderAlt)
	chatNumero := g.resolverNumeroPreferencial(evento.MessageSource.Chat, evento.MessageSource.RecipientAlt)
	remetenteNumero := g.resolverNumeroPreferencial(evento.MessageSource.Sender, evento.MessageSource.SenderAlt)
	acao := acaoPresenca(evento.Media, evento.State)
	dados := map[string]interface{}{
		"chat_jid":         evento.MessageSource.Chat.String(),
		"chat_numero":      chatNumero,
		"grupo":            evento.MessageSource.IsGroup,
		"remetente":        evento.MessageSource.Sender.String(),
		"remetente_jid":    evento.MessageSource.Sender.String(),
		"remetente_numero": remetenteNumero,
		"estado":           string(evento.State),
		"estado_texto":     estadoPresencaTexto(evento.State),
		"media":            string(evento.Media),
		"acao_presenca":    acao,
		"conversa": map[string]interface{}{
			"jid":    evento.MessageSource.Chat.String(),
			"numero": chatNumero,
			"grupo":  evento.MessageSource.IsGroup,
		},
		"autor": map[string]interface{}{
			"jid":    evento.MessageSource.Sender.String(),
			"numero": remetenteNumero,
		},
		"presenca": map[string]interface{}{
			"acao":         acao,
			"estado":       string(evento.State),
			"estado_texto": estadoPresencaTexto(evento.State),
			"media":        string(evento.Media),
		},
	}
	if evento.Media == types.ChatPresenceMediaAudio {
		g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookGravandoAudio, dados)
		return
	}
	g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookDigitando, dados)
}

func (g *GerenciadorInstancias) prepararAcompanhamentoPresenca(instanciaID string, client *whatsmeow.Client, evento *events.Message) {
	if client == nil || evento == nil {
		return
	}
	go func() {
		_ = g.aplicarPresencaPersistente(context.Background(), instanciaID, &runtimeInstancia{client: client})
		if evento.Info.IsGroup {
			return
		}
		for _, jid := range []types.JID{evento.Info.Chat, evento.Info.Sender, evento.Info.SenderAlt, evento.Info.RecipientAlt} {
			if jid.User == "" {
				continue
			}
			if jid.Server == types.DefaultUserServer || jid.Server == types.HiddenUserServer {
				_ = client.SubscribePresence(context.Background(), jid)
			}
		}
	}()
}
func (g *GerenciadorInstancias) resolverNumeroPreferencial(principal, alternativo types.JID) string {
	g.registrarAliasNumero(principal, alternativo)
	if numero := extrairNumeroJID(alternativo); numero != "" && alternativo.Server != types.HiddenUserServer {
		return numero
	}
	if numero := extrairNumeroJID(principal); numero != "" && principal.Server != types.HiddenUserServer {
		return numero
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, chave := range []string{principal.String(), alternativo.String(), principal.User, alternativo.User} {
		if numero := strings.TrimSpace(g.aliasesNumero[chave]); numero != "" {
			return numero
		}
	}
	if numero := extrairNumeroJID(alternativo); numero != "" {
		return numero
	}
	return extrairNumeroJID(principal)
}

func (g *GerenciadorInstancias) registrarAliasNumero(principal, alternativo types.JID) {
	numeroPrincipal := extrairNumeroJID(principal)
	numeroAlternativo := extrairNumeroJID(alternativo)
	var numeroReal string
	if alternativo.Server != types.HiddenUserServer && numeroAlternativo != "" {
		numeroReal = numeroAlternativo
	} else if principal.Server != types.HiddenUserServer && numeroPrincipal != "" {
		numeroReal = numeroPrincipal
	}
	if numeroReal == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, chave := range []string{principal.String(), alternativo.String(), principal.User, alternativo.User} {
		if strings.TrimSpace(chave) != "" {
			g.aliasesNumero[chave] = numeroReal
		}
	}
}

func extrairNumeroJID(jid types.JID) string {
	if jid.User == "" {
		return ""
	}
	normalizado := apenasDigitos.ReplaceAllString(jid.User, "")
	if normalizado == "" {
		return ""
	}
	return normalizado
}
func extrairConteudoMensagem(msg *waE2E.Message) (string, string, map[string]interface{}, map[string]interface{}) {
	msg = normalizarMensagemConteudo(msg)
	if msg == nil {
		return "", "ignorada", nil, nil
	}
	if texto := strings.TrimSpace(msg.GetConversation()); texto != "" {
		return texto, "texto", nil, nil
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		if texto := strings.TrimSpace(ext.GetText()); texto != "" {
			return texto, "texto", nil, nil
		}
	}
	if conteudo, extras, extrasMensagem, ok := extrairRespostaLista(msg); ok {
		return conteudo, "lista", extras, extrasMensagem
	}
	if conteudo, extras, extrasMensagem, ok := extrairRespostaBotao(msg); ok {
		return conteudo, "botao", extras, extrasMensagem
	}
	if img := msg.GetImageMessage(); img != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(img.GetMimetype())}
		return strings.TrimSpace(img.GetCaption()), "imagem", extras, map[string]interface{}{"mime_type": extras["mime_type"]}
	}
	if doc := msg.GetDocumentMessage(); doc != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(doc.GetMimetype()), "nome_arquivo": strings.TrimSpace(doc.GetFileName())}
		return strings.TrimSpace(doc.GetCaption()), "documento", extras, map[string]interface{}{"mime_type": extras["mime_type"], "nome_arquivo": extras["nome_arquivo"]}
	}
	if audio := msg.GetAudioMessage(); audio != nil {
		extras := map[string]interface{}{
			"mime_type":        limparMimeType(audio.GetMimetype()),
			"duracao_segundos": audio.GetSeconds(),
			"ptt":              audio.GetPTT(),
		}
		return "audio recebido", "audio", extras, map[string]interface{}{"mime_type": extras["mime_type"], "duracao_segundos": extras["duracao_segundos"], "ptt": extras["ptt"]}
	}
	if video := msg.GetVideoMessage(); video != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(video.GetMimetype())}
		return strings.TrimSpace(video.GetCaption()), "video", extras, map[string]interface{}{"mime_type": extras["mime_type"]}
	}
	if sticker := msg.GetStickerMessage(); sticker != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(sticker.GetMimetype())}
		return "sticker recebido", "sticker", extras, map[string]interface{}{"mime_type": extras["mime_type"]}
	}
	return "mensagem recebida", "desconhecido", nil, nil
}

func extrairRespostaBotao(msg *waE2E.Message) (string, map[string]interface{}, map[string]interface{}, bool) {
	if resposta := msg.GetButtonsResponseMessage(); resposta != nil {
		id := strings.TrimSpace(resposta.GetSelectedButtonID())
		texto := strings.TrimSpace(resposta.GetSelectedDisplayText())
		conteudo, extras, extrasMensagem := montarExtrasBotao(id, texto, "buttons", nil)
		return conteudo, extras, extrasMensagem, true
	}
	if resposta := msg.GetTemplateButtonReplyMessage(); resposta != nil {
		id := strings.TrimSpace(resposta.GetSelectedID())
		texto := strings.TrimSpace(resposta.GetSelectedDisplayText())
		conteudo, extras, extrasMensagem := montarExtrasBotao(id, texto, "template", nil)
		return conteudo, extras, extrasMensagem, true
	}
	if resposta := msg.GetInteractiveResponseMessage(); resposta != nil {
		native := resposta.GetNativeFlowResponseMessage()
		if native == nil {
			return "", nil, nil, false
		}
		params := parseJSONObjeto(native.GetParamsJSON())
		id := stringParam(params, "id", "selected_id", "button_id")
		texto := stringParam(params, "display_text", "text", "title")
		if texto == "" {
			texto = strings.TrimSpace(resposta.GetBody().GetText())
		}
		tipo := strings.TrimSpace(native.GetName())
		if tipo == "" {
			tipo = "native_flow"
		}
		conteudo, extras, extrasMensagem := montarExtrasBotao(id, texto, tipo, params)
		return conteudo, extras, extrasMensagem, true
	}
	return "", nil, nil, false
}

func extrairRespostaLista(msg *waE2E.Message) (string, map[string]interface{}, map[string]interface{}, bool) {
	resposta := msg.GetListResponseMessage()
	if resposta == nil || resposta.GetSingleSelectReply() == nil {
		return "", nil, nil, false
	}
	id := strings.TrimSpace(resposta.GetSingleSelectReply().GetSelectedRowID())
	titulo := strings.TrimSpace(resposta.GetTitle())
	descricao := strings.TrimSpace(resposta.GetDescription())
	conteudo := descricao
	if conteudo == "" {
		conteudo = titulo
	}
	if conteudo == "" {
		conteudo = id
	}
	lista := map[string]interface{}{
		"id":        id,
		"titulo":    titulo,
		"descricao": descricao,
	}
	extras := map[string]interface{}{
		"lista_id":        id,
		"lista_titulo":    titulo,
		"lista_descricao": descricao,
	}
	return conteudo, extras, map[string]interface{}{"lista": lista}, true
}

func montarExtrasBotao(id, texto, tipo string, params map[string]interface{}) (string, map[string]interface{}, map[string]interface{}) {
	id = strings.TrimSpace(id)
	texto = strings.TrimSpace(texto)
	tipo = strings.TrimSpace(tipo)
	if texto == "" {
		texto = id
	}
	if tipo == "" {
		tipo = "desconhecido"
	}

	botao := map[string]interface{}{
		"id":    id,
		"texto": texto,
		"tipo":  tipo,
	}
	extras := map[string]interface{}{
		"botao_id":    id,
		"botao_texto": texto,
		"botao_tipo":  tipo,
	}
	if len(params) > 0 {
		botao["params"] = params
		extras["botao_params"] = params
	}
	return texto, extras, map[string]interface{}{"botao": botao}
}

func parseJSONObjeto(valor string) map[string]interface{} {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return nil
	}
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(valor), &params); err != nil {
		return nil
	}
	return params
}

func stringParam(params map[string]interface{}, chaves ...string) string {
	for _, chave := range chaves {
		if valor, ok := params[chave]; ok {
			if texto, ok := valor.(string); ok {
				return strings.TrimSpace(texto)
			}
		}
	}
	return ""
}

func resolverDuracaoAudio(req models.EnvioMidiaRequest, mimeType string, dados []byte) uint32 {
	if req.DuracaoSegundos > 0 {
		return req.DuracaoSegundos
	}
	if mimeType == "audio/ogg; codecs=opus" {
		return duracaoOggOpusSegundos(dados)
	}
	return 0
}

func duracaoOggOpusSegundos(dados []byte) uint32 {
	if len(dados) < 27 {
		return 0
	}
	var (
		preSkip    uint64
		headLido   bool
		ultimoGran uint64
		cursor     int
	)
	for cursor < len(dados) {
		indice := bytes.Index(dados[cursor:], []byte("OggS"))
		if indice < 0 {
			break
		}
		inicio := cursor + indice
		if inicio+27 > len(dados) {
			break
		}
		qtdSegmentos := int(dados[inicio+26])
		tabelaInicio := inicio + 27
		payloadInicio := tabelaInicio + qtdSegmentos
		if payloadInicio > len(dados) {
			break
		}
		tamanhoPayload := 0
		for _, segmento := range dados[tabelaInicio:payloadInicio] {
			tamanhoPayload += int(segmento)
		}
		payloadFim := payloadInicio + tamanhoPayload
		if payloadFim > len(dados) {
			break
		}
		granule := binary.LittleEndian.Uint64(dados[inicio+6 : inicio+14])
		if granule > 0 {
			ultimoGran = granule
		}
		if !headLido {
			payload := dados[payloadInicio:payloadFim]
			if len(payload) >= 12 && bytes.HasPrefix(payload, []byte("OpusHead")) {
				preSkip = uint64(binary.LittleEndian.Uint16(payload[10:12]))
				headLido = true
			}
		}
		cursor = payloadFim
	}
	if ultimoGran == 0 {
		return 0
	}
	if ultimoGran > preSkip {
		ultimoGran -= preSkip
	}
	segundos := uint32((ultimoGran + 47999) / 48000)
	if segundos == 0 && ultimoGran > 0 {
		return 1
	}
	return segundos
}

func normalizarMensagemConteudo(msg *waE2E.Message) *waE2E.Message {
	for i := 0; i < 8 && msg != nil; i++ {
		switch {
		case msg.GetDeviceSentMessage() != nil:
			msg = msg.GetDeviceSentMessage().GetMessage()
		case msg.GetEphemeralMessage() != nil:
			msg = msg.GetEphemeralMessage().GetMessage()
		case msg.GetViewOnceMessage() != nil:
			msg = msg.GetViewOnceMessage().GetMessage()
		case msg.GetViewOnceMessageV2() != nil:
			msg = msg.GetViewOnceMessageV2().GetMessage()
		case msg.GetViewOnceMessageV2Extension() != nil:
			msg = msg.GetViewOnceMessageV2Extension().GetMessage()
		case msg.GetEditedMessage() != nil:
			msg = msg.GetEditedMessage().GetMessage()
		case msg.GetProtocolMessage() != nil && msg.GetProtocolMessage().GetEditedMessage() != nil:
			msg = msg.GetProtocolMessage().GetEditedMessage()
		default:
			return msg
		}
	}
	return msg
}

func (g *GerenciadorInstancias) consumirQRCode(instanciaID string, runtime *runtimeInstancia, qrChan <-chan whatsmeow.QRChannelItem) {
	defer runtime.cancelarFluxoQR()
	for item := range qrChan {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			estado := g.obterEstado(instanciaID)
			if estado.metodoPareamento == metodoPareamentoCodigo {
				continue
			}
			g.definirEstado(instanciaID, models.StatusInstanciaAguardandoQR, item.Code, "")
		case whatsmeow.QRChannelSuccess.Event:
			g.definirEstado(instanciaID, "autenticando", "", "")
		case whatsmeow.QRChannelTimeout.Event:
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "tempo esgotado para leitura do qrcode")
		case whatsmeow.QRChannelErrUnexpectedEvent.Event:
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "estado inesperado ao gerar qrcode")
		default:
			errTexto := ""
			if item.Error != nil {
				errTexto = item.Error.Error()
			}
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", errTexto)
		}
	}
}

func (g *GerenciadorInstancias) tratarEventoPareamentoFalho(instanciaID string, item whatsmeow.QRChannelItem) error {
	switch item.Event {
	case whatsmeow.QRChannelSuccess.Event:
		g.definirEstado(instanciaID, "autenticando", "", "")
		return fmt.Errorf("o pareamento entrou em autenticacao antes do pairing code ser emitido")
	case whatsmeow.QRChannelTimeout.Event:
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "tempo esgotado para gerar pairing code")
		return fmt.Errorf("tempo esgotado para gerar pairing code")
	case whatsmeow.QRChannelErrUnexpectedEvent.Event:
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "estado inesperado ao gerar pairing code")
		return fmt.Errorf("estado inesperado ao gerar pairing code")
	default:
		errTexto := ""
		if item.Error != nil {
			errTexto = item.Error.Error()
		}
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", errTexto)
		if errTexto == "" {
			errTexto = "erro inesperado ao gerar pairing code"
		}
		return fmt.Errorf("%s", errTexto)
	}
}

func acaoPresenca(media types.ChatPresenceMedia, estado types.ChatPresence) string {
	if media == types.ChatPresenceMediaAudio && estado == types.ChatPresenceComposing {
		return "gravando_audio"
	}
	if estado == types.ChatPresenceComposing {
		return "digitando"
	}
	if estado == types.ChatPresencePaused {
		return "pausado"
	}
	return string(estado)
}

func (g *GerenciadorInstancias) definirEstado(instanciaID, status, qrCode, ultimoErro string) {
	metodoPareamento := ""
	if qrCode != "" {
		metodoPareamento = metodoPareamentoQR
	}
	g.definirEstadoComPareamento(instanciaID, status, qrCode, "", "", metodoPareamento, ultimoErro)
}

func (g *GerenciadorInstancias) definirEstadoCodigoPareamento(instanciaID, status, codigo, numero, ultimoErro string) {
	g.definirEstadoComPareamento(instanciaID, status, "", codigo, numero, metodoPareamentoCodigo, ultimoErro)
}

func (g *GerenciadorInstancias) definirEstadoComPareamento(instanciaID, status, qrCode, pairingCode, pairingPhone, metodoPareamento, ultimoErro string) {
	agora := time.Now().UTC()
	g.mu.Lock()
	g.estados[instanciaID] = estadoRuntime{
		status:           status,
		qrCode:           qrCode,
		pairingCode:      pairingCode,
		pairingPhone:     pairingPhone,
		metodoPareamento: metodoPareamento,
		ultimoErro:       ultimoErro,
		atualizadoEm:     agora,
	}
	g.mu.Unlock()
	if g.dispatcher != nil {
		g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookStatus, map[string]interface{}{
			"status":                   status,
			"erro":                     ultimoErro,
			"qrcode_pronto":            qrCode != "",
			"codigo_pareamento_pronto": pairingCode != "",
			"numero_pareamento":        pairingPhone,
			"metodo_pareamento":        metodoPareamento,
			"atualizado_em":            agora,
		})
	}
}
func (g *GerenciadorInstancias) obterEstado(instanciaID string) estadoRuntime {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.estados[instanciaID]
}

var apenasDigitos = regexp.MustCompile(`[^0-9]`)

func direcaoMensagem(enviadaPorMim bool) string {
	if enviadaPorMim {
		return "saida"
	}
	return "entrada"
}

func estadoPresencaTexto(estado types.ChatPresence) string {
	if estado == types.ChatPresenceComposing {
		return "digitando"
	}
	if estado == types.ChatPresencePaused {
		return "pausado"
	}
	return string(estado)
}

func montarMensagemTexto(req models.EnvioTextoRequest, jid types.JID) *waE2E.Message {
	msg := &waE2E.Message{Conversation: proto.String(req.Mensagem)}
	if strings.TrimSpace(req.RespostaMensagemID) == "" {
		return msg
	}
	msg.ExtendedTextMessage = &waE2E.ExtendedTextMessage{
		Text: proto.String(req.Mensagem),
		ContextInfo: &waE2E.ContextInfo{
			StanzaID:      proto.String(strings.TrimSpace(req.RespostaMensagemID)),
			Participant:   proto.String(strings.TrimSpace(req.RespostaParticipante)),
			RemoteJID:     proto.String(jid.String()),
			QuotedMessage: &waE2E.Message{Conversation: proto.String(strings.TrimSpace(req.RespostaConteudo))},
		},
	}
	msg.Conversation = nil
	return msg
}

func montarMensagemBotoes(req models.EnvioBotoesRequest) (*waE2E.Message, string, error) {
	modo := strings.ToLower(strings.TrimSpace(req.Modo))
	switch modo {
	case "", "auto", "native_flow", "native_flow_direct", "direct":
		msg, err := montarMensagemBotoesNativeFlowDireto(req)
		return msg, "native_flow_direct", err
	case "native_flow_view_once", "view_once", "viewonce":
		msg, err := montarMensagemBotoesNativeFlowViewOnce(req)
		return msg, "native_flow_view_once", err
	case "template", "wuzapi_template", "hydrated_template":
		msg, err := montarMensagemBotoesTemplate(req)
		return msg, "template", err
	case "buttons", "legacy":
		return montarMensagemBotoesLegacy(req), "buttons", nil
	default:
		return nil, "", fmt.Errorf("modo de botoes invalido: use native_flow, native_flow_view_once, template ou buttons")
	}
}

func usarFallbackTextoBotoes(req models.EnvioBotoesRequest) bool {
	if req.FallbackTexto {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "texto", "text", "fallback_texto":
		return true
	default:
		return false
	}
}

func textoMensagemBotoes(req models.EnvioBotoesRequest) string {
	if texto := strings.TrimSpace(req.Texto); texto != "" {
		return texto
	}
	return strings.TrimSpace(req.Mensagem)
}

func textoFallbackBotoes(req models.EnvioBotoesRequest) string {
	partes := make([]string, 0, 4+len(req.Botoes))
	if titulo := strings.TrimSpace(req.Titulo); titulo != "" {
		partes = append(partes, titulo)
	}
	if texto := strings.TrimSpace(textoMensagemBotoes(req)); texto != "" {
		partes = append(partes, texto)
	}
	if len(req.Botoes) > 0 {
		opcoes := make([]string, 0, len(req.Botoes))
		for i, botao := range req.Botoes {
			opcoes = append(opcoes, fmt.Sprintf("%d. %s", i+1, textoBotao(botao)))
		}
		partes = append(partes, strings.Join(opcoes, "\n"))
	}
	if rodape := strings.TrimSpace(req.Rodape); rodape != "" {
		partes = append(partes, rodape)
	}
	return strings.Join(partes, "\n\n")
}

func montarMensagemBotoesNativeFlowDireto(req models.EnvioBotoesRequest) (*waE2E.Message, error) {
	interactive, err := montarInteractiveBotoesNativeFlow(req)
	if err != nil {
		return nil, err
	}
	return &waE2E.Message{
		InteractiveMessage: interactive,
		MessageContextInfo: &waE2E.MessageContextInfo{
			DeviceListMetadata:        &waE2E.DeviceListMetadata{},
			DeviceListMetadataVersion: proto.Int32(2),
		},
	}, nil
}

func montarMensagemBotoesNativeFlowViewOnce(req models.EnvioBotoesRequest) (*waE2E.Message, error) {
	interactive, err := montarInteractiveBotoesNativeFlow(req)
	if err != nil {
		return nil, err
	}
	return &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				InteractiveMessage: interactive,
				MessageContextInfo: &waE2E.MessageContextInfo{
					DeviceListMetadata:        &waE2E.DeviceListMetadata{},
					DeviceListMetadataVersion: proto.Int32(2),
				},
			},
		},
	}, nil
}

func montarInteractiveBotoesNativeFlow(req models.EnvioBotoesRequest) (*waE2E.InteractiveMessage, error) {
	botoes := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, 0, len(req.Botoes))
	for _, botao := range req.Botoes {
		if !ehBotaoResposta(botao) {
			return nil, fmt.Errorf("native_flow aceita apenas botoes quickreply; use modo template para url/call")
		}
		payload, err := json.Marshal(map[string]string{
			"display_text": textoBotao(botao),
			"id":           strings.TrimSpace(botao.ID),
		})
		if err != nil {
			return nil, err
		}
		botoes = append(botoes, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String("quick_reply"),
			ButtonParamsJSON: proto.String(string(payload)),
		})
	}

	interactive := &waE2E.InteractiveMessage{
		Header: &waE2E.InteractiveMessage_Header{
			Title:              proto.String(strings.TrimSpace(req.Titulo)),
			HasMediaAttachment: proto.Bool(false),
		},
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(textoMensagemBotoes(req))},
		Footer: &waE2E.InteractiveMessage_Footer{
			Text: proto.String(strings.TrimSpace(req.Rodape)),
		},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons:           botoes,
				MessageParamsJSON: proto.String(""),
				MessageVersion:    proto.Int32(1),
			},
		},
		ContextInfo: contextInfoBotoes(req),
	}
	return interactive, nil
}

func montarMensagemBotoesTemplate(req models.EnvioBotoesRequest) (*waE2E.Message, error) {
	botoes := make([]*waE2E.HydratedTemplateButton, 0, len(req.Botoes))
	for i, botao := range req.Botoes {
		hidratado, err := montarBotaoTemplate(uint32(i), botao)
		if err != nil {
			return nil, err
		}
		botoes = append(botoes, hidratado)
	}

	template := &waE2E.TemplateMessage_HydratedFourRowTemplate{
		HydratedContentText: proto.String(textoMensagemBotoes(req)),
		HydratedFooterText:  proto.String(strings.TrimSpace(req.Rodape)),
		HydratedButtons:     botoes,
	}
	if titulo := strings.TrimSpace(req.Titulo); titulo != "" {
		template.Title = &waE2E.TemplateMessage_HydratedFourRowTemplate_HydratedTitleText{
			HydratedTitleText: titulo,
		}
	}

	return &waE2E.Message{
		TemplateMessage: &waE2E.TemplateMessage{
			Format: &waE2E.TemplateMessage_HydratedFourRowTemplate_{
				HydratedFourRowTemplate: template,
			},
			ContextInfo: contextInfoBotoes(req),
		},
	}, nil
}

func montarBotaoTemplate(indice uint32, botao models.BotaoRequest) (*waE2E.HydratedTemplateButton, error) {
	texto := textoBotao(botao)
	if texto == "" {
		return nil, fmt.Errorf("botao template precisa de texto")
	}
	hidratado := &waE2E.HydratedTemplateButton{Index: proto.Uint32(indice)}
	switch tipoBotaoEnvio(botao) {
	case "", "quickreply", "quick_reply", "reply":
		id := strings.TrimSpace(botao.ID)
		if id == "" {
			id = texto
		}
		hidratado.HydratedButton = &waE2E.HydratedTemplateButton_QuickReplyButton{
			QuickReplyButton: &waE2E.HydratedTemplateButton_HydratedQuickReplyButton{
				DisplayText: proto.String(texto),
				ID:          proto.String(id),
			},
		}
	case "url":
		url := urlBotaoEnvio(botao)
		if url == "" {
			return nil, fmt.Errorf("botao url precisa de URL")
		}
		hidratado.HydratedButton = &waE2E.HydratedTemplateButton_UrlButton{
			UrlButton: &waE2E.HydratedTemplateButton_HydratedURLButton{
				DisplayText: proto.String(texto),
				URL:         proto.String(url),
			},
		}
	case "call":
		telefone := strings.TrimSpace(botao.PhoneNumber)
		if telefone == "" {
			return nil, fmt.Errorf("botao call precisa de PhoneNumber")
		}
		hidratado.HydratedButton = &waE2E.HydratedTemplateButton_CallButton{
			CallButton: &waE2E.HydratedTemplateButton_HydratedCallButton{
				DisplayText: proto.String(texto),
				PhoneNumber: proto.String(telefone),
			},
		}
	default:
		return nil, fmt.Errorf("tipo de botao template invalido")
	}
	return hidratado, nil
}

func montarMensagemBotoesLegacy(req models.EnvioBotoesRequest) *waE2E.Message {
	tipoResposta := waE2E.ButtonsMessage_Button_RESPONSE
	botoes := make([]*waE2E.ButtonsMessage_Button, 0, len(req.Botoes))
	for _, botao := range req.Botoes {
		botoes = append(botoes, &waE2E.ButtonsMessage_Button{
			ButtonID: proto.String(strings.TrimSpace(botao.ID)),
			ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{
				DisplayText: proto.String(textoBotao(botao)),
			},
			Type: tipoResposta.Enum(),
		})
	}

	tipoHeader := waE2E.ButtonsMessage_EMPTY
	buttonsMessage := &waE2E.ButtonsMessage{
		ContentText: proto.String(textoMensagemBotoes(req)),
		Buttons:     botoes,
		HeaderType:  tipoHeader.Enum(),
	}
	if titulo := strings.TrimSpace(req.Titulo); titulo != "" {
		tipoHeader = waE2E.ButtonsMessage_TEXT
		buttonsMessage.Header = &waE2E.ButtonsMessage_Text{Text: titulo}
		buttonsMessage.HeaderType = tipoHeader.Enum()
	}
	if rodape := strings.TrimSpace(req.Rodape); rodape != "" {
		buttonsMessage.FooterText = proto.String(rodape)
	}
	buttonsMessage.ContextInfo = contextInfoBotoes(req)
	return &waE2E.Message{ButtonsMessage: buttonsMessage}
}

func textoBotao(botao models.BotaoRequest) string {
	if texto := strings.TrimSpace(botao.Texto); texto != "" {
		return texto
	}
	return strings.TrimSpace(botao.DisplayText)
}

func tipoBotaoEnvio(botao models.BotaoRequest) string {
	tipo := strings.TrimSpace(botao.Tipo)
	if tipo == "" {
		tipo = strings.TrimSpace(botao.Type)
	}
	return strings.ToLower(tipo)
}

func ehBotaoResposta(botao models.BotaoRequest) bool {
	switch tipoBotaoEnvio(botao) {
	case "", "quickreply", "quick_reply", "reply":
		return true
	default:
		return false
	}
}

func urlBotaoEnvio(botao models.BotaoRequest) string {
	if url := strings.TrimSpace(botao.URL); url != "" {
		return url
	}
	return strings.TrimSpace(botao.Url)
}

func contextInfoBotoes(req models.EnvioBotoesRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return &waE2E.ContextInfo{}
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
}

func usarFallbackTextoLista(req models.EnvioListaRequest) bool {
	if req.FallbackTexto {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "texto", "text", "fallback_texto":
		return true
	default:
		return false
	}
}

func erroInterativoNaoPermitido(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, whatsmeow.ErrIQNotAllowed) {
		return true
	}
	texto := strings.ToLower(err.Error())
	return strings.Contains(texto, "server returned error 405") || strings.Contains(texto, "not-allowed")
}

func montarMensagemLista(req models.EnvioListaRequest) *waE2E.Message {
	secoes := make([]*waE2E.ListMessage_Section, 0, len(req.Secoes))
	for _, secao := range req.Secoes {
		linhas := make([]*waE2E.ListMessage_Row, 0, len(secao.Linhas))
		for _, linha := range secao.Linhas {
			linhas = append(linhas, &waE2E.ListMessage_Row{
				Title:       proto.String(strings.TrimSpace(linha.Titulo)),
				Description: proto.String(strings.TrimSpace(linha.Descricao)),
				RowID:       proto.String(strings.TrimSpace(linha.ID)),
			})
		}
		secoes = append(secoes, &waE2E.ListMessage_Section{
			Title: proto.String(strings.TrimSpace(secao.Titulo)),
			Rows:  linhas,
		})
	}

	return &waE2E.Message{
		ListMessage: &waE2E.ListMessage{
			Title:       proto.String(strings.TrimSpace(req.Titulo)),
			Description: proto.String(textoMensagemListaEnvio(req)),
			ButtonText:  proto.String(strings.TrimSpace(req.BotaoTexto)),
			ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
			Sections:    secoes,
			FooterText:  proto.String(strings.TrimSpace(req.Rodape)),
			ContextInfo: contextInfoLista(req),
		},
	}
}

func textoMensagemListaEnvio(req models.EnvioListaRequest) string {
	if texto := strings.TrimSpace(req.Descricao); texto != "" {
		return texto
	}
	return strings.TrimSpace(req.Mensagem)
}

func textoFallbackLista(req models.EnvioListaRequest) string {
	partes := make([]string, 0, 6)
	if titulo := strings.TrimSpace(req.Titulo); titulo != "" {
		partes = append(partes, titulo)
	}
	if texto := strings.TrimSpace(textoMensagemListaEnvio(req)); texto != "" {
		partes = append(partes, texto)
	}
	for _, secao := range req.Secoes {
		bloco := make([]string, 0, len(secao.Linhas)+1)
		if tituloSecao := strings.TrimSpace(secao.Titulo); tituloSecao != "" {
			bloco = append(bloco, tituloSecao)
		}
		for i, linha := range secao.Linhas {
			item := fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(linha.Titulo))
			if descricao := strings.TrimSpace(linha.Descricao); descricao != "" {
				item += " - " + descricao
			}
			item += fmt.Sprintf(" [id=%s]", strings.TrimSpace(linha.ID))
			bloco = append(bloco, item)
		}
		partes = append(partes, strings.Join(bloco, "\n"))
	}
	if rodape := strings.TrimSpace(req.Rodape); rodape != "" {
		partes = append(partes, rodape)
	}
	return strings.Join(partes, "\n\n")
}

func contextInfoLista(req models.EnvioListaRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return &waE2E.ContextInfo{}
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
}

func (g *GerenciadorInstancias) resolverDestinosEnvio(ctx context.Context, client *whatsmeow.Client, chatJID, numero string, grupo bool) ([]types.JID, error) {
	jids, err := destinosJID(chatJID, numero, grupo)
	if err != nil || grupo || strings.TrimSpace(chatJID) != "" || client == nil {
		return jids, err
	}
	consultas := consultasNumeroWhatsApp(numero)
	if len(consultas) == 0 {
		return jids, nil
	}
	respostas, consultaErr := client.IsOnWhatsApp(ctx, consultas)
	if consultaErr != nil {
		return jids, nil
	}
	porConsulta := make(map[string]types.JID, len(respostas))
	for _, resposta := range respostas {
		if !resposta.IsIn || resposta.JID.User == "" {
			continue
		}
		porConsulta[resposta.Query] = resposta.JID
	}
	resolvidos := make([]types.JID, 0, len(consultas))
	visitados := make(map[string]struct{}, len(consultas))
	for _, consulta := range consultas {
		jid, ok := porConsulta[consulta]
		if !ok {
			continue
		}
		if _, existe := visitados[jid.String()]; existe {
			continue
		}
		visitados[jid.String()] = struct{}{}
		resolvidos = append(resolvidos, jid)
	}
	if len(resolvidos) > 0 {
		return resolvidos, nil
	}
	return jids, nil
}

func destinosJID(chatJID, numero string, grupo bool) ([]types.JID, error) {
	chatJID = strings.TrimSpace(chatJID)
	if chatJID != "" {
		jid, err := types.ParseJID(chatJID)
		if err != nil {
			return nil, fmt.Errorf("chat_jid invalido")
		}
		return []types.JID{jid}, nil
	}
	return jidsNumero(numero, grupo)
}

func jidsNumero(numero string, grupo bool) ([]types.JID, error) {
	normalizado := apenasDigitos.ReplaceAllString(numero, "")
	if normalizado == "" {
		return nil, fmt.Errorf("numero invalido")
	}
	if grupo {
		return []types.JID{types.NewJID(normalizado, types.GroupServer)}, nil
	}
	candidatos := candidatosNumeroBrasil(normalizado)
	jids := make([]types.JID, 0, len(candidatos))
	for _, candidato := range candidatos {
		jids = append(jids, types.NewJID(candidato, types.DefaultUserServer))
	}
	return jids, nil
}

func candidatosNumeroBrasil(numero string) []string {
	base, local := normalizarNumeroBrasil(numero)
	candidatos := make([]string, 0, 4)
	for _, candidato := range []string{base, local, alternarNonoDigito(base), alternarNonoDigito(local)} {
		if candidato != "" {
			candidatos = append(candidatos, candidato)
		}
	}
	return deduplicarStrings(candidatos)
}

func consultasNumeroWhatsApp(numero string) []string {
	base, local := normalizarNumeroBrasil(numero)
	consultas := make([]string, 0, 4)
	for _, candidato := range []string{base, local, alternarNonoDigito(base), alternarNonoDigito(local)} {
		if candidato == "" {
			continue
		}
		consultas = append(consultas, "+"+garantirCodigoPaisBrasil(candidato))
	}
	return deduplicarStrings(consultas)
}

func normalizarNumeroBrasil(numero string) (string, string) {
	normalizado := apenasDigitos.ReplaceAllString(numero, "")
	if strings.HasPrefix(normalizado, "55") && (len(normalizado) == 12 || len(normalizado) == 13) {
		return normalizado, strings.TrimPrefix(normalizado, "55")
	}
	if len(normalizado) == 10 || len(normalizado) == 11 {
		return "55" + normalizado, normalizado
	}
	return normalizado, strings.TrimPrefix(normalizado, "55")
}

func garantirCodigoPaisBrasil(numero string) string {
	base, _ := normalizarNumeroBrasil(numero)
	return base
}

func alternarNonoDigito(numero string) string {
	prefixo := ""
	nacional := numero
	if strings.HasPrefix(nacional, "55") {
		prefixo = "55"
		nacional = strings.TrimPrefix(nacional, "55")
	}
	if len(nacional) == 11 && nacional[2] == '9' {
		return prefixo + nacional[:2] + nacional[3:]
	}
	if len(nacional) == 10 {
		return prefixo + nacional[:2] + "9" + nacional[2:]
	}
	return ""
}

func deduplicarStrings(valores []string) []string {
	unicos := make([]string, 0, len(valores))
	visitados := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		if valor == "" {
			continue
		}
		if _, ok := visitados[valor]; ok {
			continue
		}
		visitados[valor] = struct{}{}
		unicos = append(unicos, valor)
	}
	return unicos
}
