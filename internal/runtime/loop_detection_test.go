package runtime

import (
	"strings"
	"testing"
)

// ---- Helpers ----

func enabledConfig() LoopDetectionConfig {
	return LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   DefaultHistorySize,
		WarningThreshold:              DefaultWarningThreshold,
		CriticalThreshold:             DefaultCriticalThreshold,
		GlobalCircuitBreakerThreshold: DefaultGlobalCircuitBreakerThreshold,
		Detectors: DetectorsConfig{
			GenericRepeat:       true,
			KnownPollNoProgress: true,
			PingPong:            true,
		},
	}
}

func shortConfig() LoopDetectionConfig {
	return LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   4,
		WarningThreshold:              DefaultWarningThreshold,
		CriticalThreshold:             DefaultCriticalThreshold,
		GlobalCircuitBreakerThreshold: DefaultGlobalCircuitBreakerThreshold,
		Detectors: DetectorsConfig{
			GenericRepeat:       true,
			KnownPollNoProgress: true,
			PingPong:            true,
		},
	}
}

func recordSuccessful(state *LoopState, toolName string, argsJSON string, output string, idx int, cfg LoopDetectionConfig) {
	argsHash := HashToolCall(toolName, argsJSON)
	toolCallID := toolName + "-" + string(rune('0'+idx))
	RecordToolCall(state, toolName, argsHash, toolCallID, cfg)
	RecordToolCallOutcome(state, toolName, argsHash, toolCallID, HashOutcome(output, false), cfg)
}

func recordRepeatedCalls(state *LoopState, toolName, argsJSON, output string, count int, cfg LoopDetectionConfig) {
	for i := 0; i < count; i++ {
		recordSuccessful(state, toolName, argsJSON, output, i, cfg)
	}
}

func detectAfterRepeated(toolName, argsJSON, output string, count int, cfg LoopDetectionConfig) LoopDetectionResult {
	state := NewLoopState()
	recordRepeatedCalls(state, toolName, argsJSON, output, count, cfg)
	argsHash := HashToolCall(toolName, argsJSON)
	return DetectToolCallLoop(state, toolName, argsHash, cfg)
}

// ---- HashToolCall Tests ----

func TestHashToolCallConsistent(t *testing.T) {
	h1 := HashToolCall("read", `{"path":"/file.txt"}`)
	h2 := HashToolCall("read", `{"path":"/file.txt"}`)
	if h1 != h2 {
		t.Fatalf("expected consistent hashes, got %s != %s", h1, h2)
	}
}

func TestHashToolCallDifferentParams(t *testing.T) {
	h1 := HashToolCall("read", `{"path":"/file1.txt"}`)
	h2 := HashToolCall("read", `{"path":"/file2.txt"}`)
	if h1 == h2 {
		t.Fatalf("expected different hashes for different params")
	}
}

func TestHashToolCallDifferentTools(t *testing.T) {
	h1 := HashToolCall("read", `{"path":"/file.txt"}`)
	h2 := HashToolCall("write", `{"path":"/file.txt"}`)
	if h1 == h2 {
		t.Fatalf("expected different hashes for different tools")
	}
}

// ---- RecordToolCall Tests ----

func TestRecordToolCallAddsToHistory(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()
	RecordToolCall(state, "read", HashToolCall("read", `{"path":"/file.txt"}`), "call-1", cfg)

	if len(state.ToolCallHistory) != 1 {
		t.Fatalf("expected 1 record, got %d", len(state.ToolCallHistory))
	}
	if state.ToolCallHistory[0].ToolName != "read" {
		t.Fatalf("expected tool name 'read', got %s", state.ToolCallHistory[0].ToolName)
	}
	if state.ToolCallHistory[0].ToolCallID != "call-1" {
		t.Fatalf("expected tool call ID 'call-1', got %s", state.ToolCallHistory[0].ToolCallID)
	}
}

func TestRecordToolCallSlidingWindow(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()

	for i := 0; i < DefaultHistorySize+10; i++ {
		RecordToolCall(state, "tool", HashToolCall("tool", `{"i":`+string(rune('0'+i))+`}`), "", cfg)
	}

	if len(state.ToolCallHistory) != DefaultHistorySize {
		t.Fatalf("expected history size %d, got %d", DefaultHistorySize, len(state.ToolCallHistory))
	}
}

