package http

import (
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/service"
	"dyalog-api-go/internal/store"
)

func TestMiddlewareAuditoriaGravaSoSucessoDasRotasAuditadas(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st, err := store.NovoSQLStore("sqlite", filepath.Join(t.TempDir(), "aud.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	agora := time.Now().UTC()
	if _, err := st.Criar(ctx, models.Instancia{ID: "i1", Nome: "Loja", Token: "t", Status: "conectada", CriadoEm: agora, AtualizadoEm: agora}); err != nil {
		t.Fatal(err)
	}
	h := &APIHandler{}
	h.UsarAuditoria(service.NovoAuditoriaService(st, st, st, nil, "", 90))

	engine := gin.New()
	grupo := engine.Group("/api/v1", func(c *gin.Context) {
		c.Set(contextoAcessoDashboard, models.AcessoDashboard{Tipo: "usuario", Nome: "Ana", UsuarioID: "u1"})
	}, h.middlewareAuditoria())
	grupo.PUT("/instancias/:id/avancado", func(c *gin.Context) {
		if c.Query("falhar") != "" {
			c.Status(nethttp.StatusBadRequest)
			return
		}
		c.Status(nethttp.StatusOK)
	})
	grupo.GET("/instancias/:id/status", func(c *gin.Context) { c.Status(nethttp.StatusOK) })

	for _, url := range []string{"/api/v1/instancias/i1/avancado", "/api/v1/instancias/i1/avancado?falhar=1"} {
		engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(nethttp.MethodPut, url, nil))
	}
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(nethttp.MethodGet, "/api/v1/instancias/i1/status", nil))

	pagina, err := st.ListarAuditoria(ctx, models.FiltroAuditoria{InstanciaID: "i1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pagina.Registros) != 1 {
		t.Fatalf("esperado 1 registro, veio %+v", pagina.Registros)
	}
	r := pagina.Registros[0]
	if r.Acao != "instancia.comportamento" || r.AtorNome != "Ana" || r.AtorTipo != "usuario" || r.InstanciaNome != "Loja" {
		t.Fatalf("registro = %+v", r)
	}
}
