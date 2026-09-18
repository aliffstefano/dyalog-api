package whatsapp

import (
	"testing"
	"time"

	"dyalog-api-go/internal/models"
)

func TestStatusPermiteReconexaoAutomatica(t *testing.T) {
	casos := map[string]bool{
		// Sessao valida que caiu: a API deve levantar sozinha.
		models.StatusInstanciaDesconectada: true,
		models.StatusInstanciaConectada:    true,
		"sincronizando_historico":          true,
		"pareada":                          true,
		"autenticando":                     true,
		// Dependem de acao humana: reconectar sozinho nao resolve.
		models.StatusInstanciaAguardandoQR:     false,
		models.StatusInstanciaAguardandoCodigo: false,
		models.StatusInstanciaNaoInicializada:  false,
		// Ja existe um fluxo de conexao em andamento.
		models.StatusInstanciaConectando:    false,
		models.StatusInstanciaDesconectando: false,
	}
	for status, esperado := range casos {
		if obtido := statusPermiteReconexaoAutomatica(status); obtido != esperado {
			t.Fatalf("statusPermiteReconexaoAutomatica(%q) = %v, esperado %v", status, obtido, esperado)
		}
	}
}

func TestInstanciasReconectaveisIgnoraRuntimeSemCliente(t *testing.T) {
	g := &GerenciadorInstancias{
		estados:            map[string]estadoRuntime{"a": {status: models.StatusInstanciaDesconectada}},
		runtimes:           map[string]*runtimeInstancia{"a": {}},
		reconexaoBloqueada: map[string]time.Time{},
	}
	if ids := g.InstanciasReconectaveis(); len(ids) != 0 {
		t.Fatalf("runtime sem cliente nao deve ser reconectado, obtido %v", ids)
	}
}

func TestBloquearEliberarReconexao(t *testing.T) {
	g := &GerenciadorInstancias{
		estados:            map[string]estadoRuntime{},
		runtimes:           map[string]*runtimeInstancia{},
		reconexaoBloqueada: map[string]time.Time{},
	}
	g.bloquearReconexao("a", time.Hour)
	if _, ok := g.reconexaoBloqueada["a"]; !ok {
		t.Fatal("bloquearReconexao deveria registrar o prazo")
	}
	g.liberarBloqueioReconexao("a")
	if _, ok := g.reconexaoBloqueada["a"]; ok {
		t.Fatal("liberarBloqueioReconexao deveria remover o prazo")
	}
	// Duracao nao positiva nao bloqueia nada.
	g.bloquearReconexao("b", 0)
	if _, ok := g.reconexaoBloqueada["b"]; ok {
		t.Fatal("duracao zero nao deveria bloquear reconexao")
	}
}
