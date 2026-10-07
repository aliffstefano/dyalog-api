package http

import (
	"crypto/sha256"
	"fmt"
	"math"
	"net"
	nethttp "net/http"
	"strconv"
	"sync"
	"time"

	"dyalog-api-go/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// limitador conta eventos por chave (instancia, IP) com balde de fichas: ate
// porMinuto de uma vez, repondo porMinuto a cada minuto. Fica em memoria em
// cada replica; os envios de uma instancia ja caem sempre na replica dona dela
// (proxyReplica), entao a contagem por instancia e unica.
type limitador struct {
	porMinuto int
	mu        sync.Mutex
	itens     map[string]*itemLimite
	limpeza   time.Time
	// vistos guarda os tokens invalidos ja contados por IP no ultimo minuto
	// (so um resumo do token, nunca o token).
	vistos map[string]time.Time
}

type itemLimite struct {
	fichas *rate.Limiter
	uso    time.Time
}

// novoLimitador devolve nil (sem limite) quando porMinuto <= 0.
func novoLimitador(porMinuto int) *limitador {
	if porMinuto <= 0 {
		return nil
	}
	return &limitador{porMinuto: porMinuto, itens: map[string]*itemLimite{}, vistos: map[string]time.Time{}}
}

func (l *limitador) obter(chave string, agora time.Time) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Chave parada ha mais de 10 minutos ja esta com o balde cheio; descartar
	// evita que IPs de passagem acumulem na memoria.
	if agora.Sub(l.limpeza) > time.Minute {
		for k, item := range l.itens {
			if agora.Sub(item.uso) > 10*time.Minute {
				delete(l.itens, k)
			}
		}
		for k, visto := range l.vistos {
			if agora.Sub(visto) > time.Minute {
				delete(l.vistos, k)
			}
		}
		l.limpeza = agora
	}
	item, ok := l.itens[chave]
	if !ok {
		item = &itemLimite{fichas: rate.NewLimiter(rate.Limit(float64(l.porMinuto)/60), l.porMinuto)}
		l.itens[chave] = item
	}
	item.uso = agora
	return item.fichas
}

// consumir gasta uma ficha. Sem ficha, nao gasta e devolve quanto esperar.
func (l *limitador) consumir(chave string) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}
	agora := time.Now()
	reserva := l.obter(chave, agora).ReserveN(agora, 1)
	if espera := reserva.DelayFrom(agora); espera > 0 {
		reserva.CancelAt(agora)
		return false, espera
	}
	return true, 0
}

// registrarFalha conta um token invalido vindo de ip. So tokens diferentes
// contam: quem tenta adivinhar troca de token a cada tentativa, enquanto um
// cliente esquecido com token antigo (painel aberto, fluxo do n8n) repete o
// mesmo e nao deve bloquear o IP inteiro. Requisicao sem token nao conta.
func (l *limitador) registrarFalha(ip, token string) {
	if l == nil || token == "" {
		return
	}
	resumo := sha256.Sum256([]byte(token))
	chave := ip + "|" + string(resumo[:8])
	agora := time.Now()
	l.mu.Lock()
	visto, ok := l.vistos[chave]
	l.vistos[chave] = agora
	l.mu.Unlock()
	if ok && agora.Sub(visto) <= time.Minute {
		return
	}
	l.consumir(ip)
}

// bloqueado diz, sem gastar ficha, se a chave esgotou o limite.
func (l *limitador) bloqueado(chave string) (bool, time.Duration) {
	if l == nil {
		return false, 0
	}
	agora := time.Now()
	fichas := l.obter(chave, agora).TokensAt(agora)
	if fichas >= 1 {
		return false, 0
	}
	return true, time.Duration((1 - fichas) * 60 / float64(l.porMinuto) * float64(time.Second))
}

// responderLimite responde 429 com Retry-After em segundos.
func responderLimite(c *gin.Context, codigo, mensagem string, espera time.Duration) {
	segundos := int(math.Ceil(espera.Seconds()))
	if segundos < 1 {
		segundos = 1
	}
	c.Header("Retry-After", strconv.Itoa(segundos))
	c.AbortWithStatusJSON(nethttp.StatusTooManyRequests, models.NovaRespostaErro(codigo, fmt.Sprintf("%s; tente novamente em %d segundos", mensagem, segundos)))
}

// redesCloudflare sao as faixas publicadas em https://www.cloudflare.com/ips.
var redesCloudflare = func() []*net.IPNet {
	faixas := []string{
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
		"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
		"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
		"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
	}
	redes := make([]*net.IPNet, 0, len(faixas))
	for _, faixa := range faixas {
		_, rede, err := net.ParseCIDR(faixa)
		if err != nil {
			panic(err)
		}
		redes = append(redes, rede)
	}
	return redes
}()

// ipCliente e o IP usado nos limites. Atras da Cloudflare todo acesso chega
// com IP da Cloudflare; nesse caso vale o CF-Connecting-IP, que ela preenche
// com o IP do visitante. O cabecalho so e aceito quando a requisicao veio
// mesmo de um IP da Cloudflare: quem acessar o servidor direto nao consegue
// forjar o IP.
func ipCliente(c *gin.Context) string {
	ip := c.ClientIP()
	visitante := c.GetHeader("CF-Connecting-IP")
	if visitante == "" || net.ParseIP(visitante) == nil {
		return ip
	}
	if origem := net.ParseIP(ip); origem != nil {
		for _, rede := range redesCloudflare {
			if rede.Contains(origem) {
				return visitante
			}
		}
	}
	return ip
}
