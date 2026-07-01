package dashboard

const paginaInicialHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Titulo }}</title>
  <script>document.documentElement.setAttribute('data-theme',localStorage.getItem('tema')||'');</script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@24,500,0,0" rel="stylesheet">
  <link rel="stylesheet" href="/static/css/dashboard.css">
</head>
<body>
  <main class="app-shell">
    <header class="topbar glass-card">
      <div class="brand-block">
        <img src="/static/img/dyalog.png" alt="Dyalog" class="brand-logo">
        <div>
          <p class="eyebrow">Infrastructure Console</p>
          <h1>Dyalog API</h1>
        </div>
      </div>
      <nav class="topnav">
        <button class="nav-link active" data-tab="dashboard"   onclick="trocarAba('dashboard')"><span class="material-symbols-outlined">dashboard</span>Dashboard</button>
        <button class="nav-link"        data-tab="instancias"  onclick="trocarAba('instancias')"><span class="material-symbols-outlined">dns</span>Instancias</button>
        <button class="nav-link"        data-tab="atualizacao" onclick="trocarAba('atualizacao')"><span class="material-symbols-outlined">system_update_alt</span>Atualizacao</button>
      </nav>
      <div class="topbar-actions">
        <div class="theme-switcher" role="group" aria-label="Tema">
          <button class="theme-btn" data-tema=""      onclick="setTema('')"      title="Quente">&#9728;</button>
          <button class="theme-btn" data-tema="clean" onclick="setTema('clean')" title="Clean">&#9711;</button>
          <button class="theme-btn" data-tema="dark"  onclick="setTema('dark')"  title="Escuro">&#9790;</button>
        </div>
        <a href="/docs" target="_blank" style="text-decoration:none"><button class="ghost small">Docs</button></a>
        <button class="ghost" onclick="carregarTudo(true)">Atualizar</button>
        <div class="logout-block">
          <button class="ghost" onclick="sairDashboard()">Sair</button>
          <span class="app-version-label">v1.0</span>
        </div>
      </div>
    </header>

    <section id="alerta-atualizacao" class="banner hidden"></section>
    <section id="toast" class="toast hidden"></section>

    <!-- ===== TAB DASHBOARD ===== -->
    <section class="tab-panel active" data-panel="dashboard">
      <section class="hero-card glass-card">
        <div>
          <p class="eyebrow">Visao geral</p>
          <h2>Operacao central das instancias WhatsApp</h2>
          <p class="subtitle">Acompanhe a saude do sistema, conexoes ativas e situacao do whatsmeow antes de entrar no detalhe das instancias.</p>
        </div>
        <div class="hero-actions">
          <button class="primary" onclick="trocarAba('instancias')">Abrir instancias</button>
          <button class="ghost"   onclick="trocarAba('atualizacao')">Ver atualizacao</button>
        </div>
      </section>

      <section class="overview-grid">
        <article class="glass-card stats-card emphasis">
          <span class="stats-label">Aplicacao</span>
          <strong id="app-versao" class="stats-value">-</strong>
          <span class="stats-meta">Versao em execucao</span>
        </article>
        <article class="glass-card stats-card">
          <span class="stats-label">Whatsmeow</span>
          <strong id="wm-versao" class="stats-value">-</strong>
          <span class="stats-meta">Nucleo de conexao</span>
        </article>
        <article class="glass-card stats-card">
          <span class="stats-label">Instancias</span>
          <strong id="contador-instancias" class="stats-value">0</strong>
          <span class="stats-meta">Total cadastrado</span>
        </article>
        <article class="glass-card stats-card">
          <span class="stats-label">Online</span>
          <strong id="contador-online" class="stats-value">0</strong>
          <span class="stats-meta">Conectadas agora</span>
        </article>
      </section>

      <section class="dashboard-grid">
        <article class="glass-card dashboard-card spotlight">
          <div class="panel-header">
            <div><p class="panel-kicker">Resumo ativo</p><h3>Instancia selecionada</h3></div>
            <span id="status-badge-home" class="status-badge neutro">Sem selecao</span>
          </div>
          <div class="spotlight-grid">
            <div><span class="mini-label">Nome</span><strong id="home-instancia-nome">Nenhuma instancia selecionada</strong></div>
            <div><span class="mini-label">Status</span><strong id="home-instancia-status">-</strong></div>
            <div><span class="mini-label">Atualizado</span><strong id="home-instancia-atualizado">-</strong></div>
            <div><span class="mini-label">Erro</span><strong id="home-instancia-erro">-</strong></div>
          </div>
          <p id="home-status-copy" class="status-copy">Clique em uma instancia para ver QR code, status e acoes.</p>
        </article>
        <article class="glass-card dashboard-card">
          <div class="panel-header">
            <div><p class="panel-kicker">Atalhos</p><h3>Acoes rapidas</h3></div>
          </div>
          <div class="shortcut-grid">
            <button class="shortcut-tile" onclick="trocarAba('instancias')"><strong>Gerenciar instancias</strong><span>Criar, conectar, configurar webhooks, proxy e enviar mensagens.</span></button>
            <button class="shortcut-tile" onclick="trocarAba('instancias')"><strong>Auditar webhooks</strong><span>Abra uma instancia e veja a auditoria no mesmo painel.</span></button>
            <button class="shortcut-tile" onclick="trocarAba('atualizacao')"><strong>Monitorar dependencias</strong><span>Verificar versao do whatsmeow e orientacao de update.</span></button>
          </div>
        </article>
      </section>
    </section>

    <!-- ===== TAB INSTANCIAS ===== -->
    <section class="tab-panel" data-panel="instancias">
      <section class="workspace-grid">

        <!-- Sidebar -->
        <article class="glass-card sidebar-card">
          <div class="panel-header">
            <div><p class="panel-kicker">Instancias</p><h2>Operacao</h2></div>
            <span id="auto-sync-status" class="sync-pill">auto</span>
          </div>
          <form id="form-instancia" class="stack-form compact-form">
            <label><span>Nova instancia</span><input type="text" id="nome-instancia" placeholder="Ex.: Atendimento principal" required></label>
            <button type="submit" class="primary">Criar instancia</button>
          </form>
          <div id="lista-instancias" class="instance-list empty-state">Nenhuma instancia cadastrada.</div>
        </article>

        <!-- Painel principal -->
        <article class="glass-card main-card">
          <div class="panel-header">
            <div><p class="panel-kicker">Instancia selecionada</p><h2 id="instancia-titulo">Nenhuma instancia selecionada</h2></div>
            <span id="status-badge" class="status-badge neutro">Sem selecao</span>
          </div>

          <!-- QR + Status -->
          <div class="detail-grid">
            <section class="detail-card qrcode-card">
              <div class="detail-head">
                <h3>QR code e conexao</h3>
                <button class="ghost small" id="botao-ver-qr" onclick="mostrarQRCodeSelecionado()" disabled>Atualizar QR</button>
              </div>
              <div id="qrcode-box" class="qrcode-box">Clique em uma instancia para visualizar o QR code.</div>
              <div class="action-row">
                <button class="primary" id="botao-conectar"     onclick="conectarSelecionada()"     disabled>Conectar</button>
                <button class="ghost"   id="botao-desconectar"  onclick="desconectarSelecionada()"  disabled>Desconectar</button>
                <button class="danger"  id="botao-excluir"      onclick="excluirSelecionada()"      disabled>Excluir</button>
              </div>
            </section>

            <section class="detail-card status-card-panel">
              <div class="detail-head"><h3>Status detalhado</h3></div>
              <dl class="status-grid">
                <div><dt>ID</dt><dd id="detalhe-id">-</dd></div>
                <div><dt>Token</dt><dd><button class="token-inline" id="detalhe-token" type="button" onclick="revelarECopiarTokenSelecionado()" disabled>-</button></dd></div>
                <div><dt>Status</dt><dd id="detalhe-status">-</dd></div>
                <div><dt>Atualizado</dt><dd id="detalhe-atualizado">-</dd></div>
                <div><dt>Erro</dt><dd id="detalhe-erro">-</dd></div>
              </dl>
              <div id="status-copy" class="status-copy">Aguardando selecao.</div>
              <div class="action-row" style="margin-top:14px">
                <button class="ghost small" id="botao-copiar-token" onclick="copiarTokenSelecionado()" disabled>Copiar token</button>
                <button class="ghost small" id="botao-pairing" onclick="abrirModal('pairing')" disabled>Pairing code</button>
              </div>
            </section>
          </div>

          <!-- Acoes rapidas -->
          <section class="detail-card actions-card">
            <div class="detail-head"><h3>Acoes rapidas</h3><span class="helper-text">Selecione uma instancia para usar.</span></div>
            <div class="action-buttons-grid">
              <button class="action-tile" id="botao-enviar-texto" onclick="abrirModal('texto')" disabled>
                <strong>Enviar mensagem</strong>
                <span>Texto via WhatsApp</span>
              </button>
              <button class="action-tile" id="botao-teste-chamada" onclick="abrirModal('chamada')" disabled>
                <strong>Teste de chamada</strong>
                <span>Voz 1:1 pelo navegador</span>
              </button>
              <button class="action-tile" id="botao-config-token" onclick="abrirModal('token')" disabled>
                <strong>Token da instancia</strong>
                <span>Alterar token de acesso</span>
              </button>
              <button class="action-tile" id="botao-config-proxy" onclick="abrirModal('proxy')" disabled>
                <strong>Proxy</strong>
                <span>Configurar proxy HTTP/SOCKS</span>
              </button>
              <button class="action-tile" id="botao-config-historico" onclick="abrirModal('historico')" disabled>
                <strong>Historico</strong>
                <span>Dias de historico inicial</span>
              </button>
            </div>
          </section>

          <!-- Configuracoes avancadas -->
          <section class="detail-card advanced-card collapsed" id="advanced-card">
            <div class="detail-head advanced-head" role="button" tabindex="0" onclick="alternarAvancado()" onkeydown="if(event.key==='Enter'||event.key===' '){event.preventDefault();alternarAvancado()}">
              <div>
                <h3>Configuracoes avancadas</h3>
                <span class="helper-text">Presenca, leitura, chamadas e filtros de webhook.</span>
              </div>
              <button class="ghost small" id="botao-toggle-avancado" type="button" onclick="event.stopPropagation();alternarAvancado()">Abrir</button>
            </div>
            <form id="form-avancado" class="advanced-form hidden">
              <div class="advanced-option">
                <input type="checkbox" id="adv-manter-online">
                <label for="adv-manter-online"><strong>Manter sempre online</strong><small>Marcado = disponivel. Desmarcado = indisponivel.</small></label>
              </div>
              <div class="advanced-option">
                <input type="checkbox" id="adv-rejeitar-chamadas">
                <label for="adv-rejeitar-chamadas"><strong>Rejeitar chamadas</strong><small>Recusa chamadas recebidas automaticamente.</small></label>
              </div>
              <label class="advanced-message hidden" id="adv-mensagem-wrap">
                <span>Mensagem automatica apos rejeitar chamada</span>
                <textarea id="adv-mensagem-rejeitar" placeholder="No momento nao consigo atender chamadas. Envie uma mensagem por aqui."></textarea>
              </label>
              <div class="advanced-option">
                <input type="checkbox" id="adv-marcar-lida">
                <label for="adv-marcar-lida"><strong>Marcar como lida</strong><small>Aplica leitura automaticamente nas mensagens recebidas.</small></label>
              </div>
              <div class="advanced-option">
                <input type="checkbox" id="adv-ignorar-grupos">
                <label for="adv-ignorar-grupos"><strong>Ignorar grupos</strong><small>Mensagens de grupos nao vao para webhooks.</small></label>
              </div>
              <div class="advanced-option">
                <input type="checkbox" id="adv-ignorar-status">
                <label for="adv-ignorar-status"><strong>Ignorar status</strong><small>Status do WhatsApp nao vao para webhooks.</small></label>
              </div>
              <div class="advanced-footer">
                <span id="advanced-copy" class="status-copy">Selecione uma instancia para editar.</span>
                <button class="primary small" id="botao-salvar-avancado" type="submit" disabled>Salvar avancado</button>
              </div>
            </form>
          </section>

          <!-- Webhooks -->
          <section class="detail-card webhook-card">
            <div class="detail-head">
              <h3>Webhooks</h3>
              <button class="ghost small" id="botao-novo-webhook" onclick="abrirModal('webhook-novo')" disabled>+ Adicionar</button>
            </div>
            <div id="lista-webhooks" class="webhook-list empty-state">Selecione uma instancia para ver os webhooks configurados.</div>
          </section>

          <!-- Auditoria da instancia -->
          <section class="detail-card audit-card collapsed" id="audit-card">
            <div class="detail-head advanced-head" role="button" tabindex="0" onclick="alternarAuditoria()" onkeydown="if(event.key==='Enter'||event.key===' '){event.preventDefault();alternarAuditoria()}">
              <div>
                <h3>Auditoria de webhooks</h3>
                <span class="helper-text">Entregas recentes, retries, HTTP e falhas desta instancia.</span>
              </div>
              <div class="audit-head-actions">
                <button class="ghost small" type="button" onclick="event.stopPropagation();carregarAuditoria(true)" id="botao-atualizar-auditoria" disabled>Atualizar</button>
                <button class="ghost small" type="button" onclick="event.stopPropagation();alternarAuditoria()" id="botao-toggle-auditoria">Abrir</button>
              </div>
            </div>
            <div id="audit-body" class="audit-body hidden">
              <div id="resumo-entregas-webhook" class="delivery-summary hidden"></div>
              <div id="lista-entregas-webhook" class="webhook-delivery-list empty-state">Selecione uma instancia para ver a auditoria.</div>
            </div>
          </section>
        </article>
      </section>
    </section>

    <!-- ===== TAB ATUALIZACAO ===== -->
    <section class="tab-panel" data-panel="atualizacao">
      <section class="update-layout">
        <article class="glass-card update-card main-update-card">
          <div class="panel-header">
            <div><p class="panel-kicker">Dependencia monitorada</p><h2>Whatsmeow</h2></div>
            <button class="primary" onclick="verificarAtualizacoes()">Verificar agora</button>
          </div>
          <div class="update-grid">
            <div class="update-item"><span class="mini-label">Versao em uso</span><strong id="update-versao-atual">-</strong></div>
            <div class="update-item"><span class="mini-label">Ultima disponivel</span><strong id="update-versao-disponivel">-</strong></div>
            <div class="update-item"><span class="mini-label">Modo</span><strong id="update-modo">-</strong></div>
            <div class="update-item"><span class="mini-label">Status</span><strong id="update-status">-</strong></div>
            <div class="update-item"><span class="mini-label">Ultima verificacao</span><strong id="update-verificacao">-</strong></div>
            <div class="update-item"><span class="mini-label">Erro</span><strong id="update-erro">-</strong></div>
          </div>
          <p id="update-copy" class="status-copy">Aguardando dados de atualizacao.</p>
        </article>
        <article id="update-guide-card" class="glass-card update-card status-ok-card">
          <div class="panel-header"><div><p class="panel-kicker">Status do sistema</p><h3 id="update-guide-title">Sistema atualizado</h3></div></div>
          <div id="update-guide-body" class="guide-content"><p class="status-copy">Nenhuma atualizacao pendente. Nenhuma acao operacional e necessaria agora.</p></div>
        </article>
      </section>
    </section>

    <!-- Log -->
    <section class="glass-card log-card">
      <div class="panel-header"><div><p class="panel-kicker">Inspecao</p><h2>Retorno da API</h2></div></div>
      <pre id="retorno-api">Sem chamadas ainda.</pre>
    </section>
  </main>

  <!-- ===== LOGIN ===== -->
  <div id="login-overlay" class="modal-overlay hidden" role="dialog" aria-modal="true">
    <div class="modal-dialog glass-card" style="max-width:420px">
      <div class="modal-header">
        <div><p class="panel-kicker">Autenticacao</p><h3>Acesso ao painel</h3></div>
      </div>
      <form id="form-login" class="stack-form">
        <label><span>Token de acesso</span><input type="password" id="login-token" placeholder="Cole o token master aqui" required autocomplete="off"></label>
        <div id="login-feedback" class="modal-feedback hidden"></div>
        <button type="submit" class="primary" id="login-submit">Entrar</button>
      </form>
    </div>
  </div>

  <!-- ===== MODAL ===== -->
  <div id="modal-overlay" class="modal-overlay hidden" onclick="fecharModalExterno(event)" role="dialog" aria-modal="true">
    <div class="modal-dialog glass-card">
      <div class="modal-header">
        <div><p class="panel-kicker" id="modal-kicker">Acao</p><h3 id="modal-titulo">-</h3></div>
        <button class="ghost modal-close" onclick="fecharModal()" aria-label="Fechar">&#10005;</button>
      </div>
      <div id="modal-corpo"></div>
    </div>
  </div>

  <script>
  const estadoUI = { abaAtual:'dashboard', instanciaSelecionada:'', pollingConexao:null, refreshGeral:null, acesso:null, webhooks:[], entregasWebhook:[] };

  // ── Tema ─────────────────────────────────────────────────────────────────────
  function setTema(t) {
    document.documentElement.setAttribute('data-theme', t);
    localStorage.setItem('tema', t);
    document.querySelectorAll('.theme-btn').forEach(b => b.classList.toggle('ativo', (b.dataset.tema||'') === t));
  }
  (function() {
    const t = localStorage.getItem('tema') || '';
    document.documentElement.setAttribute('data-theme', t);
    document.querySelectorAll('.theme-btn').forEach(b => b.classList.toggle('ativo', (b.dataset.tema||'') === t));
  })();

  // ── Modal ─────────────────────────────────────────────────────────────────────
  const MODAIS = {
    texto: {
      kicker: 'Acao rapida', titulo: 'Enviar mensagem',
      html: '<form id="modal-form" class="stack-form"><div class="form-grid"><label><span>Numero destino</span><input type="text" id="mf-numero" placeholder="5511999999999" required autocomplete="off"></label><label><span>Instancia</span><input type="text" id="mf-instancia" readonly></label></div><label><span>Mensagem</span><textarea id="mf-texto" placeholder="Mensagem de teste..." required style="min-height:100px"></textarea></label><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Enviar</button></div></form>',
      init: function() { document.getElementById('mf-instancia').value = estadoUI.instanciaSelecionada; document.getElementById('mf-numero').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Enviando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/batepapo/enviar/texto', { method:'POST', body: JSON.stringify({ instancia: document.getElementById('mf-instancia').value, numero: document.getElementById('mf-numero').value, mensagem: document.getElementById('mf-texto').value }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Mensagem enviada com sucesso!';
          mostrarToast('Mensagem enviada.', 'success');
          setTimeout(fecharModal, 1600);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Enviar'; }
      }
    },
    chamada: {
      kicker: 'Chamadas', titulo: 'Teste de chamada WhatsApp',
      html: '<form id="modal-form" class="stack-form"><label><span>Numero destino</span><input type="text" id="mf-numero" placeholder="5511999999999" required autocomplete="off"></label><p class="status-copy" style="margin:0">O navegador pedira permissao de microfone. Para aplicacoes externas, use o webhook <code>chamadas</code> e negocie WebRTC via API.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div id="mf-call-state" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="encerrarChamadaPainel()">Encerrar</button><button type="submit" class="primary" id="mf-submit">Iniciar chamada</button></div></form>',
      init: function() { document.getElementById('mf-numero').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback'), st = document.getElementById('mf-call-state');
        btn.disabled = true; btn.textContent = 'Iniciando...'; fb.className = 'modal-feedback hidden'; st.className = 'modal-feedback hidden';
        try {
          await iniciarChamadaPainel(document.getElementById('mf-numero').value);
          st.className = 'modal-feedback success';
          st.textContent = 'Chamada iniciada. Mantenha esta janela aberta durante o teste.';
          btn.textContent = 'Chamada em andamento';
          mostrarToast('Chamada iniciada.', 'success');
        } catch(err) {
          fb.className = 'modal-feedback error';
          fb.textContent = 'Erro: ' + err.message;
          btn.disabled = false;
          btn.textContent = 'Iniciar chamada';
        }
      }
    },
    pairing: {
      kicker: 'Conexao', titulo: 'Solicitar pairing code',
      html: '<form id="modal-form" class="stack-form"><label><span>Numero do WhatsApp (com DDI)</span><input type="text" id="mf-numero" placeholder="5511999999999" required autocomplete="off"></label><p class="status-copy" style="margin:0">Use o codigo gerado para vincular sem QR code.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div id="mf-resultado" class="hidden pairing-result"><span class="pairing-code" id="mf-codigo">-</span><p class="stats-meta">Codigo de pareamento</p><button type="button" class="ghost small" onclick="copiarPairingCode()">Copiar codigo</button></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Fechar</button><button type="submit" class="primary" id="mf-submit">Gerar codigo</button></div></form>',
      init: function() { document.getElementById('mf-numero').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Gerando...'; fb.className = 'modal-feedback hidden';
        try {
          const r = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/pairing-code', { method:'POST', body: JSON.stringify({ numero: document.getElementById('mf-numero').value }) });
          document.getElementById('mf-codigo').textContent = r.dados.codigo || '-';
          document.getElementById('mf-resultado').classList.remove('hidden');
          btn.textContent = 'Novo codigo';
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; btn.textContent = 'Tentar novamente'; }
        finally { btn.disabled = false; }
      }
    },
    token: {
      kicker: 'Configuracao', titulo: 'Token da instancia',
      html: '<form id="modal-form" class="stack-form"><label><span>Novo token de acesso</span><input type="text" id="mf-token" placeholder="Cole o novo token aqui" required autocomplete="off"></label><p class="status-copy" style="margin:0">Use um token forte e unico por instancia.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar token</button></div></form>',
      init: function() { document.getElementById('mf-token').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Salvando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/token', { method:'PUT', body: JSON.stringify({ token: document.getElementById('mf-token').value }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Token atualizado com sucesso!';
          mostrarToast('Token atualizado.', 'success');
          setTimeout(fecharModal, 1600);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Salvar token'; }
      }
    },
    proxy: {
      kicker: 'Configuracao', titulo: 'Proxy da instancia',
      html: '<form id="modal-form" class="stack-form"><label><span>Modo</span><select id="mf-modo" style="width:100%;border:1px solid var(--line);background:rgba(255,255,255,.76);border-radius:16px;padding:14px 15px;outline:none;color:var(--ink)"><option value="">Sem proxy</option><option value="http">HTTP</option><option value="socks5">SOCKS5</option></select></label><label><span>URL do proxy</span><input type="text" id="mf-url" placeholder="http://usuario:senha@host:porta" autocomplete="off"></label><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar proxy</button></div></form>',
      init: function() {},
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Salvando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/proxy', { method:'PUT', body: JSON.stringify({ modo: document.getElementById('mf-modo').value, url: document.getElementById('mf-url').value }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Proxy atualizado com sucesso!';
          mostrarToast('Proxy atualizado.', 'success');
          setTimeout(fecharModal, 1600);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Salvar proxy'; }
      }
    },
    presenca: {
      kicker: 'Configuracao', titulo: 'Presenca persistente',
      html: '<form id="modal-form" class="stack-form"><label><span>Estado global da instancia</span><select id="mf-presenca" style="width:100%;border:1px solid var(--line);background:rgba(255,255,255,.76);border-radius:16px;padding:14px 15px;outline:none;color:var(--ink)"><option value="disponivel">Disponivel</option><option value="indisponivel">Indisponivel</option></select></label><p class="status-copy" style="margin:0">Esse estado sera reaplicado quando a instancia conectar ou restaurar sessao.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar presenca</button></div></form>',
      init: async function() {
        const select = document.getElementById('mf-presenca');
        try {
          const resp = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/status');
          select.value = resp?.dados?.presenca || 'disponivel';
        } catch(e) {
          select.value = 'disponivel';
        }
      },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Salvando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/presenca', { method:'PUT', body: JSON.stringify({ presenca: document.getElementById('mf-presenca').value }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Presenca atualizada com sucesso!';
          mostrarToast('Presenca atualizada.', 'success');
          await atualizarInstanciaSelecionada(false);
          setTimeout(fecharModal, 1400);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Salvar presenca'; }
      }
    },
    historico: {
      kicker: 'Configuracao', titulo: 'Historico inicial',
      html: '<form id="modal-form" class="stack-form"><label><span>Dias de historico (0 = sem historico)</span><input type="number" id="mf-dias" min="0" max="90" placeholder="0" required></label><p class="status-copy" style="margin:0">Configure antes de conectar. Alterar requer nova conexao.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar</button></div></form>',
      init: function() { document.getElementById('mf-dias').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        btn.disabled = true; btn.textContent = 'Salvando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/historico', { method:'PUT', body: JSON.stringify({ dias: parseInt(document.getElementById('mf-dias').value) || 0 }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Historico atualizado com sucesso!';
          mostrarToast('Historico atualizado.', 'success');
          setTimeout(fecharModal, 1600);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Salvar'; }
      }
    },
    'webhook-novo': {
      kicker: 'Webhooks', titulo: 'Novo webhook',
      html: '<form id="modal-form" class="stack-form"><label><span>Nome</span><input type="text" id="mf-wh-nome" placeholder="Ex.: Notificacoes" required autocomplete="off"></label><label><span>URL destino</span><input type="url" id="mf-wh-url" placeholder="https://meusite.com/webhook" required autocomplete="off"></label><label><span>Eventos (selecione um ou mais)</span><div class="checkbox-group" id="mf-wh-eventos"><label class="check-item"><input type="checkbox" value="mensagens"> Mensagens</label><label class="check-item"><input type="checkbox" value="recibos"> Recibos</label><label class="check-item"><input type="checkbox" value="chamadas"> Chamadas</label><label class="check-item"><input type="checkbox" value="status"> Status</label><label class="check-item"><input type="checkbox" value="digitando"> Digitando</label><label class="check-item"><input type="checkbox" value="gravando_audio"> Gravando audio</label></div></label><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Criar webhook</button></div></form>',
      init: function() { document.getElementById('mf-wh-nome').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        const eventos = Array.from(document.querySelectorAll('#mf-wh-eventos input:checked')).map(i => i.value);
        if (!eventos.length) { fb.className = 'modal-feedback error'; fb.textContent = 'Selecione ao menos um evento.'; return; }
        btn.disabled = true; btn.textContent = 'Criando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhooks', { method:'POST', body: JSON.stringify({ nome: document.getElementById('mf-wh-nome').value, url: document.getElementById('mf-wh-url').value, eventos: eventos }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Webhook criado com sucesso!';
          mostrarToast('Webhook criado.', 'success');
          await carregarWebhooks();
          setTimeout(fecharModal, 1400);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Criar webhook'; }
      }
    },
    'webhook-editar': {
      kicker: 'Webhooks', titulo: 'Editar webhook',
      html: '<form id="modal-form" class="stack-form"><label><span>Nome</span><input type="text" id="mf-wh-nome" required autocomplete="off"></label><label><span>URL destino</span><input type="url" id="mf-wh-url" required autocomplete="off"></label><label><span>Ativo</span><select id="mf-wh-ativo" style="width:100%;border:1px solid var(--line);background:rgba(255,255,255,.76);border-radius:16px;padding:14px 15px;outline:none;color:var(--ink)"><option value="true">Ativo</option><option value="false">Inativo</option></select></label><label><span>Eventos</span><div class="checkbox-group" id="mf-wh-eventos"><label class="check-item"><input type="checkbox" value="mensagens"> Mensagens</label><label class="check-item"><input type="checkbox" value="recibos"> Recibos</label><label class="check-item"><input type="checkbox" value="chamadas"> Chamadas</label><label class="check-item"><input type="checkbox" value="status"> Status</label><label class="check-item"><input type="checkbox" value="digitando"> Digitando</label><label class="check-item"><input type="checkbox" value="gravando_audio"> Gravando audio</label></div></label><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar webhook</button></div></form>',
      init: function() {
        const wh = estadoUI.webhookEditando;
        if (!wh) return;
        document.getElementById('mf-wh-nome').value = wh.nome || '';
        document.getElementById('mf-wh-url').value = wh.url || '';
        document.getElementById('mf-wh-ativo').value = wh.ativo ? 'true' : 'false';
        const eventos = new Set(wh.eventos || []);
        document.querySelectorAll('#mf-wh-eventos input').forEach(i => i.checked = eventos.has(i.value));
        document.getElementById('mf-wh-nome').focus();
      },
      submit: async function(e) {
        e.preventDefault();
        const wh = estadoUI.webhookEditando;
        if (!wh) return;
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        const eventos = Array.from(document.querySelectorAll('#mf-wh-eventos input:checked')).map(i => i.value);
        if (!eventos.length) { fb.className = 'modal-feedback error'; fb.textContent = 'Selecione ao menos um evento.'; return; }
        btn.disabled = true; btn.textContent = 'Salvando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhooks/' + wh.id, { method:'PUT', body: JSON.stringify({ nome: document.getElementById('mf-wh-nome').value, url: document.getElementById('mf-wh-url').value, eventos: eventos, ativo: document.getElementById('mf-wh-ativo').value === 'true' }) });
          fb.className = 'modal-feedback success'; fb.textContent = 'Webhook atualizado com sucesso!';
          mostrarToast('Webhook atualizado.', 'success');
          await carregarWebhooks();
          setTimeout(fecharModal, 1000);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Salvar webhook'; }
      }
    }
  };

  let modalTipoAtual = '';
  function abrirModal(tipo) {
    if (!estadoUI.instanciaSelecionada) { mostrarToast('Selecione uma instancia para usar as acoes.', 'error'); return; }
    const m = MODAIS[tipo]; if (!m) return;
    modalTipoAtual = tipo;
    document.getElementById('modal-kicker').textContent = m.kicker;
    document.getElementById('modal-titulo').textContent = m.titulo;
    document.getElementById('modal-corpo').innerHTML = m.html;
    document.getElementById('modal-overlay').classList.remove('hidden');
    const form = document.getElementById('modal-form');
    if (form) form.addEventListener('submit', m.submit);
    if (m.init) setTimeout(m.init, 60);
  }
  function fecharModal() {
    if (modalTipoAtual === 'chamada') encerrarChamadaPainel(true);
    document.getElementById('modal-overlay').classList.add('hidden');
    modalTipoAtual = '';
  }
  function fecharModalExterno(e) { if (e.target === document.getElementById('modal-overlay')) fecharModal(); }
  document.addEventListener('keydown', e => { if (e.key === 'Escape') fecharModal(); });

  async function iniciarChamadaPainel(numero) {
    await encerrarChamadaPainel(false);
    if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) throw new Error('Navegador sem suporte a microfone/WebRTC.');
    const inicio = await chamar('/api/v1/chamadas/iniciar', { method:'POST', body: JSON.stringify({ instancia: estadoUI.instanciaSelecionada, numero: numero }) });
    const chamadaID = inicio?.dados?.chamada_id;
    if (!chamadaID) throw new Error('A API nao retornou chamada_id.');
    const stream = await navigator.mediaDevices.getUserMedia({ audio: { echoCancellation:true, noiseSuppression:true }, video:false });
    const pc = new RTCPeerConnection();
    const remoteStream = new MediaStream();
    const audioEl = new Audio();
    audioEl.autoplay = true;
    audioEl.playsInline = true;
    audioEl.srcObject = remoteStream;
    estadoUI.chamadaPainel = { chamadaID, pc, stream, remoteStream, audioEl, nodes: [] };
    stream.getAudioTracks().forEach(track => pc.addTrack(track, stream));
    pc.ontrack = function(evt) {
      evt.streams?.[0]?.getTracks().forEach(track => remoteStream.addTrack(track));
      if (!evt.streams?.length) remoteStream.addTrack(evt.track);
      audioEl.play().catch(() => {});
    };
    pc.oniceconnectionstatechange = function() {
      if (['failed','closed','disconnected'].includes(pc.iceConnectionState)) encerrarChamadaPainel(false);
    };
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    await aguardarICECompleto(pc);
    const sinal = await chamar('/api/v1/chamadas/' + chamadaID + '/webrtc', { method:'POST', body: JSON.stringify({ instancia: estadoUI.instanciaSelecionada, sdp_offer: pc.localDescription.sdp }) });
    await pc.setRemoteDescription({ type:'answer', sdp: sinal.dados.sdp_answer });
  }

  async function encerrarChamadaPainel(chamarAPI) {
    const atual = estadoUI.chamadaPainel;
    if (!atual) return;
    estadoUI.chamadaPainel = null;
    try { atual.nodes?.forEach(n => { try { n.disconnect(); } catch(e) {} }); } catch(e) {}
    try { atual.stream?.getTracks().forEach(t => t.stop()); } catch(e) {}
    try { atual.remoteStream?.getTracks().forEach(t => t.stop()); } catch(e) {}
    try { atual.dc?.close(); } catch(e) {}
    try { atual.pc?.close(); } catch(e) {}
    try { await atual.audioCtx?.close(); } catch(e) {}
    if (chamarAPI !== false && atual.chamadaID) {
      try { await chamar('/api/v1/chamadas/' + atual.chamadaID, { method:'DELETE', body: JSON.stringify({ instancia: estadoUI.instanciaSelecionada }) }); } catch(e) {}
    }
    const btn = document.getElementById('mf-submit');
    if (btn && modalTipoAtual === 'chamada') { btn.disabled = false; btn.textContent = 'Iniciar chamada'; }
    const st = document.getElementById('mf-call-state');
    if (st && modalTipoAtual === 'chamada') { st.className = 'modal-feedback'; st.textContent = 'Chamada encerrada.'; }
  }

  function aguardarICECompleto(pc) {
    if (pc.iceGatheringState === 'complete') return Promise.resolve();
    return new Promise(resolve => {
      const done = () => {
        if (pc.iceGatheringState === 'complete') {
          pc.removeEventListener('icegatheringstatechange', done);
          resolve();
        }
      };
      pc.addEventListener('icegatheringstatechange', done);
      setTimeout(resolve, 3000);
    });
  }

  function conectarMicrofoneChamada(stream, audioCtx, dc) {
    const source = audioCtx.createMediaStreamSource(stream);
    const processor = audioCtx.createScriptProcessor(4096, 1, 1);
    source.connect(processor);
    processor.connect(audioCtx.destination);
    estadoUI.chamadaPainel.nodes.push(source, processor);
    processor.onaudioprocess = function(e) {
      if (!estadoUI.chamadaPainel || dc.readyState !== 'open') return;
      const input = e.inputBuffer.getChannelData(0);
      dc.send(floatParaPCM16LE(resampleFloat32(input, audioCtx.sampleRate, 16000)));
    };
  }

  function resampleFloat32(input, fromRate, toRate) {
    if (fromRate === toRate) return input;
    const ratio = fromRate / toRate;
    const length = Math.max(1, Math.round(input.length / ratio));
    const out = new Float32Array(length);
    for (let i = 0; i < length; i++) {
      const pos = i * ratio, idx = Math.floor(pos), frac = pos - idx;
      const a = input[idx] || 0, b = input[idx + 1] || a;
      out[i] = a + (b - a) * frac;
    }
    return out;
  }

  function floatParaPCM16LE(samples) {
    const buffer = new ArrayBuffer(samples.length * 2);
    const view = new DataView(buffer);
    for (let i = 0; i < samples.length; i++) {
      const s = Math.max(-1, Math.min(1, samples[i]));
      view.setInt16(i * 2, s < 0 ? s * 0x8000 : s * 0x7fff, true);
    }
    return buffer;
  }

  async function tocarPCMChamada(audioCtx, data) {
    const buffer = data instanceof ArrayBuffer ? data : await data.arrayBuffer();
    const view = new DataView(buffer);
    const samples = new Float32Array(buffer.byteLength / 2);
    for (let i = 0; i < samples.length; i++) samples[i] = view.getInt16(i * 2, true) / 0x8000;
    const audioBuffer = audioCtx.createBuffer(1, samples.length, 16000);
    audioBuffer.copyToChannel(samples, 0);
    const src = audioCtx.createBufferSource();
    src.buffer = audioBuffer;
    src.connect(audioCtx.destination);
    src.start();
  }

  // ── Login ─────────────────────────────────────────────────────────────────────
  function mostrarLogin() {
    document.getElementById('login-overlay').classList.remove('hidden');
    setTimeout(() => { const el = document.getElementById('login-token'); if(el) el.focus(); }, 80);
  }
  function ocultarLogin() {
    document.getElementById('login-overlay').classList.add('hidden');
    document.getElementById('login-token').value = '';
    document.getElementById('login-feedback').className = 'modal-feedback hidden';
  }
  document.getElementById('form-login').addEventListener('submit', async function(e) {
    e.preventDefault();
    const btn = document.getElementById('login-submit'), fb = document.getElementById('login-feedback');
    const token = document.getElementById('login-token').value.trim();
    btn.disabled = true; btn.textContent = 'Verificando...'; fb.className = 'modal-feedback hidden';
    try {
      const resp = await fetch('/api/v1/auth/login', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({token})});
      const dados = await resp.json();
      if (!resp.ok) { fb.className = 'modal-feedback error'; fb.textContent = dados.mensagem || 'Token invalido'; return; }
      atualizarEscopoAcesso(dados.dados);
      ocultarLogin();
      await carregarTudo(false);
      if (estadoUI.acesso?.tipo === 'instancia') trocarAba('instancias');
    } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro ao conectar ao servidor.'; }
    finally { btn.disabled = false; btn.textContent = 'Entrar'; }
  });

  // ── Utilitarios ───────────────────────────────────────────────────────────────
  async function chamar(url, opcoes) {
    const resp = await fetch(url, Object.assign({ headers:{'Content-Type':'application/json'} }, opcoes||{}));
    const tipo = resp.headers.get('content-type')||'';
    const dados = tipo.includes('application/json') ? await resp.json() : null;
    if (dados) document.getElementById('retorno-api').textContent = JSON.stringify(dados, null, 2);
    if (resp.status === 401) { mostrarLogin(); return null; }
    if (!resp.ok) throw new Error(dados?.mensagem || 'Erro na requisicao');
    return dados;
  }
  function trocarAba(aba) {
    estadoUI.abaAtual = aba;
    document.querySelectorAll('.nav-link').forEach(i => i.classList.toggle('active', i.dataset.tab === aba));
    document.querySelectorAll('.tab-panel').forEach(i => i.classList.toggle('active', i.dataset.panel === aba));
  }
  function formatarData(v) { return v ? new Date(v).toLocaleString('pt-BR') : '-'; }
  function rotuloStatus(s) { return ({nao_inicializada:'Nao inicializada',desconectada:'Desconectada',conectando:'Conectando',aguardando_qrcode:'Aguardando QR',pareada:'Pareada',autenticando:'Autenticando',sincronizando_historico:'Sincronizando',conectada:'Conectada'})[s]||s||'-'; }
  function classeStatus(s) { if (s==='conectada') return 'online'; if (['aguardando_qrcode','conectando','pareada','autenticando','sincronizando_historico'].includes(s)) return 'processing'; if (['desconectada','nao_inicializada'].includes(s)) return 'offline'; return 'neutro'; }
  function mostrarToast(txt, tipo) { const t=document.getElementById('toast'); t.textContent=txt; t.className='toast '+(tipo||'info'); clearTimeout(t._t); t._t=setTimeout(()=>t.className='toast hidden', 3600); }
  function textoSeguro(v) {
    return String(v || '').replace(/[&<>"']/g, function(c) {
      return ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[c];
    });
  }
  function atualizarEscopoAcesso(acesso) {
    if (acesso) {
      acesso = {
        tipo: acesso.tipo || acesso.Tipo || '',
        instancia_id: acesso.instancia_id || acesso.InstanciaID || '',
        nome: acesso.nome || acesso.Nome || ''
      };
    }
    estadoUI.acesso = acesso || null;
    const instancia = estadoUI.acesso && estadoUI.acesso.tipo === 'instancia';
    if (instancia && estadoUI.acesso.instancia_id && !estadoUI.instanciaSelecionada) {
      estadoUI.instanciaSelecionada = estadoUI.acesso.instancia_id;
    }
    document.getElementById('form-instancia').classList.toggle('hidden', instancia);
    document.getElementById('botao-excluir').classList.toggle('hidden', instancia);
    document.querySelectorAll('[data-tab="atualizacao"]').forEach(el => el.classList.toggle('hidden', instancia));
  }
  function gerarResumoStatus(s,e) { if(e) return 'Ultimo erro: '+e; return ({aguardando_qrcode:'QR code gerado. Escaneie com o WhatsApp para iniciar o pareamento.',pareada:'QR lido. Finalizando vinculo.',autenticando:'Autenticando sessao no WhatsApp.',sincronizando_historico:'Sincronizando historico inicial.',conectada:'Instancia conectada e pronta.',desconectada:'Desconectada. Conecte para usar.',conectando:'Abrindo conexao.'})[s]||'Estado atual da instancia.'; }
  function pairingDisponivel(status) { return ['nao_inicializada','desconectada','aguardando_qrcode'].includes(status); }
  function atualizarBotaoPairing(status) {
    const btn = document.getElementById('botao-pairing');
    const visivel = Boolean(estadoUI.instanciaSelecionada) && pairingDisponivel(status);
    btn.classList.toggle('hidden', !visivel);
    btn.disabled = !visivel;
  }
  async function sairDashboard() {
    try { await fetch('/api/v1/auth/logout', {method:'POST', headers:{'Content-Type':'application/json'}}); } catch(e) {}
    atualizarEscopoAcesso(null);
    atualizarPainelSemSelecao();
    mostrarLogin();
  }
  function mascararToken(token) {
    if (!token) return '-';
    if (token.length <= 12) return token;
    return token.slice(0, 6) + '...' + token.slice(-6);
  }
  function atualizarTokenDetalhado(token, revelar) {
    const el = document.getElementById('detalhe-token');
    el.dataset.token = token || '';
    el.dataset.revelado = revelar ? 'true' : 'false';
    el.textContent = revelar ? (token || '-') : mascararToken(token);
    el.disabled = !token;
    el.classList.toggle('revelado', Boolean(token && revelar));
    el.title = token ? 'Clique para revelar e copiar o token' : 'Token indisponivel';
  }
  async function copiarTexto(texto) {
    if (!texto) throw new Error('Texto vazio');
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(texto);
      return;
    }
    const area = document.createElement('textarea');
    area.value = texto;
    area.style.position = 'fixed';
    area.style.left = '-9999px';
    document.body.appendChild(area);
    area.focus();
    area.select();
    const ok = document.execCommand('copy');
    document.body.removeChild(area);
    if (!ok) throw new Error('Falha ao copiar');
  }
  async function copiarTokenSelecionado() {
    const token = document.getElementById('detalhe-token')?.dataset.token || '';
    if (!token) { mostrarToast('Token nao encontrado para esta instancia.', 'error'); return; }
    try {
      await copiarTexto(token);
      mostrarToast('Token copiado.', 'success');
    } catch(err) {
      mostrarToast('Nao foi possivel copiar o token.', 'error');
    }
  }
  async function revelarECopiarTokenSelecionado() {
    const token = document.getElementById('detalhe-token')?.dataset.token || '';
    if (!token) { mostrarToast('Token nao encontrado para esta instancia.', 'error'); return; }
    atualizarTokenDetalhado(token, true);
    await copiarTokenSelecionado();
  }
  async function copiarPairingCode() {
    const codigo = (document.getElementById('mf-codigo')?.textContent || '').trim();
    if (!codigo || codigo === '-') { mostrarToast('Gere um codigo primeiro.', 'error'); return; }
    try {
      await copiarTexto(codigo);
      mostrarToast('Pairing code copiado.', 'success');
    } catch(err) {
      mostrarToast('Nao foi possivel copiar o pairing code.', 'error');
    }
  }

  // ── Sistema ───────────────────────────────────────────────────────────────────
  function rotuloAtualizacao(s) {
    return ({
      nao_verificado:'Nao verificado',
      atualizado:'Atualizado',
      atualizacao_disponivel:'Atualizacao disponivel',
      verificando:'Verificando',
      preparo_planejado:'Preparo planejado',
      preparo_bloqueado:'Preparo bloqueado',
      falha_verificacao:'Falha na verificacao'
    })[s] || s || '-';
  }
  function renderizarGuia(d) {
    const card=document.getElementById('update-guide-card'), titulo=document.getElementById('update-guide-title'), corpo=document.getElementById('update-guide-body');
    if (d.status_atualizacao === 'falha_verificacao') {
      card.className='glass-card update-card status-error-card'; titulo.textContent='Verificacao falhou';
      corpo.innerHTML='<p class="status-copy">Nao foi possivel consultar a versao mais recente do whatsmeow.</p><ol class="guide-list"><li>Confirme se o servidor tem internet.</li><li>Confira <code>UPDATE_PROXY_URL</code>. Padrao: <code>https://proxy.golang.org</code>.</li><li>Teste no servidor: <code>go list -m -json go.mau.fi/whatsmeow@latest</code>.</li><li>Depois clique em verificar novamente.</li></ol><p class="status-copy"><strong>Erro:</strong> '+(d.ultimo_erro||'nao informado')+'</p>';
      return;
    }
    if (d.atualizacao_disponivel) {
      card.className='glass-card update-card status-pending-card'; titulo.textContent='Atualizacao disponivel';
      corpo.innerHTML='<p class="status-copy">Nova versao do whatsmeow disponivel. Atualizacao exige rebuild e novo processo.</p><ol class="guide-list"><li><code>go get go.mau.fi/whatsmeow@'+(d.ultima_versao_disponivel||'latest')+'</code></li><li><code>go mod tidy</code></li><li><code>go test ./...</code></li><li><code>go build ./cmd/api</code></li><li>Reinicie/substitua o processo da API.</li><li>Valide reconexao, envio e webhooks.</li></ol>';
    } else {
      card.className='glass-card update-card status-ok-card'; titulo.textContent='Sistema atualizado';
      corpo.innerHTML='<p class="status-copy">Nenhuma atualizacao pendente. A versao em uso esta igual a ultima versao consultada.</p>';
    }
  }
  async function carregarSistema() {
    if (!estadoUI.acesso || estadoUI.acesso.tipo !== 'master') {
      document.getElementById('alerta-atualizacao').className = 'banner hidden';
      return;
    }
    const [versao, atualiz] = await Promise.all([chamar('/api/v1/sistema/versao'), chamar('/api/v1/sistema/atualizacoes')]);
    if (!versao || !atualiz) return;
    document.getElementById('app-versao').textContent = versao.dados.aplicacao_versao;
    document.getElementById('wm-versao').textContent  = versao.dados.whatsmeow_versao;
    document.getElementById('update-versao-atual').textContent      = atualiz.dados.versao_em_uso || '-';
    document.getElementById('update-versao-disponivel').textContent = atualiz.dados.ultima_versao_disponivel || 'Sem novidades';
    document.getElementById('update-modo').textContent              = atualiz.dados.modo_operacao || '-';
    document.getElementById('update-status').textContent            = rotuloAtualizacao(atualiz.dados.status_atualizacao);
    document.getElementById('update-verificacao').textContent       = formatarData(atualiz.dados.ultima_verificacao_em);
    document.getElementById('update-erro').textContent              = atualiz.dados.ultimo_erro || '-';
    if (atualiz.dados.status_atualizacao === 'falha_verificacao') {
      document.getElementById('update-copy').textContent = 'Falha ao consultar a ultima versao. Veja o erro e o guia ao lado.';
    } else {
      document.getElementById('update-copy').textContent = atualiz.dados.atualizacao_disponivel ? 'Nova versao disponivel. Siga o guia ao lado.' : 'Tudo em dia.';
    }
    renderizarGuia(atualiz.dados);
    const alerta = document.getElementById('alerta-atualizacao');
    if (atualiz.dados.status_atualizacao === 'falha_verificacao') {
      alerta.className = 'banner warning';
      alerta.innerHTML = '<strong>Falha na verificacao de atualizacao:</strong> ' + (atualiz.dados.ultimo_erro || 'erro nao informado') + '.';
    } else if (atualiz.dados.atualizacao_disponivel) {
      alerta.className = 'banner warning';
      alerta.innerHTML = '<strong>Atualizacao disponivel:</strong> em uso ' + atualiz.dados.versao_em_uso + ', disponivel ' + atualiz.dados.ultima_versao_disponivel + '.';
    } else { alerta.className = 'banner hidden'; alerta.textContent = ''; }
  }
  async function verificarAtualizacoes() {
    try {
      const r = await chamar('/api/v1/sistema/atualizacoes/verificar', {method:'POST'});
      if (!r) return;
      await carregarSistema();
      const d = r.dados || {};
      if (d.status_atualizacao === 'falha_verificacao') {
        mostrarToast('Falha na verificacao: ' + (d.ultimo_erro || 'erro nao informado'), 'error');
      } else if (d.atualizacao_disponivel) {
        mostrarToast('Atualizacao disponivel: ' + d.ultima_versao_disponivel, 'info');
      } else {
        mostrarToast('Verificacao concluida. Sistema atualizado.', 'success');
      }
    } catch(err) {
      mostrarToast('Erro ao verificar atualizacao: ' + err.message, 'error');
    }
  }

  // ── Instancias ────────────────────────────────────────────────────────────────
  async function carregarInstancias(manual) {
    const resp = await chamar('/api/v1/instancias');
    if (!resp) return;
    const lista = resp.dados.instancias || [];
    document.getElementById('contador-instancias').textContent = String(lista.length);
    document.getElementById('contador-online').textContent     = String(lista.filter(i=>i.status==='conectada').length);
    document.getElementById('auto-sync-status').textContent    = manual ? 'manual' : 'auto';
    if (!lista.length) {
      if (estadoUI.acesso?.tipo === 'instancia' && estadoUI.acesso.instancia_id) {
        estadoUI.instanciaSelecionada = estadoUI.acesso.instancia_id;
        document.getElementById('contador-instancias').textContent = '1';
        document.getElementById('lista-instancias').className = 'instance-list';
        document.getElementById('lista-instancias').innerHTML =
          '<button class="instance-item selected" onclick="selecionarInstancia(\''+textoSeguro(estadoUI.acesso.instancia_id)+'\')">'+
            '<span class="instance-top"><strong>'+textoSeguro(estadoUI.acesso.nome || 'Instancia vinculada')+'</strong><span class="mini-status neutro">Carregando</span></span>'+
            '<span class="instance-id">'+textoSeguro(estadoUI.acesso.instancia_id)+'</span></button>';
        await atualizarInstanciaSelecionada(false);
        return;
      }
      document.getElementById('lista-instancias').className = 'instance-list empty-state';
      document.getElementById('lista-instancias').innerHTML = 'Nenhuma instancia cadastrada.';
      atualizarPainelSemSelecao(); return;
    }
    if (!estadoUI.instanciaSelecionada && estadoUI.acesso?.tipo === 'instancia' && lista.length === 1) {
      estadoUI.instanciaSelecionada = lista[0].id;
    }
    if (estadoUI.instanciaSelecionada && !lista.some(i=>i.id===estadoUI.instanciaSelecionada)) atualizarPainelSemSelecao();
    document.getElementById('lista-instancias').className = 'instance-list';
    document.getElementById('lista-instancias').innerHTML = lista.map(i =>
      '<button class="instance-item'+(i.id===estadoUI.instanciaSelecionada?' selected':'')+'" onclick="selecionarInstancia(\''+i.id+'\')">'+
        '<span class="instance-top"><strong>'+textoSeguro(i.nome)+'</strong><span class="mini-status '+classeStatus(i.status)+'">'+textoSeguro(rotuloStatus(i.status))+'</span></span>'+
        '<span class="instance-id">'+textoSeguro(i.id)+'</span></button>'
    ).join('');
    if (estadoUI.instanciaSelecionada) await atualizarInstanciaSelecionada(false);
  }

  const BOTOES_INSTANCIA = ['botao-conectar','botao-desconectar','botao-ver-qr','botao-excluir','botao-pairing','botao-copiar-token','botao-enviar-texto','botao-teste-chamada','botao-config-token','botao-config-proxy','botao-config-historico','botao-novo-webhook','botao-salvar-avancado'];
  const CAMPOS_AVANCADO = ['adv-manter-online','adv-rejeitar-chamadas','adv-mensagem-rejeitar','adv-marcar-lida','adv-ignorar-grupos','adv-ignorar-status'];

  function atualizarPainelSemSelecao() {
    estadoUI.instanciaSelecionada = '';
    document.getElementById('instancia-titulo').textContent = 'Nenhuma instancia selecionada';
    ['status-badge','status-badge-home'].forEach(id => { document.getElementById(id).className='status-badge neutro'; document.getElementById(id).textContent='Sem selecao'; });
    ['detalhe-id','detalhe-status','detalhe-atualizado','detalhe-erro'].forEach(id => document.getElementById(id).textContent='-');
    atualizarTokenDetalhado('', false);
    document.getElementById('status-copy').textContent = 'Clique em uma instancia da lista para operar.';
    document.getElementById('home-instancia-nome').textContent = 'Nenhuma instancia selecionada';
    ['home-instancia-status','home-instancia-atualizado','home-instancia-erro'].forEach(id => document.getElementById(id).textContent='-');
    document.getElementById('home-status-copy').textContent = 'Clique em uma instancia para ver QR code e acoes.';
    document.getElementById('qrcode-box').textContent = 'Clique em uma instancia para visualizar o QR code.';
    document.getElementById('lista-webhooks').className = 'webhook-list empty-state';
    document.getElementById('lista-webhooks').textContent = 'Selecione uma instancia para ver os webhooks.';
    const auditoria = document.getElementById('lista-entregas-webhook');
    if (auditoria) {
      auditoria.className = 'webhook-delivery-list empty-state';
      auditoria.textContent = 'Selecione uma instancia para ver a auditoria.';
    }
    const resumoAuditoria = document.getElementById('resumo-entregas-webhook');
    if (resumoAuditoria) {
      resumoAuditoria.className = 'delivery-summary hidden';
      resumoAuditoria.innerHTML = '';
    }
    BOTOES_INSTANCIA.forEach(id => document.getElementById(id).disabled = true);
    const botaoAuditoria = document.getElementById('botao-atualizar-auditoria');
    if (botaoAuditoria) botaoAuditoria.disabled = true;
    preencherConfiguracaoAvancada({}, true);
    CAMPOS_AVANCADO.forEach(id => document.getElementById(id).disabled = true);
    document.getElementById('advanced-copy').textContent = 'Selecione uma instancia para editar.';
    alternarAvancado(false);
    alternarAuditoria(false);
    document.getElementById('botao-pairing').classList.add('hidden');
    pararPollingConexao();
  }

  async function selecionarInstancia(id) {
    estadoUI.instanciaSelecionada = id;
    trocarAba('instancias');
    await carregarInstancias(true);
    await carregarWebhooks();
    if (auditoriaAberta()) await carregarAuditoria(false);
  }

  async function atualizarInstanciaSelecionada(mostrarErro) {
    if (!estadoUI.instanciaSelecionada) { atualizarPainelSemSelecao(); return; }
    try {
      const resp = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/status');
      if (!resp) return;
      const d = resp.dados, cls = classeStatus(d.status), rot = rotuloStatus(d.status);
      document.getElementById('instancia-titulo').textContent = d.nome;
      ['status-badge','status-badge-home'].forEach(id => { document.getElementById(id).className='status-badge '+cls; document.getElementById(id).textContent=rot; });
      ['detalhe-id','detalhe-status','detalhe-atualizado','detalhe-erro'].forEach((id,i) => document.getElementById(id).textContent=[d.id, rot, formatarData(d.atualizado_em), d.erro||'-'][i]);
      atualizarTokenDetalhado(d.token, false);
      document.getElementById('status-copy').textContent = gerarResumoStatus(d.status, d.erro);
      document.getElementById('home-instancia-nome').textContent      = d.nome;
      document.getElementById('home-instancia-status').textContent    = rot;
      document.getElementById('home-instancia-atualizado').textContent= formatarData(d.atualizado_em);
      document.getElementById('home-instancia-erro').textContent      = d.erro||'-';
      document.getElementById('home-status-copy').textContent         = gerarResumoStatus(d.status, d.erro);
      BOTOES_INSTANCIA.forEach(id => document.getElementById(id).disabled = false);
      const botaoAuditoria = document.getElementById('botao-atualizar-auditoria');
      if (botaoAuditoria) botaoAuditoria.disabled = false;
      CAMPOS_AVANCADO.forEach(id => document.getElementById(id).disabled = false);
      preencherConfiguracaoAvancada(d.configuracao_avancada || {});
      atualizarBotaoPairing(d.status);
      if (d.status === 'aguardando_qrcode') { await mostrarQRCodeSelecionado(); iniciarPollingConexao(); }
      else if (['conectando','pareada','autenticando','sincronizando_historico'].includes(d.status)) {
        iniciarPollingConexao();
        document.getElementById('qrcode-box').innerHTML = '<div class="qrcode-loading">'+gerarResumoStatus(d.status,d.erro)+'</div>';
      } else {
        pararPollingConexao();
        if (d.status==='conectada') document.getElementById('qrcode-box').innerHTML = '<div class="qrcode-success">Instancia conectada e pronta para uso.</div>';
      }
    } catch(err) { if (mostrarErro) mostrarToast(err.message, 'error'); }
  }

  function preencherConfiguracaoAvancada(cfg, forcar) {
    const form = document.getElementById('form-avancado');
    if (!forcar && form && form.contains(document.activeElement) && document.activeElement.id !== 'botao-salvar-avancado') {
      return;
    }
    document.getElementById('adv-manter-online').checked = Boolean(cfg.manter_online);
    document.getElementById('adv-rejeitar-chamadas').checked = Boolean(cfg.rejeitar_chamadas);
    document.getElementById('adv-mensagem-rejeitar').value = cfg.mensagem_rejeitar_chamadas || '';
    document.getElementById('adv-marcar-lida').checked = Boolean(cfg.marcar_lida_automatico);
    document.getElementById('adv-ignorar-grupos').checked = Boolean(cfg.ignorar_grupos);
    document.getElementById('adv-ignorar-status').checked = Boolean(cfg.ignorar_status);
    atualizarVisibilidadeMensagemChamada();
    if (estadoUI.instanciaSelecionada) document.getElementById('advanced-copy').textContent = 'Configuracao carregada. Salve para aplicar alteracoes.';
  }

  function atualizarVisibilidadeMensagemChamada() {
    const ligado = document.getElementById('adv-rejeitar-chamadas').checked;
    document.getElementById('adv-mensagem-wrap').classList.toggle('hidden', !ligado);
  }

  function alternarAvancado(forcarAberto) {
    const card = document.getElementById('advanced-card');
    const form = document.getElementById('form-avancado');
    const botao = document.getElementById('botao-toggle-avancado');
    if (!card || !form || !botao) return;
    const abrir = typeof forcarAberto === 'boolean' ? forcarAberto : form.classList.contains('hidden');
    form.classList.toggle('hidden', !abrir);
    card.classList.toggle('collapsed', !abrir);
    card.classList.toggle('expanded', abrir);
    botao.textContent = abrir ? 'Recolher' : 'Abrir';
  }

  function auditoriaAberta() {
    const body = document.getElementById('audit-body');
    return Boolean(body && !body.classList.contains('hidden'));
  }

  function alternarAuditoria(forcarAberto) {
    const card = document.getElementById('audit-card');
    const body = document.getElementById('audit-body');
    const botao = document.getElementById('botao-toggle-auditoria');
    if (!card || !body || !botao) return;
    const abrir = typeof forcarAberto === 'boolean' ? forcarAberto : body.classList.contains('hidden');
    body.classList.toggle('hidden', !abrir);
    card.classList.toggle('collapsed', !abrir);
    card.classList.toggle('expanded', abrir);
    botao.textContent = abrir ? 'Recolher' : 'Abrir';
    if (abrir && estadoUI.instanciaSelecionada) carregarAuditoria(false);
  }

  async function salvarConfiguracaoAvancada(e) {
    e.preventDefault();
    if (!estadoUI.instanciaSelecionada) return;
    const btn = document.getElementById('botao-salvar-avancado');
    btn.disabled = true; btn.textContent = 'Salvando...';
    try {
      await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/avancado', {
        method: 'PUT',
        body: JSON.stringify({
          manter_online: document.getElementById('adv-manter-online').checked,
          rejeitar_chamadas: document.getElementById('adv-rejeitar-chamadas').checked,
          mensagem_rejeitar_chamadas: document.getElementById('adv-mensagem-rejeitar').value,
          marcar_lida_automatico: document.getElementById('adv-marcar-lida').checked,
          ignorar_grupos: document.getElementById('adv-ignorar-grupos').checked,
          ignorar_status: document.getElementById('adv-ignorar-status').checked
        })
      });
      document.getElementById('advanced-copy').textContent = 'Configuracao avancada salva.';
      mostrarToast('Configuracao avancada salva.', 'success');
      await atualizarInstanciaSelecionada(false);
    } catch(err) {
      mostrarToast(err.message, 'error');
      document.getElementById('advanced-copy').textContent = 'Erro ao salvar: ' + err.message;
    } finally {
      btn.disabled = false; btn.textContent = 'Salvar avancado';
    }
  }

  async function conectarSelecionada()    { if (!estadoUI.instanciaSelecionada) return; document.getElementById('qrcode-box').innerHTML='<div class="qrcode-loading">Gerando QR code...</div>'; const r=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/conectar',{method:'POST'}); if(r){await carregarInstancias(true);iniciarPollingConexao();} }
  async function desconectarSelecionada() { if (!estadoUI.instanciaSelecionada) return; const r=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/desconectar',{method:'POST'}); if(r){mostrarToast('Instancia desconectada.','info');document.getElementById('qrcode-box').textContent='Instancia desconectada.';pararPollingConexao();await carregarInstancias(true);} }
  async function excluirSelecionada()     { if (!estadoUI.instanciaSelecionada) return; if (!window.confirm('Excluir esta instancia e remover a sessao local?')) return; const id=estadoUI.instanciaSelecionada; const r=await chamar('/api/v1/instancias/'+id,{method:'DELETE'}); if(r){mostrarToast('Instancia excluida.','success');atualizarPainelSemSelecao();await carregarInstancias(true);} }
  async function mostrarQRCodeSelecionado() {
    if (!estadoUI.instanciaSelecionada) return;
    const resp=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/qrcode');
    if (!resp) return;
    const cod=resp.dados.qrcode||'', box=document.getElementById('qrcode-box');
    if (!cod) { box.innerHTML='<div class="qrcode-placeholder">QR code indisponivel.</div>'; return; }
    box.innerHTML='<img src="/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/qrcode/imagem?ts='+Date.now()+'" alt="QR code" class="qrcode-image"><p class="qrcode-caption">Escaneie com o WhatsApp.</p>';
  }
  function iniciarPollingConexao() { if(estadoUI.pollingConexao) return; estadoUI.pollingConexao=setInterval(async function(){if(!estadoUI.instanciaSelecionada)return;await atualizarInstanciaSelecionada(false);await carregarInstancias(false);},2500); }
  function pararPollingConexao()   { if(estadoUI.pollingConexao){clearInterval(estadoUI.pollingConexao);estadoUI.pollingConexao=null;} }

  // ── Webhooks ──────────────────────────────────────────────────────────────────
  async function carregarWebhooks() {
    if (!estadoUI.instanciaSelecionada) return;
    const resp = await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/webhooks');
    if (!resp) return;
    const lista = resp.dados.webhooks || [];
    estadoUI.webhooks = lista;
    const box = document.getElementById('lista-webhooks');
    if (!lista.length) { box.className='webhook-list empty-state'; box.textContent='Nenhum webhook configurado. Clique em "+ Adicionar".'; return; }
    box.className = 'webhook-list';
    box.innerHTML = lista.map(wh =>
      '<div class="webhook-item">' +
        '<div class="webhook-top">' +
          '<strong>'+textoSeguro(wh.nome)+'</strong>' +
          '<span class="mini-status '+(wh.ativo?'online':'offline')+'">'+(wh.ativo?'Ativo':'Inativo')+'</span>' +
        '</div>' +
        '<div class="webhook-url">'+textoSeguro(wh.url)+'</div>' +
        '<div class="webhook-eventos">'+textoSeguro((wh.eventos||[]).join(', '))+'</div>' +
        '<div class="webhook-actions">' +
          '<button class="ghost small" onclick="editarWebhook(\''+wh.id+'\')">Editar</button>' +
          '<button class="ghost small" onclick="alternarWebhookAtivo(\''+wh.id+'\')">'+(wh.ativo?'Desativar':'Ativar')+'</button>' +
          '<button class="ghost small" onclick="excluirWebhook(\''+wh.id+'\')">Excluir</button>' +
        '</div>' +
      '</div>'
    ).join('');
  }

  async function carregarAuditoria(manual) {
    const box = document.getElementById('lista-entregas-webhook');
    if (!box) return;
    if (!estadoUI.instanciaSelecionada) {
      box.className = 'webhook-delivery-list empty-state';
      box.textContent = 'Selecione uma instancia para ver a auditoria.';
      return;
    }
    if (manual) box.innerHTML = '<div class="qrcode-loading">Consultando entregas...</div>';
    try {
      const resp = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhook-entregas?limite=60');
      const lista = resp?.dados?.entregas || [];
      estadoUI.entregasWebhook = lista;
      renderizarEntregasWebhook(lista);
    } catch(err) {
      box.className = 'webhook-delivery-list empty-state';
      box.textContent = 'Erro ao carregar auditoria: ' + err.message;
    }
  }

  function renderizarEntregasWebhook(lista) {
    const box = document.getElementById('lista-entregas-webhook');
    const resumo = document.getElementById('resumo-entregas-webhook');
    if (!box) return;
    if (!lista.length) {
      if (resumo) { resumo.className = 'delivery-summary hidden'; resumo.innerHTML = ''; }
      box.className = 'webhook-delivery-list empty-state';
      box.textContent = 'Nenhuma entrega de webhook registrada para esta instancia.';
      return;
    }
    if (resumo) {
      const total = lista.length;
      const entregue = lista.filter(e => e.status === 'entregue').length;
      const fila = lista.filter(e => e.status === 'pendente' || e.status === 'enviando').length;
      const falha = lista.filter(e => e.status === 'falha').length;
      const esgotada = lista.filter(e => e.status === 'esgotada').length;
      resumo.className = 'delivery-summary';
      resumo.innerHTML =
        '<div><span>Total</span><strong>'+total+'</strong></div>' +
        '<div><span>Entregues</span><strong class="ok">'+entregue+'</strong></div>' +
        '<div><span>Na fila</span><strong class="wait">'+fila+'</strong></div>' +
        '<div><span>Falhas</span><strong class="warn">'+falha+'</strong></div>' +
        '<div><span>Esgotadas</span><strong class="bad">'+esgotada+'</strong></div>';
    }
    box.className = 'webhook-delivery-list';
    box.innerHTML = lista.map(entrega => {
      const erro = entrega.ultimo_erro ? '<div class="delivery-error">'+textoSeguro(entrega.ultimo_erro)+'</div>' : '';
      return '<article class="delivery-item">' +
        '<div class="delivery-top">' +
          '<div><strong>'+textoSeguro(entrega.webhook_nome || entrega.url)+'</strong><span>'+textoSeguro(entrega.evento || '-')+' · '+textoSeguro(entrega.id || '-')+'</span></div>' +
          '<span class="delivery-status '+classeEntregaWebhook(entrega.status)+'">'+textoSeguro(rotuloEntregaWebhook(entrega.status))+'</span>' +
        '</div>' +
        '<div class="delivery-url">'+textoSeguro(entrega.url || '-')+'</div>' +
        '<div class="delivery-grid">' +
          '<div><span>Tentativas</span><strong>'+textoSeguro(entrega.tentativas || 0)+'/'+textoSeguro(entrega.max_tentativas || 0)+'</strong></div>' +
          '<div><span>HTTP</span><strong>'+(entrega.status_http ? textoSeguro(entrega.status_http) : '-')+'</strong></div>' +
          '<div><span>Ultima tentativa</span><strong>'+textoSeguro(formatarData(entrega.ultima_tentativa_em))+'</strong></div>' +
          '<div><span>Proxima tentativa</span><strong>'+textoSeguro(formatarData(entrega.proxima_tentativa_em))+'</strong></div>' +
          '<div><span>Criada em</span><strong>'+textoSeguro(formatarData(entrega.criado_em))+'</strong></div>' +
          '<div><span>Atualizada em</span><strong>'+textoSeguro(formatarData(entrega.atualizado_em))+'</strong></div>' +
        '</div>' + erro +
      '</article>';
    }).join('');
  }

  function rotuloEntregaWebhook(status) {
    return ({pendente:'Pendente', enviando:'Enviando', entregue:'Entregue', falha:'Falha', esgotada:'Esgotada'})[status] || status || '-';
  }

  function classeEntregaWebhook(status) {
    if (status === 'entregue') return 'online';
    if (status === 'pendente' || status === 'enviando') return 'processing';
    if (status === 'falha') return 'warning';
    if (status === 'esgotada') return 'offline';
    return 'neutro';
  }
  function editarWebhook(whId) {
    const wh = (estadoUI.webhooks || []).find(item => item.id === whId);
    if (!wh) { mostrarToast('Webhook nao encontrado na lista atual.', 'error'); return; }
    estadoUI.webhookEditando = wh;
    abrirModal('webhook-editar');
  }
  async function alternarWebhookAtivo(whId) {
    const wh = (estadoUI.webhooks || []).find(item => item.id === whId);
    if (!wh) { mostrarToast('Webhook nao encontrado na lista atual.', 'error'); return; }
    const novoAtivo = !wh.ativo;
    try {
      await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhooks/' + wh.id, {
        method: 'PUT',
        body: JSON.stringify({
          nome: wh.nome,
          url: wh.url,
          eventos: wh.eventos || [],
          ativo: novoAtivo
        })
      });
      mostrarToast(novoAtivo ? 'Webhook ativado.' : 'Webhook desativado.', 'success');
      await carregarWebhooks();
    } catch(err) {
      mostrarToast(err.message, 'error');
    }
  }
  async function excluirWebhook(whId) {
    if (!window.confirm('Excluir este webhook?')) return;
    const r=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/webhooks/'+whId,{method:'DELETE'});
    if(r){mostrarToast('Webhook excluido.','success');await carregarWebhooks();}
  }

  // ── Carregamento geral ────────────────────────────────────────────────────────
  async function carregarTudo(manual) {
    if (estadoUI.acesso?.tipo === 'master') await carregarSistema();
    await carregarInstancias(Boolean(manual));
    if (estadoUI.instanciaSelecionada) await carregarWebhooks();
    if (estadoUI.instanciaSelecionada && auditoriaAberta()) await carregarAuditoria(false);
  }

  document.getElementById('form-instancia').addEventListener('submit', async function(e) {
    e.preventDefault();
    const nome = document.getElementById('nome-instancia').value;
    const r = await chamar('/api/v1/instancias', {method:'POST', body:JSON.stringify({nome})});
    if (r) { document.getElementById('nome-instancia').value=''; mostrarToast('Instancia criada. Clique nela para abrir.','success'); await carregarTudo(true); trocarAba('instancias'); }
  });
  document.getElementById('adv-rejeitar-chamadas').addEventListener('change', atualizarVisibilidadeMensagemChamada);
  document.getElementById('form-avancado').addEventListener('submit', salvarConfiguracaoAvancada);

  estadoUI.refreshGeral = setInterval(() => carregarTudo(false), 5000);
  atualizarPainelSemSelecao();
  trocarAba('dashboard');
  // Verifica sessao antes de carregar o dashboard
  (async function() {
    const r = await fetch('/api/v1/auth/sessao', {headers:{'Content-Type':'application/json'}});
    if (r.status === 401) { mostrarLogin(); return; }
    const sessao = await r.json();
    atualizarEscopoAcesso(sessao.dados);
    await carregarTudo(false);
    if (estadoUI.acesso?.tipo === 'instancia') trocarAba('instancias');
  })();
  </script>
</body>
</html>`
