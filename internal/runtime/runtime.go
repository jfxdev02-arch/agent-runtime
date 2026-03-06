package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/memory"
	"github.com/dev/agent-runtime/internal/planner"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/tools"
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
}

// runLimits holds wall-clock and tool-call budget for a single ProcessMessage invocation.
type runLimits struct {
	deadline  time.Time
	toolCalls int
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
	}

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
	})
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

	if memoryCtx != "" {
		sb.WriteString("\n--- CONTEXTO DE CONVERSAS ANTERIORES ---\n")
		sb.WriteString(memoryCtx)
		sb.WriteString("\n--- FIM DO CONTEXTO ---\n\n")
	}
	return sb.String()
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

// ProcessMessage is the main entry point called by Web/Telegram interfaces
func (r *Runtime) ProcessMessage(sessionID, userMessage string) (string, bool) {
	log.Printf("[runtime] ProcessMessage session=%s msg=%q", sessionID, truncateLog(userMessage, 100))
	s := r.GetSession(sessionID)
	limits := r.newRunLimits()

	// Handle pending confirmation for high-risk tools
	if s.AwaitingConfirmation {
		msg := strings.ToUpper(strings.TrimSpace(userMessage))
		if msg == "YES" || msg == "SIM" || msg == "Y" {
			s.AwaitingConfirmation = false
			return r.executeApprovedToolCalls(s)
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

	// Agentic loop: keep calling LLM until it stops requesting tools
	return r.agenticLoop(s, messages, 0, limits)
}

// agenticLoop calls the LLM, handles tool calls, feeds results back, repeats
func (r *Runtime) agenticLoop(s *Session, messages []planner.Message, depth int, limits *runLimits) (string, bool) {
	log.Printf("[runtime] agenticLoop session=%s depth=%d/%d toolCalls=%d/%d",
		s.ID, depth, r.cfg.MaxTurns, limits.toolCalls, r.cfg.MaxToolCalls)

	if r.isTimedOut(limits) {
		log.Printf("[session=%s] Global timeout reached (%ds). Stopping.", s.ID, r.cfg.MaxRunSeconds)
		return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
	}

	if depth >= r.cfg.MaxTurns {
		log.Printf("[session=%s] Turn limit (%d) reached. Stopping.", s.ID, r.cfg.MaxTurns)
		summary := fmt.Sprintf("Turn limit (%d) reached. Stopping to avoid tool loops.", r.cfg.MaxTurns)
		return r.finishWithMessage(s, summary), false
	}

	resp, err := r.callLLM(s, messages, r.toolDefs)
	if err != nil {
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
	return r.executeToolCalls(s, messages, resp, depth, limits)
}

// executeApprovedToolCalls runs after user confirms
func (r *Runtime) executeApprovedToolCalls(s *Session) (string, bool) {
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

	return r.executeToolCalls(s, messages, resp, 0, r.newRunLimits())
}

// executeToolCalls runs actual tools using the beforeToolCall pattern from openclaw:
// detect loop BEFORE executing, block/warn without wasting execution time,
// then record outcome AFTER execution for no-progress tracking.
func (r *Runtime) executeToolCalls(s *Session, messages []planner.Message, resp *planner.Message, depth int, limits *runLimits) (string, bool) {
	if r.isTimedOut(limits) {
		return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
	}

	// Add the assistant's message (with tool_calls) to history
	s.History = append(s.History, *resp)
	messages = append(messages, *resp)

	loopCfg := r.loopConfig()

	for _, tc := range resp.ToolCalls {
		// Budget checks
		if r.cfg.MaxToolCalls > 0 && limits.toolCalls >= r.cfg.MaxToolCalls {
			log.Printf("[session=%s] Tool call limit reached (%d). Stopping.", s.ID, r.cfg.MaxToolCalls)
			msg := fmt.Sprintf("Tool call limit reached (%d). Stopping to avoid loops.", r.cfg.MaxToolCalls)
			return r.finishWithMessage(s, msg), false
		}
		if r.isTimedOut(limits) {
			return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
		}
		limits.toolCalls++

		argsHash := HashToolCall(tc.Function.Name, tc.Function.Arguments)

		// ---- beforeToolCall: detect loop BEFORE executing (openclaw pattern) ----
		det := DetectToolCallLoop(s.LoopState, tc.Function.Name, argsHash, loopCfg)
		if det.Stuck {
			log.Printf("[session=%s] Loop detected before exec: detector=%s level=%s count=%d msg=%s",
				s.ID, det.Detector, det.Level, det.Count, det.Message)

			if det.Level == LevelCritical {
				// Critical: block execution, inject error as tool result, abort session
				toolResultMsg := planner.Message{
					Role:       "tool",
					Content:    "BLOCKED: " + det.Message,
					ToolCallID: tc.ID,
				}
				s.History = append(s.History, toolResultMsg)
				messages = append(messages, toolResultMsg)
				return r.finishWithMessage(s, det.Message), false
			}

			// Warning: skip execution, inject warning so LLM can self-correct
			if ShouldEmitWarning(s.LoopState, det.WarningKey, det.Count) {
				log.Printf("[session=%s] Loop warning emitted: %s", s.ID, det.Message)
			}
			toolResultMsg := planner.Message{
				Role:       "tool",
				Content:    "WARNING — tool execution skipped: " + det.Message + "\nDo NOT retry with the same arguments. Try a different approach or report the task as failed.",
				ToolCallID: tc.ID,
			}
			s.History = append(s.History, toolResultMsg)
			messages = append(messages, toolResultMsg)
			// Record as a skipped call so future detection counts it
			RecordToolCall(s.LoopState, tc.Function.Name, argsHash, tc.ID, loopCfg)
			RecordToolCallOutcome(s.LoopState, tc.Function.Name, argsHash, tc.ID,
				HashOutcome("skipped:loop-warning", true), loopCfg)
			continue
		}

		// ---- Record call (before execution, resultHash pending) ----
		RecordToolCall(s.LoopState, tc.Function.Name, argsHash, tc.ID, loopCfg)

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

		// ---- Record outcome (after execution) ----
		resultHash := HashOutcome(output, isError)
		RecordToolCallOutcome(s.LoopState, tc.Function.Name, argsHash, tc.ID, resultHash, loopCfg)

		// Add tool result message to history
		toolResultMsg := planner.Message{
			Role:       "tool",
			Content:    output,
			ToolCallID: tc.ID,
		}
		s.History = append(s.History, toolResultMsg)
		messages = append(messages, toolResultMsg)
	}

	// Call LLM again with tool results — it may want more tools or give final answer
	return r.agenticLoop(s, messages, depth+1, limits)
}

func (r *Runtime) finishWithMessage(s *Session, text string) string {
	s.History = append(s.History, planner.Message{Role: "assistant", Content: text})
	r.store.LogMessage(s.ID, "assistant", text)
	return text
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

// truncateLog shortens a string for logging purposes.
func truncateLog(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
