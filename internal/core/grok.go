package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/dto"
)

// GrokClient — xAI Grok (OpenAI-compatible chat completions).
// Docs: https://docs.x.ai — base https://api.x.ai/v1
type GrokClient struct {
	apiKey, model, baseURL string
	client                 *http.Client
}

func NewGrokClient(cfg config.Config) *GrokClient {
	key := strings.TrimSpace(cfg.GrokApiKey)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("XAI_API_KEY"))
	}
	// Flagship atual para código / agentic (docs xAI). Override via LLMProviders["grok"].model
	model := "grok-4.7"
	baseURL := "https://api.x.ai/v1"
	if cfg.LLMProviders != nil {
		// aceita "grok" ou "Grok"
		for _, name := range []string{"grok", "Grok", "xai", "xAI"} {
			if p, ok := cfg.LLMProviders[name]; ok {
				if p.APIKey != "" {
					key = p.APIKey
				}
				if p.Model != "" {
					model = p.Model
				}
				if p.BaseURL != "" {
					baseURL = strings.TrimRight(p.BaseURL, "/")
				}
				break
			}
		}
	}
	return &GrokClient{
		apiKey:  key,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *GrokClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := map[string]any{
		"model":    c.model,
		"messages": messages,
	}
	if len(tools) > 0 {
		payload["tools"] = tools
		payload["tool_choice"] = "auto"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("grok %d: %s", resp.StatusCode, string(data))
	}
	var out dto.ChatResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("grok decode: %w", err)
	}
	return &out, nil
}

func (c *GrokClient) ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	payload := dto.ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.2,
		Stream:      true,
	}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		d, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("grok stream %d: %s", resp.StatusCode, string(d))
	}
	return parseStream(resp.Body, onDelta, onToolCall)
}
