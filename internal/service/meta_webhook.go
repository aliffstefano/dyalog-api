package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	webhookdispatch "dyalog-api-go/internal/webhook"
)

// ErrAssinaturaMetaInvalida indica webhook que nao veio da Meta (assinatura
// errada) ou token de verificacao incorreto.
var ErrAssinaturaMetaInvalida = errors.New("assinatura ou token de verificacao da Meta invalido")

// salvadorMidia guarda as midias recebidas (implementado pelo
// GerenciadorInstancias, o mesmo lugar das midias das instancias por QR code).
type salvadorMidia interface {
	SalvarMidiaRecebidaExterna(instanciaID, mensagemID, chatJID, remetenteJID, tipo, mimeType, nomeArquivo string, recebidaEm time.Time, dados []byte) (models.MidiaRecebida, error)
	AnexarMidiaAoPayload(dados map[string]interface{}, midia models.MidiaRecebida)
}

// MetaWebhookService recebe os webhooks da API oficial e os repassa para os
// webhooks cadastrados na instancia, no mesmo formato das instancias por QR
// code.
//
// Todo webhook e gravado no banco antes de responder 200 para a Meta: ela so
// reenvia o que nao confirmamos (por ate 7 dias), entao o que ja confirmamos
// precisa sobreviver a uma queda do container. O processamento sai do banco e
// o que ficar pendente e retomado por qualquer replica.
type MetaWebhookService struct {
	instancias  store.InstanciaStore
	metaStore   store.MetaStore
	entradas    store.MetaWebhookStore
	processadas store.MensagemProcessadaStore
	cliente     *meta.Cliente
	dispatcher  *webhookdispatch.Dispatcher
	midias      salvadorMidia
	dono        string
	// appSecret e verifyToken sao do app da Meta (provedor): valem para a URL
	// de callback unica do app, /webhook/meta, que atende todas as instancias.
	appSecret   string
	verifyToken string
	// Instagram: mesma fila de webhooks, URL /webhook/instagram.
	igStore       store.InstagramStore
	igCliente     *meta.ClienteInstagram
	igAppSecret   string
	igVerifyToken string
}

// UsarInstagram liga a URL de callback do Instagram (INSTAGRAM_APP_SECRET e
// INSTAGRAM_WEBHOOK_VERIFY_TOKEN; sem este, vale o META_WEBHOOK_VERIFY_TOKEN).
func (s *MetaWebhookService) UsarInstagram(igStore store.InstagramStore, cliente *meta.ClienteInstagram, appSecret, verifyToken string) {
	s.igStore, s.igCliente = igStore, cliente
	s.igAppSecret, s.igVerifyToken = strings.TrimSpace(appSecret), strings.TrimSpace(verifyToken)
}

func NovoMetaWebhookService(instancias store.InstanciaStore, metaStore store.MetaStore, entradas store.MetaWebhookStore, processadas store.MensagemProcessadaStore, cliente *meta.Cliente, dispatcher *webhookdispatch.Dispatcher, midias salvadorMidia) *MetaWebhookService {
	dono, _ := os.Hostname()
	return &MetaWebhookService{instancias: instancias, metaStore: metaStore, entradas: entradas, processadas: processadas, cliente: cliente, dispatcher: dispatcher, midias: midias, dono: dono}
}

// UsarApp liga a URL de callback unica do app (META_APP_SECRET e
// META_WEBHOOK_VERIFY_TOKEN).
func (s *MetaWebhookService) UsarApp(appSecret, verifyToken string) {
	s.appSecret, s.verifyToken = strings.TrimSpace(appSecret), strings.TrimSpace(verifyToken)
}

// reservaWebhook e quanto tempo um container segura um webhook antes de outro
// poder retomar (se o primeiro caiu no meio).
const reservaWebhook = 5 * time.Minute

// maxTentativasWebhook evita repetir para sempre um webhook que sempre falha.
const maxTentativasWebhook = 10

func (s *MetaWebhookService) credenciais(ctx context.Context, instanciaID string) (models.Instancia, models.CredenciaisMeta, error) {
	instancia, err := s.instancias.BuscarPorID(ctx, instanciaID)
	if err != nil || !instancia.EhMeta() {
		return models.Instancia{}, models.CredenciaisMeta{}, ErrInstanciaNaoEncontrada
	}
	cred, err := s.metaStore.ObterCredenciaisMeta(ctx, instanciaID)
	if err != nil {
		return models.Instancia{}, models.CredenciaisMeta{}, ErrInstanciaNaoEncontrada
	}
	return instancia, cred, nil
}

