package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

// AgendamentoService guarda mensagens para enviar depois e as envia na hora.
//
// Cada replica roda o mesmo laco, mas so envia agendamentos das instancias
// cuja sessao ela detem (ou que nao estao com ninguem, e as da Meta). A
// reserva no banco garante que dois containers nunca mandem o mesmo.
type AgendamentoService struct {
	store      store.AgendamentoStore
	instancias store.InstanciaStore
	mensagens  *MensagemService
	donoSessao DonoSessao
	auditoria  *AuditoriaService
	no         string
	agora      func() time.Time
}

// DonoSessao responde se esta replica pode enviar pela instancia.
type DonoSessao interface {
	PossuiRuntime(instanciaID string) bool
	EnderecoOutroContainer(ctx context.Context, instanciaID string) string
}

const (
	maxTentativasAgendamento = 5
	esperaNovaTentativa      = 2 * time.Minute
	tamanhoMaximoAgendado    = 16 << 20 // base64 de ~12 MB
	desistirDepoisDe         = 24 * time.Hour
)

var recorrenciasValidas = map[string]bool{"": true, "diaria": true, "semanal": true, "mensal": true}
var tiposAgendamento = map[string]bool{"texto": true, "imagem": true, "documento": true, "audio": true}

func NovoAgendamentoService(st store.AgendamentoStore, instancias store.InstanciaStore, mensagens *MensagemService, dono DonoSessao, auditoria *AuditoriaService, no string) *AgendamentoService {
	return &AgendamentoService{store: st, instancias: instancias, mensagens: mensagens, donoSessao: dono, auditoria: auditoria, no: cmpOr(no, "local"), agora: time.Now}
}

func (s *AgendamentoService) validar(req *models.CriarAgendamentoRequest) error {
	req.Tipo = strings.ToLower(strings.TrimSpace(req.Tipo))
	if req.Tipo == "" {
		req.Tipo = "texto"
		if req.ArquivoBase64 != "" || req.ArquivoURL != "" {
			req.Tipo = "documento"
			if strings.HasPrefix(req.MimeType, "image/") {
				req.Tipo = "imagem"
			} else if strings.HasPrefix(req.MimeType, "audio/") {
				req.Tipo = "audio"
			}
		}
	}
	if !tiposAgendamento[req.Tipo] {
		return fmt.Errorf("%w: tipo deve ser texto, imagem, documento ou audio", ErrEntradaInvalida)
	}
	req.Recorrencia = strings.ToLower(strings.TrimSpace(req.Recorrencia))
	if !recorrenciasValidas[req.Recorrencia] {
		return fmt.Errorf("%w: recorrencia deve ser diaria, semanal, mensal ou vazia", ErrEntradaInvalida)
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	if req.Tipo == "texto" && strings.TrimSpace(req.Mensagem) == "" {
		return fmt.Errorf("%w: informe a mensagem", ErrEntradaInvalida)
	}
	if req.Tipo != "texto" && req.ArquivoBase64 == "" && req.ArquivoURL == "" {
		return fmt.Errorf("%w: informe arquivo_base64 ou arquivo_url", ErrEntradaInvalida)
	}
	if len(req.ArquivoBase64) > tamanhoMaximoAgendado {
		return fmt.Errorf("%w: arquivo agendado acima de 12 MB; use arquivo_url", ErrEntradaInvalida)
	}
	agora := s.agora()
	if req.EnviarEm.IsZero() {
		return fmt.Errorf("%w: informe enviar_em (ex.: 2026-10-20T09:00:00-04:00)", ErrEntradaInvalida)
	}
	if req.EnviarEm.Before(agora.Add(-time.Minute)) {
		return fmt.Errorf("%w: enviar_em ja passou", ErrEntradaInvalida)
	}
	if req.EnviarEm.After(agora.AddDate(1, 0, 0)) {
		return fmt.Errorf("%w: agende no maximo 1 ano para frente", ErrEntradaInvalida)
	}
	return nil
}

func (s *AgendamentoService) Criar(ctx context.Context, acesso models.AcessoDashboard, req models.CriarAgendamentoRequest) (models.Agendamento, error) {
	if err := s.validar(&req); err != nil {
		return models.Agendamento{}, err
	}
	inst, err := s.instancias.BuscarPorID(ctx, strings.TrimSpace(req.Instancia))
	if err != nil {
		return models.Agendamento{}, ErrInstanciaNaoEncontrada
	}
	_, _, autor := Ator(acesso)
	return s.store.CriarAgendamento(ctx, models.Agendamento{
		InstanciaID: inst.ID, ContaID: inst.ContaID, Tipo: req.Tipo,
		Numero: strings.TrimSpace(req.Numero), ChatJID: strings.TrimSpace(req.ChatJID),
		Mensagem: req.Mensagem, ArquivoURL: strings.TrimSpace(req.ArquivoURL), ArquivoBase64: limparDataURL(req.ArquivoBase64),
		NomeArquivo: req.NomeArquivo, MimeType: req.MimeType, EnviarEm: req.EnviarEm, Recorrencia: req.Recorrencia, CriadoPor: autor,
	})
}

func limparDataURL(texto string) string {
	if i := strings.Index(texto, ";base64,"); strings.HasPrefix(texto, "data:") && i > 0 {
		return texto[i+len(";base64,"):]
	}
	return texto
}

// filtroDoAcesso limita a listagem ao que o acesso pode ver.
func filtroAgendamentosDoAcesso(acesso models.AcessoDashboard, f models.FiltroAgendamentos) (models.FiltroAgendamentos, error) {
	switch {
	case acesso.TudoLiberado():
	case acesso.Tipo == "instancia":
		if f.InstanciaID != "" && f.InstanciaID != acesso.InstanciaID {
			return f, ErrAcessoNegado
		}
		f.InstanciaID = acesso.InstanciaID
	case acesso.Tipo == "usuario" && acesso.ContaID != "":
		f.SoContas, f.ContasIDs = true, []string{acesso.ContaID}
	default:
		return f, ErrAcessoNegado
	}
	return f, nil
}

func (s *AgendamentoService) Listar(ctx context.Context, acesso models.AcessoDashboard, f models.FiltroAgendamentos) ([]models.Agendamento, error) {
	f, err := filtroAgendamentosDoAcesso(acesso, f)
	if err != nil {
		return nil, err
	}
	return s.store.ListarAgendamentos(ctx, f)
}

// Buscar devolve o agendamento se o acesso enxerga a instancia dele.
func (s *AgendamentoService) Buscar(ctx context.Context, acesso models.AcessoDashboard, id string) (models.Agendamento, error) {
	a, err := s.store.BuscarAgendamento(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrAgendamentoNaoEncontrado) {
			return a, fmt.Errorf("%w: agendamento nao encontrado", ErrEntradaInvalida)
		}
		return a, err
	}
	switch {
	case acesso.TudoLiberado():
	case acesso.Tipo == "instancia" && acesso.InstanciaID == a.InstanciaID:
	case acesso.Tipo == "usuario" && acesso.ContaID != "" && acesso.ContaID == a.ContaID:
	default:
		return a, ErrAcessoNegado
	}
	return a, nil
}

