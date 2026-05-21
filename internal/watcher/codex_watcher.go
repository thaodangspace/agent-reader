package watcher

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"agent-reader/internal/jsonl"

	"github.com/fsnotify/fsnotify"
)

// CodexWatcher uses fsnotify to tail Codex JSONL files in ~/.codex/sessions/.
type CodexWatcher struct {
	baseDir  string
	fsw      *fsnotify.Watcher
	decoders map[string]*codexDecoderEntry
	mu       sync.Mutex
	events   chan Event
	quit     chan struct{}
	wg       sync.WaitGroup
}

type codexDecoderEntry struct {
	dec    *jsonl.CodexDecoder
	proj   string
	sessID string
}

// NewCodexWatcher creates a watcher for Codex session files.
func NewCodexWatcher(baseDir string) (*CodexWatcher, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve path %s: %w", baseDir, err)
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify: %w", err)
	}

	w := &CodexWatcher{
		baseDir:  abs,
		fsw:      fsw,
		decoders: make(map[string]*codexDecoderEntry),
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
func (w *CodexWatcher) Events() <-chan Event {
	return w.events
}

// Start begins watching.
func (w *CodexWatcher) Start() {
	w.wg.Add(2)
	go w.watchLoop()
	go w.scanLoop()
}

// Stop signals the watcher to shut down.
func (w *CodexWatcher) Stop() {
	close(w.quit)
	w.fsw.Close()
	w.wg.Wait()
	w.closeDecoders()
	close(w.events)
}

func (w *CodexWatcher) watchLoop() {
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
			log.Printf("[codex-watcher] error: %v", err)
		case <-w.quit:
			return
		}
	}
}

func (w *CodexWatcher) scanLoop() {
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
func (w *CodexWatcher) addWatches() error {
	return filepath.WalkDir(w.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			w.fsw.Add(path)
		}
		return nil
	})
}

func (w *CodexWatcher) handleFSNotify(ev fsnotify.Event) {
	if filepath.Ext(ev.Name) != ".jsonl" {
		return
	}

	if ev.Op.Has(fsnotify.Create) || ev.Op.Has(fsnotify.Write) {
		w.tailFile(ev.Name)
	}
}

func (w *CodexWatcher) tailFile(path string) {
	w.mu.Lock()
	entry, exists := w.decoders[path]
	w.mu.Unlock()

	if !exists {
		meta, ok := readCodexFileMeta(path)
		if !ok {
			return
		}

		dec, err := jsonl.NewCodexDecoder(path, 0)
		if err != nil {
			log.Printf("[codex-watcher] open %s: %v", path, err)
			return
		}

		sessionID, project := extractCodexMeta(path, meta.CWD, meta.ID)
		entry = &codexDecoderEntry{
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

		ev := Event{
			SessionID: entry.sessID,
			Project:   entry.proj,
			File:      path,
			Data:      event.Raw,
			Timestamp: time.Now(),
		}
		select {
		case w.events <- ev:
		case <-w.quit:
			return
		}
	}
}

func (w *CodexWatcher) closeDecoders() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for path, entry := range w.decoders {
		if entry != nil && entry.dec != nil {
			if err := entry.dec.Close(); err != nil {
				log.Printf("[codex-watcher] close %s: %v", path, err)
			}
		}
		delete(w.decoders, path)
	}
}

func readCodexFileMeta(path string) (jsonl.CodexSessionMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return jsonl.CodexSessionMeta{}, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		meta, ok := jsonl.ParseCodexSessionMeta(scanner.Bytes())
		if !ok {
			continue
		}
		if jsonl.IsCodexUserSession(meta) {
			return meta, true
		}
		return jsonl.CodexSessionMeta{}, false
	}

	return jsonl.CodexSessionMeta{}, false
}

var codexRolloutSessionIDRe = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func extractCodexMeta(path, cwd, metaID string) (sessionID, project string) {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if match := codexRolloutSessionIDRe.FindString(base); match != "" {
		sessionID = match
	}
	if sessionID == "" {
		sessionID = metaID
	}
	if sessionID == "" {
		sessionID = base
	}

	if cwd != "" {
		project = filepath.Base(cwd)
	} else {
		project = filepath.Base(filepath.Dir(path))
	}
	return sessionID, project
}