// Verificar responde ao GET de verificacao que a Meta faz ao cadastrar a URL:
// devolve o challenge se o token bater.
func (s *MetaWebhookService) Verificar(ctx context.Context, instanciaID, modo, token, challenge string) (string, error) {
	_, cred, err := s.credenciais(ctx, instanciaID)
	if err != nil {
		return "", err
	}
	if modo != "subscribe" || token == "" || token != cred.VerifyToken {
		return "", ErrAssinaturaMetaInvalida
	}
	return challenge, nil
}

// VerificarApp e o GET de verificacao da URL de callback unica do app.
func (s *MetaWebhookService) VerificarApp(modo, token, challenge string) (string, error) {
	if s.verifyToken == "" {
		return "", ErrInstanciaNaoEncontrada
	}
	if modo != "subscribe" || token != s.verifyToken {
		return "", ErrAssinaturaMetaInvalida
	}
	return challenge, nil
}

// VerificarInstagram e o GET de verificacao da URL do Instagram.
func (s *MetaWebhookService) VerificarInstagram(modo, token, challenge string) (string, error) {
	esperado := cmp.Or(s.igVerifyToken, s.verifyToken)
	if s.igStore == nil || esperado == "" {
		return "", ErrInstanciaNaoEncontrada
	}
	if modo != "subscribe" || token != esperado {
		return "", ErrAssinaturaMetaInvalida
	}
	return challenge, nil
}

// ReceberInstagram confere a assinatura com a chave secreta do app do
// Instagram (ou o META_APP_SECRET) e grava. A instancia sai do id da conta.
func (s *MetaWebhookService) ReceberInstagram(ctx context.Context, corpo []byte, assinatura string) error {
	if s.igStore == nil || (s.igAppSecret == "" && s.appSecret == "") {
		return ErrInstanciaNaoEncontrada
	}
	if !meta.AssinaturaValida(corpo, assinatura, s.igAppSecret) && !meta.AssinaturaValida(corpo, assinatura, s.appSecret) {
		fmt.Println("instagram: webhook recusado: assinatura nao confere com INSTAGRAM_APP_SECRET nem META_APP_SECRET")
		return ErrAssinaturaMetaInvalida
	}
	fmt.Printf("instagram: webhook recebido (%d bytes)\n", len(corpo))
	return s.guardar(ctx, "", corpo)
}

// Receber valida a assinatura, grava e responde. Sem App Secret salvo, a
// assinatura nao e conferida (o painel avisa para cadastrar).
func (s *MetaWebhookService) Receber(ctx context.Context, instanciaID string, corpo []byte, assinatura string) error {
	_, cred, err := s.credenciais(ctx, instanciaID)
	if err != nil {
		return err
	}
	segredo := cmp.Or(cred.AppSecret, s.appSecret)
	if segredo != "" && !meta.AssinaturaValida(corpo, assinatura, segredo) {
		fmt.Printf("meta: webhook recusado na instancia %s: assinatura nao confere com o App Secret salvo\n", instanciaID)
		return ErrAssinaturaMetaInvalida
	}
	fmt.Printf("meta: webhook recebido na instancia %s (%d bytes)\n", instanciaID, len(corpo))
	return s.guardar(ctx, instanciaID, corpo)
}

// ReceberApp atende a URL de callback unica do app: os eventos de todos os
// numeros chegam aqui e vao para a instancia pelo phone_number_id. Exige o
// App Secret do app, senao qualquer um mandaria mensagem em nome de qualquer
// instancia.
func (s *MetaWebhookService) ReceberApp(ctx context.Context, corpo []byte, assinatura string) error {
	if s.appSecret == "" {
		return ErrInstanciaNaoEncontrada
	}
	if !meta.AssinaturaValida(corpo, assinatura, s.appSecret) {
		fmt.Println("meta: webhook do app recusado: assinatura nao confere com META_APP_SECRET")
		return ErrAssinaturaMetaInvalida
	}
	fmt.Printf("meta: webhook do app recebido (%d bytes)\n", len(corpo))
	return s.guardar(ctx, "", corpo)
}

