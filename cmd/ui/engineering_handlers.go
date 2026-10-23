package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core"
	"github.com/rafapasa/etoolscode/internal/engineering"
)

// ListEngineeringStyles GET /api/engineering/styles
func (h *Handler) ListEngineeringStyles(c *fiber.Ctx) error {
	return c.JSON(engineering.AvailableStyles())
}

// DetectEngineeringStyle GET /api/engineering/detect?project=name
func (h *Handler) DetectEngineeringStyle(c *fiber.Ctx) error {
	project := strings.TrimSpace(c.Query("project"))
	if project == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project obrigatório"})
	}
	root := h.cfg.ResolveProjectRoot(project)
	if root == "" {
		return c.Status(404).JSON(fiber.Map{"error": "raiz do projeto não encontrada"})
	}
	style, reason := engineering.DetectStyle(root)
	return c.JSON(fiber.Map{
		"project":   project,
		"root":      root,
		"style":     style,
		"reason":    reason,
		"internal":  engineering.ListTopInternal(root),
	})
}

// BuildEngineeringPlan POST /api/engineering/plan
// Body: Brief + opcional enrich_llm bool
func (h *Handler) BuildEngineeringPlan(c *fiber.Ctx) error {
	var body struct {
		engineering.Brief
		EnrichLLM bool `json:"enrich_llm"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	plan, err := engineering.BuildPlan(body.Brief)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	var enrichErr string
	if body.EnrichLLM {
		llm := safeNewLlm(h.cfg)
		if llm != nil {
			if p2, err := engineering.EnrichPlanWithLLM(llm, plan, body.Description); err != nil {
				enrichErr = err.Error()
			} else {
				plan = p2
			}
		} else {
			enrichErr = "LLM indisponível — plano template mantido"
		}
	}
	return c.JSON(fiber.Map{"plan": plan, "enrich_error": enrichErr})
}

// PreviewEngineeringScaffold POST /api/engineering/preview — não grava disco
func (h *Handler) PreviewEngineeringScaffold(c *fiber.Ctx) error {
	var body struct {
		Plan engineering.Plan `json:"plan"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	files, err := engineering.GenerateFiles(&body.Plan)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"files": files})
}

// ScaffoldEngineering POST /api/engineering/scaffold
func (h *Handler) ScaffoldEngineering(c *fiber.Ctx) error {
	var body struct {
		Project      string           `json:"project"`
		Plan         engineering.Plan `json:"plan"`
		SkipExisting bool             `json:"skip_existing"`
		CreateIssue  bool             `json:"create_issue"`
		IssueName    string           `json:"issue_name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	if strings.TrimSpace(body.Project) == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project obrigatório"})
	}
	if len(body.Plan.Interfaces) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "plano sem interfaces"})
	}
	root := h.cfg.ResolveProjectRoot(body.Project)
	if root == "" {
		return c.Status(404).JSON(fiber.Map{"error": "raiz do projeto não encontrada no Config"})
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
		iface0 := body.Plan.Interfaces[0]
		issue := map[string]interface{}{
			"demanda": body.Plan.Summary + "\n\nImplemente os stubs gerados pelo Assistente de Engenharia até os testes unitários passarem. Não altere as interfaces geradas.",
			"arquivos": filesFromPlan(&body.Plan),
			"rules":    body.Plan.Rules,
			"tarefas": []map[string]string{
				{
					"id":        "1",
					"arquivo":   iface0.StubFile,
					"tipo":      "edit",
					"descricao": "Implementar métodos do stub até os testes em " + iface0.TestFile + " passarem.",
				},
			},
		}
		if strings.TrimSpace(body.Plan.Branch) != "" {
			issue["branch"] = body.Plan.Branch
		}
		raw, _ := json.MarshalIndent(issue, "", "  ")
		if err := os.WriteFile(issuePath, raw, 0644); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "scaffold ok, issue falhou: " + err.Error(), "scaffold": res})
		}
		res.Issue = issuePath
	}

	return c.JSON(fiber.Map{"ok": true, "root": root, "scaffold": res})
}

func safeNewLlm(cfg *config.Config) core.LlmInterface {
	if cfg == nil {
		return nil
	}
	defer func() { recover() }()
	return core.NewLlmClient(*cfg)
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
