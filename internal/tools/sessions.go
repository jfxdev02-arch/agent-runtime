package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SessionCoordinator is the interface that the runtime must satisfy for sessions_* tools.
type SessionCoordinator interface {
	ListActiveSessions() []map[string]interface{}
	GetSessionHistory(sessionID string, limit int) string
	SendToSession(targetSessionID, message, senderSessionID string) (string, error)
}

// --- sessions_list ---

type SessionsListTool struct {
	coordinator SessionCoordinator
}

func NewSessionsListTool(coordinator SessionCoordinator) *SessionsListTool {
	return &SessionsListTool{coordinator: coordinator}
}

func (t *SessionsListTool) Name() string        { return "sessions_list" }
func (t *SessionsListTool) Description() string {
	return "List all active sessions/agents with their metadata. Use this to discover other sessions for agent-to-agent coordination."
}
func (t *SessionsListTool) Risk() string { return "LOW" }

func (t *SessionsListTool) Parameters() []ToolParam {
	return []ToolParam{}
}

func (t *SessionsListTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	sessions := t.coordinator.ListActiveSessions()
	if len(sessions) == 0 {
		return "No active sessions found.", nil
	}
	out, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sessions: %v", err)
	}
	return string(out), nil
}

// --- sessions_history ---

type SessionsHistoryTool struct {
	coordinator SessionCoordinator
}

func NewSessionsHistoryTool(coordinator SessionCoordinator) *SessionsHistoryTool {
	return &SessionsHistoryTool{coordinator: coordinator}
}

func (t *SessionsHistoryTool) Name() string        { return "sessions_history" }
func (t *SessionsHistoryTool) Description() string {
	return "Fetch the conversation transcript of another session. Use this to read what another agent/session has been working on."
}
func (t *SessionsHistoryTool) Risk() string { return "LOW" }

func (t *SessionsHistoryTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "session_id", Type: "string", Description: "The session ID to fetch history from", Required: true},
		{Name: "limit", Type: "string", Description: "Max number of messages to return (default: 20)", Required: false},
	}
}

func (t *SessionsHistoryTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	sessionID := strings.TrimSpace(args["session_id"])
	if sessionID == "" {
		return "", fmt.Errorf("missing 'session_id' parameter")
	}

	limit := 20
	if l := args["limit"]; l != "" {
		n := 0
		for _, c := range l {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			limit = n
		}
	}

	history := t.coordinator.GetSessionHistory(sessionID, limit)
	if history == "" {
		return fmt.Sprintf("No history found for session %s", sessionID), nil
	}
	return history, nil
}

// --- sessions_send ---

type SessionsSendTool struct {
	coordinator SessionCoordinator
}

func NewSessionsSendTool(coordinator SessionCoordinator) *SessionsSendTool {
	return &SessionsSendTool{coordinator: coordinator}
}

func (t *SessionsSendTool) Name() string        { return "sessions_send" }
func (t *SessionsSendTool) Description() string {
	return "Send a message to another session/agent and get its reply. Use this for agent-to-agent coordination — asking another session to perform a task or provide information."
}
func (t *SessionsSendTool) Risk() string { return "LOW" }

func (t *SessionsSendTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "target_session_id", Type: "string", Description: "The session ID to send the message to", Required: true},
		{Name: "message", Type: "string", Description: "The message to send to the target session", Required: true},
	}
}

func (t *SessionsSendTool) Execute(ctx ToolContext, args map[string]string) (string, error) {
	targetID := strings.TrimSpace(args["target_session_id"])
	if targetID == "" {
		return "", fmt.Errorf("missing 'target_session_id' parameter")
	}
	message := strings.TrimSpace(args["message"])
	if message == "" {
		return "", fmt.Errorf("missing 'message' parameter")
	}

	// Prevent sending to self
	if targetID == ctx.SessionID {
		return "", fmt.Errorf("cannot send message to own session")
	}

	reply, err := t.coordinator.SendToSession(targetID, message, ctx.SessionID)
	if err != nil {
		return "", fmt.Errorf("failed to send to session %s: %v", targetID, err)
	}

	return fmt.Sprintf("Reply from session %s:\n\n%s", targetID, reply), nil
}
