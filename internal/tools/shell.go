package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type ShellTool struct{}

func NewShellTool() *ShellTool { return &ShellTool{} }

func (t *ShellTool) Name() string        { return "shell" }
func (t *ShellTool) Description() string { return "Execute any shell command in the workspace. Supports pipes, redirects, chaining." }
func (t *ShellTool) Risk() string         { return "LOW" }

func (t *ShellTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "command", Type: "string", Description: "The full shell command to execute (bash -c)", Required: true},
	}
}

func (t *ShellTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	command := args["command"]
	if command == "" {
		return "", fmt.Errorf("empty command")
	}

	cmdCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	cmd.Dir = ctx.Workspace

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("timeout (60s) — command took too long, it may be a long-running daemon that should be run as a background service instead")
		}
		combined := out.String() + "\n" + stderr.String()
		return combined, fmt.Errorf("exit error: %v", err)
	}

	result := out.String()
	if stderr.Len() > 0 {
		result += "\nstderr: " + stderr.String()
	}
	if len(result) > 16000 {
		result = result[:16000] + "\n...[TRUNCATED]"
	}
	return result, nil
}
