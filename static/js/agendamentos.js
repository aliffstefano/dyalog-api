// Agendamentos (menu principal e aba da instancia) e "agendar" a partir do chat.
(function () {
  'use strict';

  async function api(url, opcoes) {
    const r = await fetch(url, Object.assign({ headers: { 'Content-Type': 'application/json' } }, opcoes || {}));
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) { const e = new Error(d?.mensagem || 'Erro na requisicao'); e.codigo = d?.erro; throw e; }
    return d?.dados;
  }
  const vazio = msg => '<div class="dc-lista-vazia"><svg class="dc-ico"><use href="#i-relogio"/></svg>' + textoSeguro(msg) + '</div>';

  const STATUS = [['pendente', 'Pendentes'], ['enviado', 'Enviados'], ['falhou', 'Falhas'], ['cancelado', 'Cancelados'], ['', 'Todos']];
  const ROTULO_STATUS = { pendente: 'Pendente', enviando: 'Enviando', enviado: 'Enviado', falhou: 'Falhou', cancelado: 'Cancelado' };
  const ROTULO_REPETIR = { diaria: 'Todo dia', semanal: 'Toda semana', mensal: 'Todo mes' };
  // agp = pagina principal, agi = aba da instancia.
  const ui = { agp: { status: 'pendente', lista: [] }, agi: { status: 'pendente', lista: [] } };

  function chips(prefixo) {
    const el = document.getElementById(prefixo + '-status');
    el.innerHTML = STATUS.map(([v, r]) => '<button type="button" class="dc-chip' + (ui[prefixo].status === v ? ' ativo' : '') + '" data-status="' + v + '">' + r + '</button>').join('');
    el.querySelectorAll('[data-status]').forEach(b => b.addEventListener('click', () => { ui[prefixo].status = b.dataset.status; chips(prefixo); carregarAgendamentos(prefixo); }));
  }

  window.carregarAgendamentos = async function (prefixo) {
    const caixa = document.getElementById(prefixo + '-lista');
    const p = new URLSearchParams();
    if (ui[prefixo].status) p.set('status', ui[prefixo].status);
    const instancia = prefixo === 'agi' ? estadoUI.instanciaSelecionada : document.getElementById('agp-instancia').value;
    if (instancia) p.set('instancia_id', instancia);
    if (!ui[prefixo].lista.length) caixa.innerHTML = vazio('Carregando...');
    try {
      const d = await api('/api/v1/agendamentos?' + p.toString());
      if (!d) return;
      ui[prefixo].lista = d.agendamentos || [];
      renderizar(prefixo);
    } catch (err) { caixa.innerHTML = vazio(err.message); }
  };

  function para(a) {
    if (a.chat_jid && a.chat_jid.endsWith('@g.us')) return 'Grupo ' + a.chat_jid.split('@')[0];
    return formatarTelefone(a.numero || (a.chat_jid || '').split('@')[0]) || a.numero || a.chat_jid;
  }

  function renderizar(prefixo) {
    const lista = ui[prefixo].lista, caixa = document.getElementById(prefixo + '-lista');
    const comInstancia = prefixo === 'agp';
    if (!lista.length) { caixa.innerHTML = vazio(ui[prefixo].status === 'pendente' ? 'Nenhuma mensagem agendada.' : 'Nada por aqui.'); return; }
    caixa.innerHTML = '<table class="dc-tabela"><thead><tr><th>Quando</th>' + (comInstancia ? '<th>Instancia</th>' : '') + '<th>Para</th><th>Mensagem</th><th>Status</th><th></th></tr></thead><tbody>' +
      lista.map((a, i) => {
        const conteudo = (a.tem_arquivo ? '📎 ' + (a.nome_arquivo || a.tipo) + (a.mensagem ? ' · ' : '') : '') + (a.mensagem || '');
        const pendente = a.status === 'pendente';
        return '<tr><td class="dc-nowrap">' + textoSeguro(formatarData(a.status === 'enviado' && a.enviado_em ? a.enviado_em : a.enviar_em)) +
          (a.recorrencia ? '<br><small class="dc-selo neutro">' + textoSeguro(ROTULO_REPETIR[a.recorrencia] || a.recorrencia) + '</small>' : '') + '</td>' +
          (comInstancia ? '<td>' + textoSeguro(a.instancia_nome || '-') + '</td>' : '') +
          '<td class="dc-nowrap">' + textoSeguro(para(a)) + '</td>' +
          '<td class="dc-ag-texto" title="' + textoSeguro(conteudo) + '">' + textoSeguro(conteudo) + (a.criado_por ? '<small class="helper-text">por ' + textoSeguro(a.criado_por) + '</small>' : '') + '</td>' +
          '<td><span class="dc-ag-status ' + textoSeguro(a.status) + '">' + textoSeguro(ROTULO_STATUS[a.status] || a.status) + '</span>' +
            (a.erro ? '<small class="dc-ag-erro" title="' + textoSeguro(a.erro) + '">' + textoSeguro(a.erro) + (pendente && a.tentativas ? ' (tentativa ' + a.tentativas + ')' : '') + '</small>' : '') + '</td>' +
          '<td class="dc-nowrap">' + (pendente ? '<button class="ghost small" data-editar="' + i + '">Editar</button> <button class="ghost small dc-texto-perigo" data-cancelar="' + i + '">Cancelar</button>' : '') + '</td></tr>';
      }).join('') + '</tbody></table>';
    caixa.querySelectorAll('[data-editar]').forEach(b => b.addEventListener('click', () => abrirAgendamento({ editar: lista[Number(b.dataset.editar)], prefixo })));
    caixa.querySelectorAll('[data-cancelar]').forEach(b => b.addEventListener('click', () => cancelar(lista[Number(b.dataset.cancelar)], prefixo)));
  }

  async function cancelar(a, prefixo) {
    if (!confirm('Cancelar a mensagem agendada para ' + para(a) + '?')) return;
    try {
      await api('/api/v1/agendamentos/' + encodeURIComponent(a.id), { method: 'DELETE' });
      mostrarToast('Agendamento cancelado.', 'success');
      carregarAgendamentos(prefixo);
    } catch (err) { mostrarToast(err.message, 'error'); }
  }

  // Data no fuso do navegador para o input datetime-local.
  function paraInputLocal(data) {
    const d = new Date(data.getTime() - data.getTimezoneOffset() * 60000);
    return d.toISOString().slice(0, 16);
  }

  // abrirAgendamento({ instancia, numero, chat_jid, mensagem, editar, prefixo })
  const agUI = { opcoes: {}, arquivo: null };
  window.abrirAgendamento = function (opcoes) {
    agUI.opcoes = opcoes || {}; agUI.arquivo = null;
    abrirModal('agendamento');
  };

  MODAIS['agendamento'] = {
    kicker: 'Agendamentos', titulo: 'Agendar mensagem', semInstancia: true,
    html: '<form id="modal-form">' +
      '<label id="ag-inst-wrap"><span>Instancia</span><select id="ag-instancia"></select></label>' +
      '<label id="ag-para-wrap"><span>Para (numero com DDI ou JID do grupo)</span><input id="ag-para" placeholder="5567999999999" autocomplete="off"></label>' +
      '<label><span>Mensagem</span><textarea id="ag-mensagem" rows="4" placeholder="Texto ou legenda do arquivo"></textarea></label>' +
      '<label id="ag-arquivo-wrap"><span>Arquivo (opcional)</span><input type="file" id="ag-arquivo"></label>' +
      '<div class="form-grid"><label><span>Enviar em</span><input type="datetime-local" id="ag-quando" required></label>' +
      '<label><span>Repetir</span><select id="ag-repetir"><option value="">Nao repetir</option><option value="diaria">Todo dia</option><option value="semanal">Toda semana</option><option value="mensal">Todo mes</option></select></label></div>' +
      '<div id="mf-feedback" class="modal-feedback hidden"></div>' +
      '<div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Agendar</button></div></form>',
    init: function () {
      const o = agUI.opcoes, ed = o.editar;
      const sel = document.getElementById('ag-instancia');
      sel.innerHTML = (estadoUI.instancias || []).map(i => '<option value="' + textoSeguro(i.id) + '">' + textoSeguro(i.nome) + '</option>').join('');
      sel.value = ed ? ed.instancia_id : (o.instancia || estadoUI.instanciaSelecionada || sel.value);
      document.getElementById('ag-inst-wrap').classList.toggle('hidden', !!(ed || o.instancia));
      document.getElementById('ag-para').value = ed ? (ed.chat_jid || ed.numero) : (o.chat_jid && o.chat_jid.endsWith('@g.us') ? o.chat_jid : (o.numero || ''));
      document.getElementById('ag-para-wrap').classList.toggle('hidden', !!ed);
      document.getElementById('ag-arquivo-wrap').classList.toggle('hidden', !!ed);
      document.getElementById('ag-mensagem').value = ed ? (ed.mensagem || '') : (o.mensagem || '');
      const quando = ed ? new Date(ed.enviar_em) : new Date(Date.now() + 3600 * 1000);
      if (!ed) quando.setMinutes(0, 0, 0);
      document.getElementById('ag-quando').value = paraInputLocal(quando);
      document.getElementById('ag-quando').min = paraInputLocal(new Date());
      document.getElementById('ag-repetir').value = ed ? (ed.recorrencia || '') : '';
      document.getElementById('modal-titulo').textContent = ed ? 'Editar agendamento' : 'Agendar mensagem';
      document.getElementById('mf-submit').textContent = ed ? 'Salvar' : 'Agendar';
      document.getElementById('ag-arquivo').addEventListener('change', e => { agUI.arquivo = e.target.files[0] || null; });
      document.getElementById(ed || o.numero || o.chat_jid ? 'ag-mensagem' : 'ag-para').focus();
    },
    submit: async function (e) {
      e.preventDefault();
      const o = agUI.opcoes, ed = o.editar;
      const fb = document.getElementById('mf-feedback'), btn = document.getElementById('mf-submit');
      const quando = new Date(document.getElementById('ag-quando').value);
      if (isNaN(quando)) { fb.className = 'modal-feedback error'; fb.textContent = 'Informe a data e a hora.'; return; }
      const corpo = { mensagem: document.getElementById('ag-mensagem').value, enviar_em: quando.toISOString(), recorrencia: document.getElementById('ag-repetir').value };
      btn.disabled = true; fb.className = 'modal-feedback hidden';
      try {
        if (ed) {
          await api('/api/v1/agendamentos/' + encodeURIComponent(ed.id), { method: 'PUT', body: JSON.stringify(corpo) });
        } else {
          const destino = document.getElementById('ag-para').value.trim();
          corpo.instancia = document.getElementById('ag-instancia').value;
          if (destino.includes('@')) corpo.chat_jid = destino; else corpo.numero = destino.replace(/\D/g, '');
          if (agUI.arquivo) {
            if (agUI.arquivo.size > 12 * 1024 * 1024) throw new Error('Arquivo acima de 12 MB.');
            corpo.arquivo_base64 = await lerArquivoBase64(agUI.arquivo);
            corpo.nome_arquivo = agUI.arquivo.name;
            corpo.mime_type = agUI.arquivo.type || 'application/octet-stream';
          }
          await api('/api/v1/agendamentos', { method: 'POST', body: JSON.stringify(corpo) });
        }
        fecharModal();
        mostrarToast(ed ? 'Agendamento atualizado.' : 'Mensagem agendada para ' + quando.toLocaleString('pt-BR') + '.', 'success');
        if (o.depois) o.depois();
        ['agp', 'agi'].forEach(p => { if (document.getElementById(p + '-lista').offsetParent) carregarAgendamentos(p); });
      } catch (err) { fb.className = 'modal-feedback error'; fb.textContent = err.message; }
      finally { btn.disabled = false; }
    },
  };

  // Botao do relogio no chat: agenda o texto digitado para a conversa aberta.
  window.agendarDoChat = function () {
    if (!chatUI.chat) { mostrarToast('Abra uma conversa primeiro.', 'error'); return; }
    const campo = document.getElementById('chat-texto');
    const grupo = chatUI.chat.endsWith('@g.us');
    abrirAgendamento({
      instancia: estadoUI.instanciaSelecionada,
      numero: grupo ? '' : chatUI.chat.split('@')[0],
      chat_jid: grupo ? chatUI.chat : '',
      mensagem: campo.value,
      depois: () => { campo.value = ''; ajustarAlturaChat(); },
    });
  };

  // Ganchos nas trocas de aba.
  const principalOriginal = window.abrirAbaPrincipalExtra2;
  window.abrirAbaPrincipalExtra2 = function (aba) {
    if (principalOriginal) principalOriginal(aba);
    if (aba === 'agendamentos') {
      const sel = document.getElementById('agp-instancia'), atual = sel.value;
      sel.innerHTML = '<option value="">Todas as instancias</option>' + (estadoUI.instancias || []).map(i => '<option value="' + textoSeguro(i.id) + '">' + textoSeguro(i.nome) + '</option>').join('');
      sel.value = atual;
      chips('agp'); carregarAgendamentos('agp');
    }
  };
  const instanciaOriginal = window.abrirAbaExtra;
  window.abrirAbaExtra = function (aba) {
    if (instanciaOriginal) instanciaOriginal(aba);
    if (aba === 'agendamentos') { ui.agi.lista = []; chips('agi'); carregarAgendamentos('agi'); }
  };
})();
