# 🤖 Agent Runtime

A lightweight, autonomous agentic runtime written in Go, designed to run on low-power devices like Raspberry Pi. It acts as an intelligent development agent that plans and executes tasks using external LLMs, with **Web** and **Telegram** interfaces.

---

## ✨ Features

- **🧠 Multi-Turn Agentic Loop** — Executes tools, feeds results back to the LLM, and loops until the task is done
- **🔧 Native Tool Calling** — Uses the LLM's function calling API (OpenAI-compatible)
- **� Multi-Model Failover** — Configure multiple LLM providers with automatic failover and priority-based routing
- **🎛️ Session Management** — Per-session model switch, think levels (off/low/medium/high), verbose mode, session compaction
- **🤝 Agent-to-Agent** — `sessions_list`, `sessions_history`, `sessions_send` tools for inter-session coordination
- **🚀 Onboarding Wizard** — Interactive 4-step setup wizard with real-time LLM connection validation
- **💬 Web Interface** — Full dashboard with Chat, Projects, Providers, Settings, Logs, and System Status
- **📁 Project Management** — Auto-scan workspace, Git operations (commit/push/pull/branch), project notes
- **📱 Telegram Bot** — Interactive chat via Telegram
- **🧩 Memory Agent** — Retrieves relevant context from past conversations
- **💾 SQLite** — Persists messages, tool executions, settings, and projects
- **🌍 Multi-language UI** — English, Portuguese, Spanish, French, German, Japanese, Chinese
- **🔓 Unrestricted** — Full system access, no allowlists or sandboxing
- **⚡ Lightweight** — Single binary ~10MB, <50MB RAM typical usage

---

## 📦 Project Structure

```
agent-runtime/
├── cmd/agent/main.go              # Entry point
├── internal/
│   ├── config/config.go           # Configuration (env vars)
│   ├── interfaces/
│   │   ├── telegram/bot.go        # Telegram bot
│   │   └── web/
│   │       ├── server.go          # HTTP server + API endpoints
│   │       └── ui.go              # Embedded web UI
│   ├── memory/agent.go            # Memory Agent (RAG via LLM)
│   ├── planner/
│   │   ├── client.go              # LLM client with native tool calling
│   │   └── provider.go            # Multi-model provider management & failover
│   ├── runtime/
│   │   ├── runtime.go             # Core agentic engine
│   │   ├── session.go             # Session management & per-session settings
│   │   ├── loop_detection.go      # Loop detection & circuit breaker
│   │   └── loop_detection_test.go # Loop detection tests
│   ├── storage/sqlite.go          # SQLite persistence
│   └── tools/
│       ├── registry.go            # Tool interface & registry
│       ├── echo.go                # Echo tool (testing)
│       ├── shell.go               # Shell execution (bash -c)
│       ├── workspace.go           # File listing & reading
│       ├── files.go               # File write/patch/delete
│       ├── delegate.go            # Sub-agent delegation
│       └── sessions.go            # Agent-to-agent coordination tools
├── prompts/
│   ├── soul.md                    # Agent identity & capabilities
│   ├── rules.md                   # Behavior rules
│   └── tools.md                   # Tool usage instructions
├── scripts/
│   ├── install.sh                 # Auto-install as systemd service
│   └── uninstall.sh               # Remove systemd service
├── .env.example                   # Configuration template
├── Makefile                       # Build & management commands
├── go.mod
└── README.md
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+**
- **GCC** (required for SQLite via cgo)
- An **OpenAI-compatible LLM API** (e.g., Z.AI, OpenAI, local models)
- *(Optional)* Telegram bot via [@BotFather](https://t.me/BotFather)

### Option 1: Automated Install (recommended)

```bash
git clone https://github.com/your-username/agent-runtime.git
cd agent-runtime

# This will: create .env, build, install as systemd service
make install
```

Then edit `.env` with your API keys:
```bash
nano .env
sudo systemctl restart agent-runtime
```

### Option 2: Manual Run

```bash
git clone https://github.com/your-username/agent-runtime.git
cd agent-runtime

cp .env.example .env
nano .env                    # Fill in your API keys

make run                     # Build and run locally
```

### Cross-compile for ARM64 (Raspberry Pi)

```bash
make build-pi
scp agent-runtime user@raspberry-pi:~/agent-runtime/
```

---

## 🔧 Makefile Commands

| Command | Description |
|---|---|
| `make build` | Build the binary |
| `make install` | Build + install as systemd service (auto-start on boot) |
| `make uninstall` | Remove systemd service |
| `make run` | Build and run locally (not as service) |
| `make start` | Start the service |
| `make stop` | Stop the service |
| `make restart` | Restart the service |
| `make status` | Check service status |
| `make logs` | Tail service logs |
| `make build-pi` | Cross-compile for Raspberry Pi (ARM64) |
| `make clean` | Remove built binary |

---

## ⚙️ Configuration

All configuration is via **environment variables**:

| Variable | Required | Default | Description |
|---|---|---|---|
| `ZAI_API_KEY` | ✅ | — | LLM API Key |
| `ZAI_ENDPOINT` | ✅ | — | LLM API endpoint (OpenAI-compatible) |
| `TELEGRAM_TOKEN` | ❌ | — | Telegram bot token |
| `TELEGRAM_ALLOW_ID` | ❌ | — | Authorized Telegram ID/username |
| `WORKSPACE_ROOT` | ❌ | `.` | Working directory for the agent |
| `PROMPTS_DIR` | ❌ | `.` | Directory containing prompt files |
| `DB_PATH` | ❌ | `agent.db` | SQLite database path |
| `PORT` | ❌ | `8080` | Web server port |
| `MAX_HISTORY` | ❌ | `25` | Messages in context (sliding window) |
| `MAX_TURNS` | ❌ | `12` | Max turns in agentic loop |
| `MAX_RUN_SECONDS` | ❌ | `0` | Optional hard wall-clock cutoff for one run (`0` disables) |
| `MAX_TOOL_CALLS` | ❌ | `20` | Max tool calls in one run |
| `LOOP_HISTORY_SIZE` | ❌ | `30` | Number of recent tool calls used for loop detection |
| `LOOP_WARN_AT` | ❌ | `10` | Warning threshold for repeated tool patterns |
| `LOOP_CRITICAL_AT` | ❌ | `20` | Critical threshold for ping-pong no-progress loops |
| `LOOP_GLOBAL_AT` | ❌ | `30` | Global breaker threshold for identical no-progress results |
| `AGENT_NAME` | ❌ | `Cronos` | Agent display name |
| `LANGUAGE` | ❌ | `en` | UI language (`en`, `pt-BR`, `es`, `fr`, `de`, `ja`, `zh`) |
| `MODELS` | ❌ | — | Multi-model providers (see below) |

### Example (Single Provider)

```bash
export ZAI_API_KEY="your-api-key"
export ZAI_ENDPOINT="https://api.example.com/v1/chat/completions"
export WORKSPACE_ROOT="/home/user"
export PROMPTS_DIR="/home/user/agent-runtime/prompts"
export AGENT_NAME="Jarvis"
export LANGUAGE="en"

