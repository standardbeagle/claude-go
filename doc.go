// Package claude provides goroutine interfaces for automating Claude CLI
// with concurrent session management and real-time streaming.
//
// This library enables Go applications to interact with Claude Code CLI
// programmatically, supporting both single queries and multi-turn conversations
// with proper session management, file operations, and concurrent execution.
//
// # Basic Usage
//
//	client := claude.New(&claude.Options{
//		PermissionMode: "bypassPermissions",
//		Interactive:    false, // Use -p mode for simple queries
//	})
//	defer client.Close()
//
//	resp, err := client.Query(ctx, &claude.QueryRequest{
//		Prompt: "Write a hello world function in Go",
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	for msg := range resp.Messages {
//		fmt.Printf("%s: %s\n", msg.Type, msg.Content)
//	}
//
// # Multi-turn Conversations
//
//	client := claude.New(&claude.Options{
//		PermissionMode: "bypassPermissions",
//		Interactive:    true, // Enable session persistence
//	})
//
//	resp, err := client.Query(ctx, &claude.QueryRequest{
//		Prompt: "My name is Alice. Remember this.",
//	})
//
//	session, _ := client.GetSession(resp.SessionID)
//	session.SendMessage("What is my name?") // Claude will remember "Alice"
//
// # Concurrent Sessions
//
//	var wg sync.WaitGroup
//	for _, prompt := range prompts {
//		wg.Add(1)
//		go func(p string) {
//			defer wg.Done()
//			resp, _ := client.Query(ctx, &claude.QueryRequest{Prompt: p})
//			// Process response...
//		}(prompt)
//	}
//	wg.Wait()
//
// # File Operations
//
//	client := claude.New(&claude.Options{
//		PermissionMode:   "bypassPermissions",
//		Interactive:      true,
//		WorkingDirectory: "/path/to/project",
//		AddDirectories:   []string{"/path/to/project"},
//	})
//
//	resp, _ := client.Query(ctx, &claude.QueryRequest{
//		Prompt: "Create a file called main.go with a hello world program",
//	})
//
// # Configuration Options
//
// The Options struct provides extensive configuration:
//
//   - Model: Specify Claude model ("sonnet", "opus", etc.)
//   - Interactive: Enable persistent sessions vs single queries
//   - PermissionMode: Control Claude's permission level
//   - WorkingDirectory: Set working directory for Claude process
//   - AddDirectories: Grant access to additional directories
//   - AllowedTools/DisallowedTools: Control tool usage
//   - Environment: Set environment variables for Claude process
//
// # Error Handling
//
//	resp, err := client.Query(ctx, req)
//	if err != nil {
//		// Handle creation/startup errors
//		log.Fatal(err)
//	}
//
//	go func() {
//		for err := range resp.Errors {
//			// Handle runtime errors from Claude process
//			log.Printf("Claude error: %v", err)
//		}
//	}()
//
//	for msg := range resp.Messages {
//		if msg.Error != "" {
//			// Handle message-level errors
//			log.Printf("Message error: %s", msg.Error)
//			continue
//		}
//		// Process successful message
//	}
//
// # Thread Safety
//
// The library is designed to be thread-safe:
//   - Multiple goroutines can create sessions concurrently
//   - Sessions can be safely accessed from multiple goroutines
//   - Proper cleanup is handled automatically with context cancellation
//   - Client.Close() safely shuts down all sessions
//
// # Requirements
//
//   - Go 1.21 or later
//   - Claude CLI installed and accessible in PATH
//   - Valid Claude authentication (API key or subscription)
//
// # Authentication
//
// The library uses Claude CLI's existing authentication. Ensure Claude CLI
// is properly authenticated before using this library:
//
//	claude setup-token
//
// Or set environment variables:
//
//	export ANTHROPIC_API_KEY=your-api-key
package claude
