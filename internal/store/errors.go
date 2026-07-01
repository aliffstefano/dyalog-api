package store

import "errors"

var (
	ErrInstanciaNaoEncontrada   = errors.New("instancia nao encontrada")
	ErrDependenciaNaoEncontrada = errors.New("dependencia nao encontrada")
	ErrWebhookNaoEncontrado     = errors.New("webhook nao encontrado")
	ErrMidiaNaoEncontrada       = errors.New("midia nao encontrada")
)
