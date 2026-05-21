package watcher

import (
	"os"
	"path/filepath"
	"testing"

	"agent-reader/internal/jsonl"
)

func TestExtractCodexMeta(t *testing.T) {
	path := filepath.Join("/Users/dt/.codex/sessions/2026/05/19", "rollout-2026-05-19T09-39-36-019e3e1a-5f70-7511-84e4-fb07e05f6234.jsonl")
	sessionID, project := extractCodexMeta(path, "/Users/dt/code/dotfiles", "meta-session-id")
	if sessionID != "019e3e1a-5f70-7511-84e4-fb07e05f6234" {
		t.Fatalf("unexpected session id: %q", sessionID)
	}
	if project != "dotfiles" {
		t.Fatalf("unexpected project: %q", project)
	}

	path = filepath.Join("/Users/dt/.codex/sessions/2026/05/19", "rollout-test.jsonl")
	sessionID, project = extractCodexMeta(path, "/tmp/project", "meta-session-id")
	if sessionID != "meta-session-id" {
		t.Fatalf("unexpected fallback session id: %q", sessionID)
	}
	if project != "project" {
		t.Fatalf("unexpected fallback project: %q", project)
	}
}

func TestCodexFileIsUserFacing(t *testing.T) {
	dir := t.TempDir()
	userFile := filepath.Join(dir, "rollout-user.jsonl")
	subFile := filepath.Join(dir, "rollout-sub.jsonl")
	if err := os.WriteFile(userFile, []byte(`{"type":"session_meta","payload":{"id":"user-1","cwd":"/tmp/project","thread_source":"user"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subFile, []byte(`{"type":"session_meta","payload":{"id":"sub-1","cwd":"/tmp/project","thread_source":"subagent","source":{"subagent":{"other":"guardian"}}}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	meta, ok := readCodexFileMeta(userFile)
	if !ok || meta.ID != "user-1" {
		t.Fatalf("expected user meta, got %#v ok=%v", meta, ok)
	}
	if _, ok := readCodexFileMeta(subFile); ok {
		t.Fatal("expected subagent file to be filtered")
	}
}

func TestCodexWatcherCloseDecoders(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-test.jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session_meta","payload":{"id":"user-1","cwd":"/tmp/project","thread_source":"user"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	dec, err := jsonl.NewCodexDecoder(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	w := &CodexWatcher{
		decoders: map[string]*codexDecoderEntry{
			path: {dec: dec},
		},
	}

	w.closeDecoders()
	if len(w.decoders) != 0 {
		t.Fatalf("expected decoder map to be cleared, got %d entries", len(w.decoders))
	}
	if _, err := dec.Next(); err == nil {
		t.Fatal("expected closed decoder to reject reads")
	}
}
