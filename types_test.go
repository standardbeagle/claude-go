package claude

import (
	"encoding/json"
	"testing"
)

func TestParseTextBlock(t *testing.T) {
	raw := json.RawMessage(`{"type": "text", "text": "Hello, world!"}`)
	block, err := ParseContentBlock(raw)
	if err != nil {
		t.Fatalf("ParseContentBlock failed: %v", err)
	}

	textBlock, ok := block.(TextBlock)
	if !ok {
		t.Fatal("Expected TextBlock")
	}

	if textBlock.Text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", textBlock.Text)
	}
}

func TestParseToolUseBlock(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "tool_use",
		"id": "tool-123",
		"name": "read_file",
		"input": {"path": "/tmp/test.txt"}
	}`)

	block, err := ParseContentBlock(raw)
	if err != nil {
		t.Fatalf("ParseContentBlock failed: %v", err)
	}

	toolBlock, ok := block.(ToolUseBlock)
	if !ok {
		t.Fatal("Expected ToolUseBlock")
	}

	if toolBlock.ID != "tool-123" {
		t.Errorf("Expected ID 'tool-123', got '%s'", toolBlock.ID)
	}
	if toolBlock.Name != "read_file" {
		t.Errorf("Expected name 'read_file', got '%s'", toolBlock.Name)
	}
	if toolBlock.Input["path"] != "/tmp/test.txt" {
		t.Errorf("Expected path '/tmp/test.txt', got '%v'", toolBlock.Input["path"])
	}
}

func TestParseToolResultBlock(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "tool_result",
		"tool_use_id": "tool-123",
		"content": "File contents here",
		"is_error": false
	}`)

	block, err := ParseContentBlock(raw)
	if err != nil {
		t.Fatalf("ParseContentBlock failed: %v", err)
	}

	resultBlock, ok := block.(ToolResultBlock)
	if !ok {
		t.Fatal("Expected ToolResultBlock")
	}

	if resultBlock.ToolUseID != "tool-123" {
		t.Errorf("Expected tool_use_id 'tool-123', got '%s'", resultBlock.ToolUseID)
	}
	if resultBlock.Content != "File contents here" {
		t.Errorf("Expected content 'File contents here', got '%s'", resultBlock.Content)
	}
	if resultBlock.IsError {
		t.Error("Expected is_error to be false")
	}
}

func TestParseThinkingBlock(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "thinking",
		"thinking": "Let me think about this...",
		"signature": "sig123"
	}`)

	block, err := ParseContentBlock(raw)
	if err != nil {
		t.Fatalf("ParseContentBlock failed: %v", err)
	}

	thinkingBlock, ok := block.(ThinkingBlock)
	if !ok {
		t.Fatal("Expected ThinkingBlock")
	}

	if thinkingBlock.Thinking != "Let me think about this..." {
		t.Errorf("Expected thinking text, got '%s'", thinkingBlock.Thinking)
	}
	if thinkingBlock.Signature != "sig123" {
		t.Errorf("Expected signature 'sig123', got '%s'", thinkingBlock.Signature)
	}
}

func TestParseContentBlocks(t *testing.T) {
	raw := json.RawMessage(`[
		{"type": "text", "text": "Here is the file:"},
		{"type": "tool_use", "id": "t1", "name": "read", "input": {}}
	]`)

	blocks, err := ParseContentBlocks(raw)
	if err != nil {
		t.Fatalf("ParseContentBlocks failed: %v", err)
	}

	if len(blocks) != 2 {
		t.Fatalf("Expected 2 blocks, got %d", len(blocks))
	}

	if _, ok := blocks[0].(TextBlock); !ok {
		t.Error("Expected first block to be TextBlock")
	}
	if _, ok := blocks[1].(ToolUseBlock); !ok {
		t.Error("Expected second block to be ToolUseBlock")
	}
}

func TestParseContentBlocksAsString(t *testing.T) {
	raw := json.RawMessage(`"Just a simple string"`)

	blocks, err := ParseContentBlocks(raw)
	if err != nil {
		t.Fatalf("ParseContentBlocks failed: %v", err)
	}

	if len(blocks) != 1 {
		t.Fatalf("Expected 1 block, got %d", len(blocks))
	}

	textBlock, ok := blocks[0].(TextBlock)
	if !ok {
		t.Fatal("Expected TextBlock")
	}
	if textBlock.Text != "Just a simple string" {
		t.Errorf("Expected 'Just a simple string', got '%s'", textBlock.Text)
	}
}

func TestParseAssistantMessage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "assistant",
		"uuid": "msg-123",
		"model": "claude-3-opus",
		"content": [
			{"type": "text", "text": "Hello!"}
		]
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	assistantMsg, ok := msg.(AssistantMessage)
	if !ok {
		t.Fatal("Expected AssistantMessage")
	}

	if assistantMsg.UUID != "msg-123" {
		t.Errorf("Expected UUID 'msg-123', got '%s'", assistantMsg.UUID)
	}
	if assistantMsg.Model != "claude-3-opus" {
		t.Errorf("Expected model 'claude-3-opus', got '%s'", assistantMsg.Model)
	}
	if len(assistantMsg.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(assistantMsg.Content))
	}
}

func TestParseAssistantMessage_Nested(t *testing.T) {
	// Claude Code stream-json format: content is nested under message
	raw := json.RawMessage(`{
		"type": "assistant",
		"uuid": "msg-nested",
		"message": {
			"content": [
				{"type": "text", "text": "Hello from nested!"},
				{"type": "tool_use", "id": "t1", "name": "Read", "input": {"file": "test.go"}}
			],
			"model": "claude-sonnet-4-20250514"
		}
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	assistantMsg, ok := msg.(AssistantMessage)
	if !ok {
		t.Fatal("Expected AssistantMessage")
	}

	if assistantMsg.UUID != "msg-nested" {
		t.Errorf("Expected UUID 'msg-nested', got '%s'", assistantMsg.UUID)
	}
	if assistantMsg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected model 'claude-sonnet-4-20250514', got '%s'", assistantMsg.Model)
	}
	if len(assistantMsg.Content) != 2 {
		t.Fatalf("Expected 2 content blocks, got %d", len(assistantMsg.Content))
	}

	textBlock, ok := assistantMsg.Content[0].(TextBlock)
	if !ok {
		t.Fatalf("Expected TextBlock, got %T", assistantMsg.Content[0])
	}
	if textBlock.Text != "Hello from nested!" {
		t.Errorf("Expected text 'Hello from nested!', got '%s'", textBlock.Text)
	}

	toolBlock, ok := assistantMsg.Content[1].(ToolUseBlock)
	if !ok {
		t.Fatalf("Expected ToolUseBlock, got %T", assistantMsg.Content[1])
	}
	if toolBlock.Name != "Read" {
		t.Errorf("Expected tool name 'Read', got '%s'", toolBlock.Name)
	}
}

