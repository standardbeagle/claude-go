package claude

import (
	"context"
	"fmt"
	"testing"
)

func TestToolBuilder(t *testing.T) {
	tool := Tool("greet", "Greet a user").
		Param("name", "string", "The name to greet").
		Param("greeting", "string", "The greeting to use").
		ParamWithDefault("formal", "boolean", "Use formal greeting", false).
		Required("name").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			name := args["name"].(string)
			greeting := "Hello"
			if g, ok := args["greeting"].(string); ok {
				greeting = g
			}
			return fmt.Sprintf("%s, %s!", greeting, name), nil
		})

	if tool.Name != "greet" {
		t.Errorf("Expected name 'greet', got '%s'", tool.Name)
	}
	if tool.Description != "Greet a user" {
		t.Errorf("Expected description 'Greet a user', got '%s'", tool.Description)
	}

	// Check schema
	if tool.InputSchema.Type != "object" {
		t.Errorf("Expected schema type 'object', got '%s'", tool.InputSchema.Type)
	}
	if len(tool.InputSchema.Properties) != 3 {
		t.Errorf("Expected 3 properties, got %d", len(tool.InputSchema.Properties))
	}
	if len(tool.InputSchema.Required) != 1 {
		t.Errorf("Expected 1 required field, got %d", len(tool.InputSchema.Required))
	}
	if tool.InputSchema.Required[0] != "name" {
		t.Errorf("Expected required field 'name', got '%s'", tool.InputSchema.Required[0])
	}

	// Check handler works
	result, err := tool.Handler(context.Background(), map[string]interface{}{
		"name":     "Alice",
		"greeting": "Hi",
	})
	if err != nil {
		t.Fatalf("Handler failed: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != "Hi, Alice!" {
		t.Errorf("Expected 'Hi, Alice!', got '%s'", result.Content[0].Text)
	}
}

func TestSDKMCPServer(t *testing.T) {
	// Create tools
	addTool := Tool("add", "Add two numbers").
		Param("a", "number", "First number").
		Param("b", "number", "Second number").
		Required("a", "b").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return fmt.Sprintf("%g", a+b), nil
		})

	subtractTool := Tool("subtract", "Subtract two numbers").
		Param("a", "number", "First number").
		Param("b", "number", "Second number").
		Required("a", "b").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			return fmt.Sprintf("%g", a-b), nil
		})

	// Create server
	server := CreateSDKMCPServer("calculator", "1.0.0", addTool, subtractTool)

	if server.Name() != "calculator" {
		t.Errorf("Expected server name 'calculator', got '%s'", server.Name())
	}
	if server.Version() != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", server.Version())
	}

	// Check tools
	tools := server.ListTools()
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(tools))
	}

	// Get tool
	tool, ok := server.GetTool("add")
	if !ok {
		t.Fatal("Tool 'add' not found")
	}
	if tool.Name != "add" {
		t.Errorf("Expected tool name 'add', got '%s'", tool.Name)
	}

	// Execute tool
	result, err := server.ExecuteTool(context.Background(), "add", map[string]interface{}{
		"a": 5.0,
		"b": 3.0,
	})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}
	if result.Content[0].Text != "8" {
		t.Errorf("Expected result '8', got '%s'", result.Content[0].Text)
	}
}

func TestMCPServerManager(t *testing.T) {
	manager := NewMCPServerManager()

	// Create and register SDK server
	addTool := Tool("add", "Add numbers").
		Param("a", "number", "First").
		Param("b", "number", "Second").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			return "result", nil
		})

	server := CreateSDKMCPServer("math", "1.0", addTool)
	manager.RegisterSDKServer("math", server)

	// Register stdio server
	stdioCfg := MCPStdioServerConfig{
		Type:    MCPServerTypeStdio,
		Command: "my-mcp-server",
		Args:    []string{"--port", "8080"},
	}
	manager.Register("external", stdioCfg)

	// Check SDK server
	sdkServer, ok := manager.GetSDKServer("math")
	if !ok {
		t.Fatal("SDK server 'math' not found")
	}
	if sdkServer.Name() != "math" {
		t.Errorf("Expected server name 'math', got '%s'", sdkServer.Name())
	}

	// Check external server
	cfg, ok := manager.Get("external")
	if !ok {
		t.Fatal("External server not found")
	}
	if cfg.GetType() != MCPServerTypeStdio {
		t.Errorf("Expected type stdio, got %s", cfg.GetType())
	}

	// Check allowed tool names
	names := manager.AllowedToolNames()
	if len(names) != 1 {
		t.Fatalf("Expected 1 allowed tool name, got %d", len(names))
	}
	if names[0] != "mcp__math__add" {
		t.Errorf("Expected 'mcp__math__add', got '%s'", names[0])
	}

	// Execute SDK tool
	result, err := manager.ExecuteSDKTool(context.Background(), "math", "add", map[string]interface{}{
		"a": 1.0,
		"b": 2.0,
	})
	if err != nil {
		t.Fatalf("ExecuteSDKTool failed: %v", err)
	}
	if result.Content[0].Text != "result" {
		t.Errorf("Expected 'result', got '%s'", result.Content[0].Text)
	}
}

func TestTextResult(t *testing.T) {
	result := TextResult("Hello, world!")

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("Expected type 'text', got '%s'", result.Content[0].Type)
	}
	if result.Content[0].Text != "Hello, world!" {
		t.Errorf("Expected 'Hello, world!', got '%s'", result.Content[0].Text)
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
}

func TestErrorResult(t *testing.T) {
	result := ErrorResult(fmt.Errorf("something went wrong"))

	if !result.IsError {
		t.Error("Expected IsError to be true")
	}
	if result.Content[0].Text != "something went wrong" {
		t.Errorf("Expected error message, got '%s'", result.Content[0].Text)
	}
}

func TestImageResult(t *testing.T) {
	result := ImageResult("base64data", "image/png")

	if result.Content[0].Type != "image" {
		t.Errorf("Expected type 'image', got '%s'", result.Content[0].Type)
	}
	if result.Content[0].Data != "base64data" {
		t.Errorf("Expected data 'base64data', got '%s'", result.Content[0].Data)
	}
	if result.Content[0].MimeType != "image/png" {
		t.Errorf("Expected mimeType 'image/png', got '%s'", result.Content[0].MimeType)
	}
}

func TestMCPServerConfigTypes(t *testing.T) {
	// Test stdio config
	stdio := MCPStdioServerConfig{
		Type:    MCPServerTypeStdio,
		Command: "my-server",
	}
	if stdio.GetType() != MCPServerTypeStdio {
		t.Errorf("Expected stdio type")
	}

	// Test SSE config
	sse := MCPSSEServerConfig{
		Type: MCPServerTypeSSE,
		URL:  "http://localhost:8080",
	}
	if sse.GetType() != MCPServerTypeSSE {
		t.Errorf("Expected sse type")
	}

	// Test HTTP config
	http := MCPHTTPServerConfig{
		Type: MCPServerTypeHTTP,
		URL:  "http://localhost:8080",
	}
	if http.GetType() != MCPServerTypeHTTP {
		t.Errorf("Expected http type")
	}

	// Test SDK config
	server := NewSDKMCPServer("test", "1.0")
	sdk := server.ToConfig()
	if sdk.GetType() != MCPServerTypeSDK {
		t.Errorf("Expected sdk type")
	}
}
