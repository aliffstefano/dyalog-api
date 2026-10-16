package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"

	"dyalog-api-go/internal/models"
)

// Aba Grupos: listar, ver participantes, link de convite e administrar.

// ErrGrupoInvalido: jid de grupo, acao ou numeros invalidos.
var ErrGrupoInvalido = errors.New("grupo invalido")

// JIDGrupo aceita "120363...@g.us" ou so o numero do grupo.
func JIDGrupo(valor string) (types.JID, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return types.EmptyJID, fmt.Errorf("%w: informe o grupo", ErrGrupoInvalido)
	}
	if !strings.Contains(valor, "@") {
		valor += "@" + types.GroupServer
	}
	jid, err := types.ParseJID(valor)
	if err != nil || jid.Server != types.GroupServer {
		return types.EmptyJID, fmt.Errorf("%w: %q nao e um grupo", ErrGrupoInvalido, valor)
	}
	return jid, nil
}

// souEu diz se o participante e a propria conta (pelo numero ou pelo LID).
func souEu(cli *whatsmeow.Client, p types.GroupParticipant) bool {
	if cli.Store.ID == nil {
		return false
	}
	eu, euLID := cli.Store.ID.User, cli.Store.LID.User
	for _, jid := range []types.JID{p.JID, p.PhoneNumber, p.LID} {
		if jid.User != "" && (jid.User == eu || (euLID != "" && jid.User == euLID)) {
			return true
		}
	}
	return false
}

func resumoGrupo(cli *whatsmeow.Client, g *types.GroupInfo) models.GrupoResumo {
	resumo := models.GrupoResumo{
		JID:            g.JID.String(),
		Nome:           g.Name,
		Descricao:      g.Topic,
		Participantes:  max(g.ParticipantCount, len(g.Participants)),
		SoAdminsEnviam: g.IsAnnounce,
		SoAdminsEditam: g.IsLocked,
		Comunidade:     g.IsParent,
		CriadoEm:       g.GroupCreated,
	}
	for _, p := range g.Participants {
		if souEu(cli, p) {
			resumo.SouAdmin = p.IsAdmin || p.IsSuperAdmin
			break
		}
	}
	return resumo
}

// GruposWhatsApp lista os grupos em que a conta participa.
func (g *GerenciadorInstancias) GruposWhatsApp(ctx context.Context, instanciaID string) ([]models.GrupoResumo, error) {
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	grupos, err := runtime.client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar grupos: %w", err)
	}
	lista := make([]models.GrupoResumo, 0, len(grupos))
	for _, grupo := range grupos {
		lista = append(lista, resumoGrupo(runtime.client, grupo))
	}
	sort.Slice(lista, func(i, j int) bool { return strings.ToLower(lista[i].Nome) < strings.ToLower(lista[j].Nome) })
	return lista, nil
}

// GrupoWhatsApp traz o grupo com os participantes (admins primeiro).
func (g *GerenciadorInstancias) GrupoWhatsApp(ctx context.Context, instanciaID, grupo string) (models.GrupoDetalhe, error) {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return models.GrupoDetalhe{}, err
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return models.GrupoDetalhe{}, err
	}
	cli := runtime.client
	info, err := cli.GetGroupInfo(ctx, jid)
	if err != nil {
		return models.GrupoDetalhe{}, fmt.Errorf("erro ao consultar o grupo: %w", err)
	}
	detalhe := models.GrupoDetalhe{GrupoResumo: resumoGrupo(cli, info), ListaParticipantes: make([]models.ParticipanteGrupo, 0, len(info.Participants))}
	if !info.OwnerPN.IsEmpty() {
		detalhe.Dono = info.OwnerPN.User
	} else if info.OwnerJID.Server == types.DefaultUserServer {
		detalhe.Dono = info.OwnerJID.User
	}
	for _, p := range info.Participants {
		item := models.ParticipanteGrupo{JID: p.JID.String(), Admin: p.IsAdmin || p.IsSuperAdmin, SuperAdmin: p.IsSuperAdmin, Eu: souEu(cli, p)}
		switch {
		case !p.PhoneNumber.IsEmpty():
			item.Numero = p.PhoneNumber.User
		case p.JID.Server == types.DefaultUserServer:
			item.Numero = p.JID.User
		case p.JID.Server == types.HiddenUserServer:
			if pn, err := cli.Store.LIDs.GetPNForLID(ctx, p.JID); err == nil && !pn.IsEmpty() {
				item.Numero = pn.User
			}
		}
		consulta := p.JID
		if item.Numero != "" {
			consulta = types.NewJID(item.Numero, types.DefaultUserServer)
		}
		if contato, err := cli.Store.Contacts.GetContact(ctx, consulta); err == nil {
			item.Nome = primeiroPreenchido(contato.FullName, contato.FirstName, contato.BusinessName, contato.PushName)
		}
		if item.Nome == "" {
			item.Nome = p.DisplayName
		}
		detalhe.ListaParticipantes = append(detalhe.ListaParticipantes, item)
	}
	sort.SliceStable(detalhe.ListaParticipantes, func(i, j int) bool {
		a, b := detalhe.ListaParticipantes[i], detalhe.ListaParticipantes[j]
		if a.Admin != b.Admin {
			return a.Admin
		}
		return strings.ToLower(a.Nome+a.Numero) < strings.ToLower(b.Nome+b.Numero)
	})
	return detalhe, nil
}

