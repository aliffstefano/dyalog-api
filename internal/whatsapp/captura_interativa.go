package whatsapp

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Captura de mensagens recebidas e enviadas, para comparar o formato que uma empresa
// envia (e o cliente renderiza) com o que nos enviamos. Ligada por
// CAPTURAR_MENSAGENS=true; desligada por padrao porque grava o conteudo das
// mensagens em disco. Use so numa instancia de teste.
//
// Cada captura traz tudo de uma vez, para nao precisar repetir o envio:
//   - stanza: o no <message> bruto como chegou do servidor, com atributos e
//     filhos (<biz>, <enc>, etc). Os bytes do <enc> ficam em base64.
//   - raw_message: a mensagem decifrada, antes de o whatsmeow desembrulhar
//     view once / ephemeral / device sent.
//   - raw_message_base64: o mesmo protobuf em bytes. Preserva campos que a
//     versao do whatsmeow nao conhece, que o JSON acima descartaria.
//   - info: remetente, tipo, verified name etc.

const maxStanzasCapturadas = 500

func capturaMensagensAtiva() bool {
	ativo, _ := strconv.ParseBool(strings.TrimSpace(os.Getenv("CAPTURAR_MENSAGENS")))
	return ativo
}

// stanzasCapturadas guarda os nos <message> recebidos ate o evento decifrado
// chegar, indexados por instancia + id da mensagem.
var stanzasCapturadas = struct {
	sync.Mutex
	nos   map[string]*waBinary.Node
	ordem []string
}{nos: map[string]*waBinary.Node{}}

func chaveStanza(instanciaID, mensagemID string) string {
	return instanciaID + "|" + mensagemID
}

func guardarStanza(instanciaID string, no *waBinary.Node) {
	id, _ := no.Attrs["id"].(string)
	if id == "" {
		return
	}
	chave := chaveStanza(instanciaID, id)
	stanzasCapturadas.Lock()
	defer stanzasCapturadas.Unlock()
	if _, existe := stanzasCapturadas.nos[chave]; !existe {
		stanzasCapturadas.ordem = append(stanzasCapturadas.ordem, chave)
	}
	stanzasCapturadas.nos[chave] = no
	for len(stanzasCapturadas.ordem) > maxStanzasCapturadas {
		delete(stanzasCapturadas.nos, stanzasCapturadas.ordem[0])
		stanzasCapturadas.ordem = stanzasCapturadas.ordem[1:]
	}
}

func retirarStanza(instanciaID, mensagemID string) *waBinary.Node {
	chave := chaveStanza(instanciaID, mensagemID)
	stanzasCapturadas.Lock()
	defer stanzasCapturadas.Unlock()
	no := stanzasCapturadas.nos[chave]
	delete(stanzasCapturadas.nos, chave)
	return no
}

// loggerCaptura intercepta os logs "Recv" e "Send" do whatsmeow, que recebem
// cada no do socket. Funciona em qualquer nivel de log, porque o whatsmeow chama
// Debugf sempre e o filtro fica no logger.
//
//   - Recv <message>: guardado ate o evento decifrado chegar (capturarMensagem)
//   - Recv <ack error="...">: recusa do servidor a algo que enviamos, gravada na hora
//   - Send <message>: o stanza exato que enviamos, gravado na hora
type loggerCaptura struct {
	waLog.Logger
	instanciaID string
	diretorio   string
	modulo      string
}

func loggerComCaptura(base waLog.Logger, instanciaID, diretorioBase string) waLog.Logger {
	if !capturaMensagensAtiva() {
		return base
	}
	return loggerCaptura{Logger: base, instanciaID: instanciaID, diretorio: diretorioBase}
}

func (l loggerCaptura) Sub(modulo string) waLog.Logger {
	return loggerCaptura{Logger: l.Logger.Sub(modulo), instanciaID: l.instanciaID, diretorio: l.diretorio, modulo: modulo}
}

func (l loggerCaptura) Debugf(msg string, args ...any) {
	for _, arg := range args {
		no, ok := arg.(*waBinary.Node)
		if !ok || no == nil {
			continue
		}
		id, _ := no.Attrs["id"].(string)
		switch {
		case l.modulo == "Recv" && no.Tag == "message":
			guardarStanza(l.instanciaID, no)
		case l.modulo == "Recv" && no.Tag == "ack" && no.Attrs["error"] != nil:
			gravarArquivoCaptura(l.diretorio, l.instanciaID+"_recusa_"+id, map[string]interface{}{
				"instancia":   l.instanciaID,
				"mensagem_id": id,
				"recusa":      noParaJSON(no),
			})
		case l.modulo == "Send" && no.Tag == "message":
			gravarArquivoCaptura(l.diretorio, l.instanciaID+"_enviada_"+id, map[string]interface{}{
				"instancia":   l.instanciaID,
				"mensagem_id": id,
				"stanza":      noParaJSON(no),
			})
		}
	}
	l.Logger.Debugf(msg, args...)
}

