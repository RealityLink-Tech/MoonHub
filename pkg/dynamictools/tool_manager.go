package dynamictools

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// ToolManager provides SQLite-backed CRUD for DynamicTool with content_hash dedup.
type ToolManager struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewToolManager opens (or creates) the SQLite database at dbPath, applies WAL/NORMAL
// pragmas and initializes the schema.
func NewToolManager(dbPath string) (*ToolManager, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", p, err)
		}
	}

	tm := &ToolManager{db: db}
	if err := tm.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return tm, nil
}

// Close releases the database connection.
func (tm *ToolManager) Close() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.db.Close()
}

// initSchema creates the dynamic_tools table and associated indexes.
func (tm *ToolManager) initSchema() error {
	_, err := tm.db.Exec(`
		CREATE TABLE IF NOT EXISTS dynamic_tools (
			id              TEXT PRIMARY KEY,
			name            TEXT NOT NULL,
			description     TEXT NOT NULL DEFAULT '',
			category        TEXT NOT NULL DEFAULT '',
			chat_schema     TEXT NOT NULL DEFAULT '{}',
			space_schema    TEXT NOT NULL DEFAULT '{}',
			engine          TEXT NOT NULL DEFAULT 'schema',
			fetch_config    TEXT,
			content_hash    TEXT NOT NULL DEFAULT '',
			created_at      INTEGER NOT NULL,
			updated_at      INTEGER NOT NULL,
			is_ai_generated INTEGER NOT NULL DEFAULT 0,
			is_on_home      INTEGER NOT NULL DEFAULT 0,
			version         INTEGER NOT NULL DEFAULT 1
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_dynamic_tools_content_hash
			ON dynamic_tools(content_hash);
	`)
	return err
}

// ---------------------------------------------------------------------------
// Content hash - dedup key
// ---------------------------------------------------------------------------

// chineseStrippers lists Chinese particles, tones and punctuation runes to
// strip before hashing so that semantically identical prompts produce the
// same hash regardless of formatting.
var chineseStrippers = []struct {
	r rune
}{
	{0x7684}, // 的
	{0x4e86}, // 了
	{0x5417}, // 吗
	{0x5462}, // 呢
	{0x5427}, // 吧
	{0x554a}, // 啊
	{0x5440}, // 呀
	{0x54e6}, // 哦
	{0x54c8}, // 哈
	{0x55ef}, // 嗯
	{0x561b}, // 嘛
	{0x54ce}, // 哎
	{0xff5e}, // ～
	{0xff01}, // ！
	{0xff1f}, // ？
	{0xff0c}, // ，
	{0x3002}, // 。
	{0xff1b}, // ；
	{0x201c}, // "
	{0x201d}, // "
	{0x3001}, // 、
}

// ComputeContentHash returns a SHA-256 hex digest of the normalised prompt
// (Chinese particles/tones/punctuation stripped, lowercased, trimmed).
func ComputeContentHash(category, prompt string) string {
	// Strip Chinese particles and punctuation.
	var b strings.Builder
	b.Grow(len(prompt))
	for _, r := range prompt {
		strip := false
		for _, s := range chineseStrippers {
			if r == s.r {
				strip = true
				break
			}
		}
		if !strip {
			b.WriteRune(r)
		}
	}

	normalised := b.String()
	normalised = strings.ToLower(strings.TrimSpace(normalised))

	// Collapse multiple whitespace into a single space.
	spaceRe := regexp.MustCompile(`\s+`)
	normalised = spaceRe.ReplaceAllString(normalised, " ")

	combined := category + ":" + normalised
	h := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(h[:])
}

// ---------------------------------------------------------------------------
// CRUD operations
// ---------------------------------------------------------------------------

