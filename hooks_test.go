package claude

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHookRegistry(t *testing.T) {
	registry := NewHookRegistry()

	called := false
	hook := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		called = true
		return &HookOutput{
			PermissionDecision: PermissionBehaviorAllow,
		}, nil
	}

	registry.Register(HookEventPreToolUse, []HookMatcher{
		{Matcher: "Bash", Hooks: []HookCallback{hook}},
	})

	matchers := registry.GetMatchers(HookEventPreToolUse)
	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher, got %d", len(matchers))
	}

	// Execute hooks
	input := PreToolUseInput{
		BaseHookInput: BaseHookInput{SessionID: "test"},
		ToolName:      "Bash",
		ToolInput:     map[string]interface{}{"command": "ls"},
	}

	output, err := registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input, "tool-123", HookContext{})
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if !called {
		t.Error("Hook was not called")
	}
	if output.PermissionDecision != PermissionBehaviorAllow {
		t.Errorf("Expected allow, got %s", output.PermissionDecision)
	}
}

func TestHookMatcherWildcard(t *testing.T) {
	registry := NewHookRegistry()

	callCount := 0
	hook := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		callCount++
		return nil, nil
	}

	registry.Register(HookEventPreToolUse, []HookMatcher{
		{Matcher: "*", Hooks: []HookCallback{hook}},
	})

	// Should match any tool
	input1 := PreToolUseInput{ToolName: "Bash"}
	_, _ = registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input1, "", HookContext{})

	input2 := PreToolUseInput{ToolName: "Read"}
	_, _ = registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input2, "", HookContext{})

	if callCount != 2 {
		t.Errorf("Expected 2 calls, got %d", callCount)
	}
}

func TestHookMatcherNoMatch(t *testing.T) {
	registry := NewHookRegistry()

	called := false
	hook := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		called = true
		return nil, nil
	}

	registry.Register(HookEventPreToolUse, []HookMatcher{
		{Matcher: "Bash", Hooks: []HookCallback{hook}},
	})

	// Should NOT match
	input := PreToolUseInput{ToolName: "Read"}
	_, _ = registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input, "", HookContext{})

	if called {
		t.Error("Hook should not have been called for non-matching tool")
	}
}

func TestDenyToolHook(t *testing.T) {
	hook := DenyTool("Command not allowed")

	input := PreToolUseInput{
		ToolName:  "Bash",
		ToolInput: map[string]interface{}{"command": "rm -rf /"},
	}

	output, err := hook(context.Background(), input, "tool-123", HookContext{})
	if err != nil {
		t.Fatalf("DenyTool failed: %v", err)
	}

	if output.PermissionDecision != PermissionBehaviorDeny {
		t.Errorf("Expected deny, got %s", output.PermissionDecision)
	}
	if output.PermissionDecisionReason != "Command not allowed" {
		t.Errorf("Expected reason, got '%s'", output.PermissionDecisionReason)
	}
}

func TestAllowToolHook(t *testing.T) {
	hook := AllowTool()

	input := PreToolUseInput{ToolName: "Bash"}

	output, err := hook(context.Background(), input, "", HookContext{})
	if err != nil {
		t.Fatalf("AllowTool failed: %v", err)
	}

	if output.PermissionDecision != PermissionBehaviorAllow {
		t.Errorf("Expected allow, got %s", output.PermissionDecision)
	}
}

func TestStopExecutionHook(t *testing.T) {
	hook := StopExecution("Max turns reached")

	input := StopInput{}

	output, err := hook(context.Background(), input, "", HookContext{})
	if err != nil {
		t.Fatalf("StopExecution failed: %v", err)
	}

	if output.Continue == nil || *output.Continue != false {
		t.Error("Expected continue to be false")
	}
	if output.StopReason != "Max turns reached" {
		t.Errorf("Expected stop reason, got '%s'", output.StopReason)
	}
}

func TestAddContextHook(t *testing.T) {
	hook := AddContext("Additional context here")

	input := UserPromptSubmitInput{Prompt: "Hello"}

	output, err := hook(context.Background(), input, "", HookContext{})
	if err != nil {
		t.Fatalf("AddContext failed: %v", err)
	}

	if output.Context != "Additional context here" {
		t.Errorf("Expected context, got '%s'", output.Context)
	}
}

