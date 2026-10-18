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

type OpenAIClient struct {
	apiKey, model, baseURL string
	client                 *http.Client
}

func NewOpenAIClient(cfg config.Config) *OpenAIClient {
	key := cfg.OpenAIApiKey
	model := "gpt-4o"
	baseURL := "https://api.openai.com/v1"
	if cfg.LLMProviders != nil {
		if p, ok := cfg.LLMProviders["openai"]; ok {
			if p.APIKey != "" {
				key = p.APIKey
			}
			if p.Model != "" {
				model = p.Model
			}
		}
	}
	return &OpenAIClient{apiKey: key, model: model, baseURL: baseURL, client: &http.Client{Timeout: 120 * time.Second}}
}
func (c *OpenAIClient) Chat(messages []dto.Message, tools []dto.Tool) (*dto.ChatResponse, error) {
	payload := map[string]any{"model": c.model, "messages": messages}
	if len(tools) > 0 {
		payload["tools"] = tools
		payload["tool_choice"] = "auto"
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai %d: %s", resp.StatusCode, string(data))
	}
	var out dto.ChatResponse
	json.Unmarshal(data, &out)
	return &out, nil
}
func (c *OpenAIClient) ChatStream(messages []dto.Message, tools []dto.Tool, onDelta func(string), onToolCall func(dto.ToolCall)) error {
	payload := dto.ChatRequest{Model: c.model, Messages: messages, Temperature: 0.2, Stream: true}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", c.baseURL+"/chat/completions", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		d, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai stream %d: %s", resp.StatusCode, string(d))
	}
	return parseStream(resp.Body, onDelta, onToolCall)
}
