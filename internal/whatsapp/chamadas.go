package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"dyalog-api-go/internal/chamadas"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/voip/call"
	"dyalog-api-go/internal/voip/core"
	"dyalog-api-go/internal/voip/signaling"
	"dyalog-api-go/internal/wa"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

type chamadaAtiva struct {
	manager *call.CallManager
	bridge  *chamadas.Bridge
}

func (g *GerenciadorInstancias) IniciarChamada(ctx context.Context, req models.IniciarChamadaRequest) (models.ResultadoChamada, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoChamada{}, err
	}
	destinos, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, false)
	if err != nil {
		return models.ResultadoChamada{}, err
	}
	if len(destinos) == 0 {
		return models.ResultadoChamada{}, fmt.Errorf("nenhum destino encontrado para chamada")
	}
	callID := signaling.GenerateCallID()
	cm := g.novoCallManager(req.Instancia, runtime, callID)
	g.adicionarChamada(runtime, callID, &chamadaAtiva{manager: cm})
	if err := cm.StartCall(ctx, callID, destinos[0], req.Video); err != nil {
		g.removerChamada(runtime, callID)
		return models.ResultadoChamada{}, err
	}
	info := cm.CurrentCall()
	return g.resultadoChamada(req.Instancia, info), nil
}

func (g *GerenciadorInstancias) AceitarChamada(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	_, ativa, err := g.obterChamadaAtiva(ctx, req.Instancia, req.ChamadaID)
	if err != nil {
		return models.ResultadoChamada{}, err
	}
	if err := ativa.manager.AcceptCall(ctx, req.ChamadaID); err != nil {
		return models.ResultadoChamada{}, err
	}
	return g.resultadoChamada(req.Instancia, ativa.manager.CurrentCall()), nil
}

func (g *GerenciadorInstancias) RejeitarChamada(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	runtime, ativa, err := g.obterChamadaAtiva(ctx, req.Instancia, req.ChamadaID)
	if err != nil {
		return models.ResultadoChamada{}, err
	}
	if err := ativa.manager.RejectCall(ctx, req.ChamadaID, core.EndCallReasonDeclined); err != nil {
		return models.ResultadoChamada{}, err
	}
	info := ativa.manager.CurrentCall()
	g.removerChamada(runtime, req.ChamadaID)
	return g.resultadoChamada(req.Instancia, info), nil
}

func (g *GerenciadorInstancias) EncerrarChamada(ctx context.Context, req models.AcaoChamadaRequest) (models.ResultadoChamada, error) {
	runtime, ativa, err := g.obterChamadaAtiva(ctx, req.Instancia, req.ChamadaID)
	if err != nil {
		return models.ResultadoChamada{}, err
	}
	if err := ativa.manager.EndCall(ctx, core.EndCallReasonUserEnded); err != nil {
		return models.ResultadoChamada{}, err
	}
	info := ativa.manager.CurrentCall()
	g.removerChamada(runtime, req.ChamadaID)
	return g.resultadoChamada(req.Instancia, info), nil
}

func (g *GerenciadorInstancias) SinalizarWebRTC(ctx context.Context, req models.SinalizacaoWebRTCRequest) (models.ResultadoWebRTC, error) {
	runtime, ativa, err := g.obterChamadaAtiva(ctx, req.Instancia, req.ChamadaID)
	if err != nil {
		return models.ResultadoWebRTC{}, err
	}
	bridge, resposta, err := chamadas.NovoBridge(req.SDPOffer, slog.Default())
	if err != nil {
		return models.ResultadoWebRTC{}, err
	}
	bridge.OnBrowserPCM = func(pcm []float32) {
		ativa.manager.FeedCapturedPCM(pcm)
	}
	bridge.OnTerminalICE = func() {
		_ = ativa.manager.EndCall(context.Background(), core.EndCallReasonUserEnded)
		g.removerChamada(runtime, req.ChamadaID)
	}
	if ativa.bridge != nil {
		ativa.bridge.Close()
	}
	ativa.bridge = bridge
	return models.ResultadoWebRTC{
		Instancia:    req.Instancia,
		ChamadaID:    req.ChamadaID,
		SDPAnswer:    resposta,
		Transporte:   "media_track",
		AudioEnvio:   "opus_track_browser_para_api",
		AudioRetorno: "pcmu_track_api_para_browser",
	}, nil
}

