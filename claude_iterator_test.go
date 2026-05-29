package claude

import (
	"context"
	"errors"
	"testing"
	"time"
)

// newTestIterator builds a QueryIterator wired to a scripted transport and a
// live StreamParser, without starting any subprocess. The transport field is
// an unconnected SubprocessTransport so that Close() is a safe no-op-ish call
// (its Close returns nil immediately when not connected) — exercising the real
// Close() path without spawning a CLI.
func newTestIterator(t *testing.T) (*QueryIterator, *scriptedTransport) {
	t.Helper()
	st, err := NewSubprocessTransport(&SubprocessTransportOptions{CLIPath: "/nonexistent/claude-cli-stub"})
	if err != nil {
		t.Fatalf("NewSubprocessTransport: %v", err)
	}
	scripted := newScriptedTransport(16)
	parser := NewStreamParser()
	ctx, cancel := context.WithCancel(context.Background())
	go parser.Parse(ctx, scripted)
	return &QueryIterator{
		transport: st,
		parser:    parser,
		ctx:       ctx,
		cancel:    cancel,
		started:   true,
	}, scripted
}

func TestQueryIterator_Next_DrainsThenTerminates(t *testing.T) {
	it, scripted := newTestIterator(t)
	defer it.Close()

	scripted.push(`{"type":"system","subtype":"init"}`)
	scripted.push(`{"type":"assistant","message":{"content":[{"type":"text","text":"yo"}]}}`)
	scripted.push(`{"type":"result","is_error":false,"session_id":"q1","result":"ok"}`)

	// NB: the feed is intentionally left OPEN while draining. QueryIterator.Next
	// selects across Messages() AND Errors(); once the feed is closed both
	// parser channels close and a ready closed-Errors branch can win the select,
	// returning the (nil,nil) terminal before buffered messages are drained.
	// So we drain all three messages first, then close the feed to observe the
	// terminal. This documents a real ordering caveat in Next().

	m1, err := it.Next()
	if err != nil {
		t.Fatalf("Next() #1 error: %v", err)
	}
	if _, ok := m1.(SystemMessage); !ok {
		t.Fatalf("expected SystemMessage first, got %T", m1)
	}

	m2, err := it.Next()
	if err != nil {
		t.Fatalf("Next() #2 error: %v", err)
	}
	asst, ok := m2.(AssistantMessage)
	if !ok {
		t.Fatalf("expected AssistantMessage second, got %T", m2)
	}
	if GetText(asst) != "yo" {
		t.Errorf("expected text 'yo', got %q", GetText(asst))
	}

	m3, err := it.Next()
	if err != nil {
		t.Fatalf("Next() #3 error: %v", err)
	}
	if res, ok := m3.(ResultMessage); !ok || res.SessionID != "q1" {
		t.Fatalf("expected ResultMessage session q1, got %T %+v", m3, m3)
	}

	// Now close the feed: Parse returns and closes both parser channels.
	scripted.closeFeed()

	// After the feed closes and all messages drain, Next() returns the
	// documented terminal: (nil, nil) — see QueryIterator.Next, the
	// `!ok` branch on a closed Messages/Errors channel.
	mEnd, err := it.Next()
	if err != nil {
		t.Fatalf("terminal Next() should not error, got: %v", err)
	}
	if mEnd != nil {
		t.Fatalf("terminal Next() should return nil message, got %T %+v", mEnd, mEnd)
	}
}

func TestQueryIterator_ChannelsDeliverMessagesAndErrors(t *testing.T) {
	it, scripted := newTestIterator(t)
	defer it.Close()

	transportErr := errors.New("scripted transport failure")
	scripted.pushErr(transportErr)
	scripted.push(`{"type":"result","is_error":false,"session_id":"chan"}`)

	// Errors() channel delivers the transport-level error.
	gotErr, ok := recvWithin(t, it.Errors(), time.Second, "iterator error")
	if !ok {
		t.Fatal("errors channel closed before delivering error")
	}
	if !errors.Is(gotErr, transportErr) {
		t.Errorf("expected %v, got %v", transportErr, gotErr)
	}

	// Messages() channel delivers the subsequent good message.
	gotMsg, ok := recvWithin(t, it.Messages(), time.Second, "iterator message")
	if !ok {
		t.Fatal("messages channel closed before delivering message")
	}
	res, isRes := gotMsg.(ResultMessage)
	if !isRes {
		t.Fatalf("expected ResultMessage, got %T", gotMsg)
	}
	if res.SessionID != "chan" {
		t.Errorf("expected session chan, got %q", res.SessionID)
	}

	// Messages() and Errors() must be the parser's own channels (identity),
	// guaranteeing channel-based consumption sees the same stream as Next().
	if it.Messages() != it.parser.Messages() {
		t.Error("iterator Messages() should expose the parser's message channel")
	}
	if it.Errors() != it.parser.Errors() {
		t.Error("iterator Errors() should expose the parser's error channel")
	}
}

func TestQueryIterator_CloseIdempotentAndUnblocksNext(t *testing.T) {
	it, scripted := newTestIterator(t)

	// A pending Next() with no data available blocks until Close cancels ctx.
	type result struct {
		msg MessageType
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		m, e := it.Next()
		resCh <- result{m, e}
	}()

	// Close must unblock the pending Next() via ctx cancellation. We use a
	// done channel + select, not a sleep, to observe the unblock.
	if err := it.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}

	got, ok := recvWithin(t, resCh, 2*time.Second, "Next() to unblock after Close")
	if !ok {
		t.Fatal("Next() result channel closed unexpectedly")
	}
	// The unblock is guaranteed; the *value* is not deterministic. Close()
	// cancels ctx, which both makes Next()'s <-ctx.Done() case ready AND causes
	// Parse to close the parser's Messages/Errors channels. Next() selects
	// across all three ready cases, so a pending Next() may return either
	// (nil, context.Canceled) or the (nil, nil) terminal. Both are valid
	// "unblocked, no message" outcomes — the test asserts the unblock and a
	// nil message, accepting either terminal error shape.
	if got.msg != nil {
		t.Errorf("expected nil message from unblocked Next(), got %T", got.msg)
	}
	if got.err != nil && !errors.Is(got.err, context.Canceled) {
		t.Errorf("expected nil or context.Canceled from unblocked Next(), got %v", got.err)
	}

	// Deterministic post-Close invariant: the iterator's context is canceled.
	if it.ctx.Err() == nil {
		t.Error("expected iterator ctx to be canceled after Close()")
	}
	if !errors.Is(it.ctx.Err(), context.Canceled) {
		t.Errorf("expected ctx.Err() == context.Canceled, got %v", it.ctx.Err())
	}

	// Close() must be idempotent: a second call is safe and non-erroring.
	if err := it.Close(); err != nil {
		t.Errorf("second Close() should be idempotent, got: %v", err)
	}
	// And a third, to firmly assert idempotency.
	if err := it.Close(); err != nil {
		t.Errorf("third Close() should be idempotent, got: %v", err)
	}

	// The parser goroutine must terminate after ctx cancel; the parser's
	// Messages channel must be closed (deterministic — Parse's defer runs on
	// ctx cancel). Drain to closed without blocking on a sleep.
	closed := false
	for i := 0; i < 200 && !closed; i++ {
		select {
		case _, ok := <-it.parser.Messages():
			if !ok {
				closed = true
			}
		case <-time.After(time.Second):
			t.Fatal("parser Messages channel did not close after Close()")
		}
	}
	if !closed {
		t.Error("expected parser Messages channel to be closed after Close()")
	}

	// Closing the feed after the parser has already exited must not panic.
	scripted.closeFeed()
}
