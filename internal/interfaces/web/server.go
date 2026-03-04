package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	rt "github.com/dev/agent-runtime/internal/runtime"
	"github.com/dev/agent-runtime/internal/storage"
)

type Server struct {
	rt    *rt.Runtime
	store *storage.Storage
	port  string
	start time.Time
}

func NewServer(runtime *rt.Runtime, store *storage.Storage, port string) *Server {
	return &Server{rt: runtime, store: store, port: port, start: time.Now()}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/chat", s.handleChat)
	http.HandleFunc("/api/logs", s.handleLogs)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/settings", s.handleSettings)
	fmt.Printf("Web server listening on http://0.0.0.0:%s\n", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if req.SessionID == "" {
		req.SessionID = "web-default"
	}
	reply, _ := s.rt.ProcessMessage(req.SessionID, req.Message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.GetRecentToolLogs(50)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if logs == nil {
		logs = []storage.ToolLog{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats, _ := s.store.GetStats()

	hostname, _ := os.Hostname()
	status := map[string]interface{}{
		"hostname":        hostname,
		"uptime_seconds":  int(time.Since(s.start).Seconds()),
		"go_version":      runtime.Version(),
		"os_arch":         runtime.GOOS + "/" + runtime.GOARCH,
		"goroutines":      runtime.NumGoroutine(),
		"mem_alloc_mb":    float64(m.Alloc) / 1024 / 1024,
		"mem_sys_mb":      float64(m.Sys) / 1024 / 1024,
		"db_stats":        stats,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		settings, err := s.store.GetAllSettings()
		if err != nil {
			settings = make(map[string]string)
		}
		// Merge with env defaults (don't expose actual secrets, show masked)
		defaults := map[string]string{
			"telegram_token":    maskSecret(os.Getenv("TELEGRAM_TOKEN")),
			"telegram_allow_id": os.Getenv("TELEGRAM_ALLOW_ID"),
			"zai_endpoint":      os.Getenv("ZAI_ENDPOINT"),
			"zai_api_key":       maskSecret(os.Getenv("ZAI_API_KEY")),
			"workspace_root":    os.Getenv("WORKSPACE_ROOT"),
			"model":             "glm-5",
			"max_history":       fmt.Sprintf("%d", 25),
			"max_turns":         fmt.Sprintf("%d", 50),
		}
		for k, v := range defaults {
			if _, exists := settings[k]; !exists {
				settings[k] = v
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)

	case "POST":
		var settings map[string]string
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := s.store.SaveAllSettings(settings); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})

	default:
		http.Error(w, "method not allowed", 405)
	}
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "••••••••"
	}
	return s[:4] + "••••" + s[len(s)-4:]
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

const indexHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Cronos — Agentic Runtime</title>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
<style>
:root {
  --bg-primary: #0a0e17;
  --bg-secondary: #111827;
  --bg-card: #1a2332;
  --bg-card-hover: #1f2b3d;
  --bg-input: #0d1420;
  --border: #1e2d3d;
  --border-active: #3b82f6;
  --text-primary: #e2e8f0;
  --text-secondary: #94a3b8;
  --text-muted: #64748b;
  --accent: #3b82f6;
  --accent-glow: rgba(59,130,246,0.3);
  --success: #10b981;
  --warning: #f59e0b;
  --error: #ef4444;
  --gradient-1: linear-gradient(135deg, #3b82f6, #8b5cf6);
  --gradient-2: linear-gradient(135deg, #06b6d4, #3b82f6);
  --glass: rgba(17,24,39,0.8);
  --shadow: 0 8px 32px rgba(0,0,0,0.4);
  --radius: 12px;
  --transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

* { margin:0; padding:0; box-sizing:border-box; }
body {
  font-family: 'Inter', sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  height: 100vh;
  overflow: hidden;
  display: flex;
}

/* Sidebar */
.sidebar {
  width: 72px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
  gap: 8px;
  z-index: 10;
}

.sidebar .logo {
  width: 44px;
  height: 44px;
  background: var(--gradient-1);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 18px;
  margin-bottom: 20px;
  box-shadow: 0 4px 15px var(--accent-glow);
}

.nav-btn {
  width: 48px;
  height: 48px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  border-radius: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--transition);
  font-size: 20px;
}

.nav-btn:hover { background: var(--bg-card); color: var(--text-primary); }
.nav-btn.active { background: var(--accent); color: white; box-shadow: 0 4px 15px var(--accent-glow); }

.sidebar-spacer { flex: 1; }

/* Main Content */
.main { flex:1; display:flex; flex-direction:column; overflow:hidden; }

.page { display:none; flex:1; flex-direction:column; overflow:hidden; }
.page.active { display:flex; }

/* Header */
.header {
  padding: 20px 28px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.header h1 {
  font-size: 20px;
  font-weight: 600;
  background: var(--gradient-1);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.header .badge {
  font-size: 11px;
  padding: 4px 10px;
  border-radius: 20px;
  font-weight: 500;
}

.badge-online { background: rgba(16,185,129,0.15); color: var(--success); }

/* Chat Page */
.chat-container { flex:1; display:flex; flex-direction:column; overflow:hidden; }

.messages {
  flex:1;
  overflow-y: auto;
  padding: 24px 28px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.messages::-webkit-scrollbar { width: 6px; }
.messages::-webkit-scrollbar-track { background: transparent; }
.messages::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }

.msg {
  max-width: 80%;
  padding: 14px 18px;
  border-radius: 16px;
  font-size: 14px;
  line-height: 1.6;
  animation: fadeIn 0.3s ease;
  word-wrap: break-word;
  white-space: pre-wrap;
}

@keyframes fadeIn {
  from { opacity:0; transform: translateY(8px); }
  to { opacity:1; transform: translateY(0); }
}

.msg-user {
  align-self: flex-end;
  background: var(--accent);
  color: white;
  border-bottom-right-radius: 4px;
}

.msg-assistant {
  align-self: flex-start;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-bottom-left-radius: 4px;
}

.msg-time {
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 6px;
  opacity: 0.7;
}

.msg-user .msg-time { color: rgba(255,255,255,0.6); }

.chat-input-area {
  padding: 16px 28px 24px;
  border-top: 1px solid var(--border);
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.chat-input-area textarea {
  flex: 1;
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-primary);
  border-radius: var(--radius);
  padding: 14px 18px;
  font-family: 'Inter', sans-serif;
  font-size: 14px;
  resize: none;
  height: 52px;
  max-height: 120px;
  transition: var(--transition);
  outline: none;
}

.chat-input-area textarea:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-glow); }

.send-btn {
  width: 52px;
  height: 52px;
  background: var(--gradient-1);
  border: none;
  border-radius: var(--radius);
  color: white;
  font-size: 20px;
  cursor: pointer;
  transition: var(--transition);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.send-btn:hover { transform: scale(1.05); box-shadow: 0 4px 15px var(--accent-glow); }
.send-btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }

/* Settings Page */
.settings-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 24px 28px;
}

.settings-section {
  margin-bottom: 28px;
}

.settings-section h2 {
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: var(--text-muted);
  margin-bottom: 16px;
}

.setting-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 20px;
  margin-bottom: 12px;
  transition: var(--transition);
}

.setting-card:hover { border-color: var(--border-active); }

.setting-card label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.setting-card input, .setting-card select {
  width: 100%;
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-primary);
  border-radius: 8px;
  padding: 10px 14px;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  outline: none;
  transition: var(--transition);
}

.setting-card input:focus, .setting-card select:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-glow);
}

