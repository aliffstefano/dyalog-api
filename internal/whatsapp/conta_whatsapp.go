package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"dyalog-api-go/internal/models"
)

// Perfil, privacidade e contatos da conta conectada (abas Perfil e Contatos
// do painel). Tudo roda na replica dona da sessao: as rotas sao
// /instancias/:id/..., que o proxy de replicas ja encaminha.

// ErrValorPrivacidade indica um campo ou valor de privacidade que o WhatsApp
// nao aceita.
var ErrValorPrivacidade = errors.New("valor de privacidade invalido")

// camposPrivacidade liga o nome usado na API ao tipo do WhatsApp e aos valores
// que o servidor aceita para ele.
var camposPrivacidade = []struct {
	campo   string
	tipo    types.PrivacySettingType
	validos []string
}{
	{"visto_por_ultimo", types.PrivacySettingTypeLastSeen, []string{"all", "contacts", "contact_blacklist", "none"}},
	{"foto_perfil", types.PrivacySettingTypeProfile, []string{"all", "contacts", "contact_blacklist", "none"}},
	{"recado", types.PrivacySettingTypeStatus, []string{"all", "contacts", "contact_blacklist", "none"}},
	{"online", types.PrivacySettingTypeOnline, []string{"all", "match_last_seen"}},
	{"chamadas", types.PrivacySettingTypeCallAdd, []string{"all", "known"}},
	{"grupos", types.PrivacySettingTypeGroupAdd, []string{"all", "contacts", "contact_blacklist", "none"}},
	{"confirmacao_leitura", types.PrivacySettingTypeReadReceipts, []string{"all", "none"}},
}

func opcoesPrivacidade() map[string][]string {
	opcoes := make(map[string][]string, len(camposPrivacidade))
	for _, c := range camposPrivacidade {
		opcoes[c.campo] = c.validos
	}
	return opcoes
}

func privacidadeParaModelo(p types.PrivacySettings) models.PrivacidadeWhatsApp {
	return models.PrivacidadeWhatsApp{
		VistoPorUltimo:     string(p.LastSeen),
		FotoPerfil:         string(p.Profile),
		Recado:             string(p.Status),
		Online:             string(p.Online),
		Chamadas:           string(p.CallAdd),
		Grupos:             string(p.GroupAdd),
		ConfirmacaoLeitura: string(p.ReadReceipts),
	}
}

// PerfilWhatsApp le identidade, recado, foto e privacidade da conta.
func (g *GerenciadorInstancias) PerfilWhatsApp(ctx context.Context, instanciaID string) (models.PerfilWhatsApp, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return models.PerfilWhatsApp{}, err
	}
	cli := runtime.client
	loja := cli.Store
	if loja.ID == nil {
		return models.PerfilWhatsApp{}, ErrInstanciaNaoConectada
	}
	proprio := loja.ID.ToNonAD()
	perfil := models.PerfilWhatsApp{
		Instancia:     instanciaID,
		Numero:        proprio.User,
		JID:           proprio.String(),
		Nome:          loja.PushName,
		NomeEmpresa:   loja.BusinessName,
		Plataforma:    loja.Platform,
		Business:      contaBusiness(cli),
		Privacidade:   privacidadeParaModelo(cli.GetPrivacySettings(ctx)),
		OpcoesValidas: opcoesPrivacidade(),
	}
	if !loja.LID.IsEmpty() {
		perfil.LID = loja.LID.ToNonAD().String()
	}
	if info, err := cli.GetUserInfo(ctx, []types.JID{proprio}); err == nil {
		perfil.Recado = info[proprio].Status
	}
	if foto, err := cli.GetProfilePictureInfo(ctx, proprio, &whatsmeow.GetProfilePictureParams{}); err == nil && foto != nil {
		perfil.FotoURL = foto.URL
	}
	return perfil, nil
}

// AtualizarPerfilWhatsApp troca o nome (push name) e/ou o recado.
func (g *GerenciadorInstancias) AtualizarPerfilWhatsApp(ctx context.Context, instanciaID string, req models.AtualizarPerfilRequest) error {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	cli := runtime.client
	if req.Nome != nil {
		nome := strings.TrimSpace(*req.Nome)
		if nome != cli.Store.PushName {
			if err := cli.SendAppState(ctx, appstate.BuildSettingPushName(nome)); err != nil {
				return fmt.Errorf("erro ao trocar o nome no WhatsApp: %w", err)
			}
			cli.Store.PushName = nome
			if err := cli.Store.Save(ctx); err != nil {
				return fmt.Errorf("nome trocado, mas nao foi possivel salvar na sessao: %w", err)
			}
			g.esquecerFotoPerfil(instanciaID)
		}
	}
	if req.Recado != nil {
		recado := strings.TrimSpace(*req.Recado)
		if err := cli.SetStatusMessage(ctx, types.SetStatusInput{Text: &recado}); err != nil {
			return fmt.Errorf("erro ao trocar o recado no WhatsApp: %w", err)
		}
	}
	return nil
}

