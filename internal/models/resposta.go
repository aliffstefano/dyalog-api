package models

type RespostaSucesso struct {
	Sucesso  bool        `json:"sucesso"`
	Mensagem string      `json:"mensagem"`
	Dados    interface{} `json:"dados"`
}

type RespostaErro struct {
	Sucesso  bool   `json:"sucesso"`
	Erro     string `json:"erro"`
	Mensagem string `json:"mensagem"`
}

func NovaRespostaSucesso(mensagem string, dados interface{}) RespostaSucesso {
	return RespostaSucesso{
		Sucesso:  true,
		Mensagem: mensagem,
		Dados:    dados,
	}
}

func NovaRespostaErro(codigo, mensagem string) RespostaErro {
	return RespostaErro{
		Sucesso:  false,
		Erro:     codigo,
		Mensagem: mensagem,
	}
}
