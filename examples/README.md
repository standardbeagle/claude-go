# Examples

This directory contains complete examples demonstrating different usage patterns of the claude-go library.

## Running Examples

Make sure Claude CLI is in your PATH and authenticated:

```bash
# Check Claude CLI is available
claude --version

# Authenticate if needed
claude setup-token
```

Then run examples:

```bash
# Build all examples
make examples

# Run specific examples
make run-basic
make run-concurrent  
make run-interactive
```

## Example Descriptions

### [basic/](basic/) - Simple Single Query
Demonstrates the simplest usage pattern with a single query and response.

**Features:**
- Single query execution
- Basic error handling
- Timeout management
- Non-interactive mode (`-p` flag)

**Use case:** Quick one-off queries, scripting, batch processing

### [concurrent/](concurrent/) - Multiple Concurrent Sessions
Shows how to run multiple Claude sessions simultaneously using goroutines.

**Features:**
- Concurrent session management
- WaitGroup synchronization
- Independent error handling per session
- Parallel processing of different prompts

**Use case:** Batch processing, parallel analysis, concurrent automation

### [interactive/](interactive/) - Command Line Interface
A full interactive CLI that maintains conversation state across multiple turns.

**Features:**
- Multi-turn conversations
- Session persistence
- Command-line interface
- Interactive session management (`new`, `quit` commands)

**Use case:** Interactive development, debugging sessions, conversational workflows

## Advanced Patterns

### File Operations
All examples can be configured for file operations by adding:

```go
client := claude.New(&claude.Options{
    PermissionMode:   "bypassPermissions",
    Interactive:      true,
    WorkingDirectory: "/path/to/project", 
    AddDirectories:   []string{"/path/to/project"},
})
```

### Custom Configuration
Examples can be customized with different options:

```go
client := claude.New(&claude.Options{
    Model:            "sonnet",           // or "opus"
    PermissionMode:   "acceptEdits",      // or "bypassPermissions" 
    AllowedTools:     []string{"Bash", "Edit"},
    DisallowedTools:  []string{"WebFetch"},
    SystemPrompt:     "You are a helpful assistant.",
    Environment:      map[string]string{"VAR": "value"},
})
```

### Error Handling Patterns
Robust error handling across all examples:

```go
// Handle creation errors
resp, err := client.Query(ctx, req)
if err != nil {
    log.Fatalf("Query failed: %v", err)
}

// Handle runtime errors
go func() {
    for err := range resp.Errors {
        log.Printf("Runtime error: %v", err)  
    }
}()

// Handle message errors
for msg := range resp.Messages {
    if msg.Error != "" {
        log.Printf("Message error: %s", msg.Error)
        continue
    }
    // Process successful message
}
```

## Testing Examples

The examples are tested as part of the main test suite:

```bash
# Test examples build correctly
make examples

# Run integration tests that use examples
make test-integration
```

## Extending Examples

Feel free to copy and modify these examples for your specific use case. Common extensions:

1. **Database Integration**: Add database queries/updates based on Claude responses
2. **Web API**: Wrap examples in HTTP handlers for web service integration  
3. **File Processing**: Extend file operations for document processing workflows
4. **Monitoring**: Add metrics, logging, and observability
5. **Configuration**: Add YAML/JSON configuration file support