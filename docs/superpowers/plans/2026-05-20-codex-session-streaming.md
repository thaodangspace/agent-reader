# Codex Session Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add read-only Codex session listing, replay, and live streaming from `~/.codex/sessions`.

**Architecture:** Add a Codex JSONL decoder and watcher parallel to the existing Claude support. The backend filters to user-facing Codex rollouts, normalizes visible Codex records into the existing pi-agent-style `message` events, and exposes Codex sessions through the current `/api/sessions` and WebSocket flow. The frontend only needs read-only capability handling for `agent:"codex"`.

**Tech Stack:** Go, fsnotify, existing `internal/jsonl`, `internal/watcher`, `internal/server`, Svelte frontend helpers, Node built-in test runner.

---

## File Structure

- Create `internal/jsonl/codex_types.go`: tolerant structs for Codex JSONL records and metadata.
- Create `internal/jsonl/codex_decoder.go`: line decoder, user-facing session detection, metadata extraction, and normalization to `jsonl.Event`.
- Create `internal/jsonl/codex_decoder_test.go`: unit tests for filtering, metadata, message normalization, tool calls, tool results, and dropped bookkeeping.
- Create `internal/watcher/codex_watcher.go`: fsnotify watcher for `~/.codex/sessions`.
- Create `internal/watcher/codex_watcher_test.go`: watcher metadata/filter tests that do not require fsnotify events.
- Modify `internal/hub/hub.go`: subscribe to the Codex watcher or factor common watcher subscription.
- Modify `internal/server/server.go`: accept Codex directory, start/stop watcher, scan Codex sessions, replay Codex sessions.
- Modify `cmd/server/main.go`: add `-codex-sessions` flag with default `~/.codex/sessions`.
- Modify `frontend/src/lib/utils/sessionCapabilities.js`: mark Codex as read-only.
- Modify `frontend/src/lib/utils/sessionCapabilities.test.mjs`: add Codex read-only test.
- Modify `frontend/src/lib/actions/rpc.js` and `frontend/src/lib/components/ChatArea.svelte`: use generic read-only text or Codex-specific read-only text where Claude is currently hard-coded.

---

### Task 1: Codex Types, Metadata, And Filtering

**Files:**
- Create: `internal/jsonl/codex_types.go`
- Create: `internal/jsonl/codex_decoder.go`
- Create: `internal/jsonl/codex_decoder_test.go`

- [ ] **Step 1: Write failing tests for Codex session metadata and filtering**

Create `internal/jsonl/codex_decoder_test.go` with:

```go
package jsonl

import "testing"

func TestParseCodexSessionMeta_UserFacing(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:39:55.659Z","type":"session_meta","payload":{"id":"019e3e1a-5f70-7511-84e4-fb07e05f6234","timestamp":"2026-05-19T02:39:36.304Z","cwd":"/Users/dt/code/dotfiles","source":"cli","thread_source":"user","model_provider":"openai"}}`

	meta, ok := ParseCodexSessionMeta([]byte(line))
	if !ok {
		t.Fatal("expected session_meta to parse")
	}
	if meta.ID != "019e3e1a-5f70-7511-84e4-fb07e05f6234" {
		t.Fatalf("unexpected id: %q", meta.ID)
	}
	if meta.CWD != "/Users/dt/code/dotfiles" {
		t.Fatalf("unexpected cwd: %q", meta.CWD)
	}
	if !IsCodexUserSession(meta) {
		t.Fatal("expected user-facing Codex session")
	}
}

func TestParseCodexSessionMeta_DropsSubagent(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:42:17.656Z","type":"session_meta","payload":{"id":"019e3e1c-d053-7d50-b72b-a85cbf675322","cwd":"/Users/dt/code/dotfiles","source":{"subagent":{"other":"guardian"}},"thread_source":"subagent","model_provider":"openai"}}`

	meta, ok := ParseCodexSessionMeta([]byte(line))
	if !ok {
		t.Fatal("expected session_meta to parse")
	}
	if IsCodexUserSession(meta) {
		t.Fatal("expected subagent session to be excluded")
	}
}

