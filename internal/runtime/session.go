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

// ProgressPhase represents a phase in the agentic loop.
type ProgressPhase string

const (
	PhaseThinking  ProgressPhase = "thinking"
	PhaseToolStart ProgressPhase = "tool_start"
	PhaseToolEnd   ProgressPhase = "tool_end"
	PhaseError     ProgressPhase = "error"
)

// ProgressEvent is emitted during the agentic loop so callers can show real-time status.
type ProgressEvent struct {
	Phase    ProgressPhase
	ToolName string // set for tool_start / tool_end
	ToolArgs string // set for tool_start (truncated)
	Message  string // human-readable status line
	Depth    int    // current turn depth
}

// ProgressCallback is called by the runtime whenever an agentic event occurs.
// Implementations should be non-blocking.
type ProgressCallback func(event ProgressEvent)

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
