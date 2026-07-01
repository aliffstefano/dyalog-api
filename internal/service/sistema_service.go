package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"dyalog-api-go/internal/config"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"
)

type SistemaService struct {
	cfg        *config.Config
	store      store.SistemaStore
	httpClient *http.Client
}

type latestModuleResponse struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

func NovoSistemaService(cfg *config.Config, store store.SistemaStore) *SistemaService {
	return &SistemaService{
		cfg:   cfg,
		store: store,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *SistemaService) IniciarMonitoramento(ctx context.Context) {
	_ = s.sincronizarEstado(ctx)
	if !s.cfg.AtualizacaoMonitoramento {
		return
	}

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				deve, err := s.deveVerificarAgora(ctx, time.Now())
				if err == nil && deve {
					_, _ = s.VerificarAtualizacao(ctx)
				}
			}
		}
	}()
}

func (s *SistemaService) ObterVersao(ctx context.Context) (models.VersaoSistema, error) {
	status, err := s.estadoAtual(ctx)
	if err != nil {
		return models.VersaoSistema{}, err
	}

	return models.VersaoSistema{
		AplicacaoNome:         s.cfg.NomeAplicacao,
		AplicacaoVersao:       s.cfg.VersaoAplicacao,
		AplicacaoCommit:       s.cfg.CommitAplicacao,
		AplicacaoBuildDate:    s.cfg.DataBuildAplicacao,
		WhatsmeowVersao:       status.VersaoEmUso,
		UltimaVerificacaoEm:   status.UltimaVerificacaoEm,
		AtualizacaoDisponivel: status.AtualizacaoDisponivel,
		StatusAtualizacao:     status.StatusAtualizacao,
		ModoAtualizacao:       status.ModoOperacao,
		RebuildNecessario:     true,
		ReinicioNecessario:    true,
		ObservacaoOperacional: "Atualizacao de dependencia Go exige novo build e novo processo. Nao ha hot reload do whatsmeow no binario em execucao.",
	}, nil
}

func (s *SistemaService) ObterAtualizacoes(ctx context.Context) (models.StatusDependencia, error) {
	return s.estadoAtual(ctx)
}

func (s *SistemaService) VerificarAtualizacao(ctx context.Context) (models.StatusDependencia, error) {
	status, err := s.estadoAtual(ctx)
	if err != nil {
		return models.StatusDependencia{}, err
	}

	agora := time.Now()
	status.StatusAtualizacao = models.StatusAtualizacaoVerificando
	status.UltimoErro = ""
	status.ModoOperacao = s.cfg.AtualizacaoModo
	_ = s.store.SalvarStatusDependencia(ctx, status)

	latest, err := s.buscarUltimaVersao(ctx)
	if err != nil {
		status.StatusAtualizacao = models.StatusAtualizacaoFalhaVerificacao
		status.UltimoErro = err.Error()
		status.UltimaVerificacaoEm = &agora
		_ = s.store.SalvarStatusDependencia(ctx, status)
		return status, nil
	}

	status.UltimaVerificacaoEm = &agora
	status.UltimaVersaoDisponivel = latest.Version
	status.AtualizacaoDisponivel = latest.Version != "" && latest.Version != status.VersaoEmUso
	if status.AtualizacaoDisponivel {
		status.StatusAtualizacao = models.StatusAtualizacaoDisponivel
	} else {
		status.StatusAtualizacao = models.StatusAtualizacaoAtualizado
	}
	status.UltimoErro = ""
	status.ModoOperacao = s.cfg.AtualizacaoModo

	if err := s.store.SalvarStatusDependencia(ctx, status); err != nil {
		return models.StatusDependencia{}, err
	}

	return status, nil
}

