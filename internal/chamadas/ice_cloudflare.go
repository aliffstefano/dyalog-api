package chamadas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// clienteCloudflareTURN busca credenciais no TURN gerenciado da Cloudflare.
//
// Diferente do coturn, a Cloudflare nao usa HMAC local: a credencial vem de uma
// chamada HTTP a API deles. Por isso o resultado fica em cache ate perto do fim
// da validade. Sem cache, cada inicio de chamada viraria uma ida a rede, o que
// atrasaria a ligacao e gastaria quota a toa.
type clienteCloudflareTURN struct {
	keyID    string
	apiToken string
	ttl      time.Duration
	baseURL  string
	http     *http.Client

	mu        sync.Mutex
	cache     []ServidorICE
	renovarEm time.Time
}

const cloudflareTURNBaseURL = "https://rtc.live.cloudflare.com"

func novoClienteCloudflareTURN(keyID, apiToken, baseURL string, ttl time.Duration) *clienteCloudflareTURN {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = cloudflareTURNBaseURL
	}
	return &clienteCloudflareTURN{
		keyID:    strings.TrimSpace(keyID),
		apiToken: strings.TrimSpace(apiToken),
		ttl:      ttl,
		baseURL:  strings.TrimRight(baseURL, "/"),
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Servidores devolve a lista, usando o cache quando ainda vale.
//
// Em caso de erro, devolve o cache antigo se houver: uma credencial vencendo
// ainda tem chance de funcionar, enquanto nao devolver nada garante chamada sem
// audio em rede restritiva.
func (c *clienteCloudflareTURN) Servidores(ctx context.Context, agora time.Time) ([]ServidorICE, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.cache) > 0 && agora.Before(c.renovarEm) {
		return c.cache, nil
	}
	servidores, err := c.buscar(ctx)
	if err != nil {
		if len(c.cache) > 0 {
			return c.cache, nil
		}
		return nil, err
	}
	c.cache = servidores
	// Renova aos 80% da validade, para nunca entregar credencial na iminencia
	// de expirar.
	c.renovarEm = agora.Add(c.ttl * 4 / 5)
	return servidores, nil
}

func (c *clienteCloudflareTURN) buscar(ctx context.Context) ([]ServidorICE, error) {
	corpo, err := json.Marshal(map[string]int{"ttl": int(c.ttl.Seconds())})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/v1/turn/keys/%s/credentials/generate-ice-servers", c.baseURL, c.keyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar TURN da Cloudflare: %w", err)
	}
	defer resp.Body.Close()
	dados, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta do TURN da Cloudflare: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("TURN da Cloudflare retornou status %d: %s", resp.StatusCode, strings.TrimSpace(string(dados)))
	}

	var payload respostaCloudflareTURN
	if err := json.Unmarshal(dados, &payload); err != nil {
		return nil, fmt.Errorf("resposta do TURN da Cloudflare em formato inesperado: %w", err)
	}
	if len(payload.IceServers) == 0 {
		return nil, fmt.Errorf("TURN da Cloudflare nao devolveu nenhum servidor")
	}
	return payload.IceServers, nil
}

type respostaCloudflareTURN struct {
	IceServers []ServidorICE
}

// UnmarshalJSON aceita iceServers como lista ou como objeto unico. A API ja
// documentou os dois formatos, e errar aqui significaria chamada sem audio.
func (r *respostaCloudflareTURN) UnmarshalJSON(dados []byte) error {
	var comLista struct {
		IceServers []ServidorICE `json:"iceServers"`
	}
	if err := json.Unmarshal(dados, &comLista); err == nil && len(comLista.IceServers) > 0 {
		r.IceServers = comLista.IceServers
		return nil
	}
	var comObjeto struct {
		IceServers ServidorICE `json:"iceServers"`
	}
	if err := json.Unmarshal(dados, &comObjeto); err != nil {
		return err
	}
	if len(comObjeto.IceServers.URLs) == 0 {
		return fmt.Errorf("iceServers vazio")
	}
	r.IceServers = []ServidorICE{comObjeto.IceServers}
	return nil
}
