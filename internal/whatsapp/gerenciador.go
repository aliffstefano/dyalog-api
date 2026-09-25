package whatsapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
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

	"dyalog-api-go/internal/chamadas"
	"dyalog-api-go/internal/models"
	mediastorage "dyalog-api-go/internal/storage"
	"dyalog-api-go/internal/store"
	webhookdispatch "dyalog-api-go/internal/webhook"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
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
	EditarTexto(ctx context.Context, req models.EditarTextoRequest) (models.ResultadoEnvio, error)
	ApagarMensagem(ctx context.Context, req models.ApagarMensagemRequest) (models.ResultadoEnvio, error)
	ReagirMensagem(ctx context.Context, req models.ReagirMensagemRequest) (models.ResultadoEnvio, error)
	EnviarPresenca(ctx context.Context, req models.EnvioPresencaRequest) (models.ResultadoPresenca, error)
	MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error)
	EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error)
	EnviarLista(ctx context.Context, req models.EnvioListaRequest) (models.ResultadoEnvio, error)
	EnviarEnquete(ctx context.Context, req models.EnvioEnqueteRequest) (models.ResultadoEnvio, error)
	EnviarCobrancaPix(ctx context.Context, req models.EnvioCobrancaPixRequest) (models.ResultadoEnvio, error)
	EnviarLocalizacao(ctx context.Context, req models.EnvioLocalizacaoRequest) (models.ResultadoEnvio, error)
	EnviarContato(ctx context.Context, req models.EnvioContatoRequest) (models.ResultadoEnvio, error)
	EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
	EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
	EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
	EnviarFigurinha(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error)
}

var ErrMidiaInvalida = errors.New("midia invalida")
var ErrAvatarNaoEncontrado = errors.New("avatar nao encontrado")
var ErrInstanciaPertenceOutroNode = errors.New("instancia pertence a outro container")

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
	client          *whatsmeow.Client
	container       *sqlstore.Container
	fecharContainer bool
	qrCancel        context.CancelFunc
	chamadas        map[string]*chamadaAtiva
}

func (r *runtimeInstancia) cancelarFluxoQR() {
	if r.qrCancel != nil {
		r.qrCancel()
		r.qrCancel = nil
	}
}

func (r *runtimeInstancia) fecharStore() {
	if r == nil || r.container == nil {
		return
	}
	if r.fecharContainer {
		_ = r.container.Close()
	}
	r.container = nil
}

type GerenciadorInstancias struct {
	mu              sync.RWMutex
	storeMu         sync.Mutex
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
	deviceStore     store.WhatsAppDeviceStore
	lockStore       store.InstanciaRuntimeLockStore
	nodeID          string
	nodeEndereco    string
	lockTTL         time.Duration
	aliasesNumero   map[string]string
	enquetesOpcoes  map[string][]string
	historicoDias   map[string]int
	recuperacoes    map[string]*janelaRecuperacao
	quedasConexao   map[string]time.Time
	// reconexaoBloqueada guarda ate quando uma instancia nao deve ser
	// reconectada automaticamente. Usado para banimento temporario e cliente
	// desatualizado, onde insistir so piora a situacao.
	reconexaoBloqueada map[string]time.Time

	whatsAppStoreDriver    string
	whatsAppStoreDSN       string
	whatsAppStoreContainer *sqlstore.Container

	configICE chamadas.ConfigICE

	recuperacaoWebhookHabilitada bool
	recuperacaoMargem            time.Duration
	recuperacaoQuantidade        int
}

type janelaRecuperacao struct {
	Inicio           time.Time
	Fim              time.Time
	Quantidade       int
	ChatsSolicitados map[string]bool
}

func NovoGerenciadorInstancias(diretorioBase, diretorioMidias, baseURL, nomeDispositivo, tipoCliente, nomePareamento, nivelLog, whatsAppStoreDriver, whatsAppStoreDSN, nodeID, nodeEndereco string, lockTTL time.Duration, dispatcher *webhookdispatch.Dispatcher, midiaStore store.MidiaStore, midiaUploader mediastorage.MidiaUploader, proxyStore store.ProxyConfigStore, mensagemStore store.MensagemProcessadaStore, deviceStore store.WhatsAppDeviceStore, lockStore store.InstanciaRuntimeLockStore) *GerenciadorInstancias {
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
	whatsAppStoreDriver = normalizarDriverStoreWhatsApp(whatsAppStoreDriver)
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		nodeID = "node-local"
	}
	if lockTTL < 15*time.Second {
		lockTTL = 90 * time.Second
	}
	configurarIdentidadeDispositivo(nomeDispositivo, tipoClientePareamento)
	return &GerenciadorInstancias{
		estados:            make(map[string]estadoRuntime),
		runtimes:           make(map[string]*runtimeInstancia),
		diretorioBase:      diretorioBase,
		diretorioMidias:    diretorioMidias,
		baseURL:            strings.TrimRight(baseURL, "/"),
		nomeDispositivo:    nomeDispositivo,
		tipoCliente:        tipoClientePareamento,
		nomePareamento:     nomePareamento,
		nivelLog:           nivelLog,
		dispatcher:         dispatcher,
		midiaStore:         midiaStore,
		midiaUploader:      midiaUploader,
		proxyStore:         proxyStore,
		mensagemStore:      mensagemStore,
		deviceStore:        deviceStore,
		lockStore:          lockStore,
		nodeID:             nodeID,
		nodeEndereco:       strings.TrimRight(strings.TrimSpace(nodeEndereco), "/"),
		lockTTL:            lockTTL,
		aliasesNumero:      make(map[string]string),
		enquetesOpcoes:     make(map[string][]string),
		historicoDias:      make(map[string]int),
		recuperacoes:       make(map[string]*janelaRecuperacao),
		quedasConexao:      make(map[string]time.Time),
		reconexaoBloqueada: make(map[string]time.Time),

		whatsAppStoreDriver: whatsAppStoreDriver,
		whatsAppStoreDSN:    strings.TrimSpace(whatsAppStoreDSN),
	}
}

