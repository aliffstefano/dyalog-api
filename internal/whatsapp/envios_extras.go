package whatsapp

import (
	"cmp"
	"context"
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"

	"dyalog-api-go/internal/models"
)

// EnviarEvento manda um convite de evento (o mesmo do menu "Evento" do app),
// com nome, horario, local e link de chamada opcionais. O destinatario
// responde "Vou"/"Nao vou" no proprio cartao.
//
// Depende do ajuste de build (tools/whatsmeow-overlay): o stanza precisa sair
// com type="event" e <meta event_type="creation"/>; o whatsmeow puro manda
// como texto e o evento nao aparece.
func (g *GerenciadorInstancias) EnviarEvento(ctx context.Context, req models.EnvioEventoRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	msg, err := montarMensagemEvento(req)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	resp, jidUsado, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
		return runtime.client.SendMessage(ctx, jid, msg)
	})
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar evento no WhatsApp: %w", err)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: cmp.Or(req.MensagemID, string(resp.ID)),
		Status:     "enviada",
		Tipo:       "evento",
	}, nil
}

func montarMensagemEvento(req models.EnvioEventoRequest) (*waE2E.Message, error) {
	// Como na enquete, o segredo da mensagem e o que permite ler as respostas
	// (Vou/Nao vou) criptografadas que voltam.
	segredo := make([]byte, 32)
	if _, err := rand.Read(segredo); err != nil {
		return nil, fmt.Errorf("erro ao gerar segredo do evento: %w", err)
	}
	evento := &waE2E.EventMessage{
		Name:               proto.String(strings.TrimSpace(req.Nome)),
		StartTime:          proto.Int64(req.InicioEm.Unix()),
		IsCanceled:         proto.Bool(false),
		IsScheduleCall:     proto.Bool(req.ChamadaAgendada),
		ExtraGuestsAllowed: proto.Bool(req.PermitirAcompanhantes),
		ContextInfo:        contextInfoResposta(req.RespostaMensagemID, req.RespostaParticipante),
	}
	if req.Lembrete {
		evento.HasReminder = proto.Bool(true)
		evento.ReminderOffsetSec = proto.Int64(req.LembreteSegundos)
	}
	if descricao := strings.TrimSpace(req.Descricao); descricao != "" {
		evento.Description = proto.String(descricao)
	}
	if !req.FimEm.IsZero() {
		evento.EndTime = proto.Int64(req.FimEm.Unix())
	}
	if link := strings.TrimSpace(req.LinkChamada); link != "" {
		evento.JoinLink = proto.String(link)
	}
	local, endereco := strings.TrimSpace(req.Local), strings.TrimSpace(req.Endereco)
	if local != "" || endereco != "" || req.Latitude != 0 || req.Longitude != 0 {
		evento.Location = &waE2E.LocationMessage{}
		// Sem coordenadas, so nome/endereco: 0,0 colocaria o evento no oceano.
		if req.Latitude != 0 || req.Longitude != 0 {
			evento.Location.DegreesLatitude = proto.Float64(req.Latitude)
			evento.Location.DegreesLongitude = proto.Float64(req.Longitude)
		}
		if local != "" {
			evento.Location.Name = proto.String(local)
		}
		if endereco != "" {
			evento.Location.Address = proto.String(endereco)
		}
	}
	// MessageContextInfo igual ao do evento criado no app oficial (capturado):
	// segredo + deviceListMetadataVersion 2.
	return &waE2E.Message{
		EventMessage: evento,
		MessageContextInfo: &waE2E.MessageContextInfo{
			MessageSecret:             segredo,
			DeviceListMetadataVersion: proto.Int32(2),
		},
	}, nil
}

// PostarStatus publica um status (stories) de texto, imagem ou video. Quem ve
// e quem a privacidade de status do numero permitir (por padrao, os contatos
// salvos no celular).
func (g *GerenciadorInstancias) PostarStatus(ctx context.Context, req models.EnvioStatusRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	var msg *waE2E.Message
	tipo := tipoStatus(req)
	if tipo == "texto" {
		msg = montarStatusTexto(req)
	} else {
		msg, err = montarStatusMidia(ctx, runtime.client, req, tipo)
		if err != nil {
			return models.ResultadoEnvio{}, err
		}
	}
	resp, err := runtime.client.SendMessage(ctx, types.StatusBroadcastJID, msg)
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao postar status no WhatsApp: %w", err)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		ChatJID:    types.StatusBroadcastJID.String(),
		MensagemID: cmp.Or(req.MensagemID, string(resp.ID)),
		Status:     "enviada",
		Tipo:       "status_" + tipo,
	}, nil
}

// tipoStatus devolve texto, imagem ou video.
func tipoStatus(req models.EnvioStatusRequest) string {
	switch strings.ToLower(strings.TrimSpace(req.Tipo)) {
	case "imagem", "image", "foto":
		return "imagem"
	case "video":
		return "video"
	case "texto", "text":
		return "texto"
	}
	if req.ArquivoURL != "" || req.ArquivoBase64 != "" || req.CaminhoLocal != "" {
		if strings.HasPrefix(strings.ToLower(req.MimeType), "video/") {
			return "video"
		}
		return "imagem"
	}
	return "texto"
}

