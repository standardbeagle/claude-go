package claude

import (
	"encoding/json"
	"testing"
	"time"
)

// TestParseMessage_StreamEvent covers the stream_event branch of ParseMessage,
// which is otherwise only exercised by integration tests.
func TestParseMessage_StreamEvent(t *testing.T) {
	raw := json.RawMessage(`{
		"type": "stream_event",
		"uuid": "se-1",
		"session_id": "sess-se",
		"event": "message_delta",
		"parent_tool_use_id": "ptid-1",
		"raw_event": {"delta": {"text": "hi"}}
	}`)

	msg, err := ParseMessage(raw)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	se, ok := msg.(StreamEvent)
	if !ok {
		t.Fatalf("Expected StreamEvent, got %T", msg)
	}
	if se.Type != "stream_event" {
		t.Errorf("Expected type 'stream_event', got '%s'", se.Type)
	}
	if se.UUID != "se-1" {
		t.Errorf("Expected uuid 'se-1', got '%s'", se.UUID)
	}
	if se.SessionID != "sess-se" {
		t.Errorf("Expected session_id 'sess-se', got '%s'", se.SessionID)
	}
	if se.Event != "message_delta" {
		t.Errorf("Expected event 'message_delta', got '%s'", se.Event)
	}
	if se.ParentToolUseID != "ptid-1" {
		t.Errorf("Expected parent_tool_use_id 'ptid-1', got '%s'", se.ParentToolUseID)
	}
	if len(se.RawEvent) == 0 {
		t.Error("Expected raw_event to be populated")
	}
}

// TestParseMessage_UnknownAndMalformed sweeps the default (unknown-type) branch
// and the top-level malformed-JSON branch of ParseMessage. The default branch
// must yield SystemMessage{Type: <type>, Subtype: "unknown"} without an error,
// and malformed JSON must return an error (never panic).
func TestParseMessage_UnknownAndMalformed(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantErr     bool
		wantType    string // expected SystemMessage.Type when no error
		wantSubtype string // expected SystemMessage.Subtype when no error
	}{
		{
			name:        "unknown type",
			raw:         `{"type": "control_request", "data": 1}`,
			wantErr:     false,
			wantType:    "control_request",
			wantSubtype: "unknown",
		},
		{
			name:        "another unknown type",
			raw:         `{"type": "summary"}`,
			wantErr:     false,
			wantType:    "summary",
			wantSubtype: "unknown",
		},
		{
			name:        "empty type string",
			raw:         `{"foo": "bar"}`,
			wantErr:     false,
			wantType:    "",
			wantSubtype: "unknown",
		},
		{
			name:    "malformed json",
			raw:     `{"type": "user", `,
			wantErr: true,
		},
		{
			name:    "not an object",
			raw:     `12345`,
			wantErr: true,
		},
		{
			name:    "garbage",
			raw:     `not json at all`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// guard against panic on any input
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseMessage panicked on %q: %v", tt.raw, r)
				}
			}()

			msg, err := ParseMessage(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error for %q, got nil (msg=%#v)", tt.raw, msg)
				}
				if msg != nil {
					t.Errorf("Expected nil message on error, got %#v", msg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for %q: %v", tt.raw, err)
			}
			sm, ok := msg.(SystemMessage)
			if !ok {
				t.Fatalf("Expected SystemMessage, got %T", msg)
			}
			if sm.Type != tt.wantType {
				t.Errorf("Expected Type '%s', got '%s'", tt.wantType, sm.Type)
			}
			if sm.Subtype != tt.wantSubtype {
				t.Errorf("Expected Subtype '%s', got '%s'", tt.wantSubtype, sm.Subtype)
			}
		})
	}
}

