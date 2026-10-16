package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"dyalog-api-go/internal/meta"
	"dyalog-api-go/internal/models"
	"dyalog-api-go/internal/whatsapp"
)

// Perfil, privacidade e contatos da conta de WhatsApp (instancias por QR
// code; a API oficial nao tem nada disso).

// tamanhoMaximoFotoPerfil limita a imagem recebida antes de recortar.
const tamanhoMaximoFotoPerfil = 10 << 20

func (s *InstanciaService) garantirInstanciaQR(ctx context.Context, id string) error {
	inst, err := s.store.BuscarPorID(ctx, strings.TrimSpace(id))
	if err != nil {
		return s.mapearErro(err)
	}
	if inst.Tipo == models.TipoInstanciaMeta {
		return fmt.Errorf("%w: perfil, privacidade e contatos so existem nas instancias por QR code", meta.ErrNaoSuportado)
	}
	return nil
}

func traduzirErroWhatsApp(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, whatsapp.ErrValorPrivacidade), errors.Is(err, whatsapp.ErrMidiaInvalida):
		return fmt.Errorf("%w: %s", ErrEntradaInvalida, err.Error())
	}
	return err
}

func (s *InstanciaService) PerfilWhatsApp(ctx context.Context, id string) (models.PerfilWhatsApp, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.PerfilWhatsApp{}, err
	}
	perfil, err := s.gerenciador.PerfilWhatsApp(ctx, id)
	return perfil, traduzirErroWhatsApp(err)
}

func (s *InstanciaService) AtualizarPerfilWhatsApp(ctx context.Context, id string, req models.AtualizarPerfilRequest) (models.PerfilWhatsApp, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.PerfilWhatsApp{}, err
	}
	if req.Nome == nil && req.Recado == nil {
		return models.PerfilWhatsApp{}, fmt.Errorf("%w: informe nome e/ou recado", ErrEntradaInvalida)
	}
	if req.Nome != nil {
		nome := strings.TrimSpace(*req.Nome)
		if nome == "" || len([]rune(nome)) > 25 {
			return models.PerfilWhatsApp{}, fmt.Errorf("%w: o nome deve ter de 1 a 25 caracteres", ErrEntradaInvalida)
		}
	}
	if req.Recado != nil && len([]rune(strings.TrimSpace(*req.Recado))) > 139 {
		return models.PerfilWhatsApp{}, fmt.Errorf("%w: o recado aceita no maximo 139 caracteres", ErrEntradaInvalida)
	}
	if err := s.gerenciador.AtualizarPerfilWhatsApp(ctx, id, req); err != nil {
		return models.PerfilWhatsApp{}, traduzirErroWhatsApp(err)
	}
	return s.PerfilWhatsApp(ctx, id)
}

// AtualizarFotoPerfilWhatsApp aceita base64 (com ou sem "data:...;base64,")
// ou URL. Imagem vazia remove a foto.
func (s *InstanciaService) AtualizarFotoPerfilWhatsApp(ctx context.Context, id string, req models.AtualizarFotoPerfilRequest, remover bool) error {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return err
	}
	if remover {
		return traduzirErroWhatsApp(s.gerenciador.AtualizarFotoPerfilWhatsApp(ctx, id, nil))
	}
	dados, err := lerImagemPerfil(ctx, req)
	if err != nil {
		return err
	}
	jpeg, err := whatsapp.PrepararFotoPerfil(dados)
	if err != nil {
		return traduzirErroWhatsApp(err)
	}
	return traduzirErroWhatsApp(s.gerenciador.AtualizarFotoPerfilWhatsApp(ctx, id, jpeg))
}

func lerImagemPerfil(ctx context.Context, req models.AtualizarFotoPerfilRequest) ([]byte, error) {
	if texto := strings.TrimSpace(req.Imagem); texto != "" {
		if i := strings.Index(texto, ";base64,"); strings.HasPrefix(texto, "data:") && i > 0 {
			texto = texto[i+len(";base64,"):]
		}
		dados, err := base64.StdEncoding.DecodeString(texto)
		if err != nil {
			return nil, fmt.Errorf("%w: imagem em base64 invalida", ErrEntradaInvalida)
		}
		if len(dados) > tamanhoMaximoFotoPerfil {
			return nil, fmt.Errorf("%w: imagem acima de 10 MB", ErrEntradaInvalida)
		}
		return dados, nil
	}
	url := strings.TrimSpace(req.URL)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("%w: envie imagem (base64) ou url", ErrEntradaInvalida)
	}
	ctx, cancelar := context.WithTimeout(ctx, 20*time.Second)
	defer cancelar()
	pedido, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: url invalida", ErrEntradaInvalida)
	}
	resp, err := http.DefaultClient.Do(pedido)
	if err != nil {
		return nil, fmt.Errorf("%w: nao foi possivel baixar a imagem", ErrEntradaInvalida)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: a url da imagem respondeu %d", ErrEntradaInvalida, resp.StatusCode)
	}
	dados, err := io.ReadAll(io.LimitReader(resp.Body, tamanhoMaximoFotoPerfil+1))
	if err != nil || len(dados) > tamanhoMaximoFotoPerfil {
		return nil, fmt.Errorf("%w: imagem acima de 10 MB ou download interrompido", ErrEntradaInvalida)
	}
	return dados, nil
}

