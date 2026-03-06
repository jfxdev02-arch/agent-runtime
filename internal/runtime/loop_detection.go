package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// ---- Types ----

// DetectorKind identifies which detector triggered a loop detection.
type DetectorKind string

const (
	DetectorGenericRepeat        DetectorKind = "generic_repeat"
	DetectorKnownPollNoProgress  DetectorKind = "known_poll_no_progress"
	DetectorGlobalCircuitBreaker DetectorKind = "global_circuit_breaker"
	DetectorPingPong             DetectorKind = "ping_pong"
)

// DetectionLevel indicates whether a detection is a soft warning or a hard block.
type DetectionLevel string

const (
	LevelWarning  DetectionLevel = "warning"
	LevelCritical DetectionLevel = "critical"
)

// LoopDetectionResult is the outcome of running loop detection before a tool call.
type LoopDetectionResult struct {
	Stuck          bool
	Level          DetectionLevel
	Detector       DetectorKind
	Count          int
	Message        string
	PairedToolName string
	WarningKey     string
}

// LoopDetectionConfig holds thresholds and feature toggles.
type LoopDetectionConfig struct {
	Enabled                      bool
	HistorySize                  int
	WarningThreshold             int
	CriticalThreshold            int
	GlobalCircuitBreakerThreshold int
	Detectors                    DetectorsConfig
}

// DetectorsConfig controls which detectors are active.
type DetectorsConfig struct {
	GenericRepeat       bool
	KnownPollNoProgress bool
	PingPong            bool
}

// ToolCallRecord is one entry in the sliding-window history.
type ToolCallRecord struct {
	ToolName   string
	ArgsHash   string
	ToolCallID string
	ResultHash string // empty until outcome is recorded
	Timestamp  int64
}

// LoopState holds per-session state for loop detection.
// Stored in the Session struct, analogous to openclaw's SessionState.
type LoopState struct {
	ToolCallHistory    []ToolCallRecord
	WarningBuckets     map[string]int
}

// ---- Default config ----

const (
	DefaultHistorySize                  = 30
	DefaultWarningThreshold             = 10
	DefaultCriticalThreshold            = 20
	DefaultGlobalCircuitBreakerThreshold = 30
)

