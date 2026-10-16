// Pagina de documentacao: exemplos de curl de todas as funcoes da API.
// __BASE__, __TOKEN__, __INSTANCIA__ e __NUMERO__ sao trocados pelo que a
// pessoa digita no topo (so no navegador, nada e enviado).
(function () {
  'use strict';
  const BASE = 'https://connect.dyalog.com.br';

  // ex(metodo, caminho, titulo, descricao, corpo, notas)
  const ex = (metodo, caminho, titulo, descricao, corpo, notas) => ({ metodo, caminho, titulo, descricao, corpo, notas });

  const SECOES = [
    {
      id: 'inicio', titulo: 'Primeiros passos', itens: [
        ex('GET', '/api/v1/saude', 'Testar se a API esta no ar', 'Nao precisa de token.', null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/status', 'Status da instancia', 'Se esta conectada, numero e perfil.', null),
        ex('GET', '/api/v1/instancias', 'Listar instancias', 'Com o token master (ou de usuario) lista todas; com token de instancia, so ela.', null),
      ],
    },
    {
      id: 'instancia', titulo: 'Instancia e conexao', itens: [
        ex('POST', '/api/v1/instancias', 'Criar instancia (QR code)', 'Precisa do token master. Devolve o id e o token da nova instancia.', { nome: 'Atendimento' }, 'Para a API oficial da Meta use "tipo": "meta" e "meta": {"phone_number_id": "...", "access_token": "...", "waba_id": "..."}.'),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/conectar', 'Conectar (gerar QR code)', 'Depois busque o QR na rota abaixo e escaneie no WhatsApp > Aparelhos conectados.', {}),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/qrcode', 'QR code (texto/base64)', 'Use /qrcode/imagem para receber o PNG direto.', null),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/pairing-code', 'Conectar por codigo (sem QR)', 'Gera o codigo de 8 digitos para digitar no celular em Aparelhos conectados > Conectar com numero.', { numero: '__NUMERO__' }),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/desconectar', 'Desconectar', 'Encerra a sessao do WhatsApp (precisa ler o QR de novo para voltar).', {}),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/token', 'Trocar o token da instancia', 'O token antigo para de funcionar na hora.', { token: 'novo-token-forte' }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/avancado', 'Comportamento', 'Manter online, marcar como lida, rejeitar chamadas, ignorar grupos e status.', { manter_online: false, marcar_lida_automatico: true, rejeitar_chamadas: true, mensagem_rejeitar_chamadas: 'No momento nao atendo ligacoes, mande uma mensagem.', ignorar_grupos: false, ignorar_status: true }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/presenca', 'Presenca padrao', 'disponivel (aparece online) ou indisponivel.', { presenca: 'indisponivel' }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/proxy', 'Proxy da instancia', 'modo: herdar (usa o global), proprio (informe url) ou direto (sem proxy).', { modo: 'proprio', url: 'socks5://usuario:senha@proxy.exemplo.com:1080' }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/historico', 'Historico ao conectar', 'Quantos dias de mensagens antigas vao para o webhook ao conectar (defina antes de ler o QR).', { dias: 7 }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/uso', 'Uso e risco de bloqueio', 'Envios do dia, contatos novos e rajadas.', null),
        ex('DELETE', '/api/v1/instancias/__INSTANCIA__', 'Excluir instancia', 'Remove a instancia e a sessao. Nao da para desfazer.', null),
      ],
    },
    {
      id: 'texto', titulo: 'Mensagens de texto', itens: [
        ex('POST', '/api/v1/batepapo/enviar/texto', 'Enviar texto', null, { numero: '__NUMERO__', mensagem: 'Ola! Tudo bem? 😊' }),
        ex('POST', '/api/v1/batepapo/enviar/texto', 'Texto com "digitando..." antes', 'Mostra digitando por alguns segundos antes de enviar (parece humano).', { numero: '__NUMERO__', mensagem: 'Ja vou te passar os valores.', digitando: true, delay_segundos: 3 }),
        ex('POST', '/api/v1/batepapo/enviar/texto', 'Responder citando uma mensagem', 'resposta_mensagem_id e o id que chegou no webhook. Em grupo, informe tambem resposta_participante (quem escreveu).', { numero: '__NUMERO__', mensagem: 'Sim, temos disponivel!', resposta_mensagem_id: '3EB0ABC123DEF456', resposta_conteudo: 'Tem o produto X?' }),
        ex('POST', '/api/v1/batepapo/enviar/texto', 'Texto para grupo', 'Use chat_jid do grupo (aparece no webhook e em Grupos no painel).', { chat_jid: '120363025246125486@g.us', mensagem: 'Bom dia, equipe!' }),
        ex('POST', '/api/v1/batepapo/editar/texto', 'Editar mensagem enviada', 'Ate 15 minutos depois do envio.', { numero: '__NUMERO__', mensagem_id: '3EB0ABC123DEF456', mensagem: 'Texto corrigido' }),
        ex('POST', '/api/v1/batepapo/apagar', 'Apagar para todos', null, { numero: '__NUMERO__', mensagem_id: '3EB0ABC123DEF456' }),
        ex('POST', '/api/v1/batepapo/reagir', 'Reagir com emoji', 'emoji vazio ("") remove a reacao.', { numero: '__NUMERO__', mensagem_id: '3EB0ABC123DEF456', emoji: '👍' }),
        ex('POST', '/api/v1/batepapo/marcar-lida', 'Marcar como lida', 'Envia o tique azul.', { numero: '__NUMERO__', mensagens_id: ['3EB0ABC123DEF456'] }),
        ex('POST', '/api/v1/batepapo/enviar/presenca', 'Mostrar "digitando" ou "gravando audio"', 'acao: digitando, gravando_audio, pausado, disponivel ou indisponivel.', { numero: '__NUMERO__', acao: 'digitando', delay_segundos: 5 }),
      ],
    },
    {
      id: 'midia', titulo: 'Imagem, audio e arquivos', itens: [
        ex('POST', '/api/v1/batepapo/enviar/imagem', 'Imagem por URL', 'A URL precisa ser publica.', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/produto.jpg', legenda: 'Confira nosso lancamento!' }),
        ex('POST', '/api/v1/batepapo/enviar/imagem', 'Imagem em base64', 'Aceita com ou sem o prefixo data:image/...;base64,', { numero: '__NUMERO__', arquivo_base64: 'iVBORw0KGgoAAAANSUhEUgAA...', mime_type: 'image/png', legenda: 'Comprovante' }),
        ex('POST', '/api/v1/batepapo/enviar/audio', 'Audio como nota de voz (gravado)', 'ptt: true chega como audio gravado na hora. Aceita OGG/Opus, e tambem WebM (gravacao do navegador), que a API converte.', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/recado.ogg', ptt: true }),
        ex('POST', '/api/v1/batepapo/enviar/audio', 'Audio como arquivo (MP3)', 'Sem ptt chega como arquivo de audio.', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/musica.mp3' }),
        ex('POST', '/api/v1/batepapo/enviar/documento', 'Documento (PDF, planilha, video...)', 'nome_arquivo e o nome que aparece para o contato.', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/boleto.pdf', nome_arquivo: 'Boleto-outubro.pdf', legenda: 'Segue o boleto' }),
        ex('POST', '/api/v1/batepapo/enviar/figurinha', 'Figurinha', 'Precisa ser WEBP.', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/figurinha.webp' }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/midias/ID_DA_MIDIA', 'Baixar midia recebida', 'O midia_id chega no webhook de mensagens.', null),
      ],
    },
    {
      id: 'interativos', titulo: 'Lista, botoes, enquete e carrossel', itens: [
        ex('POST', '/api/v1/batepapo/enviar/lista', 'Lista (menu de opcoes)', 'Funciona em WhatsApp pessoal e Business. A escolha chega no webhook com lista_id.', {
          numero: '__NUMERO__', descricao: 'Escolha o que deseja:', botao_texto: 'Ver opcoes', rodape: 'Autoescola Exemplo',
          secoes: [
            { titulo: 'Habilitacao', linhas: [{ id: 'cat_a', titulo: 'Categoria A', descricao: 'Moto' }, { id: 'cat_b', titulo: 'Categoria B', descricao: 'Carro' }] },
            { titulo: 'Outros', linhas: [{ id: 'atendente', titulo: 'Falar com atendente' }] },
          ],
        }, 'Ate 10 secoes e 10 linhas no total. Se o WhatsApp recusar, as opcoes vao como texto numerado.'),
        ex('POST', '/api/v1/batepapo/enviar/botoes', 'Botoes de resposta', 'Ate 3 botoes. O clique chega no webhook com o id do botao.', { numero: '__NUMERO__', texto: 'Confirma o horario de amanha as 8h?', rodape: 'Responda pelo botao', botoes: [{ id: 'sim', texto: 'Confirmar' }, { id: 'nao', texto: 'Remarcar' }] }),
        ex('POST', '/api/v1/batepapo/enviar/botoes', 'Botoes de link, ligar e copiar', 'tipo: resposta, url (URL), ligar (telefone) ou copiar (codigo). Podem ser misturados.', {
          numero: '__NUMERO__', titulo: 'Pedido #1234', texto: 'Seu pedido foi aprovado!',
          botoes: [{ tipo: 'url', texto: 'Acompanhar', URL: 'https://exemplo.com/pedido/1234' }, { tipo: 'ligar', texto: 'Ligar para a loja', telefone: '+5567999999999' }, { tipo: 'copiar', texto: 'Copiar codigo', codigo: 'BR123456789' }],
        }),
        ex('POST', '/api/v1/batepapo/enviar/enquete', 'Enquete', 'opcoes_selecionaveis: quantas o contato pode marcar (1 = so uma). Os votos chegam no webhook.', { numero: '__NUMERO__', nome: 'Qual o melhor horario para a aula?', opcoes: ['Manha', 'Tarde', 'Noite'], opcoes_selecionaveis: 1 }),
        ex('POST', '/api/v1/batepapo/enviar/carrossel', 'Carrossel de cartoes', 'Ate 10 cartoes, cada um com imagem, texto e 1 a 3 botoes.', {
          numero: '__NUMERO__', texto: 'Veja nossos planos:',
          cartoes: [
            { imagem_url: 'https://exemplo.com/plano-basico.jpg', titulo: 'Basico', texto: 'R$ 99/mes', botoes: [{ id: 'plano_basico', texto: 'Quero esse' }] },
            { imagem_url: 'https://exemplo.com/plano-pro.jpg', titulo: 'Pro', texto: 'R$ 199/mes', botoes: [{ id: 'plano_pro', texto: 'Quero esse' }, { tipo: 'url', texto: 'Detalhes', URL: 'https://exemplo.com/pro' }] },
          ],
        }),
        ex('POST', '/api/v1/batepapo/enviar/cobranca-pix', 'Cobranca Pix', 'tipo_chave: cpf, cnpj, email, telefone ou aleatoria. Mostra o botao de copiar a chave.', { numero: '__NUMERO__', chave_pix: '12345678000199', tipo_chave: 'cnpj', nome_beneficiario: 'Loja Exemplo LTDA', valor: 150.00, referencia: 'PEDIDO1234', descricao: 'Pedido #1234' }),
      ],
    },
    {
      id: 'outros', titulo: 'Localizacao, contato, evento e status', itens: [
        ex('POST', '/api/v1/batepapo/enviar/localizacao', 'Localizacao', null, { numero: '__NUMERO__', latitude: -20.4697, longitude: -54.6201, nome: 'Loja Centro', endereco: 'Rua 14 de Julho, 1000 - Campo Grande/MS' }),
        ex('POST', '/api/v1/batepapo/enviar/contato', 'Cartao de contato', null, { numero: '__NUMERO__', nome: 'Suporte Loja', telefone: '+5567999999999', organizacao: 'Loja Exemplo' }),
        ex('POST', '/api/v1/batepapo/enviar/evento', 'Evento (convite com data)', 'inicio/fim em ISO com fuso. lembrete avisa antes.', { numero: '__NUMERO__', nome: 'Aula pratica', descricao: 'Traga documento com foto', inicio: '2026-10-20T08:00:00-04:00', fim: '2026-10-20T09:00:00-04:00', local: 'Autoescola Exemplo', endereco: 'Rua 14 de Julho, 1000 - Campo Grande/MS', lembrete: true }),
        ex('POST', '/api/v1/chamadas/link', 'Link de chamada', 'Gera um link de ligacao do WhatsApp (tipo voz ou video) para mandar ao cliente.', { tipo: 'video' }),
        ex('POST', '/api/v1/status/postar', 'Postar status (texto)', 'Ate 700 caracteres.', { tipo: 'texto', texto: 'Promocao de hoje: 20% off!', cor_fundo: '#1995bd' }),
        ex('POST', '/api/v1/status/postar', 'Postar status (imagem)', 'tipo imagem ou video, com arquivo_url ou arquivo_base64.', { tipo: 'imagem', arquivo_url: 'https://exemplo.com/promo.jpg', legenda: 'So hoje!' }),
      ],
    },
    {
      id: 'agendamentos', titulo: 'Agendar mensagens', itens: [
        ex('POST', '/api/v1/agendamentos', 'Agendar texto', 'enviar_em em ISO com fuso. recorrencia: diaria, semanal, mensal ou vazio. Com token master, informe "instancia".', { numero: '__NUMERO__', mensagem: 'Lembrete: sua aula e amanha as 8h', enviar_em: '2026-10-20T09:00:00-04:00', recorrencia: 'semanal' }),
        ex('POST', '/api/v1/agendamentos', 'Agendar arquivo', 'tipo imagem, documento ou audio (deduzido do mime_type).', { numero: '__NUMERO__', arquivo_url: 'https://exemplo.com/boleto.pdf', nome_arquivo: 'Boleto.pdf', mime_type: 'application/pdf', mensagem: 'Seu boleto', enviar_em: '2026-10-25T08:00:00-04:00' }),
        ex('GET', '/api/v1/agendamentos?status=pendente', 'Listar agendamentos', 'status: pendente, enviado, falhou, cancelado (ou vazio para todos).', null),
        ex('DELETE', '/api/v1/agendamentos/ID_DO_AGENDAMENTO', 'Cancelar agendamento', null, null),
      ],
    },
    {
      id: 'contatos', titulo: 'Contatos e perfil', itens: [
        ex('POST', '/api/v1/instancias/__INSTANCIA__/contatos/verificar', 'Verificar se o numero tem WhatsApp', 'Ate 50 numeros por vez.', { numeros: ['__NUMERO__', '5511988887777'] }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/contatos/avatar?numero=__NUMERO__', 'Foto do contato', 'formato=arquivo redireciona para a imagem; formato=base64 devolve em base64.', null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/contatos?busca=maria&pagina=1&por_pagina=50', 'Listar contatos', 'tipo: pessoal, grupo, salvo ou bloqueado.', null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/contatos/exportar', 'Exportar contatos (CSV)', null, null),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/contatos/bloqueio', 'Bloquear / desbloquear', 'bloquear: false desbloqueia.', { numero: '__NUMERO__', bloquear: true }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/perfil', 'Ver perfil e privacidade', null, null),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/perfil', 'Trocar nome e recado', 'Nome ate 25 caracteres, recado ate 139.', { nome: 'Loja Exemplo', recado: 'Atendimento das 8h as 18h' }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/perfil/foto', 'Trocar foto do perfil', 'imagem (base64) ou url. A API recorta em quadrado.', { url: 'https://exemplo.com/logo.png' }),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/privacidade', 'Privacidade', 'Envie so o que muda. Valores: all, contacts, contact_blacklist, none (online: all/match_last_seen; chamadas: all/known; confirmacao_leitura: all/none).', { visto_por_ultimo: 'contacts', foto_perfil: 'all', confirmacao_leitura: 'all' }),
      ],
    },
    {
      id: 'grupos', titulo: 'Grupos', itens: [
        ex('GET', '/api/v1/instancias/__INSTANCIA__/grupos', 'Listar grupos', null, null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/grupos/120363025246125486@g.us', 'Grupo com participantes', null, null),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/grupos', 'Criar grupo', null, { nome: 'Turma Outubro', participantes: ['__NUMERO__', '5511988887777'] }),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/grupos/120363025246125486@g.us/participantes', 'Adicionar, remover, promover', 'acao: adicionar, remover, promover (admin) ou rebaixar. Quem nao pode ser adicionado volta com o motivo.', { acao: 'adicionar', numeros: ['__NUMERO__'] }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/grupos/120363025246125486@g.us/convite', 'Link de convite', '?redefinir=1 gera um novo e invalida o antigo.', null),
        ex('PUT', '/api/v1/instancias/__INSTANCIA__/grupos/120363025246125486@g.us', 'Nome, descricao e permissoes', null, { nome: 'Turma Outubro 2026', descricao: 'Avisos da turma', so_admins_enviam: true }),
        ex('POST', '/api/v1/instancias/__INSTANCIA__/grupos/120363025246125486@g.us/sair', 'Sair do grupo', null, {}),
      ],
    },
    {
      id: 'webhooks', titulo: 'Webhooks (receber mensagens)', itens: [
        ex('POST', '/api/v1/instancias/__INSTANCIA__/webhooks', 'Cadastrar webhook', 'eventos: mensagens, recibos (entregue/lido), status (conexao), chamadas, digitando, gravando_audio. incluir_base64: false nao manda a midia em base64 (so o link).', { nome: 'n8n', url: 'https://seu-n8n.com/webhook/whatsapp', eventos: ['mensagens', 'recibos', 'status'], ativo: true, incluir_base64: false }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/webhooks', 'Listar webhooks', null, null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/webhook-entregas', 'Historico de entregas', 'Ultimas entregas com status HTTP e erro, para depurar.', null),
      ],
      extra: '<h3>O que chega no seu webhook</h3><p>Exemplo de mensagem de texto recebida (evento <code>mensagens</code>):</p><pre class="dd-codigo">{\n  "evento": "mensagens",\n  "instancia_id": "ID_DA_INSTANCIA",\n  "ocorrido_em": "2026-10-16T15:57:42Z",\n  "dados": {\n    "acao": "recebida",\n    "tipo": "texto",\n    "conteudo": "Oi, quero saber o preco",\n    "mensagem_id": "3AE98CDE59F5243D6DC1",\n    "chat_jid": "556799999999@s.whatsapp.net",\n    "chat_numero": "556799999999",\n    "nome_remetente": "Maria",\n    "direcao": "entrada",\n    "grupo": false\n  }\n}</pre><p>Resposta de lista traz <code>lista_id</code> e <code>lista_titulo</code>; clique em botao traz <code>botao_id</code>; midias trazem <code>midia_id</code>, <code>mime_type</code> e o link para baixar. Cabecalhos: <code>X-Dyalog-Evento</code>, <code>X-Dyalog-Webhook-Id</code> e <code>X-Dyalog-Entrega-Id</code>.</p>',
    },
    {
      id: 'chat', titulo: 'Historico do chat', itens: [
        ex('GET', '/api/v1/instancias/__INSTANCIA__/chat/conversas', 'Conversas recentes', 'Guardadas pelo tempo configurado em CHAT_RETENTION_DAYS (padrao 7 dias).', null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/chat/mensagens?chat_jid=__NUMERO__@s.whatsapp.net', 'Mensagens de uma conversa', null, null),
      ],
    },
    {
      id: 'meta', titulo: 'API oficial (Meta)', itens: [
        ex('GET', '/api/v1/instancias/__INSTANCIA__/meta/modelos', 'Listar modelos aprovados', 'So instancias do tipo meta.', null),
        ex('POST', '/api/v1/batepapo/enviar/template', 'Enviar modelo (template)', 'Obrigatorio para iniciar conversa ou responder depois de 24h. parametros preenche as variaveis do texto, em ordem.', { numero: '__NUMERO__', nome: 'confirmacao_aula', idioma: 'pt_BR', parametros: ['Maria', '20/10 as 8h'] }, 'Na instancia Meta, texto, imagem, audio, documento, lista e botoes usam as mesmas rotas acima (dentro da janela de 24h).'),
      ],
    },
    {
      id: 'conta', titulo: 'Respostas rapidas, auditoria e status', itens: [
        ex('POST', '/api/v1/respostas-rapidas', 'Criar resposta rapida', 'Variaveis: {nome}, {saudacao}, {data}, {hora}. No chat do painel, digite / para usar.', { atalho: 'oi', texto: '{saudacao}, {nome}! Como posso ajudar?' }),
        ex('POST', '/api/v1/etiquetas', 'Criar etiqueta', null, { nome: 'Novo cliente', cor: '#22c55e' }),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/auditoria?tipo=conexao', 'Auditoria da instancia', 'Quem fez o que: conexoes, quedas, configuracoes. Filtros: tipo, severidade, ator, de, ate.', null),
        ex('GET', '/api/v1/instancias/__INSTANCIA__/chamadas', 'Chamadas em andamento', 'As chamadas recebidas tambem chegam no webhook (evento chamadas).', null),
      ],
    },
  ];

  const campos = { token: '', instancia: '', numero: '' };
  const esc = s => String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]);

  function preencher(texto) {
    return texto
      .replace(/__BASE__/g, BASE)
      .replace(/__TOKEN__/g, campos.token || 'TOKEN_DA_INSTANCIA')
      .replace(/__INSTANCIA__/g, campos.instancia || 'ID_DA_INSTANCIA')
      .replace(/__NUMERO__/g, campos.numero || '5567999999999');
  }

  function curl(item) {
    const linhas = ['curl -X ' + item.metodo + ' "__BASE__' + item.caminho + '"'];
    if (item.caminho !== '/api/v1/saude') linhas.push('  -H "X-Access-Token: __TOKEN__"');
    if (item.corpo) {
      linhas.push('  -H "Content-Type: application/json"');
      const json = JSON.stringify(item.corpo, null, 2).replace(/'/g, "'\\''");
      linhas.push("  -d '" + json + "'");
    }
    return preencher(linhas.join(' \\\n'));
  }

  function renderizar() {
    const busca = document.getElementById('dd-busca').value.trim().toLowerCase();
    const menu = [], corpo = [];
    SECOES.forEach(secao => {
      // Busca nos exemplos; se nenhum bater, vale o nome da secao.
      let itens = secao.itens.filter(i => !busca || (i.titulo + ' ' + (i.descricao || '') + ' ' + i.caminho + ' ' + (i.notas || '')).toLowerCase().includes(busca));
      if (busca && !itens.length && secao.titulo.toLowerCase().includes(busca)) itens = secao.itens;
      if (!itens.length && busca) return;
      menu.push('<a href="#' + secao.id + '">' + esc(secao.titulo) + '<small>' + itens.length + '</small></a>');
      corpo.push('<section class="dd-secao" id="' + secao.id + '"><h2>' + esc(secao.titulo) + '</h2>' +
        itens.map((item, i) => {
          const id = secao.id + '-' + i;
          return '<article class="dd-exemplo"><header><span class="dd-metodo ' + item.metodo.toLowerCase() + '">' + item.metodo + '</span><h3>' + esc(item.titulo) + '</h3></header>' +
            '<code class="dd-rota">' + esc(preencher(item.caminho)) + '</code>' +
            (item.descricao ? '<p>' + esc(item.descricao) + '</p>' : '') +
            '<div class="dd-bloco"><button type="button" class="dd-copiar" data-copiar="' + id + '">Copiar</button><pre class="dd-codigo" id="' + id + '">' + esc(curl(item)) + '</pre></div>' +
            (item.notas ? '<p class="dd-nota">' + esc(item.notas) + '</p>' : '') + '</article>';
        }).join('') + (!busca && secao.extra ? '<div class="dd-extra">' + secao.extra + '</div>' : '') + '</section>');
    });
    document.getElementById('dd-menu').innerHTML = menu.join('');
    document.getElementById('dd-secoes').innerHTML = corpo.join('') || '<p class="dd-vazio">Nada encontrado para "' + esc(busca) + '".</p>';
  }

  document.addEventListener('click', async e => {
    const botao = e.target.closest('[data-copiar]');
    if (!botao) return;
    const texto = document.getElementById(botao.dataset.copiar).textContent;
    try {
      if (navigator.clipboard && window.isSecureContext) await navigator.clipboard.writeText(texto);
      else {
        const area = document.createElement('textarea');
        area.value = texto; area.style.position = 'fixed'; area.style.opacity = '0';
        document.body.appendChild(area); area.select(); document.execCommand('copy'); area.remove();
      }
      botao.textContent = 'Copiado!';
    } catch (err) { botao.textContent = 'Nao copiou'; }
    setTimeout(() => { botao.textContent = 'Copiar'; }, 1500);
  });

  ['token', 'instancia', 'numero'].forEach(nome => {
    document.getElementById('dd-' + nome).addEventListener('input', e => {
      campos[nome] = nome === 'numero' ? e.target.value.replace(/\D/g, '') : e.target.value.trim();
      renderizar();
    });
  });
  document.getElementById('dd-busca').addEventListener('input', renderizar);
  document.getElementById('dd-base-texto').textContent = BASE;
  renderizar();
})();
