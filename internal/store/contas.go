package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"dyalog-api-go/internal/models"
)

var (
	ErrContaNaoEncontrada   = errors.New("conta nao encontrada")
	ErrPlanoNaoEncontrado   = errors.New("plano nao encontrado")
	ErrUsuarioNaoEncontrado = errors.New("usuario nao encontrado")
	ErrEmailEmUso           = errors.New("ja existe um usuario com esse e-mail")
	ErrConviteInvalido      = errors.New("convite invalido, expirado ou ja usado")
	ErrSessaoInvalida       = errors.New("sessao invalida ou expirada")
	ErrRedefinicaoInvalida  = errors.New("link de redefinicao invalido, expirado ou ja usado")
)

// ContaStore guarda contas (empresas), planos, usuarios, convites, sessoes do
// painel e pedidos de redefinicao de senha.
type ContaStore interface {
	ListarPlanos(ctx context.Context) ([]models.Plano, error)
	BuscarPlano(ctx context.Context, id string) (models.Plano, error)
	SalvarPlano(ctx context.Context, plano models.Plano) error

	CriarConta(ctx context.Context, conta models.Conta) error
	AtualizarConta(ctx context.Context, conta models.Conta) error
	BuscarConta(ctx context.Context, id string) (models.Conta, error)
	ListarContasResumo(ctx context.Context) ([]models.ContaResumo, error)
	ContarInstanciasConta(ctx context.Context, contaID string) (int, error)
	AtualizarContaInstancia(ctx context.Context, instanciaID, contaID string) error

	CriarUsuario(ctx context.Context, usuario models.Usuario) error
	BuscarUsuarioPorID(ctx context.Context, id string) (models.Usuario, error)
	BuscarUsuarioPorEmail(ctx context.Context, email string) (models.Usuario, error)
	ListarUsuarios(ctx context.Context, contaID string, superadmins bool) ([]models.Usuario, error)
	AtualizarUsuario(ctx context.Context, usuario models.Usuario) error
	AtualizarSenhaUsuario(ctx context.Context, usuarioID, senhaHash string) error
	RegistrarLoginUsuario(ctx context.Context, usuarioID string) error
	ExcluirUsuario(ctx context.Context, usuarioID string) error
	ContarDonosAtivos(ctx context.Context, contaID string) (int, error)

	CriarConvite(ctx context.Context, convite models.Convite) error
	BuscarConvitePendente(ctx context.Context, tokenHash string) (models.Convite, error)
	ListarConvitesPendentes(ctx context.Context, contaID string, superadmins bool) ([]models.Convite, error)
	ExcluirConvite(ctx context.Context, id, contaID string) error
	MarcarConviteAceito(ctx context.Context, id string) (bool, error)

	CriarSessao(ctx context.Context, tokenHash, usuarioID string, expiraEm time.Time) error
	BuscarSessao(ctx context.Context, tokenHash string) (string, error)
	ExcluirSessao(ctx context.Context, tokenHash string) error
	ExcluirSessoesUsuario(ctx context.Context, usuarioID string) error

	CriarRedefinicaoSenha(ctx context.Context, tokenHash, usuarioID string, expiraEm time.Time) error
	UsarRedefinicaoSenha(ctx context.Context, tokenHash string) (string, error)
}

// ── Planos ───────────────────────────────────────────────────────────────────

func (s *SQLStore) ListarPlanos(ctx context.Context) ([]models.Plano, error) {
	linhas, err := s.db.QueryContext(ctx, `SELECT id, nome, preco_centavos, limite_instancias, ativo, criado_em FROM planos ORDER BY preco_centavos, nome`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar planos: %w", err)
	}
	defer linhas.Close()
	var planos []models.Plano
	for linhas.Next() {
		var p models.Plano
		var ativo any
		if err := linhas.Scan(&p.ID, &p.Nome, &p.PrecoCentavos, &p.LimiteInstancias, &ativo, &p.CriadoEm); err != nil {
			return nil, err
		}
		p.Ativo = s.boolFromDB(ativo)
		planos = append(planos, p)
	}
	return planos, linhas.Err()
}

