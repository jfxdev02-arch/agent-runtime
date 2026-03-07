package streaming

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// EventType represents the type of SSE event.
type EventType string

const (
	EventThinking  EventType = "thinking"
	EventToken     EventType = "token"
	EventToolStart EventType = "tool_start"
	EventToolEnd   EventType = "tool_end"
	EventError     EventType = "error"
	EventDone      EventType = "done"
	EventStatus    EventType = "status"
)

// Event is a single SSE event sent to the client.
type Event struct {
	Type     EventType   `json:"type"`
	Data     string      `json:"data,omitempty"`
	Tool     string      `json:"tool,omitempty"`
	Args     string      `json:"args,omitempty"`
	Depth    int         `json:"depth,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// Writer wraps an http.ResponseWriter for SSE streaming.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
	mu      sync.Mutex
	closed  bool
}

// NewWriter creates a new SSE writer, sets appropriate headers,
// and returns nil if the ResponseWriter doesn't support flushing.
func NewWriter(w http.ResponseWriter) *Writer {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	return &Writer{w: w, flusher: flusher}
}

// Send writes a single SSE event.
func (sw *Writer) Send(event Event) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.closed {
		return fmt.Errorf("stream closed")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(sw.w, "event: %s\ndata: %s\n\n", event.Type, data)
	if err != nil {
		sw.closed = true
		return err
	}

	sw.flusher.Flush()
	return nil
}

// SendToken sends a single token in the stream.
func (sw *Writer) SendToken(token string) error {
	return sw.Send(Event{Type: EventToken, Data: token})
}

// SendDone sends the final event with the complete response.
func (sw *Writer) SendDone(fullText string) error {
	return sw.Send(Event{Type: EventDone, Data: fullText})
}

// Close marks the writer as closed.
func (sw *Writer) Close() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.closed = true
}

// StreamCallback is called by the planner with each token chunk.
type StreamCallback func(token string)
