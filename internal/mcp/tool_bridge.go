package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/dev/agent-runtime/internal/tools"
)

// MCPTool wraps an MCP server tool as a native Tool interface.
type MCPTool struct {
	qualifiedName string
	toolDef       ToolDef
	manager       *Manager
}

// NewMCPTool creates a Tool wrapper for an MCP tool.
func NewMCPTool(qualifiedName string, def ToolDef, mgr *Manager) *MCPTool {
	return &MCPTool{
		qualifiedName: qualifiedName,
		toolDef:       def,
		manager:       mgr,
	}
}

func (t *MCPTool) Name() string        { return t.qualifiedName }
func (t *MCPTool) Description() string  { return fmt.Sprintf("[MCP:%s] %s", t.toolDef.ServerName, t.toolDef.Description) }
func (t *MCPTool) Risk() string         { return "LOW" }

func (t *MCPTool) Parameters() []tools.ToolParam {
	// Parse inputSchema to extract parameters
	if t.toolDef.InputSchema == nil {
		return []tools.ToolParam{
			{Name: "args_json", Type: "string", Description: "JSON arguments for the MCP tool", Required: false},
		}
	}

	var schema struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(t.toolDef.InputSchema, &schema); err != nil {
		return []tools.ToolParam{
			{Name: "args_json", Type: "string", Description: "JSON arguments for the MCP tool", Required: false},
		}
	}

	reqSet := make(map[string]bool)
	for _, r := range schema.Required {
		reqSet[r] = true
	}

	var params []tools.ToolParam
	for name, propRaw := range schema.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		json.Unmarshal(propRaw, &prop)
		if prop.Type == "" {
			prop.Type = "string"
		}
		params = append(params, tools.ToolParam{
			Name:        name,
			Type:        prop.Type,
			Description: prop.Description,
			Required:    reqSet[name],
		})
	}
	return params
}

func (t *MCPTool) Execute(ctx tools.ToolContext, args map[string]string) (string, error) {
	// Convert string args to interface{} args for MCP
	mcpArgs := make(map[string]interface{})
	for k, v := range args {
		mcpArgs[k] = v
	}
	return t.manager.CallTool(t.qualifiedName, mcpArgs)
}

// RegisterMCPTools registers all MCP tools into a tool registry.
func RegisterMCPTools(mgr *Manager, registry *tools.Registry) int {
	mcpTools := mgr.ListTools()
	count := 0
	for _, td := range mcpTools {
		qualifiedName := fmt.Sprintf("mcp_%s_%s", td.ServerName, td.Name)
		tool := NewMCPTool(qualifiedName, td, mgr)
		registry.Register(tool)
		count++
	}
	return count
}
