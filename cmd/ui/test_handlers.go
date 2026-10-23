package ui

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// DetectTestCommand GET /api/test/suggest?project=
func (h *Handler) SuggestTestCommand(c *fiber.Ctx) error {
	project := strings.TrimSpace(c.Query("project"))
	if project == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project obrigatório"})
	}
	root := h.cfg.ResolveProjectRoot(project)
	if root == "" {
		return c.Status(404).JSON(fiber.Map{"error": "raiz não encontrada"})
	}
	cmd, reason := suggestTestCmd(root)
	return c.JSON(fiber.Map{"project": project, "root": root, "command": cmd, "reason": reason})
}

// RunProjectTests POST /api/test/run  { project, command? }
func (h *Handler) RunProjectTests(c *fiber.Ctx) error {
	var body struct {
		Project string `json:"project"`
		Command string `json:"command"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	project := strings.TrimSpace(body.Project)
	if project == "" {
		return c.Status(400).JSON(fiber.Map{"error": "project obrigatório"})
	}
	root := h.cfg.ResolveProjectRoot(project)
	if root == "" {
		return c.Status(404).JSON(fiber.Map{"error": "raiz não encontrada"})
	}
	cmdStr := strings.TrimSpace(body.Command)
	if cmdStr == "" {
		cmdStr, _ = suggestTestCmd(root)
	}
	if cmdStr == "" {
		return c.Status(400).JSON(fiber.Map{"error": "não foi possível sugerir comando de teste; informe command"})
	}
	// allowlist simples: um único comando, sem pipes
	if strings.ContainsAny(cmdStr, ";|&`") || strings.Contains(cmdStr, "$(") {
		return c.Status(400).JSON(fiber.Map{"error": "comando contém operadores não permitidos"})
	}
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "comando vazio"})
	}
	ctxTimeout := 3 * time.Minute
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Run() }()
	var runErr error
	select {
	case runErr = <-errCh:
	case <-time.After(ctxTimeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return c.Status(504).JSON(fiber.Map{"error": "timeout 3m", "command": cmdStr})
	}
	out := stdout.String() + stderr.String()
	if len(out) > 200000 {
		out = out[:200000] + "\n...[truncated]"
	}
	ok := runErr == nil
	return c.JSON(fiber.Map{
		"ok":      ok,
		"command": cmdStr,
		"root":    root,
		"output":  out,
		"error":   errString(runErr),
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func suggestTestCmd(root string) (string, string) {
	checks := []struct {
		file string
		cmd  string
		why  string
	}{
		{"go.mod", "go test ./...", "projeto Go"},
		{"pubspec.yaml", "flutter test", "projeto Flutter/Dart"},
		{"package.json", "npm test", "projeto Node (npm test)"},
		{"pytest.ini", "pytest -q", "pytest.ini"},
		{"pyproject.toml", "pytest -q", "pyproject.toml"},
		{"requirements.txt", "pytest -q", "requirements.txt"},
		{"pom.xml", "mvn test", "Maven"},
		{"build.gradle", "gradle test", "Gradle"},
		{"*.csproj", "", ""}, // handled below
	}
	for _, c := range checks {
		if c.file == "*.csproj" {
			continue
		}
		if fileExists(filepath.Join(root, c.file)) {
			return c.cmd, c.why
		}
	}
	// csharp
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if strings.HasSuffix(strings.ToLower(e.Name()), ".csproj") {
			return "dotnet test", "projeto .NET"
		}
	}
	return "", "nenhum gerenciador de teste detectado"
}
