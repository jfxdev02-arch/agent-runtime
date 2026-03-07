package runtime

import (
"crypto/sha256"
"encoding/hex"
"fmt"
"log"
"math"
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
DetectorTokenBudget          DetectorKind = "token_budget"
DetectorToolBudget           DetectorKind = "tool_budget"
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
// SelfReflection is injected into the next LLM call as a system message
SelfReflection string
// BackoffMs is the recommended delay before the next tool call
BackoffMs int64
}

// LoopDetectionConfig holds thresholds and feature toggles.
type LoopDetectionConfig struct {
Enabled                       bool
HistorySize                   int
WarningThreshold              int
CriticalThreshold             int
GlobalCircuitBreakerThreshold int
Detectors                     DetectorsConfig
// Per-tool budgets: tool_name -> max calls per session
ToolBudgets map[string]int
// Token budget: max cumulative tokens per agentic run
TokenBudget int
// Backoff settings
BackoffBaseMs  int64
BackoffMaxMs   int64
BackoffEnabled bool
}

// DetectorsConfig controls which detectors are active.
type DetectorsConfig struct {
GenericRepeat       bool
KnownPollNoProgress bool
PingPong            bool
}

// ToolCallRecord is one entry in the sliding-window history.
type ToolCallRecord struct {
ToolName    string
ArgsHash    string
ToolCallID  string
ResultHash  string // empty until outcome is recorded
Timestamp   int64
TokensUsed  int
ExecutionMs int64
}

// LoopState holds per-session state for loop detection.
type LoopState struct {
ToolCallHistory []ToolCallRecord
WarningBuckets  map[string]int
// Token tracking (cumulative budget)
TotalTokensUsed int
TurnTokensUsed  int
// Per-tool call counts for budget enforcement
ToolCallCounts map[string]int
// Backoff state: consecutive no-progress count per tool signature
ConsecutiveNoProgress map[string]int
// Self-reflection tracking
ReflectionsSent int
// Abort flag: set by client-side streaming abort
Aborted     bool
AbortReason string
}

// ---- Default config ----

const (
DefaultHistorySize                   = 30
DefaultWarningThreshold              = 10
DefaultCriticalThreshold             = 20
DefaultGlobalCircuitBreakerThreshold = 30
DefaultTokenBudget                   = 500000
DefaultBackoffBaseMs                 = 500
DefaultBackoffMaxMs                  = 16000
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
ToolBudgets:    nil,
TokenBudget:    DefaultTokenBudget,
BackoffBaseMs:  DefaultBackoffBaseMs,
BackoffMaxMs:   DefaultBackoffMaxMs,
BackoffEnabled: true,
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
if c.TokenBudget <= 0 {
c.TokenBudget = DefaultTokenBudget
}
if c.BackoffBaseMs <= 0 {
c.BackoffBaseMs = DefaultBackoffBaseMs
}
if c.BackoffMaxMs <= 0 {
c.BackoffMaxMs = DefaultBackoffMaxMs
}
return c
}

// NewLoopState creates a fresh loop detection state.
func NewLoopState() *LoopState {
return &LoopState{
ToolCallHistory:       make([]ToolCallRecord, 0),
WarningBuckets:        make(map[string]int),
ToolCallCounts:        make(map[string]int),
ConsecutiveNoProgress: make(map[string]int),
}
}

// ---- Abort ----

// Abort sets the abort flag on the loop state (called from streaming client).
func (ls *LoopState) Abort(reason string) {
ls.Aborted = true
if reason == "" {
reason = "client requested abort"
}
ls.AbortReason = reason
}

// IsAborted returns true if the client has requested an abort.
func (ls *LoopState) IsAborted() bool {
return ls.Aborted
}

// ResetAbort clears the abort flag.
func (ls *LoopState) ResetAbort() {
ls.Aborted = false
ls.AbortReason = ""
}

// ---- Token Budget Tracking ----

// RecordTokenUsage adds tokens consumed to the running total.
func RecordTokenUsage(state *LoopState, inputTokens, outputTokens int) {
total := inputTokens + outputTokens
state.TotalTokensUsed += total
state.TurnTokensUsed += total
}

