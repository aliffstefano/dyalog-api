// Configuracao da pagina publica de status.
(function () {
  'use strict';
  async function api(url, opcoes) {
    const r = await fetch(url, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opcoes || {}));
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) throw new Error(d?.mensagem || 'Erro na requisicao');
    return d?.dados;
  }
  const ui = { pagina: null, contasCarregadas: false };
  function contaSelecionada() {
    return acessoTudo() ? document.getElementById('ps-conta').value : (estadoUI.acesso?.conta_id || '');
  }

  async function carregarContasSelect() {
    if (!acessoTudo() || ui.contasCarregadas) return;
    const sel = document.getElementById('ps-conta');
    try {
      const contas = (await api('/api/v1/contas'))?.contas || [];
      sel.innerHTML = '<option value="">Instancias sem conta</option>' + contas.map(c => '<option value="' + textoSeguro(c.id) + '">' + textoSeguro(c.nome) + '</option>').join('');
      ui.contasCarregadas = true;
    } catch (e) { sel.innerHTML = '<option value="">Instancias sem conta</option>'; }
  }

  window.carregarPaginaStatus = async function () {
    await carregarContasSelect();
    const conta = contaSelecionada();
    try {
      const p = await api('/api/v1/pagina-status' + (acessoTudo() ? '?conta_id=' + encodeURIComponent(conta) : ''));
      if (!p) return;
      ui.pagina = p;
      // Sem API_BASE_URL a API devolve o caminho; completa com o endereco do painel.
      if (p.link && p.link.startsWith('/')) p.link = location.origin + p.link;
      document.getElementById('ps-ativa').checked = !!p.ativa;
      document.getElementById('ps-titulo').value = p.titulo || '';
      document.getElementById('ps-link').textContent = p.link || 'Salve para gerar o link.';
      const abrir = document.getElementById('ps-abrir');
      abrir.classList.toggle('hidden', !p.link);
      if (p.link) abrir.href = p.link;
      const ocultas = p.instancias_ocultas || [];
      const daConta = (estadoUI.instancias || []).filter(i => (i.conta_id || '') === conta);
      document.getElementById('ps-instancias').innerHTML = daConta.length ? daConta.map(i => '<label class="dc-opcao"><input type="checkbox" data-ps-inst="' + textoSeguro(i.id) + '"' + (ocultas.includes(i.id) ? '' : ' checked') + '><span class="dc-opcao-texto"><strong>' + textoSeguro(i.nome) + '</strong><small>' + textoSeguro(rotuloStatus(i.status)) + '</small></span><span class="dc-chave" aria-hidden="true"></span></label>').join('')
        : '<span class="helper-text">Nenhuma instancia nesta conta.</span>';
    } catch (err) { mostrarToast(err.message, 'error'); }
  };

  window.salvarPaginaStatus = async function (e, novoLink) {
    e.preventDefault();
    if (novoLink && !confirm('Gerar um link novo? O link atual para de funcionar para quem ja tem.')) return;
    const conta = contaSelecionada();
    const ocultas = [...document.querySelectorAll('[data-ps-inst]')].filter(i => !i.checked).map(i => i.dataset.psInst);
    try {
      await api('/api/v1/pagina-status' + (acessoTudo() ? '?conta_id=' + encodeURIComponent(conta) : ''), { method: 'PUT', body: JSON.stringify({
        titulo: document.getElementById('ps-titulo').value, ativa: document.getElementById('ps-ativa').checked, instancias_ocultas: ocultas, novo_link: novoLink,
      }) });
      mostrarToast(novoLink ? 'Link novo gerado.' : 'Pagina de status salva.', 'success');
      carregarPaginaStatus();
    } catch (err) { mostrarToast(err.message, 'error'); }
  };

  const original = window.abrirAbaPrincipalExtra2;
  window.abrirAbaPrincipalExtra2 = function (aba) {
    if (original) original(aba);
    if (aba === 'pagina-status') carregarPaginaStatus();
  };
})();
