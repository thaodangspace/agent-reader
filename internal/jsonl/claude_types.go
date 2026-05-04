// Package jsonl defines Go structs for both pi-agent and Claude Code JSONL event formats.
package jsonl

import "encoding/json"

// --- Claude Code event types ---

// ClaudeEvent is the top-level JSONL line for Claude Code sessions.
// Claude Code uses top-level type values like "user", "assistant", "attachment", etc.
// instead of wrapping everything in a "message" type.
type ClaudeEvent struct {
	Type       string          `json:"type"`
	UUID       string          `json:"uuid"`
	ParentUUID *string         `json:"parentUuid"`
	Timestamp  string          `json:"timestamp"`
	CWD        string          `json:"cwd"`
	SessionID  string          `json:"sessionId"`
	Message    *ClaudeMessage  `json:"message,omitempty"`
	Raw        json.RawMessage `json:"-"`
}

// ClaudeMessage is the "message" field inside assistant/user events.
type ClaudeMessage struct {
	Role       string               `json:"role"`
	Content    []ClaudeContentBlock `json:"content"`
	Model      string               `json:"model,omitempty"`
	Usage      *ClaudeUsage         `json:"usage,omitempty"`
	StopReason string               `json:"stop_reason,omitempty"`
}

// ClaudeContentBlock is a single block inside a message's content array.
type ClaudeContentBlock struct {
	Type      string          `json:"type"` // "text" | "thinking" | "tool_use" | "tool_result"
	Text      string          `json:"text,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
}

// ClaudeUsage tracks token usage for assistant messages (snake_case).
type ClaudeUsage struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadTokens     int64 `json:"cache_read_input_tokens"`
}