func (g *GerenciadorInstancias) ListarChamadas(ctx context.Context, instanciaID string) ([]models.ResultadoChamada, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	resultado := make([]models.ResultadoChamada, 0, len(runtime.chamadas))
	for _, ativa := range runtime.chamadas {
		resultado = append(resultado, g.resultadoChamada(instanciaID, ativa.manager.CurrentCall()))
	}
	return resultado, nil
}

func (g *GerenciadorInstancias) tratarEventoChamada(instanciaID string, runtime *runtimeInstancia, evt interface{}) bool {
	ctx := context.Background()
	switch evento := evt.(type) {
	case *events.CallOffer:
		configuracao := g.obterConfiguracaoInstancia(ctx, instanciaID)
		if configuracao.RejeitarChamadas {
			go g.rejeitarChamadaSeConfigurado(instanciaID, runtime.client, evento.BasicCallMeta)
			return true
		}
		node := wrapCall(evento.From, evento.Data)
		callID := callIDFromNode(node)
		if callID == "" {
			return true
		}
		cm := g.novoCallManager(instanciaID, runtime, callID)
		g.adicionarChamada(runtime, callID, &chamadaAtiva{manager: cm})
		cm.HandleCallOffer(ctx, node, evento.From)
		return true
	case *events.CallOfferNotice:
		go g.rejeitarChamadaSeConfigurado(instanciaID, runtime.client, evento.BasicCallMeta)
		return true
	case *events.CallAccept:
		if ativa := g.chamadaPorEvento(runtime, evento.From, evento.Data); ativa != nil {
			ativa.manager.HandleCallAccept(ctx, wrapCall(evento.From, evento.Data), evento.From)
		}
		return true
	case *events.CallTransport:
		if ativa := g.chamadaPorEvento(runtime, evento.From, evento.Data); ativa != nil {
			ativa.manager.HandleCallTransport(ctx, wrapCall(evento.From, evento.Data), evento.From)
		}
		return true
	case *events.CallTerminate:
		if ativa := g.chamadaPorEvento(runtime, evento.From, evento.Data); ativa != nil {
			ativa.manager.HandleCallTerminate(wrapCall(evento.From, evento.Data))
		}
		return true
	case *events.CallReject:
		if ativa := g.chamadaPorEvento(runtime, evento.From, evento.Data); ativa != nil {
			ativa.manager.HandleCallTerminate(wrapCall(evento.From, evento.Data))
		}
		return true
	default:
		return false
	}
}

func (g *GerenciadorInstancias) novoCallManager(instanciaID string, runtime *runtimeInstancia, callID string) *call.CallManager {
	cm := call.NewCallManager(wa.NewSocket(runtime.client), slog.Default())
	cm.OnIncoming = func(info *call.CallInfo) {
		g.dispararEventoChamada(instanciaID, "recebida", info)
	}
	cm.OnStateChange = func(info *call.CallInfo) {
		if info == nil {
			return
		}
		g.dispararEventoChamada(instanciaID, "estado", info)
		if info.IsEnded() {
			g.removerChamada(runtime, info.CallID)
		}
	}
	cm.OnEnded = func(info *call.CallInfo) {
		if info == nil {
			return
		}
		g.dispararEventoChamada(instanciaID, "encerrada", info)
		g.removerChamada(runtime, info.CallID)
	}
	cm.OnPeerAudio = func(pcm []float32) {
		g.mu.RLock()
		ativa := runtime.chamadas[callID]
		g.mu.RUnlock()
		if ativa != nil && ativa.bridge != nil {
			_ = ativa.bridge.WritePCM(pcm)
		}
	}
	return cm
}

func (g *GerenciadorInstancias) obterRuntimeConectado(ctx context.Context, instanciaID string) (*runtimeInstancia, error) {
	instanciaID = strings.TrimSpace(instanciaID)
	if instanciaID == "" {
		return nil, fmt.Errorf("instancia obrigatoria")
	}
	runtime, err := g.obterOuCriarRuntime(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	if runtime.client == nil || !runtime.client.IsConnected() || !runtime.client.IsLoggedIn() {
		return nil, fmt.Errorf("instancia nao conectada")
	}
	return runtime, nil
}

func (g *GerenciadorInstancias) obterChamadaAtiva(ctx context.Context, instanciaID, chamadaID string) (*runtimeInstancia, *chamadaAtiva, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, nil, err
	}
	chamadaID = strings.TrimSpace(chamadaID)
	if chamadaID == "" {
		return nil, nil, fmt.Errorf("chamada_id obrigatorio")
	}
	g.mu.RLock()
	ativa := runtime.chamadas[chamadaID]
	g.mu.RUnlock()
	if ativa == nil {
		return nil, nil, fmt.Errorf("chamada nao encontrada")
	}
	return runtime, ativa, nil
}

