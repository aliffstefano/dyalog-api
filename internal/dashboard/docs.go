package dashboard

import "github.com/gin-gonic/gin"

const paginaDocsHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Dyalog API GO - Documentacao</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; font-family: "Segoe UI", sans-serif; }
    .swagger-ui .topbar { background: #0d6b5f; }
    .swagger-ui .topbar .link { display: none; }
    .swagger-ui .topbar-wrapper img { display: none; }
    .swagger-ui .topbar-wrapper::before { content: "Dyalog API GO"; color: white; font-size: 20px; font-weight: 700; }
    .back-link { display: inline-flex; align-items: center; gap: 8px; padding: 10px 18px; margin: 16px; background: #0d6b5f; color: white; text-decoration: none; border-radius: 12px; font-size: 14px; font-weight: 600; }
    .back-link:hover { background: #0f7f71; }
  </style>
</head>
<body>
  <a class="back-link" href="/">&#8592; Voltar ao painel</a>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/static/swagger.json",
      dom_id: '#swagger-ui',
      deepLinking: true,
      tryItOutEnabled: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout"
    });
  </script>
</body>
</html>`

func (h *Handler) Docs(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	_, _ = c.Writer.WriteString(paginaDocsHTML)
}
