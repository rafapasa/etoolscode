package ui

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core/tools"
	"github.com/rafapasa/etoolscode/internal/dto"
	"github.com/rafapasa/etoolscode/internal/service"
)

func StartPro(cfg *config.Config) error {
	app := fiber.New(fiber.Config{
		AppName:      "eToolsCode IA Pro",
		BodyLimit:    10 * 1024 * 1024,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  24 * time.Hour,
	})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{AllowOrigins: "*", AllowHeaders: "Origin, Content-Type, Accept, Authorization", AllowMethods: "GET, POST, PATCH, DELETE, OPTIONS"}))

	auth, err := NewAuthFromEnv()
	if err == nil {
		auth.Register(app)
		app.Use(auth.Middleware())
	}

	h := NewHandler(cfg)
	ch := NewChatHandler(cfg)

	app.Get("/static/*", func(c *fiber.Ctx) error {
		rawPath := strings.TrimPrefix(c.Params("*"), "/")
		cleanPath := path.Clean(rawPath)
		data, err := fs.ReadFile(staticSub, cleanPath)
		if err != nil {
			return c.Status(404).SendString("static not found: " + cleanPath)
		}
		switch {
		case strings.HasSuffix(cleanPath, ".css"):
			c.Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".js"):
			c.Set("Content-Type", "application/javascript; charset=utf-8")
		}
		c.Set("Cache-Control", "no-cache")
		return c.Send(data)
	})

	getProHTML := func() []byte {
		data, err := fs.ReadFile(templatesFS, "templates/layout_pro.html")
		if err != nil {
			log.Printf("layout_pro.html não encontrado: %v", err)
			return indexHTML
		}
		return data
	}

	app.Get("/pro", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.Send(getProHTML())
	})

	app.Get("/", func(c *fiber.Ctx) error {
		if os.Getenv("DSAC_UI_PRO") == "1" || os.Getenv("DSAC_UI_PRO") == "true" {
			c.Set("Content-Type", "text/html; charset=utf-8")
			return c.Send(getProHTML())
		}
		return c.Send(indexHTML)
	})

	api := app.Group("/api")
	api.Get("/config", h.GetConfig)
	api.Post("/config", h.SaveConfig)
	api.Get("/env", h.CheckEnv)
	api.Get("/issues", h.ListIssues)
	api.Get("/projects", h.ListProjects)
	api.Get("/tree/:project", h.ProjectTree)
	api.Get("/base", h.GetBase)
	api.Post("/base", h.SaveBase)
	api.Post("/base/upload", h.UploadBase)
	api.Get("/projeto/:project", h.GetProjeto)
	api.Post("/projeto/:project", h.SaveProjeto)
	api.Post("/projeto/:project/upload", h.UploadProjeto)
	api.Post("/issues/create", h.CreateIssue)
	api.Post("/issues/upload", h.UploadIssue)
	api.Post("/run", h.Run)
	api.Get("/run/:id/logs", h.RunLogs)
	api.Post("/stop", h.Stop)
	api.Get("/file/:project/*", h.GetIssueFile)
	api.Post("/file/:project/*", h.SaveIssueFile)
	api.Post("/file/:project/*/upload", h.UploadIssueReplace)

	chatApi := api.Group("/chat")
	chatApi.Post("/new", ch.CreateChat)
	chatApi.Get("/", ch.ListChats)
	chatApi.Get("/:id/resumo", ch.ExportResumo)
	chatApi.Get("/:id", ch.GetChat)
	chatApi.Patch("/:id", ch.RenameChat)
	chatApi.Delete("/:id", ch.DeleteChat)
	chatApi.Post("/:id/message", ch.PostMessage)
	chatApi.Post("/:id/export", ch.ExportChat)

	proUI := api.Group("/ui")
	proUI.Get("/explorer", func(c *fiber.Ctx) error {
		var html strings.Builder
		entries, _ := os.ReadDir(cfg.IssuesDir)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			html.WriteString(fmt.Sprintf(`<div class="mb-2 text-xs">📁 %s</div>`, e.Name()))
		}
		c.Set("Content-Type", "text/html")
		return c.SendString(html.String())
	})

	// Git API (JSON)
	resolveGitRoot := func(c *fiber.Ctx) string {
		if r := strings.TrimSpace(c.Query("root")); r != "" {
			return r
		}
		proj := strings.TrimSpace(c.Query("project"))
		if proj == "" {
			var body struct {
				Root    string `json:"root"`
				Project string `json:"project"`
			}
			_ = c.BodyParser(&body)
			if body.Root != "" {
				return body.Root
			}
			proj = body.Project
		}
		if proj != "" && cfg.IssuesDir != "" {
			// projeto.json -> raiz
			data, err := os.ReadFile(filepath.Join(cfg.IssuesDir, proj, "projeto.json"))
			if err == nil {
				var meta struct {
					Raiz string `json:"raiz"`
				}
				if json.Unmarshal(data, &meta) == nil && strings.TrimSpace(meta.Raiz) != "" {
					return strings.TrimSpace(meta.Raiz)
				}
			}
			cand := filepath.Join(filepath.Dir(cfg.IssuesDir), proj)
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				return cand
			}
			return filepath.Join(os.Getenv("HOME"), "prj", proj)
		}
		return ""
	}

	api.Get("/git/status", func(c *fiber.Ctx) error {
		root := resolveGitRoot(c)
		files, err := tools.GitStatus(root)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error(), "root": root})
		}
		branch, _ := tools.GitCurrentBranch(root)
		return c.JSON(fiber.Map{"root": root, "branch": branch, "files": files})
	})

	api.Get("/git/diff", func(c *fiber.Ctx) error {
		root := resolveGitRoot(c)
		file := c.Query("file")
		var (
			diff string
			err  error
		)
		if file != "" {
			diff, err = tools.GitDiffFile(root, file)
		} else {
			diff, err = tools.GitDiff(root, c.Query("staged") == "1")
		}
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error(), "root": root})
		}
		return c.JSON(fiber.Map{"root": root, "file": file, "diff": diff})
	})

	api.Post("/git/branch", func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name"`
			Root string `json:"root"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		root := body.Root
		if root == "" {
			root = resolveGitRoot(c)
		}
		if err := tools.GitCreateBranch(root, body.Name); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		branch, _ := tools.GitCurrentBranch(root)
		return c.JSON(fiber.Map{"ok": true, "root": root, "branch": branch})
	})

	api.Post("/git/checkout", func(c *fiber.Ctx) error {
		var body struct {
			Name string `json:"name"`
			Root string `json:"root"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		root := body.Root
		if root == "" {
			root = resolveGitRoot(c)
		}
		if err := tools.GitCheckout(root, body.Name); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		branch, _ := tools.GitCurrentBranch(root)
		return c.JSON(fiber.Map{"ok": true, "root": root, "branch": branch})
	})

	api.Post("/rag/rebuild", func(c *fiber.Ctx) error {
		var req struct{ Project string }
		_ = c.BodyParser(&req)
		home, _ := os.UserHomeDir()
		root := filepath.Join(home, "prj", req.Project)
		rag := service.NewRAGService(root)
		_ = rag.BuildIndex(root)
		return c.JSON(fiber.Map{"files": len(rag.ListFiles())})
	})

	api.Post("/plan/generate", func(c *fiber.Ctx) error {
		var req struct{ Project, Issue string }
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).SendString(err.Error())
		}
		home, _ := os.UserHomeDir()
		planner := service.NewPlannerService(service.NewRAGService(filepath.Join(home, "prj", req.Project)))
		demanda := req.Issue
		issuePath := filepath.Join(cfg.IssuesDir, req.Project, req.Issue)
		if data, err := os.ReadFile(issuePath); err == nil {
			demanda = string(data)
			if len(demanda) > 500 {
				demanda = demanda[:500]
			}
		}
		plan, err := planner.GeneratePlan(filepath.Join(home, "prj", req.Project), req.Issue, dto.Request{Demanda: demanda}, "")
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.JSON(plan)
	})

	
	api.Get("/git/file", func(c *fiber.Ctx) error {
		root := resolveGitRoot(c)
		file := c.Query("file")
		if file == "" {
			return c.Status(400).JSON(fiber.Map{"error": "file obrigatorio"})
		}
		original, _ := tools.GitFileAtHEAD(root, file)
		modified, err2 := tools.GitFileWorking(root, file)
		if err2 != nil && original == "" {
			return c.Status(400).JSON(fiber.Map{"error": err2.Error()})
		}
		if modified == "" && err2 != nil {
			modified = ""
		}
		return c.JSON(fiber.Map{
			"root":     root,
			"file":     file,
			"original": original,
			"modified": modified,
		})
	})

	api.Get("/audit/recent", func(c *fiber.Ctx) error {
		limit := 20
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}
		project := c.Query("project")
		items := service.NewAuditService().List(project, limit)
		return c.JSON(fiber.Map{"items": items, "count": len(items)})
	})

	fmt.Printf("[eTools-Code Pro] http://localhost:%d\n", cfg.HttpPort)
	return app.Listen(fmt.Sprintf(":%d", cfg.HttpPort))
}
