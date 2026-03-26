package friends

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// FriendStatus represents the status of a friend relationship.
type FriendStatus string

const (
	StatusAny      FriendStatus = ""
	StatusPending  FriendStatus = "pending"
	StatusAccepted FriendStatus = "accepted"
	StatusRejected FriendStatus = "rejected"
	StatusRevoked  FriendStatus = "revoked"
)

// Friend represents a friend agent relationship.
type Friend struct {
	AgentID   string       `json:"agentId"`
	AgentName string       `json:"agentName"`
	PublicKey []byte       `json:"publicKey"`
	Status    FriendStatus `json:"status"`
	AddedAt   time.Time    `json:"addedAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Message   string       `json:"message,omitempty"`
}

// Store manages persistent friend relationships using SQLite.
type Store struct {
	mu   sync.RWMutex
	db   *sql.DB
	path string
}

// NewStore creates or opens a friend store at the given SQLite path.
func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open friends db: %w", err)
	}

	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &Store{db: db, path: path}, nil
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS friends (
			agent_id    TEXT PRIMARY KEY,
			agent_name  TEXT NOT NULL DEFAULT '',
			public_key  BLOB,
			status      TEXT NOT NULL DEFAULT 'pending',
			added_at    INTEGER NOT NULL,
			updated_at  INTEGER NOT NULL,
			message     TEXT DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_friends_status ON friends(status);
	`)
	return err
}

// Close closes the underlying database connection. Safe to call multiple times.
func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
}

func (s *Store) now() time.Time {
	return time.Now().UTC()
}

// AddFriend inserts or replaces a friend record.
func (s *Store) AddFriend(f *Friend) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if f.AddedAt.IsZero() {
		f.AddedAt = now
	}
	f.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO friends (agent_id, agent_name, public_key, status, added_at, updated_at, message)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.AgentID, f.AgentName, f.PublicKey, string(f.Status),
		f.AddedAt.Unix(), f.UpdatedAt.Unix(), f.Message,
	)
	return err
}

// GetFriend retrieves a friend by agent ID. Returns nil if not found.
func (s *Store) GetFriend(agentID string) *Friend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(
		"SELECT agent_id, agent_name, public_key, status, added_at, updated_at, message FROM friends WHERE agent_id = ?",
		agentID,
	)

	var f Friend
	var status, name, message string
	var addedAt, updatedAt int64
	var pubKey []byte

	err := row.Scan(&f.AgentID, &name, &pubKey, &status, &addedAt, &updatedAt, &message)
	if err != nil {
		return nil
	}

	f.AgentName = name
	f.PublicKey = pubKey
	f.Status = FriendStatus(status)
	f.AddedAt = time.Unix(addedAt, 0).UTC()
	f.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	f.Message = message
	return &f
}

// RemoveFriend deletes a friend by agent ID. Returns os.ErrNotExist if not found.
func (s *Store) RemoveFriend(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec("DELETE FROM friends WHERE agent_id = ?", agentID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return os.ErrNotExist
	}
	return nil
}

// ListFriends returns friends filtered by status. Use StatusAny to return all.
func (s *Store) ListFriends(status FriendStatus) []*Friend {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sql.Rows
	var err error
	if status == StatusAny {
		rows, err = s.db.Query(
			"SELECT agent_id, agent_name, public_key, status, added_at, updated_at, message FROM friends ORDER BY added_at")
	} else {
		rows, err = s.db.Query(
			"SELECT agent_id, agent_name, public_key, status, added_at, updated_at, message FROM friends WHERE status = ? ORDER BY added_at",
			string(status))
	}
	if err != nil {
		return nil
	}
	defer rows.Close()

	var friends []*Friend
	for rows.Next() {
		var f Friend
		var statusStr, name, message string
		var addedAt, updatedAt int64
		var pubKey []byte
		if err := rows.Scan(&f.AgentID, &name, &pubKey, &statusStr, &addedAt, &updatedAt, &message); err != nil {
			return nil
		}
		f.AgentName = name
		f.PublicKey = pubKey
		f.Status = FriendStatus(statusStr)
		f.AddedAt = time.Unix(addedAt, 0).UTC()
		f.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		f.Message = message
		friends = append(friends, &f)
	}
	return friends
}

// UpdateStatus changes the status of a friend. Returns os.ErrNotExist if not found.
func (s *Store) UpdateStatus(agentID string, status FriendStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(
		"UPDATE friends SET status = ?, updated_at = ? WHERE agent_id = ?",
		string(status), s.now().Unix(), agentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return os.ErrNotExist
	}
	return nil
}

// IsFriend returns true if the agent exists and has StatusAccepted.
func (s *Store) IsFriend(agentID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var status string
	err := s.db.QueryRow(
		"SELECT status FROM friends WHERE agent_id = ?", agentID,
	).Scan(&status)
	if err != nil {
		return false
	}
	return FriendStatus(status) == StatusAccepted
}