.save-btn {
  background: var(--gradient-1);
  border: none;
  color: white;
  padding: 12px 32px;
  border-radius: var(--radius);
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: var(--transition);
  margin-top: 8px;
}

.save-btn:hover { transform: translateY(-2px); box-shadow: var(--shadow); }

.toast {
  position: fixed;
  bottom: 24px;
  right: 24px;
  padding: 12px 24px;
  border-radius: var(--radius);
  font-size: 14px;
  font-weight: 500;
  animation: slideUp 0.3s ease;
  z-index: 1000;
  box-shadow: var(--shadow);
}

.toast-success { background: var(--success); color: white; }
.toast-error { background: var(--error); color: white; }

@keyframes slideUp {
  from { opacity:0; transform: translateY(20px); }
  to { opacity:1; transform: translateY(0); }
}

/* Logs Page */
.logs-scroll { flex:1; overflow-y:auto; padding:24px 28px; }

.log-entry {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 14px 18px;
  margin-bottom: 10px;
  transition: var(--transition);
  cursor: pointer;
}

.log-entry:hover { border-color: var(--border-active); }

.log-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
}

.log-tool {
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  font-weight: 500;
  color: var(--accent);
}

.log-status {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 600;
}

.log-status-ok { background: rgba(16,185,129,0.15); color: var(--success); }
.log-status-error { background: rgba(239,68,68,0.15); color: var(--error); }

