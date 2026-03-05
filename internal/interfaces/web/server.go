package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	rt "github.com/dev/agent-runtime/internal/runtime"
	"github.com/dev/agent-runtime/internal/storage"
	"github.com/dev/agent-runtime/internal/updater"
)

type Server struct {
	rt         *rt.Runtime
	store      *storage.Storage
	cfg        map[string]string
	port       string
	start      time.Time
	projectDir string
}

func NewServer(runtime *rt.Runtime, store *storage.Storage, port string) *Server {
	return &Server{rt: runtime, store: store, port: port, start: time.Now(), projectDir: updater.GetProjectDir()}
}

func (s *Server) SetConfig(agentName, language string) {
	s.cfg = map[string]string{"agent_name": agentName, "language": language, "version": updater.Version}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/chat", s.handleChat)
	http.HandleFunc("/api/chat/new", s.handleNewChat)
	http.HandleFunc("/api/chat/history", s.handleChatHistory)
	http.HandleFunc("/api/chats", s.handleChats)
	http.HandleFunc("/api/logs", s.handleLogs)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/settings", s.handleSettings)
	http.HandleFunc("/api/projects", s.handleProjects)
	http.HandleFunc("/api/projects/scan", s.handleProjectScan)
	http.HandleFunc("/api/projects/git", s.handleProjectGit)
	http.HandleFunc("/api/projects/git/action", s.handleProjectGitAction)
	http.HandleFunc("/api/app-config", s.handleAppConfig)
	http.HandleFunc("/api/update/check", s.handleUpdateCheck)
	http.HandleFunc("/api/update/apply", s.handleUpdateApply)
	fmt.Printf("Web server listening on http://0.0.0.0:%s\n", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

func (s *Server) handleAppConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.cfg == nil {
		s.cfg = map[string]string{"agent_name": "Agent", "language": "en", "version": updater.Version}
	}
	json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.SessionID == "" {
		req.SessionID = "web-default"
	}
	reply, _ := s.rt.ProcessMessage(req.SessionID, req.Message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"reply": reply, "session_id": req.SessionID})
}

func (s *Server) handleNewChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Prefix string `json:"prefix"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	prefix := strings.TrimSpace(req.Prefix)
	if prefix == "" {
		prefix = "web"
	}
	sessionID := s.rt.NewSessionID(prefix)
	// Ensure session exists in memory even before first user message.
	s.rt.GetSession(sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID})
}

func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id is required", 400)
		return
	}
	limit := 0
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		limit, _ = strconv.Atoi(q)
	}
	history, err := s.rt.GetChatHistory(sessionID, limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (s *Server) handleChats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "method not allowed", 405)
		return
	}
	limit := 30
	if q := strings.TrimSpace(r.URL.Query().Get("limit")); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			limit = n
		}
	}
	prefix := strings.TrimSpace(r.URL.Query().Get("prefix"))
	sessions, err := s.rt.ListChatSessions(prefix, limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	logs, _ := s.store.GetRecentToolLogs(50)
	if logs == nil {
		logs = []storage.ToolLog{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats, _ := s.store.GetStats()
	hostname, _ := os.Hostname()
	status := map[string]interface{}{
		"hostname": hostname, "uptime_seconds": int(time.Since(s.start).Seconds()),
		"go_version": runtime.Version(), "os_arch": runtime.GOOS + "/" + runtime.GOARCH,
		"goroutines": runtime.NumGoroutine(), "mem_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"mem_sys_mb": float64(m.Sys) / 1024 / 1024, "db_stats": stats,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		settings, _ := s.store.GetAllSettings()
		if settings == nil {
			settings = make(map[string]string)
		}
		defaults := map[string]string{
			"telegram_token":    maskSecret(os.Getenv("TELEGRAM_TOKEN")),
			"telegram_allow_id": os.Getenv("TELEGRAM_ALLOW_ID"),
			"zai_endpoint":      os.Getenv("ZAI_ENDPOINT"),
			"zai_api_key":       maskSecret(os.Getenv("ZAI_API_KEY")),
			"workspace_root":    os.Getenv("WORKSPACE_ROOT"),
			"model":             "glm-5", "max_history": "25", "max_turns": "50",
			"github_token": "", "github_username": "",
			"agent_name": os.Getenv("AGENT_NAME"), "language": os.Getenv("LANGUAGE"),
		}
		if defaults["agent_name"] == "" {
			defaults["agent_name"] = "Cronos"
		}
		if defaults["language"] == "" {
			defaults["language"] = "en"
		}
		for k, v := range defaults {
			if _, exists := settings[k]; !exists {
				settings[k] = v
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	case "POST":
		var settings map[string]string
		json.NewDecoder(r.Body).Decode(&settings)
		s.store.SaveAllSettings(settings)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
	}
}

// --- Projects ---

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		projects, _ := s.store.GetAllProjects()
		if projects == nil {
			projects = []storage.Project{}
		}
		// Enrich with live git info
		for i := range projects {
			projects[i].GitRemote = getGitInfo(projects[i].Path, "branch")
		}
		json.NewEncoder(w).Encode(projects)

	case "POST":
		var p struct {
			Name string `json:"name"`
			Path string `json:"path"`
			Desc string `json:"description"`
			Tech string `json:"tech_stack"`
		}
		json.NewDecoder(r.Body).Decode(&p)
		id, err := s.store.CreateProject(p.Name, p.Path, p.Desc, p.Tech, "")
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		json.NewEncoder(w).Encode(map[string]int64{"id": id})

	case "PUT":
		var p struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Desc   string `json:"description"`
			Status string `json:"status"`
			Notes  string `json:"notes"`
		}
		json.NewDecoder(r.Body).Decode(&p)
		s.store.UpdateProject(p.ID, p.Name, p.Desc, p.Status, p.Notes)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	case "DELETE":
		var p struct {
			ID int `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&p)
		s.store.DeleteProject(p.ID)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

