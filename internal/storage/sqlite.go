package storage

import (
	"database/sql"

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
	`
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

// GetRecentMessages returns the last N messages for a given session
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
	// Reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// SearchOlderMessages returns older messages (beyond the recent window) across ALL sessions
// that might be relevant. Returns up to `limit` messages, skipping the most recent `skip` ones.
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
	// Reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}
