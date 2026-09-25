package chamadas

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pion/webrtc/v4"
)

// ServidorICE descreve um servidor STUN ou TURN no formato que o navegador
// espera em RTCPeerConnection({iceServers: [...]}).
type ServidorICE struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// ConfigICE guarda como montar a lista de servidores ICE entregue ao cliente e
// usada pela ponte de audio.
//
// O TURN usa credencial temporaria no padrao use-auth-secret do coturn: o
// usuario e "<expiracao>:<nome>" e a senha e o HMAC-SHA1 desse usuario com um
// segredo compartilhado. O segredo fica so no servidor; o navegador recebe um
// par que expira sozinho, entao nao ha senha fixa embutida no front.
type ConfigICE struct {
	servidoresFixos []ServidorICE
	turnURLs        []string
	turnSegredo     string
	turnTTL         time.Duration
	cloudflare      *clienteCloudflareTURN
}

// NovaConfigICE monta a configuracao a partir das variaveis de ambiente.
//
// servidoresJSON aceita a lista completa em JSON, no mesmo formato do navegador.
// turnURLs, turnSegredo e turnTTL configuram o TURN com credencial temporaria.
// Os dois caminhos convivem: o JSON costuma trazer o STUN, e o TURN entra por
// cima ja com usuario e senha gerados.
func NovaConfigICE(servidoresJSON, turnURLs, turnSegredo, cloudflareKeyID, cloudflareToken string, turnTTLSegundos int) (ConfigICE, error) {
	cfg := ConfigICE{turnSegredo: strings.TrimSpace(turnSegredo)}

	if texto := strings.TrimSpace(servidoresJSON); texto != "" {
		if err := json.Unmarshal([]byte(texto), &cfg.servidoresFixos); err != nil {
			return ConfigICE{}, fmt.Errorf("WEBRTC_ICE_SERVERS invalido: %w", err)
		}
		for i, servidor := range cfg.servidoresFixos {
			if len(servidor.URLs) == 0 {
				return ConfigICE{}, fmt.Errorf("WEBRTC_ICE_SERVERS invalido: servidor %d sem urls", i+1)
			}
		}
	}

	for _, url := range strings.Split(turnURLs, ",") {
		if url = strings.TrimSpace(url); url != "" {
			cfg.turnURLs = append(cfg.turnURLs, url)
		}
	}
	if len(cfg.turnURLs) > 0 && cfg.turnSegredo == "" {
		return ConfigICE{}, fmt.Errorf("TURN_URLS configurado sem TURN_SECRET: a credencial temporaria precisa do segredo")
	}

	cloudflareKeyID = strings.TrimSpace(cloudflareKeyID)
	cloudflareToken = strings.TrimSpace(cloudflareToken)
	if (cloudflareKeyID == "") != (cloudflareToken == "") {
		return ConfigICE{}, fmt.Errorf("CLOUDFLARE_TURN_KEY_ID e CLOUDFLARE_TURN_API_TOKEN precisam ser informados juntos")
	}

	cfg.turnTTL = time.Duration(turnTTLSegundos) * time.Second
	if cfg.turnTTL <= 0 {
		cfg.turnTTL = 12 * time.Hour
	}
	if cloudflareKeyID != "" {
		cfg.cloudflare = novoClienteCloudflareTURN(cloudflareKeyID, cloudflareToken, os.Getenv("CLOUDFLARE_TURN_BASE_URL"), cfg.turnTTL)
	}
	return cfg, nil
}

// Configurado informa se ha algum servidor ICE definido. Sem nenhum, a conexao
// depende dos candidatos que o proprio navegador trouxer.
func (c ConfigICE) Configurado() bool {
	return len(c.servidoresFixos) > 0 || len(c.turnURLs) > 0 || c.cloudflare != nil
}

// ServidoresParaCliente devolve a lista pronta para o navegador, com a
// credencial TURN ja gerada e com validade a partir de agora.
func (c ConfigICE) ServidoresParaCliente(ctx context.Context, agora time.Time) []ServidorICE {
	servidores := make([]ServidorICE, 0, len(c.servidoresFixos)+1)
	servidores = append(servidores, c.servidoresFixos...)
	if c.cloudflare != nil {
		// A Cloudflare ja devolve STUN e TURN juntos, no formato do navegador.
		if daCloudflare, err := c.cloudflare.Servidores(ctx, agora); err == nil {
			servidores = append(servidores, daCloudflare...)
		} else {
			fmt.Printf("erro ao obter TURN da Cloudflare: %v\n", err)
		}
	}
	if len(c.turnURLs) == 0 {
		return servidores
	}
	usuario, senha := c.credencialTemporaria(agora)
	return append(servidores, ServidorICE{
		URLs:       append([]string(nil), c.turnURLs...),
		Username:   usuario,
		Credential: senha,
	})
}

// ParaPion converte a lista para o formato do pion, usado pela ponte de audio.
func (c ConfigICE) ParaPion(ctx context.Context, agora time.Time) []webrtc.ICEServer {
	clientes := c.ServidoresParaCliente(ctx, agora)
	servidores := make([]webrtc.ICEServer, 0, len(clientes))
	for _, servidor := range clientes {
		item := webrtc.ICEServer{URLs: servidor.URLs}
		if servidor.Username != "" {
			item.Username = servidor.Username
			item.Credential = servidor.Credential
			item.CredentialType = webrtc.ICECredentialTypePassword
		}
		servidores = append(servidores, item)
	}
	return servidores
}

// credencialTemporaria implementa a TURN REST API: usuario "<expiracao>:dyalog"
// e senha HMAC-SHA1 desse usuario com o segredo. O coturn valida sozinho, sem
// precisar cadastrar usuario nenhum.
func (c ConfigICE) credencialTemporaria(agora time.Time) (string, string) {
	expiracao := agora.Add(c.turnTTL).Unix()
	usuario := strconv.FormatInt(expiracao, 10) + ":dyalog"
	mac := hmac.New(sha1.New, []byte(c.turnSegredo))
	mac.Write([]byte(usuario))
	return usuario, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ValidadeSegundos informa por quanto tempo a credencial devolvida vale, para o
// cliente saber quando pedir outra.
func (c ConfigICE) ValidadeSegundos() int {
	if len(c.turnURLs) == 0 && c.cloudflare == nil {
		return 0
	}
	return int(c.turnTTL.Seconds())
}
