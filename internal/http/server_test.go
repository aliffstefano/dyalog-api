package http

import (
	"testing"
	"time"
)

func TestDuracaoAteProximoHorarioAindaNaoChegou(t *testing.T) {
	// Se o horario alvo (04:00) ainda nao chegou hoje, deve apontar pra hoje.
	agora := time.Date(2026, 1, 10, 2, 0, 0, 0, time.UTC)
	espera := duracaoAteProximoHorarioReferencia(agora, 4, 0)
	esperado := 2 * time.Hour
	if espera != esperado {
		t.Fatalf("espera = %v, esperado %v", espera, esperado)
	}
}

func TestDuracaoAteProximoHorarioJaPassou(t *testing.T) {
	// Se o horario alvo (04:00) ja passou hoje, deve apontar pra amanha.
	agora := time.Date(2026, 1, 10, 5, 0, 0, 0, time.UTC)
	espera := duracaoAteProximoHorarioReferencia(agora, 4, 0)
	esperado := 23 * time.Hour
	if espera != esperado {
		t.Fatalf("espera = %v, esperado %v", espera, esperado)
	}
}

func TestDuracaoAteProximoHorarioExatamenteAgora(t *testing.T) {
	// No instante exato do horario alvo, deve rolar para amanha (nao dispara
	// duas vezes seguidas nem fica com duracao zero/negativa).
	agora := time.Date(2026, 1, 10, 4, 0, 0, 0, time.UTC)
	espera := duracaoAteProximoHorarioReferencia(agora, 4, 0)
	esperado := 24 * time.Hour
	if espera != esperado {
		t.Fatalf("espera = %v, esperado %v", espera, esperado)
	}
}