func (s *SistemaService) AplicarAtualizacao(ctx context.Context) (models.PreparacaoAtualizacao, error) {
	status, err := s.estadoAtual(ctx)
	if err != nil {
		return models.PreparacaoAtualizacao{}, err
	}

	if s.cfg.AtualizacaoModo != models.ModoAtualizacaoPreparo {
		return models.PreparacaoAtualizacao{}, ErrModoAtualizacaoInvalido
	}
	if !s.cfg.AtualizacaoAplicarHabilitado || s.cfg.Ambiente == "production" {
		status.StatusAtualizacao = models.StatusAtualizacaoPreparoBloqueado
		status.UltimoErro = "aplicacao de atualizacao bloqueada por configuracao de seguranca"
		_ = s.store.SalvarStatusDependencia(ctx, status)
		return models.PreparacaoAtualizacao{}, ErrAtualizacaoBloqueada
	}
	if !status.AtualizacaoDisponivel || status.UltimaVersaoDisponivel == "" {
		return models.PreparacaoAtualizacao{}, ErrNenhumaAtualizacao
	}

	agora := time.Now()
	baseDir := filepath.Join(s.cfg.AtualizacaoDiretorioArtefatos, "whatsmeow", agora.Format("20060102-150405"))
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return models.PreparacaoAtualizacao{}, fmt.Errorf("erro ao criar diretorio de artefato: %w", err)
	}

	branch := fmt.Sprintf("chore/whatsmeow-%s", strings.TrimPrefix(status.UltimaVersaoDisponivel, "v"))
	plano := models.PreparacaoAtualizacao{
		Dependencia:      status.Dependencia,
		VersaoAtual:      status.VersaoEmUso,
		VersaoAlvo:       status.UltimaVersaoDisponivel,
		BranchSugerida:   branch,
		ArtefatoGeradoEm: agora.Format(time.RFC3339),
		ArtefatoCaminho:  filepath.Join(baseDir, "plano-atualizacao.json"),
		ComandosSugeridos: []string{
			"git checkout -b " + branch,
			"go get go.mau.fi/whatsmeow@" + status.UltimaVersaoDisponivel,
			"go mod tidy",
			"go test ./...",
			"go build ./cmd/api",
		},
		PlanoRollingUpdate: []string{
			"Gerar novo binario ou imagem com a dependencia atualizada.",
			"Publicar em ambiente de homologacao e executar smoke tests.",
			"Fazer rolling update gradual por instancias/processos.",
			"Monitorar healthcheck, conexoes WhatsApp e erros de envio durante a troca.",
		},
		PlanoRollback: []string{
			"Manter o artefato anterior disponivel para reversao imediata.",
			"Se houver regressao, restaurar a imagem/binario anterior e reiniciar os processos.",
			"Revalidar /api/v1/saude e /api/v1/sistema/versao apos rollback.",
		},
		Observacao: "Preparo controlado: o sistema gera o artefato de plano. A troca efetiva exige rebuild e novo deploy; nao ha hot reload da biblioteca em producao.",
	}

	conteudo, err := json.MarshalIndent(plano, "", "  ")
	if err != nil {
		return models.PreparacaoAtualizacao{}, fmt.Errorf("erro ao serializar artefato: %w", err)
	}
	if err := os.WriteFile(plano.ArtefatoCaminho, conteudo, 0o644); err != nil {
		return models.PreparacaoAtualizacao{}, fmt.Errorf("erro ao gravar artefato: %w", err)
	}

	status.StatusAtualizacao = models.StatusAtualizacaoPreparoPlanejado
	status.ArtefatoPreparoCaminho = plano.ArtefatoCaminho
	status.UltimaAplicacaoEm = &agora
	status.UltimoErro = ""
	if err := s.store.SalvarStatusDependencia(ctx, status); err != nil {
		return models.PreparacaoAtualizacao{}, err
	}

	return plano, nil
}

func (s *SistemaService) AutorizarAplicacao(token string) error {
	if s.cfg.AtualizacaoAplicarToken == "" {
		return ErrAutorizacaoAtualizacao
	}
	if token != s.cfg.AtualizacaoAplicarToken {
		return ErrAutorizacaoAtualizacao
	}
	return nil
}

