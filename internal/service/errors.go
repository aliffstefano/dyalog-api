package service

import "errors"

var (
	ErrInstanciaNaoEncontrada   = errors.New("instancia nao encontrada")
	ErrEntradaInvalida          = errors.New("entrada invalida")
	ErrDependenciaNaoEncontrada = errors.New("dependencia nao encontrada")
	ErrWebhookNaoEncontrado     = errors.New("webhook nao encontrado")
	ErrMidiaNaoEncontrada       = errors.New("midia nao encontrada")
	ErrAtualizacaoBloqueada     = errors.New("atualizacao bloqueada")
	ErrHistoricoBloqueado       = errors.New("historico bloqueado")
	ErrNenhumaAtualizacao       = errors.New("nenhuma atualizacao disponivel")
	ErrModoAtualizacaoInvalido  = errors.New("modo de atualizacao invalido")
	ErrAutorizacaoAtualizacao   = errors.New("nao autorizado para aplicar atualizacao")
)