func normalizarDriverStoreWhatsApp(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres", "postgresql", "pgx", "supabase":
		return "postgres"
	default:
		return "sqlite"
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

func (g *GerenciadorInstancias) ConfigurarRecuperacaoWebhook(habilitada bool, margem time.Duration, quantidade int) {
	if margem < 0 {
		margem = 0
	}
	if quantidade <= 0 {
		quantidade = 50
	}
	if quantidade > 200 {
		quantidade = 200
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recuperacaoWebhookHabilitada = habilitada
	g.recuperacaoMargem = margem
	g.recuperacaoQuantidade = quantidade
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

func (g *GerenciadorInstancias) registrarInicioIndisponibilidade(instanciaID string, inicio time.Time) {
	if instanciaID == "" || inicio.IsZero() {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.recuperacaoWebhookHabilitada {
		return
	}
	if atual, ok := g.quedasConexao[instanciaID]; ok && !atual.IsZero() {
		return
	}
	g.quedasConexao[instanciaID] = inicio.UTC()
}

func (g *GerenciadorInstancias) registrarFimIndisponibilidade(instanciaID string, fim time.Time) {
	if instanciaID == "" || fim.IsZero() {
		return
	}

	g.mu.Lock()
	if !g.recuperacaoWebhookHabilitada {
		delete(g.quedasConexao, instanciaID)
		g.mu.Unlock()
		return
	}
	inicio := g.quedasConexao[instanciaID]
	delete(g.quedasConexao, instanciaID)
	margem := g.recuperacaoMargem
	quantidade := g.recuperacaoQuantidade
	g.mu.Unlock()

	if inicio.IsZero() || !fim.After(inicio) {
		return
	}
	if margem > 0 {
		inicio = inicio.Add(-margem)
		fim = fim.Add(margem)
	}
	g.RegistrarJanelaRecuperacao(instanciaID, inicio, fim, quantidade)
	fmt.Printf("recuperacao de webhook agendada: instancia %s ficou sem conexao WhatsApp de %s a %s\n", instanciaID, inicio.UTC().Format(time.RFC3339), fim.UTC().Format(time.RFC3339))
}

func (g *GerenciadorInstancias) limparIndisponibilidade(instanciaID string) {
	if instanciaID == "" {
		return
	}
	g.mu.Lock()
	delete(g.quedasConexao, instanciaID)
	g.mu.Unlock()
}

func (g *GerenciadorInstancias) salvarDeviceJIDInstancia(ctx context.Context, instanciaID string, runtime *runtimeInstancia) {
	if g.whatsAppStoreDriver != "postgres" || g.deviceStore == nil || runtime == nil || runtime.client == nil || runtime.client.Store == nil || runtime.client.Store.ID == nil {
		return
	}
	if err := g.deviceStore.SalvarWhatsAppDeviceJID(ctx, instanciaID, runtime.client.Store.ID.String()); err != nil {
		fmt.Printf("erro ao salvar vinculo do device WhatsApp da instancia %s: %v\n", instanciaID, err)
	}
}

func (g *GerenciadorInstancias) excluirDeviceJIDInstancia(ctx context.Context, instanciaID string) {
	if g.whatsAppStoreDriver != "postgres" || g.deviceStore == nil {
		return
	}
	if err := g.deviceStore.ExcluirWhatsAppDeviceJID(ctx, instanciaID); err != nil {
		fmt.Printf("erro ao excluir vinculo do device WhatsApp da instancia %s: %v\n", instanciaID, err)
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

// PossuiRuntime informa se este processo tem a sessao da instancia viva em memoria.
// So o container dono mantem runtime, entao isso diz se o estado local vale alguma
// coisa ou se quem manda e o status salvo no banco.
func (g *GerenciadorInstancias) PossuiRuntime(instanciaID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, existe := g.runtimes[instanciaID]
	return existe
}

func (g *GerenciadorInstancias) RestaurarSessao(ctx context.Context, instanciaID string) (bool, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		// Quando a instancia pertence a outra replica, o estado local nao pode ser
		// marcado como desconectado: quem tem a sessao viva e o outro container, e
		// esse "desconectada" ficaria preso na memoria daqui para sempre, vazando
		// depois para o banco pela listagem de instancias.
		if !errors.Is(err, ErrInstanciaPertenceOutroNode) {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", err.Error())
		}
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
	g.confirmarConexaoRestaurada(instanciaID, runtime)
	return true, nil
}

// InstanciasReconectaveis devolve as instancias que cairam mas ainda tem sessao
// valida no store do WhatsApp, ou seja, as que voltam sozinhas sem pedir QR code.
//
// Ficam de fora, de proposito, as que dependem de acao humana: aguardando QR code,
// aguardando codigo de pareamento, sem dispositivo salvo (deslogadas pelo celular ou
// desconectadas manualmente) e as que estao com reconexao bloqueada por banimento
// temporario ou cliente desatualizado. Reconectar essas sozinho nao resolveria nada:
// o WhatsApp exige login novo.
func (g *GerenciadorInstancias) InstanciasReconectaveis() []string {
	agora := time.Now().UTC()
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := make([]string, 0, len(g.runtimes))
	for instanciaID, runtime := range g.runtimes {
		if runtime == nil || runtime.client == nil {
			continue
		}
		// Sem dispositivo salvo nao ha sessao para retomar: so um login novo resolve.
		if runtime.client.Store == nil || runtime.client.Store.ID == nil {
			continue
		}
		if runtime.client.IsConnected() {
			continue
		}
		if bloqueio, ok := g.reconexaoBloqueada[instanciaID]; ok && agora.Before(bloqueio) {
			continue
		}
		if !statusPermiteReconexaoAutomatica(g.estados[instanciaID].status) {
			continue
		}
		ids = append(ids, instanciaID)
	}
	return ids
}

// statusPermiteReconexaoAutomatica separa os estados em que a API pode reconectar
// por conta propria dos estados em que reconectar seria errado.
func statusPermiteReconexaoAutomatica(status string) bool {
	switch status {
	case models.StatusInstanciaAguardandoQR,
		models.StatusInstanciaAguardandoCodigo,
		models.StatusInstanciaNaoInicializada,
		models.StatusInstanciaConectando,
		models.StatusInstanciaDesconectando:
		return false
	default:
		return true
	}
}

// liberarBloqueioReconexao remove a suspensao apos a instancia voltar a conectar.
func (g *GerenciadorInstancias) liberarBloqueioReconexao(instanciaID string) {
	g.mu.Lock()
	delete(g.reconexaoBloqueada, instanciaID)
	g.mu.Unlock()
}

// bloquearReconexao suspende a reconexao automatica da instancia por um periodo.
func (g *GerenciadorInstancias) bloquearReconexao(instanciaID string, duracao time.Duration) {
	if duracao <= 0 {
		return
	}
	g.mu.Lock()
	g.reconexaoBloqueada[instanciaID] = time.Now().UTC().Add(duracao)
	g.mu.Unlock()
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
		g.confirmarConexaoRestaurada(instanciaID, runtime)
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

func (g *GerenciadorInstancias) confirmarConexaoRestaurada(instanciaID string, runtime *runtimeInstancia) {
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		<-timer.C
		if runtime == nil || runtime.client == nil {
			return
		}
		if runtime.client.IsConnected() && runtime.client.IsLoggedIn() {
			estado := g.obterEstado(instanciaID)
			if estado.status == "sincronizando_historico" || estado.status == models.StatusInstanciaConectando || estado.status == "autenticando" || estado.status == "pareada" {
				g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
			}
			_ = g.aplicarPresencaPersistente(context.Background(), instanciaID, runtime)
		}
	}()
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
	g.excluirDeviceJIDInstancia(ctx, instanciaID)
	g.liberarOwnership(ctx, instanciaID)
	g.definirEstado(instanciaID, models.StatusInstanciaNaoInicializada, "", "")
	return nil
}

func (g *GerenciadorInstancias) limparRuntime(instanciaID string) {
	g.limparRuntimeSemLiberar(instanciaID)
}

func (g *GerenciadorInstancias) limparRuntimeSemLiberar(instanciaID string) {
	g.mu.Lock()
	runtime := g.runtimes[instanciaID]
	delete(g.runtimes, instanciaID)
	g.mu.Unlock()
	if runtime == nil {
		return
	}
	runtime.cancelarFluxoQR()
	if runtime.client != nil && runtime.client.IsConnected() {
		runtime.client.Disconnect()
	}
	runtime.fecharStore()
}

func (g *GerenciadorInstancias) liberarOwnership(ctx context.Context, instanciaID string) {
	if g.lockStore == nil {
		return
	}
	if err := g.lockStore.LiberarInstancia(ctx, instanciaID, g.nodeID); err != nil {
		fmt.Printf("erro ao liberar ownership da instancia %s: %v\n", instanciaID, err)
	}
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
	runtime.fecharStore()
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
	delete(g.reconexaoBloqueada, instanciaID)
	g.mu.Unlock()

	if runtime != nil {
		runtime.cancelarFluxoQR()
		if runtime.client.IsConnected() {
			runtime.client.Disconnect()
		}
		if runtime.client.Store != nil && runtime.client.Store.ID != nil {
			_ = runtime.client.Store.Delete(ctx)
		}
		g.excluirDeviceJIDInstancia(ctx, instanciaID)
		runtime.fecharStore()
	}
	g.liberarOwnership(ctx, instanciaID)
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

func (g *GerenciadorInstancias) ConsultarAvatar(ctx context.Context, req models.ConsultaAvatarRequest) (models.AvatarContato, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.AvatarContato{}, err
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.AvatarContato{}, err
	}
	if len(jids) == 0 {
		return models.AvatarContato{}, fmt.Errorf("numero ou chat_jid invalido")
	}

	var ultimoErr error
	for _, jid := range jids {
		info, err := runtime.client.GetProfilePictureInfo(ctx, jid, &whatsmeow.GetProfilePictureParams{Preview: false})
		if err != nil {
			if errors.Is(err, whatsmeow.ErrProfilePictureNotSet) || errors.Is(err, whatsmeow.ErrProfilePictureUnauthorized) {
				ultimoErr = ErrAvatarNaoEncontrado
				continue
			}
			ultimoErr = err
			continue
		}
		if info == nil || strings.TrimSpace(info.URL) == "" {
			ultimoErr = ErrAvatarNaoEncontrado
			continue
		}
		return models.AvatarContato{
			Instancia: req.Instancia,
			Numero:    extrairNumeroJID(jid),
			ChatJID:   jid.String(),
			Grupo:     jid.Server == types.GroupServer,
			TemAvatar: true,
			AvatarID:  info.ID,
			AvatarURL: info.URL,
			Tipo:      info.Type,
		}, nil
	}
	if ultimoErr != nil {
		return models.AvatarContato{}, ultimoErr
	}
	return models.AvatarContato{}, ErrAvatarNaoEncontrado
}

func (g *GerenciadorInstancias) Info(ctx context.Context, instanciaID string) (InfoRuntime, error) {
	_ = ctx
	g.mu.RLock()
	runtime := g.runtimes[instanciaID]
	g.mu.RUnlock()
	estado := g.obterEstado(instanciaID)
	if runtime != nil && runtime.client.IsConnected() && runtime.client.IsLoggedIn() && estado.status != models.StatusInstanciaConectada {
		g.definirEstado(instanciaID, models.StatusInstanciaConectada, "", "")
		estado = g.obterEstado(instanciaID)
	} else if runtime != nil && !runtime.client.IsConnected() && statusTransitorioPersistente(estado.status) {
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

func (g *GerenciadorInstancias) EditarTexto(ctx context.Context, req models.EditarTextoRequest) (models.ResultadoEnvio, error) {
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

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	for _, jid := range jids {
		novoConteudo := &waE2E.Message{Conversation: proto.String(strings.TrimSpace(req.Mensagem))}
		msg := runtime.client.BuildEdit(jid, types.MessageID(strings.TrimSpace(req.MensagemID)), novoConteudo)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao editar mensagem no WhatsApp: %w", ultimoErro)
	}

	mensagemID := strings.TrimSpace(req.MensagemID)
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "editada",
		Tipo:       "texto",
		Observacao: "O WhatsApp permite editar apenas mensagens enviadas pela propria instancia e dentro da janela permitida pelo aplicativo.",
	}, nil
}

func (g *GerenciadorInstancias) ApagarMensagem(ctx context.Context, req models.ApagarMensagemRequest) (models.ResultadoEnvio, error) {
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

	sender := types.EmptyJID
	if texto := strings.TrimSpace(req.RemetenteJID); texto != "" {
		sender, err = types.ParseJID(texto)
		if err != nil {
			return models.ResultadoEnvio{}, fmt.Errorf("remetente_jid invalido")
		}
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	mensagemID := types.MessageID(strings.TrimSpace(req.MensagemID))
	for _, jid := range jids {
		msg := runtime.client.BuildRevoke(jid, sender, mensagemID)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao apagar mensagem no WhatsApp: %w", ultimoErro)
	}

	id := strings.TrimSpace(req.MensagemID)
	if id == "" {
		id = string(resp.ID)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: id,
		Status:     "apagada",
		Tipo:       "mensagem",
		Observacao: "O WhatsApp permite apagar mensagens dentro da janela permitida; para apagar mensagem de outra pessoa em grupo, a instancia precisa ser admin e receber remetente_jid.",
	}, nil
}

func (g *GerenciadorInstancias) ReagirMensagem(ctx context.Context, req models.ReagirMensagemRequest) (models.ResultadoEnvio, error) {
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

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	for _, jid := range jids {
		remetente, err := g.jidRemetenteReacao(jid, req)
		if err != nil {
			ultimoErro = err
			continue
		}
		msg := runtime.client.BuildReaction(jid, remetente, types.MessageID(strings.TrimSpace(req.MensagemID)), strings.TrimSpace(req.Emoji))
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao reagir mensagem no WhatsApp: %w", ultimoErro)
	}
	mensagemID := string(resp.ID)
	if mensagemID == "" {
		mensagemID = strings.TrimSpace(req.MensagemID)
	}
	status := "reagida"
	if strings.TrimSpace(req.Emoji) == "" {
		status = "reacao_removida"
	}
	return models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: status, Tipo: "reacao"}, nil
}

func (g *GerenciadorInstancias) jidRemetenteReacao(chat types.JID, req models.ReagirMensagemRequest) (types.JID, error) {
	if participante := strings.TrimSpace(req.RemetenteJID); participante != "" {
		jid, err := types.ParseJID(participante)
		if err == nil && !jid.IsEmpty() {
			return jid, nil
		}
		if numero := apenasDigitos.ReplaceAllString(participante, ""); numero != "" {
			return types.NewJID(numero, types.DefaultUserServer), nil
		}
		return types.EmptyJID, fmt.Errorf("remetente_jid invalido")
	}
	if chat.Server == types.GroupServer {
		return types.EmptyJID, fmt.Errorf("remetente_jid obrigatorio para reacao em grupo")
	}
	return chat, nil
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

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
		modoUsado  string
	)

	for _, tentativa := range montarTentativasBotoes(req) {
		msg, err := tentativa.montar()
		if err != nil {
			ultimoErro = err
			continue
		}
		for _, jid := range jids {
			resp, ultimoErro = enviarMensagemInterativa(ctx, runtime.client, jid, msg)
			if ultimoErro == nil {
				jidUsado = jid
				modoUsado = tentativa.modo
				break
			}
		}
		if jidUsado.User != "" {
			break
		}
		if !modoBotoesAuto(req) {
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
		Modo:       modoUsado,
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

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
		modoUsado  string
	)

	for _, tentativa := range montarTentativasLista(req) {
		msg := tentativa.montar()
		for _, jid := range jids {
			resp, ultimoErro = enviarMensagemInterativaComNome(ctx, runtime.client, jid, msg, req.FlowName)
			if ultimoErro == nil {
				jidUsado = jid
				modoUsado = tentativa.modo
				break
			}
		}
		if jidUsado.User != "" {
			break
		}
		if !modoListaAuto(req) {
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
		Modo:       modoUsado,
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

func (g *GerenciadorInstancias) EnviarEnquete(ctx context.Context, req models.EnvioEnqueteRequest) (models.ResultadoEnvio, error) {
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

	opcoes := make([]string, 0, len(req.Opcoes))
	for _, opcao := range req.Opcoes {
		opcoes = append(opcoes, strings.TrimSpace(opcao))
	}
	msg := runtime.client.BuildPollCreation(strings.TrimSpace(req.Nome), opcoes, req.OpcoesSelecionaveis)
	if contexto := contextInfoEnquete(req); contexto != nil && msg.PollCreationMessage != nil {
		msg.PollCreationMessage.ContextInfo = contexto
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
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar enquete no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	g.armazenarOpcoesEnquete(req.Instancia, mensagemID, opcoes)
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: mensagemID,
		Status:     "aceita_pelo_servidor",
		Tipo:       "enquete",
		Observacao: "O WhatsApp retornou ID para a enquete; o voto sera resolvido para o texto da opcao quando o destinatario responder.",
	}, nil
}

func contextInfoEnquete(req models.EnvioEnqueteRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
}

func (g *GerenciadorInstancias) armazenarOpcoesEnquete(instanciaID, mensagemID string, opcoes []string) {
	instanciaID = strings.TrimSpace(instanciaID)
	mensagemID = strings.TrimSpace(mensagemID)
	if instanciaID == "" || mensagemID == "" || len(opcoes) == 0 {
		return
	}
	copia := make([]string, len(opcoes))
	copy(copia, opcoes)
	chave := instanciaID + "|" + mensagemID
	g.mu.Lock()
	g.enquetesOpcoes[chave] = copia
	g.mu.Unlock()
}

func (g *GerenciadorInstancias) opcoesEnqueteConhecidas(instanciaID, mensagemID string) []string {
	chave := strings.TrimSpace(instanciaID) + "|" + strings.TrimSpace(mensagemID)
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.enquetesOpcoes[chave]
}

func (g *GerenciadorInstancias) resolverOpcoesVotoEnquete(instanciaID string, voto *waE2E.PollUpdateMessage, decodificado *waE2E.PollVoteMessage) []string {
	if decodificado == nil || len(decodificado.GetSelectedOptions()) == 0 {
		return nil
	}
	opcoesConhecidas := g.opcoesEnqueteConhecidas(instanciaID, voto.GetPollCreationMessageKey().GetID())
	if len(opcoesConhecidas) == 0 {
		return nil
	}
	hashes := whatsmeow.HashPollOptions(opcoesConhecidas)
	selecionadas := make([]string, 0, len(decodificado.GetSelectedOptions()))
	for _, hashVoto := range decodificado.GetSelectedOptions() {
		for i, hashOpcao := range hashes {
			if bytes.Equal(hashVoto, hashOpcao) {
				selecionadas = append(selecionadas, opcoesConhecidas[i])
				break
			}
		}
	}
	return selecionadas
}

func (g *GerenciadorInstancias) EnviarLocalizacao(ctx context.Context, req models.EnvioLocalizacaoRequest) (models.ResultadoEnvio, error) {
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

	localizacao := &waE2E.LocationMessage{
		DegreesLatitude:  proto.Float64(req.Latitude),
		DegreesLongitude: proto.Float64(req.Longitude),
		ContextInfo:      contextInfoLocalizacao(req),
	}
	if nome := strings.TrimSpace(req.Nome); nome != "" {
		localizacao.Name = proto.String(nome)
	}
	if endereco := strings.TrimSpace(req.Endereco); endereco != "" {
		localizacao.Address = proto.String(endereco)
	}
	msg := &waE2E.Message{LocationMessage: localizacao}

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
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar localizacao no WhatsApp: %w", ultimoErro)
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
		Tipo:       "localizacao",
	}, nil
}

func contextInfoLocalizacao(req models.EnvioLocalizacaoRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
}

func (g *GerenciadorInstancias) EnviarContato(ctx context.Context, req models.EnvioContatoRequest) (models.ResultadoEnvio, error) {
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

	contatos := make([]*waE2E.ContactMessage, 0, len(req.Contatos))
	for _, contato := range req.Contatos {
		nome := strings.TrimSpace(contato.Nome)
		vcard := strings.TrimSpace(contato.VCard)
		if vcard == "" {
			vcard = montarVCardContato(nome, contato.Telefone, contato.Organizacao)
		}
		if nome == "" {
			nome = nomeContatoDoVCard(vcard)
		}
		contatos = append(contatos, &waE2E.ContactMessage{
			DisplayName: proto.String(nome),
			Vcard:       proto.String(vcard),
		})
	}

	var msg *waE2E.Message
	tipo := "contato"
	if len(contatos) == 1 {
		contatos[0].ContextInfo = contextInfoContato(req)
		msg = &waE2E.Message{ContactMessage: contatos[0]}
	} else {
		tipo = "contatos"
		msg = &waE2E.Message{ContactsArrayMessage: &waE2E.ContactsArrayMessage{
			DisplayName: proto.String(fmt.Sprintf("%d contatos", len(contatos))),
			Contacts:    contatos,
			ContextInfo: contextInfoContato(req),
		}}
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
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar contato no WhatsApp: %w", ultimoErro)
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
		Tipo:       tipo,
	}, nil
}

func contextInfoContato(req models.EnvioContatoRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
}

var regexNaoDigito = regexp.MustCompile(`\D+`)

func montarVCardContato(nome, telefone, organizacao string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		nome = "Contato"
	}
	digitos := regexNaoDigito.ReplaceAllString(telefone, "")
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\n")
	b.WriteString("VERSION:3.0\n")
	fmt.Fprintf(&b, "N:;%s;;;\n", nome)
	fmt.Fprintf(&b, "FN:%s\n", nome)
	if organizacao = strings.TrimSpace(organizacao); organizacao != "" {
		fmt.Fprintf(&b, "ORG:%s;\n", organizacao)
	}
	if digitos != "" {
		fmt.Fprintf(&b, "TEL;type=CELL;type=VOICE;waid=%s:+%s\n", digitos, digitos)
	}
	b.WriteString("END:VCARD")
	return b.String()
}

func nomeContatoDoVCard(vcard string) string {
	for _, linha := range strings.Split(vcard, "\n") {
		linha = strings.TrimSpace(linha)
		if strings.HasPrefix(linha, "FN:") {
			return strings.TrimSpace(strings.TrimPrefix(linha, "FN:"))
		}
	}
	return "Contato"
}

// EnviarCobrancaPix envia o botao nativo de pagamento do WhatsApp ("Cobrar via
// Pix"), o mesmo native_flow (name="payment_info") que o app oficial usa. E um
// recurso da Meta atrelado a contas com Pagamentos habilitado; nao ha garantia
// de entrega/renderizacao fora do app oficial, entao aplicamos o mesmo fallback
// para texto usado em botoes/lista quando o servidor rejeita com 405.
func (g *GerenciadorInstancias) EnviarCobrancaPix(ctx context.Context, req models.EnvioCobrancaPixRequest) (models.ResultadoEnvio, error) {
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

	if req.FallbackTexto {
		return g.enviarCobrancaPixComoTexto(ctx, runtime.client, req, jids)
	}

	msg, err := montarMensagemCobrancaPix(req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}

	var (
		ultimoErro error
		resp       whatsmeow.SendResponse
		jidUsado   types.JID
	)
	for _, jid := range jids {
		resp, ultimoErro = enviarMensagemInterativaComNome(ctx, runtime.client, jid, msg, req.FlowName)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		if erroInterativoNaoPermitido(ultimoErro) {
			resultadoFallback, err := g.enviarCobrancaPixComoTexto(ctx, runtime.client, req, jids)
			if err == nil {
				resultadoFallback.Observacao = "Botao de cobranca Pix rejeitado pelo servidor do WhatsApp com erro 405; dados enviados automaticamente como texto."
				return resultadoFallback, nil
			}
		}
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar cobranca pix no WhatsApp: %w", ultimoErro)
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
		Tipo:       "cobranca_pix",
		Observacao: "O WhatsApp retornou ID para o botao de cobranca; a renderizacao do botao de pagamento depende da conta ter Pagamentos habilitado pela Meta.",
	}, nil
}

func (g *GerenciadorInstancias) enviarCobrancaPixComoTexto(ctx context.Context, client *whatsmeow.Client, req models.EnvioCobrancaPixRequest, jids []types.JID) (models.ResultadoEnvio, error) {
	tipoChave, err := normalizarTipoChavePixInterno(req.TipoChave)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	chave := normalizarChavePix(tipoChave, req.ChavePix)
	texto := textoFallbackCobrancaPix(req, chave)
	msg := &waE2E.Message{Conversation: proto.String(texto)}
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
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar cobranca pix como texto no WhatsApp: %w", ultimoErro)
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
		Tipo:       "cobranca_pix",
		Modo:       "texto",
		Observacao: "Cobranca enviada como texto (chave, valor e beneficiario) em vez de botao nativo de pagamento.",
	}, nil
}

func textoFallbackCobrancaPix(req models.EnvioCobrancaPixRequest, chaveFormatada string) string {
	var b strings.Builder
	descricao := strings.TrimSpace(req.Descricao)
	if descricao != "" {
		b.WriteString(descricao)
		b.WriteString("\n\n")
	}
	b.WriteString("Pagamento via Pix\n")
	fmt.Fprintf(&b, "Chave: %s\n", chaveFormatada)
	if strings.TrimSpace(req.NomeBeneficiario) != "" {
		fmt.Fprintf(&b, "Beneficiario: %s\n", strings.TrimSpace(req.NomeBeneficiario))
	}
	if req.Valor > 0 {
		fmt.Fprintf(&b, "Valor: R$ %.2f\n", req.Valor)
	}
	return strings.TrimRight(b.String(), "\n")
}

// normalizarTipoChavePixInterno espelha a validacao feita em MensagemService
// para o pacote whatsapp, que nao importa o pacote service (evita ciclo de
// import). A validacao real (com mensagem de erro amigavel) ja aconteceu antes
// de chegar aqui; isso e so uma segunda garantia.
func normalizarTipoChavePixInterno(tipo string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(tipo)) {
	case "telefone", "phone":
		return "PHONE", nil
	case "email", "e-mail":
		return "EMAIL", nil
	case "cpf":
		return "CPF", nil
	case "cnpj":
		return "CNPJ", nil
	case "aleatoria", "aleatória", "evp", "random":
		return "EVP", nil
	default:
		return "", fmt.Errorf("tipo_chave deve ser telefone, email, cpf, cnpj ou aleatoria")
	}
}

func normalizarChavePix(tipoChave, chave string) string {
	chave = strings.TrimSpace(chave)
	if tipoChave != "PHONE" {
		return chave
	}
	if strings.HasPrefix(chave, "+") {
		return chave
	}
	digitos := regexNaoDigito.ReplaceAllString(chave, "")
	if digitos == "" {
		return chave
	}
	if len(digitos) <= 11 {
		return "+55" + digitos
	}
	return "+" + digitos
}

func montarMensagemCobrancaPix(req models.EnvioCobrancaPixRequest) (*waE2E.Message, error) {
	tipoChave, err := normalizarTipoChavePixInterno(req.TipoChave)
	if err != nil {
		return nil, err
	}
	chave := normalizarChavePix(tipoChave, req.ChavePix)
	referencia := strings.ToUpper(strings.TrimSpace(req.Referencia))
	if referencia == "" {
		referencia = gerarReferenciaCobrancaPix()
	}
	retailerID := fmt.Sprintf("custom-item-%d", time.Now().UnixNano()%1_000_000_000)
	params := construirParamsCobrancaPix(req, tipoChave, chave, referencia, retailerID)
	payload, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar payload da cobranca pix: %w", err)
	}

	// Espelha o que o app oficial enviou na cobranca real capturada via webhook:
	// corpo vazio (o botao de pagamento e o conteudo) e MessageVersion nao
	// definido. Preencher esses campos e uma das diferenca entre o nosso envio e
	// o do cliente oficial, entao so mandamos corpo quando o usuario pedir
	// explicitamente uma descricao.
	nativeFlow := &waE2E.InteractiveMessage_NativeFlowMessage{
		Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{
			Name:             proto.String("payment_info"),
			ButtonParamsJSON: proto.String(string(payload)),
		}},
		MessageParamsJSON: proto.String(""),
	}
	interactive := &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: nativeFlow,
		},
		ContextInfo: contextInfoCobrancaPix(req),
	}
	if corpo := strings.TrimSpace(req.Descricao); corpo != "" {
		interactive.Body = &waE2E.InteractiveMessage_Body{Text: proto.String(corpo)}
	}
	return &waE2E.Message{
		InteractiveMessage: interactive,
		MessageContextInfo: contextInfoMensagemInterativa(),
	}, nil
}

// construirParamsCobrancaPix monta o JSON do botao "payment_info" no mesmo
// formato observado num webhook real de cobranca Pix enviada pelo app oficial do
// WhatsApp. Quando req.Valor <= 0, replica exatamente a estrutura "sem valor
// definido" (order_type ORDER_WITHOUT_AMOUNT, value 0, offset 1) vista nesse
// exemplo real. Quando ha valor, a codificacao amount.value/10^amount.offset e a
// convencao usada pelo catalogo do WhatsApp Business (offset 100 = centavos);
// esse caminho com valor nao foi confirmado contra um exemplo real, entao vale
// testar antes de depender dele em producao.
func construirParamsCobrancaPix(req models.EnvioCobrancaPixRequest, tipoChave, chave, referencia, retailerID string) map[string]interface{} {
	semValor := req.Valor <= 0
	var quantidade int
	var amount map[string]interface{}
	orderType := "ORDER"
	if semValor {
		orderType = "ORDER_WITHOUT_AMOUNT"
		quantidade = 0
		amount = map[string]interface{}{"value": 0, "offset": 1}
	} else {
		quantidade = 1
		amount = map[string]interface{}{"value": int64(req.Valor*100 + 0.5), "offset": 100}
	}
	item := map[string]interface{}{
		"retailer_id": retailerID,
		"name":        strings.TrimSpace(req.Descricao),
		"quantity":    quantidade,
		"amount":      amount,
	}
	return map[string]interface{}{
		"type":            "physical-goods",
		"currency":        "BRL",
		"reference_id":    referencia,
		"referral":        "chat_attachment",
		"additional_note": "",
		"total_amount":    amount,
		"order": map[string]interface{}{
			"status":     "payment_requested",
			"order_type": orderType,
			"items":      []interface{}{item},
			"subtotal":   amount,
		},
		"payment_settings": []interface{}{
			map[string]interface{}{
				"type": "pix_static_code",
				"pix_static_code": map[string]interface{}{
					// flow_type=APPSWITCH era o unico campo que o app oficial
					// enviava e nos nao. Sem ele a cobranca ate renderiza no
					// WhatsApp Web, mas o app mobile nao desenha o botao de
					// pagamento (ele indica que o fluxo abre o app de pagamento).
					"flow_type":     "APPSWITCH",
					"key":           chave,
					"key_type":      tipoChave,
					"merchant_name": strings.TrimSpace(req.NomeBeneficiario),
				},
			},
			map[string]interface{}{
				"type":  "cards",
				"cards": map[string]interface{}{"enabled": false},
			},
		},
	}
}

func gerarReferenciaCobrancaPix() string {
	bruta := strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	if len(bruta) > 11 {
		return bruta[:11]
	}
	return bruta
}

func contextInfoCobrancaPix(req models.EnvioCobrancaPixRequest) *waE2E.ContextInfo {
	if strings.TrimSpace(req.RespostaMensagemID) == "" && strings.TrimSpace(req.RespostaParticipante) == "" {
		return nil
	}
	return &waE2E.ContextInfo{
		StanzaID:    proto.String(strings.TrimSpace(req.RespostaMensagemID)),
		Participant: proto.String(strings.TrimSpace(req.RespostaParticipante)),
	}
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
		msg := montarMensagemAudio(reqEnvio, mimeType, upload, gerarWaveformAudio(dados))
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

func (g *GerenciadorInstancias) EnviarFigurinha(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterOuCriarRuntime(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return models.ResultadoEnvio{}, fmt.Errorf("instancia nao esta conectada ao WhatsApp")
	}

	dados, mimeType, err := carregarFigurinhaEnvio(ctx, req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	largura, altura := dimensoesImagem(dados)
	upload, err := runtime.client.Upload(ctx, dados, whatsmeow.MediaImage)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao fazer upload da figurinha: %w", err)
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
		msg := montarMensagemFigurinha(mimeType, upload, largura, altura)
		resp, ultimoErro = runtime.client.SendMessage(ctx, jid, msg)
		if ultimoErro == nil {
			jidUsado = jid
			break
		}
	}
	if jidUsado.User == "" {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar figurinha no WhatsApp: %w", ultimoErro)
	}
	mensagemID := req.MensagemID
	if mensagemID == "" {
		mensagemID = string(resp.ID)
	}
	return models.ResultadoEnvio{Instancia: req.Instancia, Numero: req.Numero, ChatJID: jidUsado.String(), MensagemID: mensagemID, Status: "enviada", Tipo: "figurinha"}, nil
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

func carregarFigurinhaEnvio(ctx context.Context, req models.EnvioMidiaRequest) ([]byte, string, error) {
	dados, nomeArquivo, mimeCabecalho, err := carregarConteudoMidia(ctx, req)
	if err != nil {
		return nil, "", err
	}
	mimeType := detectarMimeType(nomeArquivo, primeiroValorNaoVazio(req.MimeType, mimeCabecalho), dados)
	if mimeType != "image/webp" {
		return nil, "", fmt.Errorf("%w: figurinha precisa estar em formato webp; envie arquivo_base64, arquivo_url ou caminho_local com image/webp", ErrMidiaInvalida)
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
		ext := strings.ToLower(filepath.Ext(nomeArquivo))
		if ext == ".webp" {
			return "image/webp"
		}
		if mimeType := limparMimeType(mime.TypeByExtension(ext)); mimeType != "" {
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

func montarMensagemAudio(req models.EnvioMidiaRequest, mimeType string, upload whatsmeow.UploadResponse, waveform []byte) *waE2E.Message {
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
	if len(waveform) > 0 {
		audioMsg.Waveform = waveform
	}
	return &waE2E.Message{AudioMessage: audioMsg}
}

func gerarWaveformAudio(dados []byte) []byte {
	if len(dados) == 0 {
		return nil
	}
	const pontos = 64
	waveform := make([]byte, pontos)
	tamanhoBloco := (len(dados) + pontos - 1) / pontos
	for i := 0; i < pontos; i++ {
		inicio := i * tamanhoBloco
		if inicio >= len(dados) {
			break
		}
		fim := inicio + tamanhoBloco
		if fim > len(dados) {
			fim = len(dados)
		}
		var soma int
		for _, b := range dados[inicio:fim] {
			v := int(b)
			if v < 128 {
				soma += 128 - v
			} else {
				soma += v - 128
			}
		}
		media := soma / max(1, fim-inicio)
		valor := media * 100 / 128
		if valor > 100 {
			valor = 100
		}
		if valor < 3 {
			valor = 3
		}
		waveform[i] = byte(valor)
	}
	return waveform
}

func montarMensagemFigurinha(mimeType string, upload whatsmeow.UploadResponse, largura, altura uint32) *waE2E.Message {
	mediaKeyTimestamp := time.Now().Unix()
	stickerMsg := &waE2E.StickerMessage{
		Mimetype:          proto.String(mimeType),
		URL:               &upload.URL,
		DirectPath:        &upload.DirectPath,
		MediaKey:          upload.MediaKey,
		FileEncSHA256:     upload.FileEncSHA256,
		FileSHA256:        upload.FileSHA256,
		FileLength:        &upload.FileLength,
		MediaKeyTimestamp: &mediaKeyTimestamp,
	}
	if largura > 0 {
		stickerMsg.Width = &largura
	}
	if altura > 0 {
		stickerMsg.Height = &altura
	}
	return &waE2E.Message{StickerMessage: stickerMsg}
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
		if err := g.garantirOwnership(ctx, instanciaID); err != nil {
			return nil, err
		}
		return runtime, nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if runtime, ok = g.runtimes[instanciaID]; ok {
		return runtime, nil
	}
	if err := g.assumirOwnership(ctx, instanciaID); err != nil {
		return nil, err
	}
	ownershipAssumido := true
	dirInstancia := filepath.Join(g.diretorioBase, instanciaID)
	// O diretorio so serve ao store sqlite, que guarda o whatsmeow.db por
	// instancia. Com store postgres a sessao vive toda no banco e a pasta fica
	// vazia, entao exigir disco aqui transformava um detalhe do modo sqlite em
	// requisito de volume para todo mundo: sem o volume montado, nenhuma
	// instancia subia, mesmo sem precisar de disco para nada.
	if g.whatsAppStoreDriver != "postgres" {
		if err := os.MkdirAll(dirInstancia, 0o755); err != nil {
			if ownershipAssumido {
				g.liberarOwnership(context.Background(), instanciaID)
			}
			return nil, fmt.Errorf("erro ao criar diretorio da instancia: %w", err)
		}
	}
	container, deviceStore, fecharContainer, err := g.abrirStoreDispositivo(ctx, instanciaID, dirInstancia)
	if err != nil {
		if ownershipAssumido {
			g.liberarOwnership(context.Background(), instanciaID)
		}
		return nil, fmt.Errorf("erro ao abrir store do whatsmeow: %w", err)
	}
	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("WA-"+instanciaID, g.nivelLog, false))
	client.EnableAutoReconnect = true
	client.QRClientType = g.tipoCliente
	if err := g.aplicarProxyCliente(ctx, instanciaID, client); err != nil {
		if fecharContainer {
			_ = container.Close()
		}
		if ownershipAssumido {
			g.liberarOwnership(context.Background(), instanciaID)
		}
		return nil, err
	}
	runtime = &runtimeInstancia{client: client, container: container, fecharContainer: fecharContainer, chamadas: make(map[string]*chamadaAtiva)}
	client.AddEventHandler(func(evt interface{}) { g.tratarEvento(instanciaID, runtime, evt) })
	g.runtimes[instanciaID] = runtime
	if client.Store.ID != nil {
		g.estados[instanciaID] = estadoRuntime{status: models.StatusInstanciaDesconectada, atualizadoEm: time.Now().UTC()}
	} else {
		g.estados[instanciaID] = estadoRuntime{status: models.StatusInstanciaNaoInicializada, atualizadoEm: time.Now().UTC()}
	}
	ownershipAssumido = false
	return runtime, nil
}

func (g *GerenciadorInstancias) assumirOwnership(ctx context.Context, instanciaID string) error {
	if g.lockStore == nil {
		return nil
	}
	agora := time.Now().UTC()
	ok, err := g.lockStore.TentarAssumirInstancia(ctx, instanciaID, g.nodeID, g.nodeEndereco, agora, agora.Add(g.lockTTL))
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	dono, encontrado, err := g.lockStore.ObterDonoInstancia(ctx, instanciaID)
	if err != nil {
		return err
	}
	if encontrado {
		return fmt.Errorf("%w: instancia %s esta em uso por %s ate %s", ErrInstanciaPertenceOutroNode, instanciaID, dono.NodeID, dono.ExpiraEm.Format(time.RFC3339))
	}
	return fmt.Errorf("%w: instancia %s esta em uso por outro container", ErrInstanciaPertenceOutroNode, instanciaID)
}

// EnderecoOutroContainer informa o endereco da replica que detem o lock da instancia
// quando ela pertence a outro container e o lock ainda esta valido. Retorna string
// vazia quando a instancia e local, quando nao ha dono registrado, quando o lock ja
// expirou ou quando o dono nao registrou endereco alcancavel.
func (g *GerenciadorInstancias) EnderecoOutroContainer(ctx context.Context, instanciaID string) string {
	if g.lockStore == nil {
		return ""
	}
	instanciaID = strings.TrimSpace(instanciaID)
	if instanciaID == "" {
		return ""
	}
	dono, encontrado, err := g.lockStore.ObterDonoInstancia(ctx, instanciaID)
	if err != nil || !encontrado {
		return ""
	}
	if dono.NodeID == g.nodeID {
		return ""
	}
	if !dono.ExpiraEm.After(time.Now().UTC()) {
		return ""
	}
	endereco := strings.TrimRight(strings.TrimSpace(dono.Endereco), "/")
	if endereco == g.nodeEndereco {
		return ""
	}
	return endereco
}

func (g *GerenciadorInstancias) garantirOwnership(ctx context.Context, instanciaID string) error {
	if g.lockStore == nil {
		return nil
	}
	agora := time.Now().UTC()
	ok, err := g.lockStore.RenovarInstancia(ctx, instanciaID, g.nodeID, g.nodeEndereco, agora, agora.Add(g.lockTTL))
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	g.limparRuntimeSemLiberar(instanciaID)
	return fmt.Errorf("%w: instancia %s nao pertence mais a este container", ErrInstanciaPertenceOutroNode, instanciaID)
}

func (g *GerenciadorInstancias) IniciarRenovacaoOwnership(ctx context.Context, intervalo time.Duration) {
	if g.lockStore == nil {
		return
	}
	if intervalo < 5*time.Second {
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
				g.renovarOwnerships(context.Background())
			}
		}
	}()
}

func (g *GerenciadorInstancias) renovarOwnerships(ctx context.Context) {
	g.mu.RLock()
	ids := make([]string, 0, len(g.runtimes))
	for instanciaID := range g.runtimes {
		ids = append(ids, instanciaID)
	}
	g.mu.RUnlock()
	for _, instanciaID := range ids {
		if err := g.garantirOwnership(ctx, instanciaID); err != nil {
			fmt.Printf("ownership da instancia %s perdido: %v\n", instanciaID, err)
		}
	}
}

func (g *GerenciadorInstancias) abrirStoreDispositivo(ctx context.Context, instanciaID, dirInstancia string) (*sqlstore.Container, *waStore.Device, bool, error) {
	if g.whatsAppStoreDriver == "postgres" {
		container, err := g.obterContainerWhatsAppPostgres(ctx)
		if err != nil {
			return nil, nil, false, err
		}
		deviceStore, err := g.obterDeviceStorePostgres(ctx, instanciaID, container)
		if err != nil {
			return nil, nil, false, err
		}
		return container, deviceStore, false, nil
	}

	container, err := abrirStoreWhatsmeowSQLite(ctx, instanciaID, filepath.Join(dirInstancia, "whatsmeow.db"), g.nivelLog)
	if err != nil {
		return nil, nil, false, err
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		_ = container.Close()
		return nil, nil, false, fmt.Errorf("erro ao obter device store: %w", err)
	}
	return container, deviceStore, true, nil
}

func abrirStoreWhatsmeowSQLite(ctx context.Context, instanciaID, caminhoBanco, nivelLog string) (*sqlstore.Container, error) {
	dsn := fmt.Sprintf(
		"file:%s?_foreign_keys=on&_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL",
		filepath.ToSlash(caminhoBanco),
	)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	container := sqlstore.NewWithDB(db, "sqlite3", waLog.Stdout("WA-DB-"+instanciaID, nivelLog, false))
	if err := container.Upgrade(ctx); err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("erro ao atualizar schema do banco: %w", err)
	}
	return container, nil
}

func (g *GerenciadorInstancias) obterContainerWhatsAppPostgres(ctx context.Context) (*sqlstore.Container, error) {
	g.storeMu.Lock()
	defer g.storeMu.Unlock()
	if g.whatsAppStoreContainer != nil {
		return g.whatsAppStoreContainer, nil
	}
	if strings.TrimSpace(g.whatsAppStoreDSN) == "" {
		return nil, fmt.Errorf("WHATSAPP_STORE_DSN nao configurado para store postgres do whatsmeow")
	}
	db, err := sql.Open("pgx", g.whatsAppStoreDSN)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir postgres do whatsmeow: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	container := sqlstore.NewWithDB(db, "pgx", waLog.Stdout("WA-DB-POSTGRES", g.nivelLog, false))
	if err := container.Upgrade(ctx); err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("erro ao atualizar schema postgres do whatsmeow: %w", err)
	}
	g.whatsAppStoreContainer = container
	return container, nil
}

func (g *GerenciadorInstancias) obterDeviceStorePostgres(ctx context.Context, instanciaID string, container *sqlstore.Container) (*waStore.Device, error) {
	if g.deviceStore == nil {
		return nil, fmt.Errorf("store de vinculo WhatsApp nao configurado")
	}
	deviceJID, encontrado, err := g.deviceStore.ObterWhatsAppDeviceJID(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	if encontrado && strings.TrimSpace(deviceJID) != "" {
		jid, err := types.ParseJID(deviceJID)
		if err != nil {
			_ = g.deviceStore.ExcluirWhatsAppDeviceJID(context.Background(), instanciaID)
			return container.NewDevice(), nil
		}
		deviceStore, err := container.GetDevice(ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("erro ao obter device store postgres: %w", err)
		}
		if deviceStore != nil {
			return deviceStore, nil
		}
		_ = g.deviceStore.ExcluirWhatsAppDeviceJID(context.Background(), instanciaID)
	}
	return container.NewDevice(), nil
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
		g.liberarBloqueioReconexao(instanciaID)
		g.salvarDeviceJIDInstancia(context.Background(), instanciaID, runtime)
		g.registrarFimIndisponibilidade(instanciaID, time.Now().UTC())
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
			g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "")
		} else if estado.status == models.StatusInstanciaAguardandoCodigo && estado.pairingCode != "" {
			g.definirEstadoCodigoPareamento(instanciaID, models.StatusInstanciaAguardandoCodigo, estado.pairingCode, estado.pairingPhone, "")
		} else {
			g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, estado.qrCode, "")
		}
	// StreamReplaced, ConnectFailure e StreamError fecham o socket sem emitir
	// events.Disconnected e sem disparar o auto-reconnect do whatsmeow. Sem tratar
	// aqui, a instancia ficava marcada como conectada enquanto ja estava morta, e so
	// voltava com um conectar manual.
	case *events.StreamReplaced:
		g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "sessao assumida por outra conexao")
	case *events.ConnectFailure:
		g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
		mensagem := strings.TrimSpace(evento.Message)
		if mensagem == "" {
			mensagem = evento.Reason.String()
		}
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "falha de conexao: "+mensagem)
	case *events.StreamError:
		g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "erro de stream: "+evento.Code)
	// Banimento temporario e cliente desatualizado nao se resolvem reconectando:
	// insistir aumenta o problema, entao a reconexao automatica fica suspensa.
	case *events.TemporaryBan:
		espera := evento.Expire
		if espera <= 0 {
			espera = time.Hour
		}
		g.bloquearReconexao(instanciaID, espera)
		g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "banimento temporario do WhatsApp: "+evento.String())
	case *events.ClientOutdated:
		g.bloquearReconexao(instanciaID, time.Hour)
		g.registrarInicioIndisponibilidade(instanciaID, time.Now().UTC())
		g.definirEstado(instanciaID, models.StatusInstanciaDesconectada, "", "versao do cliente desatualizada: atualize o whatsmeow")
	case *events.KeepAliveTimeout:
		inicio := evento.LastSuccess
		if inicio.IsZero() {
			inicio = time.Now().UTC()
		}
		g.registrarInicioIndisponibilidade(instanciaID, inicio)
	case *events.KeepAliveRestored:
		g.registrarFimIndisponibilidade(instanciaID, time.Now().UTC())
	case *events.LoggedOut:
		runtime.cancelarFluxoQR()
		g.limparIndisponibilidade(instanciaID)
		_ = runtime.client.Store.Delete(context.Background())
		g.excluirDeviceJIDInstancia(context.Background(), instanciaID)
		g.liberarOwnership(context.Background(), instanciaID)
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
		if evento.Message.GetPollUpdateMessage() != nil {
			fmt.Printf("enquete: evento de voto recebido na instancia %s (mensagem_id=%s, remetente=%s, from_me=%v)\n", instanciaID, evento.Info.ID, evento.Info.Sender.String(), evento.Info.IsFromMe)
		}
		configuracao := g.obterConfiguracaoInstancia(context.Background(), instanciaID)
		if g.deveIgnorarMensagem(configuracao, evento) {
			if evento.Message.GetPollUpdateMessage() != nil {
				fmt.Printf("enquete: voto ignorado por deveIgnorarMensagem na instancia %s (mensagem_id=%s)\n", instanciaID, evento.Info.ID)
			}
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
	acao, detalhesAcao := classificarAcaoMensagem(evento)
	if acao == "" {
		acao = "recebida"
	}
	if acao == "apagada" {
		historico = false
	}
	if g.mensagemStore != nil && evento.Info.ID != "" && acao == "recebida" {
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
	if tipo == "enquete" {
		if enquete := extrairCriacaoEnqueteMensagem(evento.Message); enquete != nil {
			opcoes := make([]string, 0, len(enquete.GetOptions()))
			for _, opcao := range enquete.GetOptions() {
				if opcao != nil {
					opcoes = append(opcoes, strings.TrimSpace(opcao.GetOptionName()))
				}
			}
			g.armazenarOpcoesEnquete(instanciaID, string(evento.Info.ID), opcoes)
		}
	}
	if tipo == "enquete_voto" {
		conteudo, extras, extrasMensagem = g.resolverVotoEnqueteWebhook(instanciaID, client, evento, conteudo, extras, extrasMensagem)
	}
	if acao == "apagada" {
		conteudo = "mensagem apagada"
		tipo = "apagada"
		extras = nil
		extrasMensagem = nil
	}
	if tipo == "ignorada" {
		return
	}
	mensagemID := string(evento.Info.ID)
	if idApagada, ok := detalhesAcao["mensagem_apagada_id"].(string); ok && strings.TrimSpace(idApagada) != "" {
		mensagemID = strings.TrimSpace(idApagada)
	}
	dados := map[string]interface{}{
		"mensagem_id":      mensagemID,
		"chat_jid":         evento.Info.Chat.String(),
		"chat_numero":      chatNumero,
		"grupo":            evento.Info.IsGroup,
		"enviado_por_mim":  evento.Info.IsFromMe,
		"direcao":          direcao,
		"acao":             acao,
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
			"id":              mensagemID,
			"tipo":            tipo,
			"acao":            acao,
			"conteudo":        conteudo,
			"direcao":         direcao,
			"enviado_por_mim": evento.Info.IsFromMe,
			"origem":          origem,
			"historico":       historico,
			"recebida_em":     evento.Info.Timestamp.UTC(),
		},
	}
	for chave, valor := range detalhesAcao {
		dados[chave] = valor
	}
	for chave, valor := range extras {
		dados[chave] = valor
	}
	mensagemDados, _ := dados["mensagem"].(map[string]interface{})
	for chave, valor := range detalhesAcao {
		mensagemDados[chave] = valor
	}
	for chave, valor := range extrasMensagem {
		mensagemDados[chave] = valor
	}
	if acao != "apagada" {
		midia, err := g.processarMidiaRecebida(instanciaID, client, evento)
		if err != nil {
			dados["midia_erro"] = err.Error()
			mensagemDados["midia_erro"] = err.Error()
		} else if midia != nil {
			g.anexarMidiaRecebidaAoPayload(dados, *midia)
		}
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
	if video := msg.GetPtvMessage(); video != nil {
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

// extrairConteudoMensagem classifica o conteudo da mensagem e anexa, quando presente,
// os dados de citacao/resposta (equivalentes de entrada dos campos resposta_mensagem_id,
// resposta_participante e resposta_conteudo usados no envio).
func extrairConteudoMensagem(msg *waE2E.Message) (string, string, map[string]interface{}, map[string]interface{}) {
	conteudo, tipo, extras, extrasMensagem := extrairConteudoMensagemInterno(msg)
	if resposta := extrairRespostaCitada(msg); resposta != nil {
		if extras == nil {
			extras = map[string]interface{}{}
		}
		extras["resposta_mensagem_id"] = resposta["mensagem_id"]
		extras["resposta_participante"] = resposta["participante"]
		extras["resposta_conteudo"] = resposta["conteudo"]
		extras["resposta_tipo"] = resposta["tipo"]
		if extrasMensagem == nil {
			extrasMensagem = map[string]interface{}{}
		}
		extrasMensagem["resposta"] = resposta
	}
	return conteudo, tipo, extras, extrasMensagem
}

// extrairRespostaCitada le o ContextInfo da mensagem (quando o cliente respondeu
// citando uma mensagem anterior) e retorna os dados dessa citacao, incluindo o
// conteudo resumido da mensagem original quando o WhatsApp o envia embutido.
func extrairRespostaCitada(msg *waE2E.Message) map[string]interface{} {
	msg = normalizarMensagemConteudo(msg)
	if msg == nil {
		return nil
	}
	contexto := contextInfoDaMensagem(msg)
	if contexto == nil {
		return nil
	}
	stanzaID := strings.TrimSpace(contexto.GetStanzaID())
	if stanzaID == "" {
		return nil
	}
	conteudoCitado := ""
	tipoCitado := ""
	if citada := contexto.GetQuotedMessage(); citada != nil {
		conteudoCitado, tipoCitado, _, _ = extrairConteudoMensagemInterno(citada)
	}
	return map[string]interface{}{
		"mensagem_id":  stanzaID,
		"participante": strings.TrimSpace(contexto.GetParticipant()),
		"conteudo":     conteudoCitado,
		"tipo":         tipoCitado,
	}
}

// contextInfoDaMensagem devolve o ContextInfo do tipo de mensagem concreto, ja que
// o protobuf nao oferece um getter agregado em waE2E.Message para isso.
func contextInfoDaMensagem(msg *waE2E.Message) *waE2E.ContextInfo {
	switch {
	case msg.GetExtendedTextMessage() != nil:
		return msg.GetExtendedTextMessage().GetContextInfo()
	case msg.GetImageMessage() != nil:
		return msg.GetImageMessage().GetContextInfo()
	case msg.GetVideoMessage() != nil:
		return msg.GetVideoMessage().GetContextInfo()
	case msg.GetAudioMessage() != nil:
		return msg.GetAudioMessage().GetContextInfo()
	case msg.GetDocumentMessage() != nil:
		return msg.GetDocumentMessage().GetContextInfo()
	case msg.GetStickerMessage() != nil:
		return msg.GetStickerMessage().GetContextInfo()
	case msg.GetContactMessage() != nil:
		return msg.GetContactMessage().GetContextInfo()
	case msg.GetContactsArrayMessage() != nil:
		return msg.GetContactsArrayMessage().GetContextInfo()
	case msg.GetLocationMessage() != nil:
		return msg.GetLocationMessage().GetContextInfo()
	case msg.GetLiveLocationMessage() != nil:
		return msg.GetLiveLocationMessage().GetContextInfo()
	case msg.GetButtonsMessage() != nil:
		return msg.GetButtonsMessage().GetContextInfo()
	case msg.GetButtonsResponseMessage() != nil:
		return msg.GetButtonsResponseMessage().GetContextInfo()
	case msg.GetListMessage() != nil:
		return msg.GetListMessage().GetContextInfo()
	case msg.GetListResponseMessage() != nil:
		return msg.GetListResponseMessage().GetContextInfo()
	case msg.GetInteractiveMessage() != nil:
		return msg.GetInteractiveMessage().GetContextInfo()
	case msg.GetInteractiveResponseMessage() != nil:
		return msg.GetInteractiveResponseMessage().GetContextInfo()
	case msg.GetTemplateMessage() != nil:
		return msg.GetTemplateMessage().GetContextInfo()
	case msg.GetTemplateButtonReplyMessage() != nil:
		return msg.GetTemplateButtonReplyMessage().GetContextInfo()
	case msg.GetGroupInviteMessage() != nil:
		return msg.GetGroupInviteMessage().GetContextInfo()
	case msg.GetProductMessage() != nil:
		return msg.GetProductMessage().GetContextInfo()
	case msg.GetOrderMessage() != nil:
		return msg.GetOrderMessage().GetContextInfo()
	case msg.GetPollCreationMessage() != nil:
		return msg.GetPollCreationMessage().GetContextInfo()
	default:
		return nil
	}
}

func extrairConteudoMensagemInterno(msg *waE2E.Message) (string, string, map[string]interface{}, map[string]interface{}) {
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
	if video := msg.GetPtvMessage(); video != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(video.GetMimetype()), "ptv": true}
		return strings.TrimSpace(video.GetCaption()), "video", extras, map[string]interface{}{"mime_type": extras["mime_type"], "ptv": true}
	}
	if sticker := msg.GetStickerMessage(); sticker != nil {
		extras := map[string]interface{}{"mime_type": limparMimeType(sticker.GetMimetype())}
		return "sticker recebido", "sticker", extras, map[string]interface{}{"mime_type": extras["mime_type"]}
	}
	if reacao := msg.GetReactionMessage(); reacao != nil {
		return extrairReacaoMensagem(reacao)
	}
	if enquete := extrairCriacaoEnqueteMensagem(msg); enquete != nil {
		return extrairEnqueteMensagem(enquete)
	}
	if voto := msg.GetPollUpdateMessage(); voto != nil {
		return extrairVotoEnqueteMensagem(voto)
	}
	if contato := msg.GetContactMessage(); contato != nil {
		return extrairContatoMensagem(contato)
	}
	if contatos := msg.GetContactsArrayMessage(); contatos != nil {
		return extrairContatosMensagem(contatos)
	}
	if localizacao := msg.GetLocationMessage(); localizacao != nil {
		return extrairLocalizacaoMensagem(localizacao)
	}
	if localizacao := msg.GetLiveLocationMessage(); localizacao != nil {
		return extrairLocalizacaoTempoRealMensagem(localizacao)
	}
	if botoes := msg.GetButtonsMessage(); botoes != nil {
		return extrairBotoesMensagem(botoes)
	}
	if lista := msg.GetListMessage(); lista != nil {
		return extrairListaMensagem(lista)
	}
	if interativo := msg.GetInteractiveMessage(); interativo != nil {
		return extrairInterativoMensagem(interativo)
	}
	if produto := msg.GetProductMessage(); produto != nil {
		return extrairProdutoMensagem(produto)
	}
	if pedido := msg.GetOrderMessage(); pedido != nil {
		return extrairPedidoMensagem(pedido)
	}
	if convite := msg.GetGroupInviteMessage(); convite != nil {
		return extrairConviteGrupoMensagem(convite)
	}
	if chamada := msg.GetScheduledCallCreationMessage(); chamada != nil {
		return extrairChamadaAgendadaMensagem(chamada)
	}
	if msg.GetRequestPhoneNumberMessage() != nil {
		return "solicitacao de telefone recebida", "solicitacao_telefone", nil, nil
	}
	if conteudo, tipo, extras, extrasMensagem, ok := extrairTipoMensagemNaoMapeada(msg); ok {
		return conteudo, tipo, extras, extrasMensagem
	}
	return "mensagem recebida", "desconhecido", nil, nil
}

type chaveMensagemWebhook interface {
	GetID() string
	GetRemoteJID() string
	GetFromMe() bool
	GetParticipant() string
}

func extrairReacaoMensagem(reacao *waE2E.ReactionMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	texto := strings.TrimSpace(reacao.GetText())
	conteudo := texto
	if conteudo == "" {
		conteudo = "reacao removida"
	}
	reacaoDados := map[string]interface{}{
		"texto":    texto,
		"removida": texto == "",
	}
	anexarChaveMensagemWebhook(reacaoDados, "mensagem", reacao.GetKey())
	if agrupamento := strings.TrimSpace(reacao.GetGroupingKey()); agrupamento != "" {
		reacaoDados["agrupamento"] = agrupamento
	}
	if ts := reacao.GetSenderTimestampMS(); ts > 0 {
		reacaoDados["enviada_em"] = time.UnixMilli(ts).UTC()
	}

	extras := map[string]interface{}{
		"reacao_texto":    texto,
		"reacao_removida": texto == "",
	}
	anexarChaveMensagemWebhook(extras, "mensagem_reagida", reacao.GetKey())
	if agrupamento := strings.TrimSpace(reacao.GetGroupingKey()); agrupamento != "" {
		extras["reacao_agrupamento"] = agrupamento
	}
	if ts := reacao.GetSenderTimestampMS(); ts > 0 {
		extras["reacao_enviada_em"] = time.UnixMilli(ts).UTC()
	}
	return conteudo, "reacao", extras, map[string]interface{}{"reacao": reacaoDados}
}

func extrairCriacaoEnqueteMensagem(msg *waE2E.Message) *waE2E.PollCreationMessage {
	switch {
	case msg.GetPollCreationMessage() != nil:
		return msg.GetPollCreationMessage()
	case msg.GetPollCreationMessageV2() != nil:
		return msg.GetPollCreationMessageV2()
	case msg.GetPollCreationMessageV3() != nil:
		return msg.GetPollCreationMessageV3()
	case msg.GetPollCreationMessageV5() != nil:
		return msg.GetPollCreationMessageV5()
	case msg.GetPollCreationMessageV6() != nil:
		return msg.GetPollCreationMessageV6()
	case msg.GetPollCreationMessageV4() != nil:
		return extrairCriacaoEnqueteMensagem(normalizarMensagemConteudo(msg.GetPollCreationMessageV4().GetMessage()))
	default:
		return nil
	}
}

func extrairEnqueteMensagem(enquete *waE2E.PollCreationMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	nome := strings.TrimSpace(enquete.GetName())
	conteudo := nome
	if conteudo == "" {
		conteudo = "enquete recebida"
	}
	opcoes := opcoesEnqueteWebhook(enquete.GetOptions())
	enqueteDados := map[string]interface{}{
		"nome":                        nome,
		"opcoes":                      opcoes,
		"quantidade_opcoes":           len(opcoes),
		"opcoes_selecionaveis":        enquete.GetSelectableOptionsCount(),
		"tipo":                        enquete.GetPollType().String(),
		"conteudo_tipo":               enquete.GetPollContentType().String(),
		"permite_adicionar_opcao":     enquete.GetAllowAddOption(),
		"oculta_nome_do_participante": enquete.GetHideParticipantName(),
	}
	if fim := enquete.GetEndTime(); fim > 0 {
		enqueteDados["termina_em"] = time.Unix(fim, 0).UTC()
	}
	extras := map[string]interface{}{
		"enquete_nome":                        nome,
		"enquete_opcoes":                      opcoes,
		"enquete_quantidade_opcoes":           len(opcoes),
		"enquete_opcoes_selecionaveis":        enquete.GetSelectableOptionsCount(),
		"enquete_tipo":                        enquete.GetPollType().String(),
		"enquete_conteudo_tipo":               enquete.GetPollContentType().String(),
		"enquete_permite_adicionar_opcao":     enquete.GetAllowAddOption(),
		"enquete_oculta_nome_do_participante": enquete.GetHideParticipantName(),
	}
	if fim := enquete.GetEndTime(); fim > 0 {
		extras["enquete_termina_em"] = time.Unix(fim, 0).UTC()
	}
	return conteudo, "enquete", extras, map[string]interface{}{"enquete": enqueteDados}
}

func opcoesEnqueteWebhook(opcoes []*waE2E.PollCreationMessage_Option) []map[string]interface{} {
	resultado := make([]map[string]interface{}, 0, len(opcoes))
	for _, opcao := range opcoes {
		if opcao == nil {
			continue
		}
		item := map[string]interface{}{"nome": strings.TrimSpace(opcao.GetOptionName())}
		if hash := strings.TrimSpace(opcao.GetOptionHash()); hash != "" {
			item["hash"] = hash
		}
		resultado = append(resultado, item)
	}
	return resultado
}

func extrairVotoEnqueteMensagem(voto *waE2E.PollUpdateMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	votoDados := map[string]interface{}{
		"criptografado": voto.GetVote() != nil,
	}
	anexarChaveMensagemWebhook(votoDados, "enquete", voto.GetPollCreationMessageKey())
	if ts := voto.GetSenderTimestampMS(); ts > 0 {
		votoDados["enviado_em"] = time.UnixMilli(ts).UTC()
	}
	extras := map[string]interface{}{
		"enquete_voto_criptografado": voto.GetVote() != nil,
	}
	anexarChaveMensagemWebhook(extras, "enquete", voto.GetPollCreationMessageKey())
	if ts := voto.GetSenderTimestampMS(); ts > 0 {
		extras["enquete_voto_enviado_em"] = time.UnixMilli(ts).UTC()
	}
	return "voto em enquete recebido", "enquete_voto", extras, map[string]interface{}{"enquete_voto": votoDados}
}

func (g *GerenciadorInstancias) resolverVotoEnqueteWebhook(instanciaID string, client *whatsmeow.Client, evento *events.Message, conteudo string, extras map[string]interface{}, extrasMensagem map[string]interface{}) (string, map[string]interface{}, map[string]interface{}) {
	voto := evento.Message.GetPollUpdateMessage()
	if voto == nil || client == nil {
		return conteudo, extras, extrasMensagem
	}
	decodificado, err := client.DecryptPollVote(context.Background(), evento)
	if err != nil {
		fmt.Printf("enquete: falha ao decriptar voto na instancia %s (mensagem_original=%s, remetente=%s): %v\n", instanciaID, voto.GetPollCreationMessageKey().GetID(), evento.Info.Sender.String(), err)
		return conteudo, extras, extrasMensagem
	}
	opcoesSelecionadas := g.resolverOpcoesVotoEnquete(instanciaID, voto, decodificado)
	if len(opcoesSelecionadas) == 0 {
		fmt.Printf("enquete: voto decriptado mas opcoes nao resolvidas na instancia %s (mensagem_original=%s, remetente=%s, opcoes_conhecidas=%d, hashes_votados=%d)\n", instanciaID, voto.GetPollCreationMessageKey().GetID(), evento.Info.Sender.String(), len(g.opcoesEnqueteConhecidas(instanciaID, voto.GetPollCreationMessageKey().GetID())), len(decodificado.GetSelectedOptions()))
		return conteudo, extras, extrasMensagem
	}
	conteudo = strings.Join(opcoesSelecionadas, ", ")
	if extras == nil {
		extras = map[string]interface{}{}
	}
	extras["enquete_voto_opcoes"] = opcoesSelecionadas
	votoDados, _ := extrasMensagem["enquete_voto"].(map[string]interface{})
	if votoDados == nil {
		votoDados = map[string]interface{}{}
	}
	votoDados["opcoes_selecionadas"] = opcoesSelecionadas
	votoDados["criptografado"] = false
	if extrasMensagem == nil {
		extrasMensagem = map[string]interface{}{}
	}
	extrasMensagem["enquete_voto"] = votoDados
	return conteudo, extras, extrasMensagem
}

func extrairContatoMensagem(contato *waE2E.ContactMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	contatoDados := contatoWebhook(contato)
	conteudo, _ := contatoDados["nome"].(string)
	if conteudo == "" {
		conteudo = "contato recebido"
	}
	extras := map[string]interface{}{
		"contato_nome":    contatoDados["nome"],
		"contato_vcard":   contatoDados["vcard"],
		"contato_proprio": contatoDados["proprio"],
	}
	return conteudo, "contato", extras, map[string]interface{}{"contato": contatoDados}
}

func extrairContatosMensagem(contatos *waE2E.ContactsArrayMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	itens := make([]map[string]interface{}, 0, len(contatos.GetContacts()))
	for _, contato := range contatos.GetContacts() {
		if contato == nil {
			continue
		}
		itens = append(itens, contatoWebhook(contato))
	}
	nome := strings.TrimSpace(contatos.GetDisplayName())
	conteudo := nome
	if conteudo == "" {
		conteudo = fmt.Sprintf("%d contatos recebidos", len(itens))
	}
	extras := map[string]interface{}{
		"contatos_nome":       nome,
		"contatos":            itens,
		"contatos_quantidade": len(itens),
	}
	return conteudo, "contatos", extras, map[string]interface{}{"contatos": itens}
}

func contatoWebhook(contato *waE2E.ContactMessage) map[string]interface{} {
	return map[string]interface{}{
		"nome":    strings.TrimSpace(contato.GetDisplayName()),
		"vcard":   strings.TrimSpace(contato.GetVcard()),
		"proprio": contato.GetIsSelfContact(),
	}
}

func extrairLocalizacaoMensagem(localizacao *waE2E.LocationMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	localizacaoDados := map[string]interface{}{
		"latitude":        localizacao.GetDegreesLatitude(),
		"longitude":       localizacao.GetDegreesLongitude(),
		"nome":            strings.TrimSpace(localizacao.GetName()),
		"endereco":        strings.TrimSpace(localizacao.GetAddress()),
		"url":             strings.TrimSpace(localizacao.GetURL()),
		"ao_vivo":         localizacao.GetIsLive(),
		"precisao_metros": localizacao.GetAccuracyInMeters(),
		"comentario":      strings.TrimSpace(localizacao.GetComment()),
	}
	conteudo := primeiroTextoNaoVazio(
		localizacaoDados["nome"].(string),
		localizacaoDados["endereco"].(string),
		localizacaoDados["comentario"].(string),
	)
	if conteudo == "" {
		conteudo = "localizacao recebida"
	}
	extras := map[string]interface{}{
		"localizacao": localizacaoDados,
	}
	return conteudo, "localizacao", extras, map[string]interface{}{"localizacao": localizacaoDados}
}

func extrairLocalizacaoTempoRealMensagem(localizacao *waE2E.LiveLocationMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	localizacaoDados := map[string]interface{}{
		"latitude":        localizacao.GetDegreesLatitude(),
		"longitude":       localizacao.GetDegreesLongitude(),
		"precisao_metros": localizacao.GetAccuracyInMeters(),
		"velocidade_mps":  localizacao.GetSpeedInMps(),
		"legenda":         strings.TrimSpace(localizacao.GetCaption()),
		"sequencia":       localizacao.GetSequenceNumber(),
		"tempo_offset":    localizacao.GetTimeOffset(),
	}
	conteudo := strings.TrimSpace(localizacao.GetCaption())
	if conteudo == "" {
		conteudo = "localizacao em tempo real recebida"
	}
	extras := map[string]interface{}{
		"localizacao": localizacaoDados,
	}
	return conteudo, "localizacao_tempo_real", extras, map[string]interface{}{"localizacao": localizacaoDados}
}

func extrairBotoesMensagem(botoes *waE2E.ButtonsMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	itens := make([]map[string]interface{}, 0, len(botoes.GetButtons()))
	for _, botao := range botoes.GetButtons() {
		if botao == nil {
			continue
		}
		item := map[string]interface{}{
			"id":    strings.TrimSpace(botao.GetButtonID()),
			"texto": strings.TrimSpace(botao.GetButtonText().GetDisplayText()),
			"tipo":  botao.GetType().String(),
		}
		if native := botao.GetNativeFlowInfo(); native != nil {
			item["native_flow"] = map[string]interface{}{
				"nome":   strings.TrimSpace(native.GetName()),
				"params": parseJSONObjeto(native.GetParamsJSON()),
			}
		}
		itens = append(itens, item)
	}
	conteudo := primeiroTextoNaoVazio(botoes.GetContentText(), botoes.GetText())
	if conteudo == "" {
		conteudo = "botoes recebidos"
	}
	botoesDados := map[string]interface{}{
		"texto":       strings.TrimSpace(botoes.GetContentText()),
		"rodape":      strings.TrimSpace(botoes.GetFooterText()),
		"cabecalho":   strings.TrimSpace(botoes.GetText()),
		"tipo_header": botoes.GetHeaderType().String(),
		"botoes":      itens,
	}
	extras := map[string]interface{}{
		"botoes":            itens,
		"botoes_quantidade": len(itens),
	}
	return conteudo, "botoes", extras, map[string]interface{}{"botoes": botoesDados}
}

func extrairListaMensagem(lista *waE2E.ListMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	listaDados := map[string]interface{}{
		"titulo":            strings.TrimSpace(lista.GetTitle()),
		"descricao":         strings.TrimSpace(lista.GetDescription()),
		"botao_texto":       strings.TrimSpace(lista.GetButtonText()),
		"rodape":            strings.TrimSpace(lista.GetFooterText()),
		"tipo":              lista.GetListType().String(),
		"quantidade_secoes": len(lista.GetSections()),
	}
	conteudo := primeiroTextoNaoVazio(
		listaDados["descricao"].(string),
		listaDados["titulo"].(string),
		listaDados["botao_texto"].(string),
	)
	if conteudo == "" {
		conteudo = "lista recebida"
	}
	extras := map[string]interface{}{
		"lista_titulo":            listaDados["titulo"],
		"lista_descricao":         listaDados["descricao"],
		"lista_botao_texto":       listaDados["botao_texto"],
		"lista_tipo":              listaDados["tipo"],
		"lista_quantidade_secoes": listaDados["quantidade_secoes"],
	}
	return conteudo, "lista", extras, map[string]interface{}{"lista": listaDados}
}

func extrairInterativoMensagem(interativo *waE2E.InteractiveMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	tipoInterativo := tipoInterativoMensagem(interativo)
	botoes := botoesFluxoNativo(interativo.GetNativeFlowMessage())
	interativoDados := map[string]interface{}{
		"tipo":   tipoInterativo,
		"corpo":  strings.TrimSpace(interativo.GetBody().GetText()),
		"botoes": botoes,
	}
	if native := interativo.GetNativeFlowMessage(); native != nil {
		interativoDados["params"] = parseJSONObjeto(native.GetMessageParamsJSON())
		interativoDados["versao"] = native.GetMessageVersion()
	}
	// Header/footer sao necessarios para comparar byte a byte uma mensagem
	// interativa enviada pelo cliente oficial com a que a API monta; sem eles
	// diferencas de formato passam despercebidas.
	if header := interativo.GetHeader(); header != nil {
		headerDados := map[string]interface{}{
			"titulo":             strings.TrimSpace(header.GetTitle()),
			"subtitulo":          strings.TrimSpace(header.GetSubtitle()),
			"tem_midia_anexada":  header.GetHasMediaAttachment(),
			"tipo_midia":         tipoMidiaHeaderInterativo(header),
			"tem_bloks_widget":   header.GetBloksWidget() != nil,
			"tem_jpeg_thumbnail": len(header.GetJPEGThumbnail()) > 0,
		}
		interativoDados["header"] = headerDados
	}
	if rodape := strings.TrimSpace(interativo.GetFooter().GetText()); rodape != "" {
		interativoDados["rodape"] = rodape
	}
	conteudo := strings.TrimSpace(interativo.GetBody().GetText())
	if conteudo == "" {
		conteudo = "mensagem interativa recebida"
	}
	extras := map[string]interface{}{
		"interativo_tipo":       tipoInterativo,
		"interativo_botoes":     botoes,
		"interativo_quantidade": len(botoes),
	}
	return conteudo, "interativo", extras, map[string]interface{}{"interativo": interativoDados}
}

// tipoMidiaHeaderInterativo informa qual variante do oneof de midia o header usa,
// util para reproduzir fielmente mensagens interativas do cliente oficial.
func tipoMidiaHeaderInterativo(header *waE2E.InteractiveMessage_Header) string {
	switch {
	case header.GetDocumentMessage() != nil:
		return "documento"
	case header.GetImageMessage() != nil:
		return "imagem"
	case header.GetVideoMessage() != nil:
		return "video"
	case header.GetLocationMessage() != nil:
		return "localizacao"
	case header.GetProductMessage() != nil:
		return "produto"
	case len(header.GetJPEGThumbnail()) > 0:
		return "jpeg_thumbnail"
	default:
		return ""
	}
}

func tipoInterativoMensagem(interativo *waE2E.InteractiveMessage) string {
	switch {
	case interativo.GetNativeFlowMessage() != nil:
		return "native_flow"
	case interativo.GetCollectionMessage() != nil:
		return "collection"
	case interativo.GetShopStorefrontMessage() != nil:
		return "shop_storefront"
	case interativo.GetCarouselMessage() != nil:
		return "carousel"
	default:
		return "desconhecido"
	}
}

func botoesFluxoNativo(native *waE2E.InteractiveMessage_NativeFlowMessage) []map[string]interface{} {
	if native == nil {
		return nil
	}
	botoes := make([]map[string]interface{}, 0, len(native.GetButtons()))
	for _, botao := range native.GetButtons() {
		if botao == nil {
			continue
		}
		botoes = append(botoes, map[string]interface{}{
			"nome":   strings.TrimSpace(botao.GetName()),
			"params": parseJSONObjeto(botao.GetButtonParamsJSON()),
		})
	}
	return botoes
}

func extrairProdutoMensagem(produto *waE2E.ProductMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	produtoDados := map[string]interface{}{
		"body":               strings.TrimSpace(produto.GetBody()),
		"rodape":             strings.TrimSpace(produto.GetFooter()),
		"business_owner_jid": strings.TrimSpace(produto.GetBusinessOwnerJID()),
	}
	conteudo := strings.TrimSpace(produto.GetBody())
	if conteudo == "" {
		conteudo = "produto recebido"
	}
	return conteudo, "produto", produtoDados, map[string]interface{}{"produto": produtoDados}
}

func extrairPedidoMensagem(pedido *waE2E.OrderMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	pedidoDados := map[string]interface{}{
		"id":              strings.TrimSpace(pedido.GetOrderID()),
		"titulo":          strings.TrimSpace(pedido.GetOrderTitle()),
		"mensagem":        strings.TrimSpace(pedido.GetMessage()),
		"seller_jid":      strings.TrimSpace(pedido.GetSellerJID()),
		"itens":           pedido.GetItemCount(),
		"status":          pedido.GetStatus().String(),
		"surface":         pedido.GetSurface().String(),
		"total_1000":      pedido.GetTotalAmount1000(),
		"moeda":           strings.TrimSpace(pedido.GetTotalCurrencyCode()),
		"catalogo_tipo":   strings.TrimSpace(pedido.GetCatalogType()),
		"versao_mensagem": pedido.GetMessageVersion(),
	}
	conteudo := primeiroTextoNaoVazio(
		pedidoDados["mensagem"].(string),
		pedidoDados["titulo"].(string),
		pedidoDados["id"].(string),
	)
	if conteudo == "" {
		conteudo = "pedido recebido"
	}
	return conteudo, "pedido", pedidoDados, map[string]interface{}{"pedido": pedidoDados}
}

func extrairConviteGrupoMensagem(convite *waE2E.GroupInviteMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	conviteDados := map[string]interface{}{
		"grupo_jid":  strings.TrimSpace(convite.GetGroupJID()),
		"grupo_nome": strings.TrimSpace(convite.GetGroupName()),
		"codigo":     strings.TrimSpace(convite.GetInviteCode()),
		"expira_em":  convite.GetInviteExpiration(),
		"legenda":    strings.TrimSpace(convite.GetCaption()),
	}
	conteudo := primeiroTextoNaoVazio(
		conviteDados["legenda"].(string),
		conviteDados["grupo_nome"].(string),
	)
	if conteudo == "" {
		conteudo = "convite de grupo recebido"
	}
	return conteudo, "convite_grupo", conviteDados, map[string]interface{}{"convite_grupo": conviteDados}
}

func extrairChamadaAgendadaMensagem(chamada *waE2E.ScheduledCallCreationMessage) (string, string, map[string]interface{}, map[string]interface{}) {
	chamadaDados := map[string]interface{}{
		"titulo": strings.TrimSpace(chamada.GetTitle()),
		"tipo":   chamada.GetCallType().String(),
	}
	if ts := chamada.GetScheduledTimestampMS(); ts > 0 {
		chamadaDados["agendada_para"] = time.UnixMilli(ts).UTC()
	}
	conteudo := strings.TrimSpace(chamada.GetTitle())
	if conteudo == "" {
		conteudo = "chamada agendada recebida"
	}
	return conteudo, "chamada_agendada", chamadaDados, map[string]interface{}{"chamada_agendada": chamadaDados}
}

func extrairTipoMensagemNaoMapeada(msg *waE2E.Message) (string, string, map[string]interface{}, map[string]interface{}, bool) {
	campo := campoMensagemPreenchido(msg)
	if campo == "" {
		return "", "", nil, nil, false
	}
	tipo := normalizarTipoMensagemWebhook(campo)
	extras := map[string]interface{}{"tipo_original": campo}
	extrasMensagem := map[string]interface{}{"tipo_original": campo}
	return "mensagem " + tipo + " recebida", tipo, extras, extrasMensagem, true
}

func campoMensagemPreenchido(msg proto.Message) string {
	if msg == nil {
		return ""
	}
	refletida := msg.ProtoReflect()
	campos := refletida.Descriptor().Fields()
	for i := 0; i < campos.Len(); i++ {
		campo := campos.Get(i)
		nome := campo.JSONName()
		if nome == "" {
			nome = string(campo.Name())
		}
		if campoMensagemIgnorado(nome) {
			continue
		}
		if refletida.Has(campo) {
			return nome
		}
	}
	return ""
}

func campoMensagemIgnorado(nome string) bool {
	switch nome {
	case "", "messageContextInfo":
		return true
	default:
		return false
	}
}

func normalizarTipoMensagemWebhook(nome string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return "desconhecido"
	}
	var b strings.Builder
	var anterior byte
	for i := 0; i < len(nome); i++ {
		ch := nome[i]
		proximo := byte(0)
		if i+1 < len(nome) {
			proximo = nome[i+1]
		}
		switch {
		case ch >= 'A' && ch <= 'Z':
			if b.Len() > 0 && anterior != '_' && (ehMinusculaASCII(anterior) || ehDigitoASCII(anterior) || ((anterior >= 'A' && anterior <= 'Z') && ehMinusculaASCII(proximo))) {
				b.WriteByte('_')
			}
			b.WriteByte(ch + ('a' - 'A'))
		case ch == '-' || ch == ' ':
			if b.Len() > 0 && anterior != '_' {
				b.WriteByte('_')
			}
		default:
			b.WriteByte(ch)
		}
		anterior = ch
	}
	tipo := strings.Trim(b.String(), "_")
	tipo = strings.ReplaceAll(tipo, "_message", "")
	tipo = strings.Trim(tipo, "_")
	switch {
	case strings.HasPrefix(tipo, "poll_creation"):
		return strings.Replace(tipo, "poll_creation", "enquete", 1)
	case tipo == "poll_update":
		return "enquete_voto"
	case tipo == "reaction":
		return "reacao"
	case tipo == "contacts_array":
		return "contatos"
	case tipo == "live_location":
		return "localizacao_tempo_real"
	case tipo == "request_phone_number":
		return "solicitacao_telefone"
	default:
		return tipo
	}
}

