package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ServerConfig describes how to launch a language server.
type ServerConfig struct {
	Language string   `json:"language"`
	Command  string   `json:"command"`
	Args     []string `json:"args"`
}

// Diagnostic represents a single LSP diagnostic.
type Diagnostic struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"` // "error", "warning", "info", "hint"
	Message  string `json:"message"`
	Source   string `json:"source"`
}

// Manager manages multiple language server connections.
type Manager struct {
	servers  map[string]*serverConn
	mu       sync.RWMutex
	rootPath string
	configs  []ServerConfig
}

type serverConn struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan json.RawMessage
	diags   map[string][]Diagnostic
	diagMu  sync.RWMutex
}

// jsonrpcMessage is a JSON-RPC 2.0 message.
type jsonrpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *int64           `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *json.RawMessage `json:"error,omitempty"`
}

// DefaultConfigs returns standard language server configurations.
func DefaultConfigs() []ServerConfig {
	return []ServerConfig{
		{Language: "go", Command: "gopls", Args: []string{"serve"}},
		{Language: "python", Command: "pylsp", Args: nil},
		{Language: "typescript", Command: "typescript-language-server", Args: []string{"--stdio"}},
		{Language: "javascript", Command: "typescript-language-server", Args: []string{"--stdio"}},
		{Language: "rust", Command: "rust-analyzer", Args: nil},
	}
}

// NewManager creates a new LSP manager.
func NewManager(rootPath string, configs []ServerConfig) *Manager {
	if configs == nil {
		configs = DefaultConfigs()
	}
	return &Manager{
		servers:  make(map[string]*serverConn),
		rootPath: rootPath,
		configs:  configs,
	}
}

// Start attempts to launch language servers for detected languages.
func (m *Manager) Start() {
	for _, cfg := range m.configs {
		if _, err := exec.LookPath(cfg.Command); err != nil {
			continue
		}
		if err := m.startServer(cfg); err != nil {
			log.Printf("[lsp] Failed to start %s server: %v", cfg.Language, err)
		} else {
			log.Printf("[lsp] Started %s server (%s)", cfg.Language, cfg.Command)
		}
	}
}

// Stop shuts down all language servers.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for lang, sc := range m.servers {
		m.shutdownServer(sc)
		log.Printf("[lsp] Stopped %s server", lang)
	}
	m.servers = make(map[string]*serverConn)
}

// GetDiagnostics returns current diagnostics for a file.
func (m *Manager) GetDiagnostics(file string) []Diagnostic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []Diagnostic
	for _, sc := range m.servers {
		sc.diagMu.RLock()
		if diags, ok := sc.diags[file]; ok {
			all = append(all, diags...)
		}
		sc.diagMu.RUnlock()
	}
	return all
}

// GetAllDiagnostics returns all diagnostics across all servers.
func (m *Manager) GetAllDiagnostics() []Diagnostic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []Diagnostic
	for _, sc := range m.servers {
		sc.diagMu.RLock()
		for _, diags := range sc.diags {
			all = append(all, diags...)
		}
		sc.diagMu.RUnlock()
	}
	return all
}

// DiagnosticsSummary returns a text summary for prompt injection.
func (m *Manager) DiagnosticsSummary() string {
	diags := m.GetAllDiagnostics()
	if len(diags) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("--- LSP DIAGNOSTICS ---\n")

	// Group by file
	byFile := make(map[string][]Diagnostic)
	for _, d := range diags {
		rel, _ := filepath.Rel(m.rootPath, d.File)
		if rel == "" {
			rel = d.File
		}
		byFile[rel] = append(byFile[rel], d)
	}

	count := 0
	for file, fileDiags := range byFile {
		if count >= 20 { // Cap at 20 entries
			sb.WriteString("... (more diagnostics omitted)\n")
			break
		}
		for _, d := range fileDiags {
			sb.WriteString(fmt.Sprintf("[%s] %s:%d:%d: %s\n", d.Severity, file, d.Line, d.Column, d.Message))
			count++
			if count >= 20 {
				break
			}
		}
	}
	sb.WriteString("--- END DIAGNOSTICS ---\n")
	return sb.String()
}

// ActiveServers returns the list of running language servers.
func (m *Manager) ActiveServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []string
	for lang := range m.servers {
		list = append(list, lang)
	}
	return list
}

// NotifyFileChanged notifies relevant language servers about a file change.
func (m *Manager) NotifyFileChanged(path string) {
	lang := langForFile(path)
	if lang == "" {
		return
	}

	m.mu.RLock()
	sc, ok := m.servers[lang]
	m.mu.RUnlock()
	if !ok {
		return
	}

	uri := "file://" + path
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": lang,
			"version":    1,
			"text":       string(content),
		},
	}
	sc.notify("textDocument/didOpen", params)
}

func (m *Manager) startServer(cfg ServerConfig) error {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Stderr = io.Discard

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	sc := &serverConn{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReaderSize(stdout, 64*1024),
		pending: make(map[int64]chan json.RawMessage),
		diags:   make(map[string][]Diagnostic),
	}

	// Start reading responses
	go sc.readLoop()

	// Initialize
	if err := m.initializeServer(sc); err != nil {
		cmd.Process.Kill()
		return err
	}

	m.mu.Lock()
	m.servers[cfg.Language] = sc
	m.mu.Unlock()
	return nil
}

func (m *Manager) initializeServer(sc *serverConn) error {
	params := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   "file://" + m.rootPath,
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"publishDiagnostics": map[string]interface{}{},
			},
		},
	}

	_, err := sc.request("initialize", params, 10*time.Second)
	if err != nil {
		return fmt.Errorf("initialize failed: %v", err)
	}
	sc.notify("initialized", map[string]interface{}{})
	return nil
}

func (m *Manager) shutdownServer(sc *serverConn) {
	sc.request("shutdown", nil, 3*time.Second)
	sc.notify("exit", nil)
	time.Sleep(100 * time.Millisecond)
	if sc.cmd.Process != nil {
		sc.cmd.Process.Kill()
	}
}

func (sc *serverConn) request(method string, params interface{}, timeout time.Duration) (json.RawMessage, error) {
	id := atomic.AddInt64(&sc.nextID, 1)
	ch := make(chan json.RawMessage, 1)

	sc.mu.Lock()
	sc.pending[id] = ch
	sc.mu.Unlock()

	defer func() {
		sc.mu.Lock()
		delete(sc.pending, id)
		sc.mu.Unlock()
	}()

	msg := jsonrpcMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
	}
	if params != nil {
		data, _ := json.Marshal(params)
		msg.Params = data
	}

	if err := sc.writeMessage(msg); err != nil {
		return nil, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for %s response", method)
	}
}

func (sc *serverConn) notify(method string, params interface{}) {
	msg := jsonrpcMessage{
		JSONRPC: "2.0",
		Method:  method,
	}
	if params != nil {
		data, _ := json.Marshal(params)
		msg.Params = data
	}
	sc.writeMessage(msg)
}

func (sc *serverConn) writeMessage(msg jsonrpcMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	sc.mu.Lock()
	defer sc.mu.Unlock()

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := sc.stdin.Write([]byte(header)); err != nil {
		return err
	}
	_, err = sc.stdin.Write(data)
	return err
}

func (sc *serverConn) readLoop() {
	for {
		// Read header
		var contentLength int
		for {
			line, err := sc.stdout.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if strings.HasPrefix(line, "Content-Length:") {
				fmt.Sscanf(line, "Content-Length: %d", &contentLength)
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read body
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(sc.stdout, body); err != nil {
			return
		}

		var msg jsonrpcMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			continue
		}

		// Handle response
		if msg.ID != nil && msg.Method == "" {
			sc.mu.Lock()
			if ch, ok := sc.pending[*msg.ID]; ok {
				ch <- msg.Result
			}
			sc.mu.Unlock()
			continue
		}

		// Handle notification
		if msg.Method == "textDocument/publishDiagnostics" {
			sc.handleDiagnostics(msg.Params)
		}
	}
}

func (sc *serverConn) handleDiagnostics(params json.RawMessage) {
	var p struct {
		URI         string `json:"uri"`
		Diagnostics []struct {
			Range struct {
				Start struct {
					Line      int `json:"line"`
					Character int `json:"character"`
				} `json:"start"`
			} `json:"range"`
			Severity int    `json:"severity"`
			Message  string `json:"message"`
			Source   string `json:"source"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	file := strings.TrimPrefix(p.URI, "file://")
	var diags []Diagnostic
	for _, d := range p.Diagnostics {
		sev := "info"
		switch d.Severity {
		case 1:
			sev = "error"
		case 2:
			sev = "warning"
		case 3:
			sev = "info"
		case 4:
			sev = "hint"
		}
		diags = append(diags, Diagnostic{
			File:     file,
			Line:     d.Range.Start.Line + 1,
			Column:   d.Range.Start.Character + 1,
			Severity: sev,
			Message:  d.Message,
			Source:   d.Source,
		})
	}

	sc.diagMu.Lock()
	sc.diags[file] = diags
	sc.diagMu.Unlock()
}

func langForFile(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}