func TestRecordToolCallCustomHistorySize(t *testing.T) {
	state := NewLoopState()
	cfg := shortConfig()

	for i := 0; i < 10; i++ {
		RecordToolCall(state, "tool", HashToolCall("tool", `{"i":`+string(rune('0'+i))+`}`), "", cfg)
	}

	if len(state.ToolCallHistory) != 4 {
		t.Fatalf("expected history size 4, got %d", len(state.ToolCallHistory))
	}
}

func TestRecordToolCallHasTimestamp(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()
	RecordToolCall(state, "tool", HashToolCall("tool", `{}`), "ts-call", cfg)

	ts := state.ToolCallHistory[0].Timestamp
	if ts <= 0 {
		t.Fatalf("expected positive timestamp, got %d", ts)
	}
}

// ---- RecordToolCallOutcome Tests ----

func TestRecordToolCallOutcomePatchesPending(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()
	argsHash := HashToolCall("shell", `{"cmd":"ls"}`)

	RecordToolCall(state, "shell", argsHash, "call-1", cfg)
	if state.ToolCallHistory[0].ResultHash != "" {
		t.Fatalf("expected empty resultHash before outcome")
	}

	RecordToolCallOutcome(state, "shell", argsHash, "call-1", HashOutcome("file1\nfile2", false), cfg)
	if state.ToolCallHistory[0].ResultHash == "" {
		t.Fatalf("expected resultHash to be patched after outcome")
	}
}

func TestRecordToolCallOutcomeAppendsIfNoMatch(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()
	argsHash := HashToolCall("shell", `{"cmd":"ls"}`)

	RecordToolCallOutcome(state, "shell", argsHash, "orphan-1", HashOutcome("output", false), cfg)
	if len(state.ToolCallHistory) != 1 {
		t.Fatalf("expected 1 record appended, got %d", len(state.ToolCallHistory))
	}
	if state.ToolCallHistory[0].ResultHash == "" {
		t.Fatalf("expected resultHash to be set on appended record")
	}
}

// ---- DetectToolCallLoop: disabled by default ----

func TestDetectDisabledByDefault(t *testing.T) {
	state := NewLoopState()
	cfg := LoopDetectionConfig{Enabled: false}

	for i := 0; i < 20; i++ {
		recordSuccessful(state, "read", `{"path":"/same.txt"}`, "same", i, cfg)
	}

	argsHash := HashToolCall("read", `{"path":"/same.txt"}`)
	result := DetectToolCallLoop(state, "read", argsHash, cfg)
	if result.Stuck {
		t.Fatalf("expected not stuck when detection is disabled")
	}
}

// ---- DetectToolCallLoop: unique calls don't trigger ----

func TestDetectUniqueCallsNoTrigger(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()

	for i := 0; i < 15; i++ {
		argsJSON := `{"path":"/file` + string(rune('A'+i)) + `.txt"}`
		recordSuccessful(state, "read", argsJSON, "output"+string(rune('A'+i)), i, cfg)
	}

	argsHash := HashToolCall("read", `{"path":"/new-file.txt"}`)
	result := DetectToolCallLoop(state, "read", argsHash, cfg)
	if result.Stuck {
		t.Fatalf("expected not stuck for unique calls")
	}
}

// ---- Generic repeat detector ----

func TestGenericRepeatWarning(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()

	argsJSON := `{"path":"/same.txt"}`
	for i := 0; i < DefaultWarningThreshold; i++ {
		RecordToolCall(state, "read", HashToolCall("read", argsJSON), "", cfg)
	}

	argsHash := HashToolCall("read", argsJSON)
	result := DetectToolCallLoop(state, "read", argsHash, cfg)

	if !result.Stuck {
		t.Fatalf("expected stuck=true for generic repeat")
	}
	if result.Level != LevelWarning {
		t.Fatalf("expected level=warning, got %s", result.Level)
	}
	if result.Detector != DetectorGenericRepeat {
		t.Fatalf("expected detector=generic_repeat, got %s", result.Detector)
	}
	if result.Count != DefaultWarningThreshold {
		t.Fatalf("expected count=%d, got %d", DefaultWarningThreshold, result.Count)
	}
	if !strings.Contains(result.Message, "WARNING") {
		t.Fatalf("expected WARNING in message, got %s", result.Message)
	}
}

func TestGenericRepeatKeepsWarningBelowGlobalBreaker(t *testing.T) {
	// Even at criticalThreshold, generic repeat stays warn-only
	result := detectAfterRepeated("read", `{"path":"/same.txt"}`, "same output", DefaultCriticalThreshold, enabledConfig())
	if !result.Stuck {
		t.Fatalf("expected stuck=true")
	}
	if result.Level != LevelWarning {
		t.Fatalf("expected level=warning for generic repeat below global breaker, got %s", result.Level)
	}
}

