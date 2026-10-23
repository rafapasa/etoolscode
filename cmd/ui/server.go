package ui

import (
	"embed"
	"fmt"
	"os"
	"encoding/json"
	"io/fs"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/core/tools"
	"github.com/rafapasa/etoolscode/internal/service"
)

var (
	//go:embed all:templates
	templatesFS embed.FS

	//go:embed all:static
	staticFS embed.FS

	indexHTML []byte
	staticSub fs.FS
)

func init() {
	var err error
	// Descasca a pasta raiz "static" do FS embutido
	staticSub, err = fs.Sub(staticFS, "static")
	if err != nil {
		panic("falha ao inicializar sub-filesystem estatico: " + err.Error())
	}

	data, err := fs.ReadFile(templatesFS, "templates/index.html")
	if err != nil {
		indexHTML = []byte("<h1>erro ao carregar template</h1>")
	} else {
		indexHTML = data
	}
}

func Start(cfg *config.Config) error {
	// Autenticação: falha ao iniciar se nenhuma senha estiver configurada
	auth, err := NewAuthFromEnv()
	if err != nil {
		return err
	}

	// Configuração resiliente do Fiber HTTP Server
	app := fiber.New(fiber.Config{
		AppName:      "eTools-Code",
		BodyLimit:    10 * 1024 * 1024, // Limite máximo de 10 MB para uploads e JSONs
		ReadTimeout:  30 * time.Second, // Timeout para leitura dos cabeçalhos/body da requisição
		WriteTimeout: 0,                // 0 = Sem timeout de escrita (indispensável para SSE / Stream de LLM)
		IdleTimeout:  24 * time.Hour,   // SSE de issues longas não pode cair por ociosidade
	})

	app.Use(logger.New())

	// CORS configurado explicitamente
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PATCH, DELETE, OPTIONS",
	}))

	// Login/logout + wizard de 1ª execução + proteção
	auth.Register(app)
	RegisterSetup(app, auth, cfg)
	app.Use(auth.Middleware())

	h := NewHandler(cfg)
	ch := NewChatHandler(cfg)

	// Rota estática manual garantindo Content-Type correto e caminho saneado
	app.Get("/static/*", func(c *fiber.Ctx) error {
		rawPath := strings.TrimPrefix(c.Params("*"), "/")
		cleanPath := path.Clean(rawPath)

		data, err := fs.ReadFile(staticSub, cleanPath)
		if err != nil {
			return c.Status(404).SendString("static not found: " + cleanPath)
		}

		// Força os tipos MIME explicitamente
		switch {
		case strings.HasSuffix(cleanPath, ".css"):
			c.Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".js"):
			c.Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(cleanPath, ".png"):
			c.Set("Content-Type", "image/png")
		case strings.HasSuffix(cleanPath, ".ico"):
			c.Set("Content-Type", "image/x-icon")
		default:
			if ct := mime.TypeByExtension(filepath.Ext(cleanPath)); ct != "" {
				c.Set("Content-Type", ct)
			}
		}

		c.Set("Cache-Control", "no-cache")
		return c.Send(data)
	})

	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		data, err := fs.ReadFile(staticSub, "favicon.ico")
		if err == nil {
			c.Set("Content-Type", "image/x-icon")
			c.Set("Cache-Control", "public, max-age=86400")
			return c.Send(data)
		}
		data, err = fs.ReadFile(staticSub, "logo.png")
		if err == nil {
			c.Set("Content-Type", "image/png")
			return c.Send(data)
		}
		return c.SendStatus(204)
	})

	// UI única: index.html (chat + issues + fontes + git)
	app.Get("/pro", func(c *fiber.Ctx) error { return c.Redirect("/", 302) })
	app.Get("/", h.Index)
	api := app.Group("/api")
	api.Get("/workspace", h.GetWorkspace)
	api.Put("/workspace", h.SaveWorkspace)
	api.Post("/workspace/projects", h.UpsertProject)
	api.Delete("/workspace/projects/:name", h.DeleteProject)
	api.Get("/config", h.GetConfig)
	api.Post("/config", h.SaveConfig)
	api.Get("/env", h.CheckEnv)
	api.Get("/issues", h.ListIssues)
	api.Get("/projects", h.ListProjects)
	api.Get("/tree/:project", h.ProjectTree)
	api.Get("/source/:project/*", h.GetSourceFile)
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

	
	resolveGitRoot := func(c *fiber.Ctx) string {
		if r := strings.TrimSpace(c.Query("root")); r != "" {
			return r
		}
		proj := strings.TrimSpace(c.Query("project"))
		if proj != "" && cfg.IssuesDir != "" {
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
		var diff string
		var err error
		if file != "" {
			diff, err = tools.GitDiffFile(root, file)
		} else {
			diff, err = tools.GitDiff(root, c.Query("staged") == "1")
		}
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"root": root, "file": file, "diff": diff})
	})
	api.Get("/git/file", func(c *fiber.Ctx) error {
		root := resolveGitRoot(c)
		file := c.Query("file")
		original, _ := tools.GitFileAtHEAD(root, file)
		modified, err2 := tools.GitFileWorking(root, file)
		if err2 != nil && original == "" {
			return c.Status(400).JSON(fiber.Map{"error": err2.Error()})
		}
		return c.JSON(fiber.Map{"root": root, "file": file, "original": original, "modified": modified})
	})
	api.Post("/git/branch", func(c *fiber.Ctx) error {
		var body struct {
			Name    string `json:"name"`
			Root    string `json:"root"`
			Project string `json:"project"`
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
		return c.JSON(fiber.Map{"ok": true, "branch": branch, "root": root})
	})

	api.Get("/audit/recent", func(c *fiber.Ctx) error {
		limit := 20
		items := service.NewAuditService().List(c.Query("project"), limit)
		return c.JSON(fiber.Map{"items": items, "count": len(items)})
	})

	fmt.Printf("[eTools-Code] http://localhost:%d - ISSUES_DIR=%s\n", cfg.HttpPort, cfg.IssuesDir)
	return app.Listen(fmt.Sprintf(":%d", cfg.HttpPort))
}
