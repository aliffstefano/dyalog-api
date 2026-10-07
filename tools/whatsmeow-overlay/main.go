// whatsmeow-overlay copia o whatsmeow do cache de modulos para um diretorio e
// aplica ajustes minimos no send.go:
//
//  1. Quando a mensagem ja traz um no <biz> proprio em
//     SendRequestExtra.AdditionalNodes, o whatsmeow deixa de anexar o <biz>
//     dele. Sem isso nao ha como enviar uma ListMessage com o mesmo envelope
//     que as empresas usam: o whatsmeow sempre acrescenta <biz><list v="2"/>,
//     e dois <biz> no stanza dao erro 479.
//  2. Evento (EventMessage) sai com type="event" e <meta event_type="creation"/>,
//     como o app oficial e o Baileys mandam. O whatsmeow manda como texto e o
//     evento nao chega. Vale tambem para os reenvios (retry), que passam pelas
//     mesmas funcoes.
//
// Uso (ver Dockerfile):
//
//	go run ./tools/whatsmeow-overlay /tmp/whatsmeow
//	go mod edit -replace go.mau.fi/whatsmeow=/tmp/whatsmeow
//	go build ./cmd/api
//
// O replace so existe dentro do build; o go.mod do repositorio nao muda. Sem o
// ajuste (go build direto) tudo funciona, exceto o modo de lista "lista_biz",
// que sai com <biz> duplicado e e recusado pelo servidor, e o envio de evento.
//
// Falha (e derruba o build) se o trecho esperado sumir numa versao nova do
// whatsmeow, para o ajuste nunca se perder em silencio.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ajustes sao as trocas feitas no send.go. Cada trecho original precisa
// aparecer exatamente uma vez.
var ajustes = []struct{ alvo, substituto string }{
	{
		`if buttonType := getButtonTypeFromMessage(message); buttonType != "" {`,
		`if buttonType := getButtonTypeFromMessage(message); buttonType != "" && !dyalogTemBiz(extraParams.additionalNodes) {`,
	},
	{
		`case msg.PollCreationMessage != nil, msg.PollUpdateMessage != nil:`,
		`case msg.EventMessage != nil:
		return "event"
	case msg.PollCreationMessage != nil, msg.PollUpdateMessage != nil:`,
	},
	{
		`if extraParams.botNode != nil {`,
		`if msgAttrs["type"] == "event" {
		content = append(content, waBinary.Node{Tag: "meta", Attrs: waBinary.Attrs{"event_type": "creation"}})
	}
	if extraParams.botNode != nil {`,
	},
}

const funcaoExtra = `

// dyalogTemBiz e injetado pelo overlay da Dyalog Connect (tools/whatsmeow-overlay).
func dyalogTemBiz(nos *[]waBinary.Node) bool {
	if nos == nil {
		return false
	}
	for _, no := range *nos {
		if no.Tag == "biz" {
			return true
		}
	}
	return false
}
`

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "uso: whatsmeow-overlay <diretorio-de-saida>")
		os.Exit(2)
	}
	if err := gerar(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, "whatsmeow-overlay:", err)
		os.Exit(1)
	}
}

func gerar(saida string) error {
	dir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "go.mau.fi/whatsmeow").Output()
	if err != nil {
		return fmt.Errorf("localizar whatsmeow (rode go mod download antes): %w", err)
	}
	origem := strings.TrimSpace(string(dir))
	if err := os.RemoveAll(saida); err != nil {
		return err
	}
	if err := copiarDiretorio(origem, saida); err != nil {
		return fmt.Errorf("copiar whatsmeow: %w", err)
	}

	arquivo := filepath.Join(saida, "send.go")
	conteudo, err := os.ReadFile(arquivo)
	if err != nil {
		return err
	}
	texto := string(conteudo)
	for _, ajuste := range ajustes {
		if strings.Count(texto, ajuste.alvo) != 1 {
			return fmt.Errorf("trecho %q nao encontrado em %s; o whatsmeow mudou e o ajuste precisa ser revisto", ajuste.alvo, filepath.Join(origem, "send.go"))
		}
		texto = strings.Replace(texto, ajuste.alvo, ajuste.substituto, 1)
	}
	return os.WriteFile(arquivo, []byte(texto+funcaoExtra), 0o644)
}

// copiarDiretorio copia a arvore com permissoes de escrita (o cache de modulos
// e somente leitura).
func copiarDiretorio(origem, destino string) error {
	return filepath.WalkDir(origem, func(caminho string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativo, err := filepath.Rel(origem, caminho)
		if err != nil {
			return err
		}
		alvo := filepath.Join(destino, relativo)
		if entrada.IsDir() {
			return os.MkdirAll(alvo, 0o755)
		}
		dados, err := os.ReadFile(caminho)
		if err != nil {
			return err
		}
		return os.WriteFile(alvo, dados, 0o644)
	})
}
