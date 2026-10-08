package dashboard

const paginaInicialHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="icon" href="/static/img/favicon-32.png" sizes="32x32" type="image/png">
  <link rel="icon" href="/static/img/favicon-64.png" sizes="64x64" type="image/png">
  <link rel="apple-touch-icon" href="/static/img/dyalog-connect.png">
  <title>{{ .Titulo }}</title>
  <script>document.documentElement.setAttribute('data-theme',localStorage.getItem('tema')||'');</script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700" rel="stylesheet">
  <link rel="stylesheet" href="/static/css/dashboard.css?v=20261009">
</head>
<body>
  <svg xmlns="http://www.w3.org/2000/svg" style="display:none">
    <symbol id="i-dashboard" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/></symbol>
    <symbol id="i-celular" viewBox="0 0 24 24"><rect x="6" y="2.5" width="12" height="19" rx="2.5"/><path d="M11 18h2"/></symbol>
    <symbol id="i-atualizar" viewBox="0 0 24 24"><path d="M20 11a8 8 0 0 0-14.3-4.9L4 8"/><path d="M4 3v5h5"/><path d="M4 13a8 8 0 0 0 14.3 4.9L20 16"/><path d="M20 21v-5h-5"/></symbol>
    <symbol id="i-livro" viewBox="0 0 24 24"><path d="M4 19.5V5a2 2 0 0 1 2-2h14v15H6a2 2 0 0 0-2 2Z"/><path d="M6 18h14v3H6a2 2 0 0 1 0-4"/></symbol>
    <symbol id="i-sair" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="m16 17 5-5-5-5"/><path d="M21 12H9"/></symbol>
    <symbol id="i-sol" viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></symbol>
    <symbol id="i-contraste" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="M12 3a9 9 0 0 1 0 18Z" fill="currentColor"/></symbol>
    <symbol id="i-lua" viewBox="0 0 24 24"><path d="M20 14.5A8 8 0 1 1 9.5 4a6.5 6.5 0 0 0 10.5 10.5Z"/></symbol>
    <symbol id="i-mais" viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></symbol>
    <symbol id="i-busca" viewBox="0 0 24 24"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></symbol>
    <symbol id="i-voltar" viewBox="0 0 24 24"><path d="M19 12H5"/><path d="m12 19-7-7 7-7"/></symbol>
    <symbol id="i-mensagem" viewBox="0 0 24 24"><path d="M21 12a8 8 0 0 1-11.6 7.1L4 20l1-4.6A8 8 0 1 1 21 12Z"/></symbol>
    <symbol id="i-pessoa-mais" viewBox="0 0 24 24"><circle cx="9" cy="8" r="4"/><path d="M2 21a7 7 0 0 1 14 0"/><path d="M19 8v6M16 11h6"/></symbol>
    <symbol id="i-online" viewBox="0 0 24 24"><path d="M5 12.5a10 10 0 0 1 14 0"/><path d="M8.5 16a5 5 0 0 1 7 0"/><path d="M2 9a14.5 14.5 0 0 1 20 0"/><circle cx="12" cy="19.5" r="1" fill="currentColor"/></symbol>
    <symbol id="i-alerta" viewBox="0 0 24 24"><path d="M10.3 3.9 2.4 17.5A2 2 0 0 0 4.1 20.5h15.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z"/><path d="M12 9v4M12 17h.01"/></symbol>
    <symbol id="i-ok" viewBox="0 0 24 24"><circle cx="12" cy="12" r="9"/><path d="m8 12 3 3 5-6"/></symbol>
    <symbol id="i-seta" viewBox="0 0 24 24"><path d="m9 18 6-6-6-6"/></symbol>
    <symbol id="i-pessoa" viewBox="0 0 24 24"><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></symbol>
    <symbol id="i-chave" viewBox="0 0 24 24"><circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 8.8-8.8M16 7l3 3M18.5 4.5l2 2"/></symbol>
    <symbol id="i-olho" viewBox="0 0 24 24"><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/></symbol>
    <symbol id="i-copiar" viewBox="0 0 24 24"><rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h10a1 1 0 0 1 1 1v1"/></symbol>
    <symbol id="i-telefone" viewBox="0 0 24 24"><path d="M22 16.9v3a2 2 0 0 1-2.2 2 19.8 19.8 0 0 1-8.6-3.1 19.5 19.5 0 0 1-6-6A19.8 19.8 0 0 1 2.1 4.2 2 2 0 0 1 4.1 2h3a2 2 0 0 1 2 1.7c.1 1 .4 1.9.7 2.8a2 2 0 0 1-.5 2.1L8 9.9a16 16 0 0 0 6 6l1.3-1.3a2 2 0 0 1 2.1-.5c.9.3 1.8.6 2.8.7a2 2 0 0 1 1.7 2Z"/></symbol>
    <symbol id="i-qr" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><path d="M14 14h3v3M21 14v.01M17 21h4v-4M14 21v-.01"/></symbol>
    <symbol id="i-engrenagem" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 0 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 0 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 0 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 0 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1Z"/></symbol>
    <symbol id="i-clipe" viewBox="0 0 24 24"><path d="m21.4 11.1-9.2 9.2a6 6 0 0 1-8.5-8.5l9.2-9.2a4 4 0 0 1 5.7 5.7l-9.2 9.2a2 2 0 0 1-2.8-2.8l8.5-8.5"/></symbol>
    <symbol id="i-enviar" viewBox="0 0 24 24"><path d="M22 2 11 13"/><path d="M22 2 15 22l-4-9-9-4Z"/></symbol>
    <symbol id="i-link" viewBox="0 0 24 24"><path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/></symbol>
  </svg>
  <div class="dc-layout">
    <aside class="dc-sidebar">
      <div class="dc-marca">
        <img src="/static/img/dyalog-connect.png" alt="" class="dc-marca-logo">
        <div><strong>Dyalog Connect</strong><span>Painel de instancias</span></div>
      </div>
      <nav class="dc-nav hidden" id="nav-instancia" aria-label="Menu da instancia">
        <div class="dc-nav-inst">
          <div class="dc-avatar dc-avatar-mini" id="nav-inst-avatar"><svg class="dc-ico"><use href="#i-pessoa"/></svg></div>
          <div class="dc-nav-inst-texto"><strong id="nav-inst-nome">-</strong><span id="nav-inst-sub">-</span></div>
        </div>
        <button class="nav-link so-master" onclick="trocarAba('instancias')"><svg class="dc-ico"><use href="#i-voltar"/></svg>Voltar</button>
        <p class="dc-nav-secao">Geral</p>
        <button class="nav-link" data-aba-inst="geral" onclick="trocarAbaInstancia('geral')"><svg class="dc-ico"><use href="#i-dashboard"/></svg>Visao geral</button>
        <button class="nav-link" data-aba-inst="chat" onclick="trocarAbaInstancia('chat')"><svg class="dc-ico"><use href="#i-mensagem"/></svg>Chat</button>
        <p class="dc-nav-secao">Conexoes</p>
        <button class="nav-link" data-aba-inst="webhooks" onclick="trocarAbaInstancia('webhooks')"><svg class="dc-ico"><use href="#i-link"/></svg>Webhooks</button>
        <p class="dc-nav-secao">Gerenciar</p>
        <button class="nav-link" data-aba-inst="config" onclick="trocarAbaInstancia('config')"><svg class="dc-ico"><use href="#i-engrenagem"/></svg>Configuracoes</button>
        <button class="nav-link" data-aba-inst="ferramentas" onclick="trocarAbaInstancia('ferramentas')"><svg class="dc-ico"><use href="#i-mensagem"/></svg>Ferramentas</button>
        <a class="nav-link" href="/docs" target="_blank" rel="noopener"><svg class="dc-ico"><use href="#i-livro"/></svg>Documentacao</a>
      </nav>
      <nav class="dc-nav" id="nav-principal" aria-label="Menu principal">
        <p class="dc-nav-secao">Visao geral</p>
        <button class="nav-link" data-tab="dashboard" onclick="trocarAba('dashboard')"><svg class="dc-ico"><use href="#i-dashboard"/></svg>Dashboard</button>
        <button class="nav-link" data-tab="instancias" onclick="trocarAba('instancias')"><svg class="dc-ico"><use href="#i-celular"/></svg>Instancias<span class="dc-nav-contador" id="nav-contador-instancias">0</span></button>
        <p class="dc-nav-secao so-master">Sistema</p>
        <button class="nav-link so-master" data-tab="atualizacao" onclick="trocarAba('atualizacao')"><svg class="dc-ico"><use href="#i-atualizar"/></svg>Atualizacao</button>
        <a class="nav-link" href="/docs" target="_blank" rel="noopener"><svg class="dc-ico"><use href="#i-livro"/></svg>Documentacao</a>
      </nav>
      <div class="dc-sidebar-rodape">
        <div class="dc-tema" role="group" aria-label="Tema">
          <button class="theme-btn" data-tema=""      onclick="setTema('')"      title="Azul"><svg class="dc-ico"><use href="#i-lua"/></svg></button>
          <button class="theme-btn" data-tema="clean" onclick="setTema('clean')" title="Claro"><svg class="dc-ico"><use href="#i-sol"/></svg></button>
          <button class="theme-btn" data-tema="dark"  onclick="setTema('dark')"  title="Escuro"><svg class="dc-ico"><use href="#i-contraste"/></svg></button>
        </div>
        <button class="nav-link" onclick="sairDashboard()"><svg class="dc-ico"><use href="#i-sair"/></svg>Sair</button>
      </div>
    </aside>

    <main class="dc-conteudo">
      <header class="dc-cabecalho">
        <nav class="dc-trilha" aria-label="Voce esta em"><span>Dyalog Connect</span><span class="dc-trilha-sep">/</span><span id="trilha-pagina">Dashboard</span></nav>
        <button class="ghost small dc-botao-icone" onclick="carregarTudo(true)" title="Atualizar dados"><svg class="dc-ico"><use href="#i-atualizar"/></svg></button>
      </header>

      <section id="alerta-atualizacao" class="banner hidden"></section>
      <section id="toast" class="toast hidden"></section>

      <!-- ===== DASHBOARD ===== -->
      <section class="tab-panel active" data-panel="dashboard">
        <div class="dc-pagina-topo">
          <div><h1 id="dash-saudacao">Ola</h1><p>Resumo das instancias e dos envios de hoje.</p></div>
          <button class="primary so-master" onclick="abrirModal('nova-instancia')"><svg class="dc-ico"><use href="#i-mais"/></svg>Nova instancia</button>
        </div>
        <section class="dc-kpis">
          <article class="dc-kpi"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-celular"/></svg></span><div><span>Instancias</span><strong id="contador-instancias">0</strong></div></article>
          <article class="dc-kpi"><span class="dc-kpi-icone ok"><svg class="dc-ico"><use href="#i-online"/></svg></span><div><span>Conectadas</span><strong id="contador-online">0</strong></div></article>
          <article class="dc-kpi"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-mensagem"/></svg></span><div><span>Mensagens hoje</span><strong id="kpi-envios">0</strong></div></article>
          <article class="dc-kpi"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-pessoa-mais"/></svg></span><div><span>Contatos novos hoje</span><strong id="kpi-contatos">0</strong></div></article>
        </section>
        <section class="dc-dash-grade">
          <article class="dc-card">
            <div class="dc-card-topo"><h3>Precisa de atencao</h3><span class="helper-text">Instancias fora do ar ou com risco de bloqueio</span></div>
            <div id="dash-atencao" class="dc-lista"></div>
          </article>
          <article class="dc-card">
            <div class="dc-card-topo"><h3>Envios hoje por instancia</h3><span class="helper-text">Mensagens enviadas e contatos novos</span></div>
            <div id="dash-uso" class="dc-lista"></div>
          </article>
        </section>
        <article class="dc-card dc-sistema so-master">
          <div class="dc-card-topo"><h3>Sistema</h3><button class="ghost small" onclick="trocarAba('atualizacao')">Ver atualizacao</button></div>
          <div class="dc-sistema-grade">
            <div><span>Aplicacao</span><strong id="app-versao">-</strong></div>
            <div><span>Whatsmeow</span><strong id="wm-versao">-</strong></div>
          </div>
        </article>
      </section>

      <!-- ===== INSTANCIAS ===== -->
      <section class="tab-panel" data-panel="instancias">
        <div class="dc-pagina-topo">
          <div><h1>Instancias</h1><p>Gerencie as conexoes de WhatsApp.</p></div>
          <button class="primary so-master" onclick="abrirModal('nova-instancia')"><svg class="dc-ico"><use href="#i-mais"/></svg>Nova instancia</button>
        </div>
        <div class="dc-filtros">
          <label class="dc-busca"><svg class="dc-ico"><use href="#i-busca"/></svg><input type="search" id="busca-instancias" placeholder="Buscar por nome ou ID" oninput="renderizarListaInstancias()"></label>
          <div class="dc-chips" role="group" aria-label="Filtrar por status">
            <button class="dc-chip ativo" data-filtro="todas" onclick="filtrarInstancias('todas')">Todas</button>
            <button class="dc-chip" data-filtro="conectadas" onclick="filtrarInstancias('conectadas')">Conectadas</button>
            <button class="dc-chip" data-filtro="conectando" onclick="filtrarInstancias('conectando')">Conectando</button>
            <button class="dc-chip" data-filtro="desconectadas" onclick="filtrarInstancias('desconectadas')">Desconectadas</button>
          </div>
        </div>
        <div id="lista-instancias" class="dc-instancias"></div>
      </section>

      <!-- ===== INSTANCIA (detalhe) ===== -->
      <section class="tab-panel" data-panel="instancia">
        <!-- Painel principal -->
        <article class="dc-inst">
          <header class="dc-card dc-inst-topo">
            <div class="dc-avatar" id="inst-avatar"><svg class="dc-ico"><use href="#i-pessoa"/></svg></div>
            <div class="dc-inst-info">
              <div class="dc-inst-linha">
                <h1 id="instancia-titulo">-</h1>
                <span id="status-badge" class="status-badge neutro">-</span>
                <span id="inst-business" class="dc-selo hidden">Business</span>
                <span id="inst-oficial" class="dc-selo dc-selo-oficial hidden">API oficial</span>
              </div>
              <p id="inst-perfil" class="dc-inst-perfil">Conecte para ver o perfil</p>
              <div class="dc-token">
                <svg class="dc-ico"><use href="#i-chave"/></svg>
                <code id="detalhe-token">-</code>
                <button type="button" class="dc-token-botao" id="botao-revelar-token" onclick="alternarTokenVisivel()" title="Mostrar ou esconder"><svg class="dc-ico"><use href="#i-olho"/></svg></button>
                <button type="button" class="dc-token-botao" id="botao-copiar-token" onclick="copiarTokenSelecionado()" title="Copiar token"><svg class="dc-ico"><use href="#i-copiar"/></svg></button>
              </div>
            </div>
            <div class="dc-inst-acoes">
              <button class="ghost small dc-botao-icone" onclick="atualizarInstanciaSelecionada(true)" title="Atualizar"><svg class="dc-ico"><use href="#i-atualizar"/></svg></button>
              <button class="dc-botao-perigo hidden" id="botao-desconectar" onclick="desconectarSelecionada()">Desconectar</button>
              <button class="primary" id="botao-conectar" onclick="abrirConexao()">Conectar</button>
            </div>
          </header>

          <!-- Visao geral -->
          <section class="dc-aba-painel" data-painel-inst="geral">
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Uso e risco</h3><span class="helper-text">Envios, contatos novos e rajadas. Ajuda a evitar bloqueio do numero.</span></div>
              <div id="uso-conteudo" class="uso-conteudo"><div class="dc-lista-vazia">Carregando...</div></div>
            </article>
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Conexao</h3><span id="status-copy" class="helper-text">-</span></div>
              <dl class="dc-dados">
                <div><dt>ID</dt><dd id="detalhe-id">-</dd></div>
                <div><dt>Status</dt><dd id="detalhe-status">-</dd></div>
                <div><dt>Atualizado</dt><dd id="detalhe-atualizado">-</dd></div>
                <div><dt>Ultimo erro</dt><dd id="detalhe-erro">-</dd></div>
              </dl>
            </article>
          </section>

          <!-- Webhooks -->
          <!-- Chat -->
          <section class="dc-aba-painel hidden" data-painel-inst="chat">
            <article class="dc-card dc-chat" id="chat-card">
              <aside class="dc-chat-lista">
                <form class="dc-chat-novo" onsubmit="novaConversaChat(event)">
                  <input type="tel" id="chat-novo-numero" placeholder="Novo: numero com DDI" title="Numero com DDI e DDD, ex.: 5567999999999" autocomplete="off">
                  <button class="primary small dc-botao-icone" type="submit" title="Abrir conversa"><svg class="dc-ico"><use href="#i-mais"/></svg></button>
                </form>
                <div id="chat-conversas" class="dc-chat-conversas"></div>
              </aside>
              <section class="dc-chat-janela">
                <header class="dc-chat-topo">
                  <button class="ghost small dc-botao-icone dc-chat-voltar" onclick="fecharConversaChat()" title="Conversas"><svg class="dc-ico"><use href="#i-voltar"/></svg></button>
                  <span class="dc-avatar dc-avatar-mini" id="chat-topo-avatar"><svg class="dc-ico"><use href="#i-pessoa"/></svg></span>
                  <span class="dc-linha-texto"><strong id="chat-topo-nome">Selecione uma conversa</strong><span id="chat-topo-sub">Mensagens recebidas e enviadas por esta instancia aparecem aqui.</span></span>
                </header>
                <div class="dc-chat-mensagens" id="chat-mensagens" aria-live="polite"></div>
                <form class="dc-chat-envio" onsubmit="enviarMensagemChat(event)">
                  <label class="dc-chat-anexo" title="Anexar arquivo"><input type="file" id="chat-arquivo" onchange="anexoChatSelecionado()"><svg class="dc-ico"><use href="#i-clipe"/></svg></label>
                  <div class="dc-chat-campo">
                    <span id="chat-anexo-nome" class="dc-chat-anexo-nome hidden"></span>
                    <textarea id="chat-texto" rows="1" placeholder="Digite uma mensagem" onkeydown="teclaChat(event)" oninput="ajustarAlturaChat()" disabled></textarea>
                  </div>
                  <button class="primary dc-botao-icone dc-chat-enviar" type="submit" id="chat-enviar" title="Enviar" disabled><svg class="dc-ico"><use href="#i-enviar"/></svg></button>
                </form>
              </section>
            </article>
          </section>

          <section class="dc-aba-painel hidden" data-painel-inst="webhooks">
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Webhooks</h3><button class="primary small" id="botao-novo-webhook" onclick="abrirModal('webhook-novo')"><svg class="dc-ico"><use href="#i-mais"/></svg>Adicionar</button></div>
              <div id="lista-webhooks" class="webhook-list"></div>
            </article>
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Auditoria de entregas</h3><button class="ghost small" id="botao-atualizar-auditoria" onclick="carregarAuditoria(true)">Atualizar</button></div>
              <div id="audit-body" class="audit-body">
                <div id="resumo-entregas-webhook" class="delivery-summary hidden"></div>
                <div id="lista-entregas-webhook" class="webhook-delivery-list empty-state">Carregando...</div>
              </div>
            </article>
          </section>

          <!-- Configuracoes -->
          <section class="dc-aba-painel hidden" data-painel-inst="config">
            <article class="dc-card so-meta" id="card-meta">
              <div class="dc-card-topo"><h3>API oficial (Meta)</h3><button class="ghost small" onclick="abrirModal('meta')">Atualizar credenciais</button></div>
              <dl class="dc-dados dc-dados-meta">
                <div><dt>Numero</dt><dd id="meta-numero">-</dd></div>
                <div><dt>Nome verificado</dt><dd id="meta-nome">-</dd></div>
                <div><dt>Qualidade</dt><dd id="meta-qualidade">-</dd></div>
                <div><dt>Phone number ID</dt><dd id="meta-phone-id">-</dd></div>
              </dl>
              <p class="dc-rotulo">Webhook (cole em WhatsApp &gt; Configuracao &gt; Webhook no painel da Meta e assine o campo <code>messages</code>)</p>
              <div class="dc-lista">
                <div class="dc-config-linha"><div><strong>URL de callback</strong><span class="dc-mono" id="meta-webhook-url">-</span></div><button class="ghost small" onclick="copiarTexto('meta-webhook-url')">Copiar</button></div>
                <div class="dc-config-linha"><div><strong>Token de verificacao</strong><span class="dc-mono" id="meta-verify-token">-</span></div><button class="ghost small" onclick="copiarTexto('meta-verify-token')">Copiar</button></div>
              </div>
              <p class="helper-text" id="meta-aviso-secret"></p>
            </article>
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Comportamento</h3><span class="helper-text">Presenca, leitura, chamadas e filtros de webhook.</span></div>
              <form id="form-avancado" class="dc-form-config">
                <div class="dc-opcoes">
                  <label class="dc-opcao so-qr"><input type="checkbox" id="adv-manter-online"><span class="dc-opcao-texto"><strong>Manter online</strong><small>Mostra a instancia como disponivel no WhatsApp.</small></span><span class="dc-chave" aria-hidden="true"></span></label>
                  <label class="dc-opcao"><input type="checkbox" id="adv-marcar-lida"><span class="dc-opcao-texto"><strong>Marcar como lida</strong><small>Marca as mensagens recebidas como lidas.</small></span><span class="dc-chave" aria-hidden="true"></span></label>
                  <label class="dc-opcao so-qr"><input type="checkbox" id="adv-ignorar-grupos"><span class="dc-opcao-texto"><strong>Ignorar grupos</strong><small>Mensagens de grupos nao vao para os webhooks.</small></span><span class="dc-chave" aria-hidden="true"></span></label>
                  <label class="dc-opcao so-qr"><input type="checkbox" id="adv-ignorar-status"><span class="dc-opcao-texto"><strong>Ignorar status</strong><small>Status (stories) nao vao para os webhooks.</small></span><span class="dc-chave" aria-hidden="true"></span></label>
                  <label class="dc-opcao so-qr"><input type="checkbox" id="adv-rejeitar-chamadas"><span class="dc-opcao-texto"><strong>Rejeitar chamadas</strong><small>Recusa automaticamente as chamadas recebidas.</small></span><span class="dc-chave" aria-hidden="true"></span></label>
                </div>
                <label class="dc-campo hidden" id="adv-mensagem-wrap"><span>Mensagem apos rejeitar chamada</span><textarea id="adv-mensagem-rejeitar" placeholder="No momento nao consigo atender chamadas. Envie uma mensagem por aqui."></textarea></label>
                <div class="dc-form-rodape">
                  <span id="advanced-copy" class="helper-text">-</span>
                  <button class="primary small" id="botao-salvar-avancado" type="submit">Salvar</button>
                </div>
              </form>
            </article>
            <article class="dc-card">
              <div class="dc-card-topo"><h3>Conexao e acesso</h3></div>
              <div class="dc-lista">
                <div class="dc-config-linha so-qr"><div><strong>Proxy</strong><span id="config-proxy">-</span></div><button class="ghost small" id="botao-config-proxy" onclick="abrirModal('proxy')">Alterar</button></div>
                <div class="dc-config-linha"><div><strong>Token de acesso</strong><span>Usado nas integracoes para enviar por esta instancia.</span></div><button class="ghost small" id="botao-config-token" onclick="abrirModal('token')">Trocar</button></div>
                <div class="dc-config-linha so-qr"><div><strong>Historico ao conectar</strong><span id="config-historico">-</span></div><button class="ghost small" id="botao-config-historico" onclick="abrirModal('historico')">Alterar</button></div>
              </div>
            </article>
            <article class="dc-card dc-perigo so-master">
              <div class="dc-config-linha"><div><strong>Excluir instancia</strong><span>Remove a instancia e a sessao do WhatsApp. Nao da para desfazer.</span></div><button class="dc-botao-perigo" id="botao-excluir" onclick="excluirSelecionada()">Excluir</button></div>
            </article>
          </section>

          <!-- Ferramentas -->
          <section class="dc-aba-painel hidden" data-painel-inst="ferramentas">
            <div class="dc-ferramentas">
              <button class="dc-ferramenta" id="botao-enviar-texto" onclick="abrirModal('texto')"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-mensagem"/></svg></span><strong>Enviar mensagem</strong><span>Teste um envio de texto por esta instancia.</span></button>
              <button class="dc-ferramenta so-qr" id="botao-teste-chamada" onclick="abrirModal('chamada')"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-telefone"/></svg></span><strong>Teste de chamada</strong><span>Ligacao de voz pelo navegador.</span></button>
              <button class="dc-ferramenta so-qr" id="botao-pairing" onclick="abrirModal('pairing')"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-celular"/></svg></span><strong>Codigo de pareamento</strong><span>Conectar sem QR code, digitando um codigo no celular.</span></button>
            </div>
          </section>
        </article>
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

      <details class="dc-card dc-log">
        <summary>Retorno da ultima chamada da API</summary>
        <pre id="retorno-api">Sem chamadas ainda.</pre>
      </details>
    </main>
  </div>

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
  const estadoUI = { abaAtual:'dashboard', instanciaSelecionada:'', pollingConexao:null, refreshGeral:null, acesso:null, webhooks:[], entregasWebhook:[], instancias:[], usoHoje:{}, filtroInstancias:'todas' };

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
    conectar: {
      kicker: 'Escaneie o QR code ou use o codigo de pareamento', titulo: 'Conectar WhatsApp', largo: true,
      html: '<div class="dc-conexao"><div class="dc-conexao-qr"><h4>Escaneie o QR code</h4><div id="qrcode-box" class="dc-qr"><div class="qrcode-loading">Gerando QR code...</div></div><p class="helper-text">O codigo e renovado sozinho a cada poucos segundos.</p></div><div class="dc-conexao-passos"><h4>Como conectar</h4><ol><li><strong>Abra o WhatsApp no celular</strong><span>Use a versao mais recente do app.</span></li><li><strong>Acesse Dispositivos conectados</strong><span>Configuracoes (ou os tres pontinhos) e depois Dispositivos conectados.</span></li><li><strong>Toque em Conectar um dispositivo</strong><span>E aponte a camera para o QR code ao lado.</span></li><li><strong>Aguarde a confirmacao</strong><span>Esta janela avisa quando conectar.</span></li></ol><button type="button" class="ghost small" onclick="fecharModal();abrirModal(\'pairing\')">Prefere digitar um codigo? Usar codigo de pareamento</button></div></div><p class="dc-aguardando"><span class="dc-ponto processing"></span>Aguardando conexao...</p>'
    },
    'nova-instancia': {
      kicker: 'Configure tudo antes de criar', titulo: 'Nova instancia', semInstancia: true, largo: true,
      html: '<form id="modal-form" class="dc-wizard"><nav class="dc-wizard-nav" aria-label="Secoes"><p>Basico</p><button type="button" data-passo="geral" onclick="passoNovaInstancia(\'geral\')">Geral</button><button type="button" data-passo="comportamento" onclick="passoNovaInstancia(\'comportamento\')">Comportamento</button><p>Integracoes</p><button type="button" data-passo="webhook" onclick="passoNovaInstancia(\'webhook\')">Webhook<span class="dc-tag-ativo hidden" id="ni-tag-webhook">Ativo</span></button><button type="button" class="so-qr" data-passo="proxy" onclick="passoNovaInstancia(\'proxy\')">Proxy<span class="dc-tag-ativo hidden" id="ni-tag-proxy">Ativo</span></button></nav><div class="dc-wizard-corpo"><section data-passo="geral"><h4>Geral</h4><p class="helper-text">Nome e identificacao da instancia.</p><label><span>Nome <em>*</em></span><input type="text" id="ni-nome" placeholder="Ex.: Atendimento" autocomplete="off"></label><label><span>Token personalizado (opcional)</span><span class="dc-campo-acao"><input type="text" id="ni-token" placeholder="Gerado automaticamente se ficar vazio" autocomplete="off"><button type="button" class="ghost small" onclick="gerarTokenNovaInstancia()">Gerar</button></span></label><p class="dc-rotulo">Tipo de conexao</p><div class="dc-segmentos" role="radiogroup"><label><input type="radio" name="ni-tipo" value="whatsapp" checked><span>QR code</span></label><label><input type="radio" name="ni-tipo" value="meta"><span>API oficial (Meta)</span></label></div><div id="ni-meta-campos" class="hidden"><p class="helper-text">Dados do app no Meta for Developers (WhatsApp &gt; Configuracao da API). Use um token permanente de Usuario do Sistema: o temporario expira em 24h.</p><div class="form-grid"><label><span>Phone number ID <em>*</em></span><input type="text" id="ni-meta-phone" autocomplete="off"></label><label><span>WABA ID</span><input type="text" id="ni-meta-waba" autocomplete="off"></label></div><label><span>Access token <em>*</em></span><input type="password" id="ni-meta-token" autocomplete="new-password"></label><label><span>App secret (recomendado)</span><input type="password" id="ni-meta-secret" autocomplete="new-password" placeholder="Confere a assinatura dos webhooks da Meta"></label></div></section><section data-passo="comportamento" class="hidden"><h4>Comportamento</h4><p class="helper-text">Como a instancia lida com presenca, leitura, chamadas e historico.</p><div class="dc-opcoes"><label class="dc-opcao so-qr"><input type="checkbox" id="ni-manter-online"><span class="dc-opcao-texto"><strong>Manter online</strong><small>Mostra a instancia como disponivel no WhatsApp.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" id="ni-marcar-lida"><span class="dc-opcao-texto"><strong>Marcar como lida</strong><small>Marca as mensagens recebidas como lidas.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao so-qr"><input type="checkbox" id="ni-ignorar-grupos"><span class="dc-opcao-texto"><strong>Ignorar grupos</strong><small>Mensagens de grupos nao vao para os webhooks.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao so-qr"><input type="checkbox" id="ni-ignorar-status" checked><span class="dc-opcao-texto"><strong>Ignorar status</strong><small>Status (stories) nao vao para os webhooks.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao so-qr"><input type="checkbox" id="ni-rejeitar-chamadas"><span class="dc-opcao-texto"><strong>Rejeitar chamadas</strong><small>Recusa automaticamente as chamadas recebidas.</small></span><span class="dc-chave" aria-hidden="true"></span></label></div><label class="hidden" id="ni-mensagem-wrap"><span>Mensagem apos rejeitar chamada</span><textarea id="ni-mensagem-rejeitar" placeholder="No momento nao consigo atender chamadas. Envie uma mensagem por aqui."></textarea></label><label class="so-qr"><span>Dias de historico ao conectar (0 = sem historico)</span><input type="number" id="ni-historico" min="0" max="90" value="0"></label></section><section data-passo="webhook" class="hidden"><h4>Webhook</h4><p class="helper-text">Receba os eventos desta instancia no seu sistema.</p><label class="dc-opcao"><input type="checkbox" id="ni-webhook-ativo"><span class="dc-opcao-texto"><strong>Ativar webhook</strong><small>Envia os eventos escolhidos para a URL abaixo.</small></span><span class="dc-chave" aria-hidden="true"></span></label><div id="ni-webhook-campos" class="dc-desativado"><label><span>URL <em>*</em></span><input type="url" id="ni-webhook-url" placeholder="https://seu-sistema.com/webhook" autocomplete="off"></label><p class="dc-rotulo">Eventos</p><div class="dc-opcoes"><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="mensagens" checked><span class="dc-opcao-texto"><strong>Mensagens</strong><small>Mensagens recebidas e enviadas.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="recibos"><span class="dc-opcao-texto"><strong>Recibos</strong><small>Entregue e lido.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="chamadas"><span class="dc-opcao-texto"><strong>Chamadas</strong><small>Recebida, aceita, recusada, encerrada.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="status"><span class="dc-opcao-texto"><strong>Status</strong><small>Conexao, QR e desconexao da instancia.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="digitando"><span class="dc-opcao-texto"><strong>Digitando</strong><small>Quando o contato esta digitando.</small></span><span class="dc-chave" aria-hidden="true"></span></label><label class="dc-opcao"><input type="checkbox" class="ni-evento" value="gravando_audio"><span class="dc-opcao-texto"><strong>Gravando audio</strong><small>Quando o contato esta gravando audio.</small></span><span class="dc-chave" aria-hidden="true"></span></label></div></div></section><section data-passo="proxy" class="hidden"><h4>Proxy</h4><p class="helper-text">Conecte esta instancia ao WhatsApp passando por um proxy.</p><label class="dc-opcao"><input type="checkbox" id="ni-proxy-ativo"><span class="dc-opcao-texto"><strong>Ativar proxy</strong><small>Desligado, usa o proxy global do sistema, se houver.</small></span><span class="dc-chave" aria-hidden="true"></span></label><div id="ni-proxy-campos" class="dc-desativado"><p class="dc-rotulo">Protocolo</p><div class="dc-segmentos" role="radiogroup"><label><input type="radio" name="ni-proxy-protocolo" value="http" checked><span>HTTP</span></label><label><input type="radio" name="ni-proxy-protocolo" value="socks5"><span>SOCKS5</span></label></div><div class="form-grid"><label><span>Host <em>*</em></span><input type="text" id="ni-proxy-host" placeholder="Ex.: 192.168.0.1" autocomplete="off"></label><label><span>Porta <em>*</em></span><input type="number" id="ni-proxy-porta" placeholder="Ex.: 1080"></label><label><span>Usuario</span><input type="text" id="ni-proxy-usuario" autocomplete="off"></label><label><span>Senha</span><input type="password" id="ni-proxy-senha" autocomplete="new-password"></label></div></div></section><div id="mf-feedback" class="modal-feedback hidden"></div></div><div class="dc-wizard-rodape"><span class="helper-text"><em>*</em> Obrigatorio</span><div><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Criar instancia</button></div></div></form>',
      init: function() {
        ['ni-webhook-ativo','ni-proxy-ativo','ni-rejeitar-chamadas'].forEach(id => document.getElementById(id).addEventListener('change', atualizarNovaInstancia));
        document.querySelectorAll('input[name="ni-tipo"]').forEach(el => el.addEventListener('change', atualizarNovaInstancia));
        passoNovaInstancia('geral');
        document.getElementById('modal-form').addEventListener('input', () => { document.getElementById('mf-feedback').className = 'modal-feedback hidden'; });
        atualizarNovaInstancia();
        document.getElementById('ni-nome').focus();
      },
      submit: criarNovaInstancia
    },
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
    meta: {
      kicker: 'API oficial', titulo: 'Credenciais da Meta',
      html: '<form id="modal-form" class="stack-form"><p class="helper-text">Deixe em branco o que nao quiser trocar. Tudo e validado na Meta antes de salvar.</p><label><span>Phone number ID</span><input type="text" id="mf-meta-phone" autocomplete="off"></label><label><span>WABA ID</span><input type="text" id="mf-meta-waba" autocomplete="off"></label><label><span>Access token</span><input type="password" id="mf-meta-token" autocomplete="new-password" placeholder="Token permanente de Usuario do Sistema"></label><label><span>App secret</span><input type="password" id="mf-meta-secret" autocomplete="new-password"></label><label><span>Token de verificacao do webhook</span><input type="text" id="mf-meta-verify" autocomplete="off"></label><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Validar e salvar</button></div></form>',
      init: function() {
        const m = estadoUI.metaSelecionada || {};
        document.getElementById('mf-meta-phone').placeholder = m.phone_number_id || '';
        document.getElementById('mf-meta-waba').placeholder = m.waba_id || '';
        document.getElementById('mf-meta-verify').placeholder = m.webhook_verify_token || '';
        document.getElementById('mf-meta-token').focus();
      },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        const v = id => document.getElementById(id).value.trim();
        btn.disabled = true; btn.textContent = 'Validando na Meta...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/meta', { method:'PUT', body: JSON.stringify({
            phone_number_id: v('mf-meta-phone'), waba_id: v('mf-meta-waba'), access_token: v('mf-meta-token'),
            app_secret: v('mf-meta-secret'), verify_token: v('mf-meta-verify') }) });
          mostrarToast('Credenciais validadas e salvas.', 'success');
          fecharModal();
          await atualizarInstanciaSelecionada(true);
        } catch(err) { fb.className = 'modal-feedback error'; fb.textContent = 'Erro: ' + err.message; }
        finally { btn.disabled = false; btn.textContent = 'Validar e salvar'; }
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
      html: '<form id="modal-form" class="stack-form"><label><span>Nome</span><input type="text" id="mf-wh-nome" placeholder="Ex.: Notificacoes" required autocomplete="off"></label><label><span>URL destino</span><input type="url" id="mf-wh-url" placeholder="https://meusite.com/webhook" required autocomplete="off"></label><label><span>Eventos (selecione um ou mais)</span><div class="checkbox-group" id="mf-wh-eventos"><label class="check-item"><input type="checkbox" value="mensagens"> Mensagens</label><label class="check-item"><input type="checkbox" value="recibos"> Recibos</label><label class="check-item"><input type="checkbox" value="chamadas"> Chamadas</label><label class="check-item"><input type="checkbox" value="status"> Status</label><label class="check-item"><input type="checkbox" value="digitando"> Digitando</label><label class="check-item"><input type="checkbox" value="gravando_audio"> Gravando audio</label></div></label><label class="check-item"><input type="checkbox" id="mf-wh-base64" checked> Enviar o arquivo em base64 nas midias recebidas</label><p class="status-copy" style="margin:0">Desmarcado, o webhook recebe so o link de download (<code>midia.download_url</code>). Video em base64 deixa o payload pesado e ocupa o banco na fila de entregas.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Criar webhook</button></div></form>',
      init: function() { document.getElementById('mf-wh-nome').focus(); },
      submit: async function(e) {
        e.preventDefault();
        const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
        const eventos = Array.from(document.querySelectorAll('#mf-wh-eventos input:checked')).map(i => i.value);
        if (!eventos.length) { fb.className = 'modal-feedback error'; fb.textContent = 'Selecione ao menos um evento.'; return; }
        btn.disabled = true; btn.textContent = 'Criando...'; fb.className = 'modal-feedback hidden';
        try {
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhooks', { method:'POST', body: JSON.stringify({ nome: document.getElementById('mf-wh-nome').value, url: document.getElementById('mf-wh-url').value, eventos: eventos, incluir_base64: document.getElementById('mf-wh-base64').checked }) });
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
      html: '<form id="modal-form" class="stack-form"><label><span>Nome</span><input type="text" id="mf-wh-nome" required autocomplete="off"></label><label><span>URL destino</span><input type="url" id="mf-wh-url" required autocomplete="off"></label><label><span>Ativo</span><select id="mf-wh-ativo" style="width:100%;border:1px solid var(--line);background:rgba(255,255,255,.76);border-radius:16px;padding:14px 15px;outline:none;color:var(--ink)"><option value="true">Ativo</option><option value="false">Inativo</option></select></label><label><span>Eventos</span><div class="checkbox-group" id="mf-wh-eventos"><label class="check-item"><input type="checkbox" value="mensagens"> Mensagens</label><label class="check-item"><input type="checkbox" value="recibos"> Recibos</label><label class="check-item"><input type="checkbox" value="chamadas"> Chamadas</label><label class="check-item"><input type="checkbox" value="status"> Status</label><label class="check-item"><input type="checkbox" value="digitando"> Digitando</label><label class="check-item"><input type="checkbox" value="gravando_audio"> Gravando audio</label></div></label><label class="check-item"><input type="checkbox" id="mf-wh-base64" checked> Enviar o arquivo em base64 nas midias recebidas</label><p class="status-copy" style="margin:0">Desmarcado, o webhook recebe so o link de download (<code>midia.download_url</code>). Video em base64 deixa o payload pesado e ocupa o banco na fila de entregas.</p><div id="mf-feedback" class="modal-feedback hidden"></div><div class="modal-footer"><button type="button" class="ghost" onclick="fecharModal()">Cancelar</button><button type="submit" class="primary" id="mf-submit">Salvar webhook</button></div></form>',
      init: function() {
        const wh = estadoUI.webhookEditando;
        if (!wh) return;
        document.getElementById('mf-wh-nome').value = wh.nome || '';
        document.getElementById('mf-wh-url').value = wh.url || '';
        document.getElementById('mf-wh-ativo').value = wh.ativo ? 'true' : 'false';
        document.getElementById('mf-wh-base64').checked = wh.incluir_base64 !== false;
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
          await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/webhooks/' + wh.id, { method:'PUT', body: JSON.stringify({ nome: document.getElementById('mf-wh-nome').value, url: document.getElementById('mf-wh-url').value, eventos: eventos, ativo: document.getElementById('mf-wh-ativo').value === 'true', incluir_base64: document.getElementById('mf-wh-base64').checked }) });
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
    const m = MODAIS[tipo]; if (!m) return;
    if (!m.semInstancia && !estadoUI.instanciaSelecionada) { mostrarToast('Selecione uma instancia para usar as acoes.', 'error'); return; }
    document.querySelector('#modal-overlay .modal-dialog').classList.toggle('dc-modal-largo', Boolean(m.largo));
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
      if (estadoUI.acesso?.tipo === 'instancia' && estadoUI.instanciaSelecionada) trocarAba('instancia');
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
  const TITULOS_ABA = { dashboard:'Dashboard', instancias:'Instancias', instancia:'Instancias', atualizacao:'Atualizacao' };
  function trocarAba(aba) {
    estadoUI.abaAtual = aba;
    // Dentro de uma instancia o menu lateral vira o menu dela.
    const menuInstancia = aba === 'instancia';
    document.getElementById('nav-principal').classList.toggle('hidden', menuInstancia);
    document.getElementById('nav-instancia').classList.toggle('hidden', !menuInstancia);
    document.querySelectorAll('#nav-principal .nav-link').forEach(i => i.classList.toggle('active', i.dataset.tab === aba));
    document.querySelectorAll('.tab-panel').forEach(i => i.classList.toggle('active', i.dataset.panel === aba));
    let trilha = TITULOS_ABA[aba] || aba;
    if (aba === 'instancia') trilha += ' / ' + (document.getElementById('instancia-titulo').textContent || '');
    document.getElementById('trilha-pagina').textContent = trilha;
    window.scrollTo(0, 0);
  }
  function formatarData(v) { const d = v ? new Date(v) : null; return d && d.getFullYear() > 2000 ? d.toLocaleString('pt-BR') : '-'; }
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
    document.getElementById('botao-excluir').classList.toggle('hidden', instancia);
    document.querySelectorAll('.so-master').forEach(el => el.classList.toggle('hidden', instancia));
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
    let lista = resp.dados.instancias || [];
    if (!lista.length && estadoUI.acesso?.tipo === 'instancia' && estadoUI.acesso.instancia_id) {
      lista = [{ id: estadoUI.acesso.instancia_id, nome: estadoUI.acesso.nome || 'Instancia vinculada', status: '' }];
    }
    estadoUI.instancias = lista;
    document.getElementById('contador-instancias').textContent = String(lista.length);
    document.getElementById('contador-online').textContent = String(lista.filter(i => i.status === 'conectada').length);
    document.getElementById('nav-contador-instancias').textContent = String(lista.length);
    if (!estadoUI.instanciaSelecionada && estadoUI.acesso?.tipo === 'instancia' && lista.length === 1) {
      estadoUI.instanciaSelecionada = lista[0].id;
    }
    if (estadoUI.instanciaSelecionada && !lista.some(i => i.id === estadoUI.instanciaSelecionada)) {
      atualizarPainelSemSelecao();
      if (estadoUI.abaAtual === 'instancia') trocarAba('instancias');
    }
    renderizarListaInstancias();
    renderizarDashboard();
    if (estadoUI.instanciaSelecionada) await atualizarInstanciaSelecionada(false);
  }

  function grupoStatus(s) {
    const c = classeStatus(s);
    return c === 'online' ? 'conectadas' : c === 'processing' ? 'conectando' : 'desconectadas';
  }

  function filtrarInstancias(filtro) {
    estadoUI.filtroInstancias = filtro;
    document.querySelectorAll('.dc-chip').forEach(b => b.classList.toggle('ativo', b.dataset.filtro === filtro));
    renderizarListaInstancias();
  }

  function renderizarListaInstancias() {
    const box = document.getElementById('lista-instancias');
    const todas = estadoUI.instancias || [];
    if (!todas.length) {
      box.className = 'dc-instancias dc-vazio';
      box.innerHTML = '<svg class="dc-ico dc-vazio-icone"><use href="#i-celular"/></svg><h3>Nenhuma instancia ainda</h3><p>Crie a primeira instancia para conectar um WhatsApp e comecar a enviar mensagens.</p>' +
        (estadoUI.acesso?.tipo === 'instancia' ? '' : '<button class="primary" onclick="abrirModal(\'nova-instancia\')"><svg class="dc-ico"><use href="#i-mais"/></svg>Criar primeira instancia</button>');
      return;
    }
    const busca = (document.getElementById('busca-instancias').value || '').trim().toLowerCase();
    const filtro = estadoUI.filtroInstancias;
    const lista = todas.filter(i =>
      (filtro === 'todas' || grupoStatus(i.status) === filtro) &&
      (!busca || String(i.nome || '').toLowerCase().includes(busca) || String(i.id).toLowerCase().includes(busca)));
    if (!lista.length) {
      box.className = 'dc-instancias dc-vazio';
      box.innerHTML = '<svg class="dc-ico dc-vazio-icone"><use href="#i-busca"/></svg><h3>Nada encontrado</h3><p>Nenhuma instancia com esse filtro.</p>';
      return;
    }
    box.className = 'dc-instancias';
    box.innerHTML = lista.map(i => {
      const uso = estadoUI.usoHoje[i.id] || {};
      const perfil = i.perfil || {};
      const linha = perfil.numero ? textoSeguro([perfil.nome_empresa || perfil.nome, formatarTelefone(perfil.numero)].filter(Boolean).join(' · ')) : 'ID ' + textoSeguro(String(i.id).slice(0, 8));
      return '<button class="dc-instancia" onclick="selecionarInstancia(\''+textoSeguro(i.id)+'\')">' +
        '<span class="dc-instancia-topo"><span class="dc-avatar dc-avatar-p'+(i.status === 'conectada' ? ' online' : '')+'">'+avatarHTML(perfil, i.nome)+'</span>' +
          '<span class="mini-status '+classeStatus(i.status)+'">'+textoSeguro(rotuloStatus(i.status))+'</span></span>' +
        '<strong>'+textoSeguro(i.nome)+(i.tipo === 'meta' ? ' <span class="dc-selo dc-selo-oficial">API oficial</span>' : perfil.business ? ' <span class="dc-selo">Business</span>' : '')+'</strong>' +
        '<span class="dc-instancia-id">'+linha+'</span>' +
        '<span class="dc-instancia-rodape"><span>'+numeroUso(uso.envios)+' envios hoje</span><svg class="dc-ico"><use href="#i-seta"/></svg></span>' +
      '</button>';
    }).join('') + (estadoUI.acesso?.tipo === 'instancia' || filtro !== 'todas' || busca ? '' :
      '<button class="dc-instancia dc-instancia-nova" onclick="abrirModal(\'nova-instancia\')"><span class="dc-kpi-icone"><svg class="dc-ico"><use href="#i-mais"/></svg></span><strong>Nova instancia</strong></button>');
  }

  async function carregarUsoGeral() {
    try {
      const resp = await chamar('/api/v1/uso');
      const mapa = {};
      (resp?.dados?.instancias || []).forEach(u => { mapa[u.instancia_id] = u; });
      estadoUI.usoHoje = mapa;
    } catch(err) { estadoUI.usoHoje = {}; }
    renderizarDashboard();
    renderizarListaInstancias();
  }

  function renderizarDashboard() {
    const hora = new Date().getHours();
    document.getElementById('dash-saudacao').textContent = hora < 12 ? 'Bom dia' : hora < 18 ? 'Boa tarde' : 'Boa noite';
    const instancias = estadoUI.instancias || [];
    const usos = Object.values(estadoUI.usoHoje || {});
    document.getElementById('kpi-envios').textContent = numeroUso(usos.reduce((t, u) => t + (u.envios || 0), 0));
    document.getElementById('kpi-contatos').textContent = numeroUso(usos.reduce((t, u) => t + (u.contatos_novos || 0), 0));
    const nome = id => (instancias.find(i => i.id === id) || {}).nome || String(id).slice(0, 8);
    const linha = (id, titulo, detalhe, classe) =>
      '<button class="dc-linha" onclick="selecionarInstancia(\''+textoSeguro(id)+'\')"><span class="dc-ponto '+classe+'"></span><span class="dc-linha-texto"><strong>'+textoSeguro(titulo)+'</strong><span>'+detalhe+'</span></span><svg class="dc-ico"><use href="#i-seta"/></svg></button>';

    const atencao = [];
    instancias.filter(i => i.status && i.status !== 'conectada').forEach(i =>
      atencao.push(linha(i.id, i.nome, textoSeguro(rotuloStatus(i.status)), classeStatus(i.status))));
    usos.filter(u => u.rajadas || u.limitados).forEach(u => {
      const partes = [];
      if (u.rajadas) partes.push(u.rajadas + (u.rajadas === 1 ? ' contato recebeu rajada' : ' contatos receberam rajada'));
      if (u.limitados) partes.push(u.limitados + ' envios barrados pelo limite');
      atencao.push(linha(u.instancia_id, nome(u.instancia_id), textoSeguro(partes.join(' · ')), 'processing'));
    });
    document.getElementById('dash-atencao').innerHTML = atencao.length ? atencao.join('') :
      '<div class="dc-lista-vazia"><svg class="dc-ico"><use href="#i-ok"/></svg>Tudo certo: instancias conectadas e sem sinal de risco hoje.</div>';

    const maior = Math.max(1, ...usos.map(u => u.envios || 0));
    const ordenados = usos.slice().sort((a, b) => (b.envios || 0) - (a.envios || 0)).slice(0, 8);
    document.getElementById('dash-uso').innerHTML = ordenados.length ? ordenados.map(u =>
      '<button class="dc-linha dc-uso-linha" onclick="selecionarInstancia(\''+textoSeguro(u.instancia_id)+'\')"><span class="dc-linha-texto"><strong>'+textoSeguro(nome(u.instancia_id))+'</strong>' +
        '<span class="dc-barra"><span style="width:'+Math.round((u.envios || 0) * 100 / maior)+'%"></span></span></span>' +
        '<span class="dc-uso-num"><strong>'+numeroUso(u.envios)+'</strong><span>'+numeroUso(u.contatos_novos)+' novos</span></span></button>').join('') :
      '<div class="dc-lista-vazia">Nenhuma mensagem enviada hoje.</div>';
  }

  const BOTOES_INSTANCIA = ['botao-conectar','botao-desconectar','botao-excluir','botao-pairing','botao-copiar-token','botao-enviar-texto','botao-teste-chamada','botao-config-token','botao-config-proxy','botao-config-historico','botao-novo-webhook','botao-salvar-avancado'];
  const CAMPOS_AVANCADO = ['adv-manter-online','adv-rejeitar-chamadas','adv-mensagem-rejeitar','adv-marcar-lida','adv-ignorar-grupos','adv-ignorar-status'];

  function atualizarPainelSemSelecao() {
    estadoUI.instanciaSelecionada = '';
    document.getElementById('instancia-titulo').textContent = 'Nenhuma instancia selecionada';
    document.getElementById('status-badge').className = 'status-badge neutro'; document.getElementById('status-badge').textContent = 'Sem selecao';
    ['detalhe-id','detalhe-status','detalhe-atualizado','detalhe-erro'].forEach(id => document.getElementById(id).textContent='-');
    atualizarTokenDetalhado('', false);
    document.getElementById('status-copy').textContent = 'Clique em uma instancia da lista para operar.';
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
    document.getElementById('uso-conteudo').innerHTML = '';
    renderizarCabecalhoInstancia({});
    preencherConfiguracaoAvancada({}, true);
    CAMPOS_AVANCADO.forEach(id => document.getElementById(id).disabled = true);
    document.getElementById('advanced-copy').textContent = 'Selecione uma instancia para editar.';
    pararPollingConexao();
  }

  async function selecionarInstancia(id) {
    estadoUI.instanciaSelecionada = id;
    const item = (estadoUI.instancias || []).find(i => i.id === id);
    if (item) renderizarCabecalhoInstancia(item);
    if (item) document.getElementById('instancia-titulo').textContent = item.nome;
    estadoUI.qrAtual = '';
    if (chatUI.instancia !== id) { chatUI.instancia = id; chatUI.chat = ''; chatUI.assinatura = ''; }
    trocarAba('instancia');
    trocarAbaInstancia('geral');
    await carregarInstancias(true);
    await carregarWebhooks();
  }

  async function atualizarInstanciaSelecionada(mostrarErro) {
    if (!estadoUI.instanciaSelecionada) { atualizarPainelSemSelecao(); return; }
    try {
      const resp = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/status');
      if (!resp) return;
      const d = resp.dados, cls = classeStatus(d.status), rot = rotuloStatus(d.status);
      document.getElementById('instancia-titulo').textContent = d.nome;
      document.getElementById('status-badge').className = 'status-badge '+cls; document.getElementById('status-badge').textContent = rot;
      ['detalhe-id','detalhe-status','detalhe-atualizado','detalhe-erro'].forEach((id,i) => document.getElementById(id).textContent=[d.id, rot, formatarData(d.atualizado_em), d.erro||'-'][i]);
      atualizarTokenDetalhado(d.token, false);
      document.getElementById('status-copy').textContent = gerarResumoStatus(d.status, d.erro);
      if (estadoUI.abaAtual === 'instancia') document.getElementById('trilha-pagina').textContent = 'Instancias / ' + d.nome;
      BOTOES_INSTANCIA.forEach(id => document.getElementById(id).disabled = false);
      CAMPOS_AVANCADO.forEach(id => document.getElementById(id).disabled = false);
      preencherConfiguracaoAvancada(d.configuracao_avancada || {});
      renderizarCabecalhoInstancia(d);
      estadoUI.statusSelecionada = d.status;
      if (d.status === 'aguardando_qrcode') { await mostrarQRCodeSelecionado(); iniciarPollingConexao(); }
      else if (['conectando','pareada','autenticando','sincronizando_historico'].includes(d.status)) {
        iniciarPollingConexao();
        definirQR('<div class="qrcode-loading">'+gerarResumoStatus(d.status,d.erro)+'</div>');
      } else {
        pararPollingConexao();
      }
      if (d.status === 'conectada' && modalTipoAtual === 'conectar') mostrarConexaoConcluida(d);
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

  async function conectarSelecionada()    { if (!estadoUI.instanciaSelecionada) return; definirQR('<div class="qrcode-loading">Gerando QR code...</div>'); const r=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/conectar',{method:'POST'}); if(r){await carregarInstancias(true);iniciarPollingConexao();} }
  async function desconectarSelecionada() { if (!estadoUI.instanciaSelecionada) return; const r=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/desconectar',{method:'POST'}); if(r){mostrarToast('Instancia desconectada.','info');pararPollingConexao();await carregarInstancias(true);} }
  async function excluirSelecionada()     { if (!estadoUI.instanciaSelecionada) return; if (!window.confirm('Excluir esta instancia e remover a sessao local?')) return; const id=estadoUI.instanciaSelecionada; const r=await chamar('/api/v1/instancias/'+id,{method:'DELETE'}); if(r){mostrarToast('Instancia excluida.','success');atualizarPainelSemSelecao();trocarAba('instancias');await carregarInstancias(true);} }
  async function mostrarQRCodeSelecionado() {
    if (!estadoUI.instanciaSelecionada) return;
    const resp=await chamar('/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/qrcode');
    if (!resp) return;
    const cod = resp.dados.qrcode || '';
    if (!cod) { definirQR('<div class="qrcode-loading">Gerando QR code...</div>'); estadoUI.qrAtual = ''; return; }
    // So troca a imagem quando o WhatsApp gera um QR novo (evita piscar a cada consulta).
    if (cod === estadoUI.qrAtual && document.querySelector('#qrcode-box img')) return;
    estadoUI.qrAtual = cod;
    definirQR('<img src="/api/v1/instancias/'+estadoUI.instanciaSelecionada+'/qrcode/imagem?ts='+Date.now()+'" alt="QR code" class="qrcode-image">');
  }
  function iniciarPollingConexao() { if(estadoUI.pollingConexao) return; estadoUI.pollingConexao=setInterval(async function(){if(!estadoUI.instanciaSelecionada)return;await atualizarInstanciaSelecionada(false);await carregarInstancias(false);},2500); }
  function pararPollingConexao()   { if(estadoUI.pollingConexao){clearInterval(estadoUI.pollingConexao);estadoUI.pollingConexao=null;} }


  // ── Pagina da instancia ───────────────────────────────────────────────────────
  function trocarAbaInstancia(aba) {
    estadoUI.abaInstancia = aba;
    document.querySelectorAll('#nav-instancia [data-aba-inst]').forEach(b => b.classList.toggle('active', b.dataset.abaInst === aba));
    document.querySelectorAll('.dc-aba-painel').forEach(p => p.classList.toggle('hidden', p.dataset.painelInst !== aba));
    if (!estadoUI.instanciaSelecionada) return;
    if (aba === 'geral') carregarUso(false);
    if (aba === 'webhooks') { carregarWebhooks(); carregarAuditoria(false); }
    if (aba === 'chat') iniciarChat(); else pararChat();
  }
  // ── Chat ──────────────────────────────────────────────────────────────────────
  // Mensagens guardadas pela API (recebidas pelos eventos de webhook e
  // enviadas pela API). Atualiza a cada 3s enquanto a aba esta aberta.
  const chatUI = { instancia:'', chat:'', conversas:[], assinatura:'', timer:null, arquivo:null, enviando:false };
  function chatAberto() { return estadoUI.abaAtual === 'instancia' && estadoUI.abaInstancia === 'chat'; }
  async function buscarChat(url) {
    // Sem passar pelo chamar(): o polling nao deve sobrescrever o "Retorno da ultima chamada".
    const r = await fetch(url, { headers:{'Content-Type':'application/json'} });
    if (r.status === 401) { mostrarLogin(); return null; }
    const d = await r.json().catch(() => null);
    if (!r.ok) throw new Error(d?.mensagem || 'Erro ao carregar o chat');
    return d?.dados || {};
  }
  function iniciarChat() {
    if (chatUI.instancia !== estadoUI.instanciaSelecionada) { chatUI.instancia = estadoUI.instanciaSelecionada; chatUI.chat = ''; chatUI.assinatura = ''; }
    renderizarJanelaChat();
    carregarChat();
    if (!chatUI.timer) chatUI.timer = setInterval(() => { if (chatAberto()) carregarChat(); else pararChat(); }, 3000);
  }
  function pararChat() { if (chatUI.timer) { clearInterval(chatUI.timer); chatUI.timer = null; } }
  async function carregarChat() {
    const id = estadoUI.instanciaSelecionada;
    if (!id) return;
    try {
      const d = await buscarChat('/api/v1/instancias/' + encodeURIComponent(id) + '/chat/conversas');
      if (!d || id !== estadoUI.instanciaSelecionada) return;
      chatUI.conversas = d.conversas || [];
      renderizarConversasChat();
      if (chatUI.chat) await carregarMensagensChat();
    } catch (err) {
      document.getElementById('chat-conversas').innerHTML = '<div class="dc-lista-vazia">' + textoSeguro(err.message) + '</div>';
    }
  }
  function numeroDoJid(jid) { return String(jid || '').split('@')[0]; }
  function nomeConversa(c) {
    if (!c) return '';
    if (c.grupo) return 'Grupo ' + numeroDoJid(c.chat_jid);
    return c.nome || formatarTelefone(numeroDoJid(c.chat_jid)) || c.chat_jid;
  }
  function horaChat(iso) {
    const d = new Date(iso);
    if (isNaN(d)) return '';
    const hoje = new Date();
    return d.toDateString() === hoje.toDateString() ? d.toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'}) : d.toLocaleDateString([], {day:'2-digit', month:'2-digit'}) + ' ' + d.toLocaleTimeString([], {hour:'2-digit', minute:'2-digit'});
  }
  function resumoMensagemChat(m) {
    if (!m) return '';
    const texto = m.conteudo || (m.nome_arquivo ? '📎 ' + m.nome_arquivo : '[' + (m.tipo || 'mensagem') + ']');
    return (m.direcao === 'saida' ? 'Voce: ' : '') + texto;
  }
  function renderizarConversasChat() {
    const lista = document.getElementById('chat-conversas');
    const conversas = chatUI.conversas.slice();
    // Conversa aberta pelo numero ainda sem mensagens aparece no topo.
    if (chatUI.chat && !conversas.some(c => c.chat_jid === chatUI.chat)) conversas.unshift({ chat_jid: chatUI.chat, nova: true });
    if (!conversas.length) {
      lista.innerHTML = '<div class="dc-lista-vazia">Nenhuma mensagem ainda. Mande uma mensagem para o numero desta instancia ou abra uma conversa pelo numero acima.</div>';
      return;
    }
    lista.innerHTML = conversas.map(c => '<button type="button" class="dc-chat-conversa' + (c.chat_jid === chatUI.chat ? ' ativa' : '') + '" data-jid="' + textoSeguro(c.chat_jid) + '">' +
      '<span class="dc-avatar dc-avatar-mini"><span>' + textoSeguro(c.grupo ? 'G' : iniciais(c.nome || numeroDoJid(c.chat_jid).slice(-2))) + '</span></span>' +
      '<span class="dc-linha-texto"><strong>' + textoSeguro(nomeConversa(c)) + '</strong><span>' + textoSeguro(c.nova ? 'Nova conversa' : resumoMensagemChat(c.ultima)) + '</span></span>' +
      '<small>' + textoSeguro(c.ultima ? horaChat(c.ultima.criado_em) : '') + '</small></button>').join('');
    lista.querySelectorAll('.dc-chat-conversa').forEach(b => b.addEventListener('click', () => abrirConversaChat(b.dataset.jid)));
  }
  function abrirConversaChat(jid) {
    chatUI.chat = jid;
    chatUI.assinatura = '';
    renderizarConversasChat();
    renderizarJanelaChat();
    document.getElementById('chat-mensagens').innerHTML = '<div class="dc-lista-vazia">Carregando...</div>';
    carregarMensagensChat();
    document.getElementById('chat-texto').focus();
  }
  function fecharConversaChat() { chatUI.chat = ''; chatUI.assinatura = ''; renderizarConversasChat(); renderizarJanelaChat(); }
  function renderizarJanelaChat() {
    const aberta = !!chatUI.chat;
    document.getElementById('chat-card').classList.toggle('com-conversa', aberta);
    document.getElementById('chat-texto').disabled = !aberta;
    document.getElementById('chat-enviar').disabled = !aberta;
    const conversa = chatUI.conversas.find(c => c.chat_jid === chatUI.chat) || { chat_jid: chatUI.chat, grupo: String(chatUI.chat).endsWith('@g.us') };
    document.getElementById('chat-topo-nome').textContent = aberta ? nomeConversa(conversa) : 'Selecione uma conversa';
    document.getElementById('chat-topo-sub').textContent = aberta ? (conversa.grupo ? conversa.chat_jid : formatarTelefone(numeroDoJid(conversa.chat_jid))) : 'Mensagens recebidas e enviadas por esta instancia aparecem aqui.';
    document.getElementById('chat-topo-avatar').innerHTML = aberta ? '<span>' + textoSeguro(conversa.grupo ? 'G' : iniciais(conversa.nome || numeroDoJid(conversa.chat_jid).slice(-2))) + '</span>' : '<svg class="dc-ico"><use href="#i-pessoa"/></svg>';
    if (!aberta) document.getElementById('chat-mensagens').innerHTML = '<div class="dc-lista-vazia"><svg class="dc-ico"><use href="#i-mensagem"/></svg>Escolha uma conversa ao lado.</div>';
  }
  async function carregarMensagensChat() {
    const id = estadoUI.instanciaSelecionada, jid = chatUI.chat;
    if (!id || !jid) return;
    try {
      const d = await buscarChat('/api/v1/instancias/' + encodeURIComponent(id) + '/chat/mensagens?chat_jid=' + encodeURIComponent(jid));
      if (!d || jid !== chatUI.chat) return;
      const mensagens = d.mensagens || [];
      // So redesenha quando algo mudou, para nao perder a rolagem.
      const assinatura = mensagens.map(m => m.mensagem_id + m.status + m.conteudo).join('|');
      if (assinatura === chatUI.assinatura) return;
      chatUI.assinatura = assinatura;
      renderizarMensagensChat(mensagens);
    } catch (err) {
      document.getElementById('chat-mensagens').innerHTML = '<div class="dc-lista-vazia">' + textoSeguro(err.message) + '</div>';
    }
  }
  const STATUS_CHAT = { enviada:['✓','Enviada'], entregue:['✓✓','Entregue'], lida:['✓✓','Lida'], ouvida:['✓✓','Ouvida'], falhou:['!','Nao entregue'] };
  function midiaChatHTML(m) {
    if (!m.midia_id) return '';
    const url = '/api/v1/instancias/' + encodeURIComponent(estadoUI.instanciaSelecionada) + '/midias/' + encodeURIComponent(m.midia_id);
    const mime = String(m.mime_type || '');
    if (mime.startsWith('image/')) return '<a href="' + url + '" target="_blank" rel="noopener"><img class="dc-chat-midia" src="' + url + '" alt="Imagem" loading="lazy" onerror="this.parentNode.textContent=\'📎 Abrir imagem\'"></a>';
    if (mime.startsWith('audio/')) return '<audio class="dc-chat-audio" controls preload="none" src="' + url + '"></audio>';
    if (mime.startsWith('video/')) return '<video class="dc-chat-midia" controls preload="none" src="' + url + '"></video>';
    return '<a class="dc-chat-arquivo" href="' + url + '" target="_blank" rel="noopener"><svg class="dc-ico"><use href="#i-clipe"/></svg>' + textoSeguro(m.nome_arquivo || 'arquivo') + '</a>';
  }
  function renderizarMensagensChat(mensagens) {
    const caixa = document.getElementById('chat-mensagens');
    const noFim = caixa.scrollHeight - caixa.scrollTop - caixa.clientHeight < 80;
    if (!mensagens.length) { caixa.innerHTML = '<div class="dc-lista-vazia">Nenhuma mensagem nesta conversa ainda.</div>'; return; }
    const grupo = String(chatUI.chat).endsWith('@g.us');
    caixa.innerHTML = mensagens.map(m => {
      const saida = m.direcao === 'saida';
      const st = STATUS_CHAT[m.status];
      const conteudo = m.conteudo && !(m.midia_id && m.conteudo === m.nome_arquivo) ? '<p>' + textoSeguro(m.conteudo) + '</p>' : '';
      const especial = !m.conteudo && !m.midia_id ? '<p class="dc-chat-tipo">[' + textoSeguro(m.tipo || 'mensagem') + ']</p>' : '';
      return '<div class="dc-chat-bolha ' + (saida ? 'saida' : 'entrada') + (m.status === 'falhou' ? ' falhou' : '') + (m.tipo === 'apagada' ? ' apagada' : '') + '">' +
        (grupo && !saida && m.nome ? '<strong class="dc-chat-autor">' + textoSeguro(m.nome) + '</strong>' : '') +
        midiaChatHTML(m) + conteudo + especial +
        '<span class="dc-chat-meta">' + textoSeguro(horaChat(m.criado_em)) +
        (saida && st ? ' <span class="dc-chat-status ' + textoSeguro(m.status) + '" title="' + textoSeguro(st[1]) + '">' + st[0] + '</span>' : '') + '</span>' +
        (m.erro ? '<span class="dc-chat-erro">' + textoSeguro(m.erro) + '</span>' : '') +
      '</div>';
    }).join('');
    if (noFim || !caixa.dataset.rolou) { caixa.scrollTop = caixa.scrollHeight; caixa.dataset.rolou = '1'; }
  }
  function novaConversaChat(e) {
    e.preventDefault();
    const campo = document.getElementById('chat-novo-numero');
    const numero = campo.value.replace(/\D/g, '');
    if (numero.length < 10) { mostrarToast('Informe o numero com DDI e DDD, ex.: 5567999999999.', 'error'); return; }
    campo.value = '';
    abrirConversaChat(numero + '@s.whatsapp.net');
  }
  function teclaChat(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); enviarMensagemChat(e); }
  }
  function ajustarAlturaChat() {
    const t = document.getElementById('chat-texto');
    t.style.height = 'auto';
    t.style.height = Math.min(t.scrollHeight, 160) + 'px';
  }
  function anexoChatSelecionado() {
    const arquivo = document.getElementById('chat-arquivo').files[0] || null;
    if (arquivo && arquivo.size > 15 * 1024 * 1024) { mostrarToast('Arquivo maior que 15 MB.', 'error'); document.getElementById('chat-arquivo').value = ''; return; }
    chatUI.arquivo = arquivo;
    const rotulo = document.getElementById('chat-anexo-nome');
    rotulo.classList.toggle('hidden', !arquivo);
    rotulo.innerHTML = arquivo ? textoSeguro('📎 ' + arquivo.name) + ' <button type="button" onclick="limparAnexoChat()" title="Remover">×</button>' : '';
  }
  function limparAnexoChat() { document.getElementById('chat-arquivo').value = ''; anexoChatSelecionado(); }
  function lerArquivoBase64(arquivo) {
    return new Promise((ok, falha) => { const r = new FileReader(); r.onload = () => ok(String(r.result).split(',')[1] || ''); r.onerror = () => falha(new Error('Nao foi possivel ler o arquivo')); r.readAsDataURL(arquivo); });
  }
  async function enviarMensagemChat(e) {
    e.preventDefault();
    if (!chatUI.chat || chatUI.enviando) return;
    const campo = document.getElementById('chat-texto');
    const texto = campo.value.trim(), arquivo = chatUI.arquivo;
    if (!texto && !arquivo) return;
    const destino = { instancia: estadoUI.instanciaSelecionada };
    if (chatUI.chat.endsWith('@g.us')) destino.chat_jid = chatUI.chat; else destino.numero = numeroDoJid(chatUI.chat);
    const botao = document.getElementById('chat-enviar');
    chatUI.enviando = true; botao.disabled = true;
    try {
      let resposta;
      if (arquivo) {
        const mime = arquivo.type || 'application/octet-stream';
        const rota = mime.startsWith('image/') ? 'imagem' : mime.startsWith('audio/') ? 'audio' : 'documento';
        resposta = await chamar('/api/v1/batepapo/enviar/' + rota, { method:'POST', body: JSON.stringify(Object.assign({}, destino, { arquivo_base64: await lerArquivoBase64(arquivo), nome_arquivo: arquivo.name, mime_type: mime, legenda: texto })) });
        limparAnexoChat();
      } else {
        resposta = await chamar('/api/v1/batepapo/enviar/texto', { method:'POST', body: JSON.stringify(Object.assign({}, destino, { mensagem: texto })) });
      }
      campo.value = ''; ajustarAlturaChat();
      // O WhatsApp pode trocar o numero (ex.: 9o digito); segue a conversa real.
      const jid = resposta?.dados?.chat_jid;
      if (jid && jid.endsWith('@s.whatsapp.net') && jid !== chatUI.chat) { chatUI.chat = jid; renderizarJanelaChat(); }
      chatUI.assinatura = '';
      await carregarChat();
    } catch (err) {
      mostrarToast('Nao enviou: ' + err.message, 'error');
    } finally {
      chatUI.enviando = false; botao.disabled = !chatUI.chat;
      campo.focus();
    }
  }

  function usoAberto() { return estadoUI.abaAtual === 'instancia' && estadoUI.abaInstancia === 'geral'; }
  function auditoriaAberta() { return estadoUI.abaAtual === 'instancia' && estadoUI.abaInstancia === 'webhooks'; }

  function formatarTelefone(numero) {
    const n = String(numero || '').replace(/\D/g, '');
    if (n.startsWith('55') && (n.length === 12 || n.length === 13)) {
      const ddd = n.slice(2, 4), resto = n.slice(4);
      return '+55 (' + ddd + ') ' + resto.slice(0, resto.length - 4) + '-' + resto.slice(-4);
    }
    return n ? '+' + n : '';
  }
  function iniciais(nome) {
    return String(nome || '?').trim().split(/\s+/).slice(0, 2).map(p => p[0] || '').join('').toUpperCase() || '?';
  }
  function avatarHTML(perfil, nome) {
    if (perfil && perfil.foto_url) return '<img src="'+textoSeguro(perfil.foto_url)+'" alt="" referrerpolicy="no-referrer">';
    return '<span>'+textoSeguro(iniciais((perfil && perfil.nome) || nome))+'</span>';
  }
  function renderizarCabecalhoInstancia(d) {
    const perfil = d.perfil || {};
    const conectada = d.status === 'conectada';
    document.getElementById('inst-avatar').innerHTML = d.id ? avatarHTML(perfil, d.nome) : '<svg class="dc-ico"><use href="#i-pessoa"/></svg>';
    document.getElementById('inst-avatar').classList.toggle('online', conectada);
    const oficial = d.tipo === 'meta';
    document.getElementById('inst-business').classList.toggle('hidden', !perfil.business || oficial);
    document.getElementById('inst-oficial').classList.toggle('hidden', !oficial);
    document.querySelector('.dc-inst').classList.toggle('eh-meta', oficial);
    const partes = [perfil.nome_empresa || perfil.nome, formatarTelefone(perfil.numero)].filter(Boolean);
    document.getElementById('inst-perfil').textContent = partes.length ? partes.join(' · ') : 'Conecte para ver o perfil';
    const navAvatar = document.getElementById('nav-inst-avatar');
    navAvatar.innerHTML = d.id ? avatarHTML(perfil, d.nome) : '<svg class="dc-ico"><use href="#i-pessoa"/></svg>';
    navAvatar.classList.toggle('online', conectada);
    document.getElementById('nav-inst-nome').textContent = d.nome || '-';
    document.getElementById('nav-inst-sub').textContent = perfil.nome_empresa || perfil.nome || rotuloStatus(d.status);
    document.getElementById('botao-conectar').classList.toggle('hidden', conectada);
    document.getElementById('botao-desconectar').classList.toggle('hidden', !conectada || !d.id);
    document.getElementById('botao-conectar').textContent = oficial ? 'Validar credenciais' : ['aguardando_qrcode','conectando','pareada','autenticando','sincronizando_historico'].includes(d.status) ? 'Ver QR code' : 'Conectar';
    if (oficial) document.getElementById('botao-desconectar').classList.add('hidden');
    if (oficial && d.webhook_url !== undefined) renderizarCartaoMeta(d);
    document.getElementById('config-proxy').textContent = d.proxy_configurado ? 'Proprio: ' + (d.proxy_url || '').replace(/\/\/[^@]*@/, '//***@') : 'Usa o proxy global do sistema (se houver).';
    document.getElementById('config-historico').textContent = d.historico_dias ? d.historico_dias + ' dias de historico ao conectar.' : 'Sem historico ao conectar.';
  }
  // Cartao da API oficial: numero, qualidade e o que vai no painel da Meta.
  function renderizarCartaoMeta(d) {
    estadoUI.metaSelecionada = d;
    const definir = (id, valor) => { document.getElementById(id).textContent = valor || '-'; };
    definir('meta-numero', d.numero_exibicao);
    definir('meta-nome', d.nome_verificado);
    definir('meta-qualidade', ({GREEN:'Alta', YELLOW:'Media', RED:'Baixa'})[d.qualidade] || d.qualidade);
    definir('meta-phone-id', d.phone_number_id);
    definir('meta-webhook-url', d.webhook_url);
    definir('meta-verify-token', d.webhook_verify_token);
    document.getElementById('meta-aviso-secret').textContent = d.app_secret_salvo ? '' :
      'Sem App secret salvo: a assinatura dos webhooks nao e conferida. Cadastre em Atualizar credenciais.';
  }
  async function copiarTexto(id) {
    const texto = document.getElementById(id).textContent;
    try { await navigator.clipboard.writeText(texto); mostrarToast('Copiado.', 'success'); }
    catch (err) { mostrarToast('Nao foi possivel copiar.', 'error'); }
  }
  function alternarTokenVisivel() {
    const el = document.getElementById('detalhe-token');
    atualizarTokenDetalhado(el.dataset.token || '', el.dataset.revelado !== 'true');
  }

  // ── Conexao (QR) ──────────────────────────────────────────────────────────────
  function definirQR(html) {
    const box = document.getElementById('qrcode-box');
    if (box) box.innerHTML = html;
  }
  async function abrirConexao() {
    if (!estadoUI.instanciaSelecionada) return;
    if (estadoUI.metaSelecionada && estadoUI.metaSelecionada.id === estadoUI.instanciaSelecionada) {
      try {
        await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/conectar', { method:'POST' });
        mostrarToast('Credenciais validadas na Meta.', 'success');
      } catch (err) { mostrarToast(err.message, 'error'); }
      await atualizarInstanciaSelecionada(true);
      return;
    }
    estadoUI.qrAtual = '';
    abrirModal('conectar');
    if (estadoUI.statusSelecionada === 'aguardando_qrcode') { await mostrarQRCodeSelecionado(); iniciarPollingConexao(); return; }
    try { await conectarSelecionada(); }
    catch (err) {
      definirQR('<div class="qrcode-loading">Erro ao conectar: '+textoSeguro(err.message)+'</div>');
      const rodape = document.querySelector('.dc-aguardando');
      if (rodape) { rodape.classList.add('erro'); rodape.innerHTML = '<span class="dc-ponto offline"></span>Nao foi possivel conectar. Confira o erro e o proxy da instancia.'; }
    }
  }
  function mostrarConexaoConcluida(d) {
    const perfil = d.perfil || {};
    const nome = perfil.nome_empresa || perfil.nome || d.nome;
    document.getElementById('modal-kicker').textContent = '';
    document.getElementById('modal-titulo').textContent = '';
    document.getElementById('modal-corpo').innerHTML =
      '<div class="dc-conectado">' +
        '<span class="dc-conectado-icone"><svg class="dc-ico"><use href="#i-ok"/></svg></span>' +
        '<h3>Conectado</h3><p>O WhatsApp foi conectado com sucesso.</p>' +
        '<div class="dc-conectado-perfil"><span class="dc-avatar online">'+avatarHTML(perfil, d.nome)+'</span>' +
          '<span class="dc-linha-texto"><strong>'+textoSeguro(nome)+'</strong><span>'+textoSeguro(formatarTelefone(perfil.numero))+'</span></span>' +
          (perfil.business ? '<span class="dc-selo">Business</span>' : '') + '</div>' +
        '<div class="dc-conectado-acoes"><button class="ghost" onclick="fecharModal()">Fechar</button><button class="primary" onclick="fecharModal();trocarAbaInstancia(\'geral\')">Gerenciar</button></div>' +
      '</div>';
    modalTipoAtual = 'conectado';
  }

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
        '<div class="webhook-eventos">'+textoSeguro((wh.eventos||[]).join(', '))+(wh.incluir_base64 === false ? ' · midia so com link' : ' · midia com base64')+'</div>' +
        '<div class="webhook-actions">' +
          '<button class="ghost small" onclick="editarWebhook(\''+wh.id+'\')">Editar</button>' +
          '<button class="ghost small" onclick="alternarWebhookAtivo(\''+wh.id+'\')">'+(wh.ativo?'Desativar':'Ativar')+'</button>' +
          '<button class="ghost small" onclick="excluirWebhook(\''+wh.id+'\')">Excluir</button>' +
        '</div>' +
      '</div>'
    ).join('');
  }



  async function carregarUso(manual) {
    const box = document.getElementById('uso-conteudo');
    if (!box || !estadoUI.instanciaSelecionada) return;
    if (manual) box.innerHTML = '<div class="qrcode-loading">Consultando uso...</div>';
    try {
      const resp = await chamar('/api/v1/instancias/' + estadoUI.instanciaSelecionada + '/uso');
      if (resp?.dados) renderizarUso(resp.dados);
    } catch(err) {
      box.className = 'webhook-delivery-list empty-state';
      box.textContent = 'Erro ao carregar uso: ' + err.message;
    }
  }

  function numeroUso(v) { return String(Number(v) || 0); }

  function contatoUso(jid) {
    const [usuario, servidor] = String(jid || '').split('@');
    return textoSeguro(usuario || '-') + (servidor === 'lid' ? ' <span class="uso-nota">(ID interno)</span>' : '');
  }

  function listaContatosUso(titulo, ajuda, lista, sufixo) {
    const itens = lista.length
      ? lista.map(c => '<li><span>'+contatoUso(c.chat_jid)+'</span><strong>'+numeroUso(c.envios)+sufixo+'</strong></li>').join('')
      : '<li class="uso-vazio">Nenhum hoje.</li>';
    return '<div class="uso-bloco"><h4>'+titulo+'</h4><p class="helper-text">'+ajuda+'</p><ul class="uso-lista">'+itens+'</ul></div>';
  }

  function renderizarUso(u) {
    const box = document.getElementById('uso-conteudo');
    const limite = u.limite_por_minuto || 0;
    const nivelMinuto = limite && u.ultimo_minuto >= limite * 0.8 ? 'warn' : '';
    const pico = u.pico_minuto_hoje ? numeroUso(u.pico_minuto_hoje) + (u.pico_minuto_em ? ' <small>as '+new Date(u.pico_minuto_em).toLocaleTimeString('pt-BR',{hour:'2-digit',minute:'2-digit'})+'</small>' : '') : '0';
    const dias = (u.dias || []).slice().reverse().map(d =>
      '<tr><td>'+textoSeguro(new Date(d.dia+'T12:00:00').toLocaleDateString('pt-BR',{weekday:'short',day:'2-digit',month:'2-digit'}))+'</td>' +
      '<td>'+numeroUso(d.envios)+'</td><td>'+numeroUso(d.contatos_novos)+'</td><td class="'+(d.limitados?'warn':'')+'">'+numeroUso(d.limitados)+'</td></tr>'
    ).join('');
    box.className = 'uso-conteudo';
    box.innerHTML =
      '<div class="delivery-summary uso-resumo">' +
        '<div><span>Ultimo minuto</span><strong class="'+nivelMinuto+'">'+numeroUso(u.ultimo_minuto)+(limite?'<small> / '+limite+'</small>':'')+'</strong></div>' +
        '<div><span>Ultima hora</span><strong>'+numeroUso(u.ultima_hora)+'</strong></div>' +
        '<div><span>Envios hoje</span><strong>'+numeroUso(u.hoje.envios)+'</strong></div>' +
        '<div><span>Pico por minuto</span><strong>'+pico+'</strong></div>' +
        '<div><span>Contatos novos hoje</span><strong>'+numeroUso(u.hoje.contatos_novos)+'</strong></div>' +
        '<div><span>Barrados pelo limite</span><strong class="'+(u.hoje.limitados?'warn':'')+'">'+numeroUso(u.hoje.limitados)+'</strong></div>' +
      '</div>' +
      '<p class="helper-text">Contato novo e a primeira mensagem para quem nunca conversou com este numero. Muitos contatos novos por dia e muitas mensagens seguidas para a mesma pessoa sao os principais motivos de bloqueio pelo WhatsApp.</p>' +
      '<div class="uso-colunas">' +
        listaContatosUso('Rajadas hoje', '5 ou mais mensagens para a mesma pessoa no mesmo minuto.', u.rajadas || [], ' / min') +
        listaContatosUso('Quem mais recebeu hoje', 'Destinatarios com mais envios no dia.', u.top_destinatarios || [], '') +
      '</div>' +
      '<div class="uso-bloco"><h4>Ultimos 7 dias</h4><table class="uso-tabela"><thead><tr><th>Dia</th><th>Envios</th><th>Contatos novos</th><th>Barrados</th></tr></thead><tbody>'+dias+'</tbody></table></div>';
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

  // ── Nova instancia ────────────────────────────────────────────────────────────
  function passoNovaInstancia(passo) {
    document.querySelectorAll('.dc-wizard-nav button').forEach(b => b.classList.toggle('ativo', b.dataset.passo === passo));
    document.querySelectorAll('.dc-wizard-corpo section').forEach(sec => sec.classList.toggle('hidden', sec.dataset.passo !== passo));
  }
  function atualizarNovaInstancia() {
    const webhook = document.getElementById('ni-webhook-ativo').checked;
    const proxy = document.getElementById('ni-proxy-ativo').checked;
    document.getElementById('ni-webhook-campos').classList.toggle('dc-desativado', !webhook);
    document.getElementById('ni-proxy-campos').classList.toggle('dc-desativado', !proxy);
    document.getElementById('ni-tag-webhook').classList.toggle('hidden', !webhook);
    document.getElementById('ni-tag-proxy').classList.toggle('hidden', !proxy);
    document.getElementById('ni-mensagem-wrap').classList.toggle('hidden', !document.getElementById('ni-rejeitar-chamadas').checked);
    const meta = tipoNovaInstancia() === 'meta';
    document.getElementById('ni-meta-campos').classList.toggle('hidden', !meta);
    document.getElementById('modal-form').classList.toggle('eh-meta', meta);
    document.getElementById('mf-submit').textContent = meta ? 'Validar e criar' : 'Criar instancia';
  }
  function tipoNovaInstancia() {
    return (document.querySelector('input[name="ni-tipo"]:checked') || {}).value || 'whatsapp';
  }
  function gerarTokenNovaInstancia() {
    const bytes = new Uint8Array(24);
    crypto.getRandomValues(bytes);
    document.getElementById('ni-token').value = Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('');
  }
  async function criarNovaInstancia(e) {
    e.preventDefault();
    const btn = document.getElementById('mf-submit'), fb = document.getElementById('mf-feedback');
    const valor = id => document.getElementById(id).value.trim();
    const marcado = id => document.getElementById(id).checked;
    const falhar = (passo, msg) => { passoNovaInstancia(passo); fb.className = 'modal-feedback error'; fb.textContent = msg; };
    const nome = valor('ni-nome');
    if (!nome) return falhar('geral', 'Informe o nome da instancia.');
    const webhook = marcado('ni-webhook-ativo');
    const eventos = Array.from(document.querySelectorAll('.ni-evento:checked')).map(el => el.value);
    if (webhook && (!valor('ni-webhook-url') || !eventos.length)) return falhar('webhook', 'Informe a URL e ao menos um evento do webhook.');
    const meta = tipoNovaInstancia() === 'meta';
    if (meta && (!valor('ni-meta-phone') || !document.getElementById('ni-meta-token').value.trim())) return falhar('geral', 'Informe o Phone number ID e o access token da Meta.');
    const proxy = !meta && marcado('ni-proxy-ativo');
    if (proxy && (!valor('ni-proxy-host') || !valor('ni-proxy-porta'))) return falhar('proxy', 'Informe host e porta do proxy.');

    btn.disabled = true; btn.textContent = meta ? 'Validando na Meta...' : 'Criando...'; fb.className = 'modal-feedback hidden';
    let id = '';
    const pendencias = [];
    try {
      const corpo = { nome };
      if (meta) {
        corpo.tipo = 'meta';
        corpo.meta = { phone_number_id: valor('ni-meta-phone'), waba_id: valor('ni-meta-waba'),
          access_token: document.getElementById('ni-meta-token').value.trim(), app_secret: document.getElementById('ni-meta-secret').value.trim() };
      }
      const r = await chamar('/api/v1/instancias', { method:'POST', body: JSON.stringify(corpo) });
      if (!r) return;
      id = r.dados.id;
      const base = '/api/v1/instancias/' + id;
      // Cada ajuste e uma chamada propria: se um falhar, a instancia ja existe e o resto segue.
      const etapa = async (rotulo, url, metodo, corpo) => {
        try { await chamar(url, { method: metodo, body: JSON.stringify(corpo) }); }
        catch (err) { pendencias.push(rotulo + ': ' + err.message); }
      };
      if (valor('ni-token')) await etapa('Token', base + '/token', 'PUT', { token: valor('ni-token') });
      await etapa('Comportamento', base + '/avancado', 'PUT', {
        manter_online: marcado('ni-manter-online'), rejeitar_chamadas: marcado('ni-rejeitar-chamadas'),
        mensagem_rejeitar_chamadas: valor('ni-mensagem-rejeitar'), marcar_lida_automatico: marcado('ni-marcar-lida'),
        ignorar_grupos: marcado('ni-ignorar-grupos'), ignorar_status: marcado('ni-ignorar-status') });
      const dias = meta ? 0 : parseInt(valor('ni-historico')) || 0;
      if (dias > 0) await etapa('Historico', base + '/historico', 'PUT', { dias });
      if (webhook) await etapa('Webhook', base + '/webhooks', 'POST', { nome: 'Principal', url: valor('ni-webhook-url'), eventos });
      if (proxy) {
        const protocolo = document.querySelector('input[name="ni-proxy-protocolo"]:checked').value;
        const usuario = valor('ni-proxy-usuario'), senha = document.getElementById('ni-proxy-senha').value;
        const credencial = usuario ? encodeURIComponent(usuario) + (senha ? ':' + encodeURIComponent(senha) : '') + '@' : '';
        await etapa('Proxy', base + '/proxy', 'PUT', { modo: protocolo, url: protocolo + '://' + credencial + valor('ni-proxy-host') + ':' + valor('ni-proxy-porta') });
      }
    } catch (err) {
      fb.className = 'modal-feedback error'; fb.textContent = 'Erro ao criar: ' + err.message;
      return;
    } finally { btn.disabled = false; btn.textContent = meta ? 'Validar e criar' : 'Criar instancia'; }
    fecharModal();
    await carregarTudo(true);
    if (pendencias.length) mostrarToast('Instancia criada, mas falhou: ' + pendencias.join(' | ') + '. Ajuste na pagina da instancia.', 'error');
    else mostrarToast(meta ? 'Instancia oficial criada. Cadastre a URL de webhook na Meta (aba Configuracoes).' : 'Instancia criada. Conecte para ler o QR code.', 'success');
    await selecionarInstancia(id);
    if (meta) trocarAbaInstancia('config');
  }


  // ── Carregamento geral ────────────────────────────────────────────────────────
  async function carregarTudo(manual) {
    if (estadoUI.acesso?.tipo === 'master') await carregarSistema();
    await carregarInstancias(Boolean(manual));
    await carregarUsoGeral();
    if (estadoUI.instanciaSelecionada && usoAberto()) await carregarUso(false);
    if (estadoUI.instanciaSelecionada && auditoriaAberta()) { await carregarWebhooks(); await carregarAuditoria(false); }
  }

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
    if (estadoUI.acesso?.tipo === 'instancia' && estadoUI.instanciaSelecionada) trocarAba('instancia');
  })();
  </script>
</body>
</html>`