func TestHooksMapToRegistry(t *testing.T) {
	called := false
	hooks := Hooks{
		HookEventPreToolUse: []HookMatcher{
			{
				Matcher: "Bash",
				Hooks: []HookCallback{
					func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
						called = true
						return nil, nil
					},
				},
			},
		},
	}

	registry := hooks.ToRegistry()

	input := PreToolUseInput{ToolName: "Bash"}
	_, _ = registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input, "", HookContext{})

	if !called {
		t.Error("Hook from map was not called")
	}
}

func TestParseHookInput(t *testing.T) {
	tests := []struct {
		name  string
		event HookEvent
		raw   json.RawMessage
	}{
		{
			name:  "PreToolUse",
			event: HookEventPreToolUse,
			raw:   json.RawMessage(`{"session_id": "s1", "tool_name": "Bash", "tool_input": {}}`),
		},
		{
			name:  "PostToolUse",
			event: HookEventPostToolUse,
			raw:   json.RawMessage(`{"session_id": "s1", "tool_name": "Bash", "tool_input": {}, "tool_result": "ok"}`),
		},
		{
			name:  "UserPromptSubmit",
			event: HookEventUserPromptSubmit,
			raw:   json.RawMessage(`{"session_id": "s1", "prompt": "Hello"}`),
		},
		{
			name:  "Stop",
			event: HookEventStop,
			raw:   json.RawMessage(`{"session_id": "s1", "reason": "done"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := ParseHookInput(tt.event, tt.raw)
			if err != nil {
				t.Fatalf("ParseHookInput failed: %v", err)
			}
			if input == nil {
				t.Fatal("Expected non-nil input")
			}
		})
	}
}

func TestMultipleHooksExecution(t *testing.T) {
	registry := NewHookRegistry()

	callOrder := []string{}

	hook1 := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		callOrder = append(callOrder, "hook1")
		return &HookOutput{Feedback: "feedback1"}, nil
	}

	hook2 := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		callOrder = append(callOrder, "hook2")
		return &HookOutput{Feedback: "feedback2"}, nil
	}

	registry.Register(HookEventPostToolUse, []HookMatcher{
		{Matcher: "*", Hooks: []HookCallback{hook1, hook2}},
	})

	input := PostToolUseInput{ToolName: "Bash"}
	output, err := registry.ExecuteHooks(context.Background(), HookEventPostToolUse, input, "", HookContext{})
	if err != nil {
		t.Fatalf("ExecuteHooks failed: %v", err)
	}

	if len(callOrder) != 2 {
		t.Fatalf("Expected 2 hooks called, got %d", len(callOrder))
	}
	if callOrder[0] != "hook1" || callOrder[1] != "hook2" {
		t.Errorf("Unexpected call order: %v", callOrder)
	}

	// Last hook's output should win
	if output.Feedback != "feedback2" {
		t.Errorf("Expected feedback2, got '%s'", output.Feedback)
	}
}

func TestHookDenyStopsExecution(t *testing.T) {
	registry := NewHookRegistry()

	hook1Called := false
	hook2Called := false

	hook1 := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		hook1Called = true
		return &HookOutput{PermissionDecision: PermissionBehaviorDeny}, nil
	}

	hook2 := func(ctx context.Context, input HookInput, toolUseID string, hookCtx HookContext) (*HookOutput, error) {
		hook2Called = true
		return nil, nil
	}

	registry.Register(HookEventPreToolUse, []HookMatcher{
		{Matcher: "*", Hooks: []HookCallback{hook1, hook2}},
	})

	input := PreToolUseInput{ToolName: "Bash"}
	output, _ := registry.ExecuteHooks(context.Background(), HookEventPreToolUse, input, "", HookContext{})

	if !hook1Called {
		t.Error("First hook should have been called")
	}
	if hook2Called {
		t.Error("Second hook should NOT have been called after deny")
	}
	if output.PermissionDecision != PermissionBehaviorDeny {
		t.Error("Expected deny decision")
	}
}
