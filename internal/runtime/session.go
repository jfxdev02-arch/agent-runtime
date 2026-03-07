package runtime

import (
	"fmt"
	"time"

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
	// Streaming: if true, use SSE streaming for responses
	Streaming bool `json:"streaming"`
}

func DefaultSessionSettings() SessionSettings {
	return SessionSettings{
		ThinkLevel: "medium",
		Verbose:    false,
		Streaming:  false,
	}
}

// ProgressPhase represents a phase in the agentic loop.
type ProgressPhase string

const (
	PhaseThinking  ProgressPhase = "thinking"
	PhaseToolStart ProgressPhase = "tool_start"
	PhaseToolEnd   ProgressPhase = "tool_end"
	PhaseToken     ProgressPhase = "token"
	PhaseError     ProgressPhase = "error"
	PhaseStatus    ProgressPhase = "status"
)

// ProgressEvent is emitted during the agentic loop so callers can show real-time status.
type ProgressEvent struct {
	Phase    ProgressPhase
	ToolName string // set for tool_start / tool_end
	ToolArgs string // set for tool_start (truncated)
	Message  string // human-readable status line
	Depth    int    // current turn depth
	Token    string // set for token phase (streaming)
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
	// Branching
	ParentID    string // ID of the parent session (empty if root)
	BranchPoint int    // Message index where this session branched
	BranchLabel string // Optional label for this branch
	CreatedAt   time.Time
}

func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		History:   make([]planner.Message, 0),
		LoopState: NewLoopState(),
		Settings:  DefaultSessionSettings(),
		CreatedAt: time.Now(),
	}
}

// Fork creates a new session branched from this session at the given message index.
// If msgIndex < 0, branches from the current position.
func (s *Session) Fork(newID string, msgIndex int, label string) *Session {
	if msgIndex < 0 || msgIndex > len(s.History) {
		msgIndex = len(s.History)
	}
	// Copy history up to branch point
	history := make([]planner.Message, msgIndex)
	copy(history, s.History[:msgIndex])

	return &Session{
		ID:          newID,
		History:     history,
		LoopState:   NewLoopState(),
		Settings:    s.Settings,
		ParentID:    s.ID,
		BranchPoint: msgIndex,
		BranchLabel: label,
		CreatedAt:   time.Now(),
	}
}

// BranchInfo returns metadata about this session's branch status.
func (s *Session) BranchInfo() map[string]interface{} {
	return map[string]interface{}{
		"session_id":   s.ID,
		"parent_id":    s.ParentID,
		"branch_point": s.BranchPoint,
		"branch_label": s.BranchLabel,
		"messages":     len(s.History),
		"created_at":   s.CreatedAt.Format(time.RFC3339),
		"is_branch":    s.ParentID != "",
	}
}

// CheckpointState returns a serializable snapshot of the session state (excluding history).
func (s *Session) CheckpointState() map[string]interface{} {
	return map[string]interface{}{
		"settings":    s.Settings,
		"parent_id":   s.ParentID,
		"branch_label": s.BranchLabel,
		"branch_point": s.BranchPoint,
	}
}

// RestoreState applies a previously saved state to this session.
func (s *Session) RestoreState(state map[string]interface{}) {
	if v, ok := state["parent_id"].(string); ok {
		s.ParentID = v
	}
	if v, ok := state["branch_label"].(string); ok {
		s.BranchLabel = v
	}
	if v, ok := state["branch_point"].(float64); ok {
		s.BranchPoint = int(v)
	}
}

// NewMultimodalMessage creates a multimodal message with text and images.
func NewMultimodalMessage(text string, imageURLs []string) planner.Message {
	parts := []planner.ContentPart{
		{Type: "text", Text: text},
	}
	for _, url := range imageURLs {
		parts = append(parts, planner.ContentPart{
			Type:     "image_url",
			ImageURL: &planner.ImageURL{URL: url, Detail: "auto"},
		})
	}
	return planner.Message{
		Role:         "user",
		Content:      fmt.Sprintf("%s [+%d image(s)]", text, len(imageURLs)),
		MultiContent: parts,
	}
}