func (s *AgendamentoService) Atualizar(ctx context.Context, acesso models.AcessoDashboard, id string, req models.AtualizarAgendamentoRequest) (models.Agendamento, error) {
	if _, err := s.Buscar(ctx, acesso, id); err != nil {
		return models.Agendamento{}, err
	}
	if req.Recorrencia != nil {
		r := strings.ToLower(strings.TrimSpace(*req.Recorrencia))
		if !recorrenciasValidas[r] {
			return models.Agendamento{}, fmt.Errorf("%w: recorrencia deve ser diaria, semanal, mensal ou vazia", ErrEntradaInvalida)
		}
		req.Recorrencia = &r
	}
	if req.EnviarEm != nil && req.EnviarEm.Before(s.agora().Add(-time.Minute)) {
		return models.Agendamento{}, fmt.Errorf("%w: enviar_em ja passou", ErrEntradaInvalida)
	}
	if err := s.store.AtualizarAgendamentoPendente(ctx, id, req.Mensagem, req.EnviarEm, req.Recorrencia); err != nil {
		if errors.Is(err, store.ErrAgendamentoNaoEncontrado) {
			return models.Agendamento{}, fmt.Errorf("%w: so da para alterar agendamentos pendentes", ErrEntradaInvalida)
		}
		return models.Agendamento{}, err
	}
	return s.store.BuscarAgendamento(ctx, id)
}

func (s *AgendamentoService) Cancelar(ctx context.Context, acesso models.AcessoDashboard, id string) (models.Agendamento, error) {
	a, err := s.Buscar(ctx, acesso, id)
	if err != nil {
		return a, err
	}
	if err := s.store.CancelarAgendamento(ctx, id); err != nil {
		if errors.Is(err, store.ErrAgendamentoNaoEncontrado) {
			return a, fmt.Errorf("%w: so da para cancelar agendamentos pendentes", ErrEntradaInvalida)
		}
		return a, err
	}
	return a, nil
}

