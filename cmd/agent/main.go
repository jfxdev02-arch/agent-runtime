package main

import (
	"log"

	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/interfaces/telegram"
	"github.com/dev/agent-runtime/internal/interfaces/web"
	"github.com/dev/agent-runtime/internal/orchestrator"
	"github.com/dev/agent-runtime/internal/planner"
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

	llm := planner.NewPlanner(cfg.ZAIEndpoint, cfg.ZAIApiKey)

	reg := tools.NewRegistry()
	reg.Register(tools.NewEchoTool())
	reg.Register(tools.NewShellTool())
	reg.Register(tools.NewWorkspaceTool())
	reg.Register(tools.NewFilesTool())

	orch := orchestrator.NewOrchestrator(cfg, store, reg, llm)
	reg.Register(tools.NewDelegateTool(orch))

	rt := runtime.NewRuntime(cfg, store, reg, llm)
	defer rt.Shutdown()

	// Load MCP servers if configured
	if cfg.MCPConfigPath != "" {
		rt.LoadMCPServersFromConfig(cfg.MCPConfigPath)
	}

	// Register sessions_* tools (agent-to-agent coordination)
	reg.Register(tools.NewSessionsListTool(rt))
	reg.Register(tools.NewSessionsHistoryTool(rt))
	reg.Register(tools.NewSessionsSendTool(rt))

	bot := telegram.NewBot(cfg.TelegramToken, cfg.TelegramAllowID, rt)
	go bot.Start()

	srv := web.NewServer(rt, store, cfg.Port)
	srv.SetConfig(cfg.AgentName, cfg.Language)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