func DefaultLoopDetectionConfig() LoopDetectionConfig {
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

// ResolveConfig normalises a config, enforcing threshold ordering.
func ResolveConfig(cfg LoopDetectionConfig) LoopDetectionConfig {
	c := cfg
	if c.HistorySize <= 0 {
		c.HistorySize = DefaultHistorySize
	}
	if c.WarningThreshold <= 0 {
		c.WarningThreshold = DefaultWarningThreshold
	}
	if c.CriticalThreshold <= c.WarningThreshold {
		c.CriticalThreshold = c.WarningThreshold + 1
	}
	if c.GlobalCircuitBreakerThreshold <= c.CriticalThreshold {
		c.GlobalCircuitBreakerThreshold = c.CriticalThreshold + 1
	}
	return c
}

// NewLoopState creates a fresh loop detection state.
func NewLoopState() *LoopState {
	return &LoopState{
		ToolCallHistory: make([]ToolCallRecord, 0),
		WarningBuckets:  make(map[string]int),
	}
}

// ---- Hashing ----

// HashToolCall produces a deterministic hash of toolName + args text.
func HashToolCall(toolName, argsJSON string) string {
	return toolName + ":" + digestSHA256(toolName+"|"+strings.TrimSpace(argsJSON))
}

// HashOutcome hashes a tool execution's output (or error).
func HashOutcome(output string, isError bool) string {
	prefix := "ok"
	if isError {
		prefix = "error"
	}
	return prefix + ":" + digestSHA256(strings.TrimSpace(output))
}

func digestSHA256(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// ---- Known poll tools ----

// isKnownPollTool returns true for tools that are expected to be called
// repeatedly for checking status (like command_status or process polling).
func isKnownPollTool(toolName string) bool {
	switch toolName {
	case "command_status", "process_poll", "process_log":
		return true
	}
	return false
}

// ---- Recording ----

// RecordToolCall adds a pending tool call to the history (before execution).
// The ResultHash is left empty and filled later by RecordToolCallOutcome.
func RecordToolCall(state *LoopState, toolName, argsHash, toolCallID string, cfg LoopDetectionConfig) {
	resolved := ResolveConfig(cfg)
	state.ToolCallHistory = append(state.ToolCallHistory, ToolCallRecord{
		ToolName:   toolName,
		ArgsHash:   argsHash,
		ToolCallID: toolCallID,
		Timestamp:  time.Now().UnixMilli(),
	})
	trimHistory(state, resolved.HistorySize)
}

// RecordToolCallOutcome patches the most recent matching record with its result hash.
// If no pending record is found, it appends a new complete record.
func RecordToolCallOutcome(state *LoopState, toolName, argsHash, toolCallID, resultHash string, cfg LoopDetectionConfig) {
	resolved := ResolveConfig(cfg)

	// Find the most recent matching pending record (no resultHash yet)
	matched := false
	for i := len(state.ToolCallHistory) - 1; i >= 0; i-- {
		r := &state.ToolCallHistory[i]
		if toolCallID != "" && r.ToolCallID != toolCallID {
			continue
		}
		if r.ToolName != toolName || r.ArgsHash != argsHash {
			continue
		}
		if r.ResultHash != "" {
			continue
		}
		r.ResultHash = resultHash
		matched = true
		break
	}

	if !matched {
		state.ToolCallHistory = append(state.ToolCallHistory, ToolCallRecord{
			ToolName:   toolName,
			ArgsHash:   argsHash,
			ToolCallID: toolCallID,
			ResultHash: resultHash,
			Timestamp:  time.Now().UnixMilli(),
		})
	}

	trimHistory(state, resolved.HistorySize)
}

func trimHistory(state *LoopState, maxSize int) {
	if len(state.ToolCallHistory) > maxSize {
		state.ToolCallHistory = state.ToolCallHistory[len(state.ToolCallHistory)-maxSize:]
	}
}

// ---- Detection (runs BEFORE tool execution, like openclaw's beforeToolCall) ----

// DetectToolCallLoop checks the session history and returns a detection result.
// This is called BEFORE executing the tool so we can block or warn without
// wasting execution time.
func DetectToolCallLoop(state *LoopState, toolName, argsHash string, cfg LoopDetectionConfig) LoopDetectionResult {
	resolved := ResolveConfig(cfg)
	if !resolved.Enabled {
		return LoopDetectionResult{Stuck: false}
	}

	history := state.ToolCallHistory
	knownPoll := isKnownPollTool(toolName)

	// 1. Global circuit breaker: identical tool+args+result repeated N times
	noProgress := getNoProgressStreak(history, toolName, argsHash)
	if noProgress.count >= resolved.GlobalCircuitBreakerThreshold {
		log.Printf("[loop-detection] Global circuit breaker: %s repeated %d times with no progress", toolName, noProgress.count)
		return LoopDetectionResult{
			Stuck:    true,
			Level:    LevelCritical,
			Detector: DetectorGlobalCircuitBreaker,
			Count:    noProgress.count,
			Message:  fmt.Sprintf("CRITICAL: %s has repeated identical no-progress outcomes %d times. Session execution blocked by global circuit breaker to prevent runaway loops.", toolName, noProgress.count),
			WarningKey: fmt.Sprintf("global:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
		}
	}

	// 2. Known poll no-progress (critical)
	if knownPoll && resolved.Detectors.KnownPollNoProgress && noProgress.count >= resolved.CriticalThreshold {
		log.Printf("[loop-detection] Critical polling loop: %s repeated %d times", toolName, noProgress.count)
		return LoopDetectionResult{
			Stuck:    true,
			Level:    LevelCritical,
			Detector: DetectorKnownPollNoProgress,
			Count:    noProgress.count,
			Message:  fmt.Sprintf("CRITICAL: Called %s with identical arguments and no progress %d times. This appears to be a stuck polling loop. Session execution blocked to prevent resource waste.", toolName, noProgress.count),
			WarningKey: fmt.Sprintf("poll:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
		}
	}

	// 3. Known poll no-progress (warning)
	if knownPoll && resolved.Detectors.KnownPollNoProgress && noProgress.count >= resolved.WarningThreshold {
		log.Printf("[loop-detection] Polling loop warning: %s repeated %d times", toolName, noProgress.count)
		return LoopDetectionResult{
			Stuck:    true,
			Level:    LevelWarning,
			Detector: DetectorKnownPollNoProgress,
			Count:    noProgress.count,
			Message:  fmt.Sprintf("WARNING: You have called %s %d times with identical arguments and no progress. Stop polling and either (1) increase wait time between checks, or (2) report the task as failed if the process is stuck.", toolName, noProgress.count),
			WarningKey: fmt.Sprintf("poll:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
		}
	}

	// 4. Ping-pong detection
	pingPong := getPingPongStreak(history, argsHash)
	pingPongWarningKey := fmt.Sprintf("pingpong:%s:%s", toolName, argsHash)
	if pingPong.pairedSignature != "" {
		pingPongWarningKey = "pingpong:" + canonicalPairKey(argsHash, pingPong.pairedSignature)
	}

	if resolved.Detectors.PingPong && pingPong.count >= resolved.CriticalThreshold && pingPong.noProgressEvidence {
		log.Printf("[loop-detection] Critical ping-pong loop: alternating calls count=%d tool=%s", pingPong.count, toolName)
		return LoopDetectionResult{
			Stuck:          true,
			Level:          LevelCritical,
			Detector:       DetectorPingPong,
			Count:          pingPong.count,
			Message:        fmt.Sprintf("CRITICAL: You are alternating between repeated tool-call patterns (%d consecutive calls) with no progress. This appears to be a stuck ping-pong loop. Session execution blocked to prevent resource waste.", pingPong.count),
			PairedToolName: pingPong.pairedToolName,
			WarningKey:     pingPongWarningKey,
		}
	}

	if resolved.Detectors.PingPong && pingPong.count >= resolved.WarningThreshold {
		log.Printf("[loop-detection] Ping-pong loop warning: alternating calls count=%d tool=%s", pingPong.count, toolName)
		return LoopDetectionResult{
			Stuck:          true,
			Level:          LevelWarning,
			Detector:       DetectorPingPong,
			Count:          pingPong.count,
			Message:        fmt.Sprintf("WARNING: You are alternating between repeated tool-call patterns (%d consecutive calls). This looks like a ping-pong loop; stop retrying and report the task as failed.", pingPong.count),
			PairedToolName: pingPong.pairedToolName,
			WarningKey:     pingPongWarningKey,
		}
	}

	// 5. Generic repeat (warn-only, not for known poll tools)
	recentCount := 0
	for _, h := range history {
		if h.ToolName == toolName && h.ArgsHash == argsHash {
			recentCount++
		}
	}

	if !knownPoll && resolved.Detectors.GenericRepeat && recentCount >= resolved.WarningThreshold {
		log.Printf("[loop-detection] Generic repeat warning: %s called %d times with identical arguments", toolName, recentCount)
		return LoopDetectionResult{
			Stuck:      true,
			Level:      LevelWarning,
			Detector:   DetectorGenericRepeat,
			Count:      recentCount,
			Message:    fmt.Sprintf("WARNING: You have called %s %d times with identical arguments. If this is not making progress, stop retrying and report the task as failed.", toolName, recentCount),
			WarningKey: fmt.Sprintf("generic:%s:%s", toolName, argsHash),
		}
	}

	return LoopDetectionResult{Stuck: false}
}

// ---- Warning deduplication ----

// ShouldEmitWarning returns true if a warning with this key/count should be
// emitted (bucketed deduplication to avoid log spam).
func ShouldEmitWarning(state *LoopState, warningKey string, count int) bool {
	if warningKey == "" {
		return true
	}
	if state.WarningBuckets == nil {
		state.WarningBuckets = make(map[string]int)
	}
	bucket := count / 10
	if bucket < 1 {
		bucket = 1
	}
	if prev, ok := state.WarningBuckets[warningKey]; ok && bucket <= prev {
		return false
	}
	state.WarningBuckets[warningKey] = bucket
	return true
}

// ---- Stats ----

// ToolCallStats holds summary statistics about tool call history.
type ToolCallStats struct {
	TotalCalls     int
	UniquePatterns int
	MostFrequent   *ToolFrequency
}

// ToolFrequency pairs a tool name with its call count.
type ToolFrequency struct {
	ToolName string
	Count    int
}

// GetToolCallStats returns summary statistics for diagnostic/monitoring purposes.
func GetToolCallStats(state *LoopState) ToolCallStats {
	history := state.ToolCallHistory
	patterns := make(map[string]*ToolFrequency)

	for _, call := range history {
		key := call.ArgsHash
		if existing, ok := patterns[key]; ok {
			existing.Count++
		} else {
			patterns[key] = &ToolFrequency{ToolName: call.ToolName, Count: 1}
		}
	}

	var mostFrequent *ToolFrequency
	for _, p := range patterns {
		if mostFrequent == nil || p.Count > mostFrequent.Count {
			mostFrequent = p
		}
	}

	return ToolCallStats{
		TotalCalls:     len(history),
		UniquePatterns: len(patterns),
		MostFrequent:   mostFrequent,
	}
}

// ---- Internal helpers ----

type noProgressResult struct {
	count           int
	latestResultHash string
}

func getNoProgressStreak(history []ToolCallRecord, toolName, argsHash string) noProgressResult {
	var streak int
	var latestResultHash string

	for i := len(history) - 1; i >= 0; i-- {
		r := history[i]
		if r.ToolName != toolName || r.ArgsHash != argsHash {
			continue
		}
		if r.ResultHash == "" {
			continue
		}
		if latestResultHash == "" {
			latestResultHash = r.ResultHash
			streak = 1
			continue
		}
		if r.ResultHash != latestResultHash {
			break
		}
		streak++
	}

	return noProgressResult{count: streak, latestResultHash: latestResultHash}
}

type pingPongResult struct {
	count              int
	pairedToolName     string
	pairedSignature    string
	noProgressEvidence bool
}

func getPingPongStreak(history []ToolCallRecord, currentSignature string) pingPongResult {
	if len(history) < 2 {
		return pingPongResult{}
	}

	last := history[len(history)-1]

	// Find the other signature in the tail
	var otherSignature, otherToolName string
	for i := len(history) - 2; i >= 0; i-- {
		if history[i].ArgsHash != last.ArgsHash {
			otherSignature = history[i].ArgsHash
			otherToolName = history[i].ToolName
			break
		}
	}
	if otherSignature == "" {
		return pingPongResult{}
	}

	// Count alternating tail
	alternatingCount := 0
	for i := len(history) - 1; i >= 0; i-- {
		expected := last.ArgsHash
		if alternatingCount%2 != 0 {
			expected = otherSignature
		}
		if history[i].ArgsHash != expected {
			break
		}
		alternatingCount++
	}

	if alternatingCount < 2 {
		return pingPongResult{}
	}

	// The current call should match the "other" signature to continue the pattern
	if currentSignature != otherSignature {
		return pingPongResult{}
	}

	// Check no-progress evidence: stable result hashes on both sides
	tailStart := len(history) - alternatingCount
	if tailStart < 0 {
		tailStart = 0
	}
	var firstHashA, firstHashB string
	noProgressEvidence := true
	for i := tailStart; i < len(history); i++ {
		h := history[i]
		if h.ResultHash == "" {
			noProgressEvidence = false
			break
		}
		if h.ArgsHash == last.ArgsHash {
			if firstHashA == "" {
				firstHashA = h.ResultHash
			} else if firstHashA != h.ResultHash {
				noProgressEvidence = false
				break
			}
		} else if h.ArgsHash == otherSignature {
			if firstHashB == "" {
				firstHashB = h.ResultHash
			} else if firstHashB != h.ResultHash {
				noProgressEvidence = false
				break
			}
		} else {
			noProgressEvidence = false
			break
		}
	}

	if firstHashA == "" || firstHashB == "" {
		noProgressEvidence = false
	}

	return pingPongResult{
		count:              alternatingCount + 1, // +1 for the current incoming call
		pairedToolName:     otherToolName,
		pairedSignature:    last.ArgsHash,
		noProgressEvidence: noProgressEvidence,
	}
}

func canonicalPairKey(sigA, sigB string) string {
	pair := []string{sigA, sigB}
	sort.Strings(pair)
	return pair[0] + "|" + pair[1]
}