func (s *SQLStore) BuscarPlano(ctx context.Context, id string) (models.Plano, error) {
	var p models.Plano
	var ativo any
	err := s.db.QueryRowContext(ctx, s.q(`SELECT id, nome, preco_centavos, limite_instancias, ativo, criado_em FROM planos WHERE id = ?`), id).
		Scan(&p.ID, &p.Nome, &p.PrecoCentavos, &p.LimiteInstancias, &ativo, &p.CriadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Plano{}, ErrPlanoNaoEncontrado
	}
	if err != nil {
		return models.Plano{}, fmt.Errorf("erro ao buscar plano: %w", err)
	}
	p.Ativo = s.boolFromDB(ativo)
	return p, nil
}

func (s *SQLStore) SalvarPlano(ctx context.Context, p models.Plano) error {
	if p.CriadoEm.IsZero() {
		p.CriadoEm = time.Now()
	}
	_, err := s.db.ExecContext(ctx, s.q(`
INSERT INTO planos (id, nome, preco_centavos, limite_instancias, ativo, criado_em) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET nome = excluded.nome, preco_centavos = excluded.preco_centavos,
    limite_instancias = excluded.limite_instancias, ativo = excluded.ativo`),
		p.ID, p.Nome, p.PrecoCentavos, p.LimiteInstancias, s.boolDB(p.Ativo), p.CriadoEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao salvar plano: %w", err)
	}
	return nil
}

// ── Contas ───────────────────────────────────────────────────────────────────

func (s *SQLStore) CriarConta(ctx context.Context, c models.Conta) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO contas (id, nome, plano_id, status, limite_instancias, tipos_instancia, criado_em, atualizado_em) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		c.ID, c.Nome, c.PlanoID, c.Status, c.LimiteInstancias, c.TiposInstancia, c.CriadoEm.UTC(), c.AtualizadoEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao criar conta: %w", err)
	}
	return nil
}

func (s *SQLStore) AtualizarConta(ctx context.Context, c models.Conta) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE contas SET nome = ?, plano_id = ?, status = ?, limite_instancias = ?, tipos_instancia = ?, atualizado_em = ? WHERE id = ?`),
		c.Nome, c.PlanoID, c.Status, c.LimiteInstancias, c.TiposInstancia, time.Now().UTC(), c.ID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar conta: %w", err)
	}
	return exigirLinhaAfetada(resultado, ErrContaNaoEncontrada)
}

func (s *SQLStore) BuscarConta(ctx context.Context, id string) (models.Conta, error) {
	var c models.Conta
	err := s.db.QueryRowContext(ctx, s.q(`SELECT id, nome, plano_id, status, limite_instancias, tipos_instancia, criado_em, atualizado_em FROM contas WHERE id = ?`), id).
		Scan(&c.ID, &c.Nome, &c.PlanoID, &c.Status, &c.LimiteInstancias, &c.TiposInstancia, &c.CriadoEm, &c.AtualizadoEm)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Conta{}, ErrContaNaoEncontrada
	}
	if err != nil {
		return models.Conta{}, fmt.Errorf("erro ao buscar conta: %w", err)
	}
	return c, nil
}

func (s *SQLStore) ListarContasResumo(ctx context.Context) ([]models.ContaResumo, error) {
	linhas, err := s.db.QueryContext(ctx, `
SELECT c.id, c.nome, c.plano_id, c.status, c.limite_instancias, c.tipos_instancia, c.criado_em, c.atualizado_em,
       COALESCE(p.nome, ''), COALESCE(p.limite_instancias, 0),
       (SELECT COUNT(*) FROM instancias i WHERE i.conta_id = c.id),
       (SELECT COUNT(*) FROM usuarios u WHERE u.conta_id = c.id),
       (SELECT COUNT(*) FROM convites v WHERE v.conta_id = c.id AND v.aceito_em IS NULL)