// ---- Known poll no-progress detector ----

func TestKnownPollNoProgressWarning(t *testing.T) {
	result := detectAfterRepeated("command_status", `{"sessionId":"s1"}`, "still running", DefaultWarningThreshold, enabledConfig())
	if !result.Stuck {
		t.Fatalf("expected stuck=true for poll no-progress")
	}
	if result.Level != LevelWarning {
		t.Fatalf("expected level=warning, got %s", result.Level)
	}
	if result.Detector != DetectorKnownPollNoProgress {
		t.Fatalf("expected detector=known_poll_no_progress, got %s", result.Detector)
	}
}

func TestKnownPollNoProgressCritical(t *testing.T) {
	result := detectAfterRepeated("command_status", `{"sessionId":"s1"}`, "still running", DefaultCriticalThreshold, enabledConfig())
	if !result.Stuck {
		t.Fatalf("expected stuck=true")
	}
	if result.Level != LevelCritical {
		t.Fatalf("expected level=critical, got %s", result.Level)
	}
	if result.Detector != DetectorKnownPollNoProgress {
		t.Fatalf("expected detector=known_poll_no_progress, got %s", result.Detector)
	}
}

// ---- Global circuit breaker ----

func TestGlobalCircuitBreaker(t *testing.T) {
	result := detectAfterRepeated("shell", `{"cmd":"bad"}`, "error timeout", DefaultGlobalCircuitBreakerThreshold, enabledConfig())
	if !result.Stuck {
		t.Fatalf("expected stuck=true for global circuit breaker")
	}
	if result.Level != LevelCritical {
		t.Fatalf("expected level=critical, got %s", result.Level)
	}
	if result.Detector != DetectorGlobalCircuitBreaker {
		t.Fatalf("expected detector=global_circuit_breaker, got %s", result.Detector)
	}
	if !strings.Contains(result.Message, "CRITICAL") {
		t.Fatalf("expected CRITICAL in message")
	}
}

// ---- Ping-pong detector ----

func TestPingPongWarning(t *testing.T) {
	state := NewLoopState()
	cfg := LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   30,
		WarningThreshold:              4,
		CriticalThreshold:             8,
		GlobalCircuitBreakerThreshold: 30,
		Detectors: DetectorsConfig{
			GenericRepeat:       true,
			KnownPollNoProgress: true,
			PingPong:            true,
		},
	}

	argsA := `{"path":"/a.txt"}`
	argsB := `{"dir":"/workspace"}`
	hashA := HashToolCall("read", argsA)
	hashB := HashToolCall("list", argsB)

	// Alternate: A, B, A, B (4 calls)
	for i := 0; i < 2; i++ {
		RecordToolCall(state, "read", hashA, "", cfg)
		RecordToolCallOutcome(state, "read", hashA, "", HashOutcome("contentA", false), cfg)
		RecordToolCall(state, "list", hashB, "", cfg)
		RecordToolCallOutcome(state, "list", hashB, "", HashOutcome("contentB", false), cfg)
	}

	// Next call would be "read" again — detect before executing
	result := DetectToolCallLoop(state, "read", hashA, cfg)
	if !result.Stuck {
		t.Fatalf("expected stuck=true for ping-pong")
	}
	if result.Detector != DetectorPingPong {
		t.Fatalf("expected detector=ping_pong, got %s", result.Detector)
	}
}

func TestPingPongCriticalNoProgress(t *testing.T) {
	state := NewLoopState()
	cfg := LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   30,
		WarningThreshold:              3,
		CriticalThreshold:             4,
		GlobalCircuitBreakerThreshold: 30,
		Detectors: DetectorsConfig{
			GenericRepeat:       true,
			KnownPollNoProgress: true,
			PingPong:            true,
		},
	}

	argsA := `{"path":"/a.txt"}`
	argsB := `{"dir":"/workspace"}`
	hashA := HashToolCall("read", argsA)
	hashB := HashToolCall("list", argsB)

	// Alternate with same results each time (no progress)
	for i := 0; i < 3; i++ {
		RecordToolCall(state, "read", hashA, "", cfg)
		RecordToolCallOutcome(state, "read", hashA, "", HashOutcome("A-steady", false), cfg)
		RecordToolCall(state, "list", hashB, "", cfg)
		RecordToolCallOutcome(state, "list", hashB, "", HashOutcome("B-steady", false), cfg)
	}

	result := DetectToolCallLoop(state, "read", hashA, cfg)
	if !result.Stuck {
		t.Fatalf("expected stuck=true for ping-pong critical")
	}
	if result.Level != LevelCritical {
		t.Fatalf("expected level=critical, got %s", result.Level)
	}
	if result.Detector != DetectorPingPong {
		t.Fatalf("expected detector=ping_pong, got %s", result.Detector)
	}
}

