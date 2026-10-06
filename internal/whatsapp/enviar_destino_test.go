package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
)

func TestEnviarPrimeiroDestinoUsaOPrimeiroQueAceita(t *testing.T) {
	jids := []types.JID{
		types.NewJID("556799999999", types.DefaultUserServer),
		types.NewJID("5567999999999", types.DefaultUserServer),
	}
	tentativas := 0
	resp, usado, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
		tentativas++
		if jid == jids[0] {
			return whatsmeow.SendResponse{}, errors.New("recusado")
		}
		return whatsmeow.SendResponse{ID: "ABC"}, nil
	})
	if err != nil || usado != jids[1] || resp.ID != "ABC" || tentativas != 2 {
		t.Fatalf("resp=%v usado=%v err=%v tentativas=%d", resp, usado, err, tentativas)
	}
}

func TestEnviarPrimeiroDestinoDevolveUltimoErro(t *testing.T) {
	jids := []types.JID{types.NewJID("1", types.DefaultUserServer), types.NewJID("2", types.DefaultUserServer)}
	_, usado, err := enviarPrimeiroDestino(jids, func(jid types.JID) (whatsmeow.SendResponse, error) {
		return whatsmeow.SendResponse{}, errors.New("falhou " + jid.User)
	})
	if err == nil || err.Error() != "falhou 2" || !usado.IsEmpty() {
		t.Fatalf("usado=%v err=%v", usado, err)
	}
	if _, _, err := enviarPrimeiroDestino(nil, nil); err == nil {
		t.Fatal("sem destinos deveria dar erro")
	}
}
