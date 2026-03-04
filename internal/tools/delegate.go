package tools

import (
	"encoding/json"
	"fmt"
)

type SubAgentTask struct {
	Role  string   `json:"role"`
	Task  string   `json:"task"`
	Tools []string `json:"tools,omitempty"`
}

type SubAgentResult struct {
	Role   string `json:"role"`
	Task   string `json:"task"`
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

type Executor interface {
	Execute(tasks []SubAgentTask, mode string, parentSessionID string, depth int) ([]SubAgentResult, error)
}

type DelegateTool struct {
	executor Executor
}

func NewDelegateTool(executor Executor) *DelegateTool {
	return &DelegateTool{executor: executor}
}

func (t *DelegateTool) Name() string { return "delegate" }
func (t *DelegateTool) Description() string {
	return "Delegate tasks to specialized subagents that run independently. Use for complex tasks that benefit from parallel or sequential decomposition. Each subagent gets a role, a task description, and optionally a restricted set of tools."
}
func (t *DelegateTool) Risk() string { return "LOW" }

func (t *DelegateTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "tasks", Type: "string", Description: `JSON array of tasks. Each task: {"role":"<role>","task":"<description>","tools":["tool1","tool2"]}. tools is optional (defaults to all).`, Required: true},
		{Name: "mode", Type: "string", Description: `Execution mode: "parallel" (all at once) or "sequential" (one by one). Default: "sequential".`, Required: false},
	}
}

func (t *DelegateTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	if ctx.Depth >= ctx.MaxDepth {
		return "", fmt.Errorf("max agent depth (%d) reached, cannot delegate further", ctx.MaxDepth)
	}

	tasksJSON := args["tasks"]
	if tasksJSON == "" {
		return "", fmt.Errorf("missing 'tasks' parameter")
	}

	var tasks []SubAgentTask
	if err := json.Unmarshal([]byte(tasksJSON), &tasks); err != nil {
		return "", fmt.Errorf("invalid tasks JSON: %v", err)
	}

	if len(tasks) == 0 {
		return "", fmt.Errorf("empty tasks list")
	}

	mode := args["mode"]
	if mode == "" {
		mode = "sequential"
	}
	if mode != "parallel" && mode != "sequential" {
		return "", fmt.Errorf("invalid mode '%s': use 'parallel' or 'sequential'", mode)
	}

	results, err := t.executor.Execute(tasks, mode, ctx.SessionID, ctx.Depth+1)
	if err != nil {
		return "", fmt.Errorf("orchestrator error: %v", err)
	}

	output, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %v", err)
	}

	return string(output), nil
}
