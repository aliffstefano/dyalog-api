package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"strings"
	"time"

	"dyalog-api-go/internal/whatsapp"

	"github.com/gin-gonic/gin"
)

// cabecalhoEncaminhado marca requisicoes que ja foram encaminhadas por outra
// replica. Serve como trava de seguranca: se o destino tambem nao for o dono da
// instancia (por exemplo, porque o lock mudou no meio do caminho), ele responde o
// 409 normal em vez de encaminhar de novo e criar um laco entre containers.
const cabecalhoEncaminhado = "X-Dyalog-Encaminhado"

// limiteCorpoInspecao limita quanto do corpo lemos para descobrir a instancia
// alvo. Payloads de envio sao pequenos; midia em base64 pode ser grande, entao
// evitamos carregar tudo em memoria so para ler um campo.
const limiteCorpoInspecao = 1 << 20 // 1 MiB

// middlewareProxyReplica encaminha a requisicao para a replica que detem o lock da
// instancia alvo. Cada sessao do WhatsApp vive em um unico container, mas o load
// balancer distribui as requisicoes entre todas as replicas; sem esse desvio, tudo
// que caisse no container errado falharia com 409.
func middlewareProxyReplica(gerenciador *whatsapp.GerenciadorInstancias) gin.HandlerFunc {
	transporte := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 2 * time.Minute,
	}
	return func(c *gin.Context) {
		if gerenciador == nil || c.GetHeader(cabecalhoEncaminhado) != "" {
			c.Next()
			return
		}
		instanciaID := resolverInstanciaAlvo(c)
		if instanciaID == "" {
			c.Next()
			return
		}
		endereco := gerenciador.EnderecoOutroContainer(c.Request.Context(), instanciaID)
		if endereco == "" {
			c.Next()
			return
		}
		destino, err := neturl.Parse(endereco)
		if err != nil || destino.Host == "" {
			c.Next()
			return
		}
		encaminharParaReplica(c, transporte, destino)
	}
}

// resolverInstanciaAlvo descobre qual instancia a requisicao quer atingir, olhando
// (nesta ordem) o parametro :id da rota, o corpo JSON e o token de instancia.
func resolverInstanciaAlvo(c *gin.Context) string {
	if id := strings.TrimSpace(c.Param("id")); id != "" {
		return id
	}
	if id := instanciaDoCorpo(c); id != "" {
		return id
	}
	acesso := obterAcessoDashboard(c)
	if acesso.Tipo == "instancia" {
		return strings.TrimSpace(acesso.InstanciaID)
	}
	return ""
}

// instanciaDoCorpo le o campo "instancia" do corpo JSON e devolve o corpo intacto
// para o handler seguinte (ou para o encaminhamento).
func instanciaDoCorpo(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	if !strings.Contains(strings.ToLower(c.GetHeader("Content-Type")), "json") {
		return ""
	}
	corpo, err := io.ReadAll(io.LimitReader(c.Request.Body, limiteCorpoInspecao))
	if err != nil {
		return ""
	}
	// Reposiciona o corpo lido para que o handler local (ou o proxy) o receba inteiro.
	c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(corpo), c.Request.Body))
	var payload struct {
		Instancia string `json:"instancia"`
	}
	if err := json.Unmarshal(corpo, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Instancia)
}

func encaminharParaReplica(c *gin.Context, transporte http.RoundTripper, destino *neturl.URL) {
	proxy := &httputil.ReverseProxy{
		Transport: transporte,
		Director: func(req *http.Request) {
			req.URL.Scheme = destino.Scheme
			req.URL.Host = destino.Host
			req.Host = destino.Host
			req.Header.Set(cabecalhoEncaminhado, "1")
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sucesso": false,
				"erro":    "replica_indisponivel",
				"mensagem": "nao foi possivel encaminhar a requisicao para o container que detem a instancia: " +
					err.Error(),
			})
		},
	}
	proxy.ServeHTTP(c.Writer, c.Request)
	c.Abort()
}
