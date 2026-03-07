package context

import (
	"sort"
	"strings"

	"github.com/dev/agent-runtime/internal/planner"
)

// Priority levels for message scoring.
const (
	PrioritySystem    = 100 // System prompt — never dropped
	PriorityRecent    = 80  // Last few user/assistant messages
	PriorityToolCall  = 60  // Recent tool calls/results
	PriorityMidConvo  = 40  // Middle of conversation
	PriorityOldConvo  = 20  // Old conversation messages
	PriorityToolOld   = 10  // Old tool results (most expendable)
)

// scoredMessage pairs a message with its priority score.
type scoredMessage struct {
	msg      planner.Message
	priority int
	index    int // original position
	tokens   int // estimated token count
}

// Manager handles intelligent context window management.
type Manager struct {
	maxTokens       int
	reserveSystem   int // tokens reserved for system prompt
	reserveResponse int // tokens reserved for expected response
}

// New creates a context manager with the given max token budget.
func New(maxTokens int) *Manager {
	if maxTokens <= 0 {
		maxTokens = 128000 // sane default
	}
	return &Manager{
		maxTokens:       maxTokens,
		reserveSystem:   4000,
		reserveResponse: 4000,
	}
}

// TruncateMessages intelligently truncates messages to fit within the token budget.
// It scores messages by priority and removes lowest-priority ones first.
func (m *Manager) TruncateMessages(messages []planner.Message) []planner.Message {
	if len(messages) == 0 {
		return messages
	}

	budget := m.maxTokens - m.reserveResponse

	// Score all messages
	scored := make([]scoredMessage, len(messages))
	totalTokens := 0
	for i, msg := range messages {
		tokens := estimateTokens(msg.Content)
		scored[i] = scoredMessage{
			msg:      msg,
			priority: scoreMessage(msg, i, len(messages)),
			index:    i,
			tokens:   tokens,
		}
		totalTokens += tokens
	}

	// If within budget, return as-is
	if totalTokens <= budget {
		return messages
	}

	// Sort by priority (lowest first — these get dropped)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].priority < scored[j].priority
	})

	// Drop lowest-priority messages until within budget
	dropped := make(map[int]bool)
	for _, sm := range scored {
		if totalTokens <= budget {
			break
		}
		// Never drop system messages
		if sm.msg.Role == "system" {
			continue
		}
		// Never drop the last user message
		if sm.index == len(messages)-1 && sm.msg.Role == "user" {
			continue
		}
		// Don't drop if it would orphan a tool call
		if sm.msg.Role == "tool" {
			continue
		}

		dropped[sm.index] = true
		totalTokens -= sm.tokens
	}

	// Special pass: truncate long tool results instead of dropping them entirely
	for i := range scored {
		if totalTokens <= budget {
			break
		}
		if scored[i].msg.Role == "tool" && !dropped[scored[i].index] && scored[i].tokens > 500 {
			// Truncate tool output to 500 tokens worth (~2000 chars)
			content := scored[i].msg.Content
			if len(content) > 2000 {
				scored[i].msg.Content = content[:2000] + "\n...[TRUNCATED]"
				saved := scored[i].tokens - 500
				totalTokens -= saved
				scored[i].tokens = 500
			}
		}
	}

	// If still over budget, drop old tool messages
	if totalTokens > budget {
		for i := range scored {
			if totalTokens <= budget {
				break
			}
			if scored[i].msg.Role == "tool" && !dropped[scored[i].index] {
				dropped[scored[i].index] = true
				totalTokens -= scored[i].tokens
			}
		}
	}

	// Rebuild in original order, inserting summary where messages were dropped
	var result []planner.Message
	consecutiveDropped := 0
	for i, _ := range messages {
		if dropped[i] {
			consecutiveDropped++
			continue
		}
		if consecutiveDropped > 0 {
			result = append(result, planner.Message{
				Role:    "system",
				Content: "[Earlier messages omitted to fit context window]",
			})
			consecutiveDropped = 0
		}
		// Check if message was truncated
		for _, sm := range scored {
			if sm.index == i {
				result = append(result, sm.msg)
				break
			}
		}
	}

	return result
}

// SummarizeForCompact creates a summary-aware truncation that preserves key information.
func (m *Manager) SummarizeForCompact(messages []planner.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Role == "tool" {
			continue
		}
		preview := msg.Content
		if len(preview) > 300 {
			preview = preview[:300] + "..."
		}
		sb.WriteString("[" + msg.Role + "]: " + preview + "\n")
	}
	return sb.String()
}

// estimateTokens provides a rough token count (~4 chars per token for English).
func estimateTokens(text string) int {
	return len(text) / 4
}

// scoreMessage assigns a priority score to a message based on its position and role.
func scoreMessage(msg planner.Message, index, total int) int {
	// System messages are always highest priority
	if msg.Role == "system" {
		return PrioritySystem
	}

	// Calculate position ratio (0.0 = oldest, 1.0 = newest)
	posRatio := float64(index) / float64(total)

	// Recent messages (last 20%) get high priority
	if posRatio > 0.8 {
		if msg.Role == "tool" {
			return PriorityToolCall
		}
		return PriorityRecent
	}

	// Middle of conversation
	if posRatio > 0.4 {
		if msg.Role == "tool" {
			return PriorityToolOld + 5
		}
		return PriorityMidConvo
	}

	// Old messages
	if msg.Role == "tool" {
		return PriorityToolOld
	}
	return PriorityOldConvo
}
