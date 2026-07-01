package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"

	"github.com/google/uuid"
)

type Dispatcher struct {
	webhookStore  store.WebhookStore
	entregaStore  store.WebhookEntregaStore
	client        *http.Client
	maxTentativas int
	intervaloBase time.Duration
	maxDuracao    time.Duration
	maxIntervalo  time.Duration
	concorrencia  int
	lote          int
	sinal         chan struct{}
	processando   atomic.Bool
}

func NovoDispatcher(webhookStore store.WebhookStore, entregaStore store.WebhookEntregaStore, timeout, intervaloBase, maxDuracao, maxIntervalo time.Duration, maxTentativas, concorrencia, lote int) *Dispatcher {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if intervaloBase <= 0 {
		intervaloBase = 30 * time.Second
	}
	if maxTentativas <= 0 {
		maxTentativas = 60
	}
	if maxDuracao <= 0 {
		maxDuracao = 24 * time.Hour
	}
	if maxIntervalo <= 0 {
		maxIntervalo = 30 * time.Minute
	}
	if concorrencia <= 0 {
		concorrencia = 5
	}
	if lote <= 0 {
		lote = 25
	}
	return &Dispatcher{
		webhookStore:  webhookStore,
		entregaStore:  entregaStore,
		client:        &http.Client{Timeout: timeout},
		maxTentativas: maxTentativas,
		intervaloBase: intervaloBase,
		maxDuracao:    maxDuracao,
		maxIntervalo:  maxIntervalo,
		concorrencia:  concorrencia,
		lote:          lote,
		sinal:         make(chan struct{}, 1),
	}
}

func (d *Dispatcher) Iniciar(ctx context.Context) {
	if d == nil || d.entregaStore == nil {
		return
	}
	go d.loop(ctx)
}

func (d *Dispatcher) DispararEvento(ctx context.Context, instanciaID, evento string, dados interface{}) {
	if d == nil || d.webhookStore == nil || d.entregaStore == nil {
		return
	}
	webhooks, err := d.webhookStore.ListarWebhooksAtivosPorEvento(ctx, instanciaID, evento)
	if err != nil || len(webhooks) == 0 {
		return
	}
	webhooks = filtrarWebhooksPorEvento(webhooks, evento)
	if len(webhooks) == 0 {
		return
	}
	payload := models.EventoWebhook{
		Evento:      evento,
		InstanciaID: instanciaID,
		OcorridoEm:  time.Now().UTC(),
		Dados:       dados,
	}
	corpo, err := json.Marshal(payload)
	if err != nil {
		return
	}
	agora := time.Now().UTC()
	for _, webhook := range webhooks {
		entrega := models.WebhookEntrega{
			ID:                 uuid.NewString(),
			WebhookID:          webhook.ID,
			InstanciaID:        instanciaID,
			WebhookNome:        webhook.Nome,
			URL:                webhook.URL,
			Evento:             evento,
			Payload:            append([]byte(nil), corpo...),
			Status:             models.WebhookEntregaPendente,
			MaxTentativas:      d.maxTentativas,
			ProximaTentativaEm: agora,
			CriadoEm:           agora,
			AtualizadoEm:       agora,
		}
		if _, err := d.entregaStore.EnfileirarWebhookEntrega(ctx, entrega); err != nil {
			fmt.Printf("erro ao enfileirar webhook %s: %v\n", webhook.ID, err)
			continue
		}
	}
	d.sinalizar()
	go d.processarPendentes(context.Background())
}

func filtrarWebhooksPorEvento(webhooks []models.WebhookInstancia, evento string) []models.WebhookInstancia {
	filtrados := make([]models.WebhookInstancia, 0, len(webhooks))
	for _, webhook := range webhooks {
		if webhookAssinaEvento(webhook, evento) {
			filtrados = append(filtrados, webhook)
			continue
		}
		fmt.Printf("webhook %s ignorado para evento %s: eventos configurados=%v\n", webhook.ID, evento, webhook.Eventos)
	}
	return filtrados
}

func webhookAssinaEvento(webhook models.WebhookInstancia, evento string) bool {
	for _, configurado := range webhook.Eventos {
		if configurado == evento {
			return true
		}
	}
	return false
}