func TestParseUserMessage_Nested(t *testing.T) {
	// Claude Code stream-json format: user messages also nest content under message
	raw := json.RawMessage(`{
		"type": "user",
		"uuid": "usr-nested",
		"message": {
			"content": [
				{"type": "tool_result", "tool_use_id": "t1", "content": "file contents"}
			]
		}
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	userMsg, ok := msg.(UserMessage)
	if !ok {
		t.Fatal("Expected UserMessage")
	}

	if userMsg.UUID != "usr-nested" {
		t.Errorf("Expected UUID 'usr-nested', got '%s'", userMsg.UUID)
	}
	if len(userMsg.Content) != 1 {
		t.Fatalf("Expected 1 content block, got %d", len(userMsg.Content))
	}

	resultBlock, ok := userMsg.Content[0].(ToolResultBlock)
	if !ok {
		t.Fatalf("Expected ToolResultBlock, got %T", userMsg.Content[0])
	}
	if resultBlock.ToolUseID != "t1" {
		t.Errorf("Expected tool_use_id 't1', got '%s'", resultBlock.ToolUseID)
	}
}

func TestParseUserMessage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "user",
		"uuid": "usr-456",
		"content": "What is 2+2?"
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	userMsg, ok := msg.(UserMessage)
	if !ok {
		t.Fatal("Expected UserMessage")
	}

	if userMsg.UUID != "usr-456" {
		t.Errorf("Expected UUID 'usr-456', got '%s'", userMsg.UUID)
	}
}

func TestParseResultMessage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "result",
		"duration_ms": 1500,
		"duration_api_ms": 1200,
		"is_error": false,
		"num_turns": 3,
		"session_id": "sess-789",
		"total_cost_usd": 0.0025,
		"usage": {
			"input_tokens": 100,
			"output_tokens": 50
		}
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	resultMsg, ok := msg.(ResultMessage)
	if !ok {
		t.Fatal("Expected ResultMessage")
	}

	if resultMsg.DurationMS != 1500 {
		t.Errorf("Expected duration_ms 1500, got %d", resultMsg.DurationMS)
	}
	if resultMsg.NumTurns != 3 {
		t.Errorf("Expected num_turns 3, got %d", resultMsg.NumTurns)
	}
	if resultMsg.SessionID != "sess-789" {
		t.Errorf("Expected session_id 'sess-789', got '%s'", resultMsg.SessionID)
	}
	if resultMsg.Usage == nil {
		t.Fatal("Expected usage to be non-nil")
	}
	if resultMsg.Usage.InputTokens != 100 {
		t.Errorf("Expected input_tokens 100, got %d", resultMsg.Usage.InputTokens)
	}
}

func TestParseSystemMessage(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "system",
		"subtype": "init",
		"data": {"version": "1.0"}
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	systemMsg, ok := msg.(SystemMessage)
	if !ok {
		t.Fatal("Expected SystemMessage")
	}

	if systemMsg.Subtype != "init" {
		t.Errorf("Expected subtype 'init', got '%s'", systemMsg.Subtype)
	}
}

func TestGetText(t *testing.T) {
	msg := AssistantMessage{
		Content: []ContentBlock{
			TextBlock{Type: "text", Text: "Hello"},
			TextBlock{Type: "text", Text: "World"},
		},
	}

	text := GetText(msg)
	if text != "Hello\nWorld" {
		t.Errorf("Expected 'Hello\\nWorld', got '%s'", text)
	}
}

func TestGetToolCalls(t *testing.T) {
	msg := AssistantMessage{
		Content: []ContentBlock{
			TextBlock{Type: "text", Text: "Let me read that file"},
			ToolUseBlock{Type: "tool_use", ID: "t1", Name: "read_file", Input: map[string]interface{}{}},
			ToolUseBlock{Type: "tool_use", ID: "t2", Name: "write_file", Input: map[string]interface{}{}},
		},
	}

	tools := GetToolCalls(msg)
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tool calls, got %d", len(tools))
	}
	if tools[0].Name != "read_file" {
		t.Errorf("Expected first tool 'read_file', got '%s'", tools[0].Name)
	}
	if tools[1].Name != "write_file" {
		t.Errorf("Expected second tool 'write_file', got '%s'", tools[1].Name)
	}
}