.log-time {
  font-size: 11px;
  color: var(--text-muted);
  margin-left: auto;
}

.log-detail {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.5;
  max-height: 0;
  overflow: hidden;
  transition: max-height 0.3s ease;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-entry.expanded .log-detail { max-height: 400px; overflow-y:auto; padding-top:10px; border-top:1px solid var(--border); margin-top:8px; }

/* Status Page */
.status-scroll { flex:1; overflow-y:auto; padding:24px 28px; }

.status-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
  margin-bottom: 28px;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px;
  transition: var(--transition);
}

.stat-card:hover { border-color: var(--border-active); transform: translateY(-2px); box-shadow: var(--shadow); }

.stat-label {
  font-size: 12px;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-muted);
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  background: var(--gradient-2);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.stat-sub {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 4px;
}

/* Loading spinner */
.spinner {
  display: inline-block;
  width: 18px;
  height: 18px;
  border: 2px solid rgba(255,255,255,0.3);
  border-radius: 50%;
  border-top-color: white;
  animation: spin 0.8s linear infinite;
}

@keyframes spin { to { transform: rotate(360deg); } }

/* Responsive */
@media (max-width: 768px) {
  .sidebar { width: 56px; }
  .messages { padding: 16px; }
  .msg { max-width: 90%; }
  .status-grid { grid-template-columns: 1fr 1fr; }
}
</style>
</head>
<body>

<aside class="sidebar">
  <div class="logo">C</div>
  <button class="nav-btn active" onclick="showPage('chat')" title="Chat">💬</button>
  <button class="nav-btn" onclick="showPage('settings')" title="Settings">⚙️</button>
  <button class="nav-btn" onclick="showPage('logs')" title="Logs">📋</button>
  <button class="nav-btn" onclick="showPage('status')" title="Status">📊</button>
  <div class="sidebar-spacer"></div>
</aside>

