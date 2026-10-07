package meta

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
	"dyalog-api-go/internal/whatsapp"
)

// Enviador faz, pela API oficial, os mesmos envios que o GerenciadorInstancias
// faz pelo whatsmeow. Assim as rotas /batepapo/... servem os dois tipos de
// instancia sem o cliente precisar mudar nada.
type Enviador struct {
	cliente *Cliente
	store   store.MetaStore
}

func NovoEnviador(cliente *Cliente, metaStore store.MetaStore) *Enviador {
	return &Enviador{cliente: cliente, store: metaStore}
}

func (e *Enviador) Cliente() *Cliente { return e.cliente }

// Credenciais busca as credenciais da instancia, com erro claro quando ainda
// nao foram cadastradas.
func (e *Enviador) Credenciais(ctx context.Context, instanciaID string) (models.CredenciaisMeta, error) {
	cred, err := e.store.ObterCredenciaisMeta(ctx, instanciaID)
	if errors.Is(err, store.ErrCredenciaisMetaNaoEncontradas) || (err == nil && (cred.AccessToken == "" || cred.PhoneNumberID == "")) {
		return models.CredenciaisMeta{}, fmt.Errorf("instancia da API oficial sem credenciais: cadastre phone_number_id e access_token")
	}
	return cred, err
}

// enviar resolve credenciais e destino, manda a mensagem e monta o resultado
// no formato das demais rotas.
func (e *Enviador) enviar(ctx context.Context, instanciaID, numero, chatJID, tipo string, montar func(para string) (map[string]any, error)) (models.ResultadoEnvio, error) {
	cred, err := e.Credenciais(ctx, instanciaID)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	para, err := Destinatario(numero, chatJID)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	mensagem, err := montar(para)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	resp, err := e.cliente.Enviar(ctx, cred, mensagem)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	return models.ResultadoEnvio{
		Instancia:  instanciaID,
		Numero:     numero,
		ChatJID:    cmp.Or(resp.WaID(), para) + "@s.whatsapp.net",
		MensagemID: resp.MensagemID(),
		Status:     "enviada",
		Tipo:       tipo,
		Modo:       "meta",
	}, nil
}

func (e *Enviador) EnviarTexto(ctx context.Context, req models.EnvioTextoRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "texto", func(para string) (map[string]any, error) {
		return MontarTexto(para, req.Mensagem, req.RespostaMensagemID), nil
	})
}

func (e *Enviador) EnviarTemplate(ctx context.Context, req models.EnvioTemplateRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "template", func(para string) (map[string]any, error) {
		return MontarTemplate(para, req), nil
	})
}

func (e *Enviador) ReagirMensagem(ctx context.Context, req models.ReagirMensagemRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "reacao", func(para string) (map[string]any, error) {
		return MontarReacao(para, req.MensagemID, req.Emoji), nil
	})
}

func (e *Enviador) EnviarLocalizacao(ctx context.Context, req models.EnvioLocalizacaoRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "localizacao", func(para string) (map[string]any, error) {
		return MontarLocalizacao(para, req), nil
	})
}

func (e *Enviador) EnviarContato(ctx context.Context, req models.EnvioContatoRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "contato", func(para string) (map[string]any, error) {
		return MontarContatos(para, req), nil
	})
}

func (e *Enviador) EnviarBotoes(ctx context.Context, req models.EnvioBotoesRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "botoes", func(para string) (map[string]any, error) {
		return MontarBotoes(para, req, cmp.Or(strings.TrimSpace(req.Texto), strings.TrimSpace(req.Mensagem)))
	})
}

func (e *Enviador) EnviarLista(ctx context.Context, req models.EnvioListaRequest) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, "lista", func(para string) (map[string]any, error) {
		return MontarLista(para, req, cmp.Or(strings.TrimSpace(req.Descricao), strings.TrimSpace(req.Mensagem)))
	})
}

func (e *Enviador) EnviarImagem(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "image", "imagem")
}

func (e *Enviador) EnviarAudio(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "audio", "audio")
}

