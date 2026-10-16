package dashboard

const paginaLoginHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="icon" href="/static/img/favicon-32.png" sizes="32x32" type="image/png">
  <link rel="icon" href="/static/img/favicon-64.png" sizes="64x64" type="image/png">
  <link rel="apple-touch-icon" href="/static/img/dyalog-connect.png">
  <title>{{ .Titulo }}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@24,500,0,0" rel="stylesheet">
  <link rel="stylesheet" href="/static/css/dashboard.css?v=20261018">
</head>
<body>
  <main class="login-shell">
    <section class="login-card glass-card">
      <div class="login-marca"><img src="/static/img/dyalog-connect.png" alt="" class="login-marca-icone"><div class="login-marca-nome"><strong>Dyalog</strong><span>Connect</span></div></div>
      <p class="login-slogan">Comunicacao que conecta seu negocio</p>
      <form id="form-login" class="stack-form login-form">
        <label><span>E-mail</span><input type="email" id="login-email" autocomplete="username" placeholder="voce@empresa.com.br"></label>
        <label><span>Senha</span><input type="password" id="login-senha" autocomplete="current-password" placeholder="Sua senha"></label>
        <button type="submit" class="primary login-submit" id="login-botao">Entrar</button>
        <div class="login-links"><button type="button" class="dc-link" onclick="modoLogin('esqueci')">Esqueci minha senha</button><button type="button" class="dc-link" onclick="modoLogin('token')">Entrar com token</button></div>
      </form>
      <form id="form-token" class="stack-form login-form hidden">
        <label><span>Token de acesso</span><input type="password" id="login-token" placeholder="Token master ou token da instancia"></label>
        <button type="submit" class="primary login-submit">Entrar com token</button>
        <div class="login-links"><button type="button" class="dc-link" onclick="modoLogin('login')">Entrar com e-mail e senha</button></div>
      </form>
      <form id="form-esqueci" class="stack-form login-form hidden">
        <p class="login-copy">Informe o seu e-mail. Se ele tiver cadastro, enviamos um link para criar uma nova senha.</p>
        <label><span>E-mail</span><input type="email" id="esqueci-email" autocomplete="username"></label>
        <button type="submit" class="primary login-submit">Enviar link</button>
        <div class="login-links"><button type="button" class="dc-link" onclick="modoLogin('login')">Voltar para o login</button></div>
      </form>
      <p id="login-erro" class="login-error hidden"></p>
      <p id="login-ok" class="login-ok hidden"></p>
      <p class="app-version-label login-version">Acesso por convite. Fale com o administrador da sua empresa.</p>
    </section>
  </main>
  <script>
    function modoLogin(modo) {
      document.getElementById('form-login').classList.toggle('hidden', modo !== 'login');
      document.getElementById('form-token').classList.toggle('hidden', modo !== 'token');
      document.getElementById('form-esqueci').classList.toggle('hidden', modo !== 'esqueci');
      aviso('', ''); 
      const campo = {login:'login-email', token:'login-token', esqueci:'esqueci-email'}[modo];
      setTimeout(() => document.getElementById(campo).focus(), 30);
    }
    function aviso(erro, ok) {
      const e = document.getElementById('login-erro'), o = document.getElementById('login-ok');
      e.textContent = erro; e.className = erro ? 'login-error' : 'login-error hidden';
      o.textContent = ok; o.className = ok ? 'login-ok' : 'login-ok hidden';
    }
    async function enviar(url, corpo) {
      const resposta = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(corpo) });
      const dados = await resposta.json().catch(() => null);
      return { ok: resposta.ok, mensagem: dados && dados.mensagem };
    }
    document.getElementById('form-login').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      const r = await enviar('/api/v1/auth/login', { email: document.getElementById('login-email').value, senha: document.getElementById('login-senha').value });
      if (!r.ok) { aviso(r.mensagem || 'Falha ao entrar', ''); return; }
      window.location.href = '/';
    });
    document.getElementById('form-token').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      const r = await enviar('/api/v1/auth/login', { token: document.getElementById('login-token').value });
      if (!r.ok) { aviso(r.mensagem || 'Token invalido', ''); return; }
      window.location.href = '/';
    });
    document.getElementById('form-esqueci').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      const r = await enviar('/api/v1/auth/senha/esqueci', { email: document.getElementById('esqueci-email').value });
      if (!r.ok) { aviso(r.mensagem || 'Nao foi possivel enviar', ''); return; }
      aviso('', 'Se o e-mail tiver cadastro, o link chega em alguns minutos. Confira tambem o spam.');
    });
    document.getElementById('login-email').focus();
  </script>
</body>
</html>`
