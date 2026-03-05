package runtime

import (
	"github.com/dev/agent-runtime/internal/planner"
)

type Session struct {
	ID                   string
	History              []planner.Message
	PendingToolCalls     []planner.ToolCall
	PendingIndex         int
	PendingAssistantMsg  *planner.Message
	AwaitingConfirmation bool
	LoopState            *LoopState
}

func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		History:   make([]planner.Message, 0),
		LoopState: NewLoopState(),
	}
}