func (d *Dispatcher) loop(ctx context.Context) {
	intervaloVarredura := d.intervaloBase
	if intervaloVarredura < 250*time.Millisecond {
		intervaloVarredura = 250 * time.Millisecond
	}
	if intervaloVarredura > 5*time.Second {
		intervaloVarredura = 5 * time.Second
	}
	ticker := time.NewTicker(intervaloVarredura)
	defer ticker.Stop()
	for {
		d.processarPendentes(ctx)
		select {
		case <-ctx.Done():
			return
		case <-d.sinal:
		case <-ticker.C:
		}
	}
}

func (d *Dispatcher) sinalizar() {
	if d == nil || d.sinal == nil {
		return
	}
	select {
	case d.sinal <- struct{}{}:
	default:
	}
}

func (d *Dispatcher) processarPendentes(ctx context.Context) {
	if !d.processando.CompareAndSwap(false, true) {
		return
	}
	defer d.processando.Store(false)
	entregas, err := d.entregaStore.BuscarWebhookEntregasPendentes(ctx, d.lote, time.Now().UTC())
	if err != nil {
		fmt.Printf("erro ao buscar fila de webhooks: %v\n", err)
		return
	}
	sem := make(chan struct{}, d.concorrencia)
	var wg sync.WaitGroup
	for _, entrega := range entregas {
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return
		}
		entrega := entrega
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			agora := time.Now().UTC()
			if err := d.entregaStore.MarcarWebhookEntregaEnviando(ctx, entrega.ID, agora); err != nil {
				return
			}
			d.enviarEntrega(ctx, entrega)
		}()
	}
	wg.Wait()
}

func (d *Dispatcher) enviarEntrega(ctx context.Context, entrega models.WebhookEntrega) {
	tentativas := entrega.Tentativas + 1
	statusHTTP, erroEnvio := d.executarHTTP(ctx, entrega)
	agora := time.Now().UTC()
	status := models.WebhookEntregaEntregue
	ultimoErro := ""
	proxima := agora
	if erroEnvio != "" {
		ultimoErro = erroEnvio
		prazoEsgotado := !entrega.CriadoEm.IsZero() && d.maxDuracao > 0 && agora.Sub(entrega.CriadoEm) >= d.maxDuracao
		tentativasEsgotadas := entrega.MaxTentativas > 0 && tentativas >= entrega.MaxTentativas
		if prazoEsgotado || tentativasEsgotadas {
			status = models.WebhookEntregaEsgotada
		} else {
			status = models.WebhookEntregaFalha
			proxima = agora.Add(d.backoff(tentativas))
			if !entrega.CriadoEm.IsZero() && d.maxDuracao > 0 {
				prazoFinal := entrega.CriadoEm.Add(d.maxDuracao)
				if proxima.After(prazoFinal) {
					proxima = prazoFinal
				}
			}
		}
	}
	if err := d.entregaStore.RegistrarResultadoWebhookEntrega(ctx, entrega.ID, status, tentativas, &proxima, statusHTTP, ultimoErro, agora); err != nil {
		fmt.Printf("erro ao registrar resultado do webhook %s: %v\n", entrega.ID, err)
	}
}

func (d *Dispatcher) executarHTTP(ctx context.Context, entrega models.WebhookEntrega) (int, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, entrega.URL, bytes.NewReader(entrega.Payload))
	if err != nil {
		return 0, fmt.Sprintf("webhook invalido: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Dyalog-Evento", entrega.WebhookNome)
	req.Header.Set("X-Dyalog-Evento-Tipo", entrega.Evento)
	req.Header.Set("X-Dyalog-Webhook-ID", entrega.WebhookID)
	req.Header.Set("X-Dyalog-Entrega-ID", entrega.ID)
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Sprintf("falha no envio: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		corpo, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := fmt.Sprintf("status HTTP %d", resp.StatusCode)
		if len(corpo) > 0 {
			msg += ": " + string(corpo)
		}
		return resp.StatusCode, msg
	}
	return resp.StatusCode, ""
}

func (d *Dispatcher) backoff(tentativas int) time.Duration {
	if tentativas < 1 {
		tentativas = 1
	}
	multiplicador := 1 << min(tentativas-1, 6)
	intervalo := time.Duration(multiplicador) * d.intervaloBase
	if d.maxIntervalo > 0 && intervalo > d.maxIntervalo {
		return d.maxIntervalo
	}
	return intervalo
}