<main class="main">

  <!-- CHAT -->
  <div id="page-chat" class="page active">
    <div class="header">
      <h1>Cronos — Chat</h1>
      <span class="badge badge-online">● Online</span>
    </div>
    <div class="chat-container">
      <div class="messages" id="messages"></div>
      <div class="chat-input-area">
        <textarea id="chatInput" placeholder="Envie uma mensagem para o Cronos..." rows="1"
          onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMessage()}"></textarea>
        <button class="send-btn" id="sendBtn" onclick="sendMessage()">➤</button>
      </div>
    </div>
  </div>

  <!-- SETTINGS -->
  <div id="page-settings" class="page">
    <div class="header">
      <h1>Configurações</h1>
      <button class="save-btn" onclick="saveSettings()">Salvar</button>
    </div>
    <div class="settings-scroll">
      <div class="settings-section">
        <h2>LLM / API</h2>
        <div class="setting-card">
          <label>Z.AI Endpoint</label>
          <input type="text" id="set-zai_endpoint" placeholder="https://api.z.ai/...">
        </div>
        <div class="setting-card">
          <label>Z.AI API Key</label>
          <input type="password" id="set-zai_api_key" placeholder="Sua chave API">
        </div>
        <div class="setting-card">
          <label>Modelo LLM</label>
          <input type="text" id="set-model" placeholder="glm-5">
        </div>
      </div>
      <div class="settings-section">
        <h2>Telegram</h2>
        <div class="setting-card">
          <label>Token do Bot</label>
          <input type="password" id="set-telegram_token" placeholder="123456:ABC-xyz...">
        </div>
        <div class="setting-card">
          <label>Allow ID (Chat ID autorizado)</label>
          <input type="text" id="set-telegram_allow_id" placeholder="@username ou ID numerico">
        </div>
      </div>
      <div class="settings-section">
        <h2>Runtime</h2>
        <div class="setting-card">
          <label>Workspace Root</label>
          <input type="text" id="set-workspace_root" placeholder="/home/user/projects">
        </div>
        <div class="setting-card">
          <label>Max Historico (mensagens no contexto)</label>
          <input type="number" id="set-max_history" placeholder="25">
        </div>
        <div class="setting-card">
          <label>Max Turnos (limite do loop agentico)</label>
          <input type="number" id="set-max_turns" placeholder="50">
        </div>
      </div>
    </div>
  </div>

  <!-- LOGS -->
  <div id="page-logs" class="page">
    <div class="header">
      <h1>Tool Logs</h1>
      <button class="save-btn" onclick="loadLogs()" style="font-size:13px;padding:8px 20px;">Atualizar</button>
    </div>
    <div class="logs-scroll" id="logsContainer">
      <p style="color:var(--text-muted);text-align:center;padding:40px;">Carregando...</p>
    </div>
  </div>

  <!-- STATUS -->
  <div id="page-status" class="page">
    <div class="header">
      <h1>Status do Sistema</h1>
      <button class="save-btn" onclick="loadStatus()" style="font-size:13px;padding:8px 20px;">Atualizar</button>
    </div>
    <div class="status-scroll" id="statusContainer">
      <p style="color:var(--text-muted);text-align:center;padding:40px;">Carregando...</p>
    </div>
  </div>

</main>

<script>
// Navigation
function showPage(name) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.nav-btn').forEach(b => b.classList.remove('active'));
  document.getElementById('page-' + name).classList.add('active');
  event.target.closest('.nav-btn').classList.add('active');
  if (name === 'logs') loadLogs();
  if (name === 'status') loadStatus();
  if (name === 'settings') loadSettings();
}

// Chat
async function sendMessage() {
  const input = document.getElementById('chatInput');
  const text = input.value.trim();
  if (!text) return;

  appendMessage('user', text);
  input.value = '';
  input.style.height = '52px';

  const btn = document.getElementById('sendBtn');
  btn.disabled = true;
  btn.innerHTML = '<span class="spinner"></span>';

  try {
    const res = await fetch('/api/chat', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({session_id: 'web-default', message: text})
    });
    const data = await res.json();
    appendMessage('assistant', data.reply);
  } catch(e) {
    appendMessage('assistant', 'Erro de conexao: ' + e.message);
  }

  btn.disabled = false;
  btn.innerHTML = '➤';
}

function appendMessage(role, text) {
  const container = document.getElementById('messages');
  const div = document.createElement('div');
  div.className = 'msg msg-' + role;
  const time = new Date().toLocaleTimeString('pt-BR', {hour:'2-digit',minute:'2-digit'});
  div.innerHTML = escapeHtml(text) + '<div class="msg-time">' + time + '</div>';
  container.appendChild(div);
  container.scrollTop = container.scrollHeight;
}

function escapeHtml(t) {
  return t.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/\n/g,'<br>');
}

// Settings
async function loadSettings() {
  try {
    const res = await fetch('/api/settings');
    const data = await res.json();
    const fields = ['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns'];
    fields.forEach(f => {
      const el = document.getElementById('set-' + f);
      if (el && data[f]) el.value = data[f];
    });
  } catch(e) {}
}

