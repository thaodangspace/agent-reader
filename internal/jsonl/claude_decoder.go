package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ClaudeDecoder reads a Claude Code JSONL file and emits normalized pi-agent format events.
type ClaudeDecoder struct {
	path      string
	offset    int64
	file      *os.File
	reader    *bufio.Reader
	toolNames map[string]string // tool_use_id -> tool name, populated from assistant events
}

// NewClaudeDecoder opens a Claude Code JSONL file for reading.
func NewClaudeDecoder(path string, offset int64) (*ClaudeDecoder, error) {
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
	return &ClaudeDecoder{
		path:      path,
		offset:    offset,
		file:      f,
		reader:    bufio.NewReader(f),
		toolNames: make(map[string]string),
	}, nil
}

// Offset returns the current byte offset.
func (d *ClaudeDecoder) Offset() int64 { return d.offset }

// Path returns the file path.
func (d *ClaudeDecoder) Path() string { return d.path }

// Next reads the next JSONL line and returns a normalized pi-agent format event.
// Returns (nil, nil) for dropped/blank lines, (nil, io.EOF) at end of file.
func (d *ClaudeDecoder) Next() (*Event, error) {
	for {
		line, err := d.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					return d.processLine(line)
				}
				return nil, io.EOF
			}
			return nil, fmt.Errorf("read %s: %w", d.path, err)
		}

		d.offset += int64(len(line))

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		return d.processLine(trimmed)
	}
}

func (d *ClaudeDecoder) processLine(raw string) (*Event, error) {
	normalized, drop := d.normalizeClaudeLine(raw)
	if drop {
		return nil, nil
	}

	var wrapper struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal([]byte(normalized), &wrapper); err != nil {
		return nil, nil // skip malformed
	}

	ev := &Event{
		Type: wrapper.Type,
		ID:   wrapper.ID,
		Raw:  json.RawMessage(normalized),
	}

	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(normalized), &rawMap); err == nil {
		if v, ok := rawMap["parentId"]; ok && string(v) != "null" {
			var s string
			json.Unmarshal(v, &s)
			ev.ParentID = &s
		}
		if v, ok := rawMap["timestamp"]; ok {
			json.Unmarshal(v, &ev.Timestamp)
		}
	}

	return ev, nil
}

// Close releases the file handle.
func (d *ClaudeDecoder) Close() error { return d.file.Close() }

// normalizeClaudeLine converts a Claude Code JSONL line to pi-agent format.
// Returns (normalizedJSON, drop) where drop=true means the line should be ignored.
func (d *ClaudeDecoder) normalizeClaudeLine(raw string) (string, bool) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &base); err != nil {
		return "", true
	}

	switch base.Type {
	case "permission-mode", "attachment", "file-history-snapshot", "last-prompt":
		return "", true // drop these

	case "assistant":
		return d.normalizeAssistant(raw)

	case "user":
		return d.normalizeUser(raw)

	case "ai-title", "system":
		return raw, false // forward as-is

	default:
		return raw, false // forward unknown types
	}
}

