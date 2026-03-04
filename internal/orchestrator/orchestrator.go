package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/planner"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/tools"
)

type Orchestrator struct {
	cfg      *config.Config
	store    *storage.Storage
	registry *tools.Registry
	llm      *planner.Planner
}

func NewOrchestrator(cfg *config.Config, store *storage.Storage, registry *tools.Registry, llm *planner.Planner) *Orchestrator {
	return &Orchestrator{
		cfg:      cfg,
		store:    store,
		registry: registry,
		llm:      llm,
	}
}

// Execute implements tools.Executor
func (o *Orchestrator) Execute(tasks []tools.SubAgentTask, mode string, parentSessionID string, depth int) ([]tools.SubAgentResult, error) {
	if depth >= o.cfg.MaxAgentDepth {
		return nil, fmt.Errorf("max agent depth (%d) reached, cannot spawn more subagents", o.cfg.MaxAgentDepth)
	}

	switch mode {
	case "parallel":
		return o.runParallel(tasks, parentSessionID, depth)
	default:
		return o.runSequential(tasks, parentSessionID, depth)
	}
}

func (o *Orchestrator) runParallel(tasks []tools.SubAgentTask, parentSessionID string, depth int) ([]tools.SubAgentResult, error) {
	results := make([]tools.SubAgentResult, len(tasks))
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t tools.SubAgentTask) {
			defer wg.Done()
			sessionID := fmt.Sprintf("%s:sub:%d:%d", parentSessionID, idx, depth)
			results[idx] = o.runSingleAgent(t, sessionID, depth)
		}(i, task)
	}

	wg.Wait()
	return results, nil
}

func (o *Orchestrator) runSequential(tasks []tools.SubAgentTask, parentSessionID string, depth int) ([]tools.SubAgentResult, error) {
	results := make([]tools.SubAgentResult, 0, len(tasks))

	for i, task := range tasks {
		sessionID := fmt.Sprintf("%s:sub:%d:%d", parentSessionID, i, depth)
		results = append(results, o.runSingleAgent(task, sessionID, depth))
	}

	return results, nil
}

func (o *Orchestrator) runSingleAgent(task tools.SubAgentTask, sessionID string, depth int) tools.SubAgentResult {
	result := tools.SubAgentResult{
		Role: task.Role,
		Task: task.Task,
	}

	reg := o.buildSubRegistry(task, depth)
	toolDefs := planner.BuildToolDefinitions(reg)

	systemPrompt := o.buildSubAgentPrompt(task)
	messages := []planner.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: task.Task},
	}

	maxTurns := o.cfg.MaxTurns / (depth + 1)
	if maxTurns < 5 {
		maxTurns = 5
	}

	text, err := o.subAgentLoop(reg, toolDefs, messages, sessionID, depth, 0, maxTurns)
	if err != nil {
		result.Error = err.Error()
		log.Printf("[orchestrator] subagent error (role=%s, depth=%d): %v", task.Role, depth, err)
	} else {
		result.Result = text
	}

	return result
}

func (o *Orchestrator) buildSubRegistry(task tools.SubAgentTask, depth int) *tools.Registry {
	var reg *tools.Registry
	if len(task.Tools) > 0 {
		reg = o.registry.FilterByNames(task.Tools)
	} else {
		reg = o.registry.Clone()
	}

	if depth+1 >= o.cfg.MaxAgentDepth {
		filtered := tools.NewRegistry()
		for _, t := range reg.ListTools() {
			if t.Name() != "delegate" {
				filtered.Register(t)
			}
		}
		return filtered
	}

	return reg
}

func (o *Orchestrator) buildSubAgentPrompt(task tools.SubAgentTask) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("You are a specialized subagent with role: %s\n\n", task.Role))
	sb.WriteString("Your specific task is described in the user message below.\n")
	sb.WriteString("Focus ONLY on completing this task. Use the available tools as needed.\n")
	sb.WriteString("When done, provide a clear and complete summary of your results.\n")
	sb.WriteString("Be direct, technical, and efficient.\n")
	sb.WriteString("Respond in the same language as the task description.\n")
	return sb.String()
}

func (o *Orchestrator) subAgentLoop(
	reg *tools.Registry,
	toolDefs []planner.ToolDefinition,
	messages []planner.Message,
	sessionID string,
	agentDepth int,
	turn int,
	maxTurns int,
) (string, error) {
	if turn >= maxTurns {
		return fmt.Sprintf("Subagent turn limit (%d) reached.", maxTurns), nil
	}

	resp, err := o.llm.Call(messages, toolDefs)
	if err != nil {
		return "", fmt.Errorf("LLM error: %v", err)
	}

	if len(resp.ToolCalls) == 0 {
		o.store.LogMessage(sessionID, "assistant", resp.Content)
		return resp.Content, nil
	}

	messages = append(messages, *resp)

	for _, tc := range resp.ToolCalls {
		tool, toolErr := reg.Get(tc.Function.Name)

		var output string
		if toolErr != nil {
			output = fmt.Sprintf("Tool '%s' not found", tc.Function.Name)
		} else {
			args := make(map[string]string)
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			ctx := tools.ToolContext{
				SessionID: sessionID,
				Workspace: o.cfg.WorkspaceRoot,
				Depth:     agentDepth,
				MaxDepth:  o.cfg.MaxAgentDepth,
			}
			out, execErr := tool.Execute(ctx, args)
			if execErr != nil {
				output = fmt.Sprintf("Error: %v", execErr)
			} else {
				output = out
			}
			o.store.LogToolExecution(sessionID, tc.Function.Name, tc.Function.Arguments, output, "OK")
		}

		toolResultMsg := planner.Message{
			Role:       "tool",
			Content:    output,
			ToolCallID: tc.ID,
		}
		messages = append(messages, toolResultMsg)
	}

	return o.subAgentLoop(reg, toolDefs, messages, sessionID, agentDepth, turn+1, maxTurns)
}
