package memory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dev/agent-runtime/internal/storage"
)

type MemoryAgent struct {
	endpoint string
	apiKey   string
}

func NewMemoryAgent(endpoint, apiKey string) *MemoryAgent {
	return &MemoryAgent{endpoint: endpoint, apiKey: apiKey}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

// RetrieveRelevantContext takes the current user message and older stored messages,
// asks the LLM to pick the most relevant ones, and returns a compact summary.
func (m *MemoryAgent) RetrieveRelevantContext(userMessage string, olderMsgs []storage.StoredMessage) (string, error) {
	if len(olderMsgs) == 0 {
		return "", nil
	}

	// Build a digest of older messages for the LLM to analyze
	var digest strings.Builder
	for i, msg := range olderMsgs {
		preview := msg.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		digest.WriteString(fmt.Sprintf("[%d] (%s) %s: %s\n", i+1, msg.CreatedAt, msg.Role, preview))
	}

	systemPrompt := `You are a memory retrieval agent. Analyze past conversation messages and select ONLY those relevant to the user's current question/task.

Rules:
1. Read the list of older messages provided.
2. Identify which past messages contain useful context for the current message.
3. Return a SHORT summary (max 300 words) of the relevant context found.
4. If nothing is relevant, respond with exactly: NO_RELEVANT_CONTEXT
5. Respond in the same language as the user's current message.
6. Do NOT invent information. Only summarize what exists in the messages.`

	userPrompt := fmt.Sprintf("Current user message:\n\"%s\"\n\nOlder messages from database:\n%s\n\nSummarize only the relevant context for the current message:", userMessage, digest.String())

	reqBody := chatRequest{
		Model: "glm-5",
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", m.endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("memory agent error %d: %s", resp.StatusCode, string(body))
	}

	type ChatResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", nil
	}

	result := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	if result == "NO_RELEVANT_CONTEXT" {
		return "", nil
	}
	return result, nil
}
