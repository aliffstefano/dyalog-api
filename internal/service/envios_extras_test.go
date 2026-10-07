package service

import (
	"testing"

	"dyalog-api-go/internal/models"
)

func TestNormalizarEventoCompatFormatoRyze(t *testing.T) {
	req := normalizarEventoCompat(models.EnvioEventoRequest{
		Number:            "120363312345678901@g.us",
		Name:              "Reuniao do time",
		StartAt:           "2026-04-28T14:00:00-03:00",
		EndAt:             "2026-04-28T16:00:00-03:00",
		Location:          &models.EventoLocalRequest{Name: "Sala 3", Address: "Av. Paulista, 1000"},
		ReminderOffsetSec: 900,
		IsScheduleCall:    true,
		JoinLink:          "https://meet.example.com/daily",
	})
	if req.Numero != "120363312345678901@g.us" || req.Nome != "Reuniao do time" || req.Inicio == "" || req.Fim == "" {
		t.Fatalf("campos basicos nao convertidos: %+v", req)
	}
	if req.Local != "Sala 3" || req.Endereco != "Av. Paulista, 1000" {
		t.Fatalf("local nao convertido: %q %q", req.Local, req.Endereco)
	}
	if !req.Lembrete || req.LembreteSegundos != 900 {
		t.Fatalf("lembrete = %v %d, esperado true 900", req.Lembrete, req.LembreteSegundos)
	}
	if !req.ChamadaAgendada || req.LinkChamada == "" {
		t.Fatalf("chamada agendada/link nao convertidos")
	}
}

func TestNormalizarEventoLembreteEmMinutos(t *testing.T) {
	req := normalizarEventoCompat(models.EnvioEventoRequest{LembreteMinutos: 30})
	if !req.Lembrete || req.LembreteSegundos != 1800 {
		t.Fatalf("lembrete = %v %d, esperado true 1800", req.Lembrete, req.LembreteSegundos)
	}
}
