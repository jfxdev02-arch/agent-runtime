package tools

import "fmt"

type EchoTool struct{}

func NewEchoTool() *EchoTool { return &EchoTool{} }

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "Echoes back the input text. For testing." }
func (t *EchoTool) Risk() string         { return "LOW" }

func (t *EchoTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "text", Type: "string", Description: "Text to echo back", Required: true},
	}
}

func (t *EchoTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	text := args["text"]
	if text == "" {
		return "", fmt.Errorf("missing 'text' parameter")
	}
	return fmt.Sprintf("Echo: %s", text), nil
}
