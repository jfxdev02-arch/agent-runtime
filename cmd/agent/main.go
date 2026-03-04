package main

import (
	"log"

	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/interfaces/telegram"
	"github.com/dev/agent-runtime/internal/interfaces/web"
	"github.com/dev/agent-runtime/internal/runtime"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/tools"
)

func main() {
	cfg := config.LoadConfig()

	store, err := storage.NewStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	reg := tools.NewRegistry()
	reg.Register(tools.NewEchoTool())
	reg.Register(tools.NewShellTool())
	reg.Register(tools.NewWorkspaceTool())
	reg.Register(tools.NewFilesTool())

	rt := runtime.NewRuntime(cfg, store, reg)

	bot := telegram.NewBot(cfg.TelegramToken, cfg.TelegramAllowID, rt)
	go bot.Start()

	srv := web.NewServer(rt, cfg.Port)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
