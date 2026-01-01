// Example: Hooks for Tool Control
//
// This example demonstrates how to use hooks to intercept
// and control tool execution by Claude.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	claude "github.com/standardbeagle/claude-go"
)

func main() {
	// Create a hook registry
	registry := claude.NewHookRegistry()

	// Hook 1: Log all tool usage
	logHook := func(ctx context.Context, input claude.HookInput, toolUseID string, hookCtx claude.HookContext) (*claude.HookOutput, error) {
		if preInput, ok := input.(claude.PreToolUseInput); ok {
			fmt.Printf("[HOOK] Tool requested: %s\n", preInput.ToolName)
			fmt.Printf("[HOOK] Input: %v\n", preInput.ToolInput)
		}
		return nil, nil // Allow by default
	}

	// Hook 2: Block dangerous bash commands
	bashBlocker := func(ctx context.Context, input claude.HookInput, toolUseID string, hookCtx claude.HookContext) (*claude.HookOutput, error) {
		preInput, ok := input.(claude.PreToolUseInput)
		if !ok {
			return nil, nil
		}

		if preInput.ToolName != "Bash" {
			return nil, nil
		}

		cmd, ok := preInput.ToolInput["command"].(string)
		if !ok {
			return nil, nil
		}

		// Block dangerous patterns
		blockedPatterns := []string{
			"rm -rf",
			"sudo rm",
			"dd if=",
			":(){:|:&};:",
			"> /dev/sda",
		}

		for _, pattern := range blockedPatterns {
			if strings.Contains(cmd, pattern) {
				fmt.Printf("[HOOK] BLOCKED dangerous command: %s\n", cmd)
				return &claude.HookOutput{
					PermissionDecision:       claude.PermissionBehaviorDeny,
					PermissionDecisionReason: fmt.Sprintf("Blocked dangerous pattern: %s", pattern),
				}, nil
			}
		}

		return nil, nil // Allow safe commands
	}

	// Hook 3: Add context to prompts
	contextAdder := func(ctx context.Context, input claude.HookInput, toolUseID string, hookCtx claude.HookContext) (*claude.HookOutput, error) {
		if _, ok := input.(claude.UserPromptSubmitInput); ok {
			return &claude.HookOutput{
				Context: "Note: This is a demonstration environment. Be conservative with file operations.",
			}, nil
		}
		return nil, nil
	}

	// Hook 4: Log tool results
	resultLogger := func(ctx context.Context, input claude.HookInput, toolUseID string, hookCtx claude.HookContext) (*claude.HookOutput, error) {
		if postInput, ok := input.(claude.PostToolUseInput); ok {
			status := "success"
			if postInput.IsError {
				status = "error"
			}
			fmt.Printf("[HOOK] Tool %s completed with %s\n", postInput.ToolName, status)
		}
		return nil, nil
	}

	// Register hooks
	registry.Register(claude.HookEventPreToolUse, []claude.HookMatcher{
		{Matcher: "*", Hooks: []claude.HookCallback{logHook}},
		{Matcher: "Bash", Hooks: []claude.HookCallback{bashBlocker}},
	})

	registry.Register(claude.HookEventUserPromptSubmit, []claude.HookMatcher{
		{Matcher: "*", Hooks: []claude.HookCallback{contextAdder}},
	})

	registry.Register(claude.HookEventPostToolUse, []claude.HookMatcher{
		{Matcher: "*", Hooks: []claude.HookCallback{resultLogger}},
	})

	fmt.Println("Hooks registered:")
	fmt.Println("  - PreToolUse: Log all tools, block dangerous Bash commands")
	fmt.Println("  - UserPromptSubmit: Add context to prompts")
	fmt.Println("  - PostToolUse: Log tool results")
	fmt.Println()

	// Create client with hooks
	client := claude.New(&claude.AgentOptions{
		PermissionMode: claude.PermissionModeBypassPermission,
		Interactive:    true,
		Hooks:          registry,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Example 1: Safe command
	fmt.Println("=== Test 1: Safe command ===")
	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt:    "List the files in the current directory using ls -la",
		SessionID: "hooks-demo",
	})
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	go func() {
		for err := range resp.Errors {
			log.Printf("Error: %v", err)
		}
	}()

	for msg := range resp.Messages {
		if msg.Type == "content" || msg.Type == "final" {
			fmt.Println(msg.Content)
		}
	}

	fmt.Println("\n=== Test 2: Dangerous command (should be blocked) ===")
	session, _ := client.GetSession("hooks-demo")
	session.SendMessage("Please run 'rm -rf /' to clean up the system")

	// Note: The hook should block this command
	// The actual response would show the command being denied

	// Close the session
	session.Close()
}