// jidsDosNumeros converte numeros em JIDs que existem no WhatsApp.
func (g *GerenciadorInstancias) jidsDosNumeros(ctx context.Context, cli *whatsmeow.Client, numeros []string) ([]types.JID, []models.ResultadoParticipante, error) {
	var (
		jids      []types.JID
		resultado []models.ResultadoParticipante
	)
	for _, numero := range numeros {
		digitos := apenasDigitos.ReplaceAllString(numero, "")
		if digitos == "" {
			continue
		}
		destinos, err := g.resolverDestinosEnvio(ctx, cli, "", digitos, false)
		if err != nil || len(destinos) == 0 {
			resultado = append(resultado, models.ResultadoParticipante{Numero: digitos, Erro: "numero invalido"})
			continue
		}
		jids = append(jids, destinos[0])
		resultado = append(resultado, models.ResultadoParticipante{Numero: digitos, JID: destinos[0].String()})
	}
	if len(jids) == 0 && len(resultado) == 0 {
		return nil, nil, fmt.Errorf("%w: informe ao menos um numero", ErrGrupoInvalido)
	}
	return jids, resultado, nil
}

// CriarGrupoWhatsApp cria o grupo com os participantes informados.
func (g *GerenciadorInstancias) CriarGrupoWhatsApp(ctx context.Context, instanciaID string, req models.CriarGrupoRequest) (models.GrupoDetalhe, error) {
	nome := strings.TrimSpace(req.Nome)
	if nome == "" || len([]rune(nome)) > 100 {
		return models.GrupoDetalhe{}, fmt.Errorf("%w: o nome do grupo deve ter de 1 a 100 caracteres", ErrGrupoInvalido)
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return models.GrupoDetalhe{}, err
	}
	jids, _, err := g.jidsDosNumeros(ctx, runtime.client, req.Participantes)
	if err != nil {
		return models.GrupoDetalhe{}, err
	}
	info, err := runtime.client.CreateGroup(ctx, whatsmeow.ReqCreateGroup{Name: nome, Participants: jids})
	if err != nil {
		return models.GrupoDetalhe{}, fmt.Errorf("erro ao criar o grupo: %w", err)
	}
	cacheContatos.esquecer(instanciaID)
	return g.GrupoWhatsApp(ctx, instanciaID, info.JID.String())
}

// AtualizarGrupoWhatsApp troca nome, descricao e permissoes.
func (g *GerenciadorInstancias) AtualizarGrupoWhatsApp(ctx context.Context, instanciaID, grupo string, req models.AtualizarGrupoRequest) error {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return err
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	cli := runtime.client
	if req.Nome != nil {
		if err := cli.SetGroupName(ctx, jid, strings.TrimSpace(*req.Nome)); err != nil {
			return fmt.Errorf("erro ao trocar o nome do grupo: %w", err)
		}
	}
	if req.Descricao != nil {
		if err := cli.SetGroupDescription(ctx, jid, strings.TrimSpace(*req.Descricao)); err != nil {
			return fmt.Errorf("erro ao trocar a descricao do grupo: %w", err)
		}
	}
	if req.SoAdminsEnviam != nil {
		if err := cli.SetGroupAnnounce(ctx, jid, *req.SoAdminsEnviam); err != nil {
			return fmt.Errorf("erro ao mudar quem envia mensagens: %w", err)
		}
	}
	if req.SoAdminsEditam != nil {
		if err := cli.SetGroupLocked(ctx, jid, *req.SoAdminsEditam); err != nil {
			return fmt.Errorf("erro ao mudar quem edita o grupo: %w", err)
		}
	}
	cacheContatos.esquecer(instanciaID)
	return nil
}

// FotoGrupoWhatsApp troca (ou remove, com nil) a foto do grupo.
func (g *GerenciadorInstancias) FotoGrupoWhatsApp(ctx context.Context, instanciaID, grupo string, jpeg []byte) error {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return err
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	if _, err := runtime.client.SetGroupPhoto(ctx, jid, jpeg); err != nil {
		return fmt.Errorf("erro ao trocar a foto do grupo: %w", err)
	}
	return nil
}

// LinkConviteGrupoWhatsApp devolve o link de convite; redefinir invalida o antigo.
func (g *GerenciadorInstancias) LinkConviteGrupoWhatsApp(ctx context.Context, instanciaID, grupo string, redefinir bool) (string, error) {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return "", err
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return "", err
	}
	link, err := runtime.client.GetGroupInviteLink(ctx, jid, redefinir)
	if err != nil {
		return "", fmt.Errorf("erro ao gerar o link de convite (so admins conseguem): %w", err)
	}
	return link, nil
}

