package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/dev/agent-runtime/internal/cache"
	"github.com/dev/agent-runtime/internal/checkpoint"
	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/context"
	"github.com/dev/agent-runtime/internal/git"
	"github.com/dev/agent-runtime/internal/lsp"
	"github.com/dev/agent-runtime/internal/mcp"
	"github.com/dev/agent-runtime/internal/memory"
	"github.com/dev/agent-runtime/internal/planner"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/tools"
	"github.com/dev/agent-runtime/internal/watcher"
)

type Runtime struct {
	cfg      *config.Config
	store    *storage.Storage
	registry *tools.Registry
	llm      *planner.Planner
	multi    *planner.MultiPlanner
	mem      *memory.MemoryAgent
	toolDefs []planner.ToolDefinition
	sessions map[string]*Session
	// v1.2 components
	cache      *cache.Cache
	ctxMgr     *context.Manager
	checkMgr   *checkpoint.Manager
	gitCtx     *git.Context
	lspMgr     *lsp.Manager
	mcpMgr     *mcp.Manager
	fileWatch  *watcher.Watcher
}

// runLimits holds budgets for a single ProcessMessage invocation.
type runLimits struct {
	deadline        time.Time
	toolCalls       int
	selfReflection  string // pending self-reflection to inject in next LLM call
}

func NewRuntime(cfg *config.Config, store *storage.Storage, reg *tools.Registry, llm *planner.Planner) *Runtime {
	rt := &Runtime{
		cfg:      cfg,
		store:    store,
		registry: reg,
		llm:      llm,
		multi:    planner.NewMultiPlanner(),
		mem:      memory.NewMemoryAgent(cfg.ZAIEndpoint, cfg.ZAIApiKey),
		toolDefs: planner.BuildToolDefinitions(reg),
		sessions: make(map[string]*Session),
		// v1.2 components
		cache:    cache.New(),
		ctxMgr:   context.New(cfg.MaxContextTokens),
		checkMgr: checkpoint.New(),
	}

	// Setup git context
	if cfg.EnableGitContext {
		rt.gitCtx = git.New(cfg.WorkspaceRoot)
		if rt.gitCtx.IsRepo() {
			log.Printf("[runtime] Git context enabled for %s (branch: %s)", cfg.WorkspaceRoot, rt.gitCtx.Branch())
		}
	}

	// Setup file watcher
	if cfg.EnableWatcher {
		rt.fileWatch = watcher.New(cfg.WorkspaceRoot, 5*time.Second)
		rt.fileWatch.Start()
		log.Printf("[runtime] File watcher enabled for %s", cfg.WorkspaceRoot)
	}

	// Setup LSP
	if cfg.EnableLSP {
		rt.lspMgr = lsp.NewManager(cfg.WorkspaceRoot, nil)
		rt.lspMgr.Start()
		servers := rt.lspMgr.ActiveServers()
		if len(servers) > 0 {
			log.Printf("[runtime] LSP servers active: %v", servers)
		}
	}

	// Setup MCP
	rt.mcpMgr = mcp.NewManager()

	// Setup multi-model providers
	if cfg.Models != "" {
		providers := planner.ParseProvidersFromEnv(cfg.Models)
		if len(providers) > 0 {
			rt.multi.SetProviders(providers)
			log.Printf("[runtime] Loaded %d model providers from MODELS env", len(providers))
		}
	}

	// Always add the legacy ZAI provider as fallback
	if cfg.ZAIEndpoint != "" && cfg.ZAIApiKey != "" {
		existing := rt.multi.ListProviders()
		hasLegacy := false
		for _, p := range existing {
			if p.Endpoint == cfg.ZAIEndpoint {
				hasLegacy = true
				break
			}
		}
		if !hasLegacy {
			rt.multi.AddProvider(&planner.ModelProvider{
				ID:       "default",
				Name:     "Default (ZAI)",
				Endpoint: cfg.ZAIEndpoint,
				APIKey:   cfg.ZAIApiKey,
				Model:    "glm-5",
				AuthType: "bearer",
				Priority: 999, // lowest priority — used as fallback
			})
		}
	}

	return rt
}

// GetMultiPlanner returns the multi-planner for external access (e.g., API).
func (r *Runtime) GetMultiPlanner() *planner.MultiPlanner { return r.multi }

// GetCache returns the cache for external access.
func (r *Runtime) GetCache() *cache.Cache { return r.cache }

// GetCheckpointManager returns the checkpoint manager.
func (r *Runtime) GetCheckpointManager() *checkpoint.Manager { return r.checkMgr }

// GetGitContext returns the git context provider.
func (r *Runtime) GetGitContext() *git.Context { return r.gitCtx }

// GetLSPManager returns the LSP manager.
func (r *Runtime) GetLSPManager() *lsp.Manager { return r.lspMgr }