func (s *MetaWebhookService) guardar(ctx context.Context, instanciaID string, corpo []byte) error {
	if !json.Valid(corpo) {
		return fmt.Errorf("%w: corpo do webhook invalido", ErrEntradaInvalida)
	}
	if s.entradas == nil {
		go s.processar(instanciaID, corpo)
		return nil
	}
	// Se nao der para gravar, responde erro: a Meta reenvia depois.
	id, err := s.entradas.GuardarWebhookMeta(ctx, instanciaID, corpo)
	if err != nil {
		return err
	}
	go s.processarGuardado(store.WebhookMetaPendente{ID: id, InstanciaID: instanciaID, Corpo: corpo})
	return nil
}

func (s *MetaWebhookService) processarGuardado(p store.WebhookMetaPendente) {
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	reservado, err := s.entradas.ReservarWebhookMeta(ctx, p.ID, s.dono, reservaWebhook)
	cancelar()
	if err != nil || !reservado {
		return
	}
	erro := ""
	if p.Tentativas >= maxTentativasWebhook {
		erro = "desistiu apos varias tentativas"
	} else if err := s.processar(p.InstanciaID, p.Corpo); err != nil {
		erro = err.Error()
	}
	ctx, cancelar = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	if err := s.entradas.ConcluirWebhookMeta(ctx, p.ID, erro); err != nil {
		fmt.Printf("meta: %v\n", err)
	}
}

