package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramToken   string
	TelegramAllowID string
	ZAIEndpoint     string
	ZAIApiKey       string
	WorkspaceRoot   string
	DBPath          string
	PromptsDir      string
	Port            string
	MaxHistory      int
	MaxTurns        int
	MaxRunSeconds   int
	MaxToolCalls    int
	LoopHistorySize int
	LoopWarnAt      int
	LoopCriticalAt  int
	LoopGlobalAt    int
	MaxAgentDepth   int
	AgentName       string
	Language        string
	// Multi-model: "id:name:endpoint:key:model:priority||..." or empty for legacy single-model
	Models string
	// New v1.2 features
	MCPConfigPath    string // Path to MCP servers JSON config file
	MaxContextTokens int    // Max tokens for context window management
	TokenBudget      int    // Max cumulative tokens per agentic run (0 = unlimited)
	EnableLSP        bool   // Enable LSP integration
	EnableWatcher    bool   // Enable file watcher
	EnableGitContext bool   // Enable git-aware context injection
	EnableMultimodal bool   // Enable multimodal (vision) support
}

func LoadConfig() *Config {
	return &Config{
		TelegramToken:   getEnv("TELEGRAM_TOKEN", ""),
		TelegramAllowID: getEnv("TELEGRAM_ALLOW_ID", ""),
		ZAIEndpoint:     getEnv("ZAI_ENDPOINT", ""),
		ZAIApiKey:       getEnv("ZAI_API_KEY", ""),
		WorkspaceRoot:   getEnv("WORKSPACE_ROOT", "."),
		DBPath:          getEnv("DB_PATH", "agent.db"),
		PromptsDir:      getEnv("PROMPTS_DIR", "."),
		Port:            getEnv("PORT", "8080"),
		MaxHistory:      getEnvInt("MAX_HISTORY", 25),
		MaxTurns:        getEnvInt("MAX_TURNS", 25),
		// Hard wall-clock cutoff to prevent runaway loops.
		MaxRunSeconds:   getEnvInt("MAX_RUN_SECONDS", 600),
		MaxToolCalls:    getEnvInt("MAX_TOOL_CALLS", 50),
		LoopHistorySize: getEnvInt("LOOP_HISTORY_SIZE", 40),
		LoopWarnAt:      getEnvInt("LOOP_WARN_AT", 10),
		LoopCriticalAt:  getEnvInt("LOOP_CRITICAL_AT", 20),
		LoopGlobalAt:    getEnvInt("LOOP_GLOBAL_AT", 30),
		MaxAgentDepth:   getEnvInt("MAX_AGENT_DEPTH", 3),
		AgentName:       getEnv("AGENT_NAME", "Cronos"),
		Language:        getEnv("LANGUAGE", "en"),
		Models:           getEnv("MODELS", ""),
		MCPConfigPath:    getEnv("MCP_CONFIG", ""),
		MaxContextTokens: getEnvInt("MAX_CONTEXT_TOKENS", 128000),
		TokenBudget:      getEnvInt("TOKEN_BUDGET", 500000),
		EnableLSP:        getEnvBool("ENABLE_LSP", false),
		EnableWatcher:    getEnvBool("ENABLE_WATCHER", true),
		EnableGitContext: getEnvBool("ENABLE_GIT_CONTEXT", true),
		EnableMultimodal: getEnvBool("ENABLE_MULTIMODAL", true),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		switch strings.ToLower(value) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		}
	}
	return fallback
}