func anexarChaveMensagemWebhook(destino map[string]interface{}, prefixo string, chave chaveMensagemWebhook) {
	if destino == nil || chave == nil {
		return
	}
	if id := strings.TrimSpace(chave.GetID()); id != "" {
		destino[prefixo+"_id"] = id
	}
	if chat := strings.TrimSpace(chave.GetRemoteJID()); chat != "" {
		destino[prefixo+"_chat_jid"] = chat
	}
	destino[prefixo+"_enviada_por_mim"] = chave.GetFromMe()
	if participante := strings.TrimSpace(chave.GetParticipant()); participante != "" {
		destino[prefixo+"_participante_jid"] = participante
	}
}

func primeiroTextoNaoVazio(valores ...string) string {
	for _, valor := range valores {
		if texto := strings.TrimSpace(valor); texto != "" {
			return texto
		}
	}
	return ""
}

func ehMinusculaASCII(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func ehDigitoASCII(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func classificarAcaoMensagem(evento *events.Message) (string, map[string]interface{}) {
	if evento == nil {
		return "recebida", nil
	}
	if evento.IsEdit {
		return "editada", map[string]interface{}{"editada": true}
	}
	if protocolo := evento.RawMessage.GetProtocolMessage(); protocolo != nil {
		switch protocolo.GetType() {
		case waE2E.ProtocolMessage_MESSAGE_EDIT:
			detalhes := map[string]interface{}{
				"editada":        true,
				"protocolo_tipo": "MESSAGE_EDIT",
			}
			if chave := protocolo.GetKey(); chave != nil {
				if id := strings.TrimSpace(chave.GetID()); id != "" {
					detalhes["mensagem_original_id"] = id
				}
			}
			if ts := protocolo.GetTimestampMS(); ts > 0 {
				detalhes["editada_em"] = time.UnixMilli(ts).UTC()
			}
			return "editada", detalhes
		case waE2E.ProtocolMessage_REVOKE:
			detalhes := map[string]interface{}{
				"apagada":        true,
				"protocolo_tipo": "REVOKE",
			}
			if chave := protocolo.GetKey(); chave != nil {
				if id := strings.TrimSpace(chave.GetID()); id != "" {
					detalhes["mensagem_original_id"] = id
					detalhes["mensagem_apagada_id"] = id
				}
				detalhes["mensagem_apagada_enviada_por_mim"] = chave.GetFromMe()
				if participante := strings.TrimSpace(chave.GetParticipant()); participante != "" {
					detalhes["mensagem_apagada_remetente_jid"] = participante
				}
				if chat := strings.TrimSpace(chave.GetRemoteJID()); chat != "" {
					detalhes["mensagem_apagada_chat_jid"] = chat
				}
			}
			if ts := protocolo.GetTimestampMS(); ts > 0 {
				detalhes["apagada_em"] = time.UnixMilli(ts).UTC()
			}
			return "apagada", detalhes
		}
	}
	return "recebida", nil
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
	for i := 0; i < 16 && msg != nil; i++ {
		switch {
		case msg.GetDeviceSentMessage() != nil && msg.GetDeviceSentMessage().GetMessage() != nil:
			msg = msg.GetDeviceSentMessage().GetMessage()
		case msg.GetEphemeralMessage() != nil && msg.GetEphemeralMessage().GetMessage() != nil:
			msg = msg.GetEphemeralMessage().GetMessage()
		case msg.GetViewOnceMessage() != nil && msg.GetViewOnceMessage().GetMessage() != nil:
			msg = msg.GetViewOnceMessage().GetMessage()
		case msg.GetViewOnceMessageV2() != nil && msg.GetViewOnceMessageV2().GetMessage() != nil:
			msg = msg.GetViewOnceMessageV2().GetMessage()
		case msg.GetViewOnceMessageV2Extension() != nil && msg.GetViewOnceMessageV2Extension().GetMessage() != nil:
			msg = msg.GetViewOnceMessageV2Extension().GetMessage()
		case msg.GetEditedMessage() != nil && msg.GetEditedMessage().GetMessage() != nil:
			msg = msg.GetEditedMessage().GetMessage()
		case msg.GetProtocolMessage() != nil && msg.GetProtocolMessage().GetEditedMessage() != nil:
			msg = msg.GetProtocolMessage().GetEditedMessage()
		case msg.GetDocumentWithCaptionMessage() != nil && msg.GetDocumentWithCaptionMessage().GetMessage() != nil:
			msg = msg.GetDocumentWithCaptionMessage().GetMessage()
		case msg.GetGroupMentionedMessage() != nil && msg.GetGroupMentionedMessage().GetMessage() != nil:
			msg = msg.GetGroupMentionedMessage().GetMessage()
		case msg.GetBotInvokeMessage() != nil && msg.GetBotInvokeMessage().GetMessage() != nil:
			msg = msg.GetBotInvokeMessage().GetMessage()
		case msg.GetLottieStickerMessage() != nil && msg.GetLottieStickerMessage().GetMessage() != nil:
			msg = msg.GetLottieStickerMessage().GetMessage()
		case msg.GetEventCoverImage() != nil && msg.GetEventCoverImage().GetMessage() != nil:
			msg = msg.GetEventCoverImage().GetMessage()
		case msg.GetStatusMentionMessage() != nil && msg.GetStatusMentionMessage().GetMessage() != nil:
			msg = msg.GetStatusMentionMessage().GetMessage()
		case msg.GetPollCreationOptionImageMessage() != nil && msg.GetPollCreationOptionImageMessage().GetMessage() != nil:
			msg = msg.GetPollCreationOptionImageMessage().GetMessage()
		case msg.GetAssociatedChildMessage() != nil && msg.GetAssociatedChildMessage().GetMessage() != nil:
			msg = msg.GetAssociatedChildMessage().GetMessage()
		case msg.GetGroupStatusMentionMessage() != nil && msg.GetGroupStatusMentionMessage().GetMessage() != nil:
			msg = msg.GetGroupStatusMentionMessage().GetMessage()
		case msg.GetPollCreationMessageV4() != nil && msg.GetPollCreationMessageV4().GetMessage() != nil:
			msg = msg.GetPollCreationMessageV4().GetMessage()
		case msg.GetStatusAddYours() != nil && msg.GetStatusAddYours().GetMessage() != nil:
			msg = msg.GetStatusAddYours().GetMessage()
		case msg.GetGroupStatusMessage() != nil && msg.GetGroupStatusMessage().GetMessage() != nil:
			msg = msg.GetGroupStatusMessage().GetMessage()
		case msg.GetLimitSharingMessage() != nil && msg.GetLimitSharingMessage().GetMessage() != nil:
			msg = msg.GetLimitSharingMessage().GetMessage()
		case msg.GetBotTaskMessage() != nil && msg.GetBotTaskMessage().GetMessage() != nil:
			msg = msg.GetBotTaskMessage().GetMessage()
		case msg.GetQuestionMessage() != nil && msg.GetQuestionMessage().GetMessage() != nil:
			msg = msg.GetQuestionMessage().GetMessage()
		case msg.GetGroupStatusMessageV2() != nil && msg.GetGroupStatusMessageV2().GetMessage() != nil:
			msg = msg.GetGroupStatusMessageV2().GetMessage()
		case msg.GetBotForwardedMessage() != nil && msg.GetBotForwardedMessage().GetMessage() != nil:
			msg = msg.GetBotForwardedMessage().GetMessage()
		case msg.GetQuestionReplyMessage() != nil && msg.GetQuestionReplyMessage().GetMessage() != nil:
			msg = msg.GetQuestionReplyMessage().GetMessage()
		case msg.GetNewsletterAdminProfileMessage() != nil && msg.GetNewsletterAdminProfileMessage().GetMessage() != nil:
			msg = msg.GetNewsletterAdminProfileMessage().GetMessage()
		case msg.GetSpoilerMessage() != nil && msg.GetSpoilerMessage().GetMessage() != nil:
			msg = msg.GetSpoilerMessage().GetMessage()
		case msg.GetNewsletterAdminProfileStatusMessage() != nil && msg.GetNewsletterAdminProfileStatusMessage().GetMessage() != nil:
			msg = msg.GetNewsletterAdminProfileStatusMessage().GetMessage()
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

type tentativaBotoes struct {
	modo   string
	montar func() (*waE2E.Message, error)
}

func modoBotoesAuto(req models.EnvioBotoesRequest) bool {
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "", "auto":
		return true
	default:
		return false
	}
}

func montarTentativasBotoes(req models.EnvioBotoesRequest) []tentativaBotoes {
	if !modoBotoesAuto(req) {
		return []tentativaBotoes{{
			modo: strings.ToLower(strings.TrimSpace(req.Modo)),
			montar: func() (*waE2E.Message, error) {
				msg, _, err := montarMensagemBotoes(req)
				return msg, err
			},
		}}
	}
	tentativas := []tentativaBotoes{
		{modo: "native_flow_direct", montar: func() (*waE2E.Message, error) { return montarMensagemBotoesNativeFlowDireto(req) }},
		{modo: "native_flow_view_once", montar: func() (*waE2E.Message, error) { return montarMensagemBotoesNativeFlowViewOnce(req) }},
		{modo: "template", montar: func() (*waE2E.Message, error) { return montarMensagemBotoesTemplate(req) }},
	}
	if todosBotoesResposta(req.Botoes) {
		tentativas = append(tentativas, tentativaBotoes{
			modo:   "buttons",
			montar: func() (*waE2E.Message, error) { return montarMensagemBotoesLegacy(req), nil },
		})
	}
	return tentativas
}

func todosBotoesResposta(botoes []models.BotaoRequest) bool {
	for _, botao := range botoes {
		if !ehBotaoResposta(botao) {
			return false
		}
	}
	return true
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
		MessageContextInfo: contextInfoMensagemInterativa(),
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
				MessageContextInfo: contextInfoMensagemInterativa(),
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

type tentativaLista struct {
	modo   string
	montar func() *waE2E.Message
}

func modoListaAuto(req models.EnvioListaRequest) bool {
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "", "auto":
		return true
	default:
		return false
	}
}

func montarTentativasLista(req models.EnvioListaRequest) []tentativaLista {
	nativeFlow := tentativaLista{
		modo:   "native_flow",
		montar: func() *waE2E.Message { return montarMensagemListaNativeFlow(req) },
	}
	listaClassica := tentativaLista{
		modo:   "lista",
		montar: func() *waE2E.Message { return montarMensagemLista(req) },
	}
	switch strings.ToLower(strings.TrimSpace(req.Modo)) {
	case "", "auto":
		// Nenhum formato de menu interativo funciona hoje a partir de um cliente
		// multi-device comum (ver montarMensagemListaNativeFlow). O auto tenta o
		// ListMessage - que o servidor recusa com 405 - justamente para que a
		// cadeia chegue ao fallback de texto, que e o unico resultado que o
		// destinatario realmente le. Nao inclua native_flow aqui: ele e ACEITO
		// sem erro e a cadeia pararia nele, entregando uma mensagem invisivel.
		return []tentativaLista{
			listaClassica,
			{modo: "lista_view_once", montar: func() *waE2E.Message { return montarMensagemListaViewOnce(req) }},
		}
	case "native_flow", "single_select", "nativeflow":
		return []tentativaLista{nativeFlow}
	case "lista_view_once", "list_view_once", "view_once", "viewonce":
		return []tentativaLista{{modo: "lista_view_once", montar: func() *waE2E.Message { return montarMensagemListaViewOnce(req) }}}
	default:
		return []tentativaLista{listaClassica}
	}
}

// montarMensagemListaNativeFlow monta o menu como InteractiveMessage/native_flow
// com um botao "single_select".
//
// ATENCAO: nenhum formato de menu interativo funciona hoje a partir de um
// cliente multi-device comum. Testado contra o servidor real:
//
//	ButtonsMessage (<biz><buttons/></biz>)          -> ack error 405
//	ListMessage, inclusive replicando byte a byte o
//	  stanza de uma empresa cujo menu renderiza      -> ack error 405
//	native_flow single_select                        -> aceito, nunca renderiza
//
// Os dois formatos legados dao 405 mesmo com o envelope <biz> identico ao de um
// remetente que funciona, o que indica que a restricao e no tipo de remetente
// (a empresa envia pela API oficial do WhatsApp Business), nao no formato.
//
// Esta funcao fica disponivel pelo modo "native_flow" para experimentacao, mas
// nao entra no modo auto: por ser aceita sem erro, ela impediria o fallback de
// texto, que hoje e o unico resultado que o destinatario consegue ler.
// Para menus de verdade, use enquete (ate 12 opcoes) ou botoes (ate 3).
func montarMensagemListaNativeFlow(req models.EnvioListaRequest) *waE2E.Message {
	secoes := make([]map[string]interface{}, 0, len(req.Secoes))
	for _, secao := range req.Secoes {
		linhas := make([]map[string]interface{}, 0, len(secao.Linhas))
		for _, linha := range secao.Linhas {
			item := map[string]interface{}{
				"title": strings.TrimSpace(linha.Titulo),
				"id":    strings.TrimSpace(linha.ID),
			}
			if descricao := strings.TrimSpace(linha.Descricao); descricao != "" {
				item["description"] = descricao
			}
			linhas = append(linhas, item)
		}
		secoes = append(secoes, map[string]interface{}{
			"title": strings.TrimSpace(secao.Titulo),
			"rows":  linhas,
		})
	}
	params := map[string]interface{}{
		"title":    strings.TrimSpace(req.BotaoTexto),
		"sections": secoes,
	}
	payload, err := json.Marshal(params)
	if err != nil {
		// Sem o payload nao ha menu; cai para o ListMessage classico, que o
		// chamador ja trata como proxima tentativa.
		return montarMensagemLista(req)
	}

	interactive := &waE2E.InteractiveMessage{
		Body: &waE2E.InteractiveMessage_Body{Text: proto.String(textoMensagemListaEnvio(req))},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons: []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{{
					Name:             proto.String("single_select"),
					ButtonParamsJSON: proto.String(string(payload)),
				}},
				MessageParamsJSON: proto.String(""),
				MessageVersion:    proto.Int32(1),
			},
		},
		ContextInfo: contextInfoLista(req),
	}
	if titulo := strings.TrimSpace(req.Titulo); titulo != "" {
		interactive.Header = &waE2E.InteractiveMessage_Header{
			Title:              proto.String(titulo),
			HasMediaAttachment: proto.Bool(false),
		}
	}
	if rodape := strings.TrimSpace(req.Rodape); rodape != "" {
		interactive.Footer = &waE2E.InteractiveMessage_Footer{Text: proto.String(rodape)}
	}
	return &waE2E.Message{
		InteractiveMessage: interactive,
		MessageContextInfo: contextInfoMensagemInterativa(),
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
	if strings.Contains(texto, "not-allowed") {
		return true
	}
	// Codigos que o servidor do WhatsApp usa para recusar mensagens interativas:
	//  405 - not-allowed (recusa classica de botoes/lista)
	//  473 - recusa observada ao tentar iniciar cobranca Pix por dispositivo
	//        vinculado, mesmo com Pagamentos habilitado na conta
	//  479 - stanza interativa em formato que o servidor nao aceita
	for _, codigo := range []string{"405", "473", "479"} {
		if strings.Contains(texto, "server returned error "+codigo) {
			return true
		}
	}
	return false
}

func montarMensagemLista(req models.EnvioListaRequest) *waE2E.Message {
	return &waE2E.Message{ListMessage: montarListMessage(req)}
}

func montarMensagemListaViewOnce(req models.EnvioListaRequest) *waE2E.Message {
	return &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ListMessage: montarListMessage(req),
				MessageContextInfo: &waE2E.MessageContextInfo{
					DeviceListMetadata:        &waE2E.DeviceListMetadata{},
					DeviceListMetadataVersion: proto.Int32(2),
				},
			},
		},
	}
}

func montarListMessage(req models.EnvioListaRequest) *waE2E.ListMessage {
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

	return &waE2E.ListMessage{
		Title:       proto.String(strings.TrimSpace(req.Titulo)),
		Description: proto.String(textoMensagemListaEnvio(req)),
		ButtonText:  proto.String(strings.TrimSpace(req.BotaoTexto)),
		ListType:    waE2E.ListMessage_SINGLE_SELECT.Enum(),
		Sections:    secoes,
		FooterText:  proto.String(strings.TrimSpace(req.Rodape)),
		ContextInfo: contextInfoLista(req),
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