// TestParseContentBlock_UnknownAndMalformed covers the default (unknown-type)
// branch — which wraps the raw bytes in a TextBlock rather than erroring — and
// the malformed-JSON branch, which returns an error.
func TestParseContentBlock_UnknownAndMalformed(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantErr  bool
		wantText string // expected TextBlock.Text when no error (unknown type wraps raw)
	}{
		{
			name:     "unknown block type wraps raw",
			raw:      `{"type": "image", "source": {"data": "abc"}}`,
			wantErr:  false,
			wantText: `{"type": "image", "source": {"data": "abc"}}`,
		},
		{
			name:     "redacted_thinking unknown type",
			raw:      `{"type": "redacted_thinking", "data": "xyz"}`,
			wantErr:  false,
			wantText: `{"type": "redacted_thinking", "data": "xyz"}`,
		},
		{
			name:    "malformed block json",
			raw:     `{"type": "text", `,
			wantErr: true,
		},
		{
			name:    "not an object",
			raw:     `42`,
			wantErr: true,
		},
		{
			name:    "garbage",
			raw:     `???`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			block, err := ParseContentBlock(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error for %q, got nil (block=%#v)", tt.raw, block)
				}
				if block != nil {
					t.Errorf("Expected nil block on error, got %#v", block)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for %q: %v", tt.raw, err)
			}
			tb, ok := block.(TextBlock)
			if !ok {
				t.Fatalf("Expected TextBlock for unknown type, got %T", block)
			}
			if tb.Type != "text" {
				t.Errorf("Expected Type 'text', got '%s'", tb.Type)
			}
			if tb.Text != tt.wantText {
				t.Errorf("Expected Text %q, got %q", tt.wantText, tb.Text)
			}
		})
	}
}

// TestParseContentBlocks_Malformed covers the error path of ParseContentBlocks:
// input that is neither a JSON array nor a JSON string yields an error, and an
// array containing a malformed element propagates the element's parse error.
func TestParseContentBlocks_Malformed(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name:    "object is neither array nor string",
			raw:     `{"type": "text"}`,
			wantErr: true,
		},
		{
			name:    "truncated array",
			raw:     `[{"type": "text", "text": "hi"`,
			wantErr: true,
		},
		{
			name:    "array with malformed element",
			raw:     `[{"type": "text", "text": "ok"}, {"type": "text", `,
			wantErr: true,
		},
		{
			name:    "garbage",
			raw:     `nope`,
			wantErr: true,
		},
		{
			name:    "valid array still works",
			raw:     `[{"type": "text", "text": "ok"}]`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks, err := ParseContentBlocks(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Expected error for %q, got nil (blocks=%#v)", tt.raw, blocks)
				}
				if blocks != nil {
					t.Errorf("Expected nil blocks on error, got %#v", blocks)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error for %q: %v", tt.raw, err)
			}
			if len(blocks) != 1 {
				t.Errorf("Expected 1 block, got %d", len(blocks))
			}
		})
	}
}

// TestTimestamp_RoundTrip covers Timestamp.MarshalJSON / UnmarshalJSON (0%
// coverage today), including the error paths for non-string and unparseable
// inputs.
func TestTimestamp_RoundTrip(t *testing.T) {
	cases := []time.Time{
		time.Date(2026, 5, 29, 13, 14, 15, 123456789, time.UTC),
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2038, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	for _, want := range cases {
		ts := Timestamp(want)
		data, err := json.Marshal(ts)
		if err != nil {
			t.Fatalf("MarshalJSON failed for %v: %v", want, err)
		}
		// MarshalJSON encodes as an RFC3339Nano string, so the output is quoted.
		if len(data) < 2 || data[0] != '"' {
			t.Errorf("Expected quoted RFC3339 string, got %s", string(data))
		}

		var got Timestamp
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("UnmarshalJSON failed for %s: %v", string(data), err)
		}
		if !time.Time(got).Equal(want) {
			t.Errorf("Round-trip mismatch: want %v, got %v", want, time.Time(got))
		}
	}

	// Error path: value is not a JSON string.
	var notString Timestamp
	if err := json.Unmarshal([]byte(`12345`), &notString); err == nil {
		t.Error("Expected error unmarshalling non-string into Timestamp")
	}

	// Error path: string is not a valid RFC3339 timestamp.
	var badFormat Timestamp
	if err := json.Unmarshal([]byte(`"not-a-timestamp"`), &badFormat); err == nil {
		t.Error("Expected error unmarshalling malformed timestamp string")
	}
}
