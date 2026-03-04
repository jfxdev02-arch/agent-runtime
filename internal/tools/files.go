package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FilesTool struct{}

func NewFilesTool() *FilesTool { return &FilesTool{} }
func (t *FilesTool) Name() string        { return "files" }
func (t *FilesTool) Description() string { return "Write, patch, or delete files anywhere in the system." }
func (t *FilesTool) Risk() string         { return "LOW" }

func (t *FilesTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "action", Type: "string", Description: "Either 'write', 'patch', or 'delete'", Required: true},
		{Name: "filename", Type: "string", Description: "File path (relative to workspace or absolute)", Required: true},
		{Name: "content", Type: "string", Description: "Full file content for write action", Required: false},
		{Name: "old_text", Type: "string", Description: "Text to find for patch action", Required: false},
		{Name: "new_text", Type: "string", Description: "Replacement text for patch action", Required: false},
	}
}

func (t *FilesTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	action := args["action"]
	filename := args["filename"]
	if filename == "" {
		return "", fmt.Errorf("missing 'filename'")
	}

	// Support both absolute and relative paths
	target := filename
	if !filepath.IsAbs(target) {
		target = filepath.Join(ctx.Workspace, filename)
	}

	switch action {
	case "write":
		os.MkdirAll(filepath.Dir(target), 0755)
		content := args["content"]
		err := os.WriteFile(target, []byte(content), 0644)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("File '%s' written (%d bytes)", target, len(content)), nil

	case "patch":
		oldText := args["old_text"]
		newText := args["new_text"]
		if oldText == "" {
			return "", fmt.Errorf("'old_text' required for patch")
		}
		content, err := os.ReadFile(target)
		if err != nil {
			return "", err
		}
		if !strings.Contains(string(content), oldText) {
			return "", fmt.Errorf("old_text not found in file")
		}
		updated := strings.Replace(string(content), oldText, newText, 1)
		os.WriteFile(target, []byte(updated), 0644)
		return fmt.Sprintf("File '%s' patched", target), nil

	case "delete":
		err := os.Remove(target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("File '%s' deleted", target), nil

	default:
		return "", fmt.Errorf("unknown action: %s. Use 'write', 'patch', or 'delete'", action)
	}
}
