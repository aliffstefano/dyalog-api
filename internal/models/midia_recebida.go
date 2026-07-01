package models

import "time"

type MidiaRecebida struct {
	ID              string    `json:"id"`
	InstanciaID     string    `json:"instancia_id"`
	MensagemID      string    `json:"mensagem_id"`
	ChatJID         string    `json:"chat_jid"`
	RemetenteJID    string    `json:"remetente_jid"`
	Tipo            string    `json:"tipo"`
	MimeType        string    `json:"mime_type,omitempty"`
	NomeArquivo     string    `json:"nome_arquivo,omitempty"`
	CaminhoArquivo  string    `json:"-"`
	StorageProvider string    `json:"storage_provider,omitempty"`
	StoragePath     string    `json:"storage_path,omitempty"`
	StorageURL      string    `json:"storage_url,omitempty"`
	TamanhoBytes    int64     `json:"tamanho_bytes"`
	SHA256          string    `json:"sha256,omitempty"`
	Base64          string    `json:"base64,omitempty"`
	DataURI         string    `json:"data_uri,omitempty"`
	RecebidaEm      time.Time `json:"recebida_em"`
	CriadaEm        time.Time `json:"criada_em"`
}

func (m MidiaRecebida) DownloadPath() string {
	return "/api/v1/instancias/" + m.InstanciaID + "/midia/" + m.ID
}
