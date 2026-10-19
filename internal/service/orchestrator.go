package service

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/core/tools"
	"github.com/rafapasa/etoolscode/internal/dto"
)

const (
	MaxToolResultBytes = 100 * 1024 // 100 KB max por resposta de tool para proteger o contexto
	MaxRetriesPerTurn  = 3          // Tentativas em caso de erro de API/Rede
)

type Orchestrator struct {
	client core.LlmInterface
	cfg    *config.Config
}

func NewOrchestrator(llm core.LlmInterface) *Orchestrator {
	return &Orchestrator{client: llm, cfg: config.NewConfig()}
}

func NewOrchestratorWithConfig(llm core.LlmInterface, cfg *config.Config) *Orchestrator {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	return &Orchestrator{client: llm, cfg: cfg}
}

func (o *Orchestrator) Start(req dto.Request) error {
	fmt.Println("[AutoCode] Iniciando")
	fmt.Printf("[AutoCode] Raiz: %s\n", req.Raiz)
	fmt.Printf("[AutoCode] Demanda: %s\n\n", req.Demanda)

	if o.cfg != nil && o.cfg.GitAutoBranch {
		project := req.Project
		if project == "" {
			project = filepath.Base(strings.TrimRight(req.Raiz, "/"))
		}
		issueID := req.Issue
		if issueID == "" {
			issueID = req.Demanda
			if len(issueID) > 40 {
				issueID = issueID[:40]
			}
		}
		if _, branch, err := tools.EnsureBranchNamed(req.Raiz, req.Branch, project, issueID); err != nil {
			fmt.Printf("[AutoCode] aviso git branch: %v\n", err)
			_ = NewAuditService().Log(AuditEntry{Project: project, IssueID: issueID, Tool: "git_branch", Success: false, Error: err.Error()})
		} else if branch != "" {
			fmt.Printf("[AutoCode] branch: %s\n", branch)
			_ = NewAuditService().Log(AuditEntry{Project: project, IssueID: issueID, Tool: "git_branch", Path: branch, Success: true})
		}
	}

	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(req)

	messages := []dto.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	tools := core.BuildTools()
	executor := &core.ToolExecutor{Raiz: req.Raiz}

	maxIterations := 60
	totalTokens := 0

	for i := 0; i < maxIterations; i++ {
		fmt.Printf("\n[AutoCode] --- Turno %d ---\n", i+1)

		// 1. Retry resiliente no Chat
		resp, err := o.chatWithRetry(messages, tools)
		if err != nil {
			return fmt.Errorf("falha irrecuperável na API no turno %d: %v", i+1, err)
		}

		// 2. Validação contra Panic
		if resp == nil || len(resp.Choices) == 0 {
			fmt.Println("[AutoCode] ERRO: Resposta da API veio sem escolhas (choices vazias). Tentando prosseguir...")
			messages = append(messages, dto.Message{
				Role:    "user",
				Content: "Sua última resposta veio vazia. Por favor, continue a tarefa ou forneça uma resposta.",
			})
			continue
		}

		choice := resp.Choices[0]
		totalTokens += resp.Usage.TotalTokens
		fmt.Printf("[LLM] tokens: %d (acumulado: %d) | finish: %s\n",
			resp.Usage.TotalTokens, totalTokens, choice.FinishReason)

		msg := choice.Message
		// Garante IDs nas tool_calls (alguns providers omitem quando truncam)
		for ti := range msg.ToolCalls {
			if msg.ToolCalls[ti].ID == "" {
				msg.ToolCalls[ti].ID = fmt.Sprintf("call_%d_%d", i+1, ti+1)
			}
			if msg.ToolCalls[ti].Type == "" {
				msg.ToolCalls[ti].Type = "function"
			}
		}
		messages = append(messages, msg)

		// Sem tool_calls: resposta final (ou truncada de texto)
		if len(msg.ToolCalls) == 0 {
			if choice.FinishReason == "length" {
				fmt.Println("[AutoCode] AVISO: texto truncado (max_tokens). Solicitando continuidade...")
				messages = append(messages, dto.Message{
					Role:    "user",
					Content: "Sua resposta anterior foi truncada. Continue de onde parou, de forma concisa. Se a tarefa terminou, chame done.",
				})
				continue
			}
			fmt.Printf("\n[LLM] Resposta final: %s\n", msg.Content)
			break
		}

		// IMPORTANTE: após assistant com tool_calls, a próxima mensagem DEVE ser
		// role=tool para cada tool_call_id. Nunca inserir user no meio.
		truncatedTools := choice.FinishReason == "length"
		if truncatedTools {
			fmt.Println("[AutoCode] AVISO: tool_calls possivelmente truncadas (max_tokens). Executando o que veio e pedindo reenvio se necessário...")
		}

		for ti, call := range msg.ToolCalls {
			fmt.Printf("[Tool] %s(%s)\n", call.Function.Name, truncate(call.Function.Arguments, 120))

			if call.Function.Name == "done" {
				var p struct {
					Summary string `json:"summary"`
				}
				_ = json.Unmarshal([]byte(call.Function.Arguments), &p)
				fmt.Printf("\n[AutoCode] CONCLUÍDO: %s\n", p.Summary)
				fmt.Printf("[AutoCode] Tokens totais: %d\n", totalTokens)
				return nil
			}

			startTool := time.Now()
			result, err := executor.Execute(call)
			// Argumentos truncados costumam falhar no Unmarshal JSON
			if err != nil && truncatedTools {
				err = fmt.Errorf("%w (resposta LLM truncada por max_tokens — reenvie write_file completo ou divida a edição)", err)
			}
			toolPath := ""
			if call.Function.Name == "write_file" || call.Function.Name == "search_replace" || call.Function.Name == "read_file" || call.Function.Name == "list_dir" {
				var arg struct {
					Path string `json:"path"`
				}
				_ = json.Unmarshal([]byte(call.Function.Arguments), &arg)
				toolPath = arg.Path
			}
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			_ = NewAuditService().Log(AuditEntry{
				Project:    req.Project,
				IssueID:    req.Issue,
				Tool:       call.Function.Name,
				Path:       toolPath,
				Success:    err == nil,
				Error:      errStr,
				DurationMs: time.Since(startTool).Milliseconds(),
			})
			if err != nil {
				result = fmt.Sprintf("ERRO NA FERRAMENTA: %v", err)
				fmt.Printf("[Tool] erro: %v\n", err)
			} else {
				fmt.Printf("[Tool] ok (%d bytes)\n", len(result))
			}

			if len(result) > MaxToolResultBytes {
				fmt.Printf("[Tool] AVISO: resultado da tool truncado de %d bytes para %d bytes\n", len(result), MaxToolResultBytes)
				result = result[:MaxToolResultBytes] + "\n... [CONTEÚDO TRUNCADO DEVIDO AO TAMANHO EXCESSIVO]"
			}

			messages = append(messages, dto.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    result,
			})
			_ = ti
		}

		// Só depois de responder TODAS as tool_calls podemos pedir continuidade
		if truncatedTools {
			messages = append(messages, dto.Message{
				Role: "user",
				Content: "Uma ou mais tool_calls anteriores foram truncadas por limite de tokens. " +
					"Se algum write_file falhou, reenvie-o completo em partes menores (ou só o trecho necessário). " +
					"Não repita read_file desnecessário. Continue a tarefa.",
			})
		}
	}

	fmt.Printf("\n[AutoCode] Atingiu maxIterations (%d). Tokens: %d\n", maxIterations, totalTokens)
	return nil
}

