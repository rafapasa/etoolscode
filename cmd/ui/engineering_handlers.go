package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/engineering"
)

// ListEngineeringStyles GET /api/engineering/styles
func (h *Handler) ListEngineeringStyles(c *fiber.Ctx) error {
	return c.JSON(engineering.AvailableStyles())
}

// BuildEngineeringPlan POST /api/engineering/plan
func (h *Handler) BuildEngineeringPlan(c *fiber.Ctx) error {
	var brief engineering.Brief
	if err := c.BodyParser(&brief); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	plan, err := engineering.BuildPlan(brief)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(plan)
}

// ScaffoldEngineering POST /api/engineering/scaffold
// Body: { project, plan, skip_existing, create_issue }
func (h *Handler) ScaffoldEngineering(c *fiber.Ctx) error {
	var body struct {
		Project      string            `json:"project"`
		Plan         engineering.Plan  `json:"plan"`
		SkipExisting bool              `json:"skip_existing"`
		CreateIssue  bool              `json:"create_issue"`
		IssueName    string            `json:"issue_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if strings.TrimSpace(body.Project) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project obrigatório"})
	}
	root := h.cfg.ResolveProjectRoot(body.Project)
	if root == "" {
		return c.Status(404).JSON(fiber.Map{"error": "raiz do projeto não encontrada no Config"})
	}
	if body.SkipExisting {
		// default true for safety if not set — zero value is false; UI should send true
	}
	res, err := engineering.WriteScaffold(root, &body.Plan, body.SkipExisting)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if body.CreateIssue {
		issueName := strings.TrimSpace(body.IssueName)
		if issueName == "" {
			issueName = "tdd-" + sanitizeIssueName(body.Plan.Feature) + ".json"
		}
		if !strings.HasSuffix(issueName, ".json") {
			issueName += ".json"
		}
		w := config.LoadWorkspace()
		issuesRoot := h.cfg.IssuesDir
		if w.IssuesDir != "" {
			issuesRoot = w.IssuesDir
		}
		projectIssues := filepath.Join(issuesRoot, body.Project)
		_ = os.MkdirAll(projectIssues, 0755)
		issuePath := filepath.Join(projectIssues, filepath.Base(issueName))
		issue := map[string]interface{}{
			"demanda": body.Plan.Summary + "\n\nImplemente os stubs gerados pelo Assistente de Engenharia até os testes unitários passarem. Não altere as interfaces geradas.",
			"branch":  body.Plan.Branch,
			"arquivos": append(append([]string{}, filesFromPlan(&body.Plan)...)),
			"rules": append([]string{}, body.Plan.Rules...),
			"tarefas": []map[string]string{
				{
					"id":        "1",
					"arquivo":   body.Plan.Interfaces[0].StubFile,
					"tipo":      "edit",
					"descricao": "Implementar métodos do stub até os testes em " + body.Plan.Interfaces[0].TestFile + " passarem.",
				},
			},
		}
		if body.Plan.Branch == "" {
			delete(issue, "branch")
		}
		raw, _ := json.MarshalIndent(issue, "", "  ")
		if err := os.WriteFile(issuePath, raw, 0644); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scaffold ok, issue falhou: " + err.Error(), "scaffold": res})
		}
		res.Issue = issuePath
	}

	return c.JSON(fiber.Map{"ok": true, "root": root, "scaffold": res})
}

func filesFromPlan(p *engineering.Plan) []string {
	var out []string
	for _, iface := range p.Interfaces {
		out = append(out, iface.File, iface.StubFile, iface.TestFile)
	}
	return out
}

func sanitizeIssueName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '_' || r == '-' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "scaffold"
	}
	return out
}
