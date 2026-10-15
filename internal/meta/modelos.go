package meta

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"dyalog-api-go/internal/models"
)

// ErrSemWABA indica instancia sem o WABA ID, necessario para listar os
// modelos (eles ficam na conta do WhatsApp Business, nao no numero).
var ErrSemWABA = errors.New("informe o WABA ID da instancia (Configuracoes > API oficial > Atualizar credenciais) para listar os modelos; ele tambem e preenchido sozinho quando chega o primeiro webhook da Meta")

// Modelo e um modelo de mensagem (template) da conta, ja resumido para quem
// vai preencher: textos, formato do cabecalho e as variaveis de cada parte.
type Modelo struct {
	ID                 string           `json:"id"`
	Nome               string           `json:"nome"`
	Idioma             string           `json:"idioma"`
	Status             string           `json:"status"`
	Categoria          string           `json:"categoria"`
	FormatoParametros  string           `json:"formato_parametros"`
	CabecalhoFormato   string           `json:"cabecalho_formato,omitempty"`
	Cabecalho          string           `json:"cabecalho,omitempty"`
	Corpo              string           `json:"corpo"`
	Rodape             string           `json:"rodape,omitempty"`
	Botoes             []string         `json:"botoes,omitempty"`
	VariaveisCabecalho []string         `json:"variaveis_cabecalho,omitempty"`
	VariaveisCorpo     []string         `json:"variaveis_corpo,omitempty"`
	Componentes        []map[string]any `json:"componentes"`
}

// Aprovado diz se o modelo ja pode ser enviado.
func (m Modelo) Aprovado() bool { return strings.EqualFold(m.Status, "APPROVED") }

// Renderizar troca as variaveis do corpo pelos valores ({{1}} -> valores["1"]
// ou {{nome}} -> valores["nome"]), para mostrar o texto enviado no chat.
func (m Modelo) Renderizar(valores map[string]string) string {
	return variavelModelo.ReplaceAllStringFunc(m.Corpo, func(trecho string) string {
		nome := variavelModelo.FindStringSubmatch(trecho)[1]
		if valor, ok := valores[nome]; ok {
			return valor
		}
		return trecho
	})
}

type templateGraph struct {
	ID                string           `json:"id"`
	Nome              string           `json:"name"`
	Idioma            string           `json:"language"`
	Status            string           `json:"status"`
	Categoria         string           `json:"category"`
	FormatoParametros string           `json:"parameter_format"`
	Componentes       []map[string]any `json:"components"`
}

