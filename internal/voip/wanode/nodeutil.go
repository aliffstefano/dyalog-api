package wanode

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	waBinary "go.mau.fi/whatsmeow/binary"
)

func NodeChildren(n *waBinary.Node) []waBinary.Node {
	if n == nil {
		return nil
	}
	if children, ok := n.Content.([]waBinary.Node); ok {
		return children
	}
	return nil
}

func NodeBytes(n *waBinary.Node) []byte {
	if n == nil {
		return nil
	}
	if b, ok := n.Content.([]byte); ok {
		return b
	}
	return nil
}

func AttrString(attrs waBinary.Attrs, key string) string {
	v, ok := attrs[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case uint64:
		return strconv.FormatUint(t, 10)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func AttrInt(attrs waBinary.Attrs, key string, fallback int) int {
	s := AttrString(attrs, key)
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func HasAttr(attrs waBinary.Attrs, key string) bool {
	v, ok := attrs[key]
	return ok && v != nil && AttrString(attrs, key) != ""
}

// DebugXMLEnabled liga o log do XML bruto das stanzas de chamada
// (VOIP_DEBUG_XML=1). Serve para comparar o que chega e o que enviamos.
func DebugXMLEnabled() bool { return os.Getenv("VOIP_DEBUG_XML") == "1" }

// DumpNode serializa o no em XML curto para log. Conteudo binario (enc, chaves,
// tokens de relay) vira so o tamanho, e atributos sensiveis sao omitidos.
func DumpNode(n *waBinary.Node) string {
	if n == nil {
		return "<nil>"
	}
	var b strings.Builder
	dumpNode(&b, n)
	return b.String()
}

func dumpNode(b *strings.Builder, n *waBinary.Node) {
	b.WriteString("<" + n.Tag)
	keys := make([]string, 0, len(n.Attrs))
	for k := range n.Attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := fmt.Sprint(n.Attrs[k])
		switch k {
		case "token", "key", "relay-key", "hbh_key":
			v = "<redacted>"
		}
		fmt.Fprintf(b, " %s=%q", k, v)
	}
	switch c := n.Content.(type) {
	case nil:
		b.WriteString("/>")
	case []byte:
		fmt.Fprintf(b, ">[%d bytes]</%s>", len(c), n.Tag)
	case []waBinary.Node:
		b.WriteString(">")
		for i := range c {
			dumpNode(b, &c[i])
		}
		b.WriteString("</" + n.Tag + ">")
	default:
		fmt.Fprintf(b, ">%v</%s>", c, n.Tag)
	}
}
