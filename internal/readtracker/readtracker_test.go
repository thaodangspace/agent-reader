package readtracker

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReadTrackerBasic(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_read_tracker.db")

	// Initialize tracker
	ttl := 1 * time.Hour
	rt, err := New(dbPath, ttl)
	if err != nil {
		t.Fatalf("failed to initialize ReadTracker: %v", err)
	}
	defer rt.Stop()

	sessionID := "session-123"

	// Initially, it should be unread
	if !rt.IsUnread(sessionID, 10) {
		t.Errorf("expected session %s to be unread initially", sessionID)
	}
	if rt.IsRead(sessionID, 10) {
		t.Errorf("expected session %s NOT to be read initially", sessionID)
	}

	// Mark as read with 10 lines seen
	rt.MarkRead(sessionID, 10)

	// Now it should be read (current lines = 10, last seen = 10)
	if !rt.IsRead(sessionID, 10) {
		t.Errorf("expected session %s to be marked read", sessionID)
	}
	if rt.IsUnread(sessionID, 10) {
		t.Errorf("expected session %s NOT to be unread", sessionID)
	}

	// New messages arrive → 15 lines → should be unread
	if rt.IsRead(sessionID, 15) {
		t.Errorf("expected session %s to be unread after new messages (15 > 10)", sessionID)
	}
	if !rt.IsUnread(sessionID, 15) {
		t.Errorf("expected session %s to be unread after new messages (15 > 10)", sessionID)
	}

	// User re-opens → mark read with 15 lines
	rt.MarkRead(sessionID, 15)
	if !rt.IsRead(sessionID, 15) {
		t.Errorf("expected session %s to be marked read again", sessionID)
	}
}

func TestReadTrackerTTLAndCleanup(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_read_tracker_ttl.db")

	ttl := 100 * time.Millisecond
	rt, err := New(dbPath, ttl)
	if err != nil {
		t.Fatalf("failed to initialize ReadTracker: %v", err)
	}
	defer rt.Stop()

	session1 := "session-active"
	session2 := "session-expired"

	// Mark both as read
	rt.MarkRead(session1, 10)
	rt.MarkRead(session2, 10)

	// Force-update the marked_at timestamp of session2 directly in DB to simulate expiration
	expiredTime := time.Now().Add(-5 * time.Hour)
	_, err = rt.db.Exec("UPDATE read_sessions SET marked_at = ? WHERE session_id = ?", expiredTime, session2)
	if err != nil {
		t.Fatalf("failed to manually expire session: %v", err)
	}

	// Active one should still be read, expired one should not
	if !rt.IsRead(session1, 10) {
		t.Errorf("expected %s to be read", session1)
	}
	if rt.IsRead(session2, 10) {
		t.Errorf("expected %s to be expired (not read)", session2)
	}

	// Run cleanup manually and check if expired entry is deleted from the DB
	rt.cleanup()

	var count int
	err = rt.db.QueryRow("SELECT COUNT(*) FROM read_sessions WHERE session_id = ?", session2).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected expired session to be deleted by cleanup, but count was %d", count)
	}

	// Active session should still be in the DB
	err = rt.db.QueryRow("SELECT COUNT(*) FROM read_sessions WHERE session_id = ?", session1).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected active session to remain in DB, but count was %d", count)
	}
}

func TestReadTrackerConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_read_tracker_concurrency.db")

	rt, err := New(dbPath, 1*time.Hour)
	if err != nil {
		t.Fatalf("failed to initialize ReadTracker: %v", err)
	}
	defer rt.Stop()

	// Run multiple readers and writers concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 50; j++ {
				sessionID := "session-i-j"
				rt.MarkRead(sessionID, j)
				_ = rt.IsRead(sessionID, j)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
