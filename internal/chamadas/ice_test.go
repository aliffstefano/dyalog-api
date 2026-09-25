package chamadas

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestConfigICEVaziaNaoConfigurada(t *testing.T) {
	cfg, err := NovaConfigICE("", "", "", 0)
	if err != nil {
		t.Fatalf("configuracao vazia nao deveria falhar: %v", err)
	}
	if cfg.Configurado() {
		t.Fatal("sem STUN e sem TURN a config nao pode se dizer configurada")
	}
	if len(cfg.ServidoresParaCliente(time.Now())) != 0 {
		t.Fatal("sem configuracao nao ha servidor para devolver")
	}
}

func TestConfigICEComStunJSON(t *testing.T) {
	cfg, err := NovaConfigICE(`[{"urls":["stun:stun.l.google.com:19302"]}]`, "", "", 0)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	servidores := cfg.ServidoresParaCliente(time.Now())
	if len(servidores) != 1 || servidores[0].URLs[0] != "stun:stun.l.google.com:19302" {
		t.Fatalf("servidores = %+v", servidores)
	}
	// STUN nao tem credencial: mandar usuario vazio confundiria o navegador.
	if servidores[0].Username != "" || servidores[0].Credential != "" {
		t.Fatal("STUN nao deve vir com credencial")
	}
}

func TestConfigICEJSONInvalidoFalhaNaSubida(t *testing.T) {
	if _, err := NovaConfigICE("nao-e-json", "", "", 0); err == nil {
		t.Fatal("JSON invalido deveria falhar na subida, nao na primeira chamada")
	}
	if _, err := NovaConfigICE(`[{"username":"x"}]`, "", "", 0); err == nil {
		t.Fatal("servidor sem urls deveria falhar")
	}
}

func TestTurnExigeSegredo(t *testing.T) {
	if _, err := NovaConfigICE("", "turn:turn.exemplo.com:3478", "", 0); err == nil {
		t.Fatal("TURN sem segredo deveria falhar: a credencial temporaria depende dele")
	}
}

// A credencial precisa bater exatamente com o que o coturn calcula no modo
// use-auth-secret, senao o TURN recusa e o audio nao passa em rede restritiva.
func TestCredencialTemporariaSegueTurnRestAPI(t *testing.T) {
	const segredo = "segredo-compartilhado"
	cfg, err := NovaConfigICE("", "turn:turn.exemplo.com:3478,turns:turn.exemplo.com:5349", segredo, 3600)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	agora := time.Unix(1700000000, 0).UTC()
	servidores := cfg.ServidoresParaCliente(agora)
	if len(servidores) != 1 {
		t.Fatalf("esperado um servidor TURN, obtido %d", len(servidores))
	}
	turn := servidores[0]
	if len(turn.URLs) != 2 {
		t.Fatalf("as duas URLs de TURN deveriam vir juntas, obtido %v", turn.URLs)
	}

	// usuario = "<expiracao>:<nome>"
	partes := strings.SplitN(turn.Username, ":", 2)
	if len(partes) != 2 {
		t.Fatalf("usuario fora do formato esperado: %q", turn.Username)
	}
	expiracao, err := strconv.ParseInt(partes[0], 10, 64)
	if err != nil {
		t.Fatalf("expiracao nao numerica: %q", partes[0])
	}
	if expiracao != agora.Add(time.Hour).Unix() {
		t.Fatalf("expiracao = %d, esperado %d", expiracao, agora.Add(time.Hour).Unix())
	}

	// senha = base64(HMAC-SHA1(segredo, usuario))
	mac := hmac.New(sha1.New, []byte(segredo))
	mac.Write([]byte(turn.Username))
	esperado := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if turn.Credential != esperado {
		t.Fatalf("credencial = %q, esperado %q", turn.Credential, esperado)
	}

	if cfg.ValidadeSegundos() != 3600 {
		t.Fatalf("validade = %d, esperado 3600", cfg.ValidadeSegundos())
	}
}

func TestCredencialMudaComOTempo(t *testing.T) {
	cfg, _ := NovaConfigICE("", "turn:turn.exemplo.com:3478", "segredo", 60)
	base := time.Unix(1700000000, 0).UTC()
	primeira := cfg.ServidoresParaCliente(base)[0].Credential
	segunda := cfg.ServidoresParaCliente(base.Add(time.Minute))[0].Credential
	if primeira == segunda {
		t.Fatal("credenciais geradas em momentos diferentes deveriam diferir")
	}
}

func TestParaPionConverteCredencial(t *testing.T) {
	cfg, _ := NovaConfigICE(`[{"urls":["stun:stun.exemplo:3478"]}]`, "turn:turn.exemplo:3478", "segredo", 60)
	pion := cfg.ParaPion(time.Now())
	if len(pion) != 2 {
		t.Fatalf("esperado STUN e TURN, obtido %d", len(pion))
	}
	if pion[0].Username != "" {
		t.Fatal("STUN nao deve levar credencial para o pion")
	}
	if pion[1].Username == "" || pion[1].Credential == nil {
		t.Fatal("TURN precisa levar credencial para o pion")
	}
}
