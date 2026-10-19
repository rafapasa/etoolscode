package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/core/tools"
	"github.com/rafapasa/etoolscode/internal/dto"
	"github.com/rafapasa/etoolscode/internal/service"
)

func GetTools() []dto.Tool {
	return []dto.Tool{
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "list_files", Description: "Lista arquivos de um diretório do projeto. Use '.' para raiz",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"dir": map[string]interface{}{"type": "string"}}, "required": []string{"dir"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "read_file", Description: "Lê conteúdo de um arquivo dentro do projeto",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}}, "required": []string{"path"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "write_file", Description: "Cria ou sobrescreve arquivo completo. Prefira search_replace para mudanças locais.",
				Parameters: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}}, "required": []string{"path", "content"}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name: "search_replace",
				Description: "Substitui old_string por new_string no arquivo. Preferir para edição cirúrgica. replace_all opcional.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path":        map[string]interface{}{"type": "string"},
						"old_string":  map[string]interface{}{"type": "string"},
						"new_string":  map[string]interface{}{"type": "string"},
						"replace_all": map[string]interface{}{"type": "boolean"},
					},
					"required": []string{"path", "old_string", "new_string"},
				},
			},
		},
		core.RunCommandTool(),
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "git_status",
				Description: "Lista arquivos modificados no repositório git do projeto (working tree).",
				Parameters:  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "git_diff",
				Description: "Mostra diff git. Opcional: path de um arquivo.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"path": map[string]interface{}{"type": "string", "description": "Arquivo relativo (opcional)"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "git_branch",
				Description: "Cria e faz checkout de uma branch (ex: dsac/projeto/feature-x). Use quando o usuário pedir branch ou antes de mudanças grandes.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string", "description": "Nome da branch"},
					},
					"required": []string{"name"},
				},
			},
		},
		{
			Type: "function",
			Function: dto.ToolFunction{
				Name:        "git_checkout",
				Description: "Troca para uma branch existente (ex: main).",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
					"required": []string{"name"},
				},
			},
		},
	}
}

type ToolResult struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

func ExecuteTool(projectRoot, toolName, argsJSON string) ToolResult {
	var args map[string]string
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ToolResult{Success: false, Error: fmt.Sprintf("args inválidos: %v", err)}
	}
	if len(args) == 0 {
		var generic map[string]interface{}
		if err := json.Unmarshal([]byte(argsJSON), &generic); err != nil {
			return ToolResult{Success: false, Error: fmt.Sprintf("args inválidos: %v", err)}
		}
		args = make(map[string]string)
		for k, v := range generic {
			if s, ok := v.(string); ok {
				args[k] = s
			} else {
				b, _ := json.Marshal(v)
				args[k] = string(b)
			}
		}
	}

	// Confina à raiz permitida (DSAC_SHELL_ROOT, padrão ~/prj), resolvendo symlinks
	// e bloqueando arquivos sensíveis (.env, .ssh, ...).
	safeJoin := func(p string) (string, error) {
		return core.ResolveSafePath(projectRoot, p)
	}

	switch toolName {
	case "list_files":
		dir := args["dir"]
		if dir == "" {
			dir = "."
		}
		full, err := safeJoin(dir)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		entries, err := os.ReadDir(full)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		var out []string
		for _, e := range entries {
			prefix := "FILE"
			if e.IsDir() {
				prefix = "DIR "
			}
			out = append(out, fmt.Sprintf("%s %s", prefix, e.Name()))
		}
		return ToolResult{Success: true, Content: strings.Join(out, "\n")}
	case "read_file":
		full, err := safeJoin(args["path"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		data, err := os.ReadFile(full)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		const maxBytes = 256 * 1024
		if len(data) > maxBytes {
			return ToolResult{Success: true, Content: string(data[:maxBytes]) + fmt.Sprintf("\n\n... [truncado: %d de %d bytes]", maxBytes, len(data))}
		}
		return ToolResult{Success: true, Content: string(data)}
	case "write_file":
		full, err := safeJoin(args["path"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		if err := os.WriteFile(full, []byte(args["content"]), 0644); err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		return ToolResult{Success: true, Content: fmt.Sprintf("Arquivo escrito: %s (%d bytes)", args["path"], len(args["content"]))}
	case "search_replace":
		full, err := safeJoin(args["path"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		replaceAll := args["replace_all"] == "true" || args["replace_all"] == "1"
		// args map is string-valued; also accept from raw if needed
		n, err := core.ApplySearchReplace(full, args["old_string"], args["new_string"], replaceAll)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		return ToolResult{Success: true, Content: fmt.Sprintf("search_replace ok: %s (%d ocorrência(s))", args["path"], n)}
	case "run_command":
		out, err := core.RunShellCommand(projectRoot, args["cwd"], args["command"])
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		return ToolResult{Success: !strings.HasPrefix(out, "exit=") || strings.HasPrefix(out, "exit=0\n"), Content: out}
	case "git_status":
		files, err := tools.GitStatus(projectRoot)
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		branch, _ := tools.GitCurrentBranch(projectRoot)
		var lines []string
		lines = append(lines, "branch: "+branch)
		for _, f := range files {
			lines = append(lines, fmt.Sprintf("%s%s %s", f.Index, f.Work, f.Path))
		}
		if len(files) == 0 {
			lines = append(lines, "(limpo)")
		}
		_ = service.NewAuditService().Log(service.AuditEntry{Project: filepath.Base(projectRoot), Tool: "git_status", Success: true})
		return ToolResult{Success: true, Content: strings.Join(lines, "\n")}
	case "git_diff":
		var out string
		var err error
		if p := args["path"]; p != "" {
			out, err = tools.GitDiffFile(projectRoot, p)
		} else {
			out, err = tools.GitDiff(projectRoot, false)
		}
		if err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		if out == "" {
			out = "(sem diff)"
		}
		return ToolResult{Success: true, Content: out}
	case "git_branch":
		name := args["name"]
		if name == "" {
			return ToolResult{Success: false, Error: "name obrigatorio"}
		}
		if err := tools.GitCreateBranch(projectRoot, name); err != nil {
			_ = service.NewAuditService().Log(service.AuditEntry{Project: filepath.Base(projectRoot), Tool: "git_branch", Path: name, Success: false, Error: err.Error()})
			return ToolResult{Success: false, Error: err.Error()}
		}
		cur, _ := tools.GitCurrentBranch(projectRoot)
		_ = service.NewAuditService().Log(service.AuditEntry{Project: filepath.Base(projectRoot), Tool: "git_branch", Path: cur, Success: true})
		return ToolResult{Success: true, Content: "branch atual: " + cur}
	case "git_checkout":
		name := args["name"]
		if err := tools.GitCheckout(projectRoot, name); err != nil {
			return ToolResult{Success: false, Error: err.Error()}
		}
		cur, _ := tools.GitCurrentBranch(projectRoot)
		_ = service.NewAuditService().Log(service.AuditEntry{Project: filepath.Base(projectRoot), Tool: "git_checkout", Path: cur, Success: true})
		return ToolResult{Success: true, Content: "branch atual: " + cur}
	default:
		return ToolResult{Success: false, Error: "tool desconhecida: " + toolName}
	}
}
