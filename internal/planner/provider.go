package planner

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// ModelProvider represents a single LLM provider/model configuration.
type ModelProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	// AuthType: "bearer" (default), "none"
	AuthType string `json:"auth_type"`
	// Priority: lower = preferred. Used for failover ordering.
	Priority int `json:"priority"`

	// Runtime state (not persisted)
	failures     int
	lastFailure  time.Time
	cooldownUntil time.Time
	mu           sync.Mutex
}

// CooldownDuration after consecutive failures before retry.
const cooldownBase = 30 * time.Second

func (p *ModelProvider) RecordFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures++
	p.lastFailure = time.Now()
	// Exponential backoff: 30s, 60s, 120s, max 5min
	backoff := cooldownBase * time.Duration(1<<min(p.failures-1, 4))
	if backoff > 5*time.Minute {
		backoff = 5 * time.Minute
	}
	p.cooldownUntil = time.Now().Add(backoff)
	log.Printf("[provider] %s failed (%d consecutive). Cooldown until %s", p.ID, p.failures, p.cooldownUntil.Format(time.Kitchen))
}

func (p *ModelProvider) RecordSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures = 0
	p.cooldownUntil = time.Time{}
}

func (p *ModelProvider) IsAvailable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cooldownUntil.IsZero() {
		return true
	}
	return time.Now().After(p.cooldownUntil)
}

func (p *ModelProvider) FailureCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.failures
}

// MultiPlanner manages multiple model providers with failover.
type MultiPlanner struct {
	providers []*ModelProvider
	mu        sync.RWMutex
}

func NewMultiPlanner() *MultiPlanner {
	return &MultiPlanner{}
}

// AddProvider adds a model provider. It is safe for concurrent use.
func (mp *MultiPlanner) AddProvider(p *ModelProvider) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if p.AuthType == "" {
		p.AuthType = "bearer"
	}
	mp.providers = append(mp.providers, p)
}

// SetProviders replaces all providers.
func (mp *MultiPlanner) SetProviders(providers []*ModelProvider) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.providers = providers
}

// ListProviders returns a copy of the current providers.
func (mp *MultiPlanner) ListProviders() []*ModelProvider {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	out := make([]*ModelProvider, len(mp.providers))
	copy(out, mp.providers)
	return out
}

// selectProvider picks the best available provider, optionally filtering by ID.
func (mp *MultiPlanner) selectProvider(preferredID string) (*ModelProvider, error) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	if len(mp.providers) == 0 {
		return nil, fmt.Errorf("no model providers configured")
	}

	// If preferred ID is specified, try it first
	if preferredID != "" {
		for _, p := range mp.providers {
			if p.ID == preferredID && p.IsAvailable() {
				return p, nil
			}
		}
	}

	// Fall through to priority-based selection
	var best *ModelProvider
	for _, p := range mp.providers {
		if !p.IsAvailable() {
			continue
		}
		if best == nil || p.Priority < best.Priority {
			best = p
		}
	}

	if best == nil {
		// All providers in cooldown — pick the one whose cooldown expires soonest
		var soonest *ModelProvider
		for _, p := range mp.providers {
			if soonest == nil || p.cooldownUntil.Before(soonest.cooldownUntil) {
				soonest = p
			}
		}
		if soonest != nil {
			log.Printf("[provider] All providers in cooldown. Forcing %s", soonest.ID)
			return soonest, nil
		}
		return nil, fmt.Errorf("all model providers unavailable")
	}
	return best, nil
}

// Call tries the preferred provider, then fails over to others.
func (mp *MultiPlanner) Call(messages []Message, toolDefs []ToolDefinition, preferredID string) (*Message, error) {
	mp.mu.RLock()
	count := len(mp.providers)
	mp.mu.RUnlock()

	if count == 0 {
		return nil, fmt.Errorf("no model providers configured")
	}

	// Build ordered list: preferred first, then by priority
	tried := make(map[string]bool)
	var lastErr error

	for attempt := 0; attempt < count; attempt++ {
		provider, err := mp.selectProvider(preferredID)
		if err != nil {
			return nil, err
		}

		if tried[provider.ID] {
			// Already tried this one, skip
			break
		}
		tried[provider.ID] = true

		log.Printf("[provider] Attempting call with provider=%s model=%s (attempt %d)", provider.ID, provider.Model, attempt+1)

		p := &Planner{endpoint: provider.Endpoint, apiKey: provider.APIKey}
		msg, err := p.CallWithModel(messages, toolDefs, provider.Model, provider.AuthType)
		if err != nil {
			lastErr = err
			provider.RecordFailure()
			log.Printf("[provider] Provider %s failed: %v. Trying next...", provider.ID, err)
			preferredID = "" // clear preference for next attempt
			continue
		}

		provider.RecordSuccess()
		return msg, nil
	}

	return nil, fmt.Errorf("all providers failed. Last error: %v", lastErr)
}

// ProviderStatus returns a summary of all providers' health.
func (mp *MultiPlanner) ProviderStatus() []map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	var status []map[string]interface{}
	for _, p := range mp.providers {
		s := map[string]interface{}{
			"id":        p.ID,
			"name":      p.Name,
			"model":     p.Model,
			"endpoint":  maskEndpoint(p.Endpoint),
			"priority":  p.Priority,
			"available": p.IsAvailable(),
			"failures":  p.FailureCount(),
		}
		status = append(status, s)
	}
	return status
}

func maskEndpoint(e string) string {
	if len(e) > 30 {
		return e[:25] + "..."
	}
	return e
}

// ParseProvidersFromEnv parses MODELS env var.
// Format: "id:name:endpoint:key:model:priority" separated by "||"
// Example: "zai:ZhipuAI:https://api.z.ai/v1/chat/completions:sk-xxx:glm-5:1||openai:OpenAI:https://api.openai.com/v1/chat/completions:sk-yyy:gpt-4o:2"
func ParseProvidersFromEnv(modelsStr string) []*ModelProvider {
	if modelsStr == "" {
		return nil
	}
	var providers []*ModelProvider
	entries := strings.Split(modelsStr, "||")
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 6)
		if len(parts) < 5 {
			log.Printf("[provider] Skipping malformed provider entry: %q (need at least 5 parts)", entry)
			continue
		}
		priority := i + 1
		if len(parts) >= 6 {
			if p := parseInt(parts[5]); p > 0 {
				priority = p
			}
		}
		providers = append(providers, &ModelProvider{
			ID:       parts[0],
			Name:     parts[1],
			Endpoint: parts[2],
			APIKey:   parts[3],
			Model:    parts[4],
			AuthType: "bearer",
			Priority: priority,
		})
	}
	return providers
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
