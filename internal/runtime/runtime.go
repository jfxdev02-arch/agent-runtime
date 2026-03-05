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
	return &Runtime{
		cfg:      cfg,
		store:    store,
		registry: reg,
		llm:      llm,
		mem:      memory.NewMemoryAgent(cfg.ZAIEndpoint, cfg.ZAIApiKey),
		toolDefs: planner.BuildToolDefinitions(reg),
		sessions: make(map[string]*Session),
	}
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

func (r *Runtime) buildSystemPrompt(memoryCtx string) string {
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
	systemPrompt := r.buildSystemPrompt(memoryCtx)
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

	resp, err := r.llm.Call(messages, r.toolDefs)
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
	systemPrompt := r.buildSystemPrompt("")
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