// List returns all tools. If source is "ai", only AI-generated tools are returned.
func (tm *ToolManager) List(ctx context.Context, source string) ([]*DynamicTool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	query := `SELECT id, name, description, category, chat_schema, space_schema,
	                  engine, fetch_config, content_hash, created_at, updated_at,
	                  is_ai_generated, is_on_home, version
	           FROM dynamic_tools`
	var args []any
	if source == "ai" {
		query += " WHERE is_ai_generated = 1"
	}
	query += " ORDER BY updated_at DESC"

	rows, err := tm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	defer rows.Close()

	var tools []*DynamicTool
	for rows.Next() {
		t, err := scanTool(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

// GetByID retrieves a single tool by its primary key.
func (tm *ToolManager) GetByID(ctx context.Context, id string) (*DynamicTool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	row := tm.db.QueryRowContext(ctx, `
		SELECT id, name, description, category, chat_schema, space_schema,
		       engine, fetch_config, content_hash, created_at, updated_at,
		       is_ai_generated, is_on_home, version
		FROM dynamic_tools WHERE id = ?`, id)
	t, err := scanToolRow(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// FindByHash looks up a tool by its content_hash (used for dedup).
func (tm *ToolManager) FindByHash(ctx context.Context, hash string) (*DynamicTool, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	row := tm.db.QueryRowContext(ctx, `
		SELECT id, name, description, category, chat_schema, space_schema,
		       engine, fetch_config, content_hash, created_at, updated_at,
		       is_ai_generated, is_on_home, version
		FROM dynamic_tools WHERE content_hash = ?`, hash)
	t, err := scanToolRow(row)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// Insert persists a new tool record.
func (tm *ToolManager) Insert(ctx context.Context, tool *DynamicTool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now().UnixMilli()
	tool.CreatedAt = now
	tool.UpdatedAt = now
	if tool.Version == 0 {
		tool.Version = 1
	}

	chatJSON, err := json.Marshal(tool.ChatSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal chat_schema: %w", err)
	}
	spaceJSON, err := json.Marshal(tool.SpaceSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal space_schema: %w", err)
	}

	var fetchJSON []byte
	if tool.FetchConfig != nil {
		fetchJSON, err = json.Marshal(tool.FetchConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal fetch_config: %w", err)
		}
	}

	_, err = tm.db.ExecContext(ctx, `
		INSERT INTO dynamic_tools
			(id, name, description, category, chat_schema, space_schema,
			 engine, fetch_config, content_hash, created_at, updated_at,
			 is_ai_generated, is_on_home, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		tool.ID, tool.Name, tool.Description, tool.Category,
		chatJSON, spaceJSON, tool.Engine, fetchJSON,
		tool.ContentHash, tool.CreatedAt, tool.UpdatedAt,
		boolToInt(tool.IsAIGenerated), boolToInt(tool.IsOnHome), tool.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to insert tool: %w", err)
	}
	return nil
}

// Update persists changes to an existing tool and bumps its version.
func (tm *ToolManager) Update(ctx context.Context, tool *DynamicTool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tool.UpdatedAt = time.Now().UnixMilli()
	tool.Version++

	chatJSON, err := json.Marshal(tool.ChatSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal chat_schema: %w", err)
	}
	spaceJSON, err := json.Marshal(tool.SpaceSchema)
	if err != nil {
		return fmt.Errorf("failed to marshal space_schema: %w", err)
	}

	var fetchJSON []byte
	if tool.FetchConfig != nil {
		fetchJSON, err = json.Marshal(tool.FetchConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal fetch_config: %w", err)
		}
	}

	res, err := tm.db.ExecContext(ctx, `
		UPDATE dynamic_tools
		SET name = ?, description = ?, category = ?,
		    chat_schema = ?, space_schema = ?, engine = ?, fetch_config = ?,
		    content_hash = ?, updated_at = ?, is_ai_generated = ?,
		    is_on_home = ?, version = ?
		WHERE id = ?`,
		tool.Name, tool.Description, tool.Category,
		chatJSON, spaceJSON, tool.Engine, fetchJSON,
		tool.ContentHash, tool.UpdatedAt,
		boolToInt(tool.IsAIGenerated), boolToInt(tool.IsOnHome),
		tool.Version, tool.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update tool: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SetOnHome toggles the home-display flag for a tool.
func (tm *ToolManager) SetOnHome(ctx context.Context, id string, onHome bool) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	res, err := tm.db.ExecContext(ctx, `
		UPDATE dynamic_tools SET is_on_home = ?, updated_at = ? WHERE id = ?`,
		boolToInt(onHome), time.Now().UnixMilli(), id,
	)
	if err != nil {
		return fmt.Errorf("failed to set on_home: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a tool by ID.
func (tm *ToolManager) Delete(ctx context.Context, id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	res, err := tm.db.ExecContext(ctx, `DELETE FROM dynamic_tools WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete tool: %w", err)
	}

	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ---------------------------------------------------------------------------
// ID generation
// ---------------------------------------------------------------------------

// GenerateID produces a unique ID in the form "category_hex" where hex is a
// random 12-character hex string (48 bits of entropy).
func GenerateID(category string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return category + "_" + hex.EncodeToString(b)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func intToBool(v int) bool {
	return v != 0
}

func scanTool(rows *sql.Rows) (*DynamicTool, error) {
	var t DynamicTool
	var chatJSON, spaceJSON []byte
	var fetchJSON []byte
	var isAI, isOnHome int

	err := rows.Scan(
		&t.ID, &t.Name, &t.Description, &t.Category,
		&chatJSON, &spaceJSON, &t.Engine, &fetchJSON,
		&t.ContentHash, &t.CreatedAt, &t.UpdatedAt,
		&isAI, &isOnHome, &t.Version,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan tool: %w", err)
	}

	if err := json.Unmarshal(chatJSON, &t.ChatSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat_schema: %w", err)
	}
	if err := json.Unmarshal(spaceJSON, &t.SpaceSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space_schema: %w", err)
	}
	if len(fetchJSON) > 0 {
		if err := json.Unmarshal(fetchJSON, &t.FetchConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal fetch_config: %w", err)
		}
	}
	t.IsAIGenerated = intToBool(isAI)
	t.IsOnHome = intToBool(isOnHome)
	return &t, nil
}

func scanToolRow(row *sql.Row) (*DynamicTool, error) {
	var t DynamicTool
	var chatJSON, spaceJSON []byte
	var fetchJSON []byte
	var isAI, isOnHome int

	err := row.Scan(
		&t.ID, &t.Name, &t.Description, &t.Category,
		&chatJSON, &spaceJSON, &t.Engine, &fetchJSON,
		&t.ContentHash, &t.CreatedAt, &t.UpdatedAt,
		&isAI, &isOnHome, &t.Version,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan tool: %w", err)
	}

	if err := json.Unmarshal(chatJSON, &t.ChatSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal chat_schema: %w", err)
	}
	if err := json.Unmarshal(spaceJSON, &t.SpaceSchema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal space_schema: %w", err)
	}
	if len(fetchJSON) > 0 {
		if err := json.Unmarshal(fetchJSON, &t.FetchConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal fetch_config: %w", err)
		}
	}
	t.IsAIGenerated = intToBool(isAI)
	t.IsOnHome = intToBool(isOnHome)
	return &t, nil
}