// noParaJSON converte o no binario numa estrutura legivel, sem truncar nada.
func noParaJSON(no *waBinary.Node) map[string]interface{} {
	if no == nil {
		return nil
	}
	saida := map[string]interface{}{"tag": no.Tag}
	if len(no.Attrs) > 0 {
		attrs := make(map[string]string, len(no.Attrs))
		for chave, valor := range no.Attrs {
			attrs[chave] = fmt.Sprint(valor)
		}
		saida["attrs"] = attrs
	}
	switch conteudo := no.Content.(type) {
	case []waBinary.Node:
		filhos := make([]map[string]interface{}, 0, len(conteudo))
		for i := range conteudo {
			filhos = append(filhos, noParaJSON(&conteudo[i]))
		}
		saida["content"] = filhos
	case []byte:
		saida["content_base64"] = base64.StdEncoding.EncodeToString(conteudo)
	case nil:
	default:
		saida["content"] = fmt.Sprint(conteudo)
	}
	return saida
}

// capturarMensagem grava tudo o que se sabe de uma mensagem recebida. Pula as
// que a propria instancia enviou.
func (g *GerenciadorInstancias) capturarMensagem(instanciaID string, evento *events.Message) {
	if !capturaMensagensAtiva() || evento.Info.IsFromMe {
		return
	}
	registro := map[string]interface{}{
		"instancia":   instanciaID,
		"mensagem_id": evento.Info.ID,
		"info":        evento.Info,
		"view_once":   evento.IsViewOnce,
		"ephemeral":   evento.IsEphemeral,
		"stanza":      noParaJSON(retirarStanza(instanciaID, string(evento.Info.ID))),
	}
	if evento.RawMessage != nil {
		if bruta, err := protojson.Marshal(evento.RawMessage); err == nil {
			registro["raw_message"] = json.RawMessage(bruta)
		} else {
			registro["raw_message_erro"] = err.Error()
		}
		if binario, err := proto.Marshal(evento.RawMessage); err == nil {
			registro["raw_message_base64"] = base64.StdEncoding.EncodeToString(binario)
		}
	}
	g.gravarCaptura(instanciaID, string(evento.Info.ID), registro)
}

// capturarIndecifravel grava mensagens que o whatsmeow nao conseguiu decifrar;
// ao menos o stanza fica registrado.
func (g *GerenciadorInstancias) capturarIndecifravel(instanciaID string, evento *events.UndecryptableMessage) {
	if !capturaMensagensAtiva() || evento.Info.IsFromMe {
		return
	}
	g.gravarCaptura(instanciaID, string(evento.Info.ID), map[string]interface{}{
		"instancia":    instanciaID,
		"mensagem_id":  evento.Info.ID,
		"info":         evento.Info,
		"indecifravel": true,
		"tipo_falha":   evento.DecryptFailMode,
		"indisponivel": evento.IsUnavailable,
		"tipo_indisp":  evento.UnavailableType,
		"stanza":       noParaJSON(retirarStanza(instanciaID, string(evento.Info.ID))),
	})
}

func (g *GerenciadorInstancias) gravarCaptura(instanciaID, mensagemID string, registro map[string]interface{}) {
	gravarArquivoCaptura(g.diretorioBase, instanciaID+"_"+mensagemID, registro)
}

func gravarArquivoCaptura(diretorioBase, nome string, registro map[string]interface{}) {
	conteudo, err := json.MarshalIndent(registro, "", "  ")
	if err != nil {
		fmt.Printf("captura: erro ao montar registro %s: %v\n", nome, err)
		return
	}
	diretorio := filepath.Join(diretorioBase, "capturas")
	if err := os.MkdirAll(diretorio, 0o755); err != nil {
		fmt.Printf("captura: erro ao criar %s: %v\n", diretorio, err)
		return
	}
	arquivo := filepath.Join(diretorio, nome+".json")
	if err := os.WriteFile(arquivo, conteudo, 0o644); err != nil {
		fmt.Printf("captura: erro ao gravar %s: %v\n", arquivo, err)
		return
	}
	fmt.Printf("captura: %s gravada em %s\n", nome, arquivo)
}
