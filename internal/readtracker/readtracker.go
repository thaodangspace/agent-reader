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
		marked_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_read_sessions_marked_at ON read_sessions(marked_at);
	`
	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
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

// MarkRead marks a session as read.
func (rt *ReadTracker) MarkRead(sessionID string) {
	query := `
	INSERT INTO read_sessions (session_id, marked_at)
	VALUES (?, ?)
	ON CONFLICT(session_id) DO UPDATE SET marked_at = excluded.marked_at
	`
	_, err := rt.db.Exec(query, sessionID, time.Now())
	if err != nil {
		log.Printf("[readtracker] failed to mark session %s as read: %v", sessionID, err)
	}
}

// IsRead returns true if the session has been marked as read and the entry has not expired.
func (rt *ReadTracker) IsRead(sessionID string) bool {
	var exists bool
	cutoff := time.Now().Add(-rt.ttl)
	query := `
	SELECT EXISTS(
		SELECT 1 FROM read_sessions
		WHERE session_id = ? AND marked_at >= ?
	)
	`
	err := rt.db.QueryRow(query, sessionID, cutoff).Scan(&exists)
	if err != nil {
		log.Printf("[readtracker] failed to query read status for session %s: %v", sessionID, err)
		return false
	}
	return exists
}

// IsUnread returns true if the session has NOT been marked as read (or the entry has expired).
func (rt *ReadTracker) IsUnread(sessionID string) bool {
	return !rt.IsRead(sessionID)
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
