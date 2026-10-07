// Package meta fala com a API oficial do WhatsApp (WhatsApp Cloud API, Graph
// API da Meta): envio de mensagens, upload e download de midia e leitura dos
// webhooks que a Meta manda.
package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
)

// VersaoGraphPadrao e a versao da Graph API usada quando META_GRAPH_VERSION nao
// e informada.
const VersaoGraphPadrao = "v25.0"

// Cliente faz as chamadas HTTP para a Graph API.
type Cliente struct {
	http   *http.Client
	base   string
	versao string
}

// NovoCliente cria o cliente. base vazio usa https://graph.facebook.com; outro
// valor serve para testes com um servidor falso.
func NovoCliente(versao, base string) *Cliente {
	versao = strings.TrimSpace(versao)
	if versao == "" {
		versao = VersaoGraphPadrao
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "https://graph.facebook.com"
	}
	return &Cliente{
		http:   &http.Client{Timeout: 60 * time.Second},
		base:   base,
		versao: versao,
	}
}

// Erro e a resposta de erro da Graph API.
type Erro struct {
	Status    int
	Codigo    int    `json:"code"`
	Subcodigo int    `json:"error_subcode"`
	Tipo      string `json:"type"`
	Mensagem  string `json:"message"`
	Dados     struct {
		Detalhes string `json:"details"`
	} `json:"error_data"`
}

func (e *Erro) Error() string {
	texto := e.Mensagem
	if detalhe := strings.TrimSpace(e.Dados.Detalhes); detalhe != "" {
		texto += ": " + detalhe
	}
	if dica := dicaErro(e.Codigo); dica != "" {
		texto += " (" + dica + ")"
	}
	return fmt.Sprintf("Meta respondeu erro %d: %s", e.Codigo, texto)
}

// ErroJanela24h indica mensagem livre fora da janela de 24h: so template e
// aceito.
func ErroJanela24h(err error) bool {
	var e *Erro
	return errors.As(err, &e) && e.Codigo == 131047
}

// ErroCredenciais indica token invalido, expirado ou sem permissao.
func ErroCredenciais(err error) bool {
	var e *Erro
	return errors.As(err, &e) && (e.Codigo == 190 || e.Codigo == 10 || e.Codigo == 200)
}

func dicaErro(codigo int) string {
	switch codigo {
	case 131047:
		return "passaram mais de 24h desde a ultima mensagem do cliente; envie um template"
	case 190:
		return "token de acesso invalido ou expirado"
	case 10, 200:
		return "o token nao tem permissao whatsapp_business_messaging para este numero"
	case 131030:
		return "numero fora da lista de teste do app; adicione o destinatario no painel da Meta (numero do Brasil: a Meta pode guardar sem o 9, use o wa_id que chega no webhook)"
	case 132001:
		return "template nao existe ou nao foi aprovado neste idioma"
	case 131026:
		return "o numero nao tem WhatsApp ou nao pode receber a mensagem"
	default:
		return ""
	}
}

func (c *Cliente) url(caminho string) string {
	return c.base + "/" + c.versao + "/" + strings.TrimPrefix(caminho, "/")
}

// requisitar faz a chamada e decodifica a resposta em destino. Respostas com
// erro viram *Erro.
func (c *Cliente) requisitar(ctx context.Context, metodo, url, token string, corpo io.Reader, contentType string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, metodo, url, corpo)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao chamar a API da Meta: %w", err)
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("erro ao ler resposta da Meta: %w", err)
	}
	if resp.StatusCode >= 300 {
		var envelope struct {
			Erro *Erro `json:"error"`
		}
		if json.Unmarshal(dados, &envelope) == nil && envelope.Erro != nil {
			envelope.Erro.Status = resp.StatusCode
			return envelope.Erro
		}
		return fmt.Errorf("Meta respondeu HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(dados)))
	}
	if destino == nil {
		return nil
	}
	if err := json.Unmarshal(dados, destino); err != nil {
		return fmt.Errorf("resposta inesperada da Meta: %w", err)
	}
	return nil
}

func (c *Cliente) postarJSON(ctx context.Context, caminho, token string, corpo any, destino any) error {
	dados, err := json.Marshal(corpo)
	if err != nil {
		return err
	}
	return c.requisitar(ctx, http.MethodPost, c.url(caminho), token, bytes.NewReader(dados), "application/json", destino)
}

