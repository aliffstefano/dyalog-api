package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"

	"github.com/google/uuid"
)

// UsarMeta liga as instancias da API oficial. baseURL e o endereco publico da
// API, usado para montar a URL de webhook que vai no painel da Meta.
func (s *InstanciaService) UsarMeta(e *meta.Enviador, metaStore store.MetaStore, baseURL string) {
	s.meta = e
	s.metaStore = metaStore
	s.baseURL = strings.TrimRight(baseURL, "/")
}

// CriarComTipo cria instancia por QR code (tipo vazio ou "whatsapp") ou da API
// oficial ("meta"). Na API oficial, credenciais informadas ja sao validadas na
// Meta; sem elas a instancia nasce desconectada e e configurada depois.
func (s *InstanciaService) CriarComTipo(ctx context.Context, nome, tipo string, credenciais *models.ConfigurarMetaRequest) (models.Instancia, error) {
	tipo = strings.ToLower(strings.TrimSpace(tipo))
	switch tipo {
	case "", models.TipoInstanciaWhatsApp, "qr", "qrcode":
		return s.Criar(ctx, nome)
	case models.TipoInstanciaMeta, "oficial", "cloud":
	default:
		return models.Instancia{}, fmt.Errorf("%w: tipo deve ser whatsapp ou meta", ErrEntradaInvalida)
	}
	if s.meta == nil {
		return models.Instancia{}, fmt.Errorf("%w: API oficial da Meta nao habilitada neste servidor", ErrEntradaInvalida)
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return models.Instancia{}, ErrEntradaInvalida
	}
	// Valida antes de criar, para nao sobrar instancia com credencial errada.
	var info *meta.InfoNumero
	cred := models.CredenciaisMeta{VerifyToken: uuid.NewString()}
	if credenciais != nil {
		cred = mesclarCredenciaisMeta(cred, *credenciais)
		if cred.PhoneNumberID != "" || cred.AccessToken != "" {
			validada, err := s.validarCredenciaisMeta(ctx, cred)
			if err != nil {
				return models.Instancia{}, err
			}
			info = &validada
		}
	}
	agora := time.Now().UTC()
	instancia := models.Instancia{
		ID:           uuid.NewString(),
		Nome:         nome,
		Tipo:         models.TipoInstanciaMeta,
		Token:        uuid.NewString(),
		Status:       models.StatusInstanciaDesconectada,
		ProxyModo:    models.ProxyModoHerdar,
		Presenca:     models.PresencaIndisponivel,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	}
	if info != nil {
		instancia.Status = models.StatusInstanciaConectada
		cred.NumeroExibicao, cred.NomeVerificado = info.NumeroExibicao, info.NomeVerificado
	}
	instancia, err := s.store.Criar(ctx, instancia)
	if err != nil {
		return models.Instancia{}, err
	}
	cred.InstanciaID = instancia.ID
	if err := s.metaStore.SalvarCredenciaisMeta(ctx, cred); err != nil {
		_ = s.store.Excluir(ctx, instancia.ID)
		return models.Instancia{}, fmt.Errorf("%w: %v", ErrEntradaInvalida, err)
	}
	s.preencherPerfilMeta(ctx, &instancia)
	return instancia, nil
}

// ConfigurarMeta cadastra ou troca as credenciais. Campos vazios mantem o que
// ja estava salvo; a combinacao final e validada na Meta antes de gravar.
func (s *InstanciaService) ConfigurarMeta(ctx context.Context, id string, req models.ConfigurarMetaRequest) (map[string]any, error) {
	instancia, err := s.instanciaMeta(ctx, id)
	if err != nil {
		return nil, err
	}
	atual, err := s.metaStore.ObterCredenciaisMeta(ctx, id)
	if err != nil && !errors.Is(err, store.ErrCredenciaisMetaNaoEncontradas) {
		return nil, err
	}
	atual.InstanciaID = id
	if atual.VerifyToken == "" {
		atual.VerifyToken = uuid.NewString()
	}
	cred := mesclarCredenciaisMeta(atual, req)
	info, err := s.validarCredenciaisMeta(ctx, cred)
	if err != nil {
		return nil, err
	}
	cred.NumeroExibicao, cred.NomeVerificado = info.NumeroExibicao, info.NomeVerificado
	cred.AtualizadoEm = time.Now()
	if err := s.metaStore.SalvarCredenciaisMeta(ctx, cred); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEntradaInvalida, err)
	}
	if _, err := s.store.AtualizarStatus(ctx, id, models.StatusInstanciaConectada); err != nil {
		return nil, s.mapearErro(err)
	}
	instancia.Status = models.StatusInstanciaConectada
	return s.resumoMeta(instancia, cred, &info), nil
}

