package models

import "time"

const (
	DependenciaWhatsmeow = "go.mau.fi/whatsmeow"

	ModoAtualizacaoAviso   = "aviso"
	ModoAtualizacaoPreparo = "preparo"

	StatusAtualizacaoNaoVerificado    = "nao_verificado"
	StatusAtualizacaoAtualizado       = "atualizado"
	StatusAtualizacaoDisponivel       = "atualizacao_disponivel"
	StatusAtualizacaoVerificando      = "verificando"
	StatusAtualizacaoPreparoPlanejado = "preparo_planejado"
	StatusAtualizacaoPreparoBloqueado = "preparo_bloqueado"
	StatusAtualizacaoFalhaVerificacao = "falha_verificacao"
)

type StatusDependencia struct {
	Dependencia            string     `json:"dependencia"`
	VersaoEmUso            string     `json:"versao_em_uso"`
	UltimaVersaoDisponivel string     `json:"ultima_versao_disponivel,omitempty"`
	AtualizacaoDisponivel  bool       `json:"atualizacao_disponivel"`
	StatusAtualizacao      string     `json:"status_atualizacao"`
	ModoOperacao           string     `json:"modo_operacao"`
	UltimaVerificacaoEm    *time.Time `json:"ultima_verificacao_em,omitempty"`
	UltimaAplicacaoEm      *time.Time `json:"ultima_aplicacao_em,omitempty"`
	ArtefatoPreparoCaminho string     `json:"artefato_preparo_caminho,omitempty"`
	UltimoErro             string     `json:"ultimo_erro,omitempty"`
}

type VersaoSistema struct {
	AplicacaoNome         string     `json:"aplicacao_nome"`
	AplicacaoVersao       string     `json:"aplicacao_versao"`
	AplicacaoCommit       string     `json:"aplicacao_commit"`
	AplicacaoBuildDate    string     `json:"aplicacao_build_date"`
	WhatsmeowVersao       string     `json:"whatsmeow_versao"`
	UltimaVerificacaoEm   *time.Time `json:"ultima_verificacao_em,omitempty"`
	AtualizacaoDisponivel bool       `json:"atualizacao_disponivel"`
	StatusAtualizacao     string     `json:"status_atualizacao"`
	ModoAtualizacao       string     `json:"modo_atualizacao"`
	RebuildNecessario     bool       `json:"rebuild_necessario"`
	ReinicioNecessario    bool       `json:"reinicio_necessario"`
	ObservacaoOperacional string     `json:"observacao_operacional"`
}

type PreparacaoAtualizacao struct {
	Dependencia        string   `json:"dependencia"`
	VersaoAtual        string   `json:"versao_atual"`
	VersaoAlvo         string   `json:"versao_alvo"`
	BranchSugerida     string   `json:"branch_sugerida"`
	ArtefatoGeradoEm   string   `json:"artefato_gerado_em"`
	ArtefatoCaminho    string   `json:"artefato_caminho"`
	ComandosSugeridos  []string `json:"comandos_sugeridos"`
	PlanoRollingUpdate []string `json:"plano_rolling_update"`
	PlanoRollback      []string `json:"plano_rollback"`
	Observacao         string   `json:"observacao"`
}
