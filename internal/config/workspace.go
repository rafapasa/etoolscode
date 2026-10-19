package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ProjectEntry cadastro de um projeto (código + pasta de issues opcional).
type ProjectEntry struct {
	Name      string `json:"name"`
	Root      string `json:"root"`                 // pasta do código-fonte
	IssuesDir string `json:"issues_dir,omitempty"` // override; vazio = IssuesDir global + /name
}

// Workspace configuração de pastas editável pela UI.
type Workspace struct {
	IssuesDir    string         `json:"issues_dir"`              // pasta raiz das issues
	ProjectsRoot string         `json:"projects_root,omitempty"` // fallback para resolver raiz
	Projects     []ProjectEntry `json:"projects"`
}

var (
	wsMu   sync.Mutex
	wsPath string
)

func workspacePath() string {
	if wsPath != "" {
		return wsPath
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "workspace.json")
}

func LoadWorkspace() Workspace {
	wsMu.Lock()
	defer wsMu.Unlock()
	var w Workspace
	data, err := os.ReadFile(workspacePath())
	if err == nil {
		_ = json.Unmarshal(data, &w)
	}
	if w.Projects == nil {
		w.Projects = []ProjectEntry{}
	}
	return w
}

func SaveWorkspace(w Workspace) error {
	wsMu.Lock()
	defer wsMu.Unlock()
	if w.Projects == nil {
		w.Projects = []ProjectEntry{}
	}
	path := workspacePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ResolveProjectRoot encontra a pasta do código pelo nome.
// Ordem: workspace.Projects → projeto.json.raiz → ProjectsRoot/name → sibling de IssuesDir.
func (c *Config) ResolveProjectRoot(project string) string {
	if project == "" {
		return ""
	}
	w := LoadWorkspace()
	for _, p := range w.Projects {
		if strings.EqualFold(p.Name, project) && strings.TrimSpace(p.Root) != "" {
			return strings.TrimSpace(p.Root)
		}
	}
	// projeto.json em IssuesDir
	issuesDir := c.IssuesDir
	if w.IssuesDir != "" {
		issuesDir = w.IssuesDir
	}
	if issuesDir != "" {
		data, err := os.ReadFile(filepath.Join(issuesDir, project, "projeto.json"))
		if err == nil {
			var meta struct {
				Raiz string `json:"raiz"`
			}
			if json.Unmarshal(data, &meta) == nil && strings.TrimSpace(meta.Raiz) != "" {
				return strings.TrimSpace(meta.Raiz)
			}
		}
	}
	if w.ProjectsRoot != "" {
		cand := filepath.Join(w.ProjectsRoot, project)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
	}
	if issuesDir != "" {
		cand := filepath.Join(filepath.Dir(issuesDir), project)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
	}
	return filepath.Join(os.Getenv("HOME"), "prj", project)
}

// EffectiveIssuesDir retorna a pasta de issues (workspace sobrescreve config).
func (c *Config) EffectiveIssuesDir() string {
	w := LoadWorkspace()
	if w.IssuesDir != "" {
		return w.IssuesDir
	}
	return c.IssuesDir
}