// InfoNumero e o que a Meta devolve sobre o numero cadastrado.
type InfoNumero struct {
	ID              string `json:"id"`
	NumeroExibicao  string `json:"display_phone_number"`
	NomeVerificado  string `json:"verified_name"`
	Qualidade       string `json:"quality_rating"`
	StatusNome      string `json:"name_status"`
	ModoPlataforma  string `json:"platform_type"`
	LimiteMensagens string `json:"messaging_limit_tier"`
}

// ValidarNumero confere o token e o phone_number_id buscando os dados do
// numero na Meta.
func (c *Cliente) ValidarNumero(ctx context.Context, cred models.CredenciaisMeta) (InfoNumero, error) {
	var info InfoNumero
	url := c.url(cred.PhoneNumberID) + "?fields=id,display_phone_number,verified_name,quality_rating,name_status,platform_type,messaging_limit_tier"
	err := c.requisitar(ctx, http.MethodGet, url, cred.AccessToken, nil, "", &info)
	return info, err
}

// RespostaEnvio e o retorno de POST /{phone-number-id}/messages.
type RespostaEnvio struct {
	Contatos []struct {
		Entrada string `json:"input"`
		WaID    string `json:"wa_id"`
	} `json:"contacts"`
	Mensagens []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// MensagemID devolve o id (wamid) da mensagem enviada.
func (r RespostaEnvio) MensagemID() string {
	if len(r.Mensagens) == 0 {
		return ""
	}
	return r.Mensagens[0].ID
}

// WaID devolve o numero de WhatsApp que a Meta resolveu para o destino.
func (r RespostaEnvio) WaID() string {
	if len(r.Contatos) == 0 {
		return ""
	}
	return r.Contatos[0].WaID
}

// Enviar manda uma mensagem ja montada (ver mensagens.go).
func (c *Cliente) Enviar(ctx context.Context, cred models.CredenciaisMeta, mensagem map[string]any) (RespostaEnvio, error) {
	var resp RespostaEnvio
	err := c.postarJSON(ctx, cred.PhoneNumberID+"/messages", cred.AccessToken, mensagem, &resp)
	return resp, err
}

// MarcarLida marca a mensagem recebida como lida (os dois tracinhos azuis).
func (c *Cliente) MarcarLida(ctx context.Context, cred models.CredenciaisMeta, mensagemID string) error {
	return c.postarJSON(ctx, cred.PhoneNumberID+"/messages", cred.AccessToken, map[string]any{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        mensagemID,
	}, nil)
}

// SubirMidia envia o arquivo para a Meta e devolve o id da midia, usado no
// envio. Necessario quando a midia veio em base64 ou caminho local; com URL
// publica a Meta baixa sozinha.
func (c *Cliente) SubirMidia(ctx context.Context, cred models.CredenciaisMeta, dados []byte, nomeArquivo, mimeType string) (string, error) {
	var corpo bytes.Buffer
	escritor := multipart.NewWriter(&corpo)
	_ = escritor.WriteField("messaging_product", "whatsapp")
	_ = escritor.WriteField("type", mimeType)
	cabecalho := make(textproto.MIMEHeader)
	cabecalho.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, nomeArquivo))
	cabecalho.Set("Content-Type", mimeType)
	parte, err := escritor.CreatePart(cabecalho)
	if err != nil {
		return "", err
	}
	if _, err := parte.Write(dados); err != nil {
		return "", err
	}
	if err := escritor.Close(); err != nil {
		return "", err
	}
	var resp struct {
		ID string `json:"id"`
	}
	err = c.requisitar(ctx, http.MethodPost, c.url(cred.PhoneNumberID+"/media"), cred.AccessToken, &corpo, escritor.FormDataContentType(), &resp)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// BaixarMidia busca uma midia recebida pelo id que veio no webhook.
func (c *Cliente) BaixarMidia(ctx context.Context, cred models.CredenciaisMeta, midiaID string) ([]byte, string, error) {
	var info struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
	}
	if err := c.requisitar(ctx, http.MethodGet, c.url(midiaID), cred.AccessToken, nil, "", &info); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("erro ao baixar midia da Meta: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("Meta respondeu HTTP %d ao baixar a midia", resp.StatusCode)
	}
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 100<<20))
	if err != nil {
		return nil, "", fmt.Errorf("erro ao baixar midia da Meta: %w", err)
	}
	return dados, info.MimeType, nil
}
