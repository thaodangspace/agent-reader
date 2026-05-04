package jsonl

import (
	"encoding/json"
	"io"
	"os"
	"testing"
)

func newTestDecoder() *ClaudeDecoder {
	return &ClaudeDecoder{toolNames: map[string]string{}}
}

func TestNormalizeAssistant_Basic(t *testing.T) {
	input := `{
		"type": "assistant",
		"uuid": "abc-123",
		"parentUuid": "parent-456",
		"timestamp": "2026-05-03T10:00:00Z",
		"cwd": "/Users/dt/code/test",
		"sessionId": "sess-001",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "text", "text": "Hello world"}
			],
			"model": "claude-sonnet-4-20250514",
			"usage": {
				"input_tokens": 100,
				"output_tokens": 50,
				"cache_creation_input_tokens": 0,
				"cache_read_input_tokens": 0
			},
			"stop_reason": "end_turn"
		}
	}`

	out, drop := newTestDecoder().normalizeAssistant(input)
	if drop {
		t.Fatal("expected not to drop")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["type"] != "message" {
		t.Errorf("expected type=message, got %v", result["type"])
	}
	if result["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %v", result["id"])
	}
	if result["parentId"] != "parent-456" {
		t.Errorf("expected parentId=parent-456, got %v", result["parentId"])
	}

	msg := result["message"].(map[string]interface{})
	if msg["role"] != "assistant" {
		t.Errorf("expected role=assistant, got %v", msg["role"])
	}

	content := msg["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}
	block := content[0].(map[string]interface{})
	if block["type"] != "text" {
		t.Errorf("expected content type=text, got %v", block["type"])
	}
	if block["text"] != "Hello world" {
		t.Errorf("expected text='Hello world', got %v", block["text"])
	}
}

func TestNormalizeAssistant_ToolUse(t *testing.T) {
	input := `{
		"type": "assistant",
		"uuid": "abc-123",
		"timestamp": "2026-05-03T10:00:00Z",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "call_001", "name": "Read", "input": {"file_path": "/test/file.go"}}
			]
		}
	}`

	out, drop := newTestDecoder().normalizeAssistant(input)
	if drop {
		t.Fatal("expected not to drop")
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(out), &result)

	msg := result["message"].(map[string]interface{})
	content := msg["content"].([]interface{})
	block := content[0].(map[string]interface{})

	if block["type"] != "toolCall" {
		t.Errorf("expected type=toolCall, got %v", block["type"])
	}
	if block["name"] != "Read" {
		t.Errorf("expected name=Read, got %v", block["name"])
	}
	if _, ok := block["arguments"]; !ok {
		t.Error("expected 'arguments' field (not 'input')")
	}
}

func TestNormalizeUser_ToolResult(t *testing.T) {
	input := `{
		"type": "user",
		"uuid": "user-evt-001",
		"parentUuid": "abc-123",
		"timestamp": "2026-05-03T10:00:01Z",
		"message": {
			"role": "user",
			"content": [
				{"type": "tool_result", "content": "file contents here", "tool_use_id": "call_001"}
			]
		}
	}`

	out, drop := newTestDecoder().normalizeUser(input)
	if drop {
		t.Fatal("expected not to drop")
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(out), &result)

	if result["type"] != "message" {
		t.Errorf("expected type=message, got %v", result["type"])
	}

	msg := result["message"].(map[string]interface{})
	if msg["role"] != "toolResult" {
		t.Errorf("expected role=toolResult, got %v", msg["role"])
	}
	if msg["toolCallId"] != "call_001" {
		t.Errorf("expected toolCallId=call_001, got %v", msg["toolCallId"])
	}
	if msg["content"] != "file contents here" {
		t.Errorf("expected content='file contents here', got %v", msg["content"])
	}
}

func TestNormalizeUser_TextMessage(t *testing.T) {
	// Claude Code user messages have content as a plain string (not array)
	input := `{
		"type": "user",
		"uuid": "user-evt-002",
		"timestamp": "2026-05-03T10:00:02Z",
		"message": {
			"role": "user",
			"content": "Hello from user"
		}
	}`

	out, drop := newTestDecoder().normalizeUser(input)
	if drop {
		t.Fatal("expected not to drop")
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(out), &result)

	msg := result["message"].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("expected role=user, got %v", msg["role"])
	}
	if msg["content"] != "Hello from user" {
		t.Errorf("expected content='Hello from user', got %v", msg["content"])
	}
}