// corStatusPadrao e o verde escuro que o app usa como primeira opcao.
const corStatusPadrao = 0xFF128C7E

func montarStatusTexto(req models.EnvioStatusRequest) *waE2E.Message {
	fonte := waE2E.ExtendedTextMessage_FontType(min(max(req.Fonte, 0), 9))
	return &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
		Text:           proto.String(strings.TrimSpace(req.Texto)),
		BackgroundArgb: proto.Uint32(corARGB(req.CorFundo, corStatusPadrao)),
		TextArgb:       proto.Uint32(0xFFFFFFFF),
		Font:           fonte.Enum(),
	}}
}

// corARGB converte #RRGGBB no formato ARGB do WhatsApp (opaco). Cor vazia ou
// invalida devolve o padrao.
func corARGB(cor string, padrao uint32) uint32 {
	cor = strings.TrimPrefix(strings.TrimSpace(cor), "#")
	if len(cor) != 6 {
		return padrao
	}
	valor, err := strconv.ParseUint(cor, 16, 32)
	if err != nil {
		return padrao
	}
	return 0xFF000000 | uint32(valor)
}

func montarStatusMidia(ctx context.Context, client *whatsmeow.Client, req models.EnvioStatusRequest, tipo string) (*waE2E.Message, error) {
	midia := models.EnvioMidiaRequest{
		ArquivoURL:    req.ArquivoURL,
		ArquivoBase64: req.ArquivoBase64,
		CaminhoLocal:  req.CaminhoLocal,
		Legenda:       req.Legenda,
		MimeType:      req.MimeType,
	}
	if tipo == "imagem" {
		dados, mimeType, err := carregarImagemEnvio(ctx, midia)
		if err != nil {
			return nil, err
		}
		largura, altura := dimensoesImagem(dados)
		upload, err := client.Upload(ctx, dados, whatsmeow.MediaImage)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer upload da imagem do status: %w", err)
		}
		return montarMensagemImagem(midia, mimeType, upload, largura, altura), nil
	}

	dados, nomeArquivo, mimeCabecalho, err := carregarConteudoMidia(ctx, midia)
	if err != nil {
		return nil, err
	}
	mimeType := detectarMimeType(nomeArquivo, cmp.Or(req.MimeType, mimeCabecalho), dados)
	if !strings.HasPrefix(mimeType, "video/") {
		return nil, fmt.Errorf("%w: arquivo informado nao e um video", ErrMidiaInvalida)
	}
	upload, err := client.Upload(ctx, dados, whatsmeow.MediaVideo)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload do video do status: %w", err)
	}
	video := &waE2E.VideoMessage{
		Mimetype:      proto.String(mimeType),
		URL:           &upload.URL,
		DirectPath:    &upload.DirectPath,
		MediaKey:      upload.MediaKey,
		FileEncSHA256: upload.FileEncSHA256,
		FileSHA256:    upload.FileSHA256,
		FileLength:    &upload.FileLength,
	}
	if legenda := strings.TrimSpace(req.Legenda); legenda != "" {
		video.Caption = proto.String(legenda)
	}
	return &waE2E.Message{VideoMessage: video}, nil
}

// EnviarCarrossel manda cartoes deslizaveis, cada um com imagem, texto e
// botoes. Usa o mesmo envelope dos botoes; se o servidor recusar, os cartoes
// saem como texto.
func (g *GerenciadorInstancias) EnviarCarrossel(ctx context.Context, req models.EnvioCarrosselRequest) (models.ResultadoEnvio, error) {
	runtime, err := g.obterRuntimeConectado(ctx, req.Instancia)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, req.Grupo)
	if err != nil {
		return models.ResultadoEnvio{}, err
	}
	if req.FallbackTexto {
		return g.enviarCarrosselComoTexto(ctx, runtime.client, req, jids, "")
	}

	cartoes := make([]*waE2E.InteractiveMessage, 0, len(req.Cartoes))
	for i, cartao := range req.Cartoes {
		montado, err := montarCartaoCarrossel(ctx, runtime.client, cartao)
		if err != nil {
			return models.ResultadoEnvio{}, fmt.Errorf("cartao %d: %w", i+1, err)
		}
		cartoes = append(cartoes, montado)
	}
	msg := montarMensagemCarrossel(req, cartoes)

	resp, jidUsado, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
		return enviarMensagemInterativa(ctx, runtime.client, jid, msg)
	})
	if err != nil {
		if erroInterativoNaoPermitido(err) {
			observacao := fmt.Sprintf("Carrossel rejeitado pelo servidor do WhatsApp (%v); cartoes enviados automaticamente como texto.", err)
			if resultado, errTexto := g.enviarCarrosselComoTexto(ctx, runtime.client, req, jids, observacao); errTexto == nil {
				return resultado, nil
			}
		}
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar carrossel no WhatsApp: %w", err)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: cmp.Or(req.MensagemID, string(resp.ID)),
		Status:     "aceita_pelo_servidor",
		Tipo:       "carrossel",
		Modo:       "native_flow",
		Observacao: "O WhatsApp retornou ID para o carrossel, mas o cliente ainda pode nao renderizar os cartoes.",
	}, nil
}

