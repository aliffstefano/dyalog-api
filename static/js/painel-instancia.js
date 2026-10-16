// Abas Contatos, Perfil e Auditoria da pagina da instancia.
// Usa as funcoes globais do painel (chamar, mostrarToast, textoSeguro,
// formatarTelefone, iniciais, estadoUI, MODAIS, abrirConversaChat...).
(function () {
  'use strict';

  const base = () => '/api/v1/instancias/' + encodeURIComponent(estadoUI.instanciaSelecionada);

  // Erro de API sem passar pelo "Retorno da ultima chamada" e sem toast:
  // a propria aba mostra o motivo.
  async function buscarJSON(url, opcoes) {
    const r = await fetch(url, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opcoes || {}));
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) { const e = new Error(d?.mensagem || 'Erro na requisicao'); e.codigo = d?.erro; throw e; }
    return d?.dados;
  }

  function vazio(msg, icone) {
    return '<div class="dc-lista-vazia">' + (icone ? '<svg class="dc-ico"><use href="#' + icone + '"/></svg>' : '') + textoSeguro(msg) + '</div>';
  }

  function mensagemErro(err) {
    if (err.codigo === 'instancia_desconectada') return 'Conecte a instancia ao WhatsApp para ver esta tela.';
    if (err.codigo === 'nao_suportado_api_oficial') return 'Disponivel so nas instancias por QR code.';
    return err.message;
  }

  window.abrirAbaExtra = function (aba) {
    if (aba === 'contatos') carregarContatos(false, 1);
    if (aba === 'perfil') carregarPerfil();
    if (aba === 'auditoria') carregarAuditoriaInstancia(true);
  };

  // Contatos e Perfil nao existem na API oficial: some com os itens do menu.
  const renderizarOriginal = window.renderizarCabecalhoInstancia;
  window.renderizarCabecalhoInstancia = function (d) {
    renderizarOriginal(d);
    const oficial = d && d.tipo === 'meta';
    document.querySelectorAll('.nav-so-qr').forEach(el => el.classList.toggle('hidden', oficial));
    if (oficial && ['contatos', 'perfil'].includes(estadoUI.abaInstancia)) trocarAbaInstancia('geral');
  };

  // ── Contatos ────────────────────────────────────────────────────────────────
  const contatosUI = { tipo: '', pagina: 1, paginas: 1, timer: null, lista: [] };

  window.filtrarContatos = function (tipo) {
    contatosUI.tipo = tipo;
    document.querySelectorAll('#contatos-chips .dc-chip').forEach(b => b.classList.toggle('ativo', b.dataset.tipo === tipo));
    carregarContatos(false, 1);
  };
  window.buscarContatosDepois = function () {
    clearTimeout(contatosUI.timer);
    contatosUI.timer = setTimeout(() => carregarContatos(false, 1), 300);
  };
  window.paginaContatos = function (delta) {
    const nova = contatosUI.pagina + delta;
    if (nova < 1 || nova > contatosUI.paginas) return;
    carregarContatos(false, nova);
  };

  function parametrosContatos(pagina) {
    const p = new URLSearchParams();
    const busca = document.getElementById('contatos-busca').value.trim();
    if (busca) p.set('busca', busca);
    if (contatosUI.tipo) p.set('tipo', contatosUI.tipo);
    p.set('pagina', pagina || 1);
    p.set('por_pagina', document.getElementById('contatos-por-pagina').value);
    return p;
  }

  window.carregarContatos = async function (atualizar, pagina) {
    if (!estadoUI.instanciaSelecionada) return;
    const grade = document.getElementById('contatos-grade');
    const p = parametrosContatos(pagina || contatosUI.pagina);
    if (atualizar) p.set('atualizar', '1');
    if (!contatosUI.lista.length) grade.innerHTML = vazio('Carregando...');
    try {
      const d = await buscarJSON(base() + '/contatos?' + p.toString());
      if (!d) return;
      contatosUI.pagina = d.pagina; contatosUI.paginas = d.paginas; contatosUI.lista = d.contatos || [];
      const c = d.contagem || {};
      const contagens = { '': c.todos, salvo: undefined, pessoal: c.pessoal, grupo: c.grupo, bloqueado: c.bloqueado };
      document.querySelectorAll('#contatos-chips .dc-chip').forEach(b => {
        const n = contagens[b.dataset.tipo];
        b.querySelector('span').textContent = n ? '(' + n.toLocaleString('pt-BR') + ')' : '';
      });
      document.getElementById('contatos-total').textContent = d.total.toLocaleString('pt-BR') + ' contato' + (d.total === 1 ? '' : 's');
      document.getElementById('contatos-pagina').textContent = d.pagina + ' / ' + d.paginas;
      grade.innerHTML = contatosUI.lista.length ? contatosUI.lista.map(cartaoContato).join('') : vazio('Nenhum contato encontrado.', 'i-pessoa');
      if (atualizar) mostrarToast('Contatos atualizados.', 'success');
    } catch (err) {
      contatosUI.lista = [];
      grade.innerHTML = vazio(mensagemErro(err), 'i-pessoa');
    }
  };

  function urlAvatar(c) {
    const p = new URLSearchParams({ formato: 'arquivo' });
    if (c.tipo === 'grupo') { p.set('chat_jid', c.jid); p.set('grupo', '1'); } else if (c.numero) p.set('numero', c.numero); else p.set('chat_jid', c.jid);
    return base() + '/contatos/avatar?' + p.toString();
  }

  // Iniciais so de palavras com letra ("466 - Renata Garbosa" -> "RG").
  function iniciaisContato(nome) {
    const palavras = String(nome || '').split(/\s+/).filter(p => /\p{L}/u.test(p));
    return palavras.length ? iniciais(palavras.join(' ')) : (String(nome || '?').replace(/\D/g, '').slice(-2) || '?');
  }

  function cartaoContato(c, i) {
    const nome = c.nome || c.nome_push || (c.numero ? formatarTelefone(c.numero) : c.jid);
    const sub = c.tipo === 'grupo' ? 'Grupo' : (c.numero ? formatarTelefone(c.numero) : 'Numero oculto (LID)');
    const selos = [
      c.bloqueado ? '<span class="dc-selo perigo">Bloqueado</span>' : '',
      c.empresa ? '<span class="dc-selo">Empresa</span>' : '',
      c.tipo === 'pessoal' && !c.salvo ? '<span class="dc-selo neutro">Nao salvo</span>' : '',
    ].join('');
    const gerente = acessoGerente();
    const acoes = [
      '<button type="button" onclick="abrirChatContato(' + i + ')">Abrir conversa</button>',
      c.numero ? '<button type="button" onclick="copiarNumeroContato(' + i + ')">Copiar numero</button>' : '',
      '<button type="button" onclick="copiarJidContato(' + i + ')">Copiar JID</button>',
      gerente && c.tipo === 'pessoal' ? '<button type="button" class="' + (c.bloqueado ? '' : 'dc-texto-perigo') + '" onclick="bloquearContato(' + i + ')">' + (c.bloqueado ? 'Desbloquear' : 'Bloquear') + '</button>' : '',
    ].join('');
    return '<article class="dc-contato">' +
      '<div class="dc-contato-topo">' +
        '<div class="dc-avatar" data-iniciais="' + textoSeguro(iniciaisContato(nome)) + '"><img src="' + urlAvatar(c) + '" alt="" loading="lazy" referrerpolicy="no-referrer" onerror="avatarFalhou(this)"></div>' +
        '<div class="dc-contato-texto"><strong title="' + textoSeguro(nome) + '">' + textoSeguro(nome) + '</strong>' +
          (c.nome_push && c.nome_push !== nome ? '<small>~' + textoSeguro(c.nome_push) + '</small>' : '') + '</div>' +
        '<details class="dc-menu"><summary title="Acoes">&#8942;</summary><div class="dc-menu-itens">' + acoes + '</div></details>' +
      '</div>' +
      '<div class="dc-contato-rodape"><span><svg class="dc-ico"><use href="#' + (c.tipo === 'grupo' ? 'i-pessoa-mais' : 'i-telefone') + '"/></svg>' + textoSeguro(sub) + '</span>' + selos + '</div>' +
    '</article>';
  }

  // Menu de acoes do cartao: fecha ao clicar fora ou ao escolher uma acao.
  document.addEventListener('click', e => {
    document.querySelectorAll('details.dc-menu[open]').forEach(menu => {
      if (!menu.contains(e.target) || e.target.closest('.dc-menu-itens button')) menu.removeAttribute('open');
    });
  });

  window.avatarFalhou = function (img) {
    const pai = img.parentElement;
    pai.innerHTML = '<span>' + textoSeguro(pai.dataset.iniciais || '?') + '</span>';
  };
  window.abrirChatContato = function (i) {
    const c = contatosUI.lista[i]; if (!c) return;
    trocarAbaInstancia('chat');
    abrirConversaChat(c.jid);
  };
  window.copiarNumeroContato = async function (i) {
    try { await copiarTexto(contatosUI.lista[i].numero); mostrarToast('Numero copiado.', 'success'); } catch (e) { mostrarToast('Nao foi possivel copiar.', 'error'); }
  };
  window.copiarJidContato = async function (i) {
    try { await copiarTexto(contatosUI.lista[i].jid); mostrarToast('JID copiado.', 'success'); } catch (e) { mostrarToast('Nao foi possivel copiar.', 'error'); }
  };
  window.bloquearContato = async function (i) {
    const c = contatosUI.lista[i]; if (!c) return;
    const bloquear = !c.bloqueado;
    const nome = c.nome || formatarTelefone(c.numero) || c.jid;
    if (bloquear && !confirm('Bloquear ' + nome + '? Ele nao conseguira mais mandar mensagem nem ligar para este numero.')) return;
    try {
      await buscarJSON(base() + '/contatos/bloqueio', { method: 'POST', body: JSON.stringify({ numero: c.numero, chat_jid: c.numero ? '' : c.jid, bloquear }) });
      mostrarToast(bloquear ? 'Contato bloqueado.' : 'Contato desbloqueado.', 'success');
      carregarContatos(false, contatosUI.pagina);
    } catch (err) { mostrarToast(mensagemErro(err), 'error'); }
  };
  window.exportarContatos = function () {
    const p = parametrosContatos(1);
    p.delete('pagina'); p.delete('por_pagina');
    window.location.href = base() + '/contatos/exportar?' + p.toString();
  };

  MODAIS['verificar-numero'] = {
    kicker: 'Contatos', titulo: 'Verificar numero',
    html: '<form id="modal-form"><label><span>Numeros (um por linha, com DDI e DDD)</span><textarea id="vn-numeros" rows="5" placeholder="5567999999999"></textarea></label>' +
      '<div id="vn-resultado" class="dc-lista"></div><div id="mf-feedback" class="modal-feedback hidden"></div>' +
      '<div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Fechar</button><button type="submit" class="primary" id="mf-submit">Verificar</button></div></form>',
    init: function () { document.getElementById('vn-numeros').focus(); },
    submit: async function (e) {
      e.preventDefault();
      const numeros = document.getElementById('vn-numeros').value.split(/[\n,;]+/).map(n => n.replace(/\D/g, '')).filter(Boolean);
      const fb = document.getElementById('mf-feedback'), btn = document.getElementById('mf-submit'), out = document.getElementById('vn-resultado');
      if (!numeros.length) { fb.className = 'modal-feedback error'; fb.textContent = 'Informe ao menos um numero.'; return; }
      btn.disabled = true; btn.textContent = 'Verificando...'; fb.className = 'modal-feedback hidden';
      try {
        const lista = await buscarJSON(base() + '/contatos/verificar', { method: 'POST', body: JSON.stringify({ numeros }) }) || [];
        out.innerHTML = lista.map(r => '<div class="dc-config-linha"><div><strong>' + textoSeguro(formatarTelefone(r.numero)) + '</strong><span>' +
          (r.existe ? 'Tem WhatsApp' + (r.business ? ' Business' + (r.nome ? ' · ' + textoSeguro(r.nome) : '') : '') : 'Sem WhatsApp') +
          '</span></div><span class="dc-selo ' + (r.existe ? 'ok' : 'perigo') + '">' + (r.existe ? 'Existe' : 'Nao existe') + '</span></div>').join('') || vazio('Sem resposta do WhatsApp.');
      } catch (err) { fb.className = 'modal-feedback error'; fb.textContent = mensagemErro(err); }
      finally { btn.disabled = false; btn.textContent = 'Verificar'; }
    },
  };

  // ── Perfil ──────────────────────────────────────────────────────────────────
  const CAMPOS_PRIVACIDADE = [
    ['visto_por_ultimo', 'Visto por ultimo'],
    ['foto_perfil', 'Foto do perfil'],
    ['recado', 'Recado'],
    ['online', 'Visibilidade online'],
    ['chamadas', 'Quem pode me ligar'],
    ['grupos', 'Quem pode me adicionar em grupos'],
  ];
  const ROTULOS_VALOR = {
    all: 'Todos', contacts: 'Meus contatos', contact_blacklist: 'Meus contatos, exceto...', none: 'Ninguem',
    match_last_seen: 'Igual ao visto por ultimo', known: 'Contatos conhecidos',
  };
  const perfilUI = { dados: null };

  window.contarPerfil = function () {
    document.getElementById('perfil-nome-cont').textContent = document.getElementById('perfil-nome').value.length + '/25';
    document.getElementById('perfil-recado-cont').textContent = document.getElementById('perfil-recado').value.length + '/139';
  };

  function bloquearPerfil(msg) {
    perfilUI.dados = null;
    document.getElementById('perfil-feedback').textContent = msg;
    document.getElementById('privacidade-campos').innerHTML = vazio(msg, 'i-escudo');
    ['perfil-nome', 'perfil-recado', 'perfil-salvar', 'privacidade-salvar'].forEach(id => document.getElementById(id).disabled = true);
  }

  window.carregarPerfil = async function () {
    if (!estadoUI.instanciaSelecionada) return;
    document.getElementById('perfil-feedback').textContent = 'Carregando...';
    try {
      const d = await buscarJSON(base() + '/perfil');
      if (!d) return;
      perfilUI.dados = d;
      ['perfil-nome', 'perfil-recado', 'perfil-salvar', 'privacidade-salvar'].forEach(id => document.getElementById(id).disabled = false);
      document.getElementById('perfil-avatar').innerHTML = avatarHTML({ foto_url: d.foto_url, nome: d.nome }, d.nome);
      document.getElementById('perfil-nome').value = d.nome || '';
      document.getElementById('perfil-recado').value = d.recado || '';
      document.getElementById('perfil-telefone').textContent = formatarTelefone(d.numero) || '-';
      document.getElementById('perfil-jid').textContent = d.jid || '-';
      document.getElementById('perfil-lid').textContent = d.lid || '-';
      document.getElementById('perfil-feedback').textContent = d.business ? 'Conta WhatsApp Business' + (d.nome_empresa ? ' · ' + d.nome_empresa : '') : '';
      contarPerfil();
      renderizarPrivacidade(d);
    } catch (err) { bloquearPerfil(mensagemErro(err)); }
  };

  function renderizarPrivacidade(d) {
    const p = d.privacidade || {}, opcoes = d.opcoes_validas || {};
    const linhas = CAMPOS_PRIVACIDADE.map(([campo, rotulo]) => {
      const atual = p[campo] || '';
      const valores = opcoes[campo] || [];
      const opts = (valores.includes(atual) || !atual ? valores : [atual].concat(valores))
        .map(v => '<option value="' + v + '"' + (v === atual ? ' selected' : '') + '>' + textoSeguro(ROTULOS_VALOR[v] || v) + '</option>').join('');
      return '<div class="dc-config-linha"><div><strong>' + rotulo + '</strong></div><select data-privacidade="' + campo + '" data-atual="' + atual + '">' + opts + '</select></div>';
    });
    const leitura = p.confirmacao_leitura !== 'none';
    linhas.push('<label class="dc-opcao dc-opcao-linha"><input type="checkbox" data-privacidade="confirmacao_leitura" data-atual="' + (p.confirmacao_leitura || 'all') + '"' + (leitura ? ' checked' : '') + '><span class="dc-opcao-texto"><strong>Confirmacoes de leitura</strong><small>Enviar o tique azul ao ler mensagens recebidas.</small></span><span class="dc-chave" aria-hidden="true"></span></label>');
    document.getElementById('privacidade-campos').innerHTML = linhas.join('');
  }

  window.salvarPerfil = async function (e) {
    e.preventDefault();
    const d = perfilUI.dados; if (!d) return;
    const corpo = {};
    const nome = document.getElementById('perfil-nome').value.trim();
    const recado = document.getElementById('perfil-recado').value.trim();
    if (nome !== (d.nome || '')) corpo.nome = nome;
    if (recado !== (d.recado || '')) corpo.recado = recado;
    if (!Object.keys(corpo).length) { mostrarToast('Nada mudou.', 'info'); return; }
    const btn = document.getElementById('perfil-salvar');
    btn.disabled = true; btn.textContent = 'Salvando...';
    try {
      await buscarJSON(base() + '/perfil', { method: 'PUT', body: JSON.stringify(corpo) });
      mostrarToast('Perfil atualizado.', 'success');
      await carregarPerfil();
    } catch (err) { mostrarToast(mensagemErro(err), 'error'); }
    finally { btn.disabled = false; btn.textContent = 'Salvar'; }
  };

  window.salvarPrivacidade = async function (e) {
    e.preventDefault();
    const mudancas = {};
    document.querySelectorAll('[data-privacidade]').forEach(el => {
      const valor = el.type === 'checkbox' ? (el.checked ? 'all' : 'none') : el.value;
      if (valor !== el.dataset.atual) mudancas[el.dataset.privacidade] = valor;
    });
    if (!Object.keys(mudancas).length) { mostrarToast('Nada mudou.', 'info'); return; }
    const btn = document.getElementById('privacidade-salvar');
    btn.disabled = true; btn.textContent = 'Salvando...';
    try {
      const privacidade = await buscarJSON(base() + '/privacidade', { method: 'PUT', body: JSON.stringify(mudancas) });
      if (perfilUI.dados && privacidade) { perfilUI.dados.privacidade = privacidade; renderizarPrivacidade(perfilUI.dados); }
      mostrarToast('Privacidade atualizada.', 'success');
    } catch (err) { mostrarToast(mensagemErro(err), 'error'); }
    finally { btn.disabled = false; btn.textContent = 'Salvar'; }
  };

  window.trocarFotoPerfil = function () {
    const input = document.getElementById('perfil-foto-arquivo');
    const arquivo = input.files && input.files[0];
    input.value = '';
    if (!arquivo) return;
    if (arquivo.size > 10 * 1024 * 1024) { mostrarToast('Imagem acima de 10 MB.', 'error'); return; }
    const leitor = new FileReader();
    leitor.onload = async function () {
      mostrarToast('Enviando foto...', 'info');
      try {
        await buscarJSON(base() + '/perfil/foto', { method: 'PUT', body: JSON.stringify({ imagem: leitor.result }) });
        mostrarToast('Foto atualizada.', 'success');
        await carregarPerfil();
      } catch (err) { mostrarToast(mensagemErro(err), 'error'); }
    };
    leitor.readAsDataURL(arquivo);
  };

  window.removerFotoPerfil = async function () {
    if (!confirm('Remover a foto do perfil desta conta de WhatsApp?')) return;
    try {
      await buscarJSON(base() + '/perfil/foto', { method: 'DELETE' });
      mostrarToast('Foto removida.', 'success');
      await carregarPerfil();
    } catch (err) { mostrarToast(mensagemErro(err), 'error'); }
  };

  // ── Auditoria ───────────────────────────────────────────────────────────────
  const auditoriaUI = { deslocamento: 0, timer: null, linhas: 0 };
  const ICONES_SEVERIDADE = { sucesso: '&#10003;', info: 'i', aviso: '!', erro: '&#10005;' };
  const ORIGENS = { usuario: 'Painel', master: 'Token master', token: 'API (token)', sistema: 'Automatico' };

  window.buscarAuditoriaDepois = function () {
    clearTimeout(auditoriaUI.timer);
    auditoriaUI.timer = setTimeout(() => carregarAuditoriaInstancia(true), 350);
  };
  window.limparFiltrosAuditoria = function () {
    ['aud-tipo', 'aud-severidade', 'aud-ator', 'aud-de', 'aud-ate'].forEach(id => document.getElementById(id).value = '');
    carregarAuditoriaInstancia(true);
  };

  window.carregarAuditoriaInstancia = async function (reiniciar) {
    if (!estadoUI.instanciaSelecionada) return;
    if (reiniciar) auditoriaUI.deslocamento = 0;
    const corpo = document.getElementById('aud-linhas');
    const p = new URLSearchParams({ limite: '50', deslocamento: String(auditoriaUI.deslocamento) });
    [['tipo', 'aud-tipo'], ['severidade', 'aud-severidade'], ['ator', 'aud-ator'], ['de', 'aud-de'], ['ate', 'aud-ate']].forEach(([k, id]) => {
      const v = document.getElementById(id).value.trim(); if (v) p.set(k, v);
    });
    try {
      const d = await buscarJSON(base() + '/auditoria?' + p.toString());
      if (!d) return;
      const html = (d.registros || []).map(linhaAuditoria).join('');
      if (reiniciar) { corpo.innerHTML = html || '<tr><td colspan="4">' + vazio('Nada registrado com esses filtros.', 'i-historico') + '</td></tr>'; auditoriaUI.linhas = 0; }
      else corpo.insertAdjacentHTML('beforeend', html);
      auditoriaUI.linhas += (d.registros || []).length;
      auditoriaUI.deslocamento += (d.registros || []).length;
      document.getElementById('aud-mais').classList.toggle('hidden', !d.tem_mais);
      document.getElementById('aud-resumo').textContent = auditoriaUI.linhas ? auditoriaUI.linhas + ' registro' + (auditoriaUI.linhas === 1 ? '' : 's') + (d.tem_mais ? ' (ha mais)' : '') : '';
    } catch (err) {
      corpo.innerHTML = '<tr><td colspan="4">' + vazio(mensagemErro(err)) + '</td></tr>';
    }
  };

  function linhaAuditoria(r) {
    const sev = r.severidade || 'info';
    const origem = ORIGENS[r.ator_tipo] || r.ator_tipo || '-';
    return '<tr>' +
      '<td class="dc-nowrap">' + textoSeguro(formatarData(r.criado_em)) + '</td>' +
      '<td>' + (r.ator_tipo === 'sistema' ? '<span class="helper-text">Sistema</span>' : textoSeguro(r.ator_nome || '-')) + '</td>' +
      '<td><span class="dc-aud-acao"><span class="dc-aud-icone ' + sev + '">' + (ICONES_SEVERIDADE[sev] || 'i') + '</span>' + textoSeguro(r.descricao || r.acao) + '</span></td>' +
      '<td><span>' + textoSeguro(origem) + '</span>' + (r.ip ? '<small class="helper-text dc-mono"> ' + textoSeguro(r.ip) + '</small>' : '') + '</td>' +
    '</tr>';
  }
})();