func TestNormalizeUser_RealWorldFormat(t *testing.T) {
	// Real Claude Code user message from actual session file
	input := `{"parentUuid":"6428091f-5624-4bc7-ad1c-008744e122a4","isSidechain":false,"promptId":"8dab7631-0002-44ed-9d08-3356ff8a2e9e","type":"user","message":{"role":"user","content":"remove pnpm config, for now '/Users/tdxng/Library/pnpm/.tools/pnpm/10.33.2_tmp_90621_0' -> old user, we have new user"},"uuid":"9f2c1f35-9d0f-4828-9dc1-f6c69f2097a7","timestamp":"2026-04-26T15:08:25.645Z","permissionMode":"bypassPermissions","userType":"external","entrypoint":"cli","cwd":"/Users/dt/code/pi-vibe","sessionId":"51712e20-60df-4d1c-b2bb-36df8fc51fab","version":"2.1.118","gitBranch":"HEAD"}`

	out, drop := newTestDecoder().normalizeUser(input)
	if drop {
		t.Fatal("expected not to drop real-world user message")
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(out), &result)

	if result["id"] != "9f2c1f35-9d0f-4828-9dc1-f6c69f2097a7" {
		t.Errorf("expected correct id, got %v", result["id"])
	}

	msg := result["message"].(map[string]interface{})
	if msg["role"] != "user" {
		t.Errorf("expected role=user, got %v", msg["role"])
	}
	if msg["content"] != "remove pnpm config, for now '/Users/tdxng/Library/pnpm/.tools/pnpm/10.33.2_tmp_90621_0' -> old user, we have new user" {
		t.Errorf("expected correct content, got %v", msg["content"])
	}
}

func TestNormalizeClaudeLine_DropTypes(t *testing.T) {
	dropTypes := []string{
		`{"type":"permission-mode","permissionMode":"bypassPermissions"}`,
		`{"type":"attachment","attachment":{"type":"hook_success"}}`,
		`{"type":"file-history-snapshot","messageId":"abc"}`,
		`{"type":"last-prompt","lastPrompt":"test","sessionId":"abc"}`,
	}

	for _, input := range dropTypes {
		_, drop := newTestDecoder().normalizeClaudeLine(input)
		if !drop {
			t.Errorf("expected to drop type from: %s", input[:50])
		}
	}
}

func TestNormalizeClaudeLine_ForwardUnknown(t *testing.T) {
	input := `{"type":"ai-title","aiTitle":"Test session"}`
	out, drop := newTestDecoder().normalizeClaudeLine(input)
	if drop {
		t.Fatal("expected not to drop ai-title")
	}
	if out != input {
		t.Errorf("expected passthrough, got %s", out)
	}
}

func TestClaudeDecoder_Next(t *testing.T) {
	// Claude Code JSONL files use single-line JSON
	content := `{"type":"permission-mode","permissionMode":"bypassPermissions","sessionId":"test-sess"}` + "\n" +
		`{"type":"user","uuid":"user-001","timestamp":"2026-05-03T10:00:00Z","cwd":"/Users/dt/code/test","message":{"role":"user","content":[{"type":"text","text":"hello"}]}}` + "\n" +
		`{"type":"assistant","uuid":"asst-001","parentUuid":"user-001","timestamp":"2026-05-03T10:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"hi back"}]}}` + "\n"
	tmpfile := t.TempDir() + "/test.jsonl"
	if err := os.WriteFile(tmpfile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	dec, err := NewClaudeDecoder(tmpfile, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer dec.Close()

	// First line is permission-mode → dropped
	ev, err := dec.Next()
	if err != nil || ev != nil {
		t.Fatalf("expected nil (dropped), got ev=%v err=%v", ev, err)
	}

	// Second line is user → normalized to message
	ev, err = dec.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "message" {
		t.Errorf("expected type=message, got %s", ev.Type)
	}

	// Third line is assistant → normalized to message
	ev, err = dec.Next()
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "message" {
		t.Errorf("expected type=message, got %s", ev.Type)
	}

	// EOF
	ev, err = dec.Next()
	if err != io.EOF {
		t.Fatalf("expected EOF, got err=%v", err)
	}
}

func TestNormalizeAssistant_RecordsToolNames(t *testing.T) {
	dec := &ClaudeDecoder{toolNames: map[string]string{}}

	input := `{
		"type": "assistant",
		"uuid": "asst-001",
		"timestamp": "2026-05-03T10:00:00Z",
		"message": {
			"role": "assistant",
			"content": [
				{"type": "tool_use", "id": "toolu_abc", "name": "Read", "input": {"file_path": "/x"}},
				{"type": "tool_use", "id": "toolu_def", "name": "Bash", "input": {"command": "ls"}}
			]
		}
	}`

	if _, drop := dec.normalizeAssistant(input); drop {
		t.Fatal("expected not to drop")
	}

	if got := dec.toolNames["toolu_abc"]; got != "Read" {
		t.Errorf("toolu_abc: expected Read, got %q", got)
	}
	if got := dec.toolNames["toolu_def"]; got != "Bash" {
		t.Errorf("toolu_def: expected Bash, got %q", got)
	}
}

func TestNormalizeUser_DropsIsMeta(t *testing.T) {
	// Skill-injected content arrives as a user message with isMeta:true.
	// We drop it so the skill body doesn't render as user input.
	input := `{
		"type": "user",
		"uuid": "meta-001",
		"timestamp": "2026-05-04T14:43:10.386Z",
		"isMeta": true,
		"message": {
			"role": "user",
			"content": [
				{"type": "text", "text": "Base directory for this skill: /tmp/skills/foo\n# Foo Skill\n..."}
			]
		}
	}`

	_, drop := newTestDecoder().normalizeUser(input)
	if !drop {
		t.Fatal("expected isMeta:true user message to be dropped")
	}
}