func (e *Enviador) EnviarDocumento(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "document", "documento")
}

func (e *Enviador) EnviarFigurinha(ctx context.Context, req models.EnvioMidiaRequest) (models.ResultadoEnvio, error) {
	return e.enviarMidia(ctx, req, "sticker", "figurinha")
}

// enviarMidia manda a URL publica direto para a Meta baixar; base64 e arquivo
// local sao subidos antes.
func (e *Enviador) enviarMidia(ctx context.Context, req models.EnvioMidiaRequest, tipoMeta, tipo string) (models.ResultadoEnvio, error) {
	return e.enviar(ctx, req.Instancia, req.Numero, req.ChatJID, tipo, func(para string) (map[string]any, error) {
		nomeArquivo := strings.TrimSpace(req.NomeArquivo)
		if url := strings.TrimSpace(req.ArquivoURL); url != "" {
			return MontarMidia(para, tipoMeta, map[string]any{"link": url}, req.Legenda, nomeArquivo), nil
		}
		dados, nome, mimeType, err := whatsapp.CarregarMidiaEnvio(ctx, req)
		if err != nil {
			return nil, err
		}
		cred, err := e.Credenciais(ctx, req.Instancia)
		if err != nil {
			return nil, err
		}
		id, err := e.cliente.SubirMidia(ctx, cred, dados, cmp.Or(nome, "arquivo"), mimeType)
		if err != nil {
			return nil, fmt.Errorf("erro ao subir midia na Meta: %w", err)
		}
		return MontarMidia(para, tipoMeta, map[string]any{"id": id}, req.Legenda, cmp.Or(nomeArquivo, nome)), nil
	})
}

func (e *Enviador) MarcarLida(ctx context.Context, req models.MarcarLidaRequest) (models.ResultadoMarcarLida, error) {
	cred, err := e.Credenciais(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoMarcarLida{}, err
	}
	ids := req.MensagensID
	if id := strings.TrimSpace(req.MensagemID); id != "" {
		ids = append([]string{id}, ids...)
	}
	for _, id := range ids {
		if err := e.cliente.MarcarLida(ctx, cred, id); err != nil {
			return models.ResultadoMarcarLida{}, err
		}
	}
	return models.ResultadoMarcarLida{Instancia: req.Instancia, Numero: req.Numero, ChatJID: req.ChatJID, MensagensID: ids, Status: "lida", Tipo: "read", LidaEm: time.Now().UTC()}, nil
}

// Recursos que so existem nas instancias por QR code.

func naoSuportado(recurso string) error {
	return fmt.Errorf("%w: %s", ErrNaoSuportado, recurso)
}

func (e *Enviador) EditarTexto(context.Context, models.EditarTextoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("editar mensagem")
}

func (e *Enviador) ApagarMensagem(context.Context, models.ApagarMensagemRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("apagar mensagem")
}

func (e *Enviador) EnviarPresenca(context.Context, models.EnvioPresencaRequest) (models.ResultadoPresenca, error) {
	return models.ResultadoPresenca{}, naoSuportado("presenca (digitando/gravando)")
}

func (e *Enviador) EnviarEnquete(context.Context, models.EnvioEnqueteRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("enquete; use lista ou botoes")
}

func (e *Enviador) EnviarCobrancaPix(context.Context, models.EnvioCobrancaPixRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("cobranca Pix")
}

func (e *Enviador) EnviarEvento(context.Context, models.EnvioEventoRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("evento")
}

func (e *Enviador) EnviarCarrossel(context.Context, models.EnvioCarrosselRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("carrossel fora de template")
}

func (e *Enviador) PostarStatus(context.Context, models.EnvioStatusRequest) (models.ResultadoEnvio, error) {
	return models.ResultadoEnvio{}, naoSuportado("status")
}

func (e *Enviador) CriarLinkChamada(context.Context, models.LinkChamadaRequest) (models.LinkChamadaResultado, error) {
	return models.LinkChamadaResultado{}, naoSuportado("link de chamada")
}
