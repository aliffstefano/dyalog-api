package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/store"

	"github.com/google/uuid"
)

// AppInstagram e o app da Meta do provedor no produto "API do Instagram com
// login do Instagram" (ID e chave secreta do app do Instagram).
type AppInstagram struct {
	AppID     string
	AppSecret string
}

func (a AppInstagram) habilitado() bool { return a.AppID != "" && a.AppSecret != "" }

// UsarInstagram liga as instancias do Instagram.
func (s *InstanciaService) UsarInstagram(e *meta.EnviadorInstagram, igStore store.InstagramStore, app AppInstagram) {
	s.instagram, s.igStore = e, igStore
	s.appInstagram = AppInstagram{AppID: strings.TrimSpace(app.AppID), AppSecret: strings.TrimSpace(app.AppSecret)}
}

// RedirectInstagram e a URL de retorno do login (cadastrar igual no app da
// Meta, em "Configure o login da empresa no Instagram").
func (s *InstanciaService) RedirectInstagram() string { return s.baseURL + "/instagram/retorno" }

// URLWebhookInstagram e a URL de callback unica do app para o Instagram.
func (s *InstanciaService) URLWebhookInstagram() string { return s.baseURL + "/webhook/instagram" }

// ConfigCadastroInstagram e o que o painel precisa para abrir o login do
// Instagram. O painel acrescenta o state.
func (s *InstanciaService) ConfigCadastroInstagram() map[string]any {
	login := "https://www.instagram.com/oauth/authorize?" + url.Values{
		"enable_fb_login": {"0"},
		"force_reauth":    {"true"},
		"client_id":       {s.appInstagram.AppID},
		"redirect_uri":    {s.RedirectInstagram()},
		"response_type":   {"code"},
		"scope":           {"instagram_business_basic,instagram_business_manage_messages"},
	}.Encode()
	return map[string]any{
		"habilitado":   s.instagram != nil && s.appInstagram.habilitado(),
		"app_id":       s.appInstagram.AppID,
		"redirect_uri": s.RedirectInstagram(),
		"url_login":    login,
	}
}

// CriarInstagram cria a instancia ja com a conta do token (gerado no painel
// da Meta em "Gere tokens de acesso" ou vindo do login).
func (s *InstanciaService) CriarInstagram(ctx context.Context, nome, token string) (models.Instancia, []string, error) {
	if s.instagram == nil {
		return models.Instancia{}, nil, fmt.Errorf("%w: Instagram nao habilitado neste servidor", ErrEntradaInvalida)
	}
	nome, token = strings.TrimSpace(nome), strings.TrimSpace(token)
	if nome == "" || token == "" {
		return models.Instancia{}, nil, fmt.Errorf("%w: informe nome e access_token do Instagram", ErrEntradaInvalida)
	}
	// Valida antes de criar, para nao sobrar instancia com token errado.
	perfil, err := s.instagram.Cliente().Perfil(ctx, token)
	if err != nil {
		return models.Instancia{}, nil, fmt.Errorf("%w: o Instagram recusou o token: %v", ErrEntradaInvalida, err)
	}
	agora := time.Now().UTC()
	instancia, err := s.store.Criar(ctx, models.Instancia{
		ID:           uuid.NewString(),
		Nome:         nome,
		Tipo:         models.TipoInstanciaInstagram,
		Token:        uuid.NewString(),
		Status:       models.StatusInstanciaConectada,
		ProxyModo:    models.ProxyModoHerdar,
		Presenca:     models.PresencaIndisponivel,
		CriadoEm:     agora,
		AtualizadoEm: agora,
	})
	if err != nil {
		return models.Instancia{}, nil, err
	}
	cred := credInstagram(instancia.ID, token, perfil)
	if err := s.igStore.SalvarCredenciaisInstagram(ctx, cred); err != nil {
		_ = s.store.Excluir(ctx, instancia.ID)
		return models.Instancia{}, nil, fmt.Errorf("%w: %v", ErrEntradaInvalida, err)
	}
	var avisos []string
	if err := s.instagram.Cliente().AssinarWebhooks(ctx, token); err != nil {
		avisos = append(avisos, "inscrever o app nos webhooks da conta: "+err.Error())
	}
	s.preencherPerfilInstagram(ctx, &instancia)
	return instancia, avisos, nil
}

