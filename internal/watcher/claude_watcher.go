// Package watcher monitors session directories for JSONL file changes.
package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-reader/internal/jsonl"

	"github.com/fsnotify/fsnotify"
)

// ClaudeWatcher uses fsnotify to tail Claude Code JSONL files in ~/.claude/projects/.
type ClaudeWatcher struct {
	baseDir  string
	fsw      *fsnotify.Watcher
	decoders map[string]*claudeDecoderEntry // path -> decoder
	mu       sync.Mutex
	events   chan Event
	quit     chan struct{}
	wg       sync.WaitGroup
}

type claudeDecoderEntry struct {
	dec    *jsonl.ClaudeDecoder
	proj   string // project path (from first event's cwd)
	sessID string // session ID from filename
}

// NewClaudeWatcher creates a watcher for Claude Code session files.
func NewClaudeWatcher(baseDir string) (*ClaudeWatcher, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve path %s: %w", baseDir, err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify: %w", err)
	}

	w := &ClaudeWatcher{
		baseDir:  abs,
		fsw:      fsw,
		decoders: make(map[string]*claudeDecoderEntry),
		events:   make(chan Event, 1024),
		quit:     make(chan struct{}),
	}

	if err := w.addWatches(); err != nil {
		fsw.Close()
		return nil, err
	}

	return w, nil
}

// Events returns the read-only event channel.
func (w *ClaudeWatcher) Events() <-chan Event {
	return w.events
}

// Start begins watching.
func (w *ClaudeWatcher) Start() {
	w.wg.Add(2)
	go w.watchLoop()
	go w.scanLoop()
}

// Stop signals the watcher to shut down.
func (w *ClaudeWatcher) Stop() {
	close(w.quit)
	w.fsw.Close()
	w.wg.Wait()
	close(w.events)
}

func (w *ClaudeWatcher) watchLoop() {
	defer w.wg.Done()
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			w.handleFSNotify(ev)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("[claude-watcher] error: %v", err)
		case <-w.quit:
			return
		}
	}
}

func (w *ClaudeWatcher) scanLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.addWatches()
		case <-w.quit:
			return
		}
	}
}

// addWatches walks baseDir and adds watches for all subdirectories.
// Skips "subagents/" directories — those are Claude Code subagent sessions, not top-level.
func (w *ClaudeWatcher) addWatches() error {
	return filepath.WalkDir(w.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip subagents directories
			if strings.HasSuffix(filepath.Base(path), "subagents") {
				return filepath.SkipDir
			}
			w.fsw.Add(path)
		}
		return nil
	})
}

func (w *ClaudeWatcher) handleFSNotify(ev fsnotify.Event) {
	if filepath.Ext(ev.Name) != ".jsonl" {
		return
	}

	// Skip subagents
	if strings.Contains(ev.Name, "/subagents/") {
		return
	}

	if ev.Op.Has(fsnotify.Create) || ev.Op.Has(fsnotify.Write) {
		w.tailFile(ev.Name)
	}
}

func (w *ClaudeWatcher) tailFile(path string) {
	w.mu.Lock()
	entry, exists := w.decoders[path]
	w.mu.Unlock()

	if !exists {
		dec, err := jsonl.NewClaudeDecoder(path, 0)
		if err != nil {
			log.Printf("[claude-watcher] open %s: %v", path, err)
			return
		}

		sessionID, project := extractClaudeMeta(path)

		entry = &claudeDecoderEntry{
			dec:    dec,
			proj:   project,
			sessID: sessionID,
		}
		w.mu.Lock()
		w.decoders[path] = entry
		w.mu.Unlock()
	}

	for {
		event, err := entry.dec.Next()
		if err != nil {
			break
		}
		if event == nil {
			continue
		}

		w.events <- Event{
			SessionID: entry.sessID,
			Project:   entry.proj,
			File:      path,
			Data:      event.Raw,
			Timestamp: time.Now(),
		}
	}
}

// extractClaudeMeta pulls session ID and project from a Claude Code session file path.
// Claude Code files are named <uuid>.jsonl in ~/.claude/projects/-<pathhash>/
func extractClaudeMeta(path string) (sessionID, project string) {
	base := filepath.Base(path)
	sessionID = strings.TrimSuffix(base, ".jsonl")

	// Project is the directory name (pathhash), but we'll update it from cwd later
	dir := filepath.Dir(path)
	project = filepath.Base(dir)

	return sessionID, project
}
