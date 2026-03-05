package runtime

import (
	"crypto/sha256"
	"encoding/hex"
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

type loopLimits struct {
	deadline    time.Time
	toolCalls   int
	history     []toolCallRecord
	warnBuckets map[string]int
}

type toolCallRecord struct {
	toolName   string
	argsHash   string
	resultHash string
}

type loopDetectionResult struct {
	stuck   bool
	warning bool
	key     string
	count   int
	message string
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
	s := r.GetSession(sessionID)
	limits := r.newLoopLimits()

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
func (r *Runtime) agenticLoop(s *Session, messages []planner.Message, depth int, limits *loopLimits) (string, bool) {
	if r.isTimedOut(limits) {
		return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
	}

	if depth >= r.cfg.MaxTurns {
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
		return r.finishWithMessage(s, text), false
	}

	// LLM requested tool calls — check for high-risk ones that need approval
	for _, tc := range resp.ToolCalls {
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

	return r.executeToolCalls(s, messages, resp, 0, r.newLoopLimits())
}

// executeToolCalls runs the actual tools, sends results back to LLM
func (r *Runtime) executeToolCalls(s *Session, messages []planner.Message, resp *planner.Message, depth int, limits *loopLimits) (string, bool) {
	if r.isTimedOut(limits) {
		return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
	}

	// Add the assistant's message (with tool_calls) to history
	s.History = append(s.History, *resp)
	messages = append(messages, *resp)

	var resultSummary strings.Builder

	// Execute each tool call
	for _, tc := range resp.ToolCalls {
		if r.cfg.MaxToolCalls > 0 && limits.toolCalls >= r.cfg.MaxToolCalls {
			msg := fmt.Sprintf("Tool call limit reached (%d). Stopping to avoid loops.", r.cfg.MaxToolCalls)
			return r.finishWithMessage(s, msg), false
		}
		if r.isTimedOut(limits) {
			return r.finishWithMessage(s, fmt.Sprintf("Global timeout reached (%ds). Stopping to avoid infinite loops.", r.cfg.MaxRunSeconds)), false
		}
		limits.toolCalls++

		tool, err := r.registry.Get(tc.Function.Name)

		var output string
		if err != nil {
			output = fmt.Sprintf("Tool '%s' not found", tc.Function.Name)
		} else {
			// Parse arguments
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
			} else {
				output = out
			}
			r.store.LogToolExecution(s.ID, tc.Function.Name, tc.Function.Arguments, output, "OK")
		}

		resultSummary.WriteString(fmt.Sprintf("[%s] %s\n", tc.Function.Name, output))

		argsHash := hashText(tc.Function.Name + "|" + strings.TrimSpace(tc.Function.Arguments))
		resultHash := hashText(strings.TrimSpace(output))
		r.recordToolCall(limits, tc.Function.Name, argsHash, resultHash)

		if det := r.detectToolLoop(limits, tc.Function.Name, argsHash, resultHash); det.stuck {
			msg := det.message
			if det.warning {
				msg = "Potential tool loop detected, but execution can continue: " + det.message
			} else {
				return r.finishWithMessage(s, msg), false
			}
			if r.shouldEmitLoopWarning(limits, det.key, det.count) {
				log.Printf("loop warning: %s", msg)
			}
		}

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

func (r *Runtime) newLoopLimits() *loopLimits {
	limits := &loopLimits{
		history:     make([]toolCallRecord, 0),
		warnBuckets: make(map[string]int),
	}
	if r.cfg.MaxRunSeconds > 0 {
		limits.deadline = time.Now().Add(time.Duration(r.cfg.MaxRunSeconds) * time.Second)
	}
	return limits
}

func (r *Runtime) isTimedOut(limits *loopLimits) bool {
	if limits == nil || limits.deadline.IsZero() {
		return false
	}
	return time.Now().After(limits.deadline)
}

func hashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func (r *Runtime) shouldEmitLoopWarning(limits *loopLimits, key string, count int) bool {
	if limits == nil || key == "" {
		return true
	}
	if limits.warnBuckets == nil {
		limits.warnBuckets = make(map[string]int)
	}
	bucket := count / 10
	if bucket < 1 {
		bucket = 1
	}
	if prev, ok := limits.warnBuckets[key]; ok && bucket <= prev {
		return false
	}
	limits.warnBuckets[key] = bucket
	return true
}

func (r *Runtime) recordToolCall(limits *loopLimits, toolName, argsHash, resultHash string) {
	if limits == nil {
		return
	}
	limits.history = append(limits.history, toolCallRecord{
		toolName:   toolName,
		argsHash:   argsHash,
		resultHash: resultHash,
	})

	max := r.cfg.LoopHistorySize
	if max <= 0 {
		max = 30
	}
	if len(limits.history) > max {
		limits.history = limits.history[len(limits.history)-max:]
	}
}

func (r *Runtime) detectToolLoop(limits *loopLimits, toolName, argsHash, resultHash string) loopDetectionResult {
	if limits == nil || len(limits.history) == 0 {
		return loopDetectionResult{}
	}

	warnAt := r.cfg.LoopWarnAt
	if warnAt <= 0 {
		warnAt = 10
	}
	criticalAt := r.cfg.LoopCriticalAt
	if criticalAt <= warnAt {
		criticalAt = warnAt + 1
	}
	globalAt := r.cfg.LoopGlobalAt
	if globalAt <= criticalAt {
		globalAt = criticalAt + 1
	}

	noProgressStreak := 0
	for i := len(limits.history) - 1; i >= 0; i-- {
		h := limits.history[i]
		if h.toolName != toolName || h.argsHash != argsHash {
			continue
		}
		if h.resultHash != resultHash {
			break
		}
		noProgressStreak++
	}
	if noProgressStreak >= globalAt {
		return loopDetectionResult{
			stuck:   true,
			warning: false,
			key:     "global:" + toolName + ":" + argsHash + ":" + resultHash,
			count:   noProgressStreak,
			message: fmt.Sprintf("CRITICAL: %s repeated identical no-progress outcomes %d times. Stopping to prevent runaway loop.", toolName, noProgressStreak),
		}
	}

	recentCount := 0
	for i := len(limits.history) - 1; i >= 0; i-- {
		h := limits.history[i]
		if h.toolName == toolName && h.argsHash == argsHash {
			recentCount++
		}
	}
	if recentCount >= warnAt {
		return loopDetectionResult{
			stuck:   true,
			warning: true,
			key:     "generic:" + toolName + ":" + argsHash,
			count:   recentCount,
			message: fmt.Sprintf("WARNING: %s called %d times with identical arguments.", toolName, recentCount),
		}
	}

	pingCount, pingNoProgress := detectPingPongStreak(limits.history)
	if pingCount >= criticalAt && pingNoProgress {
		return loopDetectionResult{
			stuck:   true,
			warning: false,
			key:     "pingpong",
			count:   pingCount,
			message: fmt.Sprintf("CRITICAL: alternating tool pattern repeated %d calls with no progress. Stopping to prevent ping-pong loop.", pingCount),
		}
	}
	if pingCount >= warnAt {
		return loopDetectionResult{
			stuck:   true,
			warning: true,
			key:     "pingpong",
			count:   pingCount,
			message: fmt.Sprintf("WARNING: alternating tool pattern detected (%d tail calls).", pingCount),
		}
	}

	return loopDetectionResult{}
}

func detectPingPongStreak(history []toolCallRecord) (int, bool) {
	if len(history) < 4 {
		return 0, false
	}
	last := history[len(history)-1]
	prev := history[len(history)-2]
	if last.argsHash == prev.argsHash {
		return 0, false
	}

	a := last.argsHash
	b := prev.argsHash
	count := 2
	for i := len(history) - 3; i >= 0; i-- {
		expected := a
		if count%2 == 1 {
			expected = b
		}
		if history[i].argsHash != expected {
			break
		}
		count++
	}

	if count < 4 {
		return 0, false
	}

	var hashA, hashB string
	noProgress := true
	for i := len(history) - count; i < len(history); i++ {
		h := history[i]
		if h.argsHash == a {
			if hashA == "" {
				hashA = h.resultHash
			} else if hashA != h.resultHash {
				noProgress = false
				break
			}
			continue
		}
		if h.argsHash == b {
			if hashB == "" {
				hashB = h.resultHash
			} else if hashB != h.resultHash {
				noProgress = false
				break
			}
			continue
		}
		noProgress = false
		break
	}

	if hashA == "" || hashB == "" {
		noProgress = false
	}
	return count, noProgress
}