func (s *Server) handleProjectScan(w http.ResponseWriter, r *http.Request) {
	workspace := os.Getenv("WORKSPACE_ROOT")
	if workspace == "" {
		workspace = "."
	}
	found := scanForProjects(workspace)
	for _, p := range found {
		if !s.store.ProjectExistsByPath(p.Path) {
			s.store.CreateProject(p.Name, p.Path, p.Description, p.TechStack, "")
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"scanned": len(found)})
}

func (s *Server) handleProjectGit(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)
	proj, err := s.store.GetProject(id)
	if err != nil {
		http.Error(w, "project not found", 404)
		return
	}
	info := map[string]interface{}{
		"branch":    getGitInfo(proj.Path, "branch"),
		"status":    getGitInfo(proj.Path, "status"),
		"log":       getGitInfo(proj.Path, "log"),
		"branches":  getGitInfo(proj.Path, "branches"),
		"remote":    getGitInfo(proj.Path, "remote"),
		"diff_stat": getGitInfo(proj.Path, "diff_stat"),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleProjectGitAction(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      int    `json:"id"`
		Action  string `json:"action"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	proj, err := s.store.GetProject(req.ID)
	if err != nil {
		http.Error(w, "project not found", 404)
		return
	}

	var output string
	switch req.Action {
	case "commit":
		msg := req.Message
		if msg == "" {
			msg = "Auto-commit from agent"
		}
		runGit(proj.Path, "add", "-A")
		output = runGit(proj.Path, "commit", "-m", msg)
	case "push":
		output = runGit(proj.Path, "push")
	case "pull":
		output = runGit(proj.Path, "pull")
	case "checkout":
		output = runGit(proj.Path, "checkout", req.Branch)
	case "new_branch":
		output = runGit(proj.Path, "checkout", "-b", req.Branch)
	case "init":
		output = runGit(proj.Path, "init")
	default:
		output = "unknown action"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"output": output})
}

// --- Update ---

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	info, err := updater.CheckForUpdates()
	if err != nil {
		w.WriteHeader(500)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	result := updater.ApplyUpdate(s.projectDir)
	json.NewEncoder(w).Encode(result)
}

// --- Helpers ---

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "••••••••"
	}
	return s[:4] + "••••" + s[len(s)-4:]
}

func runGit(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.String() + "\n" + out.String()
	}
	return out.String()
}

func getGitInfo(dir, what string) string {
	switch what {
	case "branch":
		return strings.TrimSpace(runGit(dir, "rev-parse", "--abbrev-ref", "HEAD"))
	case "status":
		return runGit(dir, "status", "--short")
	case "log":
		return runGit(dir, "log", "--oneline", "-10")
	case "branches":
		return runGit(dir, "branch", "-a")
	case "remote":
		return strings.TrimSpace(runGit(dir, "remote", "get-url", "origin"))
	case "diff_stat":
		return runGit(dir, "diff", "--stat")
	}
	return ""
}

type scannedProject struct {
	Name        string
	Path        string
	Description string
	TechStack   string
}

func scanForProjects(root string) []scannedProject {
	var projects []scannedProject
	entries, err := os.ReadDir(root)
	if err != nil {
		return projects
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(root, e.Name())
		tech := detectTechStack(dir)
		if tech != "" {
			projects = append(projects, scannedProject{
				Name:      e.Name(),
				Path:      dir,
				TechStack: tech,
			})
		}
	}
	return projects
}

func detectTechStack(dir string) string {
	var techs []string
	checks := map[string]string{
		"go.mod": "Go", "package.json": "Node.js", "Cargo.toml": "Rust",
		"pom.xml": "Java", "build.gradle": "Java/Kotlin", "requirements.txt": "Python",
		"pyproject.toml": "Python", "Gemfile": "Ruby", "composer.json": "PHP",
		"*.csproj": "C#/.NET", "Makefile": "Make", "CMakeLists.txt": "C/C++",
		"pubspec.yaml": "Flutter/Dart",
	}
	for file, tech := range checks {
		if strings.Contains(file, "*") {
			matches, _ := filepath.Glob(filepath.Join(dir, file))
			if len(matches) > 0 {
				techs = append(techs, tech)
			}
		} else if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			techs = append(techs, tech)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		techs = append(techs, "Git")
	}
	return strings.Join(techs, ", ")
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(getIndexHTML()))
}
