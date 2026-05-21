// Package readtracker tracks which sessions have been read by the user.
// Uses a SQLite database to store read states persistently.
package readtracker

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ReadTracker tracks read status for recent sessions in a SQLite database.
type ReadTracker struct {
	db        *sql.DB
	ttl       time.Duration
	stopCh    chan struct{}
	stoppedCh chan struct{}
}

// New creates a new SQLite-backed ReadTracker with the given dbPath and TTL.
// A background goroutine periodically cleans up expired entries.
func New(dbPath string, ttl time.Duration) (*ReadTracker, error) {
	// Ensure the parent directory of the database file exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	// Limit to a single connection to serialize writes and avoid locking issues
	db.SetMaxOpenConns(1)

	// Enable WAL mode and set busy timeout for concurrency optimization
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	// Create table and index
	query := `
	CREATE TABLE IF NOT EXISTS read_sessions (
		session_id TEXT PRIMARY KEY,
		marked_at TIMESTAMP NOT NULL,
		last_seen_lines INTEGER NOT NULL DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_read_sessions_marked_at ON read_sessions(marked_at);
	`
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	// Migration: add last_seen_lines column if it doesn't exist (for existing DBs)
	var colExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('read_sessions') WHERE name='last_seen_lines'`).Scan(&colExists); err == nil && colExists == 0 {
		if _, migrErr := db.Exec(`ALTER TABLE read_sessions ADD COLUMN last_seen_lines INTEGER NOT NULL DEFAULT 0`); migrErr != nil {
			log.Printf("[readtracker] warning: failed to add last_seen_lines column: %v", migrErr)
		} else {
			log.Printf("[readtracker] migrated: added last_seen_lines column")
		}
	}

	rt := &ReadTracker{
		db:        db,
		ttl:       ttl,
		stopCh:    make(chan struct{}),
		stoppedCh: make(chan struct{}),
	}

	go rt.cleanupLoop()

	return rt, nil
}

// MarkRead marks a session as read, recording the line count the user has seen.
// A session is considered unread again when its current line count exceeds lastSeenLines.
func (rt *ReadTracker) MarkRead(sessionID string, lastSeenLines int) {
	query := `
	INSERT INTO read_sessions (session_id, marked_at, last_seen_lines)
	VALUES (?, ?, ?)
	ON CONFLICT(session_id) DO UPDATE SET marked_at = excluded.marked_at, last_seen_lines = excluded.last_seen_lines
	`
	_, err := rt.db.Exec(query, sessionID, time.Now(), lastSeenLines)
	if err != nil {
		log.Printf("[readtracker] failed to mark session %s as read: %v", sessionID, err)
	}
}

// IsRead returns true if the session has been marked as read, the entry has not expired,
// and the current line count has not exceeded the last seen line count.
func (rt *ReadTracker) IsRead(sessionID string, currentLines int) bool {
	var lastSeenLines int
	cutoff := time.Now().Add(-rt.ttl)
	query := `
	SELECT last_seen_lines FROM read_sessions
	WHERE session_id = ? AND marked_at >= ?
	`
	err := rt.db.QueryRow(query, sessionID, cutoff).Scan(&lastSeenLines)
	if err != nil {
		// Not found or expired → unread
		return false
	}
	return currentLines <= lastSeenLines
}

// IsUnread returns true if the session has NOT been marked as read (or the entry has expired),
// or if new lines have been added since the user last viewed it.
func (rt *ReadTracker) IsUnread(sessionID string, currentLines int) bool {
	return !rt.IsRead(sessionID, currentLines)
}

// cleanupLoop periodically removes expired entries.
func (rt *ReadTracker) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	defer close(rt.stoppedCh)

	for {
		select {
		case <-ticker.C:
			rt.cleanup()
		case <-rt.stopCh:
			rt.cleanup()
			return
		}
	}
}

// cleanup removes database entries older than the TTL.
func (rt *ReadTracker) cleanup() {
	cutoff := time.Now().Add(-rt.ttl)
	_, err := rt.db.Exec("DELETE FROM read_sessions WHERE marked_at < ?", cutoff)
	if err != nil {
		log.Printf("[readtracker] failed to clean up expired sessions: %v", err)
	}
}

// Stop stops the background cleanup goroutine and closes the database connection.
func (rt *ReadTracker) Stop() {
	close(rt.stopCh)
	<-rt.stoppedCh
	if err := rt.db.Close(); err != nil {
		log.Printf("[readtracker] failed to close database connection: %v", err)
	}
}