// ---- Custom thresholds ----

func TestCustomThresholds(t *testing.T) {
	cfg := LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   30,
		WarningThreshold:              2,
		CriticalThreshold:             4,
		GlobalCircuitBreakerThreshold: 30,
		Detectors: DetectorsConfig{
			GenericRepeat:       false,
			KnownPollNoProgress: true,
			PingPong:            false,
		},
	}

	// At warningThreshold
	warnResult := detectAfterRepeated("command_status", `{"sid":"s"}`, "running", 2, cfg)
	if !warnResult.Stuck || warnResult.Level != LevelWarning {
		t.Fatalf("expected warning at threshold 2")
	}

	// At criticalThreshold
	critResult := detectAfterRepeated("command_status", `{"sid":"s"}`, "running", 4, cfg)
	if !critResult.Stuck || critResult.Level != LevelCritical {
		t.Fatalf("expected critical at threshold 4")
	}
}

// ---- Detector toggle ----

func TestDisabledDetectors(t *testing.T) {
	cfg := LoopDetectionConfig{
		Enabled:                       true,
		HistorySize:                   30,
		WarningThreshold:              DefaultWarningThreshold,
		CriticalThreshold:             DefaultCriticalThreshold,
		GlobalCircuitBreakerThreshold: DefaultGlobalCircuitBreakerThreshold,
		Detectors: DetectorsConfig{
			GenericRepeat:       false,
			KnownPollNoProgress: false,
			PingPong:            false,
		},
	}

	result := detectAfterRepeated("command_status", `{"sid":"s"}`, "running", DefaultCriticalThreshold, cfg)
	// Only global circuit breaker would fire, but we're below that threshold
	if result.Stuck {
		t.Fatalf("expected not stuck when all detectors are disabled (below global threshold)")
	}
}

// ---- ShouldEmitWarning ----

func TestShouldEmitWarningBucketed(t *testing.T) {
	state := NewLoopState()

	if !ShouldEmitWarning(state, "generic:read", 9) {
		t.Fatalf("expected first warning emission")
	}
	if ShouldEmitWarning(state, "generic:read", 10) {
		t.Fatalf("expected suppression in same bucket")
	}
	if !ShouldEmitWarning(state, "generic:read", 20) {
		t.Fatalf("expected emission when bucket increases")
	}
}

// ---- GetToolCallStats ----

func TestGetToolCallStats(t *testing.T) {
	state := NewLoopState()
	cfg := enabledConfig()

	argsA := `{"path":"/a.txt"}`
	argsB := `{"path":"/b.txt"}`
	recordRepeatedCalls(state, "read", argsA, "outputA", 3, cfg)
	recordRepeatedCalls(state, "read", argsB, "outputB", 1, cfg)

	stats := GetToolCallStats(state)
	if stats.TotalCalls != 4 {
		t.Fatalf("expected 4 total calls, got %d", stats.TotalCalls)
	}
	if stats.UniquePatterns != 2 {
		t.Fatalf("expected 2 unique patterns, got %d", stats.UniquePatterns)
	}
	if stats.MostFrequent == nil || stats.MostFrequent.Count != 3 {
		t.Fatalf("expected most frequent count=3")
	}
}

// ---- ResolveConfig ----

func TestResolveConfigEnforcesOrdering(t *testing.T) {
	cfg := ResolveConfig(LoopDetectionConfig{
		Enabled:                       true,
		WarningThreshold:              10,
		CriticalThreshold:             5, // invalid, should be bumped
		GlobalCircuitBreakerThreshold: 3, // invalid, should be bumped
	})

	if cfg.CriticalThreshold <= cfg.WarningThreshold {
		t.Fatalf("critical (%d) should be > warning (%d)", cfg.CriticalThreshold, cfg.WarningThreshold)
	}
	if cfg.GlobalCircuitBreakerThreshold <= cfg.CriticalThreshold {
		t.Fatalf("global (%d) should be > critical (%d)", cfg.GlobalCircuitBreakerThreshold, cfg.CriticalThreshold)
	}
}

// ---- Tool-name burst detector ----

