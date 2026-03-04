# 🤖 Agent Runtime

A lightweight, autonomous agentic runtime written in Go, designed to run on low-power devices like Raspberry Pi. It acts as an intelligent development agent that plans and executes tasks using external LLMs, with **Web** and **Telegram** interfaces.

---

## ✨ Features

- **🧠 Multi-Turn Agentic Loop** — Executes tools, feeds results back to the LLM, and loops until the task is done
- **🔧 Native Tool Calling** — Uses the LLM's function calling API (OpenAI-compatible)
- **💬 Web Interface** — Full dashboard with Chat, Projects, Settings, Logs, and System Status
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
│   ├── planner/client.go          # LLM client with native tool calling
│   ├── runtime/
│   │   ├── runtime.go             # Core agentic engine
│   │   └── session.go             # Session management
│   ├── storage/sqlite.go          # SQLite persistence
│   └── tools/
│       ├── registry.go            # Tool interface & registry
│       ├── echo.go                # Echo tool (testing)
│       ├── shell.go               # Shell execution (bash -c)
│       ├── workspace.go           # File listing & reading
│       └── files.go               # File write/patch/delete
├── prompts/
│   ├── soul.md                    # Agent identity & capabilities
│   ├── rules.md                   # Behavior rules
│   └── tools.md                   # Tool usage instructions
├── go.mod
└── README.md
```

---

## 🚀 Installation

### Prerequisites

- **Go 1.21+**
- **GCC** (required for SQLite via cgo)
- An **OpenAI-compatible LLM API** (e.g., Z.AI, OpenAI, local models)
- *(Optional)* Telegram bot via [@BotFather](https://t.me/BotFather)

### Build

```bash
# Clone
git clone https://github.com/your-username/agent-runtime.git
cd agent-runtime

# Install dependencies
go mod tidy

# Build (optimized)
go build -ldflags="-w -s" -o agent-runtime cmd/agent/main.go
```

### Cross-compile for ARM64 (Raspberry Pi)

```bash
GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o agent-runtime cmd/agent/main.go
```

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
| `MAX_TURNS` | ❌ | `50` | Max turns in agentic loop |
| `AGENT_NAME` | ❌ | `Cronos` | Agent display name |
| `LANGUAGE` | ❌ | `en` | UI language (`en`, `pt-BR`, `es`, `fr`, `de`, `ja`, `zh`) |

### Example

```bash
export ZAI_API_KEY="your-api-key"
export ZAI_ENDPOINT="https://api.example.com/v1/chat/completions"
export WORKSPACE_ROOT="/home/user"
export PROMPTS_DIR="/home/user/agent-runtime/prompts"
export AGENT_NAME="Jarvis"
export LANGUAGE="en"

./agent-runtime
```

---

## 🏃 Usage

### Web Interface

Open `http://<host>:8080` in your browser.

Pages:
- **💬 Chat** — Converse with the agent
- **📁 Projects** — Manage projects, Git operations, notes
- **⚙️ Settings** — Configure API keys, tokens, model, language
- **📋 Logs** — Tool execution history with expandable details
- **📊 Status** — System health dashboard

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
| `GET` | `/api/status` | System status |
| `GET` | `/api/logs` | Recent tool logs |
| `GET` | `/api/app-config` | Agent name, language, version |

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

## 🐳 Deploy as Service (systemd)

```bash
sudo tee /etc/systemd/system/agent-runtime.service << EOF
[Unit]
Description=Agent Runtime
After=network.target

[Service]
Type=simple
User=YOUR_USER
WorkingDirectory=/path/to/agent-runtime
EnvironmentFile=/path/to/agent-runtime/.env
ExecStart=/path/to/agent-runtime/agent-runtime
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable agent-runtime
sudo systemctl start agent-runtime
```

---

## 📄 License

MIT
