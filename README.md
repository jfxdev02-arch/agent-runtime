# 🤖 Cronos — Agentic Runtime

**Cronos** é um runtime agêntico autônomo escrito em Go, projetado para rodar em dispositivos de baixa potência como o **Raspberry Pi 3**. Ele atua como um agente de desenvolvimento inteligente que planeja e executa tarefas usando LLMs externos, com interfaces via **Web** e **Telegram**.

---

## ✨ Features

- **🧠 Loop Agêntico Multi-Turn** — Executa tools, alimenta resultados de volta ao LLM, e continua até concluir a tarefa
- **🔧 Tool Calling Nativo** — Usa a API de function calling do LLM (OpenAI-compatible) ao invés de parsear JSON do texto
- **💬 Interface Web** — Dashboard completo com Chat, Configurações, Logs e Status do sistema
- **📱 Bot Telegram** — Interaja com o agente direto pelo Telegram
- **🧩 Memory Agent** — Agente auxiliar que busca contexto relevante de conversas anteriores no banco de dados
- **💾 SQLite** — Persiste mensagens, execuções de tools, e configurações
- **🔓 Acesso Total** — Sem restrições de allowlist ou workspace; o agente tem liberdade total no sistema
- **⚡ Leve** — Binário único ~10MB, <50MB RAM em uso típico

---

## 📦 Estrutura do Projeto

```
agent-runtime/
├── cmd/
│   └── agent/
│       └── main.go              # Entrypoint principal
├── internal/
│   ├── config/
│   │   └── config.go            # Carregamento de configurações (env vars)
│   ├── interfaces/
│   │   ├── telegram/
│   │   │   └── bot.go           # Bot Telegram com long polling
│   │   └── web/
│   │       └── server.go        # Servidor HTTP + UI completa embutida
│   ├── memory/
│   │   └── agent.go             # Memory Agent (RAG via LLM)
│   ├── planner/
│   │   └── client.go            # Cliente LLM com tool calling nativo
│   ├── runtime/
│   │   ├── runtime.go           # Motor principal do agente
│   │   └── session.go           # Gerenciamento de sessões
│   ├── storage/
│   │   └── sqlite.go            # Persistência SQLite
│   └── tools/
│       ├── registry.go          # Interface e registro de tools
│       ├── echo.go              # Tool de teste
│       ├── shell.go             # Execução de comandos (bash -c)
│       ├── workspace.go         # Leitura/listagem de arquivos
│       └── files.go             # Escrita/patch/delete de arquivos
├── prompts/
│   ├── soul.md                  # Identidade e personalidade do agente
│   ├── rules.md                 # Regras de comportamento
│   └── tools.md                 # Instruções sobre uso de tools
├── go.mod
├── go.sum
└── README.md
```

---

## 🚀 Instalação

### Pré-requisitos

