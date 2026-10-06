package http

import (
	"fmt"
	"math"
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
	return &limitador{porMinuto: porMinuto, itens: map[string]*itemLimite{}}
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
