package dashboard

const paginaLoginHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{ .Titulo }}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@500;700&family=Material+Symbols+Outlined:opsz,wght,FILL,GRAD@24,500,0,0" rel="stylesheet">
  <link rel="stylesheet" href="/static/css/dashboard.css">
</head>
<body>
  <main class="login-shell">
    <section class="login-card glass-card">
      <img src="/static/img/dyalog.png" alt="Dyalog Internet Solutions" class="login-logo-img">
      <p class="login-copy">Entre com o token master para administrar todas as instancias ou com o token da sua instancia para abrir apenas o seu painel.</p>
      <form id="form-login" class="stack-form login-form">
        <label>
          <span>Token de acesso</span>
          <input type="password" id="login-token" placeholder="Cole o token de acesso" required>
        </label>
        <button type="submit" class="primary login-submit">Entrar</button>
      </form>
      <div class="login-info-box">
        <strong>Como funciona</strong>
        <p>Token master: ve e cria instancias. Token de instancia: acessa somente a instancia vinculada.</p>
      </div>
      <p id="login-erro" class="login-error hidden"></p>
      <p class="app-version-label login-version">v1.0</p>
    </section>
  </main>
  <script>
    document.getElementById('form-login').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      const erro = document.getElementById('login-erro');
      erro.className = 'login-error hidden';
      const resposta = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: document.getElementById('login-token').value })
      });
      const dados = await resposta.json().catch(() => null);
      if (!resposta.ok) {
        erro.textContent = dados && dados.mensagem ? dados.mensagem : 'Falha ao autenticar';
        erro.className = 'login-error';
        return;
      }
      window.location.href = '/';
    });
  </script>
</body>
</html>`