// CadastrarViaInstagram conclui o login do Instagram: troca o code pelo token
// de 60 dias e cria a instancia.
func (s *InstanciaService) CadastrarViaInstagram(ctx context.Context, nome, code string) (models.Instancia, []string, error) {
	if s.instagram == nil || !s.appInstagram.habilitado() {
		return models.Instancia{}, nil, fmt.Errorf("%w: login do Instagram nao configurado (INSTAGRAM_APP_ID e INSTAGRAM_APP_SECRET)", ErrEntradaInvalida)
	}
	if strings.TrimSpace(nome) == "" || strings.TrimSpace(code) == "" {
		return models.Instancia{}, nil, fmt.Errorf("%w: informe nome e code", ErrEntradaInvalida)
	}
	token, err := s.instagram.Cliente().TrocarCode(ctx, s.appInstagram.AppID, s.appInstagram.AppSecret, s.RedirectInstagram(), strings.TrimSpace(code))
	if err != nil {
		return models.Instancia{}, nil, fmt.Errorf("%w: o Instagram recusou o code do login (ele vale uma vez so, tente de novo): %v", ErrEntradaInvalida, err)
	}
	return s.CriarInstagram(ctx, nome, token)
}

// ConfigurarInstagram troca o token (vazio revalida o salvo) e inscreve de
// novo os webhooks.
func (s *InstanciaService) ConfigurarInstagram(ctx context.Context, id string, req models.ConfigurarInstagramRequest) (map[string]any, error) {
	instancia, err := s.instanciaInstagram(ctx, id)
	if err != nil {
		return nil, err
	}
	atual, err := s.igStore.ObterCredenciaisInstagram(ctx, id)
	if err != nil && !errors.Is(err, store.ErrCredenciaisInstagramNaoEncontradas) {
		return nil, err
	}
	token := strings.TrimSpace(req.AccessToken)
	if token == "" {
		token = atual.AccessToken
	}
	if token == "" {
		return nil, fmt.Errorf("%w: informe o access_token do Instagram", ErrEntradaInvalida)
	}
	perfil, err := s.instagram.Cliente().Perfil(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("%w: o Instagram recusou o token: %v", ErrEntradaInvalida, err)
	}
	cred := credInstagram(id, token, perfil)
	if err := s.igStore.SalvarCredenciaisInstagram(ctx, cred); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEntradaInvalida, err)
	}
	if _, err := s.store.AtualizarStatus(ctx, id, models.StatusInstanciaConectada); err != nil {
		return nil, s.mapearErro(err)
	}
	instancia.Status = models.StatusInstanciaConectada
	resumo := s.resumoInstagram(instancia, cred)
	if err := s.instagram.Cliente().AssinarWebhooks(ctx, token); err != nil {
		resumo["webhook_assinado"], resumo["webhook_assinatura_erro"] = false, err.Error()
	} else {
		resumo["webhook_assinado"] = true
	}
	return resumo, nil
}

func credInstagram(instanciaID, token string, p meta.PerfilInstagram) models.CredenciaisInstagram {
	return models.CredenciaisInstagram{
		InstanciaID: instanciaID, ContaID: p.ContaID, Usuario: p.Usuario, Nome: p.Nome,
		FotoURL: p.FotoURL, AccessToken: token, AtualizadoEm: time.Now(),
	}
}

func (s *InstanciaService) instanciaInstagram(ctx context.Context, id string) (models.Instancia, error) {
	instancia, err := s.store.BuscarPorID(ctx, id)
	if err != nil {
		return models.Instancia{}, s.mapearErro(err)
	}
	if !instancia.EhInstagram() {
		return models.Instancia{}, fmt.Errorf("%w: esta instancia nao e do Instagram", ErrEntradaInvalida)
	}
	if s.instagram == nil {
		return models.Instancia{}, fmt.Errorf("%w: Instagram nao habilitado neste servidor", ErrEntradaInvalida)
	}
	return instancia, nil
}

