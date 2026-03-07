package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileChange represents a detected file change.
type FileChange struct {
	Path    string    `json:"path"`
	Action  string    `json:"action"` // "modified", "created", "deleted"
	ModTime time.Time `json:"mod_time"`
	Size    int64     `json:"size"`
}

// Watcher polls a workspace directory for file changes.
type Watcher struct {
	root     string
	mu       sync.RWMutex
	files    map[string]fileInfo
	changes  []FileChange
	maxLog   int
	interval time.Duration
	ignores  []string
	done     chan struct{}
	running  bool
}

type fileInfo struct {
	modTime time.Time
	size    int64
}

// New creates a file watcher for the given root directory.
func New(root string, interval time.Duration) *Watcher {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Watcher{
		root:     root,
		files:    make(map[string]fileInfo),
		changes:  make([]FileChange, 0),
		maxLog:   100,
		interval: interval,
		ignores: []string{
			".git", "node_modules", "__pycache__", ".DS_Store",
			"vendor", ".idea", ".vscode", "dist", "build",
			".next", ".nuxt", "target", "bin", "obj",
		},
		done: make(chan struct{}),
	}
}

// Start begins polling for changes in a background goroutine.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	// Initial scan
	w.scan()
	log.Printf("[watcher] Initial scan: %d files in %s", len(w.files), w.root)

	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case <-ticker.C:
				w.scan()
			}
		}
	}()
}

// Stop halts the watcher.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.done)
		w.running = false
	}
}

// RecentChanges returns changes since the given time.
func (w *Watcher) RecentChanges(since time.Duration) []FileChange {
	w.mu.RLock()
	defer w.mu.RUnlock()

	cutoff := time.Now().Add(-since)
	var result []FileChange
	for _, c := range w.changes {
		if c.ModTime.After(cutoff) {
			result = append(result, c)
		}
	}
	return result
}

// Summary returns a text summary of recent changes for injection into prompts.
func (w *Watcher) Summary(since time.Duration) string {
	changes := w.RecentChanges(since)
	if len(changes) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("--- RECENT FILE CHANGES ---\n")
	for _, c := range changes {
		rel, _ := filepath.Rel(w.root, c.Path)
		if rel == "" {
			rel = c.Path
		}
		sb.WriteString(c.Action + ": " + rel + "\n")
	}
	sb.WriteString("--- END FILE CHANGES ---\n")
	return sb.String()
}

// FileCount returns the number of tracked files.
func (w *Watcher) FileCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.files)
}

func (w *Watcher) scan() {
	newFiles := make(map[string]fileInfo)

	filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()

		// Skip ignored directories
		if info.IsDir() {
			for _, ig := range w.ignores {
				if name == ig {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Skip hidden files and binary-looking files
		if strings.HasPrefix(name, ".") {
			return nil
		}
		// Skip large files (>1MB likely binary)
		if info.Size() > 1024*1024 {
			return nil
		}

		newFiles[path] = fileInfo{modTime: info.ModTime(), size: info.Size()}
		return nil
	})

	w.mu.Lock()
	defer w.mu.Unlock()

	// Detect changes
	for path, newInfo := range newFiles {
		if oldInfo, exists := w.files[path]; !exists {
			// New file
			if len(w.files) > 0 { // skip on initial scan
				w.addChange(FileChange{Path: path, Action: "created", ModTime: newInfo.modTime, Size: newInfo.size})
			}
		} else if newInfo.modTime != oldInfo.modTime || newInfo.size != oldInfo.size {
			// Modified
			w.addChange(FileChange{Path: path, Action: "modified", ModTime: newInfo.modTime, Size: newInfo.size})
		}
	}

	// Detect deletions
	if len(w.files) > 0 {
		for path := range w.files {
			if _, exists := newFiles[path]; !exists {
				w.addChange(FileChange{Path: path, Action: "deleted", ModTime: time.Now()})
			}
		}
	}

	w.files = newFiles
}

func (w *Watcher) addChange(c FileChange) {
	w.changes = append(w.changes, c)
	if len(w.changes) > w.maxLog {
		w.changes = w.changes[len(w.changes)-w.maxLog:]
	}
}