func TestParseCodexSessionMeta_DropsMissingID(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:39:55.659Z","type":"session_meta","payload":{"cwd":"/Users/dt/code/dotfiles","thread_source":"user"}}`

	meta, ok := ParseCodexSessionMeta([]byte(line))
	if !ok {
		t.Fatal("expected session_meta to parse")
	}
	if IsCodexUserSession(meta) {
		t.Fatal("expected session with no id to be excluded")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/jsonl -run 'TestParseCodexSessionMeta' -v
```

Expected: FAIL with undefined `ParseCodexSessionMeta` and `IsCodexUserSession`.

- [ ] **Step 3: Add Codex metadata structs**

Create `internal/jsonl/codex_types.go`:

```go
package jsonl

import "encoding/json"

type CodexEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type CodexSessionMeta struct {
	ID            string          `json:"id"`
	Timestamp     string          `json:"timestamp"`
	CWD           string          `json:"cwd"`
	Originator    string          `json:"originator"`
	CLIVersion    string          `json:"cli_version"`
	Source        json.RawMessage `json:"source"`
	ThreadSource  string          `json:"thread_source"`
	ModelProvider string          `json:"model_provider"`
	Model         string          `json:"model"`
}

type CodexMessage struct {
	Type    string              `json:"type"`
	Role    string              `json:"role"`
	Content []CodexContentBlock `json:"content"`
}

type CodexContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type CodexFunctionCall struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
}

type CodexFunctionCallOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}
```

- [ ] **Step 4: Add metadata parsing and filtering helpers**

Create `internal/jsonl/codex_decoder.go` with this initial content:

```go
package jsonl

import (
	"encoding/json"
	"strings"
)

func ParseCodexSessionMeta(line []byte) (CodexSessionMeta, bool) {
	var env CodexEnvelope
	if err := json.Unmarshal(line, &env); err != nil {
		return CodexSessionMeta{}, false
	}
	if env.Type != "session_meta" {
		return CodexSessionMeta{}, false
	}
	var payload CodexSessionMeta
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return CodexSessionMeta{}, false
	}
	return payload, true
}

func IsCodexUserSession(meta CodexSessionMeta) bool {
	if meta.ID == "" {
		return false
	}
	if meta.ThreadSource == "subagent" {
		return false
	}
	if sourceNamesInternalCodexSession(meta.Source) {
		return false
	}
	model := strings.ToLower(meta.Model)
	if strings.Contains(model, "codex-auto-review") || strings.Contains(model, "guardian") {
		return false
	}
	if meta.ThreadSource == "user" {
		return true
	}
	return meta.ThreadSource == ""
}

