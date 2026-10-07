package service

import (
	"context"
	"fmt"

	"dyalog-api-go/internal/models"
)

// perfisSalvos le do banco o ultimo perfil de cada instancia. Sem PerfilStore
// (testes com store falso), devolve mapa vazio.
func (s *InstanciaService) perfisSalvos(ctx context.Context) map[string]models.PerfilInstancia {
	if s.perfilStore == nil {
		return nil
	}
	perfis, err := s.perfilStore.ListarPerfisInstancias(ctx)
	if err != nil {
		fmt.Printf("erro ao ler perfis salvos: %v\n", err)
		return nil
	}
	return perfis
}

// perfilWhatsApp devolve o perfil da instancia por QR code. So a replica dona
// da sessao le o perfil da memoria; ela grava no banco para as outras
// replicas mostrarem o mesmo. Sem isso o painel alternava entre foto/numero e
// iniciais/ID conforme o balanceador escolhia a replica.
func (s *InstanciaService) perfilWhatsApp(ctx context.Context, id string, salvos map[string]models.PerfilInstancia) *models.PerfilInstancia {
	salvo, temSalvo := salvos[id]
	if s.gerenciador.PossuiRuntime(id) {
		if atual := s.gerenciador.Perfil(id); atual.Numero != "" {
			// A foto e buscada em segundo plano; enquanto nao chega, mantem a
			// salva do mesmo numero.
			if atual.FotoURL == "" && temSalvo && salvo.Numero == atual.Numero {
				atual.FotoURL = salvo.FotoURL
			}
			if !temSalvo || salvo != atual {
				if s.perfilStore != nil {
					if err := s.perfilStore.SalvarPerfilInstancia(ctx, id, atual); err != nil {
						fmt.Printf("erro ao salvar perfil da instancia %s: %v\n", id, err)
					}
				}
			}
			return &atual
		}
	}
	if temSalvo {
		return &salvo
	}
	return nil
}

func (s *InstanciaService) esquecerPerfil(ctx context.Context, id string) {
	if s.perfilStore == nil {
		return
	}
	if err := s.perfilStore.ExcluirPerfilInstancia(ctx, id); err != nil {
		fmt.Printf("erro ao excluir perfil da instancia %s: %v\n", id, err)
	}
}