FROM contas c LEFT JOIN planos p ON p.id = c.plano_id
ORDER BY c.nome`)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar contas: %w", err)
	}
	defer linhas.Close()
	var contas []models.ContaResumo
	for linhas.Next() {
		var r models.ContaResumo
		var limitePlano int
		if err := linhas.Scan(&r.ID, &r.Nome, &r.PlanoID, &r.Status, &r.LimiteInstancias, &r.TiposInstancia, &r.CriadoEm, &r.AtualizadoEm,
			&r.PlanoNome, &limitePlano, &r.Instancias, &r.Usuarios, &r.ConvitesPendentes); err != nil {
			return nil, err
		}
		r.LimiteEfetivo = r.LimiteInstancias
		if r.LimiteEfetivo == 0 {
			r.LimiteEfetivo = limitePlano
		}
		contas = append(contas, r)
	}
	return contas, linhas.Err()
}

func (s *SQLStore) ContarInstanciasConta(ctx context.Context, contaID string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, s.q(`SELECT COUNT(*) FROM instancias WHERE conta_id = ?`), contaID).Scan(&total)
	return total, err
}

func (s *SQLStore) AtualizarContaInstancia(ctx context.Context, instanciaID, contaID string) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE instancias SET conta_id = ?, atualizado_em = ? WHERE id = ?`), contaID, time.Now().UTC(), instanciaID)
	if err != nil {
		return fmt.Errorf("erro ao mover instancia de conta: %w", err)
	}
	return exigirLinhaAfetada(resultado, ErrInstanciaNaoEncontrada)
}

// ── Usuarios ─────────────────────────────────────────────────────────────────

const colunasUsuario = `id, conta_id, nome, email, papel, superadmin, ativo, senha_hash, criado_em, ultimo_login_em`

func (s *SQLStore) scanUsuario(scanner interface{ Scan(...any) error }) (models.Usuario, error) {
	var u models.Usuario
	var superadmin, ativo any
	var ultimo sql.NullTime
	if err := scanner.Scan(&u.ID, &u.ContaID, &u.Nome, &u.Email, &u.Papel, &superadmin, &ativo, &u.SenhaHash, &u.CriadoEm, &ultimo); err != nil {
		return models.Usuario{}, err
	}
	u.Superadmin, u.Ativo = s.boolFromDB(superadmin), s.boolFromDB(ativo)
	if ultimo.Valid {
		u.UltimoLoginEm = ultimo.Time
	}
	return u, nil
}

func normalizarEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (s *SQLStore) CriarUsuario(ctx context.Context, u models.Usuario) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO usuarios (id, conta_id, nome, email, papel, superadmin, ativo, senha_hash, criado_em) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		u.ID, u.ContaID, u.Nome, normalizarEmail(u.Email), u.Papel, s.boolDB(u.Superadmin), s.boolDB(u.Ativo), u.SenhaHash, u.CriadoEm.UTC())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return ErrEmailEmUso
		}
		return fmt.Errorf("erro ao criar usuario: %w", err)
	}
	return nil
}

func (s *SQLStore) buscarUsuario(ctx context.Context, coluna, valor string) (models.Usuario, error) {
	u, err := s.scanUsuario(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasUsuario+` FROM usuarios WHERE `+coluna+` = ?`), valor))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Usuario{}, ErrUsuarioNaoEncontrado
	}
	if err != nil {
		return models.Usuario{}, fmt.Errorf("erro ao buscar usuario: %w", err)
	}
	return u, nil
}

func (s *SQLStore) BuscarUsuarioPorID(ctx context.Context, id string) (models.Usuario, error) {
	return s.buscarUsuario(ctx, "id", id)
}

func (s *SQLStore) BuscarUsuarioPorEmail(ctx context.Context, email string) (models.Usuario, error) {
	return s.buscarUsuario(ctx, "email", normalizarEmail(email))
}

