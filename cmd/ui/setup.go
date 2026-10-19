package ui

import (
	"html"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/etoolscode/internal/config"
)

// RegisterSetup registra o wizard de 1ª execução.
func RegisterSetup(app *fiber.App, auth *Auth, cfg *config.Config) {
	app.Get("/setup", func(c *fiber.Ctx) error {
		if auth != nil && !auth.SetupRequired() {
			return c.Redirect("/", fiber.StatusSeeOther)
		}
		return sendSetupPage(c, "")
	})
	app.Post("/api/setup", func(c *fiber.Ctx) error {
		if auth != nil && !auth.SetupRequired() {
			return c.Status(400).JSON(fiber.Map{"error": "setup já concluído"})
		}
		var body struct {
			Password  string `json:"password"`
			LlmClient int    `json:"llm_client"`
			APIKey    string `json:"api_key"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		if len(body.Password) < 6 {
			return c.Status(400).JSON(fiber.Map{"error": "senha deve ter ao menos 6 caracteres"})
		}
		if body.APIKey == "" {
			return c.Status(400).JSON(fiber.Map{"error": "informe a API key do provedor LLM"})
		}
		if body.LlmClient == 0 {
			body.LlmClient = config.LLM_DEEPSEEK
		}
		if err := auth.SavePasswordHash(body.Password); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "falha ao salvar senha: " + err.Error()})
		}
		cfg.LlmClient = body.LlmClient
		switch body.LlmClient {
		case config.LLM_META:
			cfg.MetaApiKey = body.APIKey
		case config.LLM_GEMINI:
			cfg.GeminiApiKey = body.APIKey
		case config.LLM_GROK:
			cfg.GrokApiKey = body.APIKey
		default:
			cfg.DeepSeekApiKey = body.APIKey
		}
		if err := cfg.Save(); err != nil {
			log.Printf("[setup] cfg.Save: %v", err)
		}
		// cria sessão automática
		tok, err := auth.newSession()
		if err == nil {
			c.Cookie(&fiber.Cookie{
				Name: sessionCookie, Value: tok, Path: "/",
				HTTPOnly: true, SameSite: fiber.CookieSameSiteLaxMode, Secure: isSecure(c),
			})
		}
		log.Printf("[setup] wizard concluído")
		return c.JSON(fiber.Map{"ok": true})
	})
}

func sendSetupPage(c *fiber.Ctx, msg string) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	errHTML := ""
	if msg != "" {
		errHTML = `<p class="err">` + html.EscapeString(msg) + `</p>`
	}
	return c.SendString(`<!doctype html>
<html lang="pt-BR"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>eToolsCode — Configuração inicial</title>
<style>
  :root { --bg:#0f1419; --card:#1a2332; --border:#2d3a4d; --text:#e8eef5; --muted:#8b9cb3; --green:#00aa4f; }
  * { box-sizing:border-box; font-family: system-ui, sans-serif; }
  body { margin:0; min-height:100vh; background:var(--bg); color:var(--text);
    display:flex; align-items:center; justify-content:center; padding:24px; }
  .card { width:100%; max-width:440px; background:var(--card); border:1px solid var(--border);
    border-radius:12px; padding:28px; }
  h1 { font-size:1.25rem; margin:0 0 4px; }
  .sub { color:var(--muted); font-size:.85rem; margin-bottom:18px; }
  label { display:block; font-size:.78rem; color:var(--muted); margin:12px 0 6px; }
  input, select { width:100%; padding:10px 12px; border-radius:8px; border:1px solid var(--border);
    background:var(--bg); color:var(--text); }
  button { width:100%; margin-top:18px; padding:12px; border:none; border-radius:8px;
    background:var(--green); color:#fff; font-weight:600; cursor:pointer; }
  button:hover { filter:brightness(1.08); }
  .err { color:#fca5a5; background:#3f1d1d; border:1px solid #7f1d1d; padding:8px 10px; border-radius:8px; }
  .brand { color:var(--green); font-family:ui-monospace,monospace; font-weight:700; margin-bottom:12px; }
</style></head>
<body>
<div class="card">
  <div class="brand">&lt;/&gt; eToolsCode</div>
  <h1>Configuração inicial</h1>
  <p class="sub">Defina a senha de acesso e a API key do modelo. Só aparece na primeira execução.</p>
  ` + errHTML + `
  <label>Senha da interface</label>
  <input id="password" type="password" minlength="6" placeholder="mín. 6 caracteres" autocomplete="new-password">
  <label>Confirmar senha</label>
  <input id="password2" type="password" minlength="6" autocomplete="new-password">
  <label>Provedor LLM</label>
  <select id="llm">
    <option value="1">DeepSeek</option>
    <option value="2">Meta (Llama API)</option>
    <option value="3">Gemini</option>
    <option value="6">Grok (xAI)</option>
  </select>
  <label>API Key</label>
  <input id="apikey" type="password" placeholder="sk-...">
  <button id="go">Concluir e entrar</button>
  <p id="msg" class="err" style="display:none;margin-top:12px"></p>
</div>
<script>
document.getElementById('go').onclick = async () => {
  const password = document.getElementById('password').value;
  const password2 = document.getElementById('password2').value;
  const llm_client = parseInt(document.getElementById('llm').value, 10);
  const api_key = document.getElementById('apikey').value.trim();
  const msg = document.getElementById('msg');
  if (password.length < 6) { msg.style.display='block'; msg.textContent='Senha muito curta'; return; }
  if (password !== password2) { msg.style.display='block'; msg.textContent='Senhas não conferem'; return; }
  if (!api_key) { msg.style.display='block'; msg.textContent='Informe a API key'; return; }
  const res = await fetch('/api/setup', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({password, llm_client, api_key})
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) { msg.style.display='block'; msg.textContent = data.error || res.statusText; return; }
  location.href = '/';
};
</script>
</body></html>`)
}
