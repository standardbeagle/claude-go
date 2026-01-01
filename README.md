# Claude Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/standardbeagle/claude-go.svg)](https://pkg.go.dev/github.com/standardbeagle/claude-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/standardbeagle/claude-go)](https://goreportcard.com/report/github.com/standardbeagle/claude-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go SDK for automating Claude Code CLI with concurrent session management, real-time streaming, custom tools, and hooks. This is the Go equivalent of the [Python Claude Agent SDK](https://github.com/anthropics/claude-agent-sdk-python).

## Features

- **Simple Query Function**: One-shot queries without session management
- **Concurrent Sessions**: Run multiple Claude sessions simultaneously with goroutines
- **Structured Message Types**: Typed messages (AssistantMessage, UserMessage, ResultMessage, etc.)
- **Custom Tools (MCP)**: Define in-process MCP tools that run directly in your Go application
- **Hooks**: Intercept and control tool execution with PreToolUse, PostToolUse hooks
- **JSON Streaming**: Real-time streaming with typed content blocks
- **CLI Discovery**: Automatic detection of Claude CLI in PATH and common locations
- **Comprehensive Errors**: Typed errors for precise error handling
- **Session Management**: Create, manage, and reuse conversation sessions

## Installation

```bash
go get github.com/standardbeagle/claude-go
```

## Quick Start

### Simple One-Shot Query

```go
package main

import (
    "context"
    "fmt"
    "log"

    claude "github.com/standardbeagle/claude-go"
)

func main() {
    ctx := context.Background()

    messages, err := claude.Query(ctx, "What is 2+2?", nil)
    if err != nil {
        log.Fatal(err)
    }

    for _, msg := range messages {
        if text := claude.GetText(msg); text != "" {
            fmt.Println(text)
        }
    }
}
```

### Client-Based Usage

```go
client := claude.New(&claude.AgentOptions{
    PermissionMode: claude.PermissionModeBypassPermission,
    Model:          "sonnet",
})
defer client.Close()

resp, err := client.Query(ctx, &claude.QueryRequest{
    Prompt: "Write a hello world function in Go",
})
if err != nil {
    log.Fatal(err)
}

for msg := range resp.Messages {
    fmt.Printf("%s: %s\n", msg.Type, msg.Content)
}
```

## Custom Tools (In-Process MCP Servers)

Define custom tools that Claude can use during conversations:

```go
// Create a tool using the builder pattern
addTool := claude.Tool("add", "Add two numbers").
    Param("a", "number", "First number").
    Param("b", "number", "Second number").
    Required("a", "b").
    HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
        a := args["a"].(float64)
        b := args["b"].(float64)
        return fmt.Sprintf("%g", a+b), nil
    })

// Create an SDK MCP server
server := claude.CreateSDKMCPServer("calculator", "1.0.0", addTool)

// Register with client
client := claude.New(&claude.AgentOptions{
    PermissionMode: claude.PermissionModeBypassPermission,
    AllowedTools:   []string{"mcp__calculator__add"},
})
client.RegisterSDKMCPServer("calculator", server)
```

## Hooks

Intercept and control tool execution:

```go
// Create a hook to block dangerous commands
bashBlocker := func(ctx context.Context, input claude.HookInput, toolUseID string, hookCtx claude.HookContext) (*claude.HookOutput, error) {
    if preInput, ok := input.(claude.PreToolUseInput); ok {
        if cmd, ok := preInput.ToolInput["command"].(string); ok {
            if strings.Contains(cmd, "rm -rf") {
                return &claude.HookOutput{
                    PermissionDecision: claude.PermissionBehaviorDeny,
                    PermissionDecisionReason: "Dangerous command blocked",
                }, nil
            }
        }
    }
    return nil, nil
}

registry := claude.NewHookRegistry()
registry.Register(claude.HookEventPreToolUse, []claude.HookMatcher{
    {Matcher: "Bash", Hooks: []claude.HookCallback{bashBlocker}},
})

client := claude.New(&claude.AgentOptions{
    Hooks: registry,
})
```

## Configuration Options (AgentOptions)

