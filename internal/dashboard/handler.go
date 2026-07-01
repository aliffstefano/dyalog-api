package dashboard

import (
	"html/template"
	nethttp "net/http"

	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	cfg         *config.Config
	authService *service.AuthService
	tmpl        *template.Template
	loginTmpl   *template.Template
}

type paginaInicialData struct {
	Titulo string
}

func NovoHandler(cfg *config.Config, authService *service.AuthService) *Handler {
	tmpl := template.Must(template.New("dashboard").Parse(paginaInicialHTML))
	loginTmpl := template.Must(template.New("login").Parse(paginaLoginHTML))
	return &Handler{cfg: cfg, authService: authService, tmpl: tmpl, loginTmpl: loginTmpl}
}

func (h *Handler) PaginaInicial(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.DashboardCookieNome)
	if _, err := h.authService.Autenticar(c.Request.Context(), token); err != nil {
		c.Status(nethttp.StatusUnauthorized)
		_ = h.loginTmpl.Execute(c.Writer, paginaInicialData{Titulo: h.cfg.NomeAplicacao})
		return
	}
	c.Status(nethttp.StatusOK)
	_ = h.tmpl.Execute(c.Writer, paginaInicialData{Titulo: h.cfg.NomeAplicacao})
}
