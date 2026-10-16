// Package email envia os e-mails do painel (convites e redefinicao de senha).
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNaoConfigurado indica que nao ha provedor de e-mail: quem chamou mostra
// o link para ser copiado e enviado manualmente.
var ErrNaoConfigurado = errors.New("envio de e-mail nao configurado (RESEND_API_KEY)")

// Enviador manda um e-mail em HTML.
type Enviador interface {
	Enviar(ctx context.Context, para, assunto, html string) error
	Configurado() bool
}

// Resend envia pela API da Resend (https://resend.com).
type Resend struct {
	chave     string
	remetente string
	base      string
	http      *http.Client
}

// NovoResend devolve o enviador; sem chave, Enviar responde ErrNaoConfigurado.
// base vazia usa https://api.resend.com (outra so em testes).
func NovoResend(chave, remetente, base string) *Resend {
	if base == "" {
		base = "https://api.resend.com"
	}
	return &Resend{chave: strings.TrimSpace(chave), remetente: strings.TrimSpace(remetente), base: strings.TrimRight(base, "/"), http: &http.Client{Timeout: 15 * time.Second}}
}

func (r *Resend) Configurado() bool { return r != nil && r.chave != "" && r.remetente != "" }

func (r *Resend) Enviar(ctx context.Context, para, assunto, html string) error {
	if !r.Configurado() {
		return ErrNaoConfigurado
	}
	corpo, _ := json.Marshal(map[string]any{"from": r.remetente, "to": []string{para}, "subject": assunto, "html": html})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.base+"/emails", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.chave)
	req.Header.Set("Content-Type", "application/json")
	resp, err := r.http.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao chamar a Resend: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		detalhe, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("a Resend recusou o e-mail (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(detalhe)))
	}
	return nil
}

// Modelo monta o HTML padrao: titulo, paragrafos, botao com o link e o link
// escrito (para quem nao consegue clicar).
func Modelo(titulo string, paragrafos []string, rotuloBotao, link string) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,Helvetica,sans-serif;background:#f3f6f9;padding:32px 16px"><div style="max-width:520px;margin:0 auto;background:#ffffff;border-radius:12px;padding:32px;color:#112036">`)
	b.WriteString(`<p style="margin:0 0 4px;font-size:13px;font-weight:700;color:#1995bd;letter-spacing:.08em">DYALOG CONNECT</p>`)
	b.WriteString(`<h1 style="margin:0 0 16px;font-size:22px">` + escapar(titulo) + `</h1>`)
	for _, p := range paragrafos {
		b.WriteString(`<p style="margin:0 0 14px;font-size:15px;line-height:1.5;color:#334155">` + escapar(p) + `</p>`)
	}
	b.WriteString(`<p style="margin:24px 0"><a href="` + escapar(link) + `" style="display:inline-block;background:#1995bd;color:#ffffff;text-decoration:none;font-weight:700;padding:12px 22px;border-radius:8px">` + escapar(rotuloBotao) + `</a></p>`)
	b.WriteString(`<p style="margin:0;font-size:12px;color:#64748b">Se o botao nao funcionar, copie e cole no navegador:<br><span style="word-break:break-all">` + escapar(link) + `</span></p>`)
	b.WriteString(`</div></div>`)
	return b.String()
}

func escapar(texto string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(texto)
}
