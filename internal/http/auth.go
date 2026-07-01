package http

import (
	"net/http"
	"strings"

	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"

	"github.com/gin-gonic/gin"
)

const contextoAcessoDashboard = "acesso_dashboard"

func middlewareAutenticacaoAPI(cfg *config.Config, authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extrairTokenDashboard(c, cfg.DashboardCookieNome)
		acesso, err := authService.Autenticar(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NovaRespostaErro("nao_autenticado", "Informe um token valido para acessar a API"))
			c.Abort()
			return
		}
		c.Set(contextoAcessoDashboard, acesso)
		c.Next()
	}
}

func obterAcessoDashboard(c *gin.Context) models.AcessoDashboard {
	valor, ok := c.Get(contextoAcessoDashboard)
	if !ok {
		return models.AcessoDashboard{}
	}
	acesso, _ := valor.(models.AcessoDashboard)
	return acesso
}

func extrairTokenDashboard(c *gin.Context, cookieNome string) string {
	if bearer := strings.TrimSpace(c.GetHeader("Authorization")); strings.HasPrefix(strings.ToLower(bearer), "bearer ") {
		return strings.TrimSpace(bearer[7:])
	}
	if token := strings.TrimSpace(c.GetHeader("X-Access-Token")); token != "" {
		return token
	}
	if cookieNome != "" {
		if token, err := c.Cookie(cookieNome); err == nil {
			return strings.TrimSpace(token)
		}
	}
	return ""
}
