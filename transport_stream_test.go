package claude

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// scriptedTransport is a minimal Transport double that delivers a
// pre-scripted sequence of TransportMessages on a channel the test owns.
// It performs no I/O and starts no subprocess.
type scriptedTransport struct {
	ch        chan TransportMessage
	connected bool
}

func newScriptedTransport(buf int) *scriptedTransport {
	return &scriptedTransport{ch: make(chan TransportMessage, buf), connected: true}
}

func (s *scriptedTransport) Connect(ctx context.Context) error { s.connected = true; return nil }
func (s *scriptedTransport) Send(msg []byte) error             { return nil }
func (s *scriptedTransport) Receive() <-chan TransportMessage  { return s.ch }
func (s *scriptedTransport) Close() error                      { s.connected = false; return nil }
func (s *scriptedTransport) IsConnected() bool                 { return s.connected }

// push enqueues a raw JSON line as a successful transport message.
func (s *scriptedTransport) push(raw string) { s.ch <- TransportMessage{Data: json.RawMessage(raw)} }

// pushErr enqueues a transport-level error message.
func (s *scriptedTransport) pushErr(err error) { s.ch <- TransportMessage{Error: err} }

// closeFeed closes the underlying receive channel, mimicking the subprocess
// transport closing t.messages when stdout reaches EOF.
func (s *scriptedTransport) closeFeed() { close(s.ch) }

// recvWithin reads one value from c, failing the test if nothing arrives
// within the deadline. Deterministic upper bound — not a sleep-based wait.
func recvWithin[T any](t *testing.T, c <-chan T, d time.Duration, what string) (T, bool) {
	t.Helper()
	select {
	case v, ok := <-c:
		return v, ok
	case <-time.After(d):
		var zero T
		t.Fatalf("timed out waiting for %s", what)
		return zero, false
	}
}

func TestStreamParser_Parse_ValidMessageDelivered(t *testing.T) {
	tr := newScriptedTransport(8)
	p := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Parse(ctx, tr)

	tr.push(`{"type":"assistant","message":{"content":[{"type":"text","text":"hi"}],"model":"claude-x"}}`)
	tr.push(`{"type":"result","is_error":false,"num_turns":1,"session_id":"abc","result":"done"}`)

	msg1, ok := recvWithin(t, p.Messages(), time.Second, "first message")
	if !ok {
		t.Fatal("messages channel closed before first message")
	}
	asst, isAsst := msg1.(AssistantMessage)
	if !isAsst {
		t.Fatalf("expected AssistantMessage, got %T", msg1)
	}
	if asst.Model != "claude-x" {
		t.Errorf("expected model claude-x, got %q", asst.Model)
	}
	if GetText(asst) != "hi" {
		t.Errorf("expected text 'hi', got %q", GetText(asst))
	}

	msg2, ok := recvWithin(t, p.Messages(), time.Second, "second message")
	if !ok {
		t.Fatal("messages channel closed before second message")
	}
	res, isRes := msg2.(ResultMessage)
	if !isRes {
		t.Fatalf("expected ResultMessage, got %T", msg2)
	}
	if res.SessionID != "abc" {
		t.Errorf("expected session abc, got %q", res.SessionID)
	}
	if res.Result != "done" {
		t.Errorf("expected result 'done', got %q", res.Result)
	}

	// No errors should have been produced for valid input.
	select {
	case err, ok := <-p.Errors():
		if ok {
			t.Fatalf("unexpected error on valid input: %v", err)
		}
	default:
	}
}

func TestStreamParser_Parse_TransportErrorForwardedLoopContinues(t *testing.T) {
	tr := newScriptedTransport(8)
	p := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Parse(ctx, tr)

	wantErr := errors.New("transport boom")
	tr.pushErr(wantErr)
	tr.push(`{"type":"system","subtype":"init"}`)

	gotErr, ok := recvWithin(t, p.Errors(), time.Second, "forwarded transport error")
	if !ok {
		t.Fatal("errors channel closed before forwarded error")
	}
	if gotErr == nil {
		t.Fatal("expected non-nil error")
	}
	if !errors.Is(gotErr, wantErr) {
		t.Errorf("expected forwarded transport error %v, got %v", wantErr, gotErr)
	}

	// Loop must continue: the subsequent good message is still delivered.
	msg, ok := recvWithin(t, p.Messages(), time.Second, "message after error")
	if !ok {
		t.Fatal("messages channel closed; loop did not continue after error")
	}
	sys, isSys := msg.(SystemMessage)
	if !isSys {
		t.Fatalf("expected SystemMessage after error, got %T", msg)
	}
	if sys.Subtype != "init" {
		t.Errorf("expected subtype init, got %q", sys.Subtype)
	}
}

