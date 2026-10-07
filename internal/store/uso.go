package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
)

// UsoStore guarda o historico de envios usado nas metricas de uso do painel.
type UsoStore interface {
	RegistrarEnvio(ctx context.Context, envio models.EnvioRegistro) error
	ResumoUso(ctx context.Context, instanciaID string, agora time.Time, dias int) (models.ResumoUso, error)
	LimparEnviosAntigos(ctx context.Context, antesDe time.Time) (int64, error)
	UsoHojePorInstancia(ctx context.Context, agora time.Time) ([]models.UsoInstanciaHoje, error)
}

// rajadaMinima e quantas mensagens para o mesmo contato no mesmo minuto contam
// como rajada.
const rajadaMinima = 5

// RegistrarEnvio grava um envio. Uma mensagem enviada conta como contato novo
// quando o chat nao e grupo, nunca recebeu envio registrado desta instancia e
// nunca mandou mensagem para ela.
func (s *SQLStore) RegistrarEnvio(ctx context.Context, envio models.EnvioRegistro) error {
	if envio.CriadoEm.IsZero() {
		envio.CriadoEm = time.Now()
	}
	novo := false
	if envio.Resultado == models.EnvioResultadoEnviada && envio.ChatJID != "" && !strings.HasSuffix(envio.ChatJID, "@g.us") {
		var conhecido int
		err := s.db.QueryRowContext(ctx, s.q(`
SELECT CASE WHEN EXISTS (
    SELECT 1 FROM envios_registro WHERE instancia_id = ? AND chat_jid = ? AND resultado = ?
) OR EXISTS (
    SELECT 1 FROM mensagens_processadas WHERE instancia_id = ? AND chat_jid = ? AND enviada_por_mim = ?
) THEN 1 ELSE 0 END`),
			envio.InstanciaID, envio.ChatJID, models.EnvioResultadoEnviada,
			envio.InstanciaID, envio.ChatJID, s.boolDB(false),
		).Scan(&conhecido)
		if err != nil {
			return fmt.Errorf("erro ao consultar historico do contato: %w", err)
		}
		novo = conhecido == 0
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO envios_registro (instancia_id, chat_jid, tipo, resultado, contato_novo, dia, minuto, criado_em)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		envio.InstanciaID, envio.ChatJID, envio.Tipo, envio.Resultado, s.boolDB(novo),
		envio.CriadoEm.Format(time.DateOnly), envio.CriadoEm.Unix()/60, envio.CriadoEm.UTC(),
	)
	if err != nil {
		return fmt.Errorf("erro ao registrar envio: %w", err)
	}
	return nil
}

// ResumoUso monta os numeros do painel. "Hoje" e os dias seguem o fuso do
// servidor (TZ).
func (s *SQLStore) ResumoUso(ctx context.Context, instanciaID string, agora time.Time, dias int) (models.ResumoUso, error) {
	resumo := models.ResumoUso{Rajadas: []models.UsoContato{}, TopDestinatarios: []models.UsoContato{}}
	hoje := agora.Format(time.DateOnly)
	minutoAtual := agora.Unix() / 60
	enviada := models.EnvioResultadoEnviada

	// Totais por dia.
	porDia := map[string]models.UsoDia{}
	inicio := agora.AddDate(0, 0, -(dias - 1)).Format(time.DateOnly)
	rows, err := s.db.QueryContext(ctx, s.q(`
SELECT dia,
    SUM(CASE WHEN resultado = ? THEN 1 ELSE 0 END),
    SUM(CASE WHEN contato_novo = ? THEN 1 ELSE 0 END),
    SUM(CASE WHEN resultado = ? THEN 1 ELSE 0 END)
FROM envios_registro WHERE instancia_id = ? AND dia >= ?
GROUP BY dia`), enviada, s.boolDB(true), models.EnvioResultadoLimitada, instanciaID, inicio)
	if err != nil {
		return resumo, fmt.Errorf("erro ao resumir envios por dia: %w", err)
	}
	for rows.Next() {
		var d models.UsoDia
		if err := rows.Scan(&d.Dia, &d.Envios, &d.ContatosNovos, &d.Limitados); err != nil {
			rows.Close()
			return resumo, fmt.Errorf("erro ao ler envios por dia: %w", err)
		}
		porDia[d.Dia] = d
	}
	rows.Close()
	for i := dias - 1; i >= 0; i-- {
		dia := agora.AddDate(0, 0, -i).Format(time.DateOnly)
		d := porDia[dia]
		d.Dia = dia
		resumo.Dias = append(resumo.Dias, d)
	}
	resumo.Hoje = resumo.Dias[len(resumo.Dias)-1]

	if err := s.db.QueryRowContext(ctx, s.q(`
SELECT
    COALESCE(SUM(CASE WHEN minuto >= ? THEN 1 ELSE 0 END), 0),
    COUNT(*)
FROM envios_registro WHERE instancia_id = ? AND resultado = ? AND minuto > ?`),
		minutoAtual, instanciaID, enviada, minutoAtual-60,
	).Scan(&resumo.UltimoMinuto, &resumo.UltimaHora); err != nil {
		return resumo, fmt.Errorf("erro ao contar envios recentes: %w", err)
	}

	var picoMinuto int64
	err = s.db.QueryRowContext(ctx, s.q(`
SELECT minuto, COUNT(*) FROM envios_registro
WHERE instancia_id = ? AND dia = ? AND resultado = ?
GROUP BY minuto ORDER BY COUNT(*) DESC, minuto DESC LIMIT 1`), instanciaID, hoje, enviada).Scan(&picoMinuto, &resumo.PicoMinutoHoje)
	if err != nil && err != sql.ErrNoRows {
		return resumo, fmt.Errorf("erro ao calcular pico por minuto: %w", err)
	}
	if err == nil {
		em := time.Unix(picoMinuto*60, 0)
		resumo.PicoMinutoEm = &em
	}

	resumo.TopDestinatarios, err = s.listarUsoContatos(ctx, `
SELECT chat_jid, COUNT(*) FROM envios_registro
WHERE instancia_id = ? AND dia = ? AND resultado = ? AND chat_jid <> ''
GROUP BY chat_jid ORDER BY COUNT(*) DESC LIMIT 5`, instanciaID, hoje, enviada)
	if err != nil {
		return resumo, err
	}
	resumo.Rajadas, err = s.listarUsoContatos(ctx, `
SELECT chat_jid, MAX(total) FROM (
    SELECT chat_jid, minuto, COUNT(*) AS total FROM envios_registro
    WHERE instancia_id = ? AND dia = ? AND resultado = ? AND chat_jid <> ''
    GROUP BY chat_jid, minuto
) por_minuto WHERE total >= ?
GROUP BY chat_jid ORDER BY MAX(total) DESC LIMIT 10`, instanciaID, hoje, enviada, rajadaMinima)
	return resumo, err
}

