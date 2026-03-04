package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

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
	r.sessions[id] = s
	return s
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
	return r.agenticLoop(s, messages, 0)
}

// agenticLoop calls the LLM, handles tool calls, feeds results back, repeats
func (r *Runtime) agenticLoop(s *Session, messages []planner.Message, depth int) (string, bool) {
	if depth >= r.cfg.MaxTurns {
		summary := fmt.Sprintf("Turn limit (%d) reached. All executed actions have been saved.", r.cfg.MaxTurns)
		s.History = append(s.History, planner.Message{Role: "assistant", Content: summary})
		r.store.LogMessage(s.ID, "assistant", summary)
		return summary, false
	}

	resp, err := r.llm.Call(messages, r.toolDefs)
	if err != nil {
		return fmt.Sprintf("LLM error: %v", err), false
	}

	// No tool calls: LLM gave a direct text response
	if len(resp.ToolCalls) == 0 {
		text := resp.Content
		s.History = append(s.History, planner.Message{Role: "assistant", Content: text})
		r.store.LogMessage(s.ID, "assistant", text)
		return text, false
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
	return r.executeToolCalls(s, messages, resp, depth)
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

	return r.executeToolCalls(s, messages, resp, 0)
}

// executeToolCalls runs the actual tools, sends results back to LLM
func (r *Runtime) executeToolCalls(s *Session, messages []planner.Message, resp *planner.Message, depth int) (string, bool) {
	// Add the assistant's message (with tool_calls) to history
	s.History = append(s.History, *resp)
	messages = append(messages, *resp)

	var resultSummary strings.Builder

	// Execute each tool call
	for _, tc := range resp.ToolCalls {
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
	return r.agenticLoop(s, messages, depth+1)
}
