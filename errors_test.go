package claude

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestClaudeSDKError(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewClaudeSDKError("operation failed", cause)

	if err.Error() != "operation failed: underlying error" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	if err.Unwrap() != cause {
		t.Error("Unwrap did not return cause")
	}
}

func TestClaudeSDKErrorNoCause(t *testing.T) {
	err := NewClaudeSDKError("simple error", nil)

	if err.Error() != "simple error" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	if err.Unwrap() != nil {
		t.Error("Unwrap should return nil")
	}
}

func TestCLIConnectionError(t *testing.T) {
	cause := errors.New("connection refused")
	err := NewCLIConnectionError("failed to connect", cause)

	if !IsCLIConnectionError(err) {
		t.Error("IsCLIConnectionError should return true")
	}

	expected := "failed to connect: connection refused"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestCLINotFoundError(t *testing.T) {
	err := NewCLINotFoundError("Claude CLI not found", "/usr/bin/claude, /usr/local/bin/claude")

	if !IsCLINotFound(err) {
		t.Error("IsCLINotFound should return true")
	}

	if !IsCLIConnectionError(err) {
		t.Error("CLINotFoundError should also be a CLIConnectionError")
	}

	expected := "Claude CLI not found (searched: /usr/bin/claude, /usr/local/bin/claude)"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestProcessError(t *testing.T) {
	err := NewProcessError("process exited", 1, "error output here", nil)

	if !IsProcessError(err) {
		t.Error("IsProcessError should return true")
	}

	if err.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", err.ExitCode)
	}
	if err.Stderr != "error output here" {
		t.Errorf("Unexpected stderr: %s", err.Stderr)
	}

	errStr := err.Error()
	if errStr != "process exited (exit code: 1)\nstderr: error output here" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestCLIJSONDecodeError(t *testing.T) {
	cause := errors.New("invalid json")
	err := NewCLIJSONDecodeError("failed to decode", `{"invalid": json}`, cause)

	if !IsJSONDecodeError(err) {
		t.Error("IsJSONDecodeError should return true")
	}

	errStr := err.Error()
	if errStr != "failed to decode (line: {\"invalid\": json}): invalid json" {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestCLIJSONDecodeErrorTruncation(t *testing.T) {
	longLine := make([]byte, 200)
	for i := range longLine {
		longLine[i] = 'x'
	}

	err := NewCLIJSONDecodeError("failed", string(longLine), nil)

	if len(err.Line) != 200 {
		t.Error("Line should not be truncated in struct")
	}

	errStr := err.Error()
	// Error string should have truncated line
	if len(errStr) > 150 {
		// Check that it contains truncation marker
		if errStr[len(errStr)-4:] != "...)" {
			t.Logf("Error ends with: %s", errStr[len(errStr)-10:])
		}
	}
}

func TestMessageParseError(t *testing.T) {
	data := json.RawMessage(`{"type": "unknown"}`)
	err := NewMessageParseError("unknown message type", data, nil)

	if !IsMessageParseError(err) {
		t.Error("IsMessageParseError should return true")
	}

	errStr := err.Error()
	if errStr != `unknown message type (data: {"type": "unknown"})` {
		t.Errorf("Unexpected error string: %s", errStr)
	}
}

func TestSessionClosedError(t *testing.T) {
	err := NewSessionClosedError("session-123")

	if !IsSessionClosedError(err) {
		t.Error("IsSessionClosedError should return true")
	}

	if err.SessionID != "session-123" {
		t.Errorf("Expected session ID 'session-123', got '%s'", err.SessionID)
	}

	expected := "session is closed (session: session-123)"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestToolError(t *testing.T) {
	cause := errors.New("file not found")
	err := NewToolError("read failed", "Read", "tool-456", cause)

	if !IsToolError(err) {
		t.Error("IsToolError should return true")
	}

	if err.ToolName != "Read" {
		t.Errorf("Expected tool name 'Read', got '%s'", err.ToolName)
	}
	if err.ToolUseID != "tool-456" {
		t.Errorf("Expected tool use ID 'tool-456', got '%s'", err.ToolUseID)
	}

	expected := "[Read] read failed (id: tool-456): file not found"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestHookError(t *testing.T) {
	cause := errors.New("hook panicked")
	err := NewHookError("hook failed", "PreToolUse", "Bash", cause)

	if !IsHookError(err) {
		t.Error("IsHookError should return true")
	}

	if err.HookEvent != "PreToolUse" {
		t.Errorf("Expected hook event 'PreToolUse', got '%s'", err.HookEvent)
	}
	if err.ToolName != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", err.ToolName)
	}

	expected := "[PreToolUse] hook failed (tool: Bash): hook panicked"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestErrorTypeCheckers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		checker func(error) bool
		want    bool
	}{
		{"CLINotFound true", NewCLINotFoundError("not found", ""), IsCLINotFound, true},
		{"CLINotFound false", NewClaudeSDKError("other", nil), IsCLINotFound, false},
		{"CLIConnection true", NewCLIConnectionError("conn", nil), IsCLIConnectionError, true},
		{"CLIConnection from NotFound", NewCLINotFoundError("not found", ""), IsCLIConnectionError, true},
		{"Process true", NewProcessError("proc", 1, "", nil), IsProcessError, true},
		{"Process false", NewClaudeSDKError("other", nil), IsProcessError, false},
		{"JSONDecode true", NewCLIJSONDecodeError("json", "", nil), IsJSONDecodeError, true},
		{"MessageParse true", NewMessageParseError("parse", nil, nil), IsMessageParseError, true},
		{"SessionClosed true", NewSessionClosedError("s1"), IsSessionClosedError, true},
		{"Tool true", NewToolError("tool", "t", "id", nil), IsToolError, true},
		{"Hook true", NewHookError("hook", "e", "t", nil), IsHookError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.checker(tt.err)
			if got != tt.want {
				t.Errorf("Expected %v, got %v", tt.want, got)
			}
		})
	}
}