./agent-runtime
```

### Multi-Model Configuration

Configure multiple providers with automatic failover using the `MODELS` env var.

**Format:** `id:name:endpoint:key:model:priority` separated by `||`

```bash
export MODELS="zai:ZhipuAI:https://api.z.ai/v1/chat/completions:sk-xxx:glm-5:1||openai:OpenAI:https://api.openai.com/v1/chat/completions:sk-yyy:gpt-4o:2||ollama:Ollama:http://localhost:11434/v1/chat/completions::llama3:3"
```

| Field | Description |
|---|---|
| `id` | Unique identifier for the provider |
| `name` | Display name |
| `endpoint` | OpenAI-compatible API endpoint |
| `key` | API key (empty for local models) |
| `model` | Model name to use |
| `priority` | Lower = preferred. Higher priority providers are used as fallback |

**Failover behavior:**
- Providers are tried in priority order
- On failure, automatic failover to the next available provider
- Exponential backoff cooldown (30s → 60s → 120s → max 5min)
- Legacy `ZAI_ENDPOINT`/`ZAI_API_KEY` is always added as fallback (priority 999)
- Per-session model override via Web UI

---

## 🏃 Usage

### Web Interface

Open `http://<host>:8080` in your browser.

Pages:
- **💬 Chat** — Converse with the agent (with think level, model selector, verbose toggle, compact button)
- **📁 Projects** — Manage projects, Git operations, notes
- **🔌 Providers** — View configured model providers, health status, and failover info
- **⚙️ Settings** — Configure API keys, tokens, model, language
- **📋 Logs** — Tool execution history with expandable details
- **📊 Status** — System health dashboard

**Chat Controls:**
- **Think Level** — Set reasoning depth: Off, Low, Medium, High
- **Model Selector** — Switch the LLM model for the current session
- **Verbose** — Toggle detailed tool execution output
- **Compact** — Summarize conversation history via LLM to reduce context size

### Telegram

Send messages directly to your bot. The agent processes and responds autonomously.

### API

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/chat` | Send message. Body: `{"session_id": "...", "message": "..."}` |
| `GET` | `/api/projects` | List projects |
| `POST` | `/api/projects` | Create project |
| `PUT` | `/api/projects` | Update project |
| `DELETE` | `/api/projects` | Delete project |
| `GET` | `/api/projects/scan` | Auto-detect projects in workspace |
| `GET` | `/api/projects/git?id=N` | Git info for project |
| `POST` | `/api/projects/git/action` | Git action (commit/push/pull/branch) |
| `GET/POST` | `/api/settings` | Read/write settings |
| `POST` | `/api/chat/compact` | Compact session history. Body: `{"session_id": "..."}` |
| `GET/POST` | `/api/session/settings` | Read/update per-session settings (model, think level, verbose) |
| `GET` | `/api/providers` | List configured model providers |
| `GET` | `/api/providers/status` | Provider health status with failover info |
| `POST` | `/api/onboarding/validate` | Test LLM connection. Body: `{"endpoint": "...", "api_key": "...", "model": "..."}` |
| `GET` | `/api/status` | System status |
| `GET` | `/api/logs` | Recent tool logs |
| `GET` | `/api/app-config` | Agent name, language, version |

---

## 🤝 Agent-to-Agent Coordination

Sessions can communicate with each other using built-in tools:

| Tool | Description |
|---|---|
| `sessions_list` | List all active sessions with metadata (message count, model, status) |
| `sessions_history` | Fetch the conversation transcript of another session |
| `sessions_send` | Send a message to another session and receive its reply |

This enables multi-agent workflows where one session can delegate tasks to another, check on progress, or share context.

---

## 🔧 Adding Tools

Implement the `Tool` interface:

```go
type Tool interface {
    Name() string
    Description() string
    Risk() string                // "LOW" or "HIGH"
    Parameters() []ToolParam
    Execute(ctx ToolContext, args map[string]string) (string, error)
}
```

Register in `main.go`:

```go
reg.Register(tools.NewMyTool())
```

---

## 🐳 Deploy as Service

The easiest way to deploy is via the included install script:

```bash
make install        # Build, create .env, install systemd service
nano .env           # Edit with your API keys
make restart        # Apply changes
```

To remove:
```bash
make uninstall
```

---

## 📄 License

MIT
