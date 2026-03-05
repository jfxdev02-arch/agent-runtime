package runtime

import (
	"testing"

	"github.com/dev/agent-runtime/internal/config"
)

func newTestRuntime(cfg *config.Config) *Runtime {
	return &Runtime{cfg: cfg}
}

func TestRecordToolCallTrimsHistory(t *testing.T) {
	r := newTestRuntime(&config.Config{LoopHistorySize: 3})
	limits := r.newLoopLimits()

	for i := 0; i < 5; i++ {
		r.recordToolCall(limits, "read", hashText("args"+string(rune('a'+i))), hashText("out"+string(rune('a'+i))))
	}

	if got := len(limits.history); got != 3 {
		t.Fatalf("expected trimmed history size 3, got %d", got)
	}
}

func TestDetectToolLoopGlobalNoProgressCritical(t *testing.T) {
	r := newTestRuntime(&config.Config{
		LoopWarnAt:     2,
		LoopCriticalAt: 3,
		LoopGlobalAt:   4,
	})
	limits := r.newLoopLimits()

	argsHash := hashText("process|poll")
	resultHash := hashText("steady")
	for i := 0; i < 4; i++ {
		r.recordToolCall(limits, "process", argsHash, resultHash)
	}

	det := r.detectToolLoop(limits, "process", argsHash, resultHash)
	if !det.stuck {
		t.Fatalf("expected stuck=true for global no-progress breaker")
	}
	if det.warning {
		t.Fatalf("expected critical (warning=false), got warning=true")
	}
	if det.count != 4 {
		t.Fatalf("expected count 4, got %d", det.count)
	}
}

func TestDetectToolLoopGenericRepeatWarning(t *testing.T) {
	r := newTestRuntime(&config.Config{
		LoopWarnAt:     3,
		LoopCriticalAt: 6,
		LoopGlobalAt:   10,
	})
	limits := r.newLoopLimits()

	argsHash := hashText("read|/same.txt")
	r.recordToolCall(limits, "read", argsHash, hashText("r1"))
	r.recordToolCall(limits, "read", argsHash, hashText("r2"))
	r.recordToolCall(limits, "read", argsHash, hashText("r3"))

	det := r.detectToolLoop(limits, "read", argsHash, hashText("r3"))
	if !det.stuck {
		t.Fatalf("expected stuck=true for generic repeat warning")
	}
	if !det.warning {
		t.Fatalf("expected warning=true for generic repeat")
	}
	if det.count != 3 {
		t.Fatalf("expected count 3, got %d", det.count)
	}
}

func TestDetectToolLoopPingPongCriticalNoProgress(t *testing.T) {
	r := newTestRuntime(&config.Config{
		LoopWarnAt:     3,
		LoopCriticalAt: 4,
		LoopGlobalAt:   20,
	})
	limits := r.newLoopLimits()

	argsA := hashText("read|/a.txt")
	argsB := hashText("list|/workspace")
	resA := hashText("A-steady")
	resB := hashText("B-steady")

	r.recordToolCall(limits, "read", argsA, resA)
	r.recordToolCall(limits, "list", argsB, resB)
	r.recordToolCall(limits, "read", argsA, resA)
	r.recordToolCall(limits, "list", argsB, resB)

	det := r.detectToolLoop(limits, "list", argsB, resB)
	if !det.stuck {
		t.Fatalf("expected stuck=true for ping-pong critical")
	}
	if det.warning {
		t.Fatalf("expected warning=false for ping-pong critical, got %+v", det)
	}
}

func TestDetectToolLoopPingPongWarningWhenProgressing(t *testing.T) {
	r := newTestRuntime(&config.Config{
		LoopWarnAt:     3,
		LoopCriticalAt: 6,
		LoopGlobalAt:   20,
	})
	limits := r.newLoopLimits()

	argsA := hashText("read|/a.txt")
	argsB := hashText("list|/workspace")

	r.recordToolCall(limits, "read", argsA, hashText("A1"))
	r.recordToolCall(limits, "list", argsB, hashText("B1"))
	r.recordToolCall(limits, "read", argsA, hashText("A2"))
	r.recordToolCall(limits, "list", argsB, hashText("B2"))

	det := r.detectToolLoop(limits, "list", argsB, hashText("B2"))
	if !det.stuck {
		t.Fatalf("expected stuck=true for ping-pong warning, got %+v", det)
	}
	if !det.warning {
		t.Fatalf("expected warning=true when ping-pong has progress, got %+v", det)
	}
}

func TestShouldEmitLoopWarningBucketed(t *testing.T) {
	r := newTestRuntime(&config.Config{})
	limits := r.newLoopLimits()

	if !r.shouldEmitLoopWarning(limits, "generic:read", 9) {
		t.Fatalf("expected first warning emission")
	}
	if r.shouldEmitLoopWarning(limits, "generic:read", 10) {
		t.Fatalf("expected suppression in same bucket")
	}
	if !r.shouldEmitLoopWarning(limits, "generic:read", 20) {
		t.Fatalf("expected emission when bucket increases")
	}
}
