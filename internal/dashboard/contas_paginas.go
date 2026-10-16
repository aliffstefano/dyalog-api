package dashboard

import (
	"html/template"
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

// cabecalhoPaginaPublica e o topo comum das paginas sem login (convite e
// redefinir senha), no mesmo visual do login.
const cabecalhoPaginaPublica = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="icon" href="/static/img/favicon-32.png" sizes="32x32" type="image/png">
  <title>{{ .Titulo }}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/static/css/dashboard.css?v=20261018">
</head>
<body>
  <main class="login-shell">
    <section class="login-card glass-card">
      <div class="login-marca"><img src="/static/img/dyalog-connect.png" alt="" class="login-marca-icone"><div class="login-marca-nome"><strong>Dyalog</strong><span>Connect</span></div></div>`

const scriptsPaginaPublica = `
  <script>
    const tokenDaUrl = new URLSearchParams(location.search).get('token') || '';
    function aviso(erro, ok) {
      const e = document.getElementById('login-erro'), o = document.getElementById('login-ok');
      e.textContent = erro; e.className = erro ? 'login-error' : 'login-error hidden';
      o.textContent = ok; o.className = ok ? 'login-ok' : 'login-ok hidden';
    }
    async function enviar(url, corpo) {
      const resposta = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(corpo) });
      const dados = await resposta.json().catch(() => null);
      return { ok: resposta.ok, mensagem: dados && dados.mensagem, dados: dados && dados.dados };
    }
    function senhasConferem() {
      const a = document.getElementById('senha').value, b = document.getElementById('senha2').value;
      if (a.length < 8) { aviso('A senha precisa ter pelo menos 8 caracteres.', ''); return false; }
      if (a !== b) { aviso('As senhas nao conferem.', ''); return false; }
      return true;
    }
  </script>`

const paginaConviteHTML = cabecalhoPaginaPublica + `
      <h1 class="login-titulo">Criar sua conta</h1>
      <p class="login-copy" id="convite-info">Conferindo o convite...</p>
      <form id="form-convite" class="stack-form login-form hidden">
        <label><span>E-mail</span><input type="email" id="convite-email" disabled></label>
        <label><span>Seu nome</span><input type="text" id="nome" autocomplete="name" required></label>
        <label><span>Senha (minimo 8 caracteres)</span><input type="password" id="senha" autocomplete="new-password" required></label>
        <label><span>Repita a senha</span><input type="password" id="senha2" autocomplete="new-password" required></label>
        <button type="submit" class="primary login-submit">Criar conta e entrar</button>
      </form>
      <p id="login-erro" class="login-error hidden"></p>
      <p id="login-ok" class="login-ok hidden"></p>
    </section>
  </main>` + scriptsPaginaPublica + `
  <script>
    const papeis = { dono: 'dono', admin: 'administrador', atendente: 'atendente' };
    (async function() {
      const r = await fetch('/api/v1/auth/convite?token=' + encodeURIComponent(tokenDaUrl));
      const d = await r.json().catch(() => null);
      const info = document.getElementById('convite-info');
      if (!r.ok) { info.textContent = (d && d.mensagem) || 'Convite invalido.'; return; }
      const c = d.dados;
      info.textContent = c.superadmin ? 'Voce foi convidado como administrador do Dyalog Connect.' :
        'Voce foi convidado para a equipe de ' + (c.conta_nome || 'uma empresa') + ' como ' + (papeis[c.papel] || c.papel) + '.';
      document.getElementById('convite-email').value = c.email;
      document.getElementById('form-convite').classList.remove('hidden');
      document.getElementById('nome').focus();
    })();
    document.getElementById('form-convite').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      if (!senhasConferem()) return;
      const r = await enviar('/api/v1/auth/convite/aceitar', { token: tokenDaUrl, nome: document.getElementById('nome').value, senha: document.getElementById('senha').value });
      if (!r.ok) { aviso(r.mensagem || 'Nao foi possivel criar a conta', ''); return; }
      window.location.href = '/';
    });
  </script>
</body>
</html>`

const paginaRedefinirHTML = cabecalhoPaginaPublica + `
      <h1 class="login-titulo">Nova senha</h1>
      <form id="form-redefinir" class="stack-form login-form">
        <label><span>Nova senha (minimo 8 caracteres)</span><input type="password" id="senha" autocomplete="new-password" required></label>
        <label><span>Repita a senha</span><input type="password" id="senha2" autocomplete="new-password" required></label>
        <button type="submit" class="primary login-submit">Salvar nova senha</button>
      </form>
      <p id="login-erro" class="login-error hidden"></p>
      <p id="login-ok" class="login-ok hidden"></p>
    </section>
  </main>` + scriptsPaginaPublica + `
  <script>
    document.getElementById('senha').focus();
    document.getElementById('form-redefinir').addEventListener('submit', async function(evento) {
      evento.preventDefault();
      if (!senhasConferem()) return;
      const r = await enviar('/api/v1/auth/senha/redefinir', { token: tokenDaUrl, senha: document.getElementById('senha').value });
      if (!r.ok) { aviso(r.mensagem || 'Nao foi possivel salvar', ''); return; }
      document.getElementById('form-redefinir').classList.add('hidden');
      aviso('', 'Senha salva. Redirecionando para o login...');
      setTimeout(() => { window.location.href = '/'; }, 1500);
    });
  </script>
</body>
</html>`

var (
	conviteTmpl   = template.Must(template.New("convite").Parse(paginaConviteHTML))
	redefinirTmpl = template.Must(template.New("redefinir").Parse(paginaRedefinirHTML))
)

// PaginaConvite e onde o convidado cria nome e senha.
func (h *Handler) PaginaConvite(c *gin.Context) {
	c.Status(nethttp.StatusOK)
	_ = conviteTmpl.Execute(c.Writer, paginaInicialData{Titulo: h.cfg.NomeAplicacao})
}

// PaginaRedefinirSenha e o destino do link do e-mail de "esqueci a senha".
func (h *Handler) PaginaRedefinirSenha(c *gin.Context) {
	c.Status(nethttp.StatusOK)
	_ = redefinirTmpl.Execute(c.Writer, paginaInicialData{Titulo: h.cfg.NomeAplicacao})
}