var variavelModelo = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_]+)\s*\}\}`)

// variaveis devolve as variaveis do texto na ordem em que aparecem, sem
// repetir ({{1}}, {{2}} ou nomes como {{nome_cliente}}).
func variaveis(texto string) []string {
	var lista []string
	vistas := map[string]bool{}
	for _, m := range variavelModelo.FindAllStringSubmatch(texto, -1) {
		if !vistas[m[1]] {
			vistas[m[1]] = true
			lista = append(lista, m[1])
		}
	}
	return lista
}

func resumirModelo(t templateGraph) Modelo {
	m := Modelo{
		ID: t.ID, Nome: t.Nome, Idioma: t.Idioma, Status: t.Status, Categoria: t.Categoria,
		FormatoParametros: strings.ToLower(t.FormatoParametros), Componentes: t.Componentes,
	}
	if m.FormatoParametros == "" {
		m.FormatoParametros = "positional"
	}
	for _, c := range t.Componentes {
		texto, _ := c["text"].(string)
		switch strings.ToUpper(fmt.Sprint(c["type"])) {
		case "HEADER":
			m.CabecalhoFormato, _ = c["format"].(string)
			m.Cabecalho = texto
			m.VariaveisCabecalho = variaveis(texto)
		case "BODY":
			m.Corpo = texto
			m.VariaveisCorpo = variaveis(texto)
		case "FOOTER":
			m.Rodape = texto
		case "BUTTONS":
			botoes, _ := c["buttons"].([]any)
			for _, b := range botoes {
				if botao, ok := b.(map[string]any); ok {
					if rotulo, _ := botao["text"].(string); rotulo != "" {
						m.Botoes = append(m.Botoes, rotulo)
					}
				}
			}
		}
	}
	return m
}

// ListarModelos busca todos os modelos da conta (WABA), seguindo a paginacao.
func (c *Cliente) ListarModelos(ctx context.Context, cred models.CredenciaisMeta) ([]Modelo, error) {
	if strings.TrimSpace(cred.WABAID) == "" {
		return nil, ErrSemWABA
	}
	url := c.url(cred.WABAID+"/message_templates") + "?fields=id,name,language,status,category,parameter_format,components&limit=100"
	var modelos []Modelo
	for pagina := 0; url != "" && pagina < 20; pagina++ {
		var resp struct {
			Dados  []templateGraph `json:"data"`
			Paging struct {
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := c.requisitar(ctx, http.MethodGet, url, cred.AccessToken, nil, "", &resp); err != nil {
			return nil, err
		}
		for _, t := range resp.Dados {
			modelos = append(modelos, resumirModelo(t))
		}
		url = resp.Paging.Next
	}
	return modelos, nil
}

// cacheModelos evita buscar a lista na Meta a cada envio de modelo.
type cacheModelos struct {
	mu     sync.Mutex
	listas map[string]listaModelos
}

type listaModelos struct {
	modelos []Modelo
	em      time.Time
}

const validadeCacheModelos = 10 * time.Minute

// Modelos lista os modelos da instancia. atualizar ignora o cache.
func (e *Enviador) Modelos(ctx context.Context, instanciaID string, atualizar bool) ([]Modelo, error) {
	e.modelos.mu.Lock()
	if e.modelos.listas == nil {
		e.modelos.listas = map[string]listaModelos{}
	}
	guardada, ok := e.modelos.listas[instanciaID]
	e.modelos.mu.Unlock()
	if ok && !atualizar && time.Since(guardada.em) < validadeCacheModelos {
		return guardada.modelos, nil
	}
	cred, err := e.Credenciais(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	modelos, err := e.cliente.ListarModelos(ctx, cred)
	if err != nil {
		return nil, err
	}
	e.modelos.mu.Lock()
	e.modelos.listas[instanciaID] = listaModelos{modelos: modelos, em: time.Now()}
	e.modelos.mu.Unlock()
	return modelos, nil
}

// TextoModelo devolve o corpo do modelo com os parametros, para o chat do
// painel. Sem conseguir (lista indisponivel), devolve vazio.
func (e *Enviador) TextoModelo(ctx context.Context, instanciaID, nome, idioma string, valores map[string]string) string {
	modelos, err := e.Modelos(ctx, instanciaID, false)
	if err != nil {
		return ""
	}
	for _, m := range modelos {
		if m.Nome == nome && (idioma == "" || m.Idioma == idioma) {
			return m.Renderizar(valores)
		}
	}
	return ""
}

// EsquecerModelos descarta a lista guardada (ao trocar as credenciais).
func (e *Enviador) EsquecerModelos(instanciaID string) {
	e.modelos.mu.Lock()
	delete(e.modelos.listas, instanciaID)
	e.modelos.mu.Unlock()
}

// ValoresModelo junta os valores das variaveis do corpo de um envio de modelo,
// venham de parametros (posicionais) ou de componentes (formato da Meta).
func ValoresModelo(req models.EnvioTemplateRequest) map[string]string {
	valores := map[string]string{}
	if len(req.Componentes) == 0 {
		for i, p := range req.Parametros {
			valores[fmt.Sprint(i+1)] = p
		}
		return valores
	}
	for _, c := range req.Componentes {
		if !strings.EqualFold(fmt.Sprint(c["type"]), "body") {
			continue
		}
		parametros, _ := c["parameters"].([]any)
		for i, item := range parametros {
			p, _ := item.(map[string]any)
			texto, _ := p["text"].(string)
			nome, _ := p["parameter_name"].(string)
			if nome == "" {
				nome = fmt.Sprint(i + 1)
			}
			valores[nome] = texto
		}
	}
	return valores
}
