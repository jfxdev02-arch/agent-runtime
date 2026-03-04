package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WorkspaceTool struct{}

func NewWorkspaceTool() *WorkspaceTool { return &WorkspaceTool{} }
func (t *WorkspaceTool) Name() string        { return "workspace" }
func (t *WorkspaceTool) Description() string { return "List files or read file contents from anywhere in the system." }
func (t *WorkspaceTool) Risk() string         { return "LOW" }

func (t *WorkspaceTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "action", Type: "string", Description: "Either 'list' or 'read'", Required: true},
		{Name: "path", Type: "string", Description: "Path (relative to workspace or absolute)", Required: false},
	}
}

func (t *WorkspaceTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	action := args["action"]
	path := args["path"]

	// Support both absolute and relative paths
	resolvePath := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(ctx.Workspace, p)
	}

	switch action {
	case "list":
		target := resolvePath(path)
		entries, err := os.ReadDir(target)
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		for _, e := range entries {
			info, _ := e.Info()
			p := "F"
			if e.IsDir() {
				p = "D"
			}
			if info != nil {
				sb.WriteString(fmt.Sprintf("[%s] %s (%d bytes)\n", p, e.Name(), info.Size()))
			}
		}
		return sb.String(), nil

	case "read":
		if path == "" {
			return "", fmt.Errorf("'path' required for read")
		}
		target := resolvePath(path)
		content, err := os.ReadFile(target)
		if err != nil {
			return "", err
		}
		if len(content) > 100000 {
			return string(content[:100000]) + "\n...[TRUNCATED]", nil
		}
		return string(content), nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}