async function saveSettings() {
  const fields = ['zai_endpoint','zai_api_key','model','telegram_token','telegram_allow_id','workspace_root','max_history','max_turns'];
  const settings = {};
  fields.forEach(f => {
    const el = document.getElementById('set-' + f);
    if (el && el.value) settings[f] = el.value;
  });

  try {
    await fetch('/api/settings', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(settings)
    });
    showToast('Configuracoes salvas com sucesso!', 'success');
  } catch(e) {
    showToast('Erro ao salvar: ' + e.message, 'error');
  }
}

// Logs
async function loadLogs() {
  try {
    const res = await fetch('/api/logs');
    const logs = await res.json();
    const c = document.getElementById('logsContainer');
    if (!logs || logs.length === 0) {
      c.innerHTML = '<p style="color:var(--text-muted);text-align:center;padding:40px;">Nenhum log registrado ainda.</p>';
      return;
    }
    c.innerHTML = logs.map(l => {
      const statusClass = l.status === 'OK' ? 'ok' : 'error';
      const inp = (l.input || '').substring(0, 200);
      const out = (l.output || '').substring(0, 500);
      return '<div class="log-entry" onclick="this.classList.toggle(\'expanded\')">' +
        '<div class="log-header">' +
          '<span class="log-tool">' + escapeHtml(l.tool_name) + '</span>' +
          '<span class="log-status log-status-' + statusClass + '">' + l.status + '</span>' +
          '<span class="log-time">' + l.created_at + '</span>' +
        '</div>' +
        '<div class="log-detail">INPUT:\n' + escapeHtml(inp) + '\n\nOUTPUT:\n' + escapeHtml(out) + '</div>' +
      '</div>';
    }).join('');
  } catch(e) {
    document.getElementById('logsContainer').innerHTML = '<p style="color:var(--error);">Erro: ' + e.message + '</p>';
  }
}

// Status
async function loadStatus() {
  try {
    const res = await fetch('/api/status');
    const s = await res.json();
    const uptime = formatUptime(s.uptime_seconds);
    const db = s.db_stats || {};
    document.getElementById('statusContainer').innerHTML =
      '<div class="status-grid">' +
        statCard('Hostname', s.hostname, s.os_arch) +
        statCard('Uptime', uptime, 'Desde o inicio') +
        statCard('Memoria', (s.mem_alloc_mb || 0).toFixed(1) + ' MB', (s.mem_sys_mb || 0).toFixed(1) + ' MB total') +
        statCard('Goroutines', s.goroutines, 'Go ' + s.go_version) +
        statCard('Mensagens', db.total_messages || 0, (db.total_sessions || 0) + ' sessoes') +
        statCard('Execucoes', db.total_tool_executions || 0, (db.successful_executions || 0) + ' com sucesso') +
      '</div>';
  } catch(e) {
    document.getElementById('statusContainer').innerHTML = '<p style="color:var(--error);">Erro: ' + e.message + '</p>';
  }
}

function statCard(label, value, sub) {
  return '<div class="stat-card"><div class="stat-label">' + label + '</div><div class="stat-value">' + value + '</div><div class="stat-sub">' + sub + '</div></div>';
}

function formatUptime(s) {
  const h = Math.floor(s/3600);
  const m = Math.floor((s%3600)/60);
  return h + 'h ' + m + 'm';
}

function showToast(msg, type) {
  const t = document.createElement('div');
  t.className = 'toast toast-' + type;
  t.textContent = msg;
  document.body.appendChild(t);
  setTimeout(() => t.remove(), 3000);
}

// Auto-resize textarea
document.getElementById('chatInput').addEventListener('input', function() {
  this.style.height = '52px';
  this.style.height = Math.min(this.scrollHeight, 120) + 'px';
});

// Welcome
appendMessage('assistant', 'Ola! Eu sou o Cronos, seu assistente agentico. Como posso ajudar?');
</script>
</body>
</html>`
