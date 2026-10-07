package service

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
type MetaWebhookService struct {
	instancias  store.InstanciaStore
	metaStore   store.MetaStore
	processadas store.MensagemProcessadaStore
	cliente     *meta.Cliente
	dispatcher  *webhookdispatch.Dispatcher
	midias      salvadorMidia
}

func NovoMetaWebhookService(instancias store.InstanciaStore, metaStore store.MetaStore, processadas store.MensagemProcessadaStore, cliente *meta.Cliente, dispatcher *webhookdispatch.Dispatcher, midias salvadorMidia) *MetaWebhookService {
	return &MetaWebhookService{instancias: instancias, metaStore: metaStore, processadas: processadas, cliente: cliente, dispatcher: dispatcher, midias: midias}
}

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

// Receber valida a assinatura e processa o webhook. Sem App Secret salvo, a
// assinatura nao e conferida (o painel avisa para cadastrar).
func (s *MetaWebhookService) Receber(ctx context.Context, instanciaID string, corpo []byte, assinatura string) error {
	instancia, cred, err := s.credenciais(ctx, instanciaID)
	if err != nil {
		return err
	}
	if cred.AppSecret != "" && !meta.AssinaturaValida(corpo, assinatura, cred.AppSecret) {
		return ErrAssinaturaMetaInvalida
	}
	var webhook meta.Webhook
	if err := json.Unmarshal(corpo, &webhook); err != nil {
		return fmt.Errorf("%w: corpo do webhook invalido", ErrEntradaInvalida)
	}
	// A Meta espera resposta rapida (e reenvia se demorar); baixar midia e
	// repassar ficam em segundo plano.
	go s.processar(instancia, cred, webhook)
	return nil
}

func (s *MetaWebhookService) processar(instancia models.Instancia, cred models.CredenciaisMeta, webhook meta.Webhook) {
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelar()
	for _, entrada := range webhook.Entradas {
		for _, mudanca := range entrada.Mudancas {
			if mudanca.Campo != "messages" {
				continue
			}
			valor := mudanca.Valor
			// Uma mesma URL pode receber eventos de outro numero do app; so
			// interessa o numero desta instancia.
			if id := valor.Metadados.PhoneNumberID; id != "" && id != cred.PhoneNumberID {
				continue
			}
			for _, mensagem := range valor.Mensagens {
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