// AtualizarFotoPerfilWhatsApp troca (ou remove, com imagem nil) a foto da
// conta. A imagem ja deve vir em JPEG quadrado (ver PrepararFotoPerfil).
func (g *GerenciadorInstancias) AtualizarFotoPerfilWhatsApp(ctx context.Context, instanciaID string, jpeg []byte) error {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	// Sem target o servidor aplica na propria conta (com target seria um grupo).
	if _, err := runtime.client.SetGroupPhoto(ctx, types.EmptyJID, jpeg); err != nil {
		return fmt.Errorf("erro ao trocar a foto no WhatsApp: %w", err)
	}
	g.esquecerFotoPerfil(instanciaID)
	return nil
}

func (g *GerenciadorInstancias) esquecerFotoPerfil(instanciaID string) {
	g.fotosPerfilMu.Lock()
	delete(g.fotosPerfil, instanciaID)
	g.fotosPerfilMu.Unlock()
}

// AtualizarPrivacidadeWhatsApp aplica os campos informados (nome da API ->
// valor do WhatsApp) e devolve a privacidade final.
func (g *GerenciadorInstancias) AtualizarPrivacidadeWhatsApp(ctx context.Context, instanciaID string, valores map[string]string) (models.PrivacidadeWhatsApp, error) {
	type mudanca struct {
		tipo  types.PrivacySettingType
		valor types.PrivacySetting
	}
	var mudancas []mudanca
	for campo, valor := range valores {
		achou := false
		for _, c := range camposPrivacidade {
			if c.campo != campo {
				continue
			}
			achou = true
			valor = strings.TrimSpace(valor)
			valido := false
			for _, v := range c.validos {
				valido = valido || v == valor
			}
			if !valido {
				return models.PrivacidadeWhatsApp{}, fmt.Errorf("%w: %s aceita %s", ErrValorPrivacidade, campo, strings.Join(c.validos, ", "))
			}
			mudancas = append(mudancas, mudanca{c.tipo, types.PrivacySetting(valor)})
		}
		if !achou {
			return models.PrivacidadeWhatsApp{}, fmt.Errorf("%w: campo %q desconhecido", ErrValorPrivacidade, campo)
		}
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return models.PrivacidadeWhatsApp{}, err
	}
	// Ordem fixa para o resultado nao depender da ordem do mapa.
	sort.Slice(mudancas, func(i, j int) bool { return mudancas[i].tipo < mudancas[j].tipo })
	for _, m := range mudancas {
		if _, err := runtime.client.SetPrivacySetting(ctx, m.tipo, m.valor); err != nil {
			return models.PrivacidadeWhatsApp{}, fmt.Errorf("erro ao mudar a privacidade (%s): %w", m.tipo, err)
		}
	}
	return privacidadeParaModelo(runtime.client.GetPrivacySettings(ctx)), nil
}

// ── Contatos ───────────────────────────────────────────────────────────────

// cacheGruposContatos guarda os grupos por alguns minutos: listar grupos e uma
// consulta ao servidor e a tela de contatos pagina varias vezes seguidas.
type cacheGruposContatos struct {
	mu    sync.Mutex
	itens map[string]gruposEmCache
}

type gruposEmCache struct {
	grupos    []models.ContatoWhatsApp
	bloqueios map[string]bool
	em        time.Time
}

const validadeCacheContatos = 3 * time.Minute

var cacheContatos = &cacheGruposContatos{itens: map[string]gruposEmCache{}}

func (c *cacheGruposContatos) esquecer(instanciaID string) {
	c.mu.Lock()
	delete(c.itens, instanciaID)
	c.mu.Unlock()
}

