// Chat: gravar audio, respostas rapidas (/atalho), etiquetas das conversas e a
// pagina "Respostas rapidas".
(function () {
  'use strict';

  async function api(url, opcoes) {
    const r = await fetch(url, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opcoes || {}));
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) { const e = new Error(d?.mensagem || 'Erro na requisicao'); e.codigo = d?.erro; throw e; }
    return d?.dados;
  }
  const baseInst = () => '/api/v1/instancias/' + encodeURIComponent(estadoUI.instanciaSelecionada);

  // ── Dados compartilhados (respostas e etiquetas da conta) ─────────────────
  const dados = { respostas: null, etiquetas: null, porConversa: {}, instanciaEtiquetas: '' };
  async function carregarRespostas(forcar) {
    if (dados.respostas && !forcar) return dados.respostas;
    try { dados.respostas = (await api('/api/v1/respostas-rapidas'))?.respostas || []; } catch (e) { dados.respostas = []; }
    return dados.respostas;
  }
  async function carregarEtiquetas(forcar) {
    if (dados.etiquetas && !forcar) return dados.etiquetas;
    try { dados.etiquetas = (await api('/api/v1/etiquetas'))?.etiquetas || []; } catch (e) { dados.etiquetas = []; }
    return dados.etiquetas;
  }
  async function carregarEtiquetasConversas() {
    if (!estadoUI.instanciaSelecionada) return;
    try {
      dados.porConversa = (await api(baseInst() + '/chat/etiquetas'))?.conversas || {};
      dados.instanciaEtiquetas = estadoUI.instanciaSelecionada;
    } catch (e) { dados.porConversa = {}; }
    window.versaoEtiquetasChat = (window.versaoEtiquetasChat || 0) + 1;
  }

  // ── Etiquetas no chat ─────────────────────────────────────────────────────
  window.etiquetasDaConversa = jid => dados.porConversa[jid] || [];
  function etiquetaPorID(id) { return (dados.etiquetas || []).find(e => e.id === id); }
  function chipEtiqueta(e) {
    return '<span class="dc-etiqueta" style="--cor:' + textoSeguro(e.cor) + '">' + textoSeguro(e.nome) + '</span>';
  }
  window.etiquetasConversaHTML = function (jid) {
    const lista = etiquetasDaConversa(jid).map(etiquetaPorID).filter(Boolean);
    return lista.length ? '<span class="dc-etiquetas-linha">' + lista.map(chipEtiqueta).join('') + '</span>' : '';
  };
  window.atualizarEtiquetasTopo = function () {
    const wrap = document.getElementById('chat-etiquetar-wrap'), topo = document.getElementById('chat-topo-etiquetas');
    if (!wrap) return;
    const temEtiquetas = (dados.etiquetas || []).length > 0;
    wrap.classList.toggle('hidden', !chatUI.chat || !temEtiquetas);
    topo.innerHTML = chatUI.chat ? etiquetasDaConversa(chatUI.chat).map(etiquetaPorID).filter(Boolean).map(chipEtiqueta).join('') : '';
    const filtro = document.getElementById('chat-filtro-etiqueta'), atual = filtro.value;
    filtro.innerHTML = '<option value="">Todas as etiquetas</option>' + (dados.etiquetas || []).map(e => '<option value="' + textoSeguro(e.id) + '">' + textoSeguro(e.nome) + '</option>').join('');
    filtro.value = atual;
    filtro.classList.toggle('hidden', !temEtiquetas);
  };
  window.abrirEtiquetasConversa = function (ev) {
    ev.stopPropagation();
    const menu = document.getElementById('chat-etiquetar-menu');
    if (!menu.classList.contains('hidden')) { menu.classList.add('hidden'); return; }
    const marcadas = etiquetasDaConversa(chatUI.chat);
    menu.innerHTML = (dados.etiquetas || []).map(e => '<label class="dc-etiqueta-opcao"><input type="checkbox" value="' + textoSeguro(e.id) + '"' + (marcadas.includes(e.id) ? ' checked' : '') + '>' + chipEtiqueta(e) + '</label>').join('') +
      '<button type="button" class="dc-link" onclick="trocarAba(\'respostas-rapidas\')">Gerenciar etiquetas</button>';
    menu.classList.remove('hidden');
    menu.querySelectorAll('input').forEach(i => i.addEventListener('change', salvarEtiquetasConversa));
  };
  async function salvarEtiquetasConversa() {
    const jid = chatUI.chat;
    const escolhidas = [...document.querySelectorAll('#chat-etiquetar-menu input:checked')].map(i => i.value);
    try {
      await api(baseInst() + '/chat/etiquetas', { method: 'PUT', body: JSON.stringify({ chat_jid: jid, etiquetas: escolhidas }) });
      dados.porConversa[jid] = escolhidas;
      window.versaoEtiquetasChat = (window.versaoEtiquetasChat || 0) + 1;
      atualizarEtiquetasTopo();
      renderizarConversasChat();
    } catch (err) { mostrarToast(err.message, 'error'); }
  }
  document.addEventListener('click', e => {
    const menu = document.getElementById('chat-etiquetar-menu');
    if (menu && !menu.classList.contains('hidden') && !e.target.closest('#chat-etiquetar-wrap')) menu.classList.add('hidden');
  });

  // Ao abrir o chat: carrega respostas, etiquetas e as marcacoes desta instancia.
  const abrirOriginal = window.abrirAbaExtra;
  window.abrirAbaExtra = async function (aba) {
    if (abrirOriginal) abrirOriginal(aba);
    if (aba !== 'chat') { pararGravacao(false); return; }
    await Promise.all([carregarRespostas(false), carregarEtiquetas(false), dados.instanciaEtiquetas === estadoUI.instanciaSelecionada ? null : carregarEtiquetasConversas()]);
    atualizarEtiquetasTopo();
    chatUI.assinaturaLista = '';
    renderizarConversasChat();
  };

  // ── Respostas rapidas no campo do chat ────────────────────────────────────
  const rapidasUI = { aberta: false, itens: [], indice: 0 };
  function nomeParaResposta() {
    const conversa = chatUI.conversas.find(c => c.chat_jid === chatUI.chat);
    const nome = conversa && conversa.nome ? conversa.nome : '';
    return nome && !/^\+?\d/.test(nome) ? nome.split(/\s+/)[0] : '';
  }
  function aplicarVariaveis(texto) {
    const agora = new Date();
    const hora = agora.getHours();
    return texto
      .replace(/\{nome\}/gi, nomeParaResposta())
      .replace(/\{saudacao\}/gi, hora < 12 ? 'Bom dia' : hora < 18 ? 'Boa tarde' : 'Boa noite')
      .replace(/\{data\}/gi, agora.toLocaleDateString('pt-BR'))
      .replace(/\{hora\}/gi, agora.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }))
      .replace(/ {2,}/g, ' ').replace(/ ([,.!?])/g, '$1');
  }
  function fecharRapidas() {
    rapidasUI.aberta = false;
    document.getElementById('chat-rapidas').classList.add('hidden');
  }
  function renderizarRapidas() {
    const caixa = document.getElementById('chat-rapidas');
    if (!rapidasUI.itens.length) {
      caixa.innerHTML = '<div class="dc-rapida-vazia">' + ((dados.respostas || []).length ? 'Nenhuma resposta com esse atalho.' : 'Nenhuma resposta rapida ainda.') + ' <button type="button" class="dc-link" onclick="trocarAba(\'respostas-rapidas\')">Criar</button></div>';
    } else {
      caixa.innerHTML = rapidasUI.itens.map((r, i) => '<button type="button" class="dc-rapida' + (i === rapidasUI.indice ? ' ativa' : '') + '" data-i="' + i + '"><strong>/' + textoSeguro(r.atalho) + '</strong><span>' + textoSeguro(r.texto) + '</span></button>').join('');
      caixa.querySelectorAll('[data-i]').forEach(b => b.addEventListener('mousedown', ev => { ev.preventDefault(); usarRapida(Number(b.dataset.i)); }));
    }
    caixa.classList.remove('hidden');
    rapidasUI.aberta = true;
  }
  function usarRapida(i) {
    const r = rapidasUI.itens[i]; if (!r) return;
    const campo = document.getElementById('chat-texto');
    campo.value = aplicarVariaveis(r.texto);
    fecharRapidas();
    ajustarAlturaChat();
    campo.focus();
  }
  function aoDigitar() {
    const campo = document.getElementById('chat-texto');
    const m = /^\/(\S*)$/.exec(campo.value);
    if (!m) { if (rapidasUI.aberta) fecharRapidas(); return; }
    const termo = m[1].toLowerCase();
    carregarRespostas(false).then(lista => {
      rapidasUI.itens = lista.filter(r => r.atalho.includes(termo) || r.texto.toLowerCase().includes(termo)).slice(0, 8);
      rapidasUI.indice = 0;
      renderizarRapidas();
    });
  }
  function aoTeclar(e) {
    if (!rapidasUI.aberta) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault(); e.stopImmediatePropagation();
      const n = rapidasUI.itens.length; if (!n) return;
      rapidasUI.indice = (rapidasUI.indice + (e.key === 'ArrowDown' ? 1 : n - 1)) % n;
      renderizarRapidas();
    } else if ((e.key === 'Enter' || e.key === 'Tab') && rapidasUI.itens.length) {
      e.preventDefault(); e.stopImmediatePropagation();
      usarRapida(rapidasUI.indice);
    } else if (e.key === 'Escape') {
      e.stopImmediatePropagation(); fecharRapidas();
    }
  }
  const campoChat = document.getElementById('chat-texto');
  campoChat.addEventListener('input', aoDigitar);
  campoChat.addEventListener('keydown', aoTeclar, true);
  campoChat.addEventListener('blur', () => setTimeout(fecharRapidas, 150));

  // ── Gravar audio (nota de voz) ────────────────────────────────────────────
  const gravUI = { rec: null, partes: [], inicio: 0, timer: null, stream: null, cancelado: false };
  function tipoGravacao() {
    return ['audio/ogg;codecs=opus', 'audio/webm;codecs=opus', 'audio/webm'].find(t => window.MediaRecorder && MediaRecorder.isTypeSupported(t)) || '';
  }
  function mostrarEstadoGravacao(texto) {
    const el = document.getElementById('chat-anexo-nome');
    el.classList.toggle('hidden', !texto);
    el.classList.toggle('gravando', !!texto);
    el.innerHTML = texto || '';
  }
  window.alternarGravacaoChat = async function () {
    if (gravUI.rec) { pararGravacao(true); return; }
    if (!chatUI.chat) { mostrarToast('Abra uma conversa primeiro.', 'error'); return; }
    const tipo = tipoGravacao();
    if (!tipo || !navigator.mediaDevices) { mostrarToast('Este navegador nao grava audio.', 'error'); return; }
    try {
      gravUI.stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation: true, noiseSuppression: true } });
    } catch (e) { mostrarToast('Sem permissao para usar o microfone.', 'error'); return; }
    gravUI.partes = []; gravUI.cancelado = false;
    gravUI.rec = new MediaRecorder(gravUI.stream, { mimeType: tipo });
    gravUI.rec.ondataavailable = ev => { if (ev.data.size) gravUI.partes.push(ev.data); };
    gravUI.rec.onstop = enviarGravacao;
    gravUI.rec.start(250);
    gravUI.inicio = Date.now();
    document.getElementById('chat-gravar').classList.add('gravando');
    const atualizar = () => {
      const s = Math.floor((Date.now() - gravUI.inicio) / 1000);
      mostrarEstadoGravacao('<i class="dc-rec"></i>Gravando ' + Math.floor(s / 60) + ':' + String(s % 60).padStart(2, '0') + ' · clique no microfone para enviar <button type="button" class="dc-link" onclick="cancelarGravacaoChat()">Cancelar</button>');
      if (s >= 300) pararGravacao(true); // 5 minutos
    };
    atualizar();
    gravUI.timer = setInterval(atualizar, 500);
  };
  window.cancelarGravacaoChat = () => pararGravacao(false);
  function pararGravacao(enviar) {
    if (!gravUI.rec) return;
    gravUI.cancelado = !enviar;
    clearInterval(gravUI.timer);
    try { gravUI.rec.stop(); } catch (e) {}
    gravUI.stream?.getTracks().forEach(t => t.stop());
    document.getElementById('chat-gravar').classList.remove('gravando');
    mostrarEstadoGravacao(enviar ? 'Enviando audio...' : '');
  }
  async function enviarGravacao() {
    const rec = gravUI.rec; gravUI.rec = null;
    if (gravUI.cancelado || !gravUI.partes.length) { mostrarEstadoGravacao(''); return; }
    const duracao = Math.max(1, Math.round((Date.now() - gravUI.inicio) / 1000));
    const blob = new Blob(gravUI.partes, { type: rec.mimeType });
    const destino = { instancia: estadoUI.instanciaSelecionada };
    if (chatUI.chat.endsWith('@g.us')) destino.chat_jid = chatUI.chat; else destino.numero = chatUI.chat.split('@')[0];
    try {
      const base64 = await lerArquivoBase64(new File([blob], 'audio', { type: rec.mimeType }));
      const ogg = rec.mimeType.includes('ogg');
      await chamar('/api/v1/batepapo/enviar/audio', { method: 'POST', body: JSON.stringify(Object.assign(destino, {
        arquivo_base64: base64, nome_arquivo: ogg ? 'audio.ogg' : 'audio.webm', mime_type: ogg ? 'audio/ogg' : 'audio/webm', ptt: true, duracao_segundos: duracao,
      })) });
      chatUI.assinatura = '';
      await carregarChat();
    } catch (err) { mostrarToast('Audio nao enviado: ' + err.message, 'error'); }
    finally { mostrarEstadoGravacao(''); }
  }
  document.addEventListener('keydown', e => { if (e.key === 'Escape' && gravUI.rec) pararGravacao(false); });

  // ── Pagina "Respostas rapidas" ────────────────────────────────────────────
  async function renderizarPagina() {
    const [respostas, etiquetas] = await Promise.all([carregarRespostas(true), carregarEtiquetas(true)]);
    const lista = document.getElementById('rr-lista');
    lista.innerHTML = respostas.length ? respostas.map((r, i) => '<div class="dc-config-linha"><div class="dc-rr-texto"><strong>/' + textoSeguro(r.atalho) + '</strong><span>' + textoSeguro(r.texto) + '</span></div>' +
      '<div class="dc-acoes-topo"><button class="ghost small" data-editar-rr="' + i + '">Editar</button><button class="ghost small dc-texto-perigo" data-excluir-rr="' + i + '">Excluir</button></div></div>').join('')
      : '<div class="dc-lista-vazia"><svg class="dc-ico"><use href="#i-raio"/></svg>Nenhuma resposta rapida. Crie a primeira: ex. atalho <strong>/oi</strong> com "{saudacao}, {nome}! Como posso ajudar?"</div>';
    lista.querySelectorAll('[data-editar-rr]').forEach(b => b.addEventListener('click', () => abrirRespostaRapida(respostas[Number(b.dataset.editarRr)])));
    lista.querySelectorAll('[data-excluir-rr]').forEach(b => b.addEventListener('click', async () => {
      const r = respostas[Number(b.dataset.excluirRr)];
      if (!confirm('Excluir a resposta /' + r.atalho + '?')) return;
      try { await api('/api/v1/respostas-rapidas/' + encodeURIComponent(r.id), { method: 'DELETE' }); mostrarToast('Resposta excluida.', 'success'); renderizarPagina(); } catch (err) { mostrarToast(err.message, 'error'); }
    }));
    const et = document.getElementById('et-lista');
    et.innerHTML = etiquetas.length ? etiquetas.map((e, i) => '<span class="dc-etiqueta dc-etiqueta-grande" style="--cor:' + textoSeguro(e.cor) + '">' + textoSeguro(e.nome) + '<button type="button" title="Excluir" data-excluir-et="' + i + '">&times;</button></span>').join('') : '<span class="helper-text">Nenhuma etiqueta ainda.</span>';
    et.querySelectorAll('[data-excluir-et]').forEach(b => b.addEventListener('click', async () => {
      const e = etiquetas[Number(b.dataset.excluirEt)];
      if (!confirm('Excluir a etiqueta "' + e.nome + '"? Ela sai de todas as conversas.')) return;
      try { await api('/api/v1/etiquetas/' + encodeURIComponent(e.id), { method: 'DELETE' }); dados.instanciaEtiquetas = ''; renderizarPagina(); } catch (err) { mostrarToast(err.message, 'error'); }
    }));
  }
  window.criarEtiqueta = async function (e) {
    e.preventDefault();
    const nome = document.getElementById('et-nome').value.trim(); if (!nome) return;
    try {
      await api('/api/v1/etiquetas', { method: 'POST', body: JSON.stringify({ nome, cor: document.getElementById('et-cor').value }) });
      document.getElementById('et-nome').value = '';
      renderizarPagina();
    } catch (err) { mostrarToast(err.message, 'error'); }
  };

  const rrUI = { editando: null };
  window.abrirRespostaRapida = function (r) { rrUI.editando = r; abrirModal('resposta-rapida'); };
  MODAIS['resposta-rapida'] = {
    kicker: 'Respostas rapidas', titulo: 'Resposta rapida', semInstancia: true,
    html: '<form id="modal-form"><label><span>Atalho (sem espaco; no chat digite /atalho)</span><input id="rr-atalho" maxlength="30" placeholder="oi" autocomplete="off"></label>' +
      '<label><span>Texto</span><textarea id="rr-texto" rows="5" maxlength="4096" placeholder="{saudacao}, {nome}! Como posso ajudar?"></textarea></label>' +
      '<p class="helper-text">Variaveis: <code>{nome}</code> (primeiro nome do contato), <code>{saudacao}</code> (bom dia/boa tarde/boa noite), <code>{data}</code>, <code>{hora}</code>.</p>' +
      '<div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar</button></div></form>',
    init: function () {
      const r = rrUI.editando;
      document.getElementById('modal-titulo').textContent = r ? 'Editar resposta rapida' : 'Nova resposta rapida';
      document.getElementById('rr-atalho').value = r ? r.atalho : '';
      document.getElementById('rr-texto').value = r ? r.texto : '';
      document.getElementById('rr-atalho').focus();
    },
    submit: async function (e) {
      e.preventDefault();
      const r = rrUI.editando, fb = document.getElementById('mf-feedback');
      const corpo = { atalho: document.getElementById('rr-atalho').value, texto: document.getElementById('rr-texto').value };
      try {
        await api('/api/v1/respostas-rapidas' + (r ? '/' + encodeURIComponent(r.id) : ''), { method: r ? 'PUT' : 'POST', body: JSON.stringify(corpo) });
        fecharModal(); mostrarToast('Resposta salva.', 'success'); renderizarPagina();
      } catch (err) { fb.className = 'modal-feedback error'; fb.textContent = err.message; }
    },
  };

  const principalOriginal = window.abrirAbaPrincipalExtra2;
  window.abrirAbaPrincipalExtra2 = function (aba) {
    if (principalOriginal) principalOriginal(aba);
    if (aba === 'respostas-rapidas') renderizarPagina();
  };
})();
