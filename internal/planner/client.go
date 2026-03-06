package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dev/agent-runtime/internal/tools"
)

type Planner struct {
	endpoint string
	apiKey   string
}

func NewPlanner(endpoint, apiKey string) *Planner {
	return &Planner{endpoint: endpoint, apiKey: apiKey}
}

// --- Message types for the OpenAI-compatible API ---

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// --- Tool definition for the API ---

type ToolDefinition struct {
	Type     string         `json:"type"`
	Function FunctionSchema `json:"function"`
}

type FunctionSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// --- Request/Response ---

type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature"`
}

type ChatResponse struct {
	Choices []struct {
		Message       Message `json:"message"`
		FinishReason  string  `json:"finish_reason"`
	} `json:"choices"`
}

// BuildToolDefinitions converts our tool registry into OpenAI-compatible tool definitions
func BuildToolDefinitions(registry *tools.Registry) []ToolDefinition {
	var defs []ToolDefinition
	for _, t := range registry.ListTools() {
		params := t.Parameters()
		properties := make(map[string]interface{})
		required := []string{}

		for _, p := range params {
			properties[p.Name] = map[string]string{
				"type":        p.Type,
				"description": p.Description,
			}
			if p.Required {
				required = append(required, p.Name)
			}
		}

		schema := map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}
		schemaJSON, _ := json.Marshal(schema)

		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  schemaJSON,
			},
		})
	}
	return defs
}

// Call sends messages to the LLM and returns its response (uses default model glm-5).
func (p *Planner) Call(messages []Message, toolDefs []ToolDefinition) (*Message, error) {
	return p.CallWithModel(messages, toolDefs, "glm-5", "bearer")
}

// CallWithModel sends messages to the LLM with a specific model and auth type.
func (p *Planner) CallWithModel(messages []Message, toolDefs []ToolDefinition, model, authType string) (*Message, error) {
	if model == "" {
		model = "glm-5"
	}

	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Tools:       toolDefs,
		Temperature: 0.2,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", p.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" && authType != "none" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	return &chatResp.Choices[0].Message, nil
}