// Iniciar roda a fila a cada 15 segundos.
func (s *AgendamentoService) Iniciar(ctx context.Context) {
	if s == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			s.Processar(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Processar envia o que ja venceu (exportado para os testes).
func (s *AgendamentoService) Processar(ctx context.Context) {
	agora := s.agora()
	if _, err := s.store.LiberarAgendamentosPresos(ctx, agora.Add(-5*time.Minute)); err != nil {
		fmt.Printf("agendamentos: %v\n", err)
	}
	vencidos, err := s.store.AgendamentosVencidos(ctx, agora, 50)
	if err != nil {
		fmt.Printf("agendamentos: %v\n", err)
		return
	}
	for _, a := range vencidos {
		inst, err := s.instancias.BuscarPorID(ctx, a.InstanciaID)
		if err != nil {
			continue
		}
		if inst.Tipo != models.TipoInstanciaMeta && s.donoSessao != nil && !s.donoSessao.PossuiRuntime(inst.ID) &&
			s.donoSessao.EnderecoOutroContainer(ctx, inst.ID) != "" {
			continue // a replica dona da sessao envia
		}
		ok, err := s.store.ReservarAgendamento(ctx, a.ID, s.no, agora)
		if err != nil || !ok {
			continue
		}
		s.enviar(ctx, a)
	}
}

func (s *AgendamentoService) enviar(ctx context.Context, a models.Agendamento) {
	ctx, cancelar := context.WithTimeout(ctx, 2*time.Minute)
	defer cancelar()
	resultado, err := s.enviarAgora(ctx, a)
	agora := s.agora()
	tentativas := a.Tentativas + 1
	if err == nil {
		_ = s.store.ConcluirAgendamento(ctx, a.ID, models.AgendamentoEnviado, "", resultado.MensagemID, tentativas, agora)
		s.repetir(ctx, a)
		return
	}
	if tentativas < maxTentativasAgendamento && agora.Sub(a.EnviarEm) < desistirDepoisDe {
		_ = s.store.AdiarAgendamento(ctx, a.ID, err.Error(), tentativas, agora.Add(esperaNovaTentativa))
		return
	}
	_ = s.store.ConcluirAgendamento(ctx, a.ID, models.AgendamentoFalhou, err.Error(), "", tentativas, agora)
	s.auditoria.Registrar(ctx, models.RegistroAuditoria{
		InstanciaID: a.InstanciaID, ContaID: a.ContaID, Tipo: models.AuditoriaAgenda, Acao: "agendamento.falhou",
		Descricao: "Mensagem agendada nao foi enviada para " + cmpOr(a.Numero, a.ChatJID) + ": " + err.Error(),
		Severidade: models.SeveridadeErro, AtorTipo: "sistema", AtorNome: "Sistema",
	})
	s.repetir(ctx, a)
}

func (s *AgendamentoService) enviarAgora(ctx context.Context, a models.Agendamento) (models.ResultadoEnvio, error) {
	if a.Tipo == "texto" {
		return s.mensagens.EnviarTexto(ctx, models.EnvioTextoRequest{Instancia: a.InstanciaID, Numero: a.Numero, ChatJID: a.ChatJID, Mensagem: a.Mensagem})
	}
	midia := models.EnvioMidiaRequest{
		Instancia: a.InstanciaID, Numero: a.Numero, ChatJID: a.ChatJID, ArquivoURL: a.ArquivoURL, ArquivoBase64: a.ArquivoBase64,
		Legenda: a.Mensagem, NomeArquivo: a.NomeArquivo, MimeType: a.MimeType,
	}
	switch a.Tipo {
	case "imagem":
		return s.mensagens.EnviarImagem(ctx, midia)
	case "audio":
		return s.mensagens.EnviarAudio(ctx, midia)
	default:
		return s.mensagens.EnviarDocumento(ctx, midia)
	}
}

// proximaData calcula a proxima ocorrencia de uma recorrencia.
func proximaData(base time.Time, recorrencia string, agora time.Time) time.Time {
	proxima := base
	for !proxima.After(agora) {
		switch recorrencia {
		case "diaria":
			proxima = proxima.AddDate(0, 0, 1)
		case "semanal":
			proxima = proxima.AddDate(0, 0, 7)
		case "mensal":
			proxima = proxima.AddDate(0, 1, 0)
		default:
			return time.Time{}
		}
	}
	return proxima
}

// repetir cria a proxima ocorrencia (o arquivo vai junto).
func (s *AgendamentoService) repetir(ctx context.Context, a models.Agendamento) {
	proxima := proximaData(a.EnviarEm, a.Recorrencia, s.agora())
	if proxima.IsZero() {
		return
	}
	nova := a
	nova.ID, nova.Status, nova.Tentativas, nova.Erro, nova.MensagemID = "", "", 0, "", ""
	nova.EnviarEm, nova.TentarEm, nova.CriadoEm, nova.EnviadoEm = proxima, proxima, time.Time{}, time.Time{}
	if _, err := s.store.CriarAgendamento(ctx, nova); err != nil {
		fmt.Printf("agendamentos: erro ao criar a proxima ocorrencia: %v\n", err)
	}
}
