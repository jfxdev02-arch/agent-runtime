package checkpoint

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Checkpoint represents a saved point in a session's execution.
type Checkpoint struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Label     string    `json:"label"`
	History   string    `json:"history"`   // JSON-encoded message history
	State     string    `json:"state"`     // JSON-encoded session state (loop state, settings, etc.)
	CreatedAt time.Time `json:"created_at"`
	MessageIndex int   `json:"message_index"` // Position in conversation
}

// Manager handles checkpoint creation, storage, and restoration.
type Manager struct {
	checkpoints map[string][]Checkpoint // sessionID -> checkpoints
	mu          sync.RWMutex
	maxPerSession int
}

// New creates a new checkpoint manager.
func New() *Manager {
	return &Manager{
		checkpoints:   make(map[string][]Checkpoint),
		maxPerSession: 50,
	}
}

// Save creates a checkpoint at the current conversation state.
func (m *Manager) Save(sessionID, label string, history interface{}, state interface{}) (string, error) {
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", fmt.Errorf("failed to marshal history: %v", err)
	}
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %v", err)
	}

	cp := Checkpoint{
		ID:        fmt.Sprintf("cp-%s-%d", sessionID, time.Now().UnixNano()),
		SessionID: sessionID,
		Label:     label,
		History:   string(historyJSON),
		State:     string(stateJSON),
		CreatedAt: time.Now(),
	}

	// Count messages for index
	var msgs []interface{}
	json.Unmarshal(historyJSON, &msgs)
	cp.MessageIndex = len(msgs)

	m.mu.Lock()
	defer m.mu.Unlock()

	cps := m.checkpoints[sessionID]
	cps = append(cps, cp)
	if len(cps) > m.maxPerSession {
		cps = cps[len(cps)-m.maxPerSession:]
	}
	m.checkpoints[sessionID] = cps

	log.Printf("[checkpoint] Saved %s for session %s (label=%s, msgs=%d)", cp.ID, sessionID, label, cp.MessageIndex)
	return cp.ID, nil
}

// List returns all checkpoints for a session, newest first.
func (m *Manager) List(sessionID string) []Checkpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cps := m.checkpoints[sessionID]
	result := make([]Checkpoint, len(cps))
	copy(result, cps)
	// Reverse to newest first
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Get returns a specific checkpoint by ID.
func (m *Manager) Get(sessionID, checkpointID string) (*Checkpoint, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, cp := range m.checkpoints[sessionID] {
		if cp.ID == checkpointID {
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("checkpoint %s not found", checkpointID)
}

// Restore retrieves the history and state from a checkpoint.
// Returns (historyJSON, stateJSON, error).
func (m *Manager) Restore(sessionID, checkpointID string) (string, string, error) {
	cp, err := m.Get(sessionID, checkpointID)
	if err != nil {
		return "", "", err
	}
	log.Printf("[checkpoint] Restoring %s for session %s (msgs=%d)", checkpointID, sessionID, cp.MessageIndex)
	return cp.History, cp.State, nil
}

// DeleteBefore removes all checkpoints before a given checkpoint ID.
func (m *Manager) DeleteBefore(sessionID, checkpointID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cps := m.checkpoints[sessionID]
	idx := -1
	for i, cp := range cps {
		if cp.ID == checkpointID {
			idx = i
			break
		}
	}
	if idx <= 0 {
		return 0
	}
	m.checkpoints[sessionID] = cps[idx:]
	return idx
}

// DeleteAll removes all checkpoints for a session.
func (m *Manager) DeleteAll(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.checkpoints, sessionID)
}

// Count returns the number of checkpoints for a session.
func (m *Manager) Count(sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.checkpoints[sessionID])
}

// SessionState captures the non-history parts of a session (for serialization).
type SessionState struct {
	Settings json.RawMessage `json:"settings"`
	Branch   string          `json:"branch,omitempty"`
}
