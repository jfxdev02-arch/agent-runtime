package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Context provides git-aware context for prompt injection.
type Context struct {
	root string
}

// New creates a git context provider for the given root directory.
func New(root string) *Context {
	return &Context{root: root}
}

// IsRepo returns true if the root is a git repository.
func (g *Context) IsRepo() bool {
	_, err := g.run("rev-parse", "--is-inside-work-tree")
	return err == nil
}

// Branch returns the current branch name.
func (g *Context) Branch() string {
	out, err := g.run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Status returns short git status.
func (g *Context) Status() string {
	out, _ := g.run("status", "--short")
	return strings.TrimSpace(out)
}

// DiffStaged returns the staged diff.
func (g *Context) DiffStaged() string {
	out, _ := g.run("diff", "--cached", "--stat")
	return strings.TrimSpace(out)
}

// DiffUnstaged returns the unstaged diff summary.
func (g *Context) DiffUnstaged() string {
	out, _ := g.run("diff", "--stat")
	return strings.TrimSpace(out)
}

// RecentCommits returns the last N commit onelines.
func (g *Context) RecentCommits(n int) string {
	out, _ := g.run("log", "--oneline", fmt.Sprintf("-%d", n))
	return strings.TrimSpace(out)
}

// Blame returns blame info for a file (abbreviated).
func (g *Context) Blame(file string, startLine, endLine int) string {
	args := []string{"blame", "-L", fmt.Sprintf("%d,%d", startLine, endLine), "--porcelain", file}
	out, _ := g.run(args...)
	// Parse porcelain blame into readable format
	return parsePorcelainBlame(out)
}

// RemoteURL returns the origin remote URL.
func (g *Context) RemoteURL() string {
	out, _ := g.run("remote", "get-url", "origin")
	return strings.TrimSpace(out)
}

// Summary returns a full git context summary for prompt injection.
func (g *Context) Summary() string {
	if !g.IsRepo() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("--- GIT CONTEXT ---\n")

	branch := g.Branch()
	if branch != "" {
		sb.WriteString("Branch: " + branch + "\n")
	}

	status := g.Status()
	if status != "" {
		sb.WriteString("\nModified files:\n" + status + "\n")
	}

	unstaged := g.DiffUnstaged()
	if unstaged != "" {
		sb.WriteString("\nUnstaged changes:\n" + unstaged + "\n")
	}

	staged := g.DiffStaged()
	if staged != "" {
		sb.WriteString("\nStaged changes:\n" + staged + "\n")
	}

	commits := g.RecentCommits(5)
	if commits != "" {
		sb.WriteString("\nRecent commits:\n" + commits + "\n")
	}

	sb.WriteString("--- END GIT CONTEXT ---\n")
	return sb.String()
}

func (g *Context) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.root
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("%s: %s", err, stderr.String())
		}
		return out.String(), nil
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("git command timed out")
	}
}

func parsePorcelainBlame(raw string) string {
	lines := strings.Split(raw, "\n")
	var parts []string
	var currentAuthor, currentLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "author ") {
			currentAuthor = strings.TrimPrefix(line, "author ")
		}
		if strings.HasPrefix(line, "\t") {
			currentLine = strings.TrimPrefix(line, "\t")
			if currentAuthor != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", currentAuthor, currentLine))
				currentAuthor = ""
			}
		}
	}
	if len(parts) > 20 {
		parts = parts[:20]
		parts = append(parts, "... (truncated)")
	}
	return strings.Join(parts, "\n")
}
