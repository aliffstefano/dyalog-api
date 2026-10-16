package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// StatusPublicoService: pagina publica (link secreto) para o cliente final ver
// se as instancias da conta estao no ar. Nao mostra numero, id nem token.
type StatusPublicoService struct {
	paginas    store.PaginaStatusStore
	instancias store.InstanciaStore
	auditoria  store.AuditoriaStore
	baseURL    string
	agora      func() time.Time
}

const diasStatusPublico = 30

func NovoStatusPublicoService(paginas store.PaginaStatusStore, instancias store.InstanciaStore, auditoria store.AuditoriaStore, baseURL string) *StatusPublicoService {
	return &StatusPublicoService{paginas: paginas, instancias: instancias, auditoria: auditoria, baseURL: strings.TrimRight(baseURL, "/"), agora: time.Now}
}

// ErrStatusPublicoNaoEncontrado: link invalido ou pagina desligada.
var ErrStatusPublicoNaoEncontrado = errors.New("pagina de status nao encontrada")

func tokenStatus() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "st_" + hex.EncodeToString(b)
}

func (s *StatusPublicoService) link(token string) string {
	if token == "" {
		return ""
	}
	return s.baseURL + "/status/" + token
}

// Config devolve a pagina da conta (desligada e sem link se nunca foi criada).
func (s *StatusPublicoService) Config(ctx context.Context, contaID string) (models.PaginaStatus, error) {
	p, err := s.paginas.PaginaStatusDaConta(ctx, contaID)
	if errors.Is(err, store.ErrNaoEncontrado) {
		return models.PaginaStatus{ContaID: contaID, Titulo: "Status do atendimento", InstanciasOcultas: []string{}}, nil
	}
	if err != nil {
		return p, err
	}
	p.Link = s.link(p.Token)
	return p, nil
}

func (s *StatusPublicoService) Salvar(ctx context.Context, contaID, titulo string, ativa bool, ocultas []string, novoLink bool) (models.PaginaStatus, error) {
	p, err := s.Config(ctx, contaID)
	if err != nil {
		return p, err
	}
	titulo = strings.TrimSpace(titulo)
	if titulo == "" {
		titulo = "Status do atendimento"
	}
	if len([]rune(titulo)) > 80 {
		return p, errors.Join(ErrEntradaInvalida, errors.New("titulo ate 80 caracteres"))
	}
	p.ContaID, p.Titulo, p.Ativa = contaID, titulo, ativa
	if ocultas != nil {
		p.InstanciasOcultas = ocultas
	}
	if p.Token == "" || novoLink {
		p.Token = tokenStatus()
	}
	if err := s.paginas.SalvarPaginaStatus(ctx, p); err != nil {
		return p, err
	}
	p.Link = s.link(p.Token)
	return p, nil
}

var (
	eventosNoAr   = map[string]bool{"instancia.conectada": true}
	eventosForaAr = map[string]bool{
		"instancia.desconectada": true, "instancia.sessao_substituida": true, "instancia.falha_conexao": true,
		"instancia.banimento_temporario": true, "instancia.dispositivo_removido": true, "instancia.desconectar": true,
	}
)

func estadoPublico(status string) string {
	switch status {
	case models.StatusInstanciaConectada:
		return "operando"
	case models.StatusInstanciaConectando, models.StatusInstanciaAguardandoQR, models.StatusInstanciaAguardandoCodigo, "pareada", "autenticando", "sincronizando_historico":
		return "reconectando"
	}
	return "fora"
}

