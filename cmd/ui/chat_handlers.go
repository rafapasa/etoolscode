package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rafapasa/etoolscode/internal/chat"
	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/dto"
)

type ChatHandler struct {
	cfg     *config.Config
	service *chat.Service
}

func NewChatHandler(cfg *config.Config) *ChatHandler {
	if err := chat.MergeHistoryByProject(); err != nil {
		log.Printf("[chat] merge histórico: %v", err)
	}
	return &ChatHandler{cfg: cfg, service: chat.NewService(cfg)}
}

func (h *ChatHandler) CreateChat(c *fiber.Ctx) error {
	var body struct {
		Project string `json:"project"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	session, err := h.service.CreateSession(body.Project)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": session.ID, "project": session.Project})
}

func (h *ChatHandler) ListChats(c *fiber.Ctx) error {
	list, err := chat.ListSessions()
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"chats": list})
}

func (h *ChatHandler) GetChat(c *fiber.Ctx) error {
	s, err := chat.LoadSession(c.Params("id"))
	if err != nil {
		return c.Status(404).SendString("chat não encontrado")
	}
	return c.JSON(s)
}

func (h *ChatHandler) PostMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if body.Content == "" {
		return c.Status(400).SendString("content vazio")
	}

	session, err := h.service.AddUserMessage(id, body.Content)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	projectRoot := h.service.ResolveProjectRoot(session.Project)
	tools := chat.GetTools()
	llm := chat.NewChatLLM(h.cfg)
	maxIters := 32
	if h.cfg != nil {
		maxIters = h.cfg.EffectiveChatMaxIters()
	}

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		limitHit := false
		lastHadTools := false

		for iter := 0; iter < maxIters; iter++ {
			writeSSEChat(w, fiber.Map{
				"type":    "turn",
				"current": iter + 1,
				"max":     maxIters,
			})

			messages := chat.SanitizeMessages(session.Messages)
			var full string
			var toolCalls []dto.ToolCall

			err := llm.StreamChat(messages, tools,
				func(delta string) {
					full += delta
					writeSSEChat(w, fiber.Map{"type": "delta", "content": delta})
				},
				func(tc dto.ToolCall) {
					if tc.Type == "" {
						tc.Type = "function"
					}
					if tc.ID == "" {
						tc.ID = fmt.Sprintf("call_%d_%d", iter, len(toolCalls))
					}
					toolCalls = append(toolCalls, tc)
				},
			)
			if err != nil {
				writeSSEChat(w, fiber.Map{"type": "error", "error": err.Error()})
				return
			}

			if len(toolCalls) == 0 {
				if full != "" {
					session.Messages = append(session.Messages, dto.Message{Role: "assistant", Content: full})
					if err := h.service.PersistTurn(session); err != nil {
						log.Printf("[chat] persist: %v", err)
					}
				}
				writeSSEChat(w, fiber.Map{
					"type":    "done",
					"content": full,
					"current": iter + 1,
					"max":     maxIters,
				})
				return
			}

			lastHadTools = true
			assistant := dto.Message{Role: "assistant", Content: full, ToolCalls: toolCalls}
			session.Messages = append(session.Messages, assistant)

			for _, tc := range toolCalls {
				result := chat.ExecuteTool(projectRoot, tc.Function.Name, tc.Function.Arguments)
				writeSSEChat(w, toolEvent(tc.Function.Name, tc.Function.Arguments, result))
				content := result.Content
				if content == "" && result.Error != "" {
					content = "ERRO: " + result.Error
				}
				if content == "" {
					content = "ERRO: falha na tool " + tc.Function.Name
				}
				session.Messages = append(session.Messages, dto.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Name:       tc.Function.Name,
					Content:    content,
				})
			}
			if err := h.service.PersistTurn(session); err != nil {
				log.Printf("[chat] persist tools: %v", err)
			}

			// Última iteração e ainda há tools: não termina mudo — pede resumo em texto.
			if iter == maxIters-1 {
				limitHit = true
				break
			}
		}

		// Limite atingido: turno de fechamento só com texto (sem tools).
		if limitHit || lastHadTools {
			session.Messages = append(session.Messages, dto.Message{
				Role: "user",
				Content: "⚠️ LIMITE DE RODADAS ATINGIDO neste turno. Não use tools agora. " +
					"Resuma em português o que já foi feito, o que ficou pendente e o próximo passo recomendado. Seja objetivo.",
			})
			messages := chat.SanitizeMessages(session.Messages)
			var full string
			err := llm.StreamChat(messages, nil,
				func(delta string) {
					full += delta
					writeSSEChat(w, fiber.Map{"type": "delta", "content": delta})
				},
				func(tc dto.ToolCall) {}, // ignora tools no fechamento
			)
			// remove o user sintético do histórico persistido (evita poluir próxima rodada)
			if len(session.Messages) > 0 && strings.Contains(session.Messages[len(session.Messages)-1].Content, "LIMITE DE RODADAS") {
				session.Messages = session.Messages[:len(session.Messages)-1]
			}
			if err != nil {
				writeSSEChat(w, fiber.Map{"type": "error", "error": err.Error()})
				return
			}
			if full != "" {
				session.Messages = append(session.Messages, dto.Message{Role: "assistant", Content: full})
				if err := h.service.PersistTurn(session); err != nil {
					log.Printf("[chat] persist limit summary: %v", err)
				}
			}
			writeSSEChat(w, fiber.Map{
				"type":          "done",
				"content":       full,
				"limit_reached": true,
				"current":       maxIters,
				"max":           maxIters,
				"message":       "Limite de rodadas atingido. Use Continuar ou Resumo.",
			})
			return
		}

		writeSSEChat(w, fiber.Map{
			"type":          "done",
			"content":       "",
			"limit_reached": true,
			"current":       maxIters,
			"max":           maxIters,
			"message":       "Limite de rodadas atingido.",
		})
	})

	return nil
}

func (h *ChatHandler) RenameChat(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).SendString(err.Error())
	}
	if strings.TrimSpace(body.Title) == "" {
		return c.Status(400).SendString("título vazio")
	}
	sess, err := chat.RenameSession(id, body.Title)
	if err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"id": sess.ID, "title": sess.Title})
}

func (h *ChatHandler) DeleteChat(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := chat.DeleteSession(id); err != nil {
		return c.Status(404).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *ChatHandler) ExportResumo(c *fiber.Ctx) error {
	s, err := chat.LoadSession(c.Params("id"))
	if err != nil {
		return c.Status(404).SendString("chat não encontrado")
	}
	body := chat.BuildResumo(s)
	name := s.Title
	if name == "" {
		name = "chat"
	}
	c.Set("Content-Type", "text/markdown; charset=utf-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", sanitizeFileName(name)+".md"))
	return c.SendString(body)
}

func sanitizeFileName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "chat"
	}
	var b strings.Builder
	for _, r := range s {
		if r == '/' || r == '\\' || r == ':' || r < 32 {
			b.WriteByte('-')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "chat"
	}
	return out
}

func (h *ChatHandler) ExportChat(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Demanda string `json:"demanda"`
	}
	if err := c.BodyParser(&body); err != nil {
		log.Printf("[ERRO] %v", err.Error())
	}
	path, err := h.service.ExportToIssue(id, body.Demanda)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "path": path})
}

func writeSSEChat(w *bufio.Writer, payload interface{}) {
	line := fmt.Sprintf("data: %s\n\n", jsonStringChat(payload))
	if _, err := w.WriteString(line); err != nil {
		log.Printf("[ERRO] %v", err)
	}
	_ = w.Flush()
}

func jsonStringChat(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func toolEvent(name, argsJSON string, result chat.ToolResult) fiber.Map {
	path := ""
	detail := ""
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(argsJSON), &args)
	if args != nil {
		if v, ok := args["path"].(string); ok && strings.TrimSpace(v) != "" {
			path = v
		} else if v, ok := args["dir"].(string); ok && strings.TrimSpace(v) != "" {
			path = v
		} else if v, ok := args["command"].(string); ok && strings.TrimSpace(v) != "" {
			path = v
		} else if v, ok := args["name"].(string); ok && strings.TrimSpace(v) != "" {
			// git_branch name
			path = v
		}
	}
	if path == "" && (name == "list_files" || name == "list_dir") {
		path = "."
	}
	if path == "" && name == "git_status" {
		path = "git status"
	}

	kind := "read"
	switch name {
	case "list_files", "list_dir":
		kind = "list"
	case "write_file", "search_replace", "apply_patch":
		kind = "write"
	case "run_command":
		kind = "command"
	case "git_status", "git_diff", "git_branch", "git_checkout":
		kind = "git"
	}

	switch name {
	case "read_file":
		if result.Success {
			detail = fmt.Sprintf("%d bytes", len(result.Content))
		} else {
			detail = result.Error
		}
	case "write_file":
		if result.Success {
			detail = result.Content
			if detail == "" {
				detail = "ok"
			}
		} else {
			detail = result.Error
		}
	case "search_replace":
		if result.Success {
			detail = result.Content
		} else {
			detail = result.Error
		}
	case "list_files", "list_dir":
		if result.Success {
			itens := 0
			for _, line := range strings.Split(strings.TrimSpace(result.Content), "\n") {
				if strings.TrimSpace(line) != "" {
					itens++
				}
			}
			detail = fmt.Sprintf("%d itens", itens)
		} else {
			detail = result.Error
		}
	case "run_command":
		cmd, _ := args["command"].(string)
		detail = "$ " + cmd
		if strings.HasPrefix(result.Content, "exit=") {
			if nl := strings.Index(result.Content, "\n"); nl >= 0 {
				exitLine := result.Content[:nl]
				out := strings.TrimSpace(result.Content[nl+1:])
				detail += " — " + exitLine
				if out != "" {
					if len(out) > 300 {
						out = out[:300] + "..."
					}
					detail += "\n" + out
				}
			}
		}
	case "git_status":
		detail = "status"
	case "git_diff":
		p, _ := args["path"].(string)
		if p != "" {
			detail = "diff " + p
		} else {
			detail = "diff"
		}
	case "git_branch":
		n, _ := args["name"].(string)
		detail = "branch " + n
	case "git_checkout":
		n, _ := args["name"].(string)
		detail = "checkout " + n
	}

	if detail == "" && !result.Success {
		detail = result.Error
	}
	if detail == "" && result.Success {
		detail = "ok"
	}

	if path == "" && result.Content != "" {
		// fallback: primeira linha útil do resultado
		for _, line := range strings.Split(result.Content, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "ERRO") {
				path = line
				if len(path) > 120 {
					path = path[:120] + "…"
				}
				break
			}
		}
	}
	if path == "" {
		path = name
	}
	if detail == "" && !result.Success {
		detail = result.Error
	}
	return fiber.Map{
		"type":   "tool",
		"name":   name,
		"kind":   kind,
		"path":   path,
		"ok":     result.Success,
		"error":  result.Error,
		"bytes":  len(result.Content),
		"detail": detail,
	}
}