func (s *SistemaService) deveVerificarAgora(ctx context.Context, agora time.Time) (bool, error) {
	if !dentroDaJanela(agora, s.cfg.AtualizacaoJanelaInicio, s.cfg.AtualizacaoJanelaFim) {
		return false, nil
	}
	status, err := s.estadoAtual(ctx)
	if err != nil {
		return false, err
	}
	if status.UltimaVerificacaoEm == nil {
		return true, nil
	}
	if agora.Sub(*status.UltimaVerificacaoEm) >= time.Duration(s.cfg.AtualizacaoIntervaloMinutos)*time.Minute {
		return true, nil
	}
	return false, nil
}

func (s *SistemaService) estadoAtual(ctx context.Context) (models.StatusDependencia, error) {
	if err := s.sincronizarEstado(ctx); err != nil {
		return models.StatusDependencia{}, err
	}
	status, err := s.store.ObterStatusDependencia(ctx, models.DependenciaWhatsmeow)
	if err == nil {
		return status, nil
	}
	if err == store.ErrDependenciaNaoEncontrada {
		return models.StatusDependencia{}, ErrDependenciaNaoEncontrada
	}
	return models.StatusDependencia{}, err
}

func (s *SistemaService) sincronizarEstado(ctx context.Context) error {
	versaoAtual := versaoWhatsmeowAtual()
	status, err := s.store.ObterStatusDependencia(ctx, models.DependenciaWhatsmeow)
	if err != nil {
		if err != store.ErrDependenciaNaoEncontrada {
			return err
		}
		status = models.StatusDependencia{
			Dependencia:           models.DependenciaWhatsmeow,
			VersaoEmUso:           versaoAtual,
			StatusAtualizacao:     models.StatusAtualizacaoNaoVerificado,
			ModoOperacao:          s.cfg.AtualizacaoModo,
			AtualizacaoDisponivel: false,
		}
		return s.store.SalvarStatusDependencia(ctx, status)
	}

	status.VersaoEmUso = versaoAtual
	status.ModoOperacao = s.cfg.AtualizacaoModo
	return s.store.SalvarStatusDependencia(ctx, status)
}

func (s *SistemaService) buscarUltimaVersao(ctx context.Context) (latestModuleResponse, error) {
	url := strings.TrimRight(s.cfg.AtualizacaoProxyURL, "/") + "/go.mau.fi/whatsmeow/@latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return latestModuleResponse{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return latestModuleResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return latestModuleResponse{}, fmt.Errorf("proxy retornou status %d", resp.StatusCode)
	}

	var latest latestModuleResponse
	if err := json.NewDecoder(resp.Body).Decode(&latest); err != nil {
		return latestModuleResponse{}, err
	}
	return latest, nil
}

func versaoWhatsmeowAtual() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "indisponivel"
	}
	for _, dep := range info.Deps {
		if dep.Path == models.DependenciaWhatsmeow {
			if dep.Replace != nil && dep.Replace.Version != "" {
				return dep.Replace.Version
			}
			if dep.Version != "" {
				return dep.Version
			}
		}
	}
	return "nao_informada"
}

func dentroDaJanela(agora time.Time, inicioStr, fimStr string) bool {
	inicio, errInicio := time.Parse("15:04", inicioStr)
	fim, errFim := time.Parse("15:04", fimStr)
	if errInicio != nil || errFim != nil {
		return true
	}

	minutosAgora := agora.Hour()*60 + agora.Minute()
	minutosInicio := inicio.Hour()*60 + inicio.Minute()
	minutosFim := fim.Hour()*60 + fim.Minute()

	if minutosInicio <= minutosFim {
		return minutosAgora >= minutosInicio && minutosAgora <= minutosFim
	}
	return minutosAgora >= minutosInicio || minutosAgora <= minutosFim
}