// chatWithRetry realiza tentativas com pausa progressiva em caso de falhas da API
func (o *Orchestrator) chatWithRetry(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= MaxRetriesPerTurn; attempt++ {
		resp, err := o.client.Chat(messages, tools)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		fmt.Printf("[AutoCode] AVISO: erro na chamada LLM (tentativa %d/%d): %v\n", attempt, MaxRetriesPerTurn, err)

		if attempt < MaxRetriesPerTurn {
			waitTime := time.Duration(attempt*2) * time.Second
			time.Sleep(waitTime)
		}
	}
	return nil, lastErr
}

// ============================================================
// PROMPTS
// ============================================================

func buildSystemPrompt() string {
	return `Você é um engenheiro de software sênior executando uma tarefa cirúrgica em um projeto Go.

Você tem tools disponíveis:
- read_file: lê o conteúdo de um arquivo
- search_replace: edição cirúrgica (old_string → new_string). PREFIRA esta para mudanças locais.
- write_file: cria arquivo novo ou reescreve inteiro (evite em arquivos grandes)
- list_dir: lista diretório (use só se absolutamente necessário)
- run_command: NÃO USE. Não é sua função.
- done: sinaliza conclusão

REGRAS CRÍTICAS:
- Leia APENAS os arquivos listados em "ARQUIVOS PERMITIDOS". Nada mais.
- NUNCA leia arquivos fora da lista. Se a tarefa menciona um arquivo, ele está na lista.
- NUNCA rode comandos (go build, go test, go fmt). Não é sua função.
- NUNCA explore diretórios por curiosidade. Vá direto ao ponto.
- NUNCA re-leia um arquivo que você já leu nesta sessão.
- Execute as tarefas na ordem apresentada (id 1, 2, 3, ...).
- Para cada tarefa, edite o arquivo indicado seguindo a descrição ao pé da letra.
- Prefira search_replace a write_file sempre que a mudança for local (assinatura, trecho, import).
- Se a descrição tem blocos de código, copie-os fielmente (assinaturas, WHERE, nomes).
- Se a descrição diz "REMOVER: X, Y", remova exatamente X e Y.
- Se a descrição diz "NÃO TOCAR: Z", não toque em Z.
- Quando todas as tarefas terminarem, chame done com um resumo do que foi feito.
- Se encontrar ambiguidade, escolha a interpretação mais literal da descrição.`
}

func buildUserPrompt(req dto.Request) string {
	var b strings.Builder

	b.WriteString("DEMANDA:\n")
	b.WriteString(req.Demanda)
	b.WriteString("\n\n")

	b.WriteString("RAIZ DO PROJETO:\n")
	b.WriteString(req.Raiz)
	b.WriteString("\n\n")

	if req.Estrutura != nil {
		estruturaStr, _ := json.MarshalIndent(req.Estrutura, "", "  ")
		b.WriteString("ESTRUTURA DO PROJETO:\n")
		b.Write(estruturaStr)
		b.WriteString("\n\n")
	}

	b.WriteString("ARQUIVOS PERMITIDOS (só estes podem ser lidos e editados):\n")
	for _, a := range req.Arquivos {
		b.WriteString("- ")
		b.WriteString(a)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("TAREFAS (execute na ordem):\n\n")
	for _, t := range req.Tarefas {
		fmt.Fprintf(&b, "========================================\n")
		fmt.Fprintf(&b, "TAREFA %s — [%s] %s\n", t.ID, t.Tipo, t.Arquivo)
		fmt.Fprintf(&b, "========================================\n")
		b.WriteString(t.Descricao)
		b.WriteString("\n\n")
	}

	b.WriteString("REGRAS:\n")
	for i, r := range req.Rules {
		fmt.Fprintf(&b, "%d. %s\n", i+1, r)
	}
	b.WriteString("\n")
	b.WriteString("Comece lendo os arquivos listados em ordem. Não explore outros. Não rode comandos. Execute as tarefas e chame done ao terminar.")

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
