# Schemas JSON — eToolsCode (issues)

Documentação dos três artefatos JSON usados pelo etoolsCode na execução de **issues**.

## Layout em disco

```text
ISSUES_DIR/                      # ex.: /home/opc/prj/issues
├── base.json                    # regras globais (todos os projetos)
├── <nome-projeto>/
│   ├── projeto.json             # raiz do código + árvore de pastas
│   ├── 01-minha-issue.json      # issue executável
│   └── concluidas/              # issues já processadas (opcional)
```

Na execução, o runtime **mescla** mentalmente:

1. `base.json` → `rules` globais  
2. `projeto.json` → `raiz` + `estrutura` (+ rules do projeto, se houver)  
3. `issue.json` → `demanda` + `arquivos` + `tarefas` (+ rules extras da issue)

O DTO final (`internal/dto.Request`) fica assim:

```go
type Request struct {
    Project   string      // preenchido pelo path (pasta do projeto)
    Issue     string      // nome do arquivo da issue
    Demanda   string
    Estrutura interface{} // árvore do projeto.json
    Raiz      string      // path absoluto do código
    Rules     []string    // base + projeto + issue
    Arquivos  []string    // lista fechada de paths relativos
    Tarefas   []Tarefa
}

type Tarefa struct {
    ID        string
    Arquivo   string // path relativo dentro de Raiz
    Tipo      string // ver tabela abaixo
    Descricao string
}
```

---

## 1. `base.json` (global)

**Caminho:** `ISSUES_DIR/base.json`  
**Função:** contrato de engenharia válido para **todos** os projetos. Evita repetir as mesmas regras em cada issue.

### Schema

```json
{
  "rules": [
    "string — regra em linguagem natural ou norma de código"
  ]
}
```

| Campo   | Tipo       | Obrigatório | Descrição |
|---------|------------|-------------|-----------|
| `rules` | `string[]` | sim*        | Lista de regras globais (apperror, mapper, response, “não rodar testes”, etc.) |

\*Pode existir arquivo vazio, mas na prática sempre deve ter regras.

### Exemplo mínimo

```json
{
  "rules": [
    "Erros SEMPRE via apperror. Nunca errors.New.",
    "Conversoes DTO<->Model SEMPRE via gokit/mapper.",
    "Respostas HTTP SEMPRE via gokit/response. Nunca c.JSON direto.",
    "NAO buildar. NAO rodar testes. NAO rodar go fmt. Apenas editar os arquivos listados.",
    "Sempre verificar nomes para evitar redeclarar func, structs e variaveis globais."
  ]
}
```

### Boas práticas

- Regras **imperativas** e curtas.  
- O que vale para qualquer repo Go da eTools fica aqui.  
- Não colocar paths de um projeto específico.

---

## 2. `projeto.json` (por projeto)

**Caminho:** `ISSUES_DIR/<projeto>/projeto.json`  
**Função:** localizar o código-fonte e dar à LLM a **estrutura de pastas/arquivos** sem varrer o disco.

### Schema

```json
{
  "raiz": "/caminho/absoluto/do/repositorio",
  "estrutura": [
    {
      "type": "directory",
      "name": ".",
      "contents": [
        {
          "type": "directory",
          "name": "internal",
          "contents": [
            { "type": "file", "name": "service.go" }
          ]
        },
        { "type": "file", "name": "go.mod" }
      ]
    }
  ],
  "rules": [
    "opcional — regras só deste projeto"
  ]
}
```

| Campo        | Tipo     | Obrigatório | Descrição |
|--------------|----------|-------------|-----------|
| `raiz`       | `string` | **sim**     | Path absoluto (ou resolvível) da pasta do código |
| `estrutura`  | `array`  | recomendado | Árvore estilo `tree` (nodes `directory` / `file`) |
| `rules`      | `string[]` | não       | Regras extras só deste projeto |

### Node de `estrutura`

| Campo      | Tipo     | Descrição |
|------------|----------|-----------|
| `type`     | `string` | `"directory"` ou `"file"` |
| `name`     | `string` | Nome do arquivo ou pasta |
| `contents` | `array`  | Só em `directory`: filhos (mesmo schema) |

### Exemplo mínimo

```json
{
  "raiz": "/home/opc/prj/openerp-whatsapp",
  "estrutura": [
    {
      "type": "directory",
      "name": ".",
      "contents": [
        {
          "type": "directory",
          "name": "internal",
          "contents": [
            { "type": "file", "name": "webhook.go" },
            { "type": "file", "name": "webhook_test.go" }
          ]
        },
        { "type": "file", "name": "go.mod" }
      ]
    }
  ]
}
```

### Resolução da raiz no etoolsCode

Ordem aproximada:

1. Cadastro em **Config → Pastas** (`workspace.json`)  
2. Campo `raiz` do `projeto.json`  
3. Fallback `projects_root/<nome>` ou sibling de `ISSUES_DIR`

---

## 3. Issue JSON (por tarefa)

**Caminho:** `ISSUES_DIR/<projeto>/<nome-issue>.json`  
**Função:** uma unidade de trabalho executável (CLI ou botão Executar na UI).

### Schema canônico

```json
{
  "demanda": "objetivo curto e direto",
  "raiz": "/opcional/se-nao-vier-do-projeto.json",
  "estrutura": [],
  "rules": [],
  "regras_gerais": [],
  "arquivos": [
    "path/relativo/arquivo.go",
    "path/relativo/arquivo_test.go"
  ],
  "tarefas": [
    {
      "id": "1",
      "arquivo": "path/relativo/arquivo_test.go",
      "tipo": "test",
      "descricao": "o que fazer neste arquivo — idealmente com assinaturas em bloco de código"
    }
  ]
}
```

| Campo           | Tipo       | Obrigatório | Descrição |
|-----------------|------------|-------------|-----------|
| `demanda`       | `string`   | **sim**     | Objetivo da issue (uma frase clara) |
| `arquivos`      | `string[]` | **sim**     | Lista **fechada** de paths relativos que a LLM pode ler/escrever |
| `tarefas`       | `object[]` | **sim**     | Uma entrada por passo (em geral 1 arquivo por tarefa) |
| `rules`         | `string[]` | não         | Regras extras desta issue (somam às do base) |
| `regras_gerais` | `string[]` | não         | Alias usado em issues legadas (mesmo papel de `rules`) |
| `raiz`          | `string`   | não         | Só se não houver `projeto.json` confiável |
| `estrutura`     | `array`    | não         | Raramente na issue; preferir no `projeto.json` |

### Schema de cada `tarefa`

| Campo       | Tipo     | Obrigatório | Descrição |
|-------------|----------|-------------|-----------|
| `id`        | `string` | recomendado | Identificador estável (`"1"`, `"2"`, …) |
| `arquivo`   | `string` | **sim**     | Path relativo (deve estar em `arquivos`) |
| `tipo`      | `string` | recomendado | Ver tabela abaixo |
| `descricao` | `string` | **sim**     | Instrução; **assinaturas e WHERE em bloco de código**, não em prosa |

### Valores usuais de `tipo`

| `tipo`     | Uso |
|------------|-----|
| `test`     | Criar/estender testes |
| `edit`     | Alterar implementação existente |
| `create`   | Criar arquivo novo |
| `refactor` | Refatorar sem mudar comportamento (quando especificado) |
| `fix`     | Correção de bug pontual |

(Outros valores são aceitos; a LLM usa como dica de intenção.)

### Exemplo real (estilo produção)

```json
{
  "demanda": "[TEST] UnifiedLLM.TranscribeAudio e DescribeImage com clients fake. Sem API externa.",
  "arquivos": [
    "internal/llm/provider.go",
    "internal/llm/media_test.go"
  ],
  "regras_gerais": [
    "nao chamar API externa"
  ],
  "tarefas": [
    {
      "id": "1",
      "arquivo": "internal/llm/media_test.go",
      "tipo": "test",
      "descricao": "TranscribeAudio client nil: erro 'audio LLM nao configurado'. Client fake: devolve texto."
    },
    {
      "id": "2",
      "arquivo": "internal/llm/media_test.go",
      "tipo": "test",
      "descricao": "DescribeImage client nil: erro. Fake: devolve descricao."
    }
  ]
}
```

---

## 4. Regras de ouro ao escrever issues

1. **Código > prosa** — assinaturas, WHERE, structs em bloco compilável.  
2. **`arquivos` é lista fechada** — o que não está nela não deve ser editado.  
3. **`demanda` sem “faça o que for necessário”** — escopo explícito.  
4. **Uma preocupação por issue** — issues grandes viram várias menores.  
5. **`projeto.json` atualizado** — estrutura desatualizada faz a LLM inventar paths.  
6. **Não pedir build/test no JSON** se a regra global proíbe; rode testes fora, no CI/local.

---

## 5. Índices (opcional)

Arquivos como `00-indice-*.json` **não** são executados como issue padrão; servem de mapa humano/LLM para um conjunto de issues. Campos livres (`issues[]`, `bug_corrigido`, etc.).

---

## 6. Execução

```bash
# CLI
etoolscode /home/opc/prj/issues/<projeto>/<issue>.json

# UI
# aba Issues → selecionar JSON → Executar
```

Com `DSAC_GIT_AUTO_BRANCH` (default true), antes de rodar: branch `dsac/<projeto>/<issue>`.

---

*Documento gerado para o repositório etoolscode — schemas alinhados a `internal/dto/issue.go` e ao uso real em `/home/opc/prj/issues`.*