func sourceNamesInternalCodexSession(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		v := strings.ToLower(s)
		return strings.Contains(v, "guardian") || strings.Contains(v, "auto-review") || strings.Contains(v, "approval")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return false
	}
	if _, ok := obj["subagent"]; ok {
		return true
	}
	for key := range obj {
		v := strings.ToLower(key)
		if strings.Contains(v, "guardian") || strings.Contains(v, "auto-review") || strings.Contains(v, "approval") {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
go test ./internal/jsonl -run 'TestParseCodexSessionMeta' -v
```

Expected: PASS for the three metadata/filtering tests.

- [ ] **Step 6: Commit Task 1**

```bash
git add internal/jsonl/codex_types.go internal/jsonl/codex_decoder.go internal/jsonl/codex_decoder_test.go
git commit -m "feat(jsonl): add codex session metadata parsing"
```

---

### Task 2: Codex Decoder Normalization

**Files:**
- Modify: `internal/jsonl/codex_decoder.go`
- Modify: `internal/jsonl/codex_decoder_test.go`

- [ ] **Step 1: Add failing tests for visible message normalization**

Append to `internal/jsonl/codex_decoder_test.go`:

```go
func TestCodexDecoderNormalizeMessage_User(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:39:56.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"update tmux and kitty"}]}}`

	out, drop := normalizeCodexLine(line)
	if drop {
		t.Fatal("expected user message")
	}
	if out.Type != "message" {
		t.Fatalf("expected message event, got %q", out.Type)
	}
	var msg MessageEvent
	if err := json.Unmarshal(out.Raw, &msg); err != nil {
		t.Fatalf("invalid message JSON: %v", err)
	}
	if msg.Message.Role != "user" {
		t.Fatalf("expected user role, got %q", msg.Message.Role)
	}
	if len(msg.Message.Content) != 1 || msg.Message.Content[0].Text != "update tmux and kitty" {
		t.Fatalf("unexpected content: %#v", msg.Message.Content)
	}
}

func TestCodexDecoderNormalizeMessage_Assistant(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:41:59.106Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"The edits are in place."}]}}`

	out, drop := normalizeCodexLine(line)
	if drop {
		t.Fatal("expected assistant message")
	}
	var msg MessageEvent
	if err := json.Unmarshal(out.Raw, &msg); err != nil {
		t.Fatalf("invalid message JSON: %v", err)
	}
	if msg.Message.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", msg.Message.Role)
	}
	if msg.Message.Content[0].Text != "The edits are in place." {
		t.Fatalf("unexpected text: %q", msg.Message.Content[0].Text)
	}
}
```

- [ ] **Step 2: Add failing tests for tool calls, tool results, and dropped records**

Append to `internal/jsonl/codex_decoder_test.go`:

```go
func TestCodexDecoderNormalizeFunctionCall(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:42:10.685Z","type":"response_item","payload":{"type":"function_call","name":"exec_command","arguments":"{\"cmd\":\"git status --short\",\"workdir\":\"/Users/dt/code/dotfiles\"}","call_id":"call_j9lbx16QVDJBAHnTbcD2DPlZ"}}`

	out, drop := normalizeCodexLine(line)
	if drop {
		t.Fatal("expected tool call message")
	}
	var msg MessageEvent
	if err := json.Unmarshal(out.Raw, &msg); err != nil {
		t.Fatalf("invalid message JSON: %v", err)
	}
	if msg.Message.Role != "assistant" {
		t.Fatalf("expected assistant role, got %q", msg.Message.Role)
	}
	block := msg.Message.Content[0]
	if block.Type != "toolCall" || block.ToolCallName != "exec_command" || block.ToolCallID != "call_j9lbx16QVDJBAHnTbcD2DPlZ" {
		t.Fatalf("unexpected tool call block: %#v", block)
	}
}

func TestCodexDecoderNormalizeFunctionCallOutput(t *testing.T) {
	line := `{"timestamp":"2026-05-19T02:42:05.990Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_j9lbx16QVDJBAHnTbcD2DPlZ","output":"Chunk ID: 65fc69\nWall time: 0.0000 seconds\nProcess exited with code 0\nOutput:\n M kitty/current-theme.conf\n"}}`

	out, drop := normalizeCodexLine(line)
	if drop {
		t.Fatal("expected tool result message")
	}
	var msg MessageEvent
	if err := json.Unmarshal(out.Raw, &msg); err != nil {
		t.Fatalf("invalid message JSON: %v", err)
	}
	if msg.Message.Role != "toolResult" {
		t.Fatalf("expected toolResult role, got %q", msg.Message.Role)
	}
	if msg.Message.ToolCallID != "call_j9lbx16QVDJBAHnTbcD2DPlZ" {
		t.Fatalf("unexpected toolCallId: %q", msg.Message.ToolCallID)
	}
	if len(msg.Message.Content) != 1 || msg.Message.Content[0].Type != "text" {
		t.Fatalf("unexpected content: %#v", msg.Message.Content)
	}
}

func TestCodexDecoderDropsBookkeeping(t *testing.T) {
	lines := []string{
		`{"timestamp":"2026-05-19T02:39:55.659Z","type":"session_meta","payload":{"id":"s","thread_source":"user"}}`,
		`{"timestamp":"2026-05-19T02:39:55.660Z","type":"turn_context","payload":{"model":"gpt-5.5"}}`,
		`{"timestamp":"2026-05-19T02:39:55.661Z","type":"event_msg","payload":{"type":"token_count"}}`,
		`{"timestamp":"2026-05-19T02:39:55.662Z","type":"event_msg","payload":{"type":"agent_message","message":"duplicate visible text"}}`,
		`{"timestamp":"2026-05-19T02:39:55.663Z","type":"response_item","payload":{"type":"reasoning","summary":[]}}`,
	}
	for _, line := range lines {
		if _, drop := normalizeCodexLine(line); !drop {
			t.Fatalf("expected to drop %s", line)
		}
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run:

```bash
go test ./internal/jsonl -run 'TestCodexDecoder' -v
```

Expected: FAIL with undefined `normalizeCodexLine`.

- [ ] **Step 4: Implement `CodexDecoder`, line normalization, and `Next`**

Extend `internal/jsonl/codex_decoder.go` by adding the decoder implementation below the metadata helper functions from Task 1. Expand the import block to include all imports shown here:

```go
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Keep ParseCodexSessionMeta, IsCodexUserSession, and
// sourceNamesInternalCodexSession from Task 1 above this decoder code.

type CodexDecoder struct {
	path   string
	offset int64
	file   *os.File
	reader *bufio.Reader
}

func NewCodexDecoder(path string, offset int64) (*CodexDecoder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			f.Close()
			return nil, fmt.Errorf("seek %s: %w", path, err)
		}
	}
	return &CodexDecoder{path: path, offset: offset, file: f, reader: bufio.NewReader(f)}, nil
}