// Publico monta a pagina a partir do link secreto.
func (s *StatusPublicoService) Publico(ctx context.Context, token string) (models.StatusPublico, error) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "st_") {
		return models.StatusPublico{}, ErrStatusPublicoNaoEncontrado
	}
	p, err := s.paginas.PaginaStatusPorToken(ctx, token)
	if err != nil || !p.Ativa {
		return models.StatusPublico{}, ErrStatusPublicoNaoEncontrado
	}
	todas, err := s.instancias.Listar(ctx)
	if err != nil {
		return models.StatusPublico{}, err
	}
	ocultas := map[string]bool{}
	for _, id := range p.InstanciasOcultas {
		ocultas[id] = true
	}
	agora := s.agora()
	resultado := models.StatusPublico{Titulo: p.Titulo, AtualizadoEm: agora, Instancias: []models.StatusPublicoItem{}}
	operando, total := 0, 0
	for _, inst := range todas {
		if inst.ContaID != p.ContaID || ocultas[inst.ID] {
			continue
		}
		item := models.StatusPublicoItem{Nome: inst.Nome, Estado: estadoPublico(inst.Status), Desde: inst.AtualizadoEm}
		item.Dias, item.Disponibilidade = s.disponibilidade(ctx, inst, item.Estado == "operando", agora)
		resultado.Instancias = append(resultado.Instancias, item)
		total++
		if item.Estado == "operando" {
			operando++
		}
	}
	sort.SliceStable(resultado.Instancias, func(i, j int) bool {
		return strings.ToLower(resultado.Instancias[i].Nome) < strings.ToLower(resultado.Instancias[j].Nome)
	})
	switch {
	case total == 0 || operando == total:
		resultado.Geral = "operando"
	case operando == 0:
		resultado.Geral = "fora"
	default:
		resultado.Geral = "parcial"
	}
	return resultado, nil
}

type eventoConexao struct {
	quando time.Time
	noAr   bool
}

func (s *StatusPublicoService) disponibilidade(ctx context.Context, inst models.Instancia, noArAgora bool, agora time.Time) ([]float64, float64) {
	local := agora.Location()
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, local)
	inicio := hoje.AddDate(0, 0, -(diasStatusPublico - 1))
	var eventos []eventoConexao
	if s.auditoria != nil {
		pagina, err := s.auditoria.ListarAuditoria(ctx, models.FiltroAuditoria{InstanciaID: inst.ID, Tipo: models.AuditoriaConexao, De: inicio, Limite: 500})
		if err == nil {
			for _, r := range pagina.Registros {
				switch {
				case eventosNoAr[r.Acao]:
					eventos = append(eventos, eventoConexao{r.CriadoEm, true})
				case eventosForaAr[r.Acao]:
					eventos = append(eventos, eventoConexao{r.CriadoEm, false})
				}
			}
		}
	}
	return calcularDisponibilidade(eventos, noArAgora, inst.CriadoEm, inicio, agora, diasStatusPublico)
}

// calcularDisponibilidade divide [inicio, agora] em dias e mede a fracao de
// cada dia no ar. Dias antes da criacao da instancia ficam -1 (sem dados). O
// estado antes do primeiro evento e o contrario dele (ou o atual, sem eventos).
func calcularDisponibilidade(eventos []eventoConexao, noArAgora bool, criadaEm, inicio, agora time.Time, dias int) ([]float64, float64) {
	sort.Slice(eventos, func(i, j int) bool { return eventos[i].quando.Before(eventos[j].quando) })
	estado := noArAgora
	if len(eventos) > 0 {
		estado = !eventos[0].noAr
	}
	porDia := make([]float64, dias)
	var noArTotal, medidoTotal time.Duration
	e := 0
	for d := 0; d < dias; d++ {
		diaIni := inicio.AddDate(0, 0, d)
		diaFim := diaIni.AddDate(0, 0, 1)
		if diaFim.After(agora) {
			diaFim = agora
		}
		comeco := diaIni
		if !criadaEm.IsZero() && criadaEm.After(comeco) {
			comeco = criadaEm
		}
		if !comeco.Before(diaFim) {
			porDia[d] = -1
			// avanca o estado pelos eventos deste dia mesmo assim
			for e < len(eventos) && eventos[e].quando.Before(diaFim) {
				estado = eventos[e].noAr
				e++
			}
			continue
		}
		var noAr time.Duration
		cursor := comeco
		for e < len(eventos) && eventos[e].quando.Before(diaFim) {
			if eventos[e].quando.After(cursor) {
				if estado {
					noAr += eventos[e].quando.Sub(cursor)
				}
				cursor = eventos[e].quando
			}
			estado = eventos[e].noAr
			e++
		}
		if estado {
			noAr += diaFim.Sub(cursor)
		}
		medido := diaFim.Sub(comeco)
		porDia[d] = float64(noAr) / float64(medido)
		noArTotal += noAr
		medidoTotal += medido
	}
	if medidoTotal == 0 {
		return porDia, -1
	}
	return porDia, float64(noArTotal) / float64(medidoTotal)
}
