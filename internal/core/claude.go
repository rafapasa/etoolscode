package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rafapasa/etoolscode/internal/config"
	"github.com/rafapasa/etoolscode/internal/dto"
)

type ClaudeClient struct {
	apiKey, model, baseURL string
	client                 *http.Client
}

func NewClaudeClient(cfg config.Config) *ClaudeClient {
	key := cfg.ClaudeApiKey
	model := "claude-3-5-sonnet-20241022"
	baseURL := "https://api.anthropic.com/v1"
	if cfg.LLMProviders != nil {
		if p, ok := cfg.LLMProviders["claude"]; ok {
			if p.APIKey != "" {
				key = p.APIKey
			}
			if p.Model != "" {
				model = p.Model
			}
		}
	}
	return &ClaudeClient{apiKey: key, model: model, baseURL: baseURL, client: &http.Client{Timeout: 120 * time.Second}}
}
func (c *ClaudeClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := map[string]any{"model": c.model, "messages": messages}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.baseURL+"/v1/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("claude %d: %s", resp.StatusCode, string(data))
	}
	var out dto.ChatResponse
	json.Unmarshal(data, &out)
	return &out, nil
}
func (c *ClaudeClient) ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	resp, err := c.Chat(messages, tools)
	if err != nil {
		return err
	}
	if len(resp.Choices) > 0 {
		content := resp.Choices[0].Message.Content
		for i := 0; i < len(content); i += 30 {
			end := i + 30
			if end > len(content) {
				end = len(content)
			}
			onDelta(content[i:end])
		}
		for _, tc := range resp.Choices[0].Message.ToolCalls {
			onToolCall(tc)
		}
	}
	return nil
}
