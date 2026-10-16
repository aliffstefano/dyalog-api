package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"dyalog-api-go/internal/email"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// AuditoriaService guarda "quem fez o que" por instancia e por conta: acoes no
// painel/API, eventos de conexao e acessos. Tambem avisa por e-mail os donos e
// admins quando uma instancia cai de um jeito que exige alguem (sessao
// encerrada pelo celular, banimento, falha de conexao).
type AuditoriaService struct {
	store      store.AuditoriaStore
	instancias store.InstanciaStore
	contas     store.ContaStore
	email      email.Enviador
	baseURL    string
	retencao   time.Duration

	alertasMu sync.Mutex
	alertas   map[string]time.Time
}

// intervaloAlertaInstancia evita uma enxurrada de e-mails quando a instancia
// fica caindo e voltando.
const intervaloAlertaInstancia = time.Hour

func NovoAuditoriaService(auditoria store.AuditoriaStore, instancias store.InstanciaStore, contas store.ContaStore, enviador email.Enviador, baseURL string, retencaoDias int) *AuditoriaService {
	if retencaoDias <= 0 {
		retencaoDias = 90
	}
	return &AuditoriaService{
		store: auditoria, instancias: instancias, contas: contas, email: enviador,
		baseURL: strings.TrimRight(baseURL, "/"), retencao: time.Duration(retencaoDias) * 24 * time.Hour,
		alertas: map[string]time.Time{},
	}
}

// Iniciar apaga registros mais velhos que a retencao, uma vez por dia.
func (s *AuditoriaService) Iniciar(ctx context.Context) {
	if s == nil || s.store == nil {
		return
	}
	go func() {
		for {
			if n, err := s.store.LimparAuditoriaAntiga(ctx, time.Now().Add(-s.retencao)); err != nil {
				fmt.Printf("auditoria: erro na limpeza: %v\n", err)
			} else if n > 0 {
				fmt.Printf("auditoria: %d registros antigos apagados\n", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(24 * time.Hour):
			}
		}
	}()
}

// Ator descreve quem fez a acao a partir do acesso autenticado.
func Ator(acesso models.AcessoDashboard) (tipo, id, nome string) {
	switch acesso.Tipo {
	case "usuario":
		return "usuario", acesso.UsuarioID, strings.TrimSpace(cmpOr(acesso.Nome, acesso.Email))
	case "master":
		return "master", "", "Token master"
	case "instancia":
		return "token", acesso.InstanciaID, "Token da instancia"
	default:
		return "sistema", "", "Sistema"
	}
}

// Registrar grava o registro. Falha de auditoria nunca derruba a acao: so loga.
func (s *AuditoriaService) Registrar(ctx context.Context, r models.RegistroAuditoria) {
	if s == nil || s.store == nil {
		return
	}
	if r.ContaID == "" && r.InstanciaID != "" && s.instancias != nil {
		if inst, err := s.instancias.BuscarPorID(ctx, r.InstanciaID); err == nil {
			r.ContaID = inst.ContaID
		}
	}
	if err := s.store.RegistrarAuditoria(ctx, r); err != nil {
		fmt.Printf("auditoria: %v\n", err)
	}
}

// RegistrarAcao registra uma acao feita por um acesso autenticado.
func (s *AuditoriaService) RegistrarAcao(ctx context.Context, acesso models.AcessoDashboard, ip, instanciaID, tipo, acao, descricao, severidade, detalhes string) {
	atorTipo, atorID, atorNome := Ator(acesso)
	contaID := ""
	if instanciaID == "" {
		contaID = acesso.ContaID
	}
	s.Registrar(ctx, models.RegistroAuditoria{
		InstanciaID: instanciaID, ContaID: contaID, Tipo: tipo, Acao: acao, Descricao: descricao,
		Severidade: severidade, AtorTipo: atorTipo, AtorID: atorID, AtorNome: atorNome, IP: ip, Detalhes: detalhes,
	})
}

// EventoConexao recebe os eventos do gerenciador (conectou, caiu, baniu...).
func (s *AuditoriaService) EventoConexao(instanciaID, acao, descricao, severidade string) {
	if s == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	s.Registrar(ctx, models.RegistroAuditoria{
		InstanciaID: instanciaID, Tipo: models.AuditoriaConexao, Acao: acao, Descricao: descricao,
		Severidade: severidade, AtorTipo: "sistema", AtorNome: "Sistema",
	})
	if severidade == models.SeveridadeErro {
		s.alertarQueda(ctx, instanciaID, descricao)
	}
}

func (s *AuditoriaService) alertarQueda(ctx context.Context, instanciaID, descricao string) {
	if s.email == nil || !s.email.Configurado() || s.instancias == nil || s.contas == nil {
		return
	}
	s.alertasMu.Lock()
	if ultimo, ok := s.alertas[instanciaID]; ok && time.Since(ultimo) < intervaloAlertaInstancia {
		s.alertasMu.Unlock()
		return
	}
	s.alertas[instanciaID] = time.Now()
	s.alertasMu.Unlock()

	inst, err := s.instancias.BuscarPorID(ctx, instanciaID)
	if err != nil || inst.ContaID == "" {
		return
	}
	usuarios, err := s.contas.ListarUsuarios(ctx, inst.ContaID, false)
	if err != nil {
		return
	}
	html := email.Modelo("A instancia "+inst.Nome+" caiu", []string{
		"A instancia <strong>" + escaparHTML(inst.Nome) + "</strong> foi desconectada do WhatsApp e precisa de atencao.",
		"Motivo: " + escaparHTML(descricao) + ".",
		"Abra o painel para reconectar. Enquanto isso, mensagens nao sao enviadas nem recebidas por ela.",
	}, "Abrir o painel", s.baseURL+"/")
	for _, u := range usuarios {
		if !u.Ativo || (u.Papel != models.PapelDono && u.Papel != models.PapelAdmin) {
			continue
		}
		if err := s.email.Enviar(ctx, u.Email, "Instancia "+inst.Nome+" desconectada", html); err != nil {
			fmt.Printf("auditoria: erro ao enviar alerta para %s: %v\n", u.Email, err)
		}
	}
}

func escaparHTML(texto string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(texto)
}

// Listar aplica o escopo do acesso: master/superadmin veem tudo, usuario so a
// propria conta, token de instancia so a propria instancia.
func (s *AuditoriaService) Listar(ctx context.Context, acesso models.AcessoDashboard, filtro models.FiltroAuditoria) (models.PaginaAuditoria, error) {
	if s == nil || s.store == nil {
		return models.PaginaAuditoria{Registros: []models.RegistroAuditoria{}}, nil
	}
	switch {
	case acesso.TudoLiberado():
	case acesso.Tipo == "instancia":
		if filtro.InstanciaID != "" && filtro.InstanciaID != acesso.InstanciaID {
			return models.PaginaAuditoria{}, ErrAcessoNegado
		}
		filtro.InstanciaID = acesso.InstanciaID
	case acesso.Tipo == "usuario" && acesso.ContaID != "":
		filtro.SoContas = true
		filtro.ContasIDs = []string{acesso.ContaID}
	default:
		return models.PaginaAuditoria{}, ErrAcessoNegado
	}
	return s.store.ListarAuditoria(ctx, filtro)
}
