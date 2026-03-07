package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ServerConfig is the configuration for an MCP server.
type ServerConfig struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// ToolDef is a tool definition from an MCP server.
type ToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	ServerName  string          `json:"server_name"`
}

// Manager manages multiple MCP server connections.
type Manager struct {
	servers map[string]*serverConn
	tools   map[string]*ToolDef // toolName -> tool
	mu      sync.RWMutex
}

type serverConn struct {
	name    string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int64
	pending map[int64]chan json.RawMessage
}

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *int64           `json:"id,omitempty"`
	Result  json.RawMessage  `json:"result,omitempty"`
	Error   *json.RawMessage `json:"error,omitempty"`
	Method  string           `json:"method,omitempty"`
}

// NewManager creates a new MCP manager.
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*serverConn),
		tools:   make(map[string]*ToolDef),
	}
}

// LoadServers starts MCP servers from config and discovers their tools.
func (m *Manager) LoadServers(configs []ServerConfig) {
	for _, cfg := range configs {
		if err := m.startServer(cfg); err != nil {
			log.Printf("[mcp] Failed to start server %s: %v", cfg.Name, err)
			continue
		}
		log.Printf("[mcp] Started server %s (%s)", cfg.Name, cfg.Command)

		// Discover tools
		tools, err := m.listTools(cfg.Name)
		if err != nil {
			log.Printf("[mcp] Failed to list tools from %s: %v", cfg.Name, err)
			continue
		}
		m.mu.Lock()
		for i := range tools {
			tools[i].ServerName = cfg.Name
			// Namespace tool names to avoid collisions
			qualifiedName := fmt.Sprintf("mcp_%s_%s", cfg.Name, tools[i].Name)
			m.tools[qualifiedName] = &tools[i]
			log.Printf("[mcp]   Registered tool: %s", qualifiedName)
		}
		m.mu.Unlock()
	}
}

// Stop shuts down all MCP servers.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, sc := range m.servers {
		sc.stdin.Close()
		if sc.cmd.Process != nil {
			sc.cmd.Process.Kill()
		}
		log.Printf("[mcp] Stopped server %s", name)
	}
	m.servers = make(map[string]*serverConn)
	m.tools = make(map[string]*ToolDef)
}

// ListTools returns all discovered MCP tools.
func (m *Manager) ListTools() []ToolDef {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []ToolDef
	for _, t := range m.tools {
		list = append(list, *t)
	}
	return list
}

// CallTool invokes a tool on its MCP server.
func (m *Manager) CallTool(qualifiedName string, args map[string]interface{}) (string, error) {
	m.mu.RLock()
	tool, ok := m.tools[qualifiedName]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("MCP tool %s not found", qualifiedName)
	}

	m.mu.RLock()
	sc, ok := m.servers[tool.ServerName]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("MCP server %s not connected", tool.ServerName)
	}

	// The actual tool name (without namespace prefix)
	actualName := tool.Name

	params := map[string]interface{}{
		"name":      actualName,
		"arguments": args,
	}

	result, err := sc.request("tools/call", params, 60*time.Second)
	if err != nil {
		return "", err
	}

	// Parse MCP tool result
	var toolResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(result, &toolResult); err != nil {
		return string(result), nil
	}

	var texts []string
	for _, c := range toolResult.Content {
		if c.Type == "text" {
			texts = append(texts, c.Text)
		}
	}
	output := strings.Join(texts, "\n")
	if toolResult.IsError {
		return "", fmt.Errorf("MCP tool error: %s", output)
	}
	return output, nil
}

// ServerCount returns the number of connected MCP servers.
func (m *Manager) ServerCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.servers)
}

// ToolCount returns the number of available MCP tools.
func (m *Manager) ToolCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tools)
}

func (m *Manager) startServer(cfg ServerConfig) error {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Stderr = io.Discard

	// Set environment variables
	if len(cfg.Env) > 0 {
		env := cmd.Environ()
		for k, v := range cfg.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

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
		name:    cfg.Name,
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReaderSize(stdout, 64*1024),
		pending: make(map[int64]chan json.RawMessage),
	}

	go sc.readLoop()

	// Initialize the MCP server
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "agent-runtime",
			"version": "1.2.0",
		},
	}
	_, err = sc.request("initialize", initParams, 10*time.Second)
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("MCP initialize failed: %v", err)
	}
	sc.notify("notifications/initialized", nil)

	m.mu.Lock()
	m.servers[cfg.Name] = sc
	m.mu.Unlock()
	return nil
}

func (m *Manager) listTools(serverName string) ([]ToolDef, error) {
	m.mu.RLock()
	sc, ok := m.servers[serverName]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("server %s not found", serverName)
	}

	result, err := sc.request("tools/list", nil, 10*time.Second)
	if err != nil {
		return nil, err
	}

	var toolsResult struct {
		Tools []ToolDef `json:"tools"`
	}
	if err := json.Unmarshal(result, &toolsResult); err != nil {
		return nil, err
	}
	return toolsResult.Tools, nil
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

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := sc.writeMessage(req); err != nil {
		return nil, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for %s", method)
	}
}

func (sc *serverConn) notify(method string, params interface{}) {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		msg["params"] = params
	}
	sc.writeRawMessage(msg)
}

func (sc *serverConn) writeMessage(req jsonrpcRequest) error {
	data, err := json.Marshal(req)
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

func (sc *serverConn) writeRawMessage(msg interface{}) error {
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
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(sc.stdout, body); err != nil {
			return
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			continue
		}

		if resp.ID != nil && resp.Method == "" {
			sc.mu.Lock()
			if ch, ok := sc.pending[*resp.ID]; ok {
				ch <- resp.Result
			}
			sc.mu.Unlock()
		}
	}
}
