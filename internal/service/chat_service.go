package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// ChatService alimenta a tela de Chat do painel. As recebidas vem dos mesmos
// eventos que vao para os webhooks (QR code e API oficial); as enviadas, do
// resultado dos envios pela API. Fica no banco para qualquer replica mostrar.
type ChatService struct {
	store store.ChatStore
}

func NovoChatService(chatStore store.ChatStore) *ChatService {
	return &ChatService{store: chatStore}
}

func (s *ChatService) ListarConversas(ctx context.Context, instanciaID string) ([]models.ChatConversa, error) {
	return s.store.ListarConversasChat(ctx, instanciaID, 100)
}

func (s *ChatService) ListarMensagens(ctx context.Context, instanciaID, chatJID string) ([]models.ChatMensagem, error) {
	chatJID = strings.TrimSpace(chatJID)
	if chatJID == "" {
		return nil, fmt.Errorf("%w: informe chat_jid", ErrEntradaInvalida)
	}
	return s.store.ListarMensagensChat(ctx, instanciaID, chatJID, 200)
}

// RegistrarEnvio guarda a mensagem enviada pela API. Roda em segundo plano:
// o chat nunca atrasa nem derruba o envio.
func (s *ChatService) RegistrarEnvio(instanciaID string, resultado models.ResultadoEnvio, conteudo string) {
	if s == nil || resultado.MensagemID == "" {
		return
	}
	jid := jidConversa(resultado.ChatJID, resultado.Numero)
	if jid == "" || jid == "status@broadcast" {
		return
	}
	mensagem := models.ChatMensagem{
		InstanciaID: instanciaID,
		ChatJID:     jid,
		MensagemID:  resultado.MensagemID,
		Direcao:     "saida",
		Tipo:        resultado.Tipo,
		Conteudo:    conteudo,
		Status:      "enviada",
		CriadoEm:    time.Now(),
	}
	go func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelar()
		if err := s.store.RegistrarChatMensagem(ctx, mensagem); err != nil {
			fmt.Printf("chat: %v\n", err)
		}
	}()
}

// RegistrarEvento recebe os eventos de webhook (mensagens e recibos) e guarda
// o que a tela de Chat precisa.
func (s *ChatService) RegistrarEvento(ctx context.Context, instanciaID, evento string, dados any) {
	if s == nil {
		return
	}
	mapa, ok := dados.(map[string]any)
	if !ok {
		return
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancelar()
	var err error
	switch evento {
	case models.EventoWebhookMensagens:
		err = s.registrarMensagem(ctx, instanciaID, mapa)
	case models.EventoWebhookRecibos:
		err = s.store.AtualizarStatusChat(ctx, instanciaID, textos(mapa["mensagens_id"]), texto(mapa["status"]), erroRecibo(mapa["erros"]))
	}
	if err != nil {
		fmt.Printf("chat: %v\n", err)
	}
}

func (s *ChatService) registrarMensagem(ctx context.Context, instanciaID string, d map[string]any) error {
	// Historico sincronizado na conexao encheria o chat de conversas antigas.
	if historico, _ := d["historico"].(bool); historico {
		return nil
	}
	jid := texto(d["chat_jid"])
	if !strings.HasSuffix(jid, "@g.us") {
		jid = jidConversa(jid, texto(d["chat_numero"]))
	}
	if jid == "" || strings.HasSuffix(jid, "@broadcast") || strings.HasSuffix(jid, "@newsletter") {
		return nil
	}
	mensagemID := texto(d["mensagem_id"])
	switch texto(d["acao"]) {
	case "editada":
		original := texto(d["mensagem_original_id"])
		if original == "" {
			return nil
		}
		return s.store.EditarChatMensagem(ctx, instanciaID, original, "", texto(d["conteudo"]))
	case "apagada":
		return s.store.EditarChatMensagem(ctx, instanciaID, mensagemID, "apagada", "mensagem apagada")
	}
	if mensagemID == "" {
		return nil
	}
	direcao := texto(d["direcao"])
	status := ""
	if direcao == "saida" {
		status = "enviada"
	}
	criado, _ := d["recebida_em"].(time.Time)
	return s.store.RegistrarChatMensagem(ctx, models.ChatMensagem{
		InstanciaID:  instanciaID,
		ChatJID:      jid,
		MensagemID:   mensagemID,
		Direcao:      direcao,
		Tipo:         texto(d["tipo"]),
		Conteudo:     texto(d["conteudo"]),
		Nome:         texto(d["nome_remetente"]),
		RemetenteJID: texto(d["remetente_jid"]),
		Status:       status,
		MidiaID:      texto(d["midia_id"]),
		MimeType:     texto(d["mime_type"]),
		NomeArquivo:  texto(d["nome_arquivo"]),
		CriadoEm:     criado,
	})
}

// IniciarLimpeza apaga uma vez por dia as mensagens mais antigas que a
// retencao.
func (s *ChatService) IniciarLimpeza(ctx context.Context, retencaoDias int) {
	limpar := func() {
		apagadas, err := s.store.LimparChatAntigo(context.Background(), time.Now().AddDate(0, 0, -retencaoDias))
		if err != nil {
			fmt.Printf("chat: %v\n", err)
		} else if apagadas > 0 {
			fmt.Printf("chat: %d mensagens antigas removidas (retencao %dd)\n", apagadas, retencaoDias)
		}
	}
	go func() {
		limpar()
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				limpar()
			}
		}
	}()
}

// jidConversa junta numa conversa so o que chega como @lid e o que foi
// enviado pelo numero: com o numero conhecido, a chave e numero@s.whatsapp.net.
func jidConversa(jid, numero string) string {
	jid = strings.TrimSpace(jid)
	if strings.HasSuffix(jid, "@g.us") || strings.HasSuffix(jid, "@s.whatsapp.net") || strings.HasSuffix(jid, "@broadcast") {
		return jid
	}
	if digitos := soDigitos(numero); digitos != "" {
		return digitos + "@s.whatsapp.net"
	}
	return jid
}

func soDigitos(valor string) string {
	var b strings.Builder
	for _, r := range valor {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func texto(valor any) string {
	switch v := valor.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func textos(valor any) []string {
	switch v := valor.(type) {
	case []string:
		return v
	case []any:
		lista := make([]string, 0, len(v))
		for _, item := range v {
			if t := texto(item); t != "" {
				lista = append(lista, t)
			}
		}
		return lista
	default:
		return nil
	}
}

// erroRecibo tira o motivo do recibo de falha da API oficial.
func erroRecibo(valor any) string {
	erros, ok := valor.([]map[string]any)
	if !ok || len(erros) == 0 {
		return ""
	}
	motivo := texto(erros[0]["title"])
	if motivo == "" {
		motivo = texto(erros[0]["message"])
	}
	if codigo, ok := erros[0]["code"].(float64); ok {
		motivo = fmt.Sprintf("%d: %s", int(codigo), motivo)
	}
	return motivo
}
