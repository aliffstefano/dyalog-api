package whatsapp

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"

	"dyalog-api-go/internal/models"
)

type fotoPerfil struct {
	url       string
	buscadaEm time.Time
	buscando  bool
}

// validadeFotoPerfil e por quanto tempo a foto e reaproveitada antes de buscar
// de novo (o link do WhatsApp expira em algumas horas).
const validadeFotoPerfil = 30 * time.Minute

// Perfil devolve numero, nome e se a conta e Business, lidos da sessao salva.
// A foto vem do cache e, se estiver velha, e renovada em segundo plano para nao
// atrasar quem consulta o status.
func (g *GerenciadorInstancias) Perfil(instanciaID string) models.PerfilInstancia {
	g.mu.RLock()
	runtime := g.runtimes[instanciaID]
	g.mu.RUnlock()
	if runtime == nil || runtime.client == nil || runtime.client.Store == nil || runtime.client.Store.ID == nil {
		return models.PerfilInstancia{}
	}
	loja := runtime.client.Store
	perfil := models.PerfilInstancia{
		Numero:      loja.ID.User,
		Nome:        loja.PushName,
		NomeEmpresa: loja.BusinessName,
		Plataforma:  loja.Platform,
		// smba / smbi = app WhatsApp Business no Android / iPhone.
		Business: strings.HasPrefix(loja.Platform, "smb") || loja.BusinessName != "",
	}

	g.fotosPerfilMu.Lock()
	foto := g.fotosPerfil[instanciaID]
	buscar := !foto.buscando && time.Since(foto.buscadaEm) > validadeFotoPerfil && runtime.client.IsLoggedIn()
	if buscar {
		foto.buscando = true
		g.fotosPerfil[instanciaID] = foto
	}
	g.fotosPerfilMu.Unlock()
	perfil.FotoURL = foto.url

	if buscar {
		go g.buscarFotoPerfil(instanciaID, runtime.client)
	}
	return perfil
}

func (g *GerenciadorInstancias) buscarFotoPerfil(instanciaID string, client *whatsmeow.Client) {
	ctx, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	url := ""
	if id := client.Store.ID; id != nil {
		if info, err := client.GetProfilePictureInfo(ctx, id.ToNonAD(), &whatsmeow.GetProfilePictureParams{}); err == nil && info != nil {
			url = info.URL
		}
	}
	g.fotosPerfilMu.Lock()
	g.fotosPerfil[instanciaID] = fotoPerfil{url: url, buscadaEm: time.Now()}
	g.fotosPerfilMu.Unlock()
}
