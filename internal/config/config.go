package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	LLM_DEEPSEEK = 1
	LLM_META     = 2
	LLM_GEMINI   = 3
	LLM_CLAUDE   = 4
	LLM_OPENAI   = 5
	LLM_GROK     = 6
)

type LLMProvider struct {
	Enabled bool   `json:"enabled"`
	APIKey  string `json:"api_key,omitempty"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url,omitempty"`
}

type Config struct {
	// Fixa - vem do .env
	LlmClient      int    `json:"llm_client"`
	DeepSeekApiKey string `json:"deepseek_api_key"`
	MetaApiKey     string `json:"meta_api_key"`
	GeminiApiKey   string `json:"gemini_api_key"`
	ClaudeApiKey   string `json:"claude_api_key"`
	OpenAIApiKey   string `json:"openai_api_key"`
	GrokApiKey     string `json:"grok_api_key"`
	HttpPort       int    `json:"http_port"`

	LLMProviders map[string]LLMProvider `json:"llm_providers,omitempty"`

	// Dinâmica - vem da UI e é salva em ~/.ds-ac/config.json, mas também pode vir do .env
	IssuesDir       string `json:"issues_dir"`
	BaseJsonPath    string `json:"base_path"`
	ProjetoJsonPath string `json:"projeto_path"`

	// GitAutoBranch cria branch dsac/... antes de rodar issue (default true).
	GitAutoBranch bool `json:"git_auto_branch"`

	// interno
	configFilePath string `json:"-"`
}