func (s *InstanciaService) AtualizarPrivacidadeWhatsApp(ctx context.Context, id string, valores map[string]string) (models.PrivacidadeWhatsApp, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.PrivacidadeWhatsApp{}, err
	}
	if len(valores) == 0 {
		return models.PrivacidadeWhatsApp{}, fmt.Errorf("%w: informe ao menos um campo de privacidade", ErrEntradaInvalida)
	}
	privacidade, err := s.gerenciador.AtualizarPrivacidadeWhatsApp(ctx, id, valores)
	return privacidade, traduzirErroWhatsApp(err)
}

// FiltroContatos e a busca da tela de contatos.
type FiltroContatos struct {
	Busca     string
	Tipo      string // pessoal, grupo, bloqueado, salvo ou vazio
	Pagina    int
	PorPagina int // 0 = todos (exportacao)
	Atualizar bool
}

func (s *InstanciaService) ContatosWhatsApp(ctx context.Context, id string, f FiltroContatos) (models.ListaContatos, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return models.ListaContatos{}, err
	}
	todos, err := s.gerenciador.ContatosWhatsApp(ctx, id, f.Atualizar)
	if err != nil {
		return models.ListaContatos{}, traduzirErroWhatsApp(err)
	}
	return filtrarContatos(todos, f), nil
}

func filtrarContatos(todos []models.ContatoWhatsApp, f FiltroContatos) models.ListaContatos {
	contagem := map[string]int{"todos": len(todos)}
	busca := strings.ToLower(strings.TrimSpace(f.Busca))
	digitos := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, busca)
	filtrados := make([]models.ContatoWhatsApp, 0, len(todos))
	for _, c := range todos {
		contagem[c.Tipo]++
		if c.Bloqueado {
			contagem["bloqueado"]++
		}
		switch f.Tipo {
		case "", "todos":
		case "bloqueado":
			if !c.Bloqueado {
				continue
			}
		case "salvo":
			if !c.Salvo || c.Tipo != "pessoal" {
				continue
			}
		default:
			if c.Tipo != f.Tipo {
				continue
			}
		}
		if busca != "" {
			achou := strings.Contains(strings.ToLower(c.Nome), busca) || strings.Contains(strings.ToLower(c.NomePush), busca)
			if !achou && digitos != "" {
				achou = strings.Contains(c.Numero, digitos)
			}
			if !achou {
				continue
			}
		}
		filtrados = append(filtrados, c)
	}
	lista := models.ListaContatos{Total: len(filtrados), Contagem: contagem, Pagina: 1, PorPagina: f.PorPagina}
	if f.PorPagina <= 0 {
		lista.Contatos = filtrados
		lista.Paginas = 1
		return lista
	}
	lista.Paginas = max(1, int(math.Ceil(float64(len(filtrados))/float64(f.PorPagina))))
	lista.Pagina = min(max(f.Pagina, 1), lista.Paginas)
	inicio := (lista.Pagina - 1) * f.PorPagina
	lista.Contatos = filtrados[inicio:min(inicio+f.PorPagina, len(filtrados))]
	return lista
}

func (s *InstanciaService) VerificarNumerosWhatsApp(ctx context.Context, id string, numeros []string) ([]models.NumeroVerificado, error) {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return nil, err
	}
	if len(numeros) == 0 || len(numeros) > 50 {
		return nil, fmt.Errorf("%w: informe de 1 a 50 numeros", ErrEntradaInvalida)
	}
	resultado, err := s.gerenciador.VerificarNumerosWhatsApp(ctx, id, numeros)
	return resultado, traduzirErroWhatsApp(err)
}

func (s *InstanciaService) BloquearContatoWhatsApp(ctx context.Context, id string, req models.BloqueioRequest) error {
	if err := s.garantirInstanciaQR(ctx, id); err != nil {
		return err
	}
	if strings.TrimSpace(req.Numero) == "" && strings.TrimSpace(req.ChatJID) == "" {
		return fmt.Errorf("%w: informe numero ou chat_jid", ErrEntradaInvalida)
	}
	return traduzirErroWhatsApp(s.gerenciador.BloquearContatoWhatsApp(ctx, id, req))
}