// CheckTokenBudget checks if the token budget has been exceeded.
func CheckTokenBudget(state *LoopState, cfg LoopDetectionConfig) LoopDetectionResult {
resolved := ResolveConfig(cfg)
if !resolved.Enabled || resolved.TokenBudget <= 0 {
return LoopDetectionResult{Stuck: false}
}

used := state.TotalTokensUsed
budget := resolved.TokenBudget

if used >= budget {
return LoopDetectionResult{
Stuck:    true,
Level:    LevelCritical,
Detector: DetectorTokenBudget,
Count:    used,
Message:  fmt.Sprintf("Token budget exhausted (%d/%d tokens used). Summarize your progress and stop.", used, budget),
SelfReflection: fmt.Sprintf(
"[BUDGET EXHAUSTED] You have consumed %d of %d available tokens. "+
"You MUST now: (1) summarize what you accomplished, (2) list any remaining tasks, "+
"(3) stop making tool calls. Do not attempt further actions.",
used, budget),
}
}

warningThreshold := int(float64(budget) * 0.8)
if used >= warningThreshold {
remaining := budget - used
return LoopDetectionResult{
Stuck:    true,
Level:    LevelWarning,
Detector: DetectorTokenBudget,
Count:    used,
Message:  fmt.Sprintf("Approaching token budget limit (%d/%d tokens, %d remaining).", used, budget, remaining),
SelfReflection: fmt.Sprintf(
"[BUDGET WARNING] You have used %d of %d tokens (%d remaining). "+
"Prioritize completing your current task efficiently. Avoid unnecessary tool calls.",
used, budget, remaining),
}
}

return LoopDetectionResult{Stuck: false}
}

// ---- Per-Tool Budget ----

