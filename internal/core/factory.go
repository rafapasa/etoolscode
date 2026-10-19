package core

import "github.com/rafapasa/etoolscode/internal/config"

func NewLlmClient(cfg config.Config) LlmInterface {
	switch cfg.LlmClient {
	case config.LLM_DEEPSEEK:
		return NewDeepSeekClient(cfg.DeepSeekApiKey)
	case config.LLM_META:
		return NewLlamaClient(cfg.MetaApiKey)
	case config.LLM_GEMINI:
		return NewGeminiClient(cfg.GeminiApiKey)
	case config.LLM_CLAUDE:
		return NewClaudeClient(cfg)
	case config.LLM_OPENAI:
		return NewOpenAIClient(cfg)
	case config.LLM_GROK:
		return NewGrokClient(cfg)
	default:
		if cfg.MetaApiKey != "" {
			return NewLlamaClient(cfg.MetaApiKey)
		}
		return NewDeepSeekClient(cfg.DeepSeekApiKey)
	}
}
