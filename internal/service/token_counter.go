package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TokenUsage struct {
	ID               int64     `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	Project          string    `json:"project"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
}

var modelPricing = map[string]float64{
	"deepseek-chat":              0.00027,
	"gpt-4o":                     0.005,
	"claude-3-5-sonnet-20241022": 0.003,
	"gemini-2.0-flash":           0.0001,
}

type TokenCounter struct {
	mu   sync.Mutex
	path string
}

func NewTokenCounter() *TokenCounter {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".ds-ac", "usage.jsonl")
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	return &TokenCounter{path: p}
}

func (tc *TokenCounter) EstimateTokens(text string) int {
	return len(text)/4 + 1
}

func (tc *TokenCounter) CalculateCost(model string, prompt, completion int) float64 {
	price, ok := modelPricing[model]
	if !ok {
		price = 0.001
	}
	return float64(prompt+completion) / 1000.0 * price
}

func (tc *TokenCounter) Track(project, chatID, issueID, model string, promptTokens, completionTokens int) TokenUsage {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	usage := TokenUsage{
		ID:               time.Now().UnixNano(),
		Timestamp:        time.Now(),
		Project:          project,
		Model:            model,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		CostUSD:          tc.CalculateCost(model, promptTokens, completionTokens),
	}
	f, _ := os.OpenFile(tc.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil {
		defer f.Close()
		b, _ := json.Marshal(usage)
		f.Write(append(b, '\n'))
	}
	return usage
}

func (tc *TokenCounter) GetTotalCost(project string, since time.Time) float64 {
	data, err := os.ReadFile(tc.path)
	if err != nil {
		return 0
	}
	var total float64
	for _, line := range splitLines(string(data)) {
		if line == "" {
			continue
		}
		var u TokenUsage
		if json.Unmarshal([]byte(line), &u) != nil {
			continue
		}
		if project != "" && u.Project != project {
			continue
		}
		if !since.IsZero() && u.Timestamp.Before(since) {
			continue
		}
		total += u.CostUSD
	}
	return total
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
