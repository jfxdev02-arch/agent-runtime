package config

import (
	"os"
	"strconv"
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
	MaxToolRepeats  int
	MaxAgentDepth   int
	AgentName       string
	Language        string
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
		MaxTurns:        getEnvInt("MAX_TURNS", 12),
		MaxRunSeconds:   getEnvInt("MAX_RUN_SECONDS", 90),
		MaxToolCalls:    getEnvInt("MAX_TOOL_CALLS", 20),
		MaxToolRepeats:  getEnvInt("MAX_TOOL_REPEATS", 4),
		MaxAgentDepth:   getEnvInt("MAX_AGENT_DEPTH", 3),
		AgentName:       getEnv("AGENT_NAME", "Cronos"),
		Language:        getEnv("LANGUAGE", "en"),
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
