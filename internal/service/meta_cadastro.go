package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"dyalog-api-go/internal/models"
)

// AppMeta e o app da Meta do provedor, usado no cadastro incorporado.
type AppMeta struct {
	AppID     string
	AppSecret string
	ConfigID  string
	Versao    string
}

func (a AppMeta) habilitado() bool {
	return a.AppID != "" && a.AppSecret != "" && a.ConfigID != ""
}

// UsarAppMeta liga o cadastro incorporado (META_APP_ID, META_APP_SECRET e
// META_CONFIG_ID).
func (s *InstanciaService) UsarAppMeta(app AppMeta) {
	app.AppID, app.AppSecret, app.ConfigID = strings.TrimSpace(app.AppID), strings.TrimSpace(app.AppSecret), strings.TrimSpace(app.ConfigID)
	s.appMeta = app
}

// ConfigCadastroMeta e o que o painel precisa para abrir o login da Meta.
// O App Secret nunca sai daqui.
func (s *InstanciaService) ConfigCadastroMeta() map[string]any {
	return map[string]any{
		"habilitado": s.meta != nil && s.appMeta.habilitado(),
		"app_id":     s.appMeta.AppID,
		"config_id":  s.appMeta.ConfigID,
		"versao":     s.appMeta.Versao,
	}
}

// CadastroMetaRequest e o retorno do cadastro incorporado no navegador: o
// code do FB.login e os ids que a Meta manda na mensagem WA_EMBEDDED_SIGNUP.
type CadastroMetaRequest struct {
	Nome          string `json:"nome"`
	Code          string `json:"code"`
	WABAID        string `json:"waba_id"`
	PhoneNumberID string `json:"phone_number_id"`
}

// CadastrarViaMeta conclui o cadastro incorporado: troca o code pelo token da
// empresa, cria a instancia oficial ja validada, inscreve o app nos webhooks
// da conta e registra o numero na Cloud API. Os dois ultimos passos nao
// desfazem a instancia se falharem: voltam como avisos para resolver depois.
func (s *InstanciaService) CadastrarViaMeta(ctx context.Context, req CadastroMetaRequest) (models.Instancia, []string, error) {
	if s.meta == nil || !s.appMeta.habilitado() {
		return models.Instancia{}, nil, fmt.Errorf("%w: cadastro incorporado nao configurado (META_APP_ID, META_APP_SECRET e META_CONFIG_ID)", ErrEntradaInvalida)
	}
	req.Nome, req.Code = strings.TrimSpace(req.Nome), strings.TrimSpace(req.Code)
	req.WABAID, req.PhoneNumberID = strings.TrimSpace(req.WABAID), strings.TrimSpace(req.PhoneNumberID)
	if req.Nome == "" || req.Code == "" {
		return models.Instancia{}, nil, fmt.Errorf("%w: informe nome e code", ErrEntradaInvalida)
	}
	if req.PhoneNumberID == "" || req.WABAID == "" {
		return models.Instancia{}, nil, fmt.Errorf("%w: a Meta nao informou o numero escolhido; conclua o cadastro escolhendo um numero de telefone", ErrEntradaInvalida)
	}
	token, err := s.meta.Cliente().TrocarCode(ctx, s.appMeta.AppID, s.appMeta.AppSecret, req.Code)
	if err != nil {
		return models.Instancia{}, nil, fmt.Errorf("%w: a Meta recusou o code do cadastro (ele vale poucos segundos, tente de novo): %v", ErrEntradaInvalida, err)
	}
	instancia, err := s.CriarComTipo(ctx, req.Nome, models.TipoInstanciaMeta, &models.ConfigurarMetaRequest{
		PhoneNumberID: req.PhoneNumberID,
		WABAID:        req.WABAID,
		AccessToken:   token,
		AppSecret:     s.appMeta.AppSecret,
	})
	if err != nil {
		return models.Instancia{}, nil, err
	}

	var avisos []string
	cred, err := s.metaStore.ObterCredenciaisMeta(ctx, instancia.ID)
	if err != nil {
		return instancia, []string{"credenciais salvas, mas nao deu para concluir a configuracao: " + err.Error()}, nil
	}
	if err := s.meta.Cliente().AssinarWebhooksWABA(ctx, cred); err != nil {
		avisos = append(avisos, "inscrever o app nos webhooks da conta: "+err.Error())
	}
	pin, err := pinAleatorio()
	if err == nil {
		if err = s.meta.Cliente().RegistrarNumero(ctx, cred, pin); err == nil {
			cred.PinRegistro = pin
			if err := s.metaStore.SalvarCredenciaisMeta(ctx, cred); err != nil {
				avisos = append(avisos, "guardar o PIN do numero: "+err.Error())
			}
		}
	}
	if err != nil {
		// Numero que ja estava na Cloud API (ou com PIN proprio) recusa o
		// registro; costuma funcionar mesmo assim.
		avisos = append(avisos, "registrar o numero na Cloud API: "+err.Error())
	}
	return instancia, avisos, nil
}

// pinAleatorio gera o PIN de 6 digitos da verificacao em duas etapas.
func pinAleatorio() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
