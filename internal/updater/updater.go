package updater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Version is set at build time via -ldflags "-X ...updater.Version=vX.Y.Z"
var Version = "v0.0.0-dev"

const (
	repoOwner = "jfxdev02-arch"
	repoName  = "agent-runtime"
	githubAPI = "https://api.github.com"
)

// UpdateInfo contains the result of a version check.
type UpdateInfo struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseNotes    string `json:"release_notes,omitempty"`
	ReleaseURL      string `json:"release_url,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
}

// UpdateResult contains the result of applying an update.
type UpdateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	OldVer  string `json:"old_version"`
	NewVer  string `json:"new_version"`
	Output  string `json:"output,omitempty"`
}

// CheckForUpdates queries the GitHub API for the latest release tag
// and compares it with the current build version.
func CheckForUpdates() (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", githubAPI, repoOwner, repoName)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub API: %v", err)
	}
	defer resp.Body.Close()

	// If no releases exist, try tags instead
	if resp.StatusCode == 404 {
		return checkFromTags()
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName     string `json:"tag_name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %v", err)
	}

	return &UpdateInfo{
		CurrentVersion:  Version,
		LatestVersion:   release.TagName,
		UpdateAvailable: isNewerVersion(release.TagName, Version),
		ReleaseNotes:    release.Body,
		ReleaseURL:      release.HTMLURL,
		PublishedAt:     release.PublishedAt,
	}, nil
}

// checkFromTags falls back to checking the latest git tag via the API.
// Fetches multiple tags and picks the highest semver version.
func checkFromTags() (*UpdateInfo, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tags?per_page=30", githubAPI, repoOwner, repoName)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub API: %v", err)
	}
	defer resp.Body.Close()

	var tags []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to parse tags response: %v", err)
	}

	if len(tags) == 0 {
		return &UpdateInfo{
			CurrentVersion:  Version,
			LatestVersion:   Version,
			UpdateAvailable: false,
		}, nil
	}

	// Find the highest version tag
	best := tags[0].Name
	for _, t := range tags[1:] {
		if isNewerVersion(t.Name, best) {
			best = t.Name
		}
	}

	return &UpdateInfo{
		CurrentVersion:  Version,
		LatestVersion:   best,
		UpdateAvailable: isNewerVersion(best, Version),
	}, nil
}

// ApplyUpdate performs a git pull, rebuilds the binary, and restarts the
// systemd service. This only works on the deployment machine (e.g. Raspberry Pi).
func ApplyUpdate(projectDir string) *UpdateResult {
	oldVer := Version
	var allOutput strings.Builder

	// Step 1: git fetch + pull
	log.Println("[updater] Step 1/3: Fetching latest changes...")
	out, err := runCmd(projectDir, "git", "fetch", "--all", "--tags")
	allOutput.WriteString("=== git fetch ===\n" + out + "\n")
	if err != nil {
		return &UpdateResult{Success: false, Message: "git fetch failed: " + err.Error(), OldVer: oldVer, Output: allOutput.String()}
	}

	out, err = runCmd(projectDir, "git", "pull", "--rebase")
	allOutput.WriteString("=== git pull ===\n" + out + "\n")
	if err != nil {
		// Try without rebase
		out, err = runCmd(projectDir, "git", "pull")
		allOutput.WriteString("=== git pull (no rebase) ===\n" + out + "\n")
		if err != nil {
			return &UpdateResult{Success: false, Message: "git pull failed: " + err.Error(), OldVer: oldVer, Output: allOutput.String()}
		}
	}

	// Step 2: Get the new version from the latest git tag
	newVer, _ := runCmd(projectDir, "git", "describe", "--tags", "--abbrev=0")
	newVer = strings.TrimSpace(newVer)
	if newVer == "" {
		newVer = "unknown"
	}

	// Step 3: Tidy modules and rebuild binary with the new version embedded
	log.Println("[updater] Step 2/3: Rebuilding binary...")
	_, _ = runCmd(projectDir, "go", "mod", "tidy")
	ldflags := fmt.Sprintf("-w -s -X github.com/dev/agent-runtime/internal/updater.Version=%s", newVer)
	out, err = runCmd(projectDir, "go", "build", "-ldflags", ldflags, "-o", "agent-runtime", "cmd/agent/main.go")
	allOutput.WriteString("=== go build ===\n" + out + "\n")
	if err != nil {
		return &UpdateResult{Success: false, Message: "build failed: " + err.Error(), OldVer: oldVer, NewVer: newVer, Output: allOutput.String()}
	}

	// Step 4: Restart the systemd service
	log.Println("[updater] Step 3/3: Restarting service...")
	out, err = runCmd(projectDir, "sudo", "systemctl", "restart", "agent-runtime")
	allOutput.WriteString("=== systemctl restart ===\n" + out + "\n")
	if err != nil {
		// Even if restart "fails" (because the process is being replaced), that's expected
		allOutput.WriteString("(restart signal sent — this process may terminate)\n")
	}

	return &UpdateResult{
		Success: true,
		Message: fmt.Sprintf("Updated from %s to %s. Service is restarting...", oldVer, newVer),
		OldVer:  oldVer,
		NewVer:  newVer,
		Output:  allOutput.String(),
	}
}

// GetProjectDir tries to find the project directory by looking at the
// directory of the running binary, or the current working directory.
func GetProjectDir() string {
	// Try to get it from the git root
	out, err := runCmd(".", "git", "rev-parse", "--show-toplevel")
	if err == nil {
		return strings.TrimSpace(out)
	}
	return "."
}

func runCmd(dir string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := stdout.String() + stderr.String()
	return combined, err
}

// normalizeVersion strips the leading 'v' and trims whitespace for comparison.
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

// parseSemver extracts major, minor, patch from a version string like "v1.0.6" or "1.0.6".
// Returns (0,0,0, false) if parsing fails.
func parseSemver(v string) (int, int, int, bool) {
	v = normalizeVersion(v)
	// Strip anything after a hyphen (e.g. "1.0.6-dirty" -> "1.0.6")
	if idx := strings.Index(v, "-"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	patch, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

// isNewerVersion returns true if candidate is a higher semver than current.
func isNewerVersion(candidate, current string) bool {
	cMaj, cMin, cPat, cOk := parseSemver(candidate)
	rMaj, rMin, rPat, rOk := parseSemver(current)
	if !cOk || !rOk {
		// Fallback to string comparison if parsing fails
		return normalizeVersion(candidate) > normalizeVersion(current)
	}
	if cMaj != rMaj {
		return cMaj > rMaj
	}
	if cMin != rMin {
		return cMin > rMin
	}
	return cPat > rPat
}
