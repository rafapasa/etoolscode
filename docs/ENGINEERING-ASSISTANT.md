# Assistente de Engenharia (etoolsCode)

Fluxo multilevel:

1. **Brief** — estilo arquitetural + feature + descrição
2. **Plano** — pastas, interfaces, testes (revisável)
3. **Scaffold TDD** — grava interface + testes vermelhos + stub TODO
4. **Implementação** — humana e/ou issue/chat até `go test` passar

## Estilos

| ID | Nome |
|----|------|
| `layers` | Camadas |
| `hexagonal` | Ports & Adapters |
| `ddd_light` | DDD tático leve |
| `modular_monolith` | Monólito modular |

## API

- `GET /api/engineering/styles`
- `POST /api/engineering/plan` body: Brief → Plan
- `POST /api/engineering/scaffold` body: `{ "project", "plan", "skip_existing", "create_issue" }`

## Observação

O serviço em produção **não** é reiniciado automaticamente neste branch; use o binário/dev quando for validar a UI.