// uiPersist é o que fica em ~/.ds-ac/config.json (UI + runtime)
type uiPersist struct {
	IssuesDir      string `json:"issues_dir"`
	LlmClient      int    `json:"llm_client,omitempty"`
	DeepSeekApiKey string `json:"deepseek_api_key,omitempty"`
	MetaApiKey     string `json:"meta_api_key,omitempty"`
	GeminiApiKey   string `json:"gemini_api_key,omitempty"`
	CloudeApiKey   string `json:"Cloude_api_key,omitempty"`
	OpenAIApiKey   string `json:"openai_api_key,omitempty"`
	GrokApiKey     string `json:"grok_api_key,omitempty"`

	HttpPort int `json:"http_port,omitempty"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ds-ac", "config.json")
}

func NewConfig() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		LlmClient:       getEnvAsInt("LLM_CLIENT", LLM_DEEPSEEK),
		DeepSeekApiKey:  getEnv("DEEPSEEK_API_KEY", ""),
		MetaApiKey:      getEnv("META_API_KEY", ""),
		GeminiApiKey:    getEnv("GEMINI_API_KEY", ""),
		ClaudeApiKey:    getEnv("CLAUDE_API_KEY", ""),
		OpenAIApiKey:    getEnv("OPENAI_API_KEY", ""),
		GrokApiKey:      firstNonEmpty(getEnv("GROK_API_KEY", ""), getEnv("XAI_API_KEY", "")),
		HttpPort:        getEnvAsInt("HTTP_PORT", 8080),
		IssuesDir:       getEnv("ISSUES_DIR", "/home/opc/prj/issues"),
		BaseJsonPath:    getEnv("DS_AC_BASE", "/home/opc/prj/issues/base.json"),
		ProjetoJsonPath: getEnv("DS_AC_PROJETO", ""),
		GitAutoBranch:   getEnv("DSAC_GIT_AUTO_BRANCH", "true") != "false",
		configFilePath:  configPath(),
	}

	// Fallback: se ~/.ds-ac/config.json existir, usa pra preencher o que não veio do .env
	// Isso mantém sua UI funcionando quando o usuário salva pelo modal ⚙
	if data, err := os.ReadFile(cfg.configFilePath); err == nil {
		var persisted uiPersist
		if json.Unmarshal(data, &persisted) == nil {
			if cfg.IssuesDir == "/home/opc/prj/issues" && persisted.IssuesDir != "" {
				// só sobrescreve se o .env não definiu um custom
				if getEnv("ISSUES_DIR", "") == "" {
					cfg.IssuesDir = persisted.IssuesDir
				}
			}
			if cfg.DeepSeekApiKey == "" && persisted.DeepSeekApiKey != "" {
				cfg.DeepSeekApiKey = persisted.DeepSeekApiKey
			}
			if cfg.MetaApiKey == "" && persisted.MetaApiKey != "" {
				cfg.MetaApiKey = persisted.MetaApiKey
			}
			if cfg.GeminiApiKey == "" && persisted.GeminiApiKey != "" {
				cfg.GeminiApiKey = persisted.GeminiApiKey
			}
			if cfg.ClaudeApiKey == "" && persisted.CloudeApiKey != "" {
				cfg.ClaudeApiKey = persisted.CloudeApiKey
			}
			if cfg.OpenAIApiKey == "" && persisted.OpenAIApiKey != "" {
				cfg.OpenAIApiKey = persisted.OpenAIApiKey
			}
			if cfg.GrokApiKey == "" && persisted.GrokApiKey != "" {
				cfg.GrokApiKey = persisted.GrokApiKey
			}

			if persisted.LlmClient != 0 && getEnv("LLM_CLIENT", "") == "" {
				cfg.LlmClient = persisted.LlmClient
			}
			if persisted.HttpPort != 0 && getEnv("HTTP_PORT", "") == "" {
				cfg.HttpPort = persisted.HttpPort
			}
		}
	}

	// workspace.json pode sobrescrever IssuesDir
	if w := LoadWorkspace(); w.IssuesDir != "" {
		cfg.IssuesDir = w.IssuesDir
	}
	return cfg
}

func (c *Config) ResolveAPIKey() string {
	switch c.LlmClient {
	case LLM_META:
		if c.MetaApiKey != "" {
			return c.MetaApiKey
		}
		return c.DeepSeekApiKey
	case LLM_GEMINI:
		if c.GeminiApiKey != "" {
			return c.GeminiApiKey
		}
		return c.DeepSeekApiKey
	case LLM_DEEPSEEK:
		if c.DeepSeekApiKey != "" {
			return c.DeepSeekApiKey
		}
		return c.MetaApiKey
	case LLM_CLAUDE:
		if c.ClaudeApiKey != "" {
			return c.ClaudeApiKey
		}
		return c.DeepSeekApiKey
	case LLM_OPENAI:
		if c.OpenAIApiKey != "" {
			return c.OpenAIApiKey
		}
		return c.DeepSeekApiKey
	case LLM_GROK:
		if c.GrokApiKey != "" {
			return c.GrokApiKey
		}
		return c.DeepSeekApiKey
	default:
		if c.DeepSeekApiKey != "" {
			return c.DeepSeekApiKey
		}
		if c.MetaApiKey != "" {
			return c.MetaApiKey
		}
		return c.GeminiApiKey
	}
}

func (c *Config) Save() error {
	// Salva em ~/.ds-ac/config.json pra UI continuar funcionando
	if err := os.MkdirAll(filepath.Dir(c.configFilePath), 0755); err != nil {
		return err
	}
	persist := uiPersist{
		IssuesDir:      c.IssuesDir,
		LlmClient:      c.LlmClient,
		DeepSeekApiKey: c.DeepSeekApiKey,
		MetaApiKey:     c.MetaApiKey,
		GeminiApiKey:   c.GeminiApiKey,
		CloudeApiKey:   c.ClaudeApiKey,
		OpenAIApiKey:   c.OpenAIApiKey,
		GrokApiKey:     c.GrokApiKey,
		HttpPort:       c.HttpPort,
	}
	data, err := json.MarshalIndent(persist, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configFilePath, data, 0644)
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if iv, err := strconv.Atoi(v); err == nil {
			return iv
		}
	}
	return defaultValue
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