func montarCartaoCarrossel(ctx context.Context, client *whatsmeow.Client, cartao models.CarrosselCartaoRequest) (*waE2E.InteractiveMessage, error) {
	midia := models.EnvioMidiaRequest{ArquivoURL: cartao.ImagemURL, ArquivoBase64: cartao.ImagemBase64}
	dados, mimeType, err := carregarImagemEnvio(ctx, midia)
	if err != nil {
		return nil, err
	}
	largura, altura := dimensoesImagem(dados)
	upload, err := client.Upload(ctx, dados, whatsmeow.MediaImage)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer upload da imagem: %w", err)
	}
	botoes, err := botoesNativeFlow(cartao.Botoes)
	if err != nil {
		return nil, err
	}
	return &waE2E.InteractiveMessage{
		Header: &waE2E.InteractiveMessage_Header{
			Title:              proto.String(strings.TrimSpace(cartao.Titulo)),
			HasMediaAttachment: proto.Bool(true),
			Media: &waE2E.InteractiveMessage_Header_ImageMessage{
				ImageMessage: montarMensagemImagem(midia, mimeType, upload, largura, altura).GetImageMessage(),
			},
		},
		Body:   &waE2E.InteractiveMessage_Body{Text: proto.String(strings.TrimSpace(cartao.Texto))},
		Footer: &waE2E.InteractiveMessage_Footer{Text: proto.String(strings.TrimSpace(cartao.Rodape))},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons:           botoes,
				MessageParamsJSON: proto.String(""),
				MessageVersion:    proto.Int32(1),
			},
		},
	}, nil
}

func montarMensagemCarrossel(req models.EnvioCarrosselRequest, cartoes []*waE2E.InteractiveMessage) *waE2E.Message {
	return &waE2E.Message{
		InteractiveMessage: &waE2E.InteractiveMessage{
			Body:   &waE2E.InteractiveMessage_Body{Text: proto.String(strings.TrimSpace(req.Texto))},
			Footer: &waE2E.InteractiveMessage_Footer{Text: proto.String(strings.TrimSpace(req.Rodape))},
			InteractiveMessage: &waE2E.InteractiveMessage_CarouselMessage_{
				CarouselMessage: &waE2E.InteractiveMessage_CarouselMessage{
					Cards:          cartoes,
					MessageVersion: proto.Int32(1),
				},
			},
			ContextInfo: cmp.Or(contextInfoResposta(req.RespostaMensagemID, req.RespostaParticipante), &waE2E.ContextInfo{}),
		},
		MessageContextInfo: contextInfoMensagemInterativa(),
	}
}

func (g *GerenciadorInstancias) enviarCarrosselComoTexto(ctx context.Context, client *whatsmeow.Client, req models.EnvioCarrosselRequest, jids []types.JID, observacao string) (models.ResultadoEnvio, error) {
	msg := &waE2E.Message{Conversation: proto.String(textoFallbackCarrossel(req))}
	resp, jidUsado, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
		return client.SendMessage(ctx, jid, msg)
	})
	if err != nil {
		return models.ResultadoEnvio{}, fmt.Errorf("erro ao enviar carrossel como texto no WhatsApp: %w", err)
	}
	return models.ResultadoEnvio{
		Instancia:  req.Instancia,
		Numero:     req.Numero,
		ChatJID:    jidUsado.String(),
		MensagemID: cmp.Or(req.MensagemID, string(resp.ID)),
		Status:     "enviada",
		Tipo:       "carrossel",
		Modo:       "texto",
		Observacao: cmp.Or(observacao, "Carrossel enviado como texto."),
	}, nil
}

func textoFallbackCarrossel(req models.EnvioCarrosselRequest) string {
	partes := []string{}
	if texto := strings.TrimSpace(req.Texto); texto != "" {
		partes = append(partes, texto)
	}
	for _, cartao := range req.Cartoes {
		linhas := []string{}
		if titulo := strings.TrimSpace(cartao.Titulo); titulo != "" {
			linhas = append(linhas, "*"+titulo+"*")
		}
		if texto := strings.TrimSpace(cartao.Texto); texto != "" {
			linhas = append(linhas, texto)
		}
		for _, botao := range cartao.Botoes {
			linhas = append(linhas, "- "+textoBotaoFallback(botao))
		}
		partes = append(partes, strings.Join(linhas, "\n"))
	}
	if rodape := strings.TrimSpace(req.Rodape); rodape != "" {
		partes = append(partes, rodape)
	}
	return strings.Join(partes, "\n\n")
}
