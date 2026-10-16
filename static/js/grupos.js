// Aba Grupos da instancia.
(function () {
  'use strict';
  const base = () => '/api/v1/instancias/' + encodeURIComponent(estadoUI.instanciaSelecionada);
  const gruposUI = { lista: [], atual: null, link: '' };

  async function api(url, opcoes) {
    const r = await fetch(url, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opcoes || {}));
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) { const e = new Error(d?.mensagem || 'Erro na requisicao'); e.codigo = d?.erro; throw e; }
    return d?.dados;
  }
  function vazio(msg) { return '<div class="dc-lista-vazia">' + textoSeguro(msg) + '</div>'; }
  function erroTexto(err) {
    if (err.codigo === 'instancia_desconectada') return 'Conecte a instancia ao WhatsApp para ver os grupos.';
    return err.message;
  }
  const gerente = () => acessoGerente();

  const abrirOriginal = window.abrirAbaExtra;
  window.abrirAbaExtra = function (aba) {
    if (abrirOriginal) abrirOriginal(aba);
    if (aba === 'grupos') { gruposUI.atual = null; carregarGrupos(); }
  };

  window.carregarGrupos = async function () {
    const itens = document.getElementById('grupos-itens');
    if (!gruposUI.lista.length) itens.innerHTML = vazio('Carregando...');
    try {
      const d = await api(base() + '/grupos');
      if (!d) return;
      gruposUI.lista = d.grupos || [];
      renderizarGrupos();
      if (gruposUI.atual) abrirGrupo(gruposUI.atual.jid);
    } catch (err) { gruposUI.lista = []; itens.innerHTML = vazio(erroTexto(err)); }
  };

  window.renderizarGrupos = function () {
    const itens = document.getElementById('grupos-itens');
    const busca = document.getElementById('grupos-busca').value.trim().toLowerCase();
    const lista = gruposUI.lista.filter(g => !busca || (g.nome || '').toLowerCase().includes(busca));
    if (!lista.length) { itens.innerHTML = vazio(busca ? 'Nenhum grupo encontrado.' : 'Esta conta nao participa de nenhum grupo.'); return; }
    itens.innerHTML = lista.map(g => '<button type="button" class="dc-chat-conversa' + (gruposUI.atual && gruposUI.atual.jid === g.jid ? ' ativa' : '') + '" data-grupo="' + textoSeguro(g.jid) + '">' +
      '<span class="dc-avatar dc-avatar-mini"><span>' + textoSeguro(iniciais(g.nome || 'G')) + '</span></span>' +
      '<span class="dc-linha-texto"><strong>' + textoSeguro(g.nome || g.jid) + '</strong><span>' + g.participantes + ' participante' + (g.participantes === 1 ? '' : 's') + (g.comunidade ? ' · Comunidade' : '') + '</span></span>' +
      (g.sou_admin ? '<small><span class="dc-selo">Admin</span></small>' : '') + '</button>').join('');
    itens.querySelectorAll('[data-grupo]').forEach(b => b.addEventListener('click', () => abrirGrupo(b.dataset.grupo)));
  };

  async function abrirGrupo(jid) {
    const caixa = document.getElementById('grupo-detalhe');
    if (!gruposUI.atual || gruposUI.atual.jid !== jid) { caixa.innerHTML = vazio('Carregando...'); gruposUI.link = ''; }
    gruposUI.atual = { jid };
    renderizarGrupos();
    document.getElementById('grupos-card').classList.add('com-grupo');
    try {
      const g = await api(base() + '/grupos/' + encodeURIComponent(jid));
      if (!g || !gruposUI.atual || gruposUI.atual.jid !== jid) return;
      gruposUI.atual = g;
      renderizarDetalhe();
    } catch (err) { caixa.innerHTML = vazio(erroTexto(err)); }
  }
  window.abrirGrupoPainel = abrirGrupo;

  function renderizarDetalhe() {
    const g = gruposUI.atual, caixa = document.getElementById('grupo-detalhe');
    const admin = g.sou_admin && gerente();
    const participantes = (g.lista_participantes || []).map((p, i) => {
      const nome = p.eu ? 'Voce' : (p.nome || (p.numero ? formatarTelefone(p.numero) : 'Numero oculto'));
      const acoes = admin && !p.eu ? '<details class="dc-menu"><summary title="Acoes">&#8942;</summary><div class="dc-menu-itens">' +
        (p.admin ? (p.super_admin ? '' : '<button type="button" data-part="rebaixar" data-i="' + i + '">Remover de admin</button>') : '<button type="button" data-part="promover" data-i="' + i + '">Tornar admin</button>') +
        '<button type="button" class="dc-texto-perigo" data-part="remover" data-i="' + i + '">Remover do grupo</button></div></details>' : '';
      return '<div class="dc-participante"><span class="dc-avatar dc-avatar-mini"><span>' + textoSeguro(p.eu ? 'EU' : iniciais(p.nome || (p.numero || '').slice(-2))) + '</span></span>' +
        '<span class="dc-linha-texto"><strong>' + textoSeguro(nome) + '</strong><span>' + textoSeguro(p.numero ? formatarTelefone(p.numero) : p.jid) + '</span></span>' +
        (p.super_admin ? '<span class="dc-selo">Dono</span>' : p.admin ? '<span class="dc-selo">Admin</span>' : '') + acoes + '</div>';
    }).join('');
    caixa.innerHTML =
      '<header class="dc-grupo-topo"><button class="ghost small dc-botao-icone dc-chat-voltar" onclick="fecharGrupoPainel()" title="Grupos"><svg class="dc-ico"><use href="#i-voltar"/></svg></button>' +
        '<div class="dc-linha-texto"><strong>' + textoSeguro(g.nome || g.jid) + '</strong><span class="dc-mono">' + textoSeguro(g.jid) + '</span></div>' +
        '<button class="ghost small" onclick="conversarNoGrupo()"><svg class="dc-ico"><use href="#i-mensagem"/></svg>Abrir conversa</button></header>' +
      '<div class="dc-grupo-corpo">' +
        (admin ? '<form class="dc-form-config" onsubmit="salvarGrupo(event)">' +
          '<label class="dc-campo"><span>Nome</span><input id="grupo-nome" maxlength="100" value="' + textoSeguro(g.nome) + '"></label>' +
          '<label class="dc-campo"><span>Descricao</span><textarea id="grupo-descricao" rows="3" maxlength="2048">' + textoSeguro(g.descricao || '') + '</textarea></label>' +
          '<div class="dc-opcoes">' +
            '<label class="dc-opcao"><input type="checkbox" id="grupo-so-admins-enviam"' + (g.so_admins_enviam ? ' checked' : '') + '><span class="dc-opcao-texto"><strong>So admins enviam mensagens</strong><small>Os outros participantes so leem.</small></span><span class="dc-chave" aria-hidden="true"></span></label>' +
            '<label class="dc-opcao"><input type="checkbox" id="grupo-so-admins-editam"' + (g.so_admins_editam ? ' checked' : '') + '><span class="dc-opcao-texto"><strong>So admins editam o grupo</strong><small>Nome, foto e descricao.</small></span><span class="dc-chave" aria-hidden="true"></span></label>' +
          '</div>' +
          '<div class="dc-form-rodape"><label class="ghost small dc-botao-arquivo"><input type="file" accept="image/jpeg,image/png,image/gif" onchange="trocarFotoGrupo(this)"><svg class="dc-ico"><use href="#i-camera"/></svg>Trocar foto</label><button class="primary small" type="submit">Salvar</button></div>' +
        '</form>' : (g.descricao ? '<p class="dc-grupo-descricao">' + textoSeguro(g.descricao) + '</p>' : '') +
          '<p class="helper-text">' + (g.sou_admin ? 'Seu papel no painel nao permite alterar o grupo.' : 'Esta conta nao e admin deste grupo: da para ver os participantes, mas nao alterar.') + '</p>') +
        '<div class="dc-card-topo"><h3>Link de convite</h3></div>' +
        '<div class="dc-campo-acao"><input readonly id="grupo-link" value="' + textoSeguro(gruposUI.link) + '" placeholder="' + (g.sou_admin ? 'Clique em Gerar' : 'So admins veem o link') + '">' +
          '<button class="ghost small" onclick="linkGrupo(false)"' + (g.sou_admin ? '' : ' disabled') + '>Gerar</button>' +
          '<button class="ghost small" onclick="copiarLinkGrupo()">Copiar</button>' +
          (admin ? '<button class="ghost small dc-texto-perigo" onclick="linkGrupo(true)" title="Invalida o link anterior">Redefinir</button>' : '') + '</div>' +
        '<div class="dc-card-topo"><h3>Participantes (' + (g.lista_participantes || []).length + ')</h3></div>' +
        (admin ? '<form class="dc-campo-acao" onsubmit="adicionarParticipantes(event)"><input id="grupo-adicionar" placeholder="Numeros com DDI, separados por virgula"><button class="primary small" type="submit">Adicionar</button></form><div id="grupo-resultado" class="dc-lista"></div>' : '') +
        '<div class="dc-participantes">' + participantes + '</div>' +
        (gerente() ? '<div class="dc-perigo-linha"><button class="dc-botao-perigo" onclick="sairGrupo()">Sair do grupo</button></div>' : '') +
      '</div>';
    caixa.querySelectorAll('[data-part]').forEach(b => b.addEventListener('click', () => acaoParticipante(b.dataset.part, Number(b.dataset.i))));
  }

  window.fecharGrupoPainel = function () {
    gruposUI.atual = null;
    document.getElementById('grupos-card').classList.remove('com-grupo');
    document.getElementById('grupo-detalhe').innerHTML = vazio('Escolha um grupo ao lado.');
    renderizarGrupos();
  };
  window.conversarNoGrupo = function () {
    const jid = gruposUI.atual && gruposUI.atual.jid; if (!jid) return;
    trocarAbaInstancia('chat'); abrirConversaChat(jid);
  };

  window.salvarGrupo = async function (e) {
    e.preventDefault();
    const g = gruposUI.atual, corpo = {};
    const nome = document.getElementById('grupo-nome').value.trim(), desc = document.getElementById('grupo-descricao').value.trim();
    const enviam = document.getElementById('grupo-so-admins-enviam').checked, editam = document.getElementById('grupo-so-admins-editam').checked;
    if (nome !== (g.nome || '')) corpo.nome = nome;
    if (desc !== (g.descricao || '')) corpo.descricao = desc;
    if (enviam !== !!g.so_admins_enviam) corpo.so_admins_enviam = enviam;
    if (editam !== !!g.so_admins_editam) corpo.so_admins_editam = editam;
    if (!Object.keys(corpo).length) { mostrarToast('Nada mudou.', 'info'); return; }
    try {
      await api(base() + '/grupos/' + encodeURIComponent(g.jid), { method: 'PUT', body: JSON.stringify(corpo) });
      mostrarToast('Grupo atualizado.', 'success');
      await carregarGrupos();
    } catch (err) { mostrarToast(erroTexto(err), 'error'); }
  };

  window.trocarFotoGrupo = function (input) {
    const arquivo = input.files && input.files[0]; input.value = '';
    if (!arquivo) return;
    const leitor = new FileReader();
    leitor.onload = async () => {
      try {
        await api(base() + '/grupos/' + encodeURIComponent(gruposUI.atual.jid) + '/foto', { method: 'PUT', body: JSON.stringify({ imagem: leitor.result }) });
        mostrarToast('Foto do grupo atualizada.', 'success');
      } catch (err) { mostrarToast(erroTexto(err), 'error'); }
    };
    leitor.readAsDataURL(arquivo);
  };

  window.linkGrupo = async function (redefinir) {
    if (redefinir && !confirm('Redefinir o link? O link atual para de funcionar.')) return;
    try {
      const d = await api(base() + '/grupos/' + encodeURIComponent(gruposUI.atual.jid) + '/convite' + (redefinir ? '?redefinir=1' : ''));
      gruposUI.link = d?.link || '';
      document.getElementById('grupo-link').value = gruposUI.link;
      if (redefinir) mostrarToast('Link redefinido.', 'success');
    } catch (err) { mostrarToast(erroTexto(err), 'error'); }
  };
  window.copiarLinkGrupo = async function () {
    const link = document.getElementById('grupo-link').value;
    if (!link) { await linkGrupo(false); }
    try { await copiarTexto(document.getElementById('grupo-link').value); mostrarToast('Link copiado.', 'success'); } catch (e) { mostrarToast('Nao foi possivel copiar.', 'error'); }
  };

  async function enviarParticipantes(acao, numeros) {
    const d = await api(base() + '/grupos/' + encodeURIComponent(gruposUI.atual.jid) + '/participantes', { method: 'POST', body: JSON.stringify({ acao, numeros }) });
    return (d && d.resultado) || [];
  }

  async function acaoParticipante(acao, i) {
    const p = gruposUI.atual.lista_participantes[i]; if (!p) return;
    const nome = p.nome || formatarTelefone(p.numero) || p.jid;
    if (acao === 'remover' && !confirm('Remover ' + nome + ' do grupo?')) return;
    try {
      const r = await enviarParticipantes(acao, [p.numero || p.jid]);
      const falha = r.find(x => !x.ok);
      if (falha) mostrarToast(nome + ': ' + falha.erro, 'error'); else mostrarToast('Feito.', 'success');
      await abrirGrupo(gruposUI.atual.jid);
    } catch (err) { mostrarToast(erroTexto(err), 'error'); }
  }

  window.adicionarParticipantes = async function (e) {
    e.preventDefault();
    const campo = document.getElementById('grupo-adicionar');
    const numeros = campo.value.split(/[\s,;]+/).map(n => n.replace(/\D/g, '')).filter(Boolean);
    if (!numeros.length) return;
    try {
      const r = await enviarParticipantes('adicionar', numeros);
      campo.value = '';
      await abrirGrupo(gruposUI.atual.jid);
      const falhas = r.filter(x => !x.ok);
      const out = document.getElementById('grupo-resultado');
      if (out) out.innerHTML = falhas.map(f => '<div class="dc-config-linha"><div><strong>' + textoSeguro(formatarTelefone(f.numero)) + '</strong><span>' + textoSeguro(f.erro) + '</span></div></div>').join('');
      mostrarToast((r.length - falhas.length) + ' de ' + r.length + ' adicionado(s).', falhas.length ? 'info' : 'success');
    } catch (err) { mostrarToast(erroTexto(err), 'error'); }
  };

  window.sairGrupo = async function () {
    const g = gruposUI.atual;
    if (!confirm('Sair do grupo ' + (g.nome || g.jid) + '? Para voltar, alguem precisa adicionar esta conta de novo.')) return;
    try {
      await api(base() + '/grupos/' + encodeURIComponent(g.jid) + '/sair', { method: 'POST', body: '{}' });
      mostrarToast('Saiu do grupo.', 'success');
      fecharGrupoPainel();
      await carregarGrupos();
    } catch (err) { mostrarToast(erroTexto(err), 'error'); }
  };

  MODAIS['grupo-novo'] = {
    kicker: 'Grupos', titulo: 'Novo grupo',
    html: '<form id="modal-form"><label><span>Nome do grupo</span><input id="ng-nome" maxlength="100" required></label>' +
      '<label><span>Participantes (numeros com DDI, um por linha ou separados por virgula)</span><textarea id="ng-participantes" rows="4" placeholder="5567999999999"></textarea></label>' +
      '<div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Criar grupo</button></div></form>',
    init: () => document.getElementById('ng-nome').focus(),
    submit: async function (e) {
      e.preventDefault();
      const fb = document.getElementById('mf-feedback'), btn = document.getElementById('mf-submit');
      const participantes = document.getElementById('ng-participantes').value.split(/[\s,;]+/).map(n => n.replace(/\D/g, '')).filter(Boolean);
      btn.disabled = true; btn.textContent = 'Criando...';
      try {
        const g = await api(base() + '/grupos', { method: 'POST', body: JSON.stringify({ nome: document.getElementById('ng-nome').value, participantes }) });
        fecharModal();
        mostrarToast('Grupo criado.', 'success');
        await carregarGrupos();
        if (g) abrirGrupo(g.jid);
      } catch (err) { fb.className = 'modal-feedback error'; fb.textContent = erroTexto(err); }
      finally { btn.disabled = false; btn.textContent = 'Criar grupo'; }
    },
  };
})();