// GetMCPManager returns the MCP manager.
func (r *Runtime) GetMCPManager() *mcp.Manager { return r.mcpMgr }

// GetFileWatcher returns the file watcher.
func (r *Runtime) GetFileWatcher() *watcher.Watcher { return r.fileWatch }

// RefreshToolDefs rebuilds tool definitions (e.g., after MCP tools change).
func (r *Runtime) RefreshToolDefs() {
	r.toolDefs = planner.BuildToolDefinitions(r.registry)
	r.cache.InvalidateToolDefs()
	log.Printf("[runtime] Tool definitions refreshed. Total tools: %d", len(r.toolDefs))
}

// LoadMCPServers loads and starts MCP servers, then registers their tools.
func (r *Runtime) LoadMCPServers(configs []mcp.ServerConfig) int {
	r.mcpMgr.LoadServers(configs)
	count := mcp.RegisterMCPTools(r.mcpMgr, r.registry)
	if count > 0 {
		r.RefreshToolDefs()
		log.Printf("[runtime] Registered %d MCP tools from %d servers", count, r.mcpMgr.ServerCount())
	}
	return count
}

// LoadMCPServersFromConfig reads a JSON config file and loads MCP servers.
func (r *Runtime) LoadMCPServersFromConfig(path string) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("[runtime] Could not read MCP config %s: %v", path, err)
		return
	}
	var wrapper struct {
		Servers []mcp.ServerConfig `json:"servers"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		log.Printf("[runtime] Invalid MCP config %s: %v", path, err)
		return
	}
	r.LoadMCPServers(wrapper.Servers)
}

// ForkSession creates a new session branched from an existing one.
func (r *Runtime) ForkSession(sourceSessionID string, msgIndex int, label string) (string, error) {
	src := r.GetSession(sourceSessionID)
	newID := r.NewSessionID("branch")
	fork := src.Fork(newID, msgIndex, label)
	r.sessions[newID] = fork
	// Persist branch info
	r.store.SaveBranch(newID, sourceSessionID, label, msgIndex)
	log.Printf("[runtime] Forked session %s -> %s at message %d", sourceSessionID, newID, msgIndex)
	return newID, nil
}

// GetBranches returns all branches of a session.
func (r *Runtime) GetBranches(sessionID string) ([]map[string]interface{}, error) {
	return r.store.GetBranches(sessionID)
}

// SaveCheckpoint saves a checkpoint for the current session state.
func (r *Runtime) SaveCheckpoint(sessionID, label string) (string, error) {
	s := r.GetSession(sessionID)
	cpID, err := r.checkMgr.Save(sessionID, label, s.History, s.CheckpointState())
	if err != nil {
		return "", err
	}
	// Also persist to SQLite
	histJSON, _ := json.Marshal(s.History)
	stateJSON, _ := json.Marshal(s.CheckpointState())
	r.store.SaveCheckpoint(cpID, sessionID, label, string(histJSON), string(stateJSON), len(s.History))
	return cpID, nil
}

// RestoreCheckpoint rolls back a session to a previous checkpoint.
func (r *Runtime) RestoreCheckpoint(sessionID, checkpointID string) error {
	histJSON, stateJSON, err := r.checkMgr.Restore(sessionID, checkpointID)
	if err != nil {
		// Try from SQLite
		histJSON, stateJSON, err = r.store.GetCheckpoint(checkpointID)
		if err != nil {
			return fmt.Errorf("checkpoint not found: %v", err)
		}
	}

	s := r.GetSession(sessionID)

	// Restore history
	var history []planner.Message
	if err := json.Unmarshal([]byte(histJSON), &history); err != nil {
		return fmt.Errorf("failed to restore history: %v", err)
	}
	s.History = history

	// Restore state
	var state map[string]interface{}
	if err := json.Unmarshal([]byte(stateJSON), &state); err == nil {
		s.RestoreState(state)
	}

	// Reset loop state
	s.LoopState = NewLoopState()
	s.AwaitingConfirmation = false
	s.PendingToolCalls = nil
	s.PendingAssistantMsg = nil

	log.Printf("[runtime] Restored session %s to checkpoint %s (%d messages)", sessionID, checkpointID, len(history))
	return nil
}

// ListCheckpoints returns checkpoints for a session.
func (r *Runtime) ListCheckpoints(sessionID string) []checkpoint.Checkpoint {
	return r.checkMgr.List(sessionID)
}

// Shutdown cleanly stops all subsystems.
func (r *Runtime) Shutdown() {
	if r.fileWatch != nil {
		r.fileWatch.Stop()
	}
	if r.lspMgr != nil {
		r.lspMgr.Stop()
	}
	if r.mcpMgr != nil {
		r.mcpMgr.Stop()
	}
	log.Printf("[runtime] Shutdown complete")
}

// loopConfig builds a LoopDetectionConfig from the application config.
func (r *Runtime) loopConfig() LoopDetectionConfig {
	return ResolveConfig(LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   r.cfg.LoopHistorySize,
		WarningThreshold:              r.cfg.LoopWarnAt,
		CriticalThreshold:             r.cfg.LoopCriticalAt,
		GlobalCircuitBreakerThreshold: r.cfg.LoopGlobalAt,
		Detectors: DetectorsConfig{
			GenericRepeat:       true,
			KnownPollNoProgress: true,
			PingPong:            true,
		},
		TokenBudget:    r.cfg.TokenBudget,
		BackoffEnabled: true,
	})
}

// AbortSession sets the abort flag on a session's loop state.
func (r *Runtime) AbortSession(sessionID, reason string) {
	s := r.GetSession(sessionID)
	s.LoopState.Abort(reason)
	log.Printf("[runtime] Session %s abort requested: %s", sessionID, reason)
}

func (r *Runtime) GetSession(id string) *Session {
	if s, ok := r.sessions[id]; ok {
		return s
	}
	s := NewSession(id)
	r.hydrateSessionHistory(s)
	r.sessions[id] = s
	return s
}

func (r *Runtime) hydrateSessionHistory(s *Session) {
	msgs, err := r.store.GetRecentMessages(s.ID, r.cfg.MaxHistory*2)
	if err != nil || len(msgs) == 0 {
		return
	}
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" && m.Role != "tool" {
			continue
		}
		s.History = append(s.History, planner.Message{Role: m.Role, Content: m.Content})
	}
}

func (r *Runtime) NewSessionID(prefix string) string {
	p := strings.TrimSpace(prefix)
	if p == "" {
		p = "session"
	}
	return fmt.Sprintf("%s-%d", p, time.Now().UnixNano())
}

func (r *Runtime) ListChatSessions(prefix string, limit int) ([]storage.ChatSessionSummary, error) {
	return r.store.ListChatSessions(prefix, limit)
}

func (r *Runtime) GetChatHistory(sessionID string, limit int) ([]storage.StoredMessage, error) {
	return r.store.GetSessionMessages(sessionID, limit)
}

func (r *Runtime) ResetSession(sessionID string) {
	delete(r.sessions, sessionID)
}

// DeleteSession removes a session from memory and deletes all its messages from storage.
func (r *Runtime) DeleteSession(sessionID string) error {
	delete(r.sessions, sessionID)
	return r.store.DeleteChatSession(sessionID)
}

// GetSessionSettings returns the settings for a session.
func (r *Runtime) GetSessionSettings(sessionID string) SessionSettings {
	s := r.GetSession(sessionID)
	return s.Settings
}

// UpdateSessionSettings updates specific settings for a session.
func (r *Runtime) UpdateSessionSettings(sessionID string, settings SessionSettings) {
	s := r.GetSession(sessionID)
	if settings.ModelID != "" {
		s.Settings.ModelID = settings.ModelID
	}
	if settings.ThinkLevel != "" {
		s.Settings.ThinkLevel = settings.ThinkLevel
	}
	s.Settings.Verbose = settings.Verbose
}

// CompactSession summarizes the current session history into a compact context.
func (r *Runtime) CompactSession(sessionID string) (string, error) {
	s := r.GetSession(sessionID)
	if len(s.History) < 4 {
		return "Session too short to compact.", nil
	}

	// Build a digest of the conversation
	var digest strings.Builder
	for _, msg := range s.History {
		if msg.Role == "tool" {
			continue
		}
		preview := msg.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		digest.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, preview))
	}

	// Ask LLM to summarize
	summaryPrompt := `Summarize this conversation concisely, preserving key decisions, context, and pending tasks. 
Output a compact summary (max 500 words) that can replace the full history without losing important context.
Respond in the same language as the conversation.`

	messages := []planner.Message{
		{Role: "system", Content: summaryPrompt},
		{Role: "user", Content: digest.String()},
	}

	resp, err := r.callLLM(s, messages, nil)
	if err != nil {
		return "", fmt.Errorf("compact failed: %v", err)
	}

	summary := resp.Content
	if summary == "" {
		return "Failed to generate summary.", nil
	}

	// Replace history with a single summary message
	s.History = []planner.Message{
		{Role: "assistant", Content: fmt.Sprintf("[Session Compacted]\n\n%s", summary)},
	}
	s.LoopState = NewLoopState()

	log.Printf("[runtime] Session %s compacted. History reduced to 1 message.", sessionID)
	return summary, nil
}

// ListActiveSessions returns info about in-memory sessions for agent-to-agent coordination.
func (r *Runtime) ListActiveSessions() []map[string]interface{} {
	var sessions []map[string]interface{}
	for id, s := range r.sessions {
		lastMsg := ""
		if len(s.History) > 0 {
			last := s.History[len(s.History)-1]
			lastMsg = last.Content
			if len(lastMsg) > 100 {
				lastMsg = lastMsg[:100] + "..."
			}
		}
		sessions = append(sessions, map[string]interface{}{
			"session_id":   id,
			"messages":     len(s.History),
			"last_message": lastMsg,
			"model_id":     s.Settings.ModelID,
			"think_level":  s.Settings.ThinkLevel,
			"awaiting":     s.AwaitingConfirmation,
		})
	}
	return sessions
}

// SendToSession sends a message to another session from an external source (agent-to-agent).
func (r *Runtime) SendToSession(targetSessionID, message, senderSessionID string) (string, error) {
	prefixed := fmt.Sprintf("[From session %s]: %s", senderSessionID, message)
	reply, _ := r.ProcessMessage(targetSessionID, prefixed)
	return reply, nil
}

// GetSessionHistory returns formatted history of a session for agent-to-agent use.
func (r *Runtime) GetSessionHistory(sessionID string, limit int) string {
	s := r.GetSession(sessionID)
	history := s.History
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}
	var sb strings.Builder
	for _, msg := range history {
		if msg.Role == "tool" {
			continue
		}
		preview := msg.Content
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, preview))
	}
	return sb.String()
}

func (r *Runtime) buildSystemPrompt(memoryCtx string, settings SessionSettings) string {
	// Check cache first
	cacheKey := cache.HashKey(settings.ThinkLevel, fmt.Sprintf("%v", settings.Verbose), memoryCtx)
	if cached, ok := r.cache.GetSystemPrompt(cacheKey); ok {
		return cached
	}

	var sb strings.Builder
	readPrompt := func(name string) {
		path := filepath.Join(r.cfg.PromptsDir, name)
		content, err := ioutil.ReadFile(path)
		if err == nil {
			sb.WriteString(string(content) + "\n\n")
		}
	}
	readPrompt("soul.md")
	readPrompt("rules.md")
	readPrompt("tools.md")

	// Think level instruction
	switch settings.ThinkLevel {
	case "off":
		sb.WriteString("\n[Think Level: OFF] Be as concise as possible. Give direct answers without elaboration.\n\n")
	case "low":
		sb.WriteString("\n[Think Level: LOW] Brief reasoning, then answer. Keep explanations short.\n\n")
	case "high":
		sb.WriteString("\n[Think Level: HIGH] Think step-by-step in detail. Show your full reasoning process before concluding.\n\n")
	default: // "medium" or empty
		sb.WriteString("\n[Think Level: MEDIUM] Provide clear reasoning with moderate detail.\n\n")
	}

	// Verbose mode instruction
	if settings.Verbose {
		sb.WriteString("[Verbose Mode: ON] Include details about tool executions and intermediate steps in your response.\n\n")
	}

	// Git context injection
	if r.gitCtx != nil {
		gitSummary := r.gitCtx.Summary()
		if gitSummary != "" {
			sb.WriteString("\n" + gitSummary + "\n")
		}
	}

	// File watcher context injection
	if r.fileWatch != nil {
		watchSummary := r.fileWatch.Summary(5 * time.Minute)
		if watchSummary != "" {
			sb.WriteString("\n" + watchSummary + "\n")
		}
	}

	// LSP diagnostics injection
	if r.lspMgr != nil {
		diagSummary := r.lspMgr.DiagnosticsSummary()
		if diagSummary != "" {
			sb.WriteString("\n" + diagSummary + "\n")
		}
	}

	// MCP tools info
	if r.mcpMgr != nil && r.mcpMgr.ToolCount() > 0 {
		sb.WriteString(fmt.Sprintf("\n[MCP] %d external tools available from %d MCP servers.\n\n", r.mcpMgr.ToolCount(), r.mcpMgr.ServerCount()))
	}

	if memoryCtx != "" {
		sb.WriteString("\n--- CONTEXTO DE CONVERSAS ANTERIORES ---\n")
		sb.WriteString(memoryCtx)
		sb.WriteString("\n--- FIM DO CONTEXTO ---\n\n")
	}

	result := sb.String()
	r.cache.SetSystemPrompt(cacheKey, result)
	return result
}

func (r *Runtime) trimHistory(s *Session) {
	max := r.cfg.MaxHistory * 2
	if len(s.History) > max {
		s.History = s.History[len(s.History)-max:]
	}
}

// callLLM routes the call through MultiPlanner (with failover) or falls back to single planner.
func (r *Runtime) callLLM(s *Session, messages []planner.Message, toolDefs []planner.ToolDefinition) (*planner.Message, error) {
	providers := r.multi.ListProviders()
	if len(providers) > 0 {
		return r.multi.Call(messages, toolDefs, s.Settings.ModelID)
	}
	// Fallback to legacy single planner
	return r.llm.Call(messages, toolDefs)
}

// callLLMStream routes the call through MultiPlanner with streaming.
func (r *Runtime) callLLMStream(s *Session, messages []planner.Message, toolDefs []planner.ToolDefinition, onToken planner.StreamCallback) (*planner.Message, error) {
	providers := r.multi.ListProviders()
	if len(providers) > 0 {
		return r.multi.CallStream(messages, toolDefs, s.Settings.ModelID, onToken)
	}
	// Fallback to legacy single planner (no streaming)
	return r.llm.Call(messages, toolDefs)
}

// ProcessMessage is the main entry point called by Web/Telegram interfaces.
// For real-time progress updates, use ProcessMessageWithProgress instead.
func (r *Runtime) ProcessMessage(sessionID, userMessage string) (string, bool) {
	return r.ProcessMessageWithProgress(sessionID, userMessage, nil)
}

// ProcessMessageWithProgress is ProcessMessage with an optional progress callback.
func (r *Runtime) ProcessMessageWithProgress(sessionID, userMessage string, onProgress ProgressCallback) (string, bool) {
	log.Printf("[runtime] ProcessMessage session=%s msg=%q", sessionID, truncateLog(userMessage, 100))
	s := r.GetSession(sessionID)
	limits := r.newRunLimits()

	// Handle pending confirmation for high-risk tools
	if s.AwaitingConfirmation {
		msg := strings.ToUpper(strings.TrimSpace(userMessage))
		if msg == "YES" || msg == "SIM" || msg == "Y" {
			s.AwaitingConfirmation = false
			return r.executeApprovedToolCalls(s, onProgress)
		} else if msg == "NO" || msg == "NAO" || msg == "N" {
			s.AwaitingConfirmation = false
			s.PendingToolCalls = nil
			s.PendingAssistantMsg = nil
			return "Execution cancelled.", false
		}
		return "Pending high-risk action. Reply YES or NO.", true
	}

	// Add user message to history
	s.History = append(s.History, planner.Message{Role: "user", Content: userMessage})
	r.store.LogMessage(sessionID, "user", userMessage)
	r.trimHistory(s)

	// Auto-checkpoint before processing (every 5 messages)
	if len(s.History)%5 == 0 && len(s.History) > 0 {
		r.checkMgr.Save(sessionID, fmt.Sprintf("auto-msg-%d", len(s.History)), s.History, s.CheckpointState())
	}

	// Memory Agent: retrieve relevant older context
	memoryCtx := ""
	olderMsgs, err := r.store.SearchOlderMessages(sessionID, r.cfg.MaxHistory*2, 100)
	if err == nil && len(olderMsgs) > 0 {
		ctx, err := r.mem.RetrieveRelevantContext(userMessage, olderMsgs)
		if err != nil {
			log.Printf("Memory agent error (non-fatal): %v", err)
		} else {
			memoryCtx = ctx
		}
	}

	// Build messages with system prompt
	systemPrompt := r.buildSystemPrompt(memoryCtx, s.Settings)
	messages := []planner.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, s.History...)

	// Context window management: intelligently truncate if needed
	messages = r.ctxMgr.TruncateMessages(messages)

	// Agentic loop: keep calling LLM until it stops requesting tools
	return r.agenticLoop(s, messages, 0, limits, onProgress)
}

// agenticLoop calls the LLM, handles tool calls, feeds results back, repeats
func (r *Runtime) agenticLoop(s *Session, messages []planner.Message, depth int, limits *runLimits, onProgress ProgressCallback) (string, bool) {
	log.Printf("[runtime] agenticLoop session=%s depth=%d/%d toolCalls=%d/%d",
		s.ID, depth, r.cfg.MaxTurns, limits.toolCalls, r.cfg.MaxToolCalls)

	// Check abort
	if s.LoopState.IsAborted() {
		log.Printf("[session=%s] Aborted by client: %s", s.ID, s.LoopState.AbortReason)
		r.autoCheckpointOnBreaker(s, "abort")
		return r.finishWithSummaryRequest(s, "[ABORTED] "+s.LoopState.AbortReason), false
	}

	// Soft deadline: at 80% of time budget, inject self-reflection; at 100%, request summary
	if r.isTimedOut(limits) {
		log.Printf("[session=%s] Time budget exhausted. Requesting summary.", s.ID)
		r.autoCheckpointOnBreaker(s, "time-budget")
		return r.finishWithSummaryRequest(s,
			"You have used all available processing time. "+
			"Summarize what you accomplished and what tasks remain."), false
	}
	if r.isApproachingTimeout(limits) && limits.selfReflection == "" {
		remaining := time.Until(limits.deadline).Round(time.Second)
		limits.selfReflection = fmt.Sprintf(
			"[TIME WARNING] You have approximately %s of processing time remaining. "+
			"Prioritize completing your current task. Avoid starting new subtasks.", remaining)
	}

	if depth >= r.cfg.MaxTurns {
		log.Printf("[session=%s] Turn limit (%d) reached. Requesting summary.", s.ID, r.cfg.MaxTurns)
		r.autoCheckpointOnBreaker(s, "turn-limit")
		return r.finishWithSummaryRequest(s,
			fmt.Sprintf("You have used all %d available turns. "+
			"Summarize what you accomplished and what tasks remain.", r.cfg.MaxTurns)), false
	}

	// Emit thinking phase
	emitProgress(onProgress, ProgressEvent{
		Phase:   PhaseThinking,
		Message: "Thinking...",
		Depth:   depth,
	})

	// Inject pending self-reflection as a system message
	if limits.selfReflection != "" {
		messages = append(messages, planner.Message{
			Role:    "system",
			Content: limits.selfReflection,
		})
		s.LoopState.ReflectionsSent++
		limits.selfReflection = "" // consume it
	}

	var resp *planner.Message
	var err error

	if s.Settings.Streaming && onProgress != nil {
		tokenCb := func(token string) {
			emitProgress(onProgress, ProgressEvent{
				Phase: PhaseToken,
				Token: token,
				Depth: depth,
			})
		}
		resp, err = r.callLLMStream(s, messages, r.toolDefs, tokenCb)
	} else {
		resp, err = r.callLLM(s, messages, r.toolDefs)
	}
	if err != nil {
		emitProgress(onProgress, ProgressEvent{Phase: PhaseError, Message: err.Error(), Depth: depth})
		return fmt.Sprintf("LLM error: %v", err), false
	}

	// No tool calls: LLM gave a direct text response
	if len(resp.ToolCalls) == 0 {
		text := resp.Content
		log.Printf("[runtime] LLM final response session=%s depth=%d len=%d", s.ID, depth, len(text))
		return r.finishWithMessage(s, text), false
	}

	// LLM requested tool calls — check for high-risk ones that need approval
	log.Printf("[runtime] LLM requested %d tool call(s) session=%s depth=%d", len(resp.ToolCalls), s.ID, depth)
	for _, tc := range resp.ToolCalls {
		log.Printf("[runtime]   tool=%s args=%s", tc.Function.Name, truncateLog(tc.Function.Arguments, 120))
		tool, err := r.registry.Get(tc.Function.Name)
		if err != nil {
			continue
		}
		if tool.Risk() == "HIGH" {
			// Pause and ask for confirmation
			s.PendingToolCalls = resp.ToolCalls
			s.PendingAssistantMsg = resp
			s.AwaitingConfirmation = true

			var desc strings.Builder
			desc.WriteString("High-risk actions requested:\n\n")
			for _, tc2 := range resp.ToolCalls {
				desc.WriteString(fmt.Sprintf("⚠ Tool: %s\nArgs: %s\n\n", tc2.Function.Name, tc2.Function.Arguments))
			}
			desc.WriteString("Confirm execution? YES / NO")
			return desc.String(), true
		}
	}

	// All tools are LOW risk — execute immediately
	return r.executeToolCalls(s, messages, resp, depth, limits, onProgress)
}

// executeApprovedToolCalls runs after user confirms
func (r *Runtime) executeApprovedToolCalls(s *Session, onProgress ProgressCallback) (string, bool) {
	if s.PendingToolCalls == nil || s.PendingAssistantMsg == nil {
		return "No pending action.", false
	}

	// Rebuild messages context
	systemPrompt := r.buildSystemPrompt("", s.Settings)
	messages := []planner.Message{{Role: "system", Content: systemPrompt}}
	messages = append(messages, s.History...)

	resp := s.PendingAssistantMsg
	s.PendingToolCalls = nil
	s.PendingAssistantMsg = nil

	return r.executeToolCalls(s, messages, resp, 0, r.newRunLimits(), onProgress)
}

// executeToolCalls runs actual tools using the beforeToolCall pattern:
// detect loop BEFORE executing, block on critical only,
// then record outcome AFTER execution for no-progress tracking.
// Warning-level detections do NOT skip execution — they append a note to the
// tool result so the LLM is aware but work continues uninterrupted.
func (r *Runtime) executeToolCalls(s *Session, messages []planner.Message, resp *planner.Message, depth int, limits *runLimits, onProgress ProgressCallback) (string, bool) {
	// Check abort before executing tools
	if s.LoopState.IsAborted() {
		r.autoCheckpointOnBreaker(s, "abort")
		return r.finishWithSummaryRequest(s, "[ABORTED] "+s.LoopState.AbortReason), false
	}

	if r.isTimedOut(limits) {
		r.autoCheckpointOnBreaker(s, "time-budget")
		return r.finishWithSummaryRequest(s,
			"You have used all available processing time. "+
			"Summarize what you accomplished and what tasks remain."), false
	}

	// Add the assistant's message (with tool_calls) to history
	s.History = append(s.History, *resp)
	messages = append(messages, *resp)

	loopCfg := r.loopConfig()

	for _, tc := range resp.ToolCalls {
		// Budget checks
		if r.cfg.MaxToolCalls > 0 && limits.toolCalls >= r.cfg.MaxToolCalls {
			log.Printf("[session=%s] Tool call budget exhausted (%d).", s.ID, r.cfg.MaxToolCalls)
			r.autoCheckpointOnBreaker(s, "tool-budget")
			return r.finishWithSummaryRequest(s,
				fmt.Sprintf("You have used all %d available tool calls. "+
				"Summarize what you accomplished and what tasks remain.", r.cfg.MaxToolCalls)), false
		}
		if r.isTimedOut(limits) {
			r.autoCheckpointOnBreaker(s, "time-budget")
			return r.finishWithSummaryRequest(s,
				"You have used all available processing time. "+
				"Summarize what you accomplished and what tasks remain."), false
		}
		if s.LoopState.IsAborted() {
			r.autoCheckpointOnBreaker(s, "abort")
			return r.finishWithSummaryRequest(s, "[ABORTED] "+s.LoopState.AbortReason), false
		}
		limits.toolCalls++

		argsHash := HashToolCall(tc.Function.Name, tc.Function.Arguments)

		// ---- beforeToolCall: detect loop BEFORE executing ----
		det := DetectToolCallLoop(s.LoopState, tc.Function.Name, argsHash, loopCfg)
		warningNote := ""

		if det.Stuck {
			log.Printf("[session=%s] Loop detected before exec: detector=%s level=%s count=%d",
				s.ID, det.Detector, det.Level, det.Count)

			if det.Level == LevelCritical {
				// Critical: block execution, auto-checkpoint, inject self-reflection
				r.autoCheckpointOnBreaker(s, string(det.Detector))
				toolResultMsg := planner.Message{
					Role:       "tool",
					Content:    det.SelfReflection,
					ToolCallID: tc.ID,
				}
				s.History = append(s.History, toolResultMsg)
				messages = append(messages, toolResultMsg)
				return r.finishWithSummaryRequest(s, det.SelfReflection), false
			}

			// Warning level: apply backoff delay, then execute with self-reflection
			if det.BackoffMs > 0 {
				log.Printf("[session=%s] Backoff %dms before %s", s.ID, det.BackoffMs, tc.Function.Name)
				time.Sleep(time.Duration(det.BackoffMs) * time.Millisecond)
			}

			if ShouldEmitWarning(s.LoopState, det.WarningKey, det.Count) {
				log.Printf("[session=%s] Loop warning emitted: %s", s.ID, string(det.Detector))
			}

			// Queue self-reflection for next LLM call
			limits.selfReflection = det.SelfReflection
			warningNote = "\n\n" + det.SelfReflection
		}

		// ---- Record call (before execution, resultHash pending) ----
		RecordToolCall(s.LoopState, tc.Function.Name, argsHash, tc.ID, loopCfg)

		// ---- Emit tool_start event ----
		toolArgsPreview := tc.Function.Arguments
		if len(toolArgsPreview) > 200 {
			toolArgsPreview = toolArgsPreview[:200] + "..."
		}
		emitProgress(onProgress, ProgressEvent{
			Phase:    PhaseToolStart,
			ToolName: tc.Function.Name,
			ToolArgs: toolArgsPreview,
			Message:  toolActionMessage(tc.Function.Name),
			Depth:    depth,
		})

		// ---- Execute the tool ----
		tool, err := r.registry.Get(tc.Function.Name)

		var output string
		var isError bool
		if err != nil {
			output = fmt.Sprintf("Tool '%s' not found", tc.Function.Name)
			isError = true
		} else {
			args := make(map[string]string)
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			ctx := tools.ToolContext{
				SessionID: s.ID,
				Workspace: r.cfg.WorkspaceRoot,
				Depth:     0,
				MaxDepth:  r.cfg.MaxAgentDepth,
			}
			out, execErr := tool.Execute(ctx, args)
			if execErr != nil {
				output = fmt.Sprintf("Error: %v", execErr)
				isError = true
			} else {
				output = out
			}
			r.store.LogToolExecution(s.ID, tc.Function.Name, tc.Function.Arguments, output, "OK")
		}

		// ---- Emit tool_end event ----
		endMsg := fmt.Sprintf("%s completed", tc.Function.Name)
		if isError {
			endMsg = fmt.Sprintf("%s failed", tc.Function.Name)
		}
		emitProgress(onProgress, ProgressEvent{
			Phase:    PhaseToolEnd,
			ToolName: tc.Function.Name,
			Message:  endMsg,
			Depth:    depth,
		})

		// ---- Record outcome (after execution) ----
		resultHash := HashOutcome(output, isError)
		RecordToolCallOutcome(s.LoopState, tc.Function.Name, argsHash, tc.ID, resultHash, loopCfg)

		// Add tool result message to history (with warning note if applicable)
		toolResultMsg := planner.Message{
			Role:       "tool",
			Content:    output + warningNote,
			ToolCallID: tc.ID,
		}
		s.History = append(s.History, toolResultMsg)
		messages = append(messages, toolResultMsg)
	}

	// Call LLM again with tool results — it may want more tools or give final answer
	return r.agenticLoop(s, messages, depth+1, limits, onProgress)
}

func (r *Runtime) finishWithMessage(s *Session, text string) string {
	s.History = append(s.History, planner.Message{Role: "assistant", Content: text})
	r.store.LogMessage(s.ID, "assistant", text)
	return text
}

// finishWithSummaryRequest asks the LLM to summarize before stopping,
// instead of returning a raw system error message to the user.
func (r *Runtime) finishWithSummaryRequest(s *Session, reason string) string {
	// Build a summary request for the LLM
	summaryPrompt := fmt.Sprintf(
		"%s\n\nProvide a concise summary of:\n"+
		"1. What was accomplished\n"+
		"2. What tasks remain incomplete\n"+
		"3. Any issues encountered\n"+
		"Respond in the same language as the conversation.", reason)

	messages := []planner.Message{
		{Role: "system", Content: "You are summarizing your work session. Be concise and helpful."},
	}
	// Include last few history messages for context
	historySlice := s.History
	if len(historySlice) > 6 {
		historySlice = historySlice[len(historySlice)-6:]
	}
	messages = append(messages, historySlice...)
	messages = append(messages, planner.Message{Role: "user", Content: summaryPrompt})

	resp, err := r.callLLM(s, messages, nil)
	if err != nil {
		// If LLM fails, return the raw reason as fallback
		return r.finishWithMessage(s, reason)
	}

	summary := resp.Content
	if summary == "" {
		return r.finishWithMessage(s, reason)
	}

	return r.finishWithMessage(s, summary)
}

// autoCheckpointOnBreaker saves a checkpoint when a circuit breaker or budget limit fires.
func (r *Runtime) autoCheckpointOnBreaker(s *Session, reason string) {
	label := fmt.Sprintf("auto-breaker-%s-%d", reason, len(s.History))
	cpID, err := r.checkMgr.Save(s.ID, label, s.History, s.CheckpointState())
	if err != nil {
		log.Printf("[session=%s] Auto-checkpoint on breaker failed: %v", s.ID, err)
		return
	}
	log.Printf("[session=%s] Auto-checkpoint saved on %s: %s", s.ID, reason, cpID)
}

// emitProgress safely calls the progress callback if non-nil.
func emitProgress(cb ProgressCallback, event ProgressEvent) {
	if cb != nil {
		cb(event)
	}
}

// toolActionMessage returns a human-readable action message for a tool (OpenClaw-style).
func toolActionMessage(toolName string) string {
	switch toolName {
	case "shell":
		return "Running command..."
	case "read_file", "workspace_list":
		return "Reading file..."
	case "write_file":
		return "Writing file..."
	case "patch_file":
		return "Patching file..."
	case "delete_file":
		return "Deleting file..."
	case "delegate":
		return "Delegating to sub-agent..."
	case "sessions_list":
		return "Listing sessions..."
	case "sessions_send":
		return "Sending to session..."
	case "sessions_history":
		return "Fetching session history..."
	default:
		if strings.HasPrefix(toolName, "mcp_") {
			return fmt.Sprintf("Calling MCP tool %s...", toolName)
		}
		return fmt.Sprintf("Executing %s...", toolName)
	}
}

func (r *Runtime) newRunLimits() *runLimits {
	limits := &runLimits{}
	if r.cfg.MaxRunSeconds > 0 {
		limits.deadline = time.Now().Add(time.Duration(r.cfg.MaxRunSeconds) * time.Second)
	}
	return limits
}

func (r *Runtime) isTimedOut(limits *runLimits) bool {
	if limits == nil || limits.deadline.IsZero() {
		return false
	}
	return time.Now().After(limits.deadline)
}

// isApproachingTimeout returns true when 80% of the time budget is consumed.
func (r *Runtime) isApproachingTimeout(limits *runLimits) bool {
	if limits == nil || limits.deadline.IsZero() || r.cfg.MaxRunSeconds <= 0 {
		return false
	}
	totalDuration := time.Duration(r.cfg.MaxRunSeconds) * time.Second
	elapsed := totalDuration - time.Until(limits.deadline)
	return elapsed >= time.Duration(float64(totalDuration)*0.8)
}

// truncateLog shortens a string for logging purposes.
func truncateLog(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
