package planner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	// Multimodal: when set, overrides Content for the API request
	MultiContent []ContentPart `json:"-"`
}

// ContentPart represents a single part of a multimodal message.
type ContentPart struct {
	Type     string    `json:"type"`               // "text" or "image_url"
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

// ImageURL represents an image reference in a multimodal message.
type ImageURL struct {
	URL    string `json:"url"`              // data:image/png;base64,... or https://...
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// StreamCallback is called with each token chunk during streaming.
type StreamCallback func(token string)

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
	Messages    []interface{}    `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature"`
	Stream      bool             `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
		Delta        Message `json:"delta"`
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
		Messages:    convertMessages(messages),
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

// CallWithModelStream sends messages and streams the response token-by-token.
func (p *Planner) CallWithModelStream(messages []Message, toolDefs []ToolDefinition, model, authType string, onToken StreamCallback) (*Message, error) {
	if model == "" {
		model = "glm-5"
	}

	reqBody := ChatRequest{
		Model:       model,
		Messages:    convertMessages(messages),
		Tools:       toolDefs,
		Temperature: 0.2,
		Stream:      true,
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

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	var fullContent strings.Builder
	var toolCalls []ToolCall
	toolCallMap := make(map[int]*ToolCall)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk ChatResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta

		// Stream text content
		if delta.Content != "" {
			fullContent.WriteString(delta.Content)
			if onToken != nil {
				onToken(delta.Content)
			}
		}

		// Accumulate tool calls from deltas
		for _, tc := range delta.ToolCalls {
			// Tool calls in streaming come with an index
			idx := 0
			if existing, ok := toolCallMap[idx]; ok {
				existing.Function.Arguments += tc.Function.Arguments
			} else {
				newTC := ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: FunctionCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
				toolCallMap[idx] = &newTC
			}
		}
	}

	// Assemble final tool calls
	for _, tc := range toolCallMap {
		toolCalls = append(toolCalls, *tc)
	}

	return &Message{
		Role:      "assistant",
		Content:   fullContent.String(),
		ToolCalls: toolCalls,
	}, nil
}

// convertMessages converts Message slice to interface slice,
// handling multimodal content where needed.
func convertMessages(messages []Message) []interface{} {
	result := make([]interface{}, len(messages))
	for i, msg := range messages {
		if len(msg.MultiContent) > 0 && msg.Role == "user" {
			// Multimodal message
			result[i] = map[string]interface{}{
				"role":    msg.Role,
				"content": msg.MultiContent,
			}
		} else {
			// Standard message as a map to ensure proper serialization
			m := map[string]interface{}{
				"role": msg.Role,
			}
			if msg.Content != "" {
				m["content"] = msg.Content
			}
			if len(msg.ToolCalls) > 0 {
				m["tool_calls"] = msg.ToolCalls
			}
			if msg.ToolCallID != "" {
				m["tool_call_id"] = msg.ToolCallID
			}
			result[i] = m
		}
	}
	return result
}