// ListarUsuarios lista a equipe da conta; superadmins=true lista os
// superadmins (sem conta).
func (s *SQLStore) ListarUsuarios(ctx context.Context, contaID string, superadmins bool) ([]models.Usuario, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT `+colunasUsuario+` FROM usuarios WHERE conta_id = ? AND superadmin = ? ORDER BY nome, email`), contaID, s.boolDB(superadmins))
	if err != nil {
		return nil, fmt.Errorf("erro ao listar usuarios: %w", err)
	}
	defer linhas.Close()
	var usuarios []models.Usuario
	for linhas.Next() {
		u, err := s.scanUsuario(linhas)
		if err != nil {
			return nil, err
		}
		usuarios = append(usuarios, u)
	}
	return usuarios, linhas.Err()
}

func (s *SQLStore) AtualizarUsuario(ctx context.Context, u models.Usuario) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE usuarios SET nome = ?, papel = ?, ativo = ? WHERE id = ?`), u.Nome, u.Papel, s.boolDB(u.Ativo), u.ID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar usuario: %w", err)
	}
	return exigirLinhaAfetada(resultado, ErrUsuarioNaoEncontrado)
}

func (s *SQLStore) AtualizarSenhaUsuario(ctx context.Context, usuarioID, senhaHash string) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE usuarios SET senha_hash = ? WHERE id = ?`), senhaHash, usuarioID)
	if err != nil {
		return fmt.Errorf("erro ao trocar senha: %w", err)
	}
	return exigirLinhaAfetada(resultado, ErrUsuarioNaoEncontrado)
}

func (s *SQLStore) RegistrarLoginUsuario(ctx context.Context, usuarioID string) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE usuarios SET ultimo_login_em = ? WHERE id = ?`), time.Now().UTC(), usuarioID)
	return err
}

func (s *SQLStore) ExcluirUsuario(ctx context.Context, usuarioID string) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`DELETE FROM usuarios WHERE id = ?`), usuarioID)
	if err != nil {
		return fmt.Errorf("erro ao excluir usuario: %w", err)
	}
	if err := exigirLinhaAfetada(resultado, ErrUsuarioNaoEncontrado); err != nil {
		return err
	}
	return s.ExcluirSessoesUsuario(ctx, usuarioID)
}

func (s *SQLStore) ContarDonosAtivos(ctx context.Context, contaID string) (int, error) {
	var total int
	err := s.db.QueryRowContext(ctx, s.q(`SELECT COUNT(*) FROM usuarios WHERE conta_id = ? AND papel = ? AND ativo = ?`), contaID, models.PapelDono, s.boolDB(true)).Scan(&total)
	return total, err
}

// ── Convites ─────────────────────────────────────────────────────────────────

func (s *SQLStore) CriarConvite(ctx context.Context, c models.Convite) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO convites (id, conta_id, email, papel, superadmin, token_hash, criado_por, expira_em, criado_em) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		c.ID, c.ContaID, normalizarEmail(c.Email), c.Papel, s.boolDB(c.Superadmin), c.TokenHash, c.CriadoPor, c.ExpiraEm.UTC(), c.CriadoEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao criar convite: %w", err)
	}
	return nil
}

const colunasConvite = `id, conta_id, email, papel, superadmin, token_hash, criado_por, expira_em, criado_em`

func (s *SQLStore) scanConvite(scanner interface{ Scan(...any) error }) (models.Convite, error) {
	var c models.Convite
	var superadmin any
	if err := scanner.Scan(&c.ID, &c.ContaID, &c.Email, &c.Papel, &superadmin, &c.TokenHash, &c.CriadoPor, &c.ExpiraEm, &c.CriadoEm); err != nil {
		return models.Convite{}, err
	}
	c.Superadmin = s.boolFromDB(superadmin)
	return c, nil
}

