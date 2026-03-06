package runtime

import (
	"github.com/dev/agent-runtime/internal/planner"
)

// SessionSettings holds per-session configurable options.
type SessionSettings struct {
	// Model override: provider ID to use for this session (empty = default failover)
	ModelID string `json:"model_id"`
	// ThinkLevel: "off", "low", "medium", "high" — controls reasoning depth instruction
	ThinkLevel string `json:"think_level"`
	// Verbose: if true, include tool execution details in responses
	Verbose bool `json:"verbose"`
}

func DefaultSessionSettings() SessionSettings {
	return SessionSettings{
		ThinkLevel: "medium",
		Verbose:    false,
	}
}

type Session struct {
	ID                   string
	History              []planner.Message
	PendingToolCalls     []planner.ToolCall
	PendingIndex         int
	PendingAssistantMsg  *planner.Message
	AwaitingConfirmation bool
	LoopState            *LoopState
	Settings             SessionSettings
}

func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		History:   make([]planner.Message, 0),
		LoopState: NewLoopState(),
		Settings:  DefaultSessionSettings(),
	}
}
