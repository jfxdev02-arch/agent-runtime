package storage

import (
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

type StoredMessage struct {
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type ChatSessionSummary struct {
	SessionID     string `json:"session_id"`
	LastMessageAt string `json:"last_message_at"`
	TotalMessages int    `json:"total_messages"`
	LastMessage   string `json:"last_message"`
}

type ToolLog struct {
	ID        int    `json:"id"`
	SessionID string `json:"session_id"`
	ToolName  string `json:"tool_name"`
	Input     string `json:"input"`
	Output    string `json:"output"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Status      string `json:"status"` // active, paused, done, archived
	TechStack   string `json:"tech_stack"`
	GitRemote   string `json:"git_remote"`
	Notes       string `json:"notes"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	s := &Storage{db: db}
	if err := s.InitSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) InitSchema() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT, role TEXT, content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS tool_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT, tool_name TEXT, input TEXT, output TEXT, status TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, value TEXT);
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL, path TEXT NOT NULL UNIQUE,
		description TEXT DEFAULT '', status TEXT DEFAULT 'active',
		tech_stack TEXT DEFAULT '', git_remote TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`)
	return err
}

func (s *Storage) LogMessage(sessionID, role, content string) error {
	_, err := s.db.Exec("INSERT INTO messages (session_id, role, content) VALUES (?, ?, ?)", sessionID, role, content)
	return err
}

func (s *Storage) LogToolExecution(sessionID, toolName, input, output, status string) error {
	_, err := s.db.Exec("INSERT INTO tool_logs (session_id, tool_name, input, output, status) VALUES (?, ?, ?, ?, ?)", sessionID, toolName, input, output, status)
	return err
}

func (s *Storage) GetRecentMessages(sessionID string, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(`SELECT session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.CreatedAt)
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Storage) GetSessionMessages(sessionID string, limit int) ([]StoredMessage, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		rows, err = s.db.Query(`
			SELECT session_id, role, content, created_at
			FROM (
				SELECT id, session_id, role, content, created_at
				FROM messages
				WHERE session_id = ?
				ORDER BY id DESC
				LIMIT ?
			)
			ORDER BY id ASC`, sessionID, limit)
	} else {
		rows, err = s.db.Query(`SELECT session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`, sessionID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.CreatedAt)
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (s *Storage) ListChatSessions(prefix string, limit int) ([]ChatSessionSummary, error) {
	if limit <= 0 {
		limit = 30
	}
	like := "%"
	if prefix != "" {
		like = prefix + "%"
	}

	rows, err := s.db.Query(`
		SELECT
			m.session_id,
			MAX(m.created_at) AS last_message_at,
			COUNT(*) AS total_messages,
			(
				SELECT m2.content
				FROM messages m2
				WHERE m2.session_id = m.session_id
				ORDER BY m2.id DESC
				LIMIT 1
			) AS last_message
		FROM messages m
		WHERE m.session_id LIKE ?
		GROUP BY m.session_id
		ORDER BY MAX(m.id) DESC
		LIMIT ?`, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ChatSessionSummary
	for rows.Next() {
		var item ChatSessionSummary
		rows.Scan(&item.SessionID, &item.LastMessageAt, &item.TotalMessages, &item.LastMessage)
		sessions = append(sessions, item)
	}
	return sessions, nil
}

func (s *Storage) DeleteChatSession(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM messages WHERE session_id = ?", sessionID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM tool_logs WHERE session_id = ?", sessionID)
	return err
}

func (s *Storage) SearchOlderMessages(sessionID string, skip, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(`SELECT session_id, role, content, created_at FROM messages ORDER BY id DESC LIMIT ? OFFSET ?`, limit, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.CreatedAt)
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Storage) GetRecentToolLogs(limit int) ([]ToolLog, error) {
	rows, err := s.db.Query(`SELECT id, session_id, tool_name, input, output, status, created_at FROM tool_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []ToolLog
	for rows.Next() {
		var l ToolLog
		rows.Scan(&l.ID, &l.SessionID, &l.ToolName, &l.Input, &l.Output, &l.Status, &l.CreatedAt)
		logs = append(logs, l)
	}
	return logs, nil
}

// Settings
func (s *Storage) GetSetting(key string) (string, error) {
	var v string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v)
	return v, err
}

func (s *Storage) SetSetting(key, value string) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

func (s *Storage) GetAllSettings() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		m[k] = v
	}
	return m, nil
}

func (s *Storage) SaveAllSettings(settings map[string]string) error {
	for k, v := range settings {
		s.SetSetting(k, v)
	}
	return nil
}

// Projects
func (s *Storage) GetAllProjects() ([]Project, error) {
	rows, err := s.db.Query(`SELECT id, name, path, description, status, tech_stack, git_remote, notes, created_at, updated_at FROM projects ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []Project
	for rows.Next() {
		var p Project
		rows.Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.Status, &p.TechStack, &p.GitRemote, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Storage) GetProject(id int) (*Project, error) {
	var p Project
	err := s.db.QueryRow(`SELECT id, name, path, description, status, tech_stack, git_remote, notes, created_at, updated_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.Description, &p.Status, &p.TechStack, &p.GitRemote, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Storage) CreateProject(name, path, description, techStack, gitRemote string) (int64, error) {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO projects (name, path, description, tech_stack, git_remote) VALUES (?, ?, ?, ?, ?)`,
		name, path, description, techStack, gitRemote)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Storage) UpdateProject(id int, name, description, status, notes string) error {
	_, err := s.db.Exec(`UPDATE projects SET name=?, description=?, status=?, notes=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`,
		name, description, status, notes, id)
	return err
}

func (s *Storage) DeleteProject(id int) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

func (s *Storage) ProjectExistsByPath(path string) bool {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM projects WHERE path = ?", path).Scan(&count)
	return count > 0
}

// Stats
func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var c int
	s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&c)
	stats["total_messages"] = c
	s.db.QueryRow("SELECT COUNT(*) FROM tool_logs").Scan(&c)
	stats["total_tool_executions"] = c
	s.db.QueryRow("SELECT COUNT(*) FROM tool_logs WHERE status = 'OK'").Scan(&c)
	stats["successful_executions"] = c
	s.db.QueryRow("SELECT COUNT(DISTINCT session_id) FROM messages").Scan(&c)
	stats["total_sessions"] = c
	s.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&c)
	stats["total_projects"] = c
	return stats, nil
}

func (s *Storage) StatsJSON() (string, error) {
	st, _ := s.GetStats()
	b, _ := json.Marshal(st)
	return string(b), nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