func (d *CodexDecoder) Offset() int64 { return d.offset }
func (d *CodexDecoder) Path() string { return d.path }
func (d *CodexDecoder) Close() error { return d.file.Close() }

func (d *CodexDecoder) Next() (*Event, error) {
	for {
		line, err := d.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line == "" {
					return nil, io.EOF
				}
			} else {
				return nil, fmt.Errorf("read %s: %w", d.path, err)
			}
		}
		d.offset += int64(len(line))
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if err == io.EOF {
				return nil, io.EOF
			}
			continue
		}
		ev, drop := normalizeCodexLine(trimmed)
		if drop {
			if err == io.EOF {
				return nil, io.EOF
			}
			return nil, nil
		}
		return ev, nil
	}
}

func normalizeCodexLine(line string) (*Event, bool) {
	var env CodexEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		return nil, true
	}
	if env.Type != "response_item" {
		return nil, true
	}
	var payload struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return nil, true
	}
	switch payload.Type {
	case "message":
		return normalizeCodexMessage(env.Timestamp, env.Payload)
	case "function_call":
		return normalizeCodexFunctionCall(env.Timestamp, env.Payload)
	case "function_call_output":
		return normalizeCodexFunctionCallOutput(env.Timestamp, env.Payload)
	default:
		return nil, true
	}
}

func normalizeCodexMessage(timestamp string, raw json.RawMessage) (*Event, bool) {
	var msg CodexMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, true
	}
	var blocks []ContentBlock
	for _, c := range msg.Content {
		if (c.Type == "input_text" || c.Type == "output_text" || c.Type == "text") && c.Text != "" {
			blocks = append(blocks, ContentBlock{Type: "text", Text: c.Text})
		}
	}
	if msg.Role == "" || len(blocks) == 0 {
		return nil, true
	}
	return marshalCodexEvent(timestamp, "codex-"+shortStableID(timestamp, msg.Role), map[string]interface{}{
		"role": msg.Role,
		"content": blocks,
	})
}

func normalizeCodexFunctionCall(timestamp string, raw json.RawMessage) (*Event, bool) {
	var call CodexFunctionCall
	if err := json.Unmarshal(raw, &call); err != nil {
		return nil, true
	}
	if call.CallID == "" || call.Name == "" {
		return nil, true
	}
	args := parseCodexArguments(call.Arguments)
	return marshalCodexEvent(timestamp, call.CallID, map[string]interface{}{
		"role": "assistant",
		"content": []ContentBlock{{
			Type: call.TypeForContent(),
			ToolCallID: call.CallID,
			ToolCallName: call.Name,
			Arguments: args,
		}},
	})
}

func (c CodexFunctionCall) TypeForContent() string {
	return "toolCall"
}

func normalizeCodexFunctionCallOutput(timestamp string, raw json.RawMessage) (*Event, bool) {
	var out CodexFunctionCallOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, true
	}
	if out.CallID == "" {
		return nil, true
	}
	return marshalCodexEvent(timestamp, out.CallID+"-result", map[string]interface{}{
		"role": "toolResult",
		"toolCallId": out.CallID,
		"content": []ContentBlock{{Type: "text", Text: out.Output}},
		"isError": false,
	})
}

func parseCodexArguments(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage(`{}`)
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(s), &raw); err == nil {
		return raw
	}
	b, _ := json.Marshal(map[string]string{"value": s})
	return json.RawMessage(b)
}

func marshalCodexEvent(timestamp, id string, message map[string]interface{}) (*Event, bool) {
	out := map[string]interface{}{
		"type": "message",
		"id": id,
		"timestamp": timestamp,
		"message": message,
	}
	b, err := json.Marshal(out)
	if err != nil {
		return nil, true
	}
	return &Event{Type: "message", ID: id, Timestamp: timestamp, Raw: b}, false
}