// Iniciar retoma os webhooks que ficaram pendentes (container caiu no meio)
// e limpa os ja processados com mais de 7 dias.
func (s *MetaWebhookService) Iniciar(ctx context.Context) {
	if s.entradas == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		ultimaLimpeza := time.Time{}
		for {
			s.retomarPendentes(ctx)
			if time.Since(ultimaLimpeza) > 24*time.Hour {
				ultimaLimpeza = time.Now()
				if apagados, err := s.entradas.LimparWebhooksMetaAntigos(ctx, time.Now().AddDate(0, 0, -7)); err != nil {
					fmt.Printf("meta: %v\n", err)
				} else if apagados > 0 {
					fmt.Printf("meta: %d webhooks processados antigos removidos\n", apagados)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *MetaWebhookService) retomarPendentes(ctx context.Context) {
	// So os parados ha mais de um minuto: os recentes ainda estao sendo
	// processados por quem recebeu.
	pendentes, err := s.entradas.ListarWebhooksMetaPendentes(ctx, time.Now().Add(-time.Minute), 50)
	if err != nil {
		fmt.Printf("meta: %v\n", err)
		return
	}
	for _, p := range pendentes {
		fmt.Printf("meta: retomando webhook pendente %s\n", p.ID)
		s.processarGuardado(p)
	}
}

// processar trata o corpo gravado. instanciaFixa vem da URL por instancia;
// vazia, o destino de cada evento sai do phone_number_id (URL unica do app).
func (s *MetaWebhookService) processar(instanciaFixa string, corpo []byte) error {
	var webhook meta.Webhook
	if err := json.Unmarshal(corpo, &webhook); err != nil {
		return fmt.Errorf("corpo do webhook invalido: %w", err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	if webhook.Objeto == "instagram" {
		return s.processarInstagram(ctx, corpo)
	}
	for _, entrada := range webhook.Entradas {
		for _, mudanca := range entrada.Mudancas {
			if mudanca.Campo != "messages" {
				fmt.Printf("meta: webhook do campo %q ignorado\n", mudanca.Campo)
				continue
			}
			valor := mudanca.Valor
			instancia, cred, ok := s.destino(ctx, instanciaFixa, valor.Metadados.PhoneNumberID)
			if !ok {
				continue
			}
			// O id da entrada e o WABA ID: guarda se ainda nao foi informado,
			// para a lista de modelos funcionar sem precisar procurar.
			if cred.WABAID == "" && entrada.ID != "" {
				cred.WABAID = entrada.ID
				if err := s.metaStore.SalvarCredenciaisMeta(ctx, cred); err != nil {
					fmt.Printf("meta: erro ao guardar o WABA ID da instancia %s: %v\n", instancia.ID, err)
				}
			}
			for _, mensagem := range valor.Mensagens {
				fmt.Printf("meta: mensagem %s recebida de %s na instancia %s\n", mensagem.ID, mensagem.De, instancia.ID)
				s.processarMensagem(ctx, instancia, cred, valor, mensagem)
			}
			for _, status := range valor.Status {
				// A Meta aceita o envio (devolve wamid) e so avisa depois, por
				// aqui, que nao entregou; fica no log para dar para investigar.
				if status.Status == "failed" {
					erros, _ := json.Marshal(status.Erros)
					fmt.Printf("meta: mensagem %s para %s nao entregue na instancia %s: %s\n", status.ID, status.Destinatario, instancia.ID, erros)
				} else {
					fmt.Printf("meta: mensagem %s para %s %s\n", status.ID, status.Destinatario, status.Status)
				}
				s.dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookRecibos, meta.DadosRecibo(status))
			}
		}
	}
	return nil
}

// destino acha a instancia do evento. Na URL por instancia, so aceita o
// numero dela (a mesma URL pode receber outro numero do app, ou o exemplo do
// botao "Testar" da Meta).
func (s *MetaWebhookService) destino(ctx context.Context, instanciaFixa, phoneNumberID string) (models.Instancia, models.CredenciaisMeta, bool) {
	if instanciaFixa != "" {
		instancia, cred, err := s.credenciais(ctx, instanciaFixa)
		if err != nil {
			fmt.Printf("meta: instancia %s do webhook nao existe mais\n", instanciaFixa)
			return models.Instancia{}, models.CredenciaisMeta{}, false
		}
		if phoneNumberID != cred.PhoneNumberID {
			fmt.Printf("meta: webhook do phone_number_id %q ignorado na instancia %s (esperado %s)\n", phoneNumberID, instanciaFixa, cred.PhoneNumberID)
			return models.Instancia{}, models.CredenciaisMeta{}, false
		}
		return instancia, cred, true
	}
	cred, err := s.metaStore.ObterCredenciaisMetaPorNumero(ctx, phoneNumberID)
	if err != nil {
		fmt.Printf("meta: webhook do phone_number_id %q ignorado: nenhuma instancia com esse numero\n", phoneNumberID)
		return models.Instancia{}, models.CredenciaisMeta{}, false
	}
	instancia, err := s.instancias.BuscarPorID(ctx, cred.InstanciaID)
	if err != nil {
		return models.Instancia{}, models.CredenciaisMeta{}, false
	}
	return instancia, cred, true
}

func (s *MetaWebhookService) processarMensagem(ctx context.Context, instancia models.Instancia, cred models.CredenciaisMeta, valor meta.Valor, mensagem meta.Mensagem) {
	jid := mensagem.De + "@s.whatsapp.net"
	dados := meta.DadosMensagem(valor, mensagem)
	recebida, _ := dados["recebida_em"].(time.Time)
	if s.processadas != nil && mensagem.ID != "" {
		inserida, err := s.processadas.RegistrarMensagemProcessada(ctx, models.MensagemProcessada{
			InstanciaID:  instancia.ID,
			ChatJID:      jid,
			MensagemID:   mensagem.ID,
			RemetenteJID: jid,
			RecebidaEm:   recebida,
			Origem:       "meta",
			ProcessadaEm: time.Now().UTC(),
		})
		if err == nil && !inserida {
			return // reentrega da Meta
		}
	}
	if midia, tipo := mensagem.MidiaDe(); midia != nil && s.midias != nil {
		if err := s.anexarMidia(ctx, instancia.ID, cred, mensagem, midia, tipo, recebida, dados); err != nil {
			dados["midia_erro"] = err.Error()
		}
	}
	if instancia.MarcarLidaAutomatico {
		if err := s.cliente.MarcarLida(ctx, cred, mensagem.ID); err != nil {
			fmt.Printf("meta: erro ao marcar como lida na instancia %s: %v\n", instancia.ID, err)
		}
	}
	s.dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookMensagens, dados)
}

func (s *MetaWebhookService) anexarMidia(ctx context.Context, instanciaID string, cred models.CredenciaisMeta, mensagem meta.Mensagem, midia *meta.Midia, tipo string, recebida time.Time, dados map[string]any) error {
	arquivo, mimeType, err := s.cliente.BaixarMidia(ctx, cred, midia.ID)
	if err != nil {
		return err
	}
	jid := mensagem.De + "@s.whatsapp.net"
	salva, err := s.midias.SalvarMidiaRecebidaExterna(instanciaID, mensagem.ID, jid, jid, tipo, cmp.Or(midia.MimeType, mimeType), midia.NomeArquivo, recebida, arquivo)
	if err != nil {
		return err
	}
	s.midias.AnexarMidiaAoPayload(dados, salva)
	return nil
}

// processarInstagram trata o webhook do Direct: cada entry e uma conta
// profissional, cada messaging um evento.
func (s *MetaWebhookService) processarInstagram(ctx context.Context, corpo []byte) error {
	if s.igStore == nil {
		return nil
	}
	var webhook meta.WebhookInstagram
	if err := json.Unmarshal(corpo, &webhook); err != nil {
		return fmt.Errorf("corpo do webhook do Instagram invalido: %w", err)
	}
	for _, entrada := range webhook.Entradas {
		cred, err := s.igStore.ObterCredenciaisInstagramPorConta(ctx, entrada.ID)
		if err != nil {
			fmt.Printf("instagram: webhook da conta %q ignorado: nenhuma instancia com essa conta\n", entrada.ID)
			continue
		}
		instancia, err := s.instancias.BuscarPorID(ctx, cred.InstanciaID)
		if err != nil {
			continue
		}
		for _, evento := range entrada.Eventos {
			s.processarEventoInstagram(ctx, instancia, cred, evento)
		}
	}
	return nil
}

func (s *MetaWebhookService) processarEventoInstagram(ctx context.Context, instancia models.Instancia, cred models.CredenciaisInstagram, evento meta.EventoInstagram) {
	if evento.Lida != nil {
		s.dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookRecibos, meta.DadosReciboInstagram(evento))
		return
	}
	eco := evento.Mensagem != nil && evento.Mensagem.Eco
	nome, usuario := "", ""
	if !eco {
		nome, usuario = s.igCliente.NomeContato(ctx, cred.AccessToken, evento.Remetente.ID)
	}
	dados := meta.DadosMensagemInstagram(evento, nome, usuario)
	if dados == nil {
		return
	}
	id, _ := dados["mensagem_id"].(string)
	recebida, _ := dados["recebida_em"].(time.Time)
	jid := evento.Contato() + meta.SufixoInstagram
	// A mensagem apagada vem com o mesmo mid da original: nao entra no
	// controle de reentrega.
	if s.processadas != nil && id != "" && dados["acao"] != "apagada" {
		inserida, err := s.processadas.RegistrarMensagemProcessada(ctx, models.MensagemProcessada{
			InstanciaID: instancia.ID, ChatJID: jid, MensagemID: id, RemetenteJID: jid,
			RecebidaEm: recebida, Origem: "instagram", ProcessadaEm: time.Now().UTC(),
		})
		if err == nil && !inserida {
			return // reentrega
		}
	}
	fmt.Printf("instagram: %s %s de %s na instancia %s\n", dados["tipo"], id, evento.Contato(), instancia.ID)
	if endereco, tipo := evento.AnexoInstagram(); endereco != "" && s.midias != nil {
		if err := s.anexarMidiaInstagram(ctx, instancia.ID, id, jid, endereco, tipo, recebida, dados); err != nil {
			dados["midia_erro"] = err.Error()
		}
	}
	if instancia.MarcarLidaAutomatico && !eco && evento.Mensagem != nil {
		if _, err := s.igCliente.Enviar(ctx, cred.AccessToken, evento.Remetente.ID, map[string]any{"sender_action": "mark_seen"}); err != nil {
			fmt.Printf("instagram: erro ao marcar como vista na instancia %s: %v\n", instancia.ID, err)
		}
	}
	s.dispatcher.DispararEvento(ctx, instancia.ID, models.EventoWebhookMensagens, dados)
}

func (s *MetaWebhookService) anexarMidiaInstagram(ctx context.Context, instanciaID, mensagemID, jid, endereco, tipo string, recebida time.Time, dados map[string]any) error {
	arquivo, mimeType, err := s.igCliente.BaixarURL(ctx, endereco)
	if err != nil {
		return err
	}
	salva, err := s.midias.SalvarMidiaRecebidaExterna(instanciaID, mensagemID, jid, jid, tipo, mimeType, "", recebida, arquivo)
	if err != nil {
		return err
	}
	s.midias.AnexarMidiaAoPayload(dados, salva)
	return nil
}
