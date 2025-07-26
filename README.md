# Claude Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/standardbeagle/claude-go.svg)](https://pkg.go.dev/github.com/standardbeagle/claude-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/standardbeagle/claude-go)](https://goreportcard.com/report/github.com/standardbeagle/claude-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A Go library providing goroutine interfaces to automate Claude through the CLI with concurrent session management and real-time streaming.

## Features

- **Concurrent Sessions**: Run multiple Claude sessions simultaneously with goroutines
- **JSON Streaming**: Real-time streaming communication using Claude's `stream-json` format
- **Session Management**: Create, manage, and reuse conversation sessions
- **Configuration**: Flexible options for models, permissions, tools, and environment
- **Error Handling**: Comprehensive error reporting and graceful shutdown
- **Examples**: Complete examples for basic, concurrent, and interactive usage

## Installation

```bash
go mod init your-project
go get github.com/standardbeagle/claude-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/standardbeagle/claude-go"
)

func main() {
    client := claude.New(&claude.Options{
        PermissionMode: "bypassPermissions",
        Model:         "sonnet",
    })
    defer client.Close()

    resp, err := client.Query(context.Background(), &claude.QueryRequest{
        Prompt: "Write a hello world function in Go",
    })
    if err != nil {
        log.Fatal(err)
    }

    for msg := range resp.Messages {
        fmt.Printf("%s: %s\n", msg.Type, msg.Content)
    }
}
```

## Configuration Options

```go
type Options struct {
    Model               string            // Model to use (e.g., "sonnet", "opus")
    MaxTurns           int               // Maximum conversation turns
    Debug              bool              // Enable debug mode
    Verbose            bool              // Enable verbose output
    PermissionMode     string            // Permission mode: "bypassPermissions", "acceptEdits", etc.
    AllowedTools       []string          // Tools to allow
    DisallowedTools    []string          // Tools to disallow
    SystemPrompt       string            // Additional system prompt
    MCPConfig          string            // MCP configuration
    AddDirectories     []string          // Additional directories for tool access
    Environment        map[string]string // Environment variables
    WorkingDirectory   string            // Working directory for Claude process
}
```

## Session Management

### Creating Sessions

```go
// Create a new session
resp, err := client.Query(ctx, &claude.QueryRequest{
    Prompt: "Hello Claude",
    SessionID: "my-session-id", // Optional: specify session ID
})

// Get existing session
session, exists := client.GetSession(sessionID)
if exists {
    session.SendMessage("Continue our conversation")
}
```

### Concurrent Sessions

```go
var wg sync.WaitGroup

for i, prompt := range prompts {
    wg.Add(1)
    go func(p string) {
        defer wg.Done()
        
        resp, err := client.Query(ctx, &claude.QueryRequest{Prompt: p})
        if err != nil {
            return
        }
        
        for msg := range resp.Messages {
            fmt.Printf("Response: %s\n", msg.Content)
        }
    }(prompt)
}

wg.Wait()
```

## Message Types

Messages received from Claude have the following structure:

```go
type Message struct {
    Type      string          `json:"type"`      // Message type
    Content   string          `json:"content"`   // Message content
    Role      string          `json:"role"`      // Message role (user/assistant)
    Timestamp time.Time       `json:"timestamp"` // When received
    Metadata  json.RawMessage `json:"metadata"`  // Additional metadata
    Error     string          `json:"error"`     // Error message if any
}
```

Common message types:
- `content`: Main response content
- `tool_call`: Tool execution
- `thinking`: Claude's reasoning process
- `error`: Error messages

## Examples

### Basic Usage
See `examples/basic/main.go` for a simple query example.

### Concurrent Processing
See `examples/concurrent/main.go` for running multiple sessions simultaneously.

### Interactive Session
See `examples/interactive/main.go` for a command-line interface.

## Error Handling

The library provides two channels for handling responses and errors:

```go
resp, err := client.Query(ctx, req)
if err != nil {
    log.Fatal(err)
}

// Handle errors from the Claude process
go func() {
    for err := range resp.Errors {
        log.Printf("Claude error: %v", err)
    }
}()

// Handle messages
for msg := range resp.Messages {
    if msg.Error != "" {
        log.Printf("Message error: %s", msg.Error)
        continue
    }
    fmt.Printf("Content: %s\n", msg.Content)
}
```

## Requirements

- Go 1.21 or later
- Claude CLI installed and accessible in PATH
- Valid Claude authentication (API key or subscription)

## Authentication

The library uses the Claude CLI's existing authentication. Ensure Claude CLI is properly authenticated:

```bash
claude setup-token
```

Or set environment variables:
```bash
export ANTHROPIC_API_KEY=your-api-key
```

## Thread Safety

The library is designed to be thread-safe:
- Multiple goroutines can create sessions concurrently
- Sessions can be safely accessed from multiple goroutines
- Proper cleanup is handled automatically

## License

MIT License