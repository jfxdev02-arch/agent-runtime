package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

// Entry is a single cache entry.
type Entry struct {
	Value     string
	Hash      string
	CreatedAt time.Time
	HitCount  int64
	TTL       time.Duration
}

// IsExpired returns true if the entry has exceeded its TTL.
func (e *Entry) IsExpired() bool {
	if e.TTL <= 0 {
		return false // no expiration
	}
	return time.Since(e.CreatedAt) > e.TTL
}

// Cache provides multi-level caching for prompts and context.
type Cache struct {
	// Level 1: System prompt cache (long TTL, rarely changes)
	systemPrompts map[string]*Entry
	// Level 2: Context cache (medium TTL, changes with conversation)
	contextCache map[string]*Entry
	// Level 3: Tool definition cache (long TTL, changes on tool registration)
	toolDefCache map[string]*Entry

	mu    sync.RWMutex
	stats Stats
}

// Stats tracks cache performance.
type Stats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Evicts int64 `json:"evicts"`
	Size   int   `json:"size"`
}

// DefaultSystemTTL is the TTL for system prompt cache entries.
const DefaultSystemTTL = 30 * time.Minute

// DefaultContextTTL is the TTL for context cache entries.
const DefaultContextTTL = 5 * time.Minute

// DefaultToolDefTTL is the TTL for tool definition cache entries.
const DefaultToolDefTTL = 1 * time.Hour

// New creates a new multi-level cache.
func New() *Cache {
	c := &Cache{
		systemPrompts: make(map[string]*Entry),
		contextCache:  make(map[string]*Entry),
		toolDefCache:  make(map[string]*Entry),
	}
	// Start background cleanup
	go c.cleanupLoop()
	return c
}

// HashKey creates a deterministic hash key from content.
func HashKey(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// GetSystemPrompt retrieves a cached system prompt.
func (c *Cache) GetSystemPrompt(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.systemPrompts[key]; ok && !e.IsExpired() {
		e.HitCount++
		c.stats.Hits++
		return e.Value, true
	}
	c.stats.Misses++
	return "", false
}

// SetSystemPrompt stores a system prompt in the cache.
func (c *Cache) SetSystemPrompt(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.systemPrompts[key] = &Entry{
		Value:     value,
		Hash:      HashKey(value),
		CreatedAt: time.Now(),
		TTL:       DefaultSystemTTL,
	}
}

// GetContext retrieves a cached context entry.
func (c *Cache) GetContext(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.contextCache[key]; ok && !e.IsExpired() {
		e.HitCount++
		c.stats.Hits++
		return e.Value, true
	}
	c.stats.Misses++
	return "", false
}

// SetContext stores a context entry.
func (c *Cache) SetContext(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contextCache[key] = &Entry{
		Value:     value,
		Hash:      HashKey(value),
		CreatedAt: time.Now(),
		TTL:       DefaultContextTTL,
	}
}

// GetToolDefs retrieves cached tool definitions.
func (c *Cache) GetToolDefs(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.toolDefCache[key]; ok && !e.IsExpired() {
		e.HitCount++
		c.stats.Hits++
		return e.Value, true
	}
	c.stats.Misses++
	return "", false
}

// SetToolDefs stores tool definitions in the cache.
func (c *Cache) SetToolDefs(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toolDefCache[key] = &Entry{
		Value:     value,
		Hash:      HashKey(value),
		CreatedAt: time.Now(),
		TTL:       DefaultToolDefTTL,
	}
}

// InvalidateAll clears all cache levels.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.systemPrompts = make(map[string]*Entry)
	c.contextCache = make(map[string]*Entry)
	c.toolDefCache = make(map[string]*Entry)
	log.Printf("[cache] All caches invalidated")
}

// InvalidateContext clears only the context cache.
func (c *Cache) InvalidateContext() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.contextCache = make(map[string]*Entry)
}

// InvalidateToolDefs clears the tool definition cache.
func (c *Cache) InvalidateToolDefs() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toolDefCache = make(map[string]*Entry)
}

// GetStats returns cache performance statistics.
func (c *Cache) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s := c.stats
	s.Size = len(c.systemPrompts) + len(c.contextCache) + len(c.toolDefCache)
	return s
}

func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	evicted := 0
	for k, e := range c.systemPrompts {
		if e.IsExpired() {
			delete(c.systemPrompts, k)
			evicted++
		}
	}
	for k, e := range c.contextCache {
		if e.IsExpired() {
			delete(c.contextCache, k)
			evicted++
		}
	}
	for k, e := range c.toolDefCache {
		if e.IsExpired() {
			delete(c.toolDefCache, k)
			evicted++
		}
	}
	if evicted > 0 {
		c.stats.Evicts += int64(evicted)
	}
}