func (s *SQLStore) BuscarConvitePendente(ctx context.Context, tokenHash string) (models.Convite, error) {
	c, err := s.scanConvite(s.db.QueryRowContext(ctx, s.q(`SELECT `+colunasConvite+` FROM convites WHERE token_hash = ? AND aceito_em IS NULL AND expira_em > ?`), tokenHash, time.Now().UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Convite{}, ErrConviteInvalido
	}
	if err != nil {
		return models.Convite{}, fmt.Errorf("erro ao buscar convite: %w", err)
	}
	return c, nil
}

func (s *SQLStore) ListarConvitesPendentes(ctx context.Context, contaID string, superadmins bool) ([]models.Convite, error) {
	linhas, err := s.db.QueryContext(ctx, s.q(`SELECT `+colunasConvite+` FROM convites WHERE conta_id = ? AND superadmin = ? AND aceito_em IS NULL ORDER BY criado_em DESC`), contaID, s.boolDB(superadmins))
	if err != nil {
		return nil, fmt.Errorf("erro ao listar convites: %w", err)
	}
	defer linhas.Close()
	var convites []models.Convite
	for linhas.Next() {
		c, err := s.scanConvite(linhas)
		if err != nil {
			return nil, err
		}
		convites = append(convites, c)
	}
	return convites, linhas.Err()
}

func (s *SQLStore) ExcluirConvite(ctx context.Context, id, contaID string) error {
	resultado, err := s.db.ExecContext(ctx, s.q(`DELETE FROM convites WHERE id = ? AND conta_id = ? AND aceito_em IS NULL`), id, contaID)
	if err != nil {
		return fmt.Errorf("erro ao excluir convite: %w", err)
	}
	return exigirLinhaAfetada(resultado, ErrConviteInvalido)
}

// MarcarConviteAceito usa o convite uma vez so (false se ja tinha sido usado).
func (s *SQLStore) MarcarConviteAceito(ctx context.Context, id string) (bool, error) {
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE convites SET aceito_em = ? WHERE id = ? AND aceito_em IS NULL`), time.Now().UTC(), id)
	if err != nil {
		return false, fmt.Errorf("erro ao usar convite: %w", err)
	}
	linhas, err := resultado.RowsAffected()
	return linhas > 0, err
}

// ── Sessoes e redefinicao de senha ───────────────────────────────────────────

func (s *SQLStore) CriarSessao(ctx context.Context, tokenHash, usuarioID string, expiraEm time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO sessoes (token_hash, usuario_id, criado_em, expira_em) VALUES (?, ?, ?, ?)`),
		tokenHash, usuarioID, time.Now().UTC(), expiraEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao criar sessao: %w", err)
	}
	return nil
}

func (s *SQLStore) BuscarSessao(ctx context.Context, tokenHash string) (string, error) {
	var usuarioID string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT usuario_id FROM sessoes WHERE token_hash = ? AND expira_em > ?`), tokenHash, time.Now().UTC()).Scan(&usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSessaoInvalida
	}
	return usuarioID, err
}

func (s *SQLStore) ExcluirSessao(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM sessoes WHERE token_hash = ?`), tokenHash)
	return err
}

func (s *SQLStore) ExcluirSessoesUsuario(ctx context.Context, usuarioID string) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM sessoes WHERE usuario_id = ?`), usuarioID)
	return err
}

func (s *SQLStore) CriarRedefinicaoSenha(ctx context.Context, tokenHash, usuarioID string, expiraEm time.Time) error {
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO redefinicoes_senha (token_hash, usuario_id, expira_em) VALUES (?, ?, ?)`), tokenHash, usuarioID, expiraEm.UTC())
	if err != nil {
		return fmt.Errorf("erro ao criar redefinicao de senha: %w", err)
	}
	return nil
}

// UsarRedefinicaoSenha consome o link (uma vez so) e devolve o usuario.
func (s *SQLStore) UsarRedefinicaoSenha(ctx context.Context, tokenHash string) (string, error) {
	agora := time.Now().UTC()
	var usuarioID string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT usuario_id FROM redefinicoes_senha WHERE token_hash = ? AND usado_em IS NULL AND expira_em > ?`), tokenHash, agora).Scan(&usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrRedefinicaoInvalida
	}
	if err != nil {
		return "", err
	}
	resultado, err := s.db.ExecContext(ctx, s.q(`UPDATE redefinicoes_senha SET usado_em = ? WHERE token_hash = ? AND usado_em IS NULL`), agora, tokenHash)
	if err != nil {
		return "", err
	}
	if n, _ := resultado.RowsAffected(); n == 0 {
		return "", ErrRedefinicaoInvalida
	}
	return usuarioID, nil
}