// resumoInstagram e o que o painel e a API mostram. Nunca devolve o token.
func (s *InstanciaService) resumoInstagram(instancia models.Instancia, cred models.CredenciaisInstagram) map[string]any {
	return map[string]any{
		"id":                 instancia.ID,
		"nome":               instancia.Nome,
		"tipo":               instancia.Tipo,
		"token":              instancia.Token,
		"status":             instancia.Status,
		"conectado":          instancia.Status == models.StatusInstanciaConectada,
		"instagram_conta_id": cred.ContaID,
		"instagram_usuario":  cred.Usuario,
		"instagram_nome":     cred.Nome,
		"access_token_salvo": cred.AccessToken != "",
		"webhook_url":        s.URLWebhookInstagram(),
		"atualizado_em":      cred.AtualizadoEm,
	}
}

// StatusInstagram confere na Meta se o token continua valido.
func (s *InstanciaService) StatusInstagram(ctx context.Context, instancia models.Instancia) map[string]any {
	cred, err := s.igStore.ObterCredenciaisInstagram(ctx, instancia.ID)
	if err != nil || cred.AccessToken == "" {
		return s.resumoInstagram(instancia, cred)
	}
	status := models.StatusInstanciaConectada
	_, erroPerfil := s.instagram.Cliente().Perfil(ctx, cred.AccessToken)
	if erroPerfil != nil {
		status = models.StatusInstanciaDesconectada
	}
	if status != instancia.Status {
		_, _ = s.store.AtualizarStatus(ctx, instancia.ID, status)
		instancia.Status = status
	}
	resumo := s.resumoInstagram(instancia, cred)
	if erroPerfil != nil {
		resumo["erro"] = erroPerfil.Error()
	}
	return resumo
}

// preencherPerfilInstagram mostra o @ e a foto da conta no lugar do perfil
// do WhatsApp.
func (s *InstanciaService) preencherPerfilInstagram(ctx context.Context, instancia *models.Instancia) {
	if s.igStore == nil {
		return
	}
	cred, err := s.igStore.ObterCredenciaisInstagram(ctx, instancia.ID)
	if err != nil || cred.ContaID == "" {
		return
	}
	instancia.Perfil = &models.PerfilInstancia{
		Numero:      "@" + cred.Usuario,
		Nome:        cred.Nome,
		NomeEmpresa: cred.Nome,
		Plataforma:  "instagram",
		Business:    true,
		FotoURL:     cred.FotoURL,
	}
}

// IniciarRenovacaoInstagram renova uma vez por dia os tokens com mais de 7
// dias (vencem em 60). Varias replicas renovando juntas nao estraga nada.
func (s *InstanciaService) IniciarRenovacaoInstagram(ctx context.Context) {
	if s.instagram == nil || s.igStore == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			s.renovarTokensInstagram(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *InstanciaService) renovarTokensInstagram(ctx context.Context) {
	lista, err := s.igStore.ListarCredenciaisInstagramAntigas(ctx, time.Now().AddDate(0, 0, -7))
	if err != nil {
		fmt.Printf("instagram: %v\n", err)
		return
	}
	for _, cred := range lista {
		token, err := s.instagram.Cliente().RenovarToken(ctx, cred.AccessToken)
		if err != nil {
			fmt.Printf("instagram: erro ao renovar o token da instancia %s: %v\n", cred.InstanciaID, err)
			continue
		}
		cred.AccessToken, cred.AtualizadoEm = token, time.Now()
		if err := s.igStore.SalvarCredenciaisInstagram(ctx, cred); err != nil {
			fmt.Printf("instagram: erro ao guardar o token renovado da instancia %s: %v\n", cred.InstanciaID, err)
		}
	}
}