func shortStableID(timestamp, role string) string {
	clean := strings.NewReplacer(":", "", "-", "", ".", "", "Z", "").Replace(timestamp)
	if clean == "" {
		clean = "unknown"
	}
	return role + "-" + clean
}
```

- [ ] **Step 5: Run tests and fix compile details**

Run:

```bash
go test ./internal/jsonl -run 'TestCodexDecoder|TestParseCodexSessionMeta' -v
```

Expected: PASS. If Go rejects the helper method on `CodexFunctionCall`, replace `Type: call.TypeForContent()` with `Type: "toolCall"` and remove the method.

- [ ] **Step 6: Add decoder file-read test**

Append to `internal/jsonl/codex_decoder_test.go`:

```go
func TestCodexDecoderNext(t *testing.T) {
	content := `{"timestamp":"2026-05-19T02:39:55.659Z","type":"session_meta","payload":{"id":"s","thread_source":"user"}}` + "\n" +
		`{"timestamp":"2026-05-19T02:39:56.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}}` + "\n"
	path := t.TempDir() + "/rollout-test.jsonl"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	dec, err := NewCodexDecoder(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()
	ev, err := dec.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev != nil {
		t.Fatalf("first bookkeeping line should be dropped, got %#v", ev)
	}
	ev, err = dec.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "message" {
		t.Fatalf("expected message, got %q", ev.Type)
	}
}
```

Also add missing imports to the test file:

```go
import (
	"encoding/json"
	"os"
	"testing"
)
```

- [ ] **Step 7: Run full jsonl tests**

Run:

```bash
go test ./internal/jsonl -v
```

Expected: PASS.

- [ ] **Step 8: Commit Task 2**

```bash
git add internal/jsonl/codex_decoder.go internal/jsonl/codex_decoder_test.go
git commit -m "feat(jsonl): normalize codex session events"
```

---

### Task 3: Codex Watcher

**Files:**
- Create: `internal/watcher/codex_watcher.go`
- Create: `internal/watcher/codex_watcher_test.go`
- Modify: `internal/hub/hub.go`

- [ ] **Step 1: Write watcher helper tests**

Create `internal/watcher/codex_watcher_test.go`:

```go
package watcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractCodexMeta(t *testing.T) {
	path := filepath.Join("/Users/dt/.codex/sessions/2026/05/19", "rollout-2026-05-19T09-39-36-019e3e1a-5f70-7511-84e4-fb07e05f6234.jsonl")
	sessionID, project := extractCodexMeta(path, "/Users/dt/code/dotfiles")
	if sessionID != "019e3e1a-5f70-7511-84e4-fb07e05f6234" {
		t.Fatalf("unexpected session id: %q", sessionID)
	}
	if project != "dotfiles" {
		t.Fatalf("unexpected project: %q", project)
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
```

- [ ] **Step 2: Run watcher tests to verify they fail**

Run:

```bash
go test ./internal/watcher -run 'TestExtractCodexMeta|TestCodexFileIsUserFacing' -v
```

Expected: FAIL with undefined `extractCodexMeta` and `readCodexFileMeta`.

- [ ] **Step 3: Implement Codex watcher**

Create `internal/watcher/codex_watcher.go`:

```go
package watcher

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"agent-web/internal/jsonl"

	"github.com/fsnotify/fsnotify"
)

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
		baseDir: abs,
		fsw: fsw,
		decoders: make(map[string]*codexDecoderEntry),
		events: make(chan Event, 1024),
		quit: make(chan struct{}),
	}
	if err := w.addWatches(); err != nil {
		fsw.Close()
		return nil, err
	}
	return w, nil
}

func (w *CodexWatcher) Events() <-chan Event { return w.events }

func (w *CodexWatcher) Start() {
	w.wg.Add(2)
	go w.watchLoop()
	go w.scanLoop()
}

func (w *CodexWatcher) Stop() {
	close(w.quit)
	w.fsw.Close()
	w.wg.Wait()
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
		sessionID, project := extractCodexMeta(path, meta.CWD)
		entry = &codexDecoderEntry{dec: dec, proj: project, sessID: sessionID}
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
			Project: entry.proj,
			File: path,
			Data: event.Raw,
			Timestamp: time.Now(),
		}
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
		if ok {
			if jsonl.IsCodexUserSession(meta) {
				return meta, true
			}
			return jsonl.CodexSessionMeta{}, false
		}
	}
	return jsonl.CodexSessionMeta{}, false
}