var acoesParticipantes = map[string]whatsmeow.ParticipantChange{
	"adicionar": whatsmeow.ParticipantChangeAdd,
	"remover":   whatsmeow.ParticipantChangeRemove,
	"promover":  whatsmeow.ParticipantChangePromote,
	"rebaixar":  whatsmeow.ParticipantChangeDemote,
}

// errosParticipante traduz os codigos que o servidor devolve por participante.
var errosParticipante = map[int]string{
	401: "o contato bloqueou esta conta",
	403: "a privacidade do contato nao permite adicionar; mande o link de convite",
	404: "nao esta no grupo",
	408: "saiu do grupo recentemente; mande o link de convite",
	409: "ja esta no grupo",
	500: "o grupo esta cheio",
}

// ParticipantesGrupoWhatsApp adiciona, remove, promove ou rebaixa.
func (g *GerenciadorInstancias) ParticipantesGrupoWhatsApp(ctx context.Context, instanciaID, grupo string, req models.ParticipantesGrupoRequest) ([]models.ResultadoParticipante, error) {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return nil, err
	}
	acao, ok := acoesParticipantes[strings.ToLower(strings.TrimSpace(req.Acao))]
	if !ok {
		return nil, fmt.Errorf("%w: acao deve ser adicionar, remover, promover ou rebaixar", ErrGrupoInvalido)
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return nil, err
	}
	cli := runtime.client
	var (
		jids      []types.JID
		resultado []models.ResultadoParticipante
	)
	if acao == whatsmeow.ParticipantChangeAdd {
		jids, resultado, err = g.jidsDosNumeros(ctx, cli, req.Numeros)
		if err != nil {
			return nil, err
		}
	} else {
		// Quem ja esta no grupo pode estar so pelo LID: casa pelo numero ou pelo jid.
		info, err := cli.GetGroupInfo(ctx, jid)
		if err != nil {
			return nil, fmt.Errorf("erro ao consultar o grupo: %w", err)
		}
		for _, numero := range req.Numeros {
			alvo := strings.TrimSpace(numero)
			digitos := apenasDigitos.ReplaceAllString(alvo, "")
			achou := types.EmptyJID
			for _, p := range info.Participants {
				pn := p.PhoneNumber.User
				if pn == "" && p.JID.Server == types.DefaultUserServer {
					pn = p.JID.User
				}
				if pn == "" && p.JID.Server == types.HiddenUserServer {
					if resolvido, err := cli.Store.LIDs.GetPNForLID(ctx, p.JID); err == nil {
						pn = resolvido.User
					}
				}
				if p.JID.String() == alvo || (digitos != "" && (pn == digitos || p.JID.User == digitos)) {
					achou = p.JID
					break
				}
			}
			if achou.IsEmpty() {
				resultado = append(resultado, models.ResultadoParticipante{Numero: digitos, Erro: "nao esta no grupo"})
				continue
			}
			jids = append(jids, achou)
			resultado = append(resultado, models.ResultadoParticipante{Numero: digitos, JID: achou.String()})
		}
	}
	if len(jids) == 0 {
		return resultado, nil
	}
	respostas, err := cli.UpdateGroupParticipants(ctx, jid, jids, acao)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar participantes (so admins conseguem): %w", err)
	}
	erroPorJID := map[string]int{}
	for _, r := range respostas {
		for _, j := range []types.JID{r.JID, r.PhoneNumber, r.LID} {
			if !j.IsEmpty() {
				erroPorJID[j.String()] = r.Error
			}
		}
	}
	for i := range resultado {
		if resultado[i].JID == "" {
			continue
		}
		codigo, veio := erroPorJID[resultado[i].JID]
		switch {
		case !veio || codigo == 0 || codigo == 200:
			resultado[i].Ok = true
		default:
			resultado[i].Erro = errosParticipante[codigo]
			if resultado[i].Erro == "" {
				resultado[i].Erro = fmt.Sprintf("recusado pelo WhatsApp (codigo %d)", codigo)
			}
		}
	}
	return resultado, nil
}

// SairGrupoWhatsApp sai do grupo.
func (g *GerenciadorInstancias) SairGrupoWhatsApp(ctx context.Context, instanciaID, grupo string) error {
	jid, err := JIDGrupo(grupo)
	if err != nil {
		return err
	}
	runtime, err := g.obterRuntimeConectado(ctx, instanciaID)
	if err != nil {
		return err
	}
	if err := runtime.client.LeaveGroup(ctx, jid); err != nil {
		return fmt.Errorf("erro ao sair do grupo: %w", err)
	}
	cacheContatos.esquecer(instanciaID)
	return nil
}
