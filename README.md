# eToolsCode

Assistente de código da **eTools Tecnologia**: roda issues em JSON e chat com tools sobre o código local (Go e afins).

Interface web única (Chat, Issues, Diff, Fontes) + CLI. Roda na máquina do usuário; os arquivos não saem do ambiente dele.

**Versão:** veja o arquivo `VERSION` (atual: 1.0.0).

---

## O que faz

- **Issues** — executa tarefas definidas em JSON (com `base.json` global e `projeto.json` por projeto)
- **Chat** — conversa com a LLM, leitura/escrita de arquivos, git e comandos seguros
- **Diff** — visualiza alterações (HEAD × working tree) com editor Monaco
- **Config** — pastas de projetos/issues e provedor LLM + API keys

---

## Instalação rápida (usuário final)

### Linux

1. Baixe o binário do [Release](https://github.com/rafapasa/etoolscode/releases) (`etoolscode-linux-amd64` ou `arm64`).
2. Baixe também `scripts/install-linux.sh` (ou clone o repositório).
3. Execute:

```bash
chmod +x etoolscode-linux-amd64 install-linux.sh
./install-linux.sh ./etoolscode-linux-amd64
```

O script:

- instala o binário em `~/.local/share/etoolscode` e link em `~/.local/bin`
- pede a **senha da UI** e grava o hash
- cria `~/.ds-ac/` e pasta de dados para issues

Depois:

```bash
# opcional: edite API keys
nano ~/.local/share/etoolscode/.env

# inicie a UI
cd ~/.local/share/etoolscode && ./etoolscode --ui
# ou, se ~/.local/bin estiver no PATH:
etoolscode --ui
```

Abra **http://127.0.0.1:8080**. Na primeira execução sem senha configurada, o **wizard** pede senha + API key do modelo.

### Windows

1. Baixe `etoolscode-windows-amd64.exe` do Release.
2. Baixe `scripts/install-windows.ps1`.
3. No PowerShell:

```powershell
.\install-windows.ps1 -BinaryPath .\etoolscode-windows-amd64.exe
```

O script pede a senha, cria `%LOCALAPPDATA%\etoolscode` e o arquivo `.env`.

```powershell
& "$env:LOCALAPPDATA\etoolscode\etoolscode.exe" --ui
```

Abra **http://127.0.0.1:8080**.

---

## Docker

Imagem construída localmente (não há publicação automática no Docker Hub neste repositório):

```bash
git clone https://github.com/rafapasa/etoolscode.git
cd etoolscode
docker compose up -d --build
```

Variáveis úteis no `.env` ou no `docker-compose.yml`:

| Variável | Descrição |
|----------|-----------|
| `DEEPSEEK_API_KEY` / `META_API_KEY` / `GEMINI_API_KEY` | Chaves LLM |
| `LLM_CLIENT` | `1` DeepSeek, `2` Meta, `3` Gemini |
| `DSAC_UI_PASSWORD_HASH` | Hash bcrypt da senha da UI |
| `DSAC_UI_AUTH=off` | Desativa login (só para lab) |
| `ISSUES_DIR` | Pasta de issues (default `/data/issues` no container) |

Para publicar no Docker Hub (manual):

```bash
docker build -t SEU_USUARIO/etoolscode:1.0.0 .
docker push SEU_USUARIO/etoolscode:1.0.0
```

---

## Configuração

| Onde | O quê |
|------|--------|
| **⚙ Config → Pastas** | Pasta de issues, raiz de projetos, cadastro nome→path |
| **⚙ Config → Modelos** | Provedor ativo e API keys |
| `~/.ds-ac/config.json` | Persistência de keys/provider (UI) |
| `~/.ds-ac/workspace.json` | Pastas e projetos |
| `~/.ds-ac/auth.json` | Hash da senha (wizard / instalador) |
| `.env` | Overrides de ambiente (ver `.env.example`) |

Gere hash de senha:

```bash
etoolscode --hash-password 'sua-senha'
```

---

## Schemas das issues

Documentação completa de `base.json`, `projeto.json` e issue JSON:

→ [`docs/SCHEMAS-JSON.md`](docs/SCHEMAS-JSON.md)

## CLI (issues)

```bash
etoolscode /caminho/para/issues/<projeto>/<issue>.json
```

Com git automático (default): cria/checkout `dsac/<projeto>/<issue>` e registra audit em `~/.ds-ac/audit.jsonl`.

Desligar branch automática: `DSAC_GIT_AUTO_BRANCH=false`.

---

## Desenvolvimento

```bash
git clone https://github.com/rafapasa/etoolscode.git
cd etoolscode
make test
make build
go run ./cmd/etoolscode --ui
```

Deploy na VM eTools:

```bash
cd /home/opc/prj
./restart-all.sh
```

---

## Release (maintainers)

O workflow GitHub Actions (`.github/workflows/release.yml`) gera binários multiplataforma quando uma tag `v*` é publicada:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Artefatos: Linux/Windows/macOS (amd64 e arm64) na página de Releases.

---

## Licença de uso futuro

Na subida, o binário pode consultar `POST https://dsac.etoolstec.com.br/auth` com `{"etoolscodekey":"..."}`.  
Só bloqueia se a resposta for **HTTP 200** e a mensagem contiver **acesso negado**. Qualquer outro resultado (incluindo endpoint inexistente) **não bloqueia** — preparado para monetização futura, sem impacto hoje.

Desligar a checagem: `ETOOLSCODE_LICENSE_CHECK=off`.