func TestStreamParser_Parse_MalformedJSONDecodeErrorLoopContinues(t *testing.T) {
	tr := newScriptedTransport(8)
	p := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go p.Parse(ctx, tr)

	tr.push(`{this is not valid json`)
	tr.push(`{"type":"result","is_error":false,"session_id":"s2"}`)

	gotErr, ok := recvWithin(t, p.Errors(), time.Second, "decode error")
	if !ok {
		t.Fatal("errors channel closed before decode error")
	}
	var decErr *CLIJSONDecodeError
	if !errors.As(gotErr, &decErr) {
		t.Fatalf("expected *CLIJSONDecodeError, got %T (%v)", gotErr, gotErr)
	}
	if !IsJSONDecodeError(gotErr) {
		t.Error("IsJSONDecodeError should classify the decode error")
	}
	if decErr.Line == "" {
		t.Error("expected the offending line to be captured in the decode error")
	}

	// Loop must continue past the malformed line and deliver the next message.
	msg, ok := recvWithin(t, p.Messages(), time.Second, "message after malformed line")
	if !ok {
		t.Fatal("messages channel closed; loop did not continue after malformed line")
	}
	res, isRes := msg.(ResultMessage)
	if !isRes {
		t.Fatalf("expected ResultMessage after malformed line, got %T", msg)
	}
	if res.SessionID != "s2" {
		t.Errorf("expected session s2, got %q", res.SessionID)
	}
}

func TestStreamParser_Parse_ContextCancellationClosesBothChannels(t *testing.T) {
	tr := newScriptedTransport(8)
	p := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		p.Parse(ctx, tr)
		close(done)
	}()

	// Deliver one message and drain it to prove the parser is live first.
	tr.push(`{"type":"system","subtype":"alive"}`)
	if msg, ok := recvWithin(t, p.Messages(), time.Second, "liveness message"); !ok {
		t.Fatal("messages channel closed before liveness message")
	} else if _, isSys := msg.(SystemMessage); !isSys {
		t.Fatalf("expected SystemMessage, got %T", msg)
	}

	cancel()

	// Parse must return.
	if _, ok := recvWithin(t, done, time.Second, "Parse to return after cancel"); ok {
		// done was closed (zero value, ok==false expected on close); treat any return as success
	}

	// Both channels must be closed after Parse returns.
	if _, ok := recvWithin(t, p.Messages(), time.Second, "messages channel close"); ok {
		// Could be a buffered leftover; drain until closed.
		for {
			if _, ok := recvWithin(t, p.Messages(), time.Second, "messages channel close drain"); !ok {
				break
			}
		}
	}
	if _, ok := recvWithin(t, p.Errors(), time.Second, "errors channel close"); ok {
		for {
			if _, ok := recvWithin(t, p.Errors(), time.Second, "errors channel close drain"); !ok {
				break
			}
		}
	}
}

func TestStreamParser_Parse_ReceiveChannelClosedReturns(t *testing.T) {
	tr := newScriptedTransport(8)
	p := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.Parse(ctx, tr)
		close(done)
	}()

	tr.push(`{"type":"result","is_error":false,"session_id":"final"}`)
	msg, ok := recvWithin(t, p.Messages(), time.Second, "pre-close message")
	if !ok {
		t.Fatal("messages channel closed before pre-close message")
	}
	if res, isRes := msg.(ResultMessage); !isRes || res.SessionID != "final" {
		t.Fatalf("expected ResultMessage session 'final', got %T %+v", msg, msg)
	}

	// Closing the underlying feed must cause Parse to return.
	tr.closeFeed()

	recvWithin(t, done, time.Second, "Parse to return after feed close")

	// Both downstream channels must be closed.
	if _, ok := recvWithin(t, p.Messages(), time.Second, "messages closed after feed close"); ok {
		t.Error("expected messages channel to be closed after feed close")
	}
	if _, ok := recvWithin(t, p.Errors(), time.Second, "errors closed after feed close"); ok {
		t.Error("expected errors channel to be closed after feed close")
	}
}
