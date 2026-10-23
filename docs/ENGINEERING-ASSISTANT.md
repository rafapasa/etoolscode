# Assistente de Engenharia (etoolsCode)

Fluxo multilevel:

1. **Detectar estilo** (opcional) — inspeciona `internal/` do projeto
2. **Brief** — estilo + feature + descrição
3. **Plano** — pastas, interfaces, métodos (template; opcional enriquecimento LLM)
4. **Preview** — mostra arquivos sem gravar
5. **Scaffold TDD** — grava interface + testes vermelhos + stub TODO (+ issue opcional)
6. **Implementação** — humana e/ou issue/chat até `go test` passar

## Estilos

| ID | Nome |
|----|------|
| `layers` | Camadas |
| `hexagonal` | Ports & Adapters |
| `ddd_light` | DDD tático leve |
| `modular_monolith` | Monólito modular |

## API

- `GET /api/engineering/styles`
- `GET /api/engineering/detect?project=`
- `POST /api/engineering/plan` body: Brief + `enrich_llm`
- `POST /api/engineering/preview` body: `{ "plan" }`
- `POST /api/engineering/scaffold` body: `{ "project", "plan", "skip_existing", "create_issue" }`

## UI

Botão **Engenharia** no header da interface web.