```go
type AgentOptions struct {
    // Model configuration
    Model             string         // Model to use (e.g., "sonnet", "opus")
    MaxThinkingTokens int            // Maximum tokens for thinking

    // System prompt
    SystemPrompt string

    // Tool configuration
    AllowedTools    []string
    DisallowedTools []string

    // Permission handling
    PermissionMode PermissionMode // default, acceptEdits, plan, bypassPermissions

    // Resource limits
    MaxTurns     int
    MaxBudgetUSD float64

    // Session management
    Resume               string
    ContinueConversation bool

    // MCP servers
    MCPServers    MCPServers
    MCPConfigPath string

    // Hooks
    Hooks *HookRegistry

    // File management
    WorkingDirectory string
    AddDirectories   []string

    // Environment
    Environment map[string]string
    CLIPath     string

    // Debug
    Debug   bool
    Verbose bool
}
```

## Structured Message Types

Parse typed messages from Claude:

```go
msg, err := claude.ParseMessage(jsonData)

switch m := msg.(type) {
case claude.AssistantMessage:
    for _, block := range m.Content {
        switch b := block.(type) {
        case claude.TextBlock:
            fmt.Println(b.Text)
        case claude.ToolUseBlock:
            fmt.Printf("Tool: %s(%v)\n", b.Name, b.Input)
        }
    }
case claude.ResultMessage:
    fmt.Printf("Cost: $%.4f, Turns: %d\n", m.TotalCostUSD, m.NumTurns)
}
```

## Error Handling

The SDK provides typed errors for precise error handling:

```go
resp, err := client.Query(ctx, req)
if err != nil {
    switch {
    case claude.IsCLINotFound(err):
        log.Fatal("Install Claude CLI: npm install -g @anthropic-ai/claude-code")
    case claude.IsProcessError(err):
        log.Fatal("CLI process failed")
    case claude.IsCLIConnectionError(err):
        log.Fatal("Connection error")
    default:
        log.Fatal(err)
    }
}
```

Error types:
- `ClaudeSDKError` - Base error type
- `CLINotFoundError` - Claude CLI not installed
- `CLIConnectionError` - Connection issues
- `ProcessError` - CLI process failures
- `CLIJSONDecodeError` - JSON parsing errors
- `MessageParseError` - Message parsing errors
- `SessionClosedError` - Using a closed session
- `ToolError` - Tool execution failures
- `HookError` - Hook execution failures

## Session Management

### Multi-turn Conversations

```go
client := claude.New(&claude.AgentOptions{
    PermissionMode: claude.PermissionModeBypassPermission,
    Interactive:    true,
})

resp, _ := client.Query(ctx, &claude.QueryRequest{
    Prompt:    "My name is Alice. Remember this.",
    SessionID: "my-session",
})

// Continue the conversation
session, _ := client.GetSession("my-session")
session.SendMessage("What is my name?") // Claude remembers "Alice"
```

### Concurrent Sessions

```go
var wg sync.WaitGroup

for _, prompt := range prompts {
    wg.Add(1)
    go func(p string) {
        defer wg.Done()
        resp, _ := client.Query(ctx, &claude.QueryRequest{Prompt: p})
        for msg := range resp.Messages {
            fmt.Println(msg.Content)
        }
    }(prompt)
}

wg.Wait()
```

## Examples

- **Basic Usage**: `examples/basic/main.go`
- **Concurrent Processing**: `examples/concurrent/main.go`
- **Interactive Session**: `examples/interactive/main.go`
- **Custom MCP Tools**: `examples/mcp/main.go`
- **Hooks**: `examples/hooks/main.go`

## Requirements

- Go 1.21 or later
- Claude CLI installed and accessible in PATH
- Valid Claude authentication

## CLI Discovery

The SDK automatically finds the Claude CLI in this order:
1. `CLAUDE_CLI_PATH` environment variable
2. Current working directory
3. System PATH
4. Common installation locations (npm, yarn, homebrew)

## Authentication

```bash
# Using Claude CLI setup
claude setup-token

# Or set environment variable
export ANTHROPIC_API_KEY=your-api-key
```

## Thread Safety

- Multiple goroutines can create sessions concurrently
- Sessions can be safely accessed from multiple goroutines
- Proper cleanup handled with context cancellation
- `client.Close()` safely shuts down all sessions

## License

MIT License