// CheckToolBudget checks if a specific tool has exceeded its per-tool budget.
func CheckToolBudget(state *LoopState, toolName string, cfg LoopDetectionConfig) LoopDetectionResult {
resolved := ResolveConfig(cfg)
if !resolved.Enabled || len(resolved.ToolBudgets) == 0 {
return LoopDetectionResult{Stuck: false}
}

budget, hasBudget := resolved.ToolBudgets[toolName]
if !hasBudget {
return LoopDetectionResult{Stuck: false}
}

used := state.ToolCallCounts[toolName]
warningAt := int(float64(budget) * 0.8)

if used >= budget {
return LoopDetectionResult{
Stuck:    true,
Level:    LevelCritical,
Detector: DetectorToolBudget,
Count:    used,
Message:  fmt.Sprintf("Tool '%s' budget exhausted (%d/%d calls). Use a different approach.", toolName, used, budget),
SelfReflection: fmt.Sprintf(
"[TOOL BUDGET] You have used %s %d times (max %d). "+
"You cannot call this tool again. Try a different approach or report your progress.",
toolName, used, budget),
}
}
if used >= warningAt {
return LoopDetectionResult{
Stuck:    true,
Level:    LevelWarning,
Detector: DetectorToolBudget,
Count:    used,
Message:  fmt.Sprintf("Tool '%s' approaching budget limit (%d/%d calls).", toolName, used, budget),
SelfReflection: fmt.Sprintf(
"[TOOL BUDGET WARNING] You have used %s %d of %d allowed times. Use it carefully.",
toolName, used, budget),
}
}

return LoopDetectionResult{Stuck: false}
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

func isKnownPollTool(toolName string) bool {
switch toolName {
case "command_status", "process_poll", "process_log":
return true
}
return false
}

// ---- Backoff Calculation (exponential backoff) ----

// CalculateBackoff returns the delay in ms based on consecutive no-progress count.
// Formula: min(base * 2^(count-1), max)
func CalculateBackoff(consecutiveCount int, cfg LoopDetectionConfig) int64 {
resolved := ResolveConfig(cfg)
if !resolved.BackoffEnabled || consecutiveCount <= 1 {
return 0
}
delay := float64(resolved.BackoffBaseMs) * math.Pow(2, float64(consecutiveCount-2))
if delay > float64(resolved.BackoffMaxMs) {
delay = float64(resolved.BackoffMaxMs)
}
return int64(delay)
}

// ---- Self-Reflection Prompt Builder ----

// BuildSelfReflection generates a contextual self-reflection prompt injected into
// the next LLM system message. Instead of generic error messages, the LLM receives
// specific context about what went wrong and suggested actions.
func BuildSelfReflection(state *LoopState, det LoopDetectionResult, toolName string) string {
if det.SelfReflection != "" {
return det.SelfReflection
}

var sb strings.Builder
sb.WriteString("\n[SELF-REFLECTION — Loop Detection Active]\n")

switch det.Detector {
case DetectorGenericRepeat:
sb.WriteString(fmt.Sprintf(
"You have called '%s' %d times with identical arguments. Nothing has changed.\n"+
"ACTIONS TO CONSIDER:\n"+
"1. Try different arguments or a different tool\n"+
"2. Check if a previous step failed and needs a different approach\n"+
"3. Report your current progress and ask the user for guidance\n"+
"4. If the task is complete, stop calling tools and provide your answer\n",
toolName, det.Count))

case DetectorKnownPollNoProgress:
sb.WriteString(fmt.Sprintf(
"You have polled '%s' %d times with no change in results.\n"+
"ACTIONS TO CONSIDER:\n"+
"1. The process may be stuck — consider reporting failure\n"+
"2. If waiting for completion, increase the wait interval\n"+
"3. Try checking error logs or status from a different angle\n",
toolName, det.Count))

case DetectorPingPong:
paired := det.PairedToolName
if paired == "" {
paired = "another tool"
}
sb.WriteString(fmt.Sprintf(
"You are alternating between '%s' and '%s' in a loop (%d iterations) with no progress.\n"+
"ACTIONS TO CONSIDER:\n"+
"1. Break the cycle — try a completely different approach\n"+
"2. The underlying issue may require manual intervention\n"+
"3. Summarize what you've tried and ask the user for help\n",
toolName, paired, det.Count))

case DetectorGlobalCircuitBreaker:
sb.WriteString(fmt.Sprintf(
"CIRCUIT BREAKER TRIGGERED: '%s' has been called %d times with identical outcomes.\n"+
"This session is being stopped to prevent resource waste.\n"+
"Summarize what you accomplished and what remains.\n",
toolName, det.Count))

default:
sb.WriteString(fmt.Sprintf(
"A repetition pattern was detected for '%s' (count: %d).\n"+
"Consider changing your approach.\n",
toolName, det.Count))
}

// Add recent tool call summary
if len(state.ToolCallHistory) > 0 {
sb.WriteString("\nRecent tool calls:\n")
start := len(state.ToolCallHistory) - 5
if start < 0 {
start = 0
}
for _, h := range state.ToolCallHistory[start:] {
outcome := "pending"
if h.ResultHash != "" {
if strings.HasPrefix(h.ResultHash, "error:") {
outcome = "error"
} else {
outcome = "success"
}
}
sb.WriteString(fmt.Sprintf("  - %s [%s]\n", h.ToolName, outcome))
}
}

return sb.String()
}

// ---- Recording ----

// RecordToolCall adds a pending tool call to the history (before execution).
func RecordToolCall(state *LoopState, toolName, argsHash, toolCallID string, cfg LoopDetectionConfig) {
resolved := ResolveConfig(cfg)
state.ToolCallHistory = append(state.ToolCallHistory, ToolCallRecord{
ToolName:   toolName,
ArgsHash:   argsHash,
ToolCallID: toolCallID,
Timestamp:  time.Now().UnixMilli(),
})
state.ToolCallCounts[toolName]++
trimHistory(state, resolved.HistorySize)
}

// RecordToolCallOutcome patches the most recent matching record with its result hash.
func RecordToolCallOutcome(state *LoopState, toolName, argsHash, toolCallID, resultHash string, cfg LoopDetectionConfig) {
resolved := ResolveConfig(cfg)

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

// Update no-progress tracking for backoff calculation
sig := argsHash
noProgress := getNoProgressStreak(state.ToolCallHistory, toolName, argsHash)
if noProgress.count > 1 {
state.ConsecutiveNoProgress[sig] = noProgress.count
} else {
delete(state.ConsecutiveNoProgress, sig)
}
}

func trimHistory(state *LoopState, maxSize int) {
if len(state.ToolCallHistory) > maxSize {
state.ToolCallHistory = state.ToolCallHistory[len(state.ToolCallHistory)-maxSize:]
}
}

// ---- Detection (runs BEFORE tool execution) ----

// DetectToolCallLoop checks the session history and returns a detection result.
// Also calculates backoff delays and generates self-reflection prompts.
func DetectToolCallLoop(state *LoopState, toolName, argsHash string, cfg LoopDetectionConfig) LoopDetectionResult {
resolved := ResolveConfig(cfg)
if !resolved.Enabled {
return LoopDetectionResult{Stuck: false}
}

// 0. Check abort flag
if state.IsAborted() {
return LoopDetectionResult{
Stuck:   true,
Level:   LevelCritical,
Message: "Session aborted: " + state.AbortReason,
SelfReflection: "[ABORTED] The user has cancelled this operation. " +
"Summarize your progress so far and stop all tool calls immediately.",
}
}

// 1. Check per-tool budget
toolBudget := CheckToolBudget(state, toolName, cfg)
if toolBudget.Stuck {
toolBudget.SelfReflection = BuildSelfReflection(state, toolBudget, toolName)
return toolBudget
}

// 2. Check token budget (critical only — warning handled at end)
tokenBudget := CheckTokenBudget(state, cfg)
if tokenBudget.Stuck && tokenBudget.Level == LevelCritical {
return tokenBudget
}

history := state.ToolCallHistory
knownPoll := isKnownPollTool(toolName)

// 3. Global circuit breaker
noProgress := getNoProgressStreak(history, toolName, argsHash)
backoffMs := CalculateBackoff(noProgress.count, cfg)

if noProgress.count >= resolved.GlobalCircuitBreakerThreshold {
log.Printf("[loop-detection] Global circuit breaker: %s repeated %d times with no progress", toolName, noProgress.count)
det := LoopDetectionResult{
Stuck:      true,
Level:      LevelCritical,
Detector:   DetectorGlobalCircuitBreaker,
Count:      noProgress.count,
WarningKey: fmt.Sprintf("global:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

// 4. Known poll no-progress (critical)
if knownPoll && resolved.Detectors.KnownPollNoProgress && noProgress.count >= resolved.CriticalThreshold {
log.Printf("[loop-detection] Critical polling loop: %s repeated %d times", toolName, noProgress.count)
det := LoopDetectionResult{
Stuck:      true,
Level:      LevelCritical,
Detector:   DetectorKnownPollNoProgress,
Count:      noProgress.count,
WarningKey: fmt.Sprintf("poll:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
BackoffMs:  backoffMs,
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

// 5. Known poll no-progress (warning with backoff)
if knownPoll && resolved.Detectors.KnownPollNoProgress && noProgress.count >= resolved.WarningThreshold {
log.Printf("[loop-detection] Polling loop warning: %s repeated %d times", toolName, noProgress.count)
det := LoopDetectionResult{
Stuck:      true,
Level:      LevelWarning,
Detector:   DetectorKnownPollNoProgress,
Count:      noProgress.count,
WarningKey: fmt.Sprintf("poll:%s:%s:%s", toolName, argsHash, noProgress.latestResultHash),
BackoffMs:  backoffMs,
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

// 6. Ping-pong detection
pingPong := getPingPongStreak(history, argsHash)
pingPongWarningKey := fmt.Sprintf("pingpong:%s:%s", toolName, argsHash)
if pingPong.pairedSignature != "" {
pingPongWarningKey = "pingpong:" + canonicalPairKey(argsHash, pingPong.pairedSignature)
}

if resolved.Detectors.PingPong && pingPong.count >= resolved.CriticalThreshold && pingPong.noProgressEvidence {
log.Printf("[loop-detection] Critical ping-pong loop: alternating calls count=%d tool=%s", pingPong.count, toolName)
det := LoopDetectionResult{
Stuck:          true,
Level:          LevelCritical,
Detector:       DetectorPingPong,
Count:          pingPong.count,
PairedToolName: pingPong.pairedToolName,
WarningKey:     pingPongWarningKey,
BackoffMs:      backoffMs,
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

if resolved.Detectors.PingPong && pingPong.count >= resolved.WarningThreshold {
log.Printf("[loop-detection] Ping-pong loop warning: alternating calls count=%d tool=%s", pingPong.count, toolName)
det := LoopDetectionResult{
Stuck:          true,
Level:          LevelWarning,
Detector:       DetectorPingPong,
Count:          pingPong.count,
PairedToolName: pingPong.pairedToolName,
WarningKey:     pingPongWarningKey,
BackoffMs:      backoffMs,
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

// 7. Generic repeat
recentCount := 0
for _, h := range history {
if h.ToolName == toolName && h.ArgsHash == argsHash {
recentCount++
}
}

if !knownPoll && resolved.Detectors.GenericRepeat && recentCount >= resolved.WarningThreshold {
log.Printf("[loop-detection] Generic repeat warning: %s called %d times with identical arguments", toolName, recentCount)
det := LoopDetectionResult{
Stuck:      true,
Level:      LevelWarning,
Detector:   DetectorGenericRepeat,
Count:      recentCount,
WarningKey: fmt.Sprintf("generic:%s:%s", toolName, argsHash),
BackoffMs:  backoffMs,
}
det.SelfReflection = BuildSelfReflection(state, det, toolName)
det.Message = det.SelfReflection
return det
}

// 8. Token budget warning (non-critical)
if tokenBudget.Stuck && tokenBudget.Level == LevelWarning {
return tokenBudget
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
TotalCalls      int
UniquePatterns  int
MostFrequent    *ToolFrequency
TotalTokensUsed int
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
TotalCalls:      len(history),
UniquePatterns:  len(patterns),
MostFrequent:    mostFrequent,
TotalTokensUsed: state.TotalTokensUsed,
}
}

// ---- Internal helpers ----

type noProgressResult struct {
count            int
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

if currentSignature != otherSignature {
return pingPongResult{}
}

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
count:              alternatingCount + 1,
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
