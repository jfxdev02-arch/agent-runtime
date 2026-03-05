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
	LoopHistorySize int
	LoopWarnAt      int
	LoopCriticalAt  int
	LoopGlobalAt    int
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
		// Hard wall-clock cutoff to prevent runaway loops.
		MaxRunSeconds:   getEnvInt("MAX_RUN_SECONDS", 300),
		MaxToolCalls:    getEnvInt("MAX_TOOL_CALLS", 20),
		LoopHistorySize: getEnvInt("LOOP_HISTORY_SIZE", 30),
		LoopWarnAt:      getEnvInt("LOOP_WARN_AT", 10),
		LoopCriticalAt:  getEnvInt("LOOP_CRITICAL_AT", 20),
		LoopGlobalAt:    getEnvInt("LOOP_GLOBAL_AT", 30),
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
