package http

import (
	"errors"
	"fmt"
	"html/template"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"
)

func (h *APIHandler) UsarStatusPublico(s *service.StatusPublicoService) { h.statusPublico = s }

// contaPaginaStatus: dono/admin da conta, ou master/superadmin (com ?conta_id=).
func (h *APIHandler) contaPaginaStatus(c *gin.Context) (string, bool) {
	acesso := obterAcessoDashboard(c)
	switch {
	case acesso.TudoLiberado():
		return strings.TrimSpace(c.Query("conta_id")), true
	case acesso.Tipo == "usuario" && acesso.ContaID != "" && models.PapelGerencia(acesso.Papel):
		return acesso.ContaID, true
	}
	h.responderErro(c, nethttp.StatusForbidden, "acesso_negado", "So o dono ou um admin da conta configura a pagina de status")
	return "", false
}

func (h *APIHandler) ConfigPaginaStatus(c *gin.Context) {
	contaID, ok := h.contaPaginaStatus(c)
	if !ok || h.statusPublico == nil {
		return
	}
	p, err := h.statusPublico.Config(c.Request.Context(), contaID)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Pagina de status", p))
}

func (h *APIHandler) SalvarPaginaStatus(c *gin.Context) {
	contaID, ok := h.contaPaginaStatus(c)
	if !ok || h.statusPublico == nil {
		return
	}
	var req struct {
		Titulo            string   `json:"titulo"`
		Ativa             bool     `json:"ativa"`
		InstanciasOcultas []string `json:"instancias_ocultas"`
		NovoLink          bool     `json:"novo_link"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.responderErro(c, nethttp.StatusBadRequest, "entrada_invalida", "JSON invalido")
		return
	}
	p, err := h.statusPublico.Salvar(c.Request.Context(), contaID, req.Titulo, req.Ativa, req.InstanciasOcultas, req.NovoLink)
	if err != nil {
		h.tratarErro(c, err)
		return
	}
	descricao := "Pagina de status desligada"
	if p.Ativa {
		descricao = "Pagina de status ligada"
	}
	if req.NovoLink {
		descricao += " (link novo gerado)"
	}
	h.descreverAuditoria(c, descricao)
	c.JSON(nethttp.StatusOK, models.NovaRespostaSucesso("Pagina de status salva", p))
}

func (h *APIHandler) statusPublicoDoToken(c *gin.Context) (models.StatusPublico, bool) {
	if h.statusPublico == nil {
		c.Status(nethttp.StatusNotFound)
		return models.StatusPublico{}, false
	}
	pub, err := h.statusPublico.Publico(c.Request.Context(), c.Param("token"))
	if errors.Is(err, service.ErrStatusPublicoNaoEncontrado) {
		c.Data(nethttp.StatusNotFound, "text/html; charset=utf-8", []byte(`<!DOCTYPE html><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Pagina nao encontrada</title><body style="font-family:system-ui;padding:40px;text-align:center;color:#334"><h1>Pagina nao encontrada</h1><p>O link de status nao existe ou foi desligado.</p></body>`))
		return pub, false
	}
	if err != nil {
		c.Status(nethttp.StatusInternalServerError)
		return pub, false
	}
	return pub, true
}

// StatusPublicoJSON: mesma informacao da pagina, para integrar em outro site.
func (h *APIHandler) StatusPublicoJSON(c *gin.Context) {
	pub, ok := h.statusPublicoDoToken(c)
	if !ok {
		if !c.Writer.Written() {
			c.JSON(nethttp.StatusNotFound, models.NovaRespostaErro("nao_encontrado", "Pagina de status nao encontrada"))
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(nethttp.StatusOK, pub)
}

func (h *APIHandler) PaginaStatusPublica(c *gin.Context) {
	pub, ok := h.statusPublicoDoToken(c)
	if !ok {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Robots-Tag", "noindex")
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := modeloStatusPublico.Execute(c.Writer, pub); err != nil {
		_ = c.Error(err)
	}
}

var rotulosEstadoPublico = map[string]string{"operando": "Operando", "reconectando": "Reconectando", "fora": "Fora do ar"}
var rotulosGeralPublico = map[string]string{"operando": "Todos os canais operando", "parcial": "Alguns canais com problema", "fora": "Atendimento fora do ar"}

var modeloStatusPublico = template.Must(template.New("status").Funcs(template.FuncMap{
	"estado": func(e string) string { return rotulosEstadoPublico[e] },
	"geral":  func(g string) string { return rotulosGeralPublico[g] },
	"pct": func(v float64) string {
		if v < 0 {
			return "sem dados"
		}
		return strings.Replace(fmt.Sprintf("%.2f%%", v*100), ".", ",", 1)
	},
	"nivel": func(v float64) string {
		switch {
		case v < 0:
			return "sem"
		case v >= 0.999:
			return "ok"
		case v >= 0.95:
			return "aviso"
		default:
			return "ruim"
		}
	},
	"dia": func(i int, at time.Time) string {
		return at.AddDate(0, 0, i-29).Format("02/01")
	},
	"desde": func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		d := time.Since(t)
		switch {
		case d < time.Minute:
			return "agora"
		case d < time.Hour:
			return fmt.Sprintf("ha %d min", int(d.Minutes()))
		case d < 48*time.Hour:
			return fmt.Sprintf("ha %d h", int(d.Hours()))
		default:
			return fmt.Sprintf("ha %d dias", int(d.Hours()/24))
		}
	},
	"hora": func(t time.Time) string { return t.Format("02/01/2006 15:04") },
}).Parse(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="refresh" content="60">
<meta name="robots" content="noindex">
<title>{{ .Titulo }}</title>
<link rel="icon" href="/static/img/favicon-32.png" sizes="32x32" type="image/png">
<style>
:root { --bg:#f4f7fb; --card:#fff; --ink:#122033; --muted:#64748b; --line:#e2e8f0; --ok:#16a34a; --okbg:#dcfce7; --warn:#d97706; --warnbg:#fef3c7; --bad:#dc2626; --badbg:#fee2e2; --vazio:#e2e8f0; }
@media (prefers-color-scheme: dark) { :root { --bg:#031427; --card:#0b1c30; --ink:#eef5ff; --muted:#8ea0b8; --line:#1f3047; --okbg:rgba(22,163,74,.18); --warnbg:rgba(217,119,6,.18); --badbg:rgba(220,38,38,.18); --vazio:#1f3047; } }
* { box-sizing:border-box; }
body { margin:0; background:var(--bg); color:var(--ink); font:15px/1.5 Inter, system-ui, -apple-system, Segoe UI, sans-serif; }
main { max-width:760px; margin:0 auto; padding:32px 16px 48px; }
h1 { margin:0 0 18px; font-size:24px; letter-spacing:-.02em; }
.geral { display:flex; align-items:center; gap:12px; padding:16px 18px; border-radius:12px; font-weight:700; margin-bottom:22px; }
.geral.operando { background:var(--okbg); color:var(--ok); }
.geral.parcial { background:var(--warnbg); color:var(--warn); }
.geral.fora { background:var(--badbg); color:var(--bad); }
.ponto { width:12px; height:12px; border-radius:50%; background:currentColor; flex:none; }
.lista { background:var(--card); border:1px solid var(--line); border-radius:12px; }
.item { padding:16px 18px; border-top:1px solid var(--line); }
.item:first-child { border-top:0; }
.topo { display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; }
.nome { font-weight:700; }
.estado { font-size:13px; font-weight:700; }
.estado.operando { color:var(--ok); } .estado.reconectando { color:var(--warn); } .estado.fora { color:var(--bad); }
.estado small { color:var(--muted); font-weight:500; margin-left:6px; }
.barras { display:grid; grid-template-columns:repeat(30, 1fr); gap:3px; margin:12px 0 6px; height:30px; }
.barras span { border-radius:3px; background:var(--vazio); }
.barras .ok { background:var(--ok); } .barras .aviso { background:var(--warn); } .barras .ruim { background:var(--bad); }
.legenda { display:flex; justify-content:space-between; color:var(--muted); font-size:12px; }
.nenhum { padding:28px; text-align:center; color:var(--muted); }
footer { margin-top:22px; color:var(--muted); font-size:13px; text-align:center; }
</style>
</head>
<body>
<main>
  <h1>{{ .Titulo }}</h1>
  <div class="geral {{ .Geral }}"><span class="ponto"></span>{{ geral .Geral }}</div>
  <div class="lista">
  {{ $at := .AtualizadoEm }}
  {{ range .Instancias }}
    <section class="item">
      <div class="topo"><span class="nome">{{ .Nome }}</span><span class="estado {{ .Estado }}">{{ estado .Estado }}<small>{{ desde .Desde }}</small></span></div>
      <div class="barras" aria-label="Disponibilidade nos ultimos 30 dias">{{ range $i, $v := .Dias }}<span class="{{ nivel $v }}" title="{{ dia $i $at }}: {{ pct $v }}"></span>{{ end }}</div>
      <div class="legenda"><span>30 dias atras</span><span>{{ pct .Disponibilidade }} no ar</span><span>Hoje</span></div>
    </section>
  {{ else }}
    <div class="nenhum">Nenhum canal publicado.</div>
  {{ end }}
  </div>
  <footer>Atualizado em {{ hora .AtualizadoEm }} · a pagina se atualiza sozinha a cada minuto</footer>
</main>
</body>
</html>`))
