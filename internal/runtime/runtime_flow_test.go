package runtime

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/dev/agent-runtime/internal/config"
	"github.com/dev/agent-runtime/internal/planner"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/tools"
)

type chatRequestView struct {
	Messages []planner.Message `json:"messages"`
}

type highRiskEchoTool struct{}

func (t *highRiskEchoTool) Name() string        { return "high_risk_echo" }
func (t *highRiskEchoTool) Description() string { return "High-risk echo test tool" }
func (t *highRiskEchoTool) Risk() string        { return "HIGH" }
func (t *highRiskEchoTool) Parameters() []tools.ToolParam {
	return []tools.ToolParam{{Name: "text", Type: "string", Description: "text", Required: true}}
}
func (t *highRiskEchoTool) Execute(ctx tools.ToolContext, args map[string]string) (string, error) {
	return "HIGH: " + args["text"], nil
}

func newTestRuntimeWithServer(t *testing.T, handler http.HandlerFunc, extraTools ...tools.Tool) (*Runtime, *storage.Storage) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	tmp := t.TempDir()
	store, err := storage.NewStorage(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatalf("failed creating test storage: %v", err)
	}

	reg := tools.NewRegistry()
	reg.Register(tools.NewEchoTool())
	for _, tool := range extraTools {
		reg.Register(tool)
	}

	cfg := &config.Config{
		ZAIEndpoint:     srv.URL,
		ZAIApiKey:       "",
		WorkspaceRoot:   tmp,
		PromptsDir:      tmp,
		MaxHistory:      25,
		MaxTurns:        8,
		MaxRunSeconds:   0,
		MaxToolCalls:    10,
		LoopHistorySize: 30,
		LoopWarnAt:      10,
		LoopCriticalAt:  20,
		LoopGlobalAt:    30,
		MaxAgentDepth:   3,
	}

	llm := planner.NewPlanner(srv.URL, "")
	return NewRuntime(cfg, store, reg, llm), store
}

func TestProcessMessageHighRiskToolConfirmationYes(t *testing.T) {
	var calls int32
	rt, store := newTestRuntimeWithServer(t, func(w http.ResponseWriter, req *http.Request) {
		callNum := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if callNum == 1 {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-hr-1","type":"function","function":{"name":"high_risk_echo","arguments":"{\"text\":\"run\"}"}}]},"finish_reason":"tool_calls"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"confirmed and done"},"finish_reason":"stop"}]}`)
	}, &highRiskEchoTool{})

	firstOut, firstAwaiting := rt.ProcessMessage("sess-high-risk-yes", "do risky")
	if !firstAwaiting {
		t.Fatalf("expected awaiting=true on first high-risk turn")
	}
	if firstOut == "" || firstOut == "confirmed and done" {
		t.Fatalf("expected confirmation prompt, got %q", firstOut)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 planner call before confirmation, got %d", got)
	}

	secondOut, secondAwaiting := rt.ProcessMessage("sess-high-risk-yes", "YES")
	if secondAwaiting {
		t.Fatalf("expected awaiting=false after YES")
	}
	if secondOut != "confirmed and done" {
		t.Fatalf("expected final assistant output after YES, got %q", secondOut)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 planner calls total, got %d", got)
	}

	logs, err := store.GetRecentToolLogs(10)
	if err != nil {
		t.Fatalf("failed reading tool logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected tool execution after YES")
	}
	if logs[0].ToolName != "high_risk_echo" {
		t.Fatalf("expected high-risk tool log, got %s", logs[0].ToolName)
	}
}

func TestProcessMessageHighRiskToolConfirmationNo(t *testing.T) {
	var calls int32
	rt, _ := newTestRuntimeWithServer(t, func(w http.ResponseWriter, req *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-hr-2","type":"function","function":{"name":"high_risk_echo","arguments":"{\"text\":\"run\"}"}}]},"finish_reason":"tool_calls"}]}`)
	}, &highRiskEchoTool{})

	firstOut, firstAwaiting := rt.ProcessMessage("sess-high-risk-no", "do risky")
	if !firstAwaiting {
		t.Fatalf("expected awaiting=true on first high-risk turn")
	}
	if firstOut == "" {
		t.Fatalf("expected confirmation prompt message")
	}

	secondOut, secondAwaiting := rt.ProcessMessage("sess-high-risk-no", "NO")
	if secondAwaiting {
		t.Fatalf("expected awaiting=false after NO")
	}
	if secondOut != "Execution cancelled." {
		t.Fatalf("expected cancellation message, got %q", secondOut)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected no extra planner calls after NO, got %d", got)
	}

	s := rt.GetSession("sess-high-risk-no")
	if s.AwaitingConfirmation {
		t.Fatalf("expected AwaitingConfirmation=false after NO")
	}
	if s.PendingAssistantMsg != nil || s.PendingToolCalls != nil {
		t.Fatalf("expected pending state cleared after NO")
	}
}

func TestProcessMessageDirectAssistantReply(t *testing.T) {
	var calls int32
	rt, store := newTestRuntimeWithServer(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", req.Method)
		}
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}]}`)
	})

	out, awaiting := rt.ProcessMessage("sess-direct", "hello")
	if awaiting {
		t.Fatalf("expected awaiting=false")
	}
	if out != "done" {
		t.Fatalf("expected assistant output 'done', got %q", out)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected 1 planner call, got %d", got)
	}

	history, err := store.GetSessionMessages("sess-direct", 0)
	if err != nil {
		t.Fatalf("failed reading stored messages: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 stored messages (user+assistant), got %d", len(history))
	}
}

func TestProcessMessageToolCallThenFinalReply(t *testing.T) {
	var calls int32
	rt, store := newTestRuntimeWithServer(t, func(w http.ResponseWriter, req *http.Request) {
		callNum := atomic.AddInt32(&calls, 1)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed reading request body: %v", err)
		}

		if callNum == 2 {
			var parsed chatRequestView
			if err := json.Unmarshal(body, &parsed); err != nil {
				t.Fatalf("failed parsing second request: %v", err)
			}
			foundToolResult := false
			for _, m := range parsed.Messages {
				if m.Role == "tool" && m.Content == "Echo: hello" {
					foundToolResult = true
					break
				}
			}
			if !foundToolResult {
				t.Fatalf("expected second planner call to include tool result message")
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if callNum == 1 {
			_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"call-1","type":"function","function":{"name":"echo","arguments":"{\"text\":\"hello\"}"}}]},"finish_reason":"tool_calls"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"all done"},"finish_reason":"stop"}]}`)
	})

	out, awaiting := rt.ProcessMessage("sess-tool", "run tool")
	if awaiting {
		t.Fatalf("expected awaiting=false")
	}
	if out != "all done" {
		t.Fatalf("expected final assistant output 'all done', got %q", out)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected 2 planner calls, got %d", got)
	}

	logs, err := store.GetRecentToolLogs(10)
	if err != nil {
		t.Fatalf("failed reading tool logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatalf("expected at least one tool log")
	}
	if logs[0].ToolName != "echo" {
		t.Fatalf("expected tool log for echo, got %s", logs[0].ToolName)
	}
}