func (g *GerenciadorInstancias) adicionarChamada(runtime *runtimeInstancia, chamadaID string, ativa *chamadaAtiva) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if runtime.chamadas == nil {
		runtime.chamadas = make(map[string]*chamadaAtiva)
	}
	runtime.chamadas[chamadaID] = ativa
}

func (g *GerenciadorInstancias) removerChamada(runtime *runtimeInstancia, chamadaID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if runtime.chamadas == nil {
		return
	}
	if ativa := runtime.chamadas[chamadaID]; ativa != nil && ativa.bridge != nil {
		ativa.bridge.Close()
	}
	delete(runtime.chamadas, chamadaID)
}

func (g *GerenciadorInstancias) chamadaPorEvento(runtime *runtimeInstancia, from types.JID, data *waBinary.Node) *chamadaAtiva {
	chamadaID := callIDFromNode(wrapCall(from, data))
	if chamadaID == "" {
		return nil
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	return runtime.chamadas[chamadaID]
}

func (g *GerenciadorInstancias) dispararEventoChamada(instanciaID, acao string, info *call.CallInfo) {
	if g.dispatcher == nil || info == nil {
		return
	}
	apiBase := "/api/v1/chamadas/" + info.CallID
	numero := g.numeroChamadaPreferencial(info)
	g.dispatcher.DispararEvento(context.Background(), instanciaID, models.EventoWebhookChamadas, map[string]interface{}{
		"acao":         acao,
		"chamada_id":   info.CallID,
		"peer_jid":     info.PeerJid,
		"peer_numero":  numero,
		"numero":       numero,
		"caller_pn":    strings.TrimSpace(info.CallerPn),
		"call_creator": info.CallCreator,
		"direcao":      string(info.Direction),
		"estado":       string(info.StateData.State),
		"tipo":         string(info.MediaType),
		"criada_em":    info.CreatedAt,
		"api": map[string]string{
			"aceitar":  apiBase + "/aceitar",
			"rejeitar": apiBase + "/rejeitar",
			"encerrar": apiBase,
			"webrtc":   apiBase + "/webrtc",
		},
		"observacao": "Para audio real, a aplicacao deve negociar WebRTC usando sdp_offer no endpoint api.webrtc. O modo recomendado usa track de audio padrao; DataChannel pcm permanece como fallback legado.",
	})
}

func (g *GerenciadorInstancias) resultadoChamada(instanciaID string, info *call.CallInfo) models.ResultadoChamada {
	if info == nil {
		return models.ResultadoChamada{Instancia: instanciaID, Estado: "desconhecida", Tipo: "audio"}
	}
	return models.ResultadoChamada{
		Instancia: instanciaID,
		ChamadaID: info.CallID,
		PeerJID:   info.PeerJid,
		Numero:    g.numeroChamadaPreferencial(info),
		Direcao:   string(info.Direction),
		Estado:    string(info.StateData.State),
		Tipo:      string(info.MediaType),
		CriadaEm:  info.CreatedAt,
	}
}

func (g *GerenciadorInstancias) numeroChamadaPreferencial(info *call.CallInfo) string {
	if info == nil {
		return ""
	}
	if numero := apenasDigitos.ReplaceAllString(strings.TrimSpace(info.CallerPn), ""); numero != "" {
		return numero
	}
	peer, err := types.ParseJID(info.PeerJid)
	if err != nil {
		return ""
	}
	g.mu.RLock()
	for _, chave := range []string{peer.String(), peer.User} {
		if numero := strings.TrimSpace(g.aliasesNumero[chave]); numero != "" {
			g.mu.RUnlock()
			return numero
		}
	}
	g.mu.RUnlock()
	return extrairNumeroJID(peer)
}

func numeroChamada(jidTexto string) string {
	jid, err := types.ParseJID(jidTexto)
	if err != nil {
		return ""
	}
	return extrairNumeroJID(jid)
}

func callIDFromNode(node *waBinary.Node) string {
	info := signaling.ExtractNodeInfo(node)
	if info == nil {
		return ""
	}
	return info.CallID
}

func wrapCall(from types.JID, inner *waBinary.Node) *waBinary.Node {
	content := []waBinary.Node{}
	if inner != nil {
		content = append(content, *inner)
	}
	return &waBinary.Node{
		Tag:     "call",
		Attrs:   waBinary.Attrs{"from": from},
		Content: content,
	}
}
