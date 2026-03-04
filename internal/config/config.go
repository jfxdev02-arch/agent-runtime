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
}

func LoadConfig() *Config {
	return &Config{
		TelegramToken:   getEnv("TELEGRAM_TOKEN", ""),
		TelegramAllowID: getEnv("TELEGRAM_ALLOW_ID", ""),
		ZAIEndpoint:     getEnv("ZAI_ENDPOINT", "http://localhost:8080/v1/chat/completions"),
		ZAIApiKey:       getEnv("ZAI_API_KEY", ""),
		WorkspaceRoot:   getEnv("WORKSPACE_ROOT", "."),
		DBPath:          getEnv("DB_PATH", "agent.db"),
		PromptsDir:      getEnv("PROMPTS_DIR", "."),
		Port:            getEnv("PORT", "8080"),
		MaxHistory:      getEnvInt("MAX_HISTORY", 25),
		MaxTurns:        getEnvInt("MAX_TURNS", 50),
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