func extractCodexMeta(path, cwd string) (sessionID, project string) {
	base := strings.TrimSuffix(filepath.Base(path), ".jsonl")
	if strings.HasPrefix(base, "rollout-") {
		parts := strings.Split(base, "-")
		if len(parts) >= 4 {
			sessionID = strings.Join(parts[len(parts)-5:], "-")
		}
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
```

- [ ] **Step 4: Fix `extractCodexMeta` if the test reveals bad UUID extraction**

Run:

```bash
go test ./internal/watcher -run TestExtractCodexMeta -v
```

Expected: PASS. If it fails, replace the ID extraction block in `extractCodexMeta` with:

```go
prefixParts := strings.Split(base, "T")
if len(prefixParts) == 2 {
	rest := prefixParts[1]
	if i := strings.Index(rest, "-"); i >= 0 && i+1 < len(rest) {
		sessionID = rest[i+1:]
	}
}
```

- [ ] **Step 5: Add hub subscription for Codex watcher**

In `internal/hub/hub.go`, add:

```go
// SubscribeCodexWatcher reads events from the Codex watcher and broadcasts them.
func (h *Hub) SubscribeCodexWatcher(w *watcher.CodexWatcher) {
	for ev := range w.Events() {
		msg := WSMessage{
			Type:      "event",
			SessionID: ev.SessionID,
			Project:   ev.Project,
			Data:      ev.Data,
			Time:      ev.Timestamp,
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[hub] marshal error: %v", err)
			continue
		}
		h.broadcast <- data
	}
}
```

- [ ] **Step 6: Run watcher and hub package tests/build**

Run:

```bash
go test ./internal/watcher ./internal/hub -v
```

Expected: PASS.

- [ ] **Step 7: Commit Task 3**

```bash
git add internal/watcher/codex_watcher.go internal/watcher/codex_watcher_test.go internal/hub/hub.go
git commit -m "feat(watcher): stream codex session events"
```

---

### Task 4: Server Integration

**Files:**
- Modify: `cmd/server/main.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: Add Codex server fields and constructor parameter**

In `internal/server/server.go`, add a field to `Server`:

```go
codexWatcher      *watcher.CodexWatcher
codexSessionsDir  string
```

Change constructor signature:

```go
func New(sessionsDir, claudeProjectsDir, codexSessionsDir, allowedRootsCSV string) (*Server, error) {
```

Initialize:

```go
s := &Server{
	hub:               h,
	watcher:           w,
	rpcMgr:            newRPCManager(),
	sessionsDir:       sessionsDir,
	claudeProjectsDir: claudeProjectsDir,
	codexSessionsDir:  codexSessionsDir,
	llmClient:         llm.NewLMStudioClient(),
}
```

- [ ] **Step 2: Add optional Codex watcher startup in `server.New`**

After Claude watcher initialization in `internal/server/server.go`, add:

```go
if codexSessionsDir != "" {
	if info, err := os.Stat(codexSessionsDir); err == nil && info.IsDir() {
		cw, err := watcher.NewCodexWatcher(codexSessionsDir)
		if err != nil {
			log.Printf("[server] warning: could not create Codex watcher: %v", err)
		} else {
			s.codexWatcher = cw
			log.Printf("[server] Codex watcher enabled: %s", codexSessionsDir)
		}
	} else {
		log.Printf("[server] Codex sessions dir not found, skipping: %s", codexSessionsDir)
	}
}
```

- [ ] **Step 3: Wire Codex watcher start/stop**

In `Server.Start`, after Claude watcher startup, add:

```go
if s.codexWatcher != nil {
	go s.hub.SubscribeCodexWatcher(s.codexWatcher)
	s.codexWatcher.Start()
}
```

In `Server.Stop`, after Claude watcher stop, add:

```go
if s.codexWatcher != nil {
	s.codexWatcher.Stop()
}
```

- [ ] **Step 4: Add `-codex-sessions` flag**

In `cmd/server/main.go`, add:

```go
codexSessionsDir := flag.String("codex-sessions", "", "Path to ~/.codex/sessions directory")
```

After Claude default directory setup, add:

```go
if *codexSessionsDir == "" {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("cannot determine home directory: %v", err)
	}
	*codexSessionsDir = filepath.Join(home, ".codex", "sessions")
}
```

Change server construction to:

```go
srv, err := server.New(*sessionsDir, *claudeProjectsDir, *codexSessionsDir, *allowedRoots)
```

- [ ] **Step 5: Add Codex scan to `listSessions`**

In `internal/server/server.go`, add this block in `listSessions()` after Claude scanning and before final sort:

```go
if s.codexSessionsDir != "" {
	filepath.WalkDir(s.codexSessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		meta, ok := readCodexSessionInfo(path)
		if !ok {
			return nil
		}
		info := SessionInfo{
			ID: meta.ID,
			File: path,
			Agent: "codex",
			CWD: meta.CWD,
			Project: filepath.Base(meta.CWD),
			Model: meta.Model,
		}
		info.LineCount, _, info.Model, info.InputTokens, info.OutputTokens, info.TotalTokens, info.TotalCost, info.ContextWindow = aggregateSessionData(path, "codex")
		if info.CWD == "" {
			info.CWD = meta.CWD
		}
		if info.Project == "." || info.Project == "" {
			info.Project = filepath.Base(filepath.Dir(path))
		}
		info.FirstUserMessage = getFirstUserMessage(path, "codex")
		info.LastMessageTime = getLastMessageTime(path)
		if fi, err := d.Info(); err == nil {
			info.Timestamp = fi.ModTime()
		}
		sessions = append(sessions, info)
		return nil
	})
}
```

Add helper near other server helpers:

```go
func readCodexSessionInfo(path string) (jsonl.CodexSessionMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return jsonl.CodexSessionMeta{}, false
	}
	defer f.Close()
	scanner := NewLineScanner(f, make([]byte, 32*1024))
	for scanner.Scan() {
		meta, ok := jsonl.ParseCodexSessionMeta(scanner.Bytes())
		if ok {
			if jsonl.IsCodexUserSession(meta) {
				return meta, true
			}
			return jsonl.CodexSessionMeta{}, false
		}
	}
	return jsonl.CodexSessionMeta{}, false
}
```

- [ ] **Step 6: Extend aggregate and first-message helpers for Codex**

In `aggregateSessionData`, add an `agent == "codex"` branch:

```go
if agent == "codex" {
	if cwd == "" {
		if meta, ok := jsonl.ParseCodexSessionMeta(line); ok {
			cwd = meta.CWD
			model = meta.Model
		}
	}
	if model == "" {
		var ctx struct {
			Type string `json:"type"`
			Payload struct {
				Model string `json:"model"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &ctx) == nil && ctx.Type == "turn_context" && ctx.Payload.Model != "" {
			model = ctx.Payload.Model
		}
	}
	continue
}
```

In `getFirstUserMessage`, add an `agent == "codex"` branch before the Claude branch:

```go
} else if agent == "codex" {
	var env jsonl.CodexEnvelope
	if json.Unmarshal(line, &env) == nil && env.Type == "response_item" {
		var msg jsonl.CodexMessage
		if json.Unmarshal(env.Payload, &msg) == nil && msg.Type == "message" && msg.Role == "user" {
			for _, block := range msg.Content {
				if (block.Type == "input_text" || block.Type == "text") && block.Text != "" {
					return truncateMessage(block.Text)
				}
			}
		}
	}
```

- [ ] **Step 7: Add Codex replay in `onSubscribe`**

In `onSubscribe`, add a Codex branch before Claude:

```go
if sessionAgent == "codex" {
	dec, err := jsonl.NewCodexDecoder(sessionFile, 0)
	if err != nil {
		log.Printf("[server] open codex decoder: %v", err)
		return
	}
	defer dec.Close()
	for {
		event, err := dec.Next()
		if err != nil {
			break
		}
		if event == nil {
			continue
		}
		msg := hub.WSMessage{
			Type: "event",
			SessionID: sessionID,
			Data: event.Raw,
			Time: time.Now(),
		}
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		select {
		case <-client.Closed():
			return
		default:
		}
		select {
		case client.Send() <- data:
		default:
		}
	}
} else if sessionAgent == "claude" {
```

- [ ] **Step 8: Run server build**

Run:

```bash
go test ./internal/server ./cmd/server -run '^$' -count=0
```

Expected: PASS compile check.

- [ ] **Step 9: Commit Task 4**

```bash
git add cmd/server/main.go internal/server/server.go
git commit -m "feat(server): list and replay codex sessions"
```

---

### Task 5: Frontend Read-Only Codex Capability

**Files:**
- Modify: `frontend/src/lib/utils/sessionCapabilities.js`
- Modify: `frontend/src/lib/utils/sessionCapabilities.test.mjs`
- Modify: `frontend/src/lib/actions/rpc.js`
- Modify: `frontend/src/lib/components/ChatArea.svelte`

- [ ] **Step 1: Add failing capability test for Codex**

Append to `frontend/src/lib/utils/sessionCapabilities.test.mjs`:

```js
test('codex sessions do not support RPC chat', () => {
  assert.equal(sessionSupportsRPC({ id: 'x1', agent: 'codex' }), false);
});
```

- [ ] **Step 2: Run frontend helper tests**

Run:

```bash
node --test frontend/src/lib/utils/sessionCapabilities.test.mjs
```

Expected: PASS already if `sessionSupportsRPC` only returns true for `pi`. Keep the test because it locks Codex behavior.

- [ ] **Step 3: Add read-only reason helper**

Modify `frontend/src/lib/utils/sessionCapabilities.js`:

```js
export function findSession(sessions, sessionId) {
  return (sessions || []).find((session) => session.id === sessionId) || null;
}

export function sessionSupportsRPC(session) {
  return (session?.agent || 'pi') === 'pi';
}

export function readOnlySessionLabel(session) {
  const agent = session?.agent || 'pi';
  if (agent === 'claude') return 'Claude Code sessions are read-only here';
  if (agent === 'codex') return 'Codex sessions are read-only here';
  return 'This session is read-only here';
}
```

- [ ] **Step 4: Update capability tests for read-only labels**

Append to `frontend/src/lib/utils/sessionCapabilities.test.mjs`:

```js
import { readOnlySessionLabel } from './sessionCapabilities.js';

test('read-only label names codex sessions', () => {
  assert.equal(readOnlySessionLabel({ agent: 'codex' }), 'Codex sessions are read-only here');
});
```

If the file already imports named functions in one import statement, merge `readOnlySessionLabel` into that existing import instead of adding a second import:

```js
import { findSession, readOnlySessionLabel, sessionSupportsRPC } from './sessionCapabilities.js';
```

- [ ] **Step 5: Use generic read-only label in RPC action**

In `frontend/src/lib/actions/rpc.js`, change the import:

```js
import { findSession, readOnlySessionLabel, sessionSupportsRPC } from '$lib/utils/sessionCapabilities.js';
```

Replace the hard-coded Claude message:

```js
addSystemMessage(readOnlySessionLabel(session));
```

- [ ] **Step 6: Use generic read-only label in chat placeholder**

In `frontend/src/lib/components/ChatArea.svelte`, update the utility import to include `readOnlySessionLabel`, then replace the hard-coded placeholder expression with:

```svelte
placeholder={$activeSession ? (activeSessionCanChat ? (isDragOver ? 'Drop image here...' : 'Message the agent...') : readOnlySessionLabel(activeSessionInfo)) : 'Select a session to begin...'}
```

- [ ] **Step 7: Run frontend helper tests**

Run:

```bash
node --test frontend/src/lib/utils/sessionCapabilities.test.mjs
```

Expected: PASS.

- [ ] **Step 8: Commit Task 5**

```bash
git add frontend/src/lib/utils/sessionCapabilities.js frontend/src/lib/utils/sessionCapabilities.test.mjs frontend/src/lib/actions/rpc.js frontend/src/lib/components/ChatArea.svelte
git commit -m "feat(frontend): mark codex sessions read-only"
```

---

### Task 6: End-To-End Verification

**Files:**
- Verify only unless failures require fixes.

- [ ] **Step 1: Run Go tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend unit tests**

Run:

```bash
node --test frontend/src/lib/utils/*.test.mjs
```

Expected: PASS.

- [ ] **Step 3: Build frontend**

Run:

```bash
npm --prefix frontend run build
```

Expected: PASS with Vite build output and generated `frontend/dist`.

- [ ] **Step 4: Build server**

Run:

```bash
GOCACHE=/tmp/go-cache go build -buildvcs=false -o bin/server ./cmd/server/
```

Expected: PASS and `bin/server` updated.

- [ ] **Step 5: Smoke test session listing manually**

Run server:

```bash
./bin/server -addr :8081
```

In another terminal, run:

```bash
curl -s 'http://localhost:8081/api/sessions?page=1' | rg '"agent":"codex"|"agent":"claude"|"agent":"pi"'
```

Expected: output includes Codex sessions when `~/.codex/sessions` has user-facing session files. It does not include `codex-auto-review`, guardian, or `thread_source":"subagent"` sessions.

- [ ] **Step 6: Commit verification fixes only if needed**

If verification required code changes, inspect the changed files:

```bash
git status --short
```

Stage only files that were changed to fix verification failures, then commit:

```bash
git add internal/jsonl/codex_types.go internal/jsonl/codex_decoder.go internal/jsonl/codex_decoder_test.go internal/watcher/codex_watcher.go internal/watcher/codex_watcher_test.go internal/hub/hub.go internal/server/server.go cmd/server/main.go frontend/src/lib/utils/sessionCapabilities.js frontend/src/lib/utils/sessionCapabilities.test.mjs frontend/src/lib/actions/rpc.js frontend/src/lib/components/ChatArea.svelte
git commit -m "fix: complete codex session verification"
```

If verification required no code changes, do not create an empty commit.