// ContatosWhatsApp devolve contatos salvos/conhecidos e grupos da instancia.
// atualizar=true ignora o cache de grupos e bloqueios.
func (g *GerenciadorInstancias) ContatosWhatsApp(ctx context.Context, instanciaID string, atualizar bool) ([]models.ContatoWhatsApp, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	cli := runtime.client
	todos, err := cli.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler contatos: %w", err)
	}

	cacheContatos.mu.Lock()
	emCache, temCache := cacheContatos.itens[instanciaID]
	cacheContatos.mu.Unlock()
	if atualizar || !temCache || time.Since(emCache.em) > validadeCacheContatos {
		emCache = gruposEmCache{bloqueios: map[string]bool{}, em: time.Now()}
		if grupos, err := cli.GetJoinedGroups(ctx); err == nil {
			for _, grupo := range grupos {
				emCache.grupos = append(emCache.grupos, models.ContatoWhatsApp{
					JID: grupo.JID.String(), Nome: grupo.Name, Tipo: "grupo", Salvo: true,
				})
			}
		}
		if lista, err := cli.GetBlocklist(ctx, ""); err == nil && lista != nil {
			for _, item := range lista.Items {
				if !item.PN.IsEmpty() {
					emCache.bloqueios[item.PN.ToNonAD().String()] = true
				}
				if !item.LID.IsEmpty() {
					emCache.bloqueios[item.LID.ToNonAD().String()] = true
				}
			}
		}
		cacheContatos.mu.Lock()
		cacheContatos.itens[instanciaID] = emCache
		cacheContatos.mu.Unlock()
	}

	contatos := make([]models.ContatoWhatsApp, 0, len(todos)+len(emCache.grupos))
	vistos := map[string]bool{}
	for jid, info := range todos {
		jid = jid.ToNonAD()
		contato := models.ContatoWhatsApp{
			JID:       jid.String(),
			Nome:      primeiroPreenchido(info.FullName, info.FirstName, info.BusinessName, info.PushName),
			NomePush:  info.PushName,
			Empresa:   info.BusinessName,
			Tipo:      "pessoal",
			Salvo:     info.FullName != "" || info.FirstName != "",
			Bloqueado: emCache.bloqueios[jid.String()],
		}
		switch jid.Server {
		case types.DefaultUserServer:
			contato.Numero = jid.User
		case types.HiddenUserServer:
			// Contato visto so pelo LID: se o numero e conhecido e ja existe
			// a entrada pelo numero, nao repete.
			if pn, err := cli.Store.LIDs.GetPNForLID(ctx, jid); err == nil && !pn.IsEmpty() {
				if _, existe := todos[pn]; existe {
					continue
				}
				contato.Numero = pn.User
				contato.JID = pn.String()
			}
		default:
			continue
		}
		if vistos[contato.JID] {
			continue
		}
		vistos[contato.JID] = true
		contatos = append(contatos, contato)
	}
	contatos = append(contatos, emCache.grupos...)
	sort.SliceStable(contatos, func(i, j int) bool {
		a, b := strings.ToLower(contatos[i].Nome), strings.ToLower(contatos[j].Nome)
		if (a == "") != (b == "") {
			return a != "" // sem nome vai para o fim
		}
		if a != b {
			return a < b
		}
		return contatos[i].JID < contatos[j].JID
	})
	return contatos, nil
}

// VerificarNumerosWhatsApp diz quais numeros tem WhatsApp (e se sao Business).
func (g *GerenciadorInstancias) VerificarNumerosWhatsApp(ctx context.Context, instanciaID string, numeros []string) ([]models.NumeroVerificado, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	consulta := make([]string, 0, len(numeros))
	for _, numero := range numeros {
		if digitos := apenasDigitos.ReplaceAllString(numero, ""); digitos != "" {
			consulta = append(consulta, "+"+digitos)
		}
	}
	if len(consulta) == 0 {
		return []models.NumeroVerificado{}, nil
	}
	respostas, err := runtime.client.IsOnWhatsApp(ctx, consulta)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar numeros no WhatsApp: %w", err)
	}
	resultado := make([]models.NumeroVerificado, 0, len(respostas))
	for _, r := range respostas {
		item := models.NumeroVerificado{
			Numero:     strings.TrimPrefix(r.Query, "+"),
			ExisteNoWA: r.IsIn,
			Business:   r.VerifiedName != nil,
		}
		if r.IsIn {
			item.JID = r.JID.String()
		}
		if r.VerifiedName != nil && r.VerifiedName.Details != nil {
			item.Nome = r.VerifiedName.Details.GetVerifiedName()
		}
		resultado = append(resultado, item)
	}
	return resultado, nil
}

// BloquearContatoWhatsApp bloqueia ou desbloqueia um contato.
func (g *GerenciadorInstancias) BloquearContatoWhatsApp(ctx context.Context, instanciaID string, req models.BloqueioRequest) error {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	jids, err := g.resolverDestinosEnvio(ctx, runtime.client, req.ChatJID, req.Numero, false)
	if err != nil {
		return err
	}
	if len(jids) == 0 {
		return fmt.Errorf("numero ou chat_jid invalido")
	}
	acao := events.BlocklistChangeActionUnblock
	if req.Bloquear {
		acao = events.BlocklistChangeActionBlock
	}
	if _, err := runtime.client.UpdateBlocklist(ctx, jids[0], acao, ""); err != nil {
		return fmt.Errorf("erro ao atualizar bloqueio no WhatsApp: %w", err)
	}
	cacheContatos.esquecer(instanciaID)
	return nil
}

func primeiroPreenchido(valores ...string) string {
	for _, v := range valores {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}
