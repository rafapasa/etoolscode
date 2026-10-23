# eToolsCode

**Assistente de engenharia de software da eTools Tecnologia.**

O eToolsCode roda **na sua máquina**, lê e altera o código **local**, e usa uma LLM (DeepSeek, Grok/xAI, etc.) para executar tarefas de forma controlada — por **chat** ou por **issues em JSON**.

Pense nele como um colega de time que:
- conhece as regras do projeto (`base.json` + `projeto.json`);
- só mexe nos arquivos que a tarefa permite;
- mostra o que fez (log + diff);
- pode seguir um fluxo de **issue automatizada** ou de **conversa interativa**.

> Interface web única · Chat · Issues · Diff · Fontes · Config  
> Versão: veja o arquivo `VERSION`.

---

## Sumário

1. [Para quem é](#1-para-quem-é)
2. [Conceitos em 5 minutos](#2-conceitos-em-5-minutos)
3. [Instalação](#3-instalação)
4. [Primeiro acesso (passo a passo)](#4-primeiro-acesso-passo-a-passo)
5. [Configurar projetos e LLM](#5-configurar-projetos-e-llm)
6. [Usar o Chat](#6-usar-o-chat)
7. [Usar Issues (JSON)](#7-usar-issues-json)
8. [Painéis da interface](#8-painéis-da-interface)
9. [Arquivos base.json e projeto.json](#9-arquivos-basejson-e-projetojson)
10. [Boas práticas para juniores](#10-boas-práticas-para-juniores)
11. [CLI e desenvolvimento](#11-cli-e-desenvolvimento)
12. [Problemas comuns](#12-problemas-comuns)

---

## 1. Para quem é

| Perfil | Como usa |
|--------|----------|
| **Júnior** | Chat para entender código; issues pequenas com lista de arquivos fechada |
| **Pleno / Sênior** | Issues cirúrgicas, branch por tarefa, review no Diff |
| **Tech lead** | `base.json` com padrões do time; vários projetos no Config |

Você **não precisa** ser expert em IA. Precisa saber:
- onde está o código do projeto no disco;
- o que quer mudar (em português claro);
- revisar o que a ferramenta alterou (aba Diff / git).

---

## 2. Conceitos em 5 minutos

### Chat vs Issue

| | **Chat** | **Issue** |
|--|----------|-----------|
| Entrada | Você digita na caixa de mensagem | Arquivo `.json` com demanda e regras |
| Uso típico | Explorar, tirar dúvida, ajuste pontual | Tarefa repetível, auditável, “pronta” |
| Controle | Conversa livre + tools | Lista de arquivos + rules + tarefas |

### Pasta de **código** vs pasta de **issues**

- **Root do projeto** — onde está o código-fonte (ex.: `/home/opc/prj/openerp-whatsapp`). Cadastrado em **Config → Pastas**.
- **Pasta de issues** — onde ficam os JSON de tarefas (ex.: `/home/opc/prj/issues/openerp-whatsapp/`). Criada automaticamente ao cadastrar o projeto.

### `base.json` e `projeto.json`

- **`base.json`** (na raiz da pasta de issues) — regras **globais** do time (ex.: usar `apperror`, não inventar DTO…).
- **`projeto.json`** (dentro da pasta do projeto nas issues) — dados **daquele** repositório (raiz, estrutura, dicas para a LLM).

### Tools (ferramentas da LLM)

A LLM não “mexe no disco sozinha”: ela pede ações como:
- `read_file` / `list_dir`
- `search_replace` (edição cirúrgica — preferir)
- `write_file` (arquivo novo ou reescrita completa)
- `run_command` (só quando a issue pedir)
- `done` (fim da tarefa)

No **chat**, essas ações vão para o **painel Log** no rodapé (não poluem a conversa).

---

## 3. Instalação

### Opção A — Binário (usuário final)

1. Baixe o release em [GitHub Releases](https://github.com/rafapasa/etoolscode/releases).
2. No Linux:

```bash
chmod +x etoolscode-linux-* install-linux.sh
./install-linux.sh ./etoolscode-linux-amd64   # ou arm64
```

3. Suba a UI:

```bash
etoolscode --ui
# ou
cd ~/.local/share/etoolscode && ./etoolscode --ui
```

4. Abra **http://127.0.0.1:8080**.

### Opção B — Do código-fonte (time eTools)

```bash
cd /caminho/para/etoolscode
go test ./...
go run ./cmd/etoolscode --ui
```

> Em produção na VM da eTools o serviço systemd chama-se `dsac` e a UI pode estar em um domínio interno.  
> **Não reinicie o serviço** em horário de uso sem alinhamento com o time.

### Windows

Use o script `scripts/install-windows.ps1` do repositório (PowerShell) e depois `etoolscode --ui`.

---

## 4. Primeiro acesso (passo a passo)

1. Abra a UI no navegador.
2. Se aparecer o **wizard de primeira execução**, defina:
   - senha de acesso à UI (se pedido);
   - provedor LLM e **API key**.
3. Clique em **Config** (canto superior).
4. Aba **Pastas**:
   - informe a **pasta raiz de issues** (ex.: `/home/opc/prj/issues`);
   - **adicione projetos**: nome + caminho do código (`root`).
5. Salve. O eToolsCode cria `issues/<nome-do-projeto>/` se ainda não existir.
6. No **combo de projeto** (topo), selecione o projeto.
7. Vá à aba **Chat**, clique em **+ novo**, e envie:  
   `Liste a estrutura de pastas do projeto e diga onde ficam os services.`

Se a resposta fizer sentido, a configuração básica está ok.

---

## 5. Configurar projetos e LLM

### Config → Pastas

| Campo | Significado |
|-------|-------------|
| Pasta de issues | Onde ficam `base.json` e subpastas por projeto |
| Lista de projetos | Nome amigável + **root** do código |

O **combo de projetos** da barra superior usa essa lista (não pasta soltas em disco, salvo fallback se a lista estiver vazia).

### Config → Modelos

Cadastre a API key do provedor (DeepSeek, Grok, etc.) e o modelo desejado.

Sem key válida o chat e as issues **não** conseguem chamar a LLM.

---

## 6. Usar o Chat

1. Selecione o **projeto** no combo.
2. Aba **Chat** → **+ novo** (ou abra um chat existente na lateral).
3. Escreva o pedido de forma **específica**:

**Bom**
> No arquivo `internal/webhook/processor.go`, explique a função `Process` e liste os pontos que dependem de tenant.

**Ruim**
> Arruma o webhook.

4. Acompanhe:
   - respostas da LLM no centro (bolhas);
   - leituras/escritas/erros no **Log** (rodapé, retrátil).
5. Se houver mudança em arquivo, use a aba **Diff** e a coluna **Alterados**.

### Dicas de chat

- Peça **um objetivo por mensagem** quando for editar código.
- Prefira: *“altere só a assinatura X e as 2 call sites”* em vez de *“refatore o service”*.
- Use a árvore **Fontes** (direita): clique no arquivo → aba **Fonte** com highlight.

---

## 7. Usar Issues (JSON)

Issue = tarefa **fechada** e **repetível**.

### Criar pela UI

1. Aba **Issues**.
2. **Nova issue** (ou upload de JSON).
3. Preencha os campos do formulário (eles mapeiam o schema):

| Campo | O que colocar |
|-------|----------------|
| Projeto | Nome cadastrado no Config |
| Arquivo JSON | Ex.: `01-extrair-dto.json` |
| Branch | Ex.: `dev-123` (opcional, recomendado) |
| Demanda | Objetivo em linguagem clara |
| Arquivos | Um path relativo por linha |
| Rules | Regras extras (uma por linha) |
| Tarefas | `arquivo \| tipo \| descrição` |

Também pode colar um **JSON completo** só no campo Demanda (se começar com `{`).

### Schema resumido

```json
{
  "demanda": "O que fazer, em uma frase clara.",
  "branch": "dev-123",
  "arquivos": ["internal/foo/bar.go"],
  "rules": ["Não alterar assinatura pública de X"],
  "tarefas": [
    {
      "id": "1",
      "arquivo": "internal/foo/bar.go",
      "tipo": "edit",
      "descricao": "Passo concreto e verificável."
    }
  ]
}
```

### Executar

1. No explorer de issues, selecione o JSON.
2. **▶ Executar** (ou menu ⋮ → Executar).
3. Acompanhe o log no painel central / Log do rodapé.
4. Ao terminar, revise o **Diff** e rode os testes do projeto se a issue não tiver pedido isso.

### `search_replace` vs `write_file`

- Mudança **local** → a LLM deve preferir `search_replace`.
- Arquivo **novo** ou reescrita inevitável → `write_file`.
- Arquivos grandes: quebre a issue; evite reescrever o arquivo inteiro.

---

## 8. Painéis da interface

| Área | Função |
|------|--------|
| **Combo projeto** | Filtra chats, issues e fontes |
| **Issues** | Lista JSON, executa, edita |
| **Chat** | Conversas (várias ao mesmo tempo) |
| **Diff** | Diff Monaco (HEAD × working tree) |
| **Fonte** | Conteúdo do arquivo clicado em Fontes |
| **Fontes / Alterados** (direita) | Árvore do código e arquivos modificados no git |
| **Log** (rodapé) | Leituras, escritas, comandos e erros |
| **Config** | Pastas + modelos LLM |
| **Engenharia** (branch `dev`) | Assistente de plano + scaffold TDD |

---

## 9. Arquivos base.json e projeto.json

### base.json (global)

Fica em `{pasta_de_issues}/base.json`.

Exemplo de regras:

```json
{
  "rules": [
    "Erros SEMPRE via apperror. Nunca errors.New.",
    "Prefira search_replace a write_file para edicoes locais."
  ]
}
```

### projeto.json (por projeto)

Fica em `{pasta_de_issues}/{nome-projeto}/projeto.json`.

Costuma trazer `raiz`, estrutura e dicas. Ajuda a LLM a ser mais assertiva.

Documentação detalhada dos schemas: `docs/SCHEMAS-JSON.md` (quando presente no repo).

---

## 10. Boas práticas para juniores

1. **Sempre** selecione o projeto certo no combo antes de chat/issue.
2. **Sempre** revise o Diff antes de considerar a tarefa pronta.
3. Issues **pequenas** > uma issue gigante.
4. Na demanda, diga o **resultado esperado** e o que **não** pode mudar.
5. Não peça “refatore tudo”; peça o trecho e o critério de aceite.
6. Se o log mostrar erro de path (`caminho fora da raiz`), confira o root no Config e o `projeto.json`.
7. Em dúvida, use o chat só para **ler e explicar**; edite em issue com arquivos listados.

---

## 11. CLI e desenvolvimento

```bash
# UI
etoolscode --ui

# Hash de senha (se necessário)
etoolscode -hash-password 'sua-senha'

# Testes do próprio eToolsCode
go test ./...
```

Branches do repositório:
- **`main`** — estável / produção alinhada ao serviço
- **`dev`** — evolução (ex.: Assistente de Engenharia)

Documentos úteis:
- `docs/ENGINEERING-ASSISTANT.md` — fluxo de plano + scaffold TDD
- `docs/SCHEMAS-JSON.md` — schemas de issue / base / projeto

---

## 12. Problemas comuns

| Sintoma | O que checar |
|---------|----------------|
| Combo de projetos vazio | Config → Pastas: cadastrou projetos? |
| Chat não responde | API key / modelo em Config → Modelos |
| `caminho fora da raiz` | Root do projeto no Config; não usar path absoluto errado |
| Issue termina sem editar | Demanda ambígua ou arquivos não listados |
| Log com “Falhou” | Abra o detalhe no rodapé; leia a mensagem completa |
| Diff vazio | Não há mudança no git working tree |
| UI não reflete código novo | Serviço não foi reiniciado (normal em horário de uso) |

---

## Segurança e privacidade

- O código é lido/escrito **localmente** no ambiente onde o binário roda.
- A LLM recebe trechos enviados nas tools (prompts + conteúdos lidos) — use provedores e políticas alinhados à empresa.
- A UI pode exigir senha (hash em configuração local).

---

## Licença e contato

Produto **eTools Tecnologia**.  
Repositório: [github.com/rafapasa/etoolscode](https://github.com/rafapasa/etoolscode)

Dúvidas de uso interno: time eTools / canal de suporte da empresa.

---

*Este README é orientado a quem está começando. Se algo divergir da UI que você está vendo, confira o branch (`main` vs `dev`) e a versão em `VERSION`.*
