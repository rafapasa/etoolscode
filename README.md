# etoolscode

Assistente de código da eTools Tecnologia: executa issues JSON e chat com tools sobre repositórios Go (e afins).

## Requisitos

- Go 1.26+
- Git
- systemd (deploy VM)

## Build e restart (VM)

```bash
cd /home/opc/prj/etoolscode
./restart-all.sh          # ou: make restart
```

Binário: `/usr/local/bin/etoolscode`  
Compat: symlink `/usr/local/bin/deepseek-autocode` → `etoolscode`  
Serviço: `dsac.service`

## UI

- Clássica: `https://dsac.etoolstec.com.br/`
- Pro (diff/audit): `https://dsac.etoolstec.com.br/pro`

Senha obrigatória via env (`DSAC_UI_PASSWORD_HASH` ou equivalente — ver `.env.example`).

## CLI

```bash
etoolscode /home/opc/prj/issues/<projeto>/<issue>.json
```

Antes de executar (se `DSAC_GIT_AUTO_BRANCH` ≠ `false`):

- cria/checkout `dsac/<projeto>/<issue>`
- grava audit em `~/.ds-ac/audit.jsonl`

## API útil

| Endpoint | Descrição |
|----------|-----------|
| `GET /api/projects` | lista projetos em ISSUES_DIR |
| `GET /api/tree/:project` | árvore de fontes (raiz do projeto.json) |
| `GET /api/git/status?project=` | arquivos modificados |
| `GET /api/git/file?project=&file=` | conteúdo HEAD × working tree |
| `GET /api/git/diff?project=&file=` | unified diff |
| `POST /api/git/branch` | `{ "name", "project" }` |
| `GET /api/audit/recent?project=&limit=` | últimas ações |

## Variáveis

Veja `.env.example`.

## Desenvolvimento

```bash
make test
make build
go run ./cmd/etoolscode --ui   # não exige API key
```

## Versão

Ver `VERSION`.