func (s *SQLStore) listarUsoContatos(ctx context.Context, query string, args ...any) ([]models.UsoContato, error) {
	rows, err := s.db.QueryContext(ctx, s.q(query), args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar destinatarios: %w", err)
	}
	defer rows.Close()
	lista := []models.UsoContato{}
	for rows.Next() {
		var c models.UsoContato
		if err := rows.Scan(&c.ChatJID, &c.Envios); err != nil {
			return nil, fmt.Errorf("erro ao ler destinatario: %w", err)
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

// UsoHojePorInstancia devolve os totais de hoje de cada instancia que teve
// movimento. Rajadas e quantos contatos receberam 5+ mensagens no mesmo minuto.
func (s *SQLStore) UsoHojePorInstancia(ctx context.Context, agora time.Time) ([]models.UsoInstanciaHoje, error) {
	hoje := agora.Format(time.DateOnly)
	enviada := models.EnvioResultadoEnviada
	porInstancia := map[string]*models.UsoInstanciaHoje{}
	ordem := []string{}
	rows, err := s.db.QueryContext(ctx, s.q(`
SELECT instancia_id,
    SUM(CASE WHEN resultado = ? THEN 1 ELSE 0 END),
    SUM(CASE WHEN contato_novo = ? THEN 1 ELSE 0 END),
    SUM(CASE WHEN resultado = ? THEN 1 ELSE 0 END)
FROM envios_registro WHERE dia = ?
GROUP BY instancia_id`), enviada, s.boolDB(true), models.EnvioResultadoLimitada, hoje)
	if err != nil {
		return nil, fmt.Errorf("erro ao resumir uso de hoje: %w", err)
	}
	for rows.Next() {
		u := &models.UsoInstanciaHoje{}
		if err := rows.Scan(&u.InstanciaID, &u.Envios, &u.ContatosNovos, &u.Limitados); err != nil {
			rows.Close()
			return nil, fmt.Errorf("erro ao ler uso de hoje: %w", err)
		}
		porInstancia[u.InstanciaID] = u
		ordem = append(ordem, u.InstanciaID)
	}
	rows.Close()

	rows, err = s.db.QueryContext(ctx, s.q(`
SELECT instancia_id, COUNT(DISTINCT chat_jid) FROM (
    SELECT instancia_id, chat_jid FROM envios_registro
    WHERE dia = ? AND resultado = ? AND chat_jid <> ''
    GROUP BY instancia_id, chat_jid, minuto HAVING COUNT(*) >= ?
) rajadas GROUP BY instancia_id`), hoje, enviada, rajadaMinima)
	if err != nil {
		return nil, fmt.Errorf("erro ao contar rajadas de hoje: %w", err)
	}
	for rows.Next() {
		var id string
		var total int
		if err := rows.Scan(&id, &total); err != nil {
			rows.Close()
			return nil, fmt.Errorf("erro ao ler rajadas de hoje: %w", err)
		}
		if u, ok := porInstancia[id]; ok {
			u.Rajadas = total
		}
	}
	rows.Close()

	lista := make([]models.UsoInstanciaHoje, 0, len(ordem))
	for _, id := range ordem {
		lista = append(lista, *porInstancia[id])
	}
	return lista, nil
}

func (s *SQLStore) LimparEnviosAntigos(ctx context.Context, antesDe time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx, s.q(`DELETE FROM envios_registro WHERE criado_em < ?`), antesDe.UTC())
	if err != nil {
		return 0, fmt.Errorf("erro ao limpar historico de envios: %w", err)
	}
	apagadas, _ := result.RowsAffected()
	return apagadas, nil
}
