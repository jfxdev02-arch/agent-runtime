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
	SessionID string
	Role      string
	Content   string
	CreatedAt string
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
	query := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS tool_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT,
		tool_name TEXT,
		input TEXT,
		output TEXT,
		status TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`
	_, err := s.db.Exec(query)
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
	rows, err := s.db.Query(
		`SELECT session_id, role, content, created_at FROM messages
		 WHERE session_id = ? ORDER BY id DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Storage) SearchOlderMessages(sessionID string, skip, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT session_id, role, content, created_at FROM messages
		 ORDER BY id DESC LIMIT ? OFFSET ?`, limit, skip)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Storage) GetRecentToolLogs(limit int) ([]ToolLog, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, tool_name, input, output, status, created_at
		 FROM tool_logs ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []ToolLog
	for rows.Next() {
		var l ToolLog
		if err := rows.Scan(&l.ID, &l.SessionID, &l.ToolName, &l.Input, &l.Output, &l.Status, &l.CreatedAt); err != nil {
			continue
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func (s *Storage) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
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
	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		settings[k] = v
	}
	return settings, nil
}

func (s *Storage) SaveAllSettings(settings map[string]string) error {
	for k, v := range settings {
		if err := s.SetSetting(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	stats["total_messages"] = count
	s.db.QueryRow("SELECT COUNT(*) FROM tool_logs").Scan(&count)
	stats["total_tool_executions"] = count
	s.db.QueryRow("SELECT COUNT(*) FROM tool_logs WHERE status = 'OK'").Scan(&count)
	stats["successful_executions"] = count
	s.db.QueryRow("SELECT COUNT(DISTINCT session_id) FROM messages").Scan(&count)
	stats["total_sessions"] = count
	return stats, nil
}

func (s *Storage) StatsJSON() (string, error) {
	stats, err := s.GetStats()
	if err != nil {
		return "{}", err
	}
	b, _ := json.Marshal(stats)
	return string(b), nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