// normalizeAssistant converts a Claude "assistant" event to pi "message" format.
func (d *ClaudeDecoder) normalizeAssistant(raw string) (string, bool) {
	var evt ClaudeEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		return "", true
	}
	if evt.Message == nil {
		return "", true
	}

	// Build normalized content blocks from the content array
	var normalizedContent []json.RawMessage
	for _, block := range evt.Message.Content.AsBlocks {
		switch block.Type {
		case "text":
			b, _ := json.Marshal(map[string]string{
				"type": "text",
				"text": block.Text,
			})
			normalizedContent = append(normalizedContent, b)

		case "thinking":
			b, _ := json.Marshal(map[string]string{
				"type":     "thinking",
				"thinking": block.Thinking,
			})
			normalizedContent = append(normalizedContent, b)

		case "tool_use":
			if d.toolNames != nil && block.ID != "" && block.Name != "" {
				d.toolNames[block.ID] = block.Name
			}
			// tool_use → toolCall, input → arguments
			b, _ := json.Marshal(map[string]interface{}{
				"type":      "toolCall",
				"id":        block.ID,
				"name":      block.Name,
				"arguments": block.Input,
			})
			normalizedContent = append(normalizedContent, b)
		}
	}

	// Build the normalized message
	msg := map[string]interface{}{
		"role":    "assistant",
		"content": normalizedContent,
	}
	if evt.Message.Model != "" {
		msg["model"] = evt.Message.Model
	}
	if evt.Message.Usage != nil {
		msg["usage"] = map[string]interface{}{
			"input":       evt.Message.Usage.InputTokens,
			"output":      evt.Message.Usage.OutputTokens,
			"cacheRead":   evt.Message.Usage.CacheReadTokens,
			"cacheWrite":  evt.Message.Usage.CacheCreationTokens,
			"totalTokens": evt.Message.Usage.InputTokens + evt.Message.Usage.OutputTokens + evt.Message.Usage.CacheCreationTokens + evt.Message.Usage.CacheReadTokens,
			"cost":        map[string]float64{"total": 0},
		}
	}
	if evt.Message.StopReason != "" {
		msg["stopReason"] = evt.Message.StopReason
	}

	result := map[string]interface{}{
		"type":      "message",
		"id":        evt.UUID,
		"timestamp": evt.Timestamp,
		"message":   msg,
	}
	if evt.ParentUUID != nil {
		result["parentId"] = *evt.ParentUUID
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", true
	}
	return string(out), false
}

// normalizeUser converts a Claude "user" event to pi "message" format.
// Tool results are extracted into separate toolResult messages.
// User messages have content as a plain string OR as an array of blocks (for tool_result).
func (d *ClaudeDecoder) normalizeUser(raw string) (string, bool) {
	var evt ClaudeEvent
	if err := json.Unmarshal([]byte(raw), &evt); err != nil {
		return "", true
	}
	if evt.Message == nil {
		return "", true
	}
	if evt.IsMeta {
		return "", true
	}

	// If content is a plain string, it's a regular user text message
	if evt.Message.Content.IsString {
		content := evt.Message.Content.AsString
		if content == "" {
			return "", true
		}
		result := map[string]interface{}{
			"type":      "message",
			"id":        evt.UUID,
			"timestamp": evt.Timestamp,
			"message": map[string]interface{}{
				"role":    "user",
				"content": content,
			},
		}
		if evt.ParentUUID != nil {
			result["parentId"] = *evt.ParentUUID
		}
		out, err := json.Marshal(result)
		if err != nil {
			return "", true
		}
		return string(out), false
	}

	// Content is an array of blocks — check for tool_result
	for _, block := range evt.Message.Content.AsBlocks {
		if block.Type == "tool_result" {
			result := map[string]interface{}{
				"type":      "message",
				"id":        evt.UUID,
				"timestamp": evt.Timestamp,
				"message": map[string]interface{}{
					"role":       "toolResult",
					"toolCallId": block.ToolUseID,
					"content":    block.Content,
					"isError":    false,
				},
			}
			if evt.ParentUUID != nil {
				result["parentId"] = *evt.ParentUUID
			}
			out, err := json.Marshal(result)
			if err != nil {
				return "", true
			}
			return string(out), false
		}
	}

	// Non-tool_result block content — extract text blocks
	var texts []string
	for _, block := range evt.Message.Content.AsBlocks {
		if block.Type == "text" {
			texts = append(texts, block.Text)
		}
	}

	content := strings.Join(texts, "\n")
	if content == "" {
		return "", true
	}

	result := map[string]interface{}{
		"type":      "message",
		"id":        evt.UUID,
		"timestamp": evt.Timestamp,
		"message": map[string]interface{}{
			"role":    "user",
			"content": content,
		},
	}
	if evt.ParentUUID != nil {
		result["parentId"] = *evt.ParentUUID
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", true
	}
	return string(out), false
}