func mesclarCredenciaisMeta(atual models.CredenciaisMeta, req models.ConfigurarMetaRequest) models.CredenciaisMeta {
	trocar := func(destino *string, valor string) {
		if valor = strings.TrimSpace(valor); valor != "" {
			*destino = valor
		}
	}
	trocar(&atual.PhoneNumberID, req.PhoneNumberID)
	trocar(&atual.WABAID, req.WABAID)
	trocar(&atual.AccessToken, req.AccessToken)
	trocar(&atual.AppSecret, req.AppSecret)
	trocar(&atual.VerifyToken, req.VerifyToken)
	return atual
}

func (s *InstanciaService) validarCredenciaisMeta(ctx context.Context, cred models.CredenciaisMeta) (meta.InfoNumero, error) {
	if cred.PhoneNumberID == "" || cred.AccessToken == "" {
		return meta.InfoNumero{}, fmt.Errorf("%w: informe phone_number_id e access_token", ErrEntradaInvalida)
	}
	info, err := s.meta.Cliente().ValidarNumero(ctx, cred)
	if err != nil {
		return meta.InfoNumero{}, fmt.Errorf("%w: a Meta recusou as credenciais: %v", ErrEntradaInvalida, err)
	}
	return info, nil
}

func (s *InstanciaService) instanciaMeta(ctx context.Context, id string) (models.Instancia, error) {
	instancia, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	if !instancia.EhMeta() {
		return models.Instancia{}, fmt.Errorf("%w: esta instancia conecta por QR code, nao pela API oficial", ErrEntradaInvalida)
	}
	if s.meta == nil {
		return models.Instancia{}, fmt.Errorf("%w: API oficial da Meta nao habilitada neste servidor", ErrEntradaInvalida)
	}
	return instancia, nil
}

// URLWebhookMeta e o endereco que vai no painel da Meta (Webhooks > Callback
// URL) para a instancia receber mensagens.
func (s *InstanciaService) URLWebhookMeta(id string) string {
	return s.baseURL + "/webhook/meta/" + id
}

// resumoMeta e o que o painel e a API mostram da instancia oficial. Nunca
// devolve o access token nem o app secret.
func (s *InstanciaService) resumoMeta(instancia models.Instancia, cred models.CredenciaisMeta, info *meta.InfoNumero) map[string]any {
	resumo := map[string]any{
		"id":                   instancia.ID,
		"nome":                 instancia.Nome,
		"tipo":                 instancia.Tipo,
		"token":                instancia.Token,
		"status":               instancia.Status,
		"conectado":            instancia.Status == models.StatusInstanciaConectada,
		"phone_number_id":      cred.PhoneNumberID,
		"waba_id":              cred.WABAID,
		"numero_exibicao":      cred.NumeroExibicao,
		"nome_verificado":      cred.NomeVerificado,
		"access_token_salvo":   cred.AccessToken != "",
		"app_secret_salvo":     cred.AppSecret != "",
		"webhook_url":          s.URLWebhookMeta(instancia.ID),
		"webhook_verify_token": cred.VerifyToken,
		"atualizado_em":        cred.AtualizadoEm,
	}
	if info != nil {
		resumo["qualidade"] = info.Qualidade
		resumo["limite_mensagens"] = info.LimiteMensagens
		resumo["status_nome"] = info.StatusNome
	}
	return resumo
}

// StatusMeta consulta a Meta para confirmar que o token continua valido.
func (s *InstanciaService) StatusMeta(ctx context.Context, instancia models.Instancia) (map[string]any, error) {
	cred, err := s.metaStore.ObterCredenciaisMeta(ctx, instancia.ID)
	if errors.Is(err, store.ErrCredenciaisMetaNaoEncontradas) {
		return s.resumoMeta(instancia, models.CredenciaisMeta{}, nil), nil
	}
	if err != nil {
		return nil, err
	}
	var info *meta.InfoNumero
	if cred.AccessToken != "" && cred.PhoneNumberID != "" {
		validada, err := s.meta.Cliente().ValidarNumero(ctx, cred)
		status := models.StatusInstanciaConectada
		if err != nil {
			status = models.StatusInstanciaDesconectada
		} else {
			info = &validada
		}
		if status != instancia.Status {
			_, _ = s.store.AtualizarStatus(ctx, instancia.ID, status)
			instancia.Status = status
		}
		resumo := s.resumoMeta(instancia, cred, info)
		if err != nil {
			resumo["erro"] = err.Error()
		}
		return resumo, nil
	}
	return s.resumoMeta(instancia, cred, nil), nil
}

// preencherPerfilMeta mostra no painel o numero e o nome verificado da Meta,
// no mesmo lugar do perfil das instancias por QR code.
func (s *InstanciaService) preencherPerfilMeta(ctx context.Context, instancia *models.Instancia) {
	if s.metaStore == nil {
		return
	}
	cred, err := s.metaStore.ObterCredenciaisMeta(ctx, instancia.ID)
	if err != nil || cred.NumeroExibicao == "" {
		return
	}
	numero := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cred.NumeroExibicao)
	instancia.Perfil = &models.PerfilInstancia{
		Numero:      numero,
		Nome:        cred.NomeVerificado,
		NomeEmpresa: cred.NomeVerificado,
		Plataforma:  "cloud_api",
		Business:    true,
	}
}