- **Go 1.21+** instalado
- **GCC** (necessário para o SQLite via cgo)
- Conta na **[Z.AI](https://z.ai)** para a API Key
- *(Opcional)* Bot do Telegram criado via [@BotFather](https://t.me/BotFather)

### No Raspberry Pi (ARM64)

```bash
# Instalar Go (se ainda não tiver)
sudo apt update && sudo apt install -y golang gcc git

# Clonar o repositório
git clone https://github.com/seu-usuario/agent-runtime.git
cd agent-runtime

# Instalar dependências
go mod tidy

# Compilar (otimizado para Pi)
go build -ldflags="-w -s" -o cronos cmd/agent/main.go
```

### Cross-compile do Windows/Mac para Pi

```bash
# Windows PowerShell
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED="1"
go build -ldflags="-w -s" -o cronos cmd/agent/main.go

# Mac/Linux
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="-w -s" -o cronos cmd/agent/main.go
```

> **Nota:** Cross-compile com CGO requer um cross-compiler ARM64. No Pi nativo é mais simples.

---

## ⚙️ Configuração

Todas as configurações são via **variáveis de ambiente**:

| Variável | Obrigatória | Default | Descrição |
|---|---|---|---|
| `ZAI_API_KEY` | ✅ | — | Chave de API da Z.AI |
| `ZAI_ENDPOINT` | ✅ | — | Endpoint da API (ex: `https://api.z.ai/api/coding/paas/v4/chat/completions`) |
| `TELEGRAM_TOKEN` | ❌ | — | Token do bot Telegram |
| `TELEGRAM_ALLOW_ID` | ❌ | — | ID ou @username autorizado no Telegram |
| `WORKSPACE_ROOT` | ❌ | `.` | Diretório raiz de trabalho do agente |
| `PROMPTS_DIR` | ❌ | `.` | Diretório com os arquivos de prompt |
| `DB_PATH` | ❌ | `agent.db` | Caminho do banco SQLite |
| `PORT` | ❌ | `8080` | Porta do servidor web |
| `MAX_HISTORY` | ❌ | `25` | Máximo de mensagens no contexto (sliding window) |
| `MAX_TURNS` | ❌ | `50` | Máximo de turnos no loop agêntico |

### Exemplo de configuração

```bash
export ZAI_API_KEY="sua-chave-aqui"
export ZAI_ENDPOINT="https://api.z.ai/api/coding/paas/v4/chat/completions"
export TELEGRAM_TOKEN="123456:ABC-xyz"
export TELEGRAM_ALLOW_ID="@seu_username"
export WORKSPACE_ROOT="/home/usuario"
export PROMPTS_DIR="/home/usuario/agent-runtime/prompts"
export MAX_TURNS=50
```

---

## 🏃 Uso

### Iniciar o agente

```bash
# Configurar variáveis e rodar
export ZAI_API_KEY="sua-chave"
export ZAI_ENDPOINT="https://api.z.ai/api/coding/paas/v4/chat/completions"
export WORKSPACE_ROOT="/home/usuario"
export PROMPTS_DIR="/home/usuario/agent-runtime/prompts"

./cronos
```

### Interface Web

Acesse `http://<ip-do-pi>:8080` no navegador.

A interface possui 4 seções:

- **💬 Chat** — Converse com o Cronos, peça para executar tarefas
- **⚙️ Settings** — Configure tokens, endpoints, modelo e limites
- **📋 Logs** — Veja o histórico de execuções de tools com input/output
- **📊 Status** — Monitor de saúde: uptime, memória, goroutines, estatísticas

### Telegram

Envie mensagens diretamente para o bot no Telegram. Comandos disponíveis:

- `/start` — Mensagem de boas-vindas
- Qualquer mensagem de texto — O agente processa e responde

### Exemplos de uso via chat

```
Você: liste os arquivos do meu diretório home
Cronos: [executa workspace list] ...

Você: crie um arquivo hello.py com um hello world
Cronos: [executa files write] Arquivo criado com sucesso!

Você: execute o comando python3 hello.py
Cronos: [executa shell] Hello, World!

Você: como voce se chama?
Cronos: Meu nome é Cronos, sou um assistente agêntico...
```

---

## 🧠 Arquitetura

### Fluxo de Execução

```
Usuário → [Telegram/Web]
              │
              ▼
         ┌─────────┐
         │ Runtime  │ ← Gerencia sessões e histórico
         └────┬─────┘
              │
    ┌─────────┼──────────┐
    ▼         ▼          ▼
 Memory    Planner    Tool Registry
 Agent     (LLM)      (shell, files, workspace, echo)
    │         │          │
    │    ┌────┴────┐     │
    │    │ tool_calls?   │
    │    └────┬────┘     │
    │    Sim  │   Não    │
    │    ┌────┘   └──── Resposta direta
    │    ▼
    │  Executa tools
    │    │
    │    ▼
    │  Envia resultados ao LLM
    │    │
    │    ▼
    └──► Loop até concluir (max N turnos)
```

### Memory Agent

Antes de cada chamada ao LLM principal, o **Memory Agent** faz uma chamada separada para buscar contexto relevante:

1. Busca até 100 mensagens antigas do SQLite (além da sliding window atual)
2. Envia ao LLM pedindo para selecionar apenas o que é relevante à pergunta atual
3. O resumo é injetado no system prompt como contexto adicional

Isso permite ao Cronos "lembrar" de conversas anteriores sem sobrecarregar o contexto.

### Tools Disponíveis

| Tool | Descrição | Risco |
|---|---|---|
| `echo` | Ecoa texto de volta (teste) | LOW |
| `shell` | Executa qualquer comando via `bash -c` | LOW |
| `workspace` | Lista diretórios e lê arquivos | LOW |
| `files` | Escreve, edita (patch), ou deleta arquivos | LOW |

### Modelo LLM

O Cronos usa o modelo **GLM-5** da Z.AI via API compatível com OpenAI, incluindo:

- **Tool/Function Calling** nativo
- **Temperature** baixa (0.2) para respostas mais determinísticas
- Autenticação via **Bearer Token**

---

## 🔧 Customização

### Prompts

Os arquivos em `prompts/` definem a personalidade e regras do agente:

- **`soul.md`** — Identidade, capacidades, e instruções gerais
- **`rules.md`** — Regras de comportamento
- **`tools.md`** — Instruções sobre uso de tools

Edite esses arquivos para customizar o comportamento do agente.

### Adicionando Tools

Para criar uma nova tool, crie um arquivo em `internal/tools/` implementando a interface:

```go
type Tool interface {
    Name() string
    Description() string
    Risk() string                // "LOW" ou "HIGH"
    Parameters() []ToolParam
    Execute(ctx ToolContext, args map[string]string) (string, error)
}
```

Registre no `main.go`:

```go
reg.Register(tools.NewSuaTool())
```

---

## 🐳 Deploy com Systemd

Para manter o Cronos rodando como serviço:

```bash
sudo tee /etc/systemd/system/cronos.service << EOF
[Unit]
Description=Cronos Agentic Runtime
After=network.target

[Service]
Type=simple
User=jotajota
WorkingDirectory=/home/jotajota/agent-runtime
EnvironmentFile=/home/jotajota/agent-runtime/.env
ExecStart=/home/jotajota/agent-runtime/cronos
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Criar arquivo .env
cat > /home/jotajota/agent-runtime/.env << EOF
ZAI_API_KEY=sua-chave
ZAI_ENDPOINT=https://api.z.ai/api/coding/paas/v4/chat/completions
TELEGRAM_TOKEN=seu-token
TELEGRAM_ALLOW_ID=@seu_user
WORKSPACE_ROOT=/home/jotajota
PROMPTS_DIR=/home/jotajota/agent-runtime/prompts
MAX_TURNS=50
PORT=8080
EOF

# Ativar e iniciar
sudo systemctl daemon-reload
sudo systemctl enable cronos
sudo systemctl start cronos

# Ver logs
sudo journalctl -u cronos -f
```

---

## 📊 API REST

| Método | Endpoint | Descrição |
|---|---|---|
| `POST` | `/api/chat` | Enviar mensagem ao agente. Body: `{"session_id": "...", "message": "..."}` |
| `GET` | `/api/logs` | Últimas 50 execuções de tools |
| `GET` | `/api/status` | Status do sistema (uptime, memória, stats) |
| `GET` | `/api/settings` | Configurações atuais |
| `POST` | `/api/settings` | Salvar configurações. Body: `{"key": "value", ...}` |

---

## 🤝 Inspiração

Arquitetura inspirada no [pi-mono](https://github.com/badlogic/pi-mono) — especialmente o loop agêntico multi-turn, tool calling nativo, e o conceito de contexto transformável.

---

## 📄 Licença

MIT

---

<p align="center">
  Feito com 💙 para rodar no Raspberry Pi 3
</p>
