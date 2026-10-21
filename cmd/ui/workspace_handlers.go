package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/etoolscode/internal/config"
)

// GetWorkspace GET /api/workspace
func (h *Handler) GetWorkspace(c *fiber.Ctx) error {
	w := config.LoadWorkspace()
	if w.IssuesDir == "" {
		w.IssuesDir = h.cfg.IssuesDir
	}
	return c.JSON(w)
}

// ensureProjectIssuesDir cria issues_dir/<nome> se ainda não existir.
func ensureProjectIssuesDir(issuesDir, name string) error {
	name = strings.TrimSpace(name)
	if issuesDir == "" || name == "" {
		return nil
	}
	// evita path traversal no nome
	name = filepath.Base(name)
	if name == "." || name == ".." {
		return nil
	}
	return os.MkdirAll(filepath.Join(issuesDir, name), 0755)
}

// SaveWorkspace PUT /api/workspace — issues_dir + projects_root + lista
func (h *Handler) SaveWorkspace(c *fiber.Ctx) error {
	var body config.Workspace
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if body.Projects == nil {
		body.Projects = []config.ProjectEntry{}
	}
	// normaliza nomes
	for i := range body.Projects {
		body.Projects[i].Name = strings.TrimSpace(body.Projects[i].Name)
		body.Projects[i].Root = strings.TrimSpace(body.Projects[i].Root)
		body.Projects[i].IssuesDir = strings.TrimSpace(body.Projects[i].IssuesDir)
	}
	if err := config.SaveWorkspace(body); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if body.IssuesDir != "" {
		h.cfg.IssuesDir = body.IssuesDir
	}
	issuesRoot := h.cfg.IssuesDir
	if body.IssuesDir != "" {
		issuesRoot = body.IssuesDir
	}
	// garante pasta de issues por projeto cadastrado
	for _, p := range body.Projects {
		_ = ensureProjectIssuesDir(issuesRoot, p.Name)
	}
	return c.JSON(fiber.Map{"ok": true, "workspace": body})
}

// UpsertProject POST /api/workspace/projects
func (h *Handler) UpsertProject(c *fiber.Ctx) error {
	var body config.ProjectEntry
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Root = strings.TrimSpace(body.Root)
	if body.Name == "" || body.Root == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name e root obrigatorios"})
	}
	if st, err := os.Stat(body.Root); err != nil || !st.IsDir() {
		return c.Status(400).JSON(fiber.Map{"error": "root nao e diretorio valido: " + body.Root})
	}
	w := config.LoadWorkspace()
	found := false
	for i := range w.Projects {
		if strings.EqualFold(w.Projects[i].Name, body.Name) {
			w.Projects[i] = body
			found = true
			break
		}
	}
	if !found {
		w.Projects = append(w.Projects, body)
	}
	if err := config.SaveWorkspace(w); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	// pasta de issues do projeto
	issuesRoot := w.IssuesDir
	if issuesRoot == "" {
		issuesRoot = h.cfg.IssuesDir
	}
	if err := ensureProjectIssuesDir(issuesRoot, body.Name); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "projeto salvo, mas falha ao criar pasta de issues: " + err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true, "project": body})
}

// DeleteProject DELETE /api/workspace/projects/:name
func (h *Handler) DeleteProject(c *fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	if name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name obrigatorio"})
	}
	w := config.LoadWorkspace()
	var next []config.ProjectEntry
	for _, p := range w.Projects {
		if !strings.EqualFold(p.Name, name) {
			next = append(next, p)
		}
	}
	w.Projects = next
	if err := config.SaveWorkspace(w); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}
