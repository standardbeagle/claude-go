// Example: Custom MCP Tools
//
// This example demonstrates how to create in-process MCP tools
// that Claude can use during conversations.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	claude "github.com/standardbeagle/claude-go"
)

func main() {
	// Create calculator tools using the builder pattern
	addTool := claude.Tool("add", "Add two numbers together").
		Param("a", "number", "First number to add").
		Param("b", "number", "Second number to add").
		Required("a", "b").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			result := a + b
			return fmt.Sprintf("%.2f + %.2f = %.2f", a, b, result), nil
		})

	subtractTool := claude.Tool("subtract", "Subtract one number from another").
		Param("a", "number", "Number to subtract from").
		Param("b", "number", "Number to subtract").
		Required("a", "b").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			result := a - b
			return fmt.Sprintf("%.2f - %.2f = %.2f", a, b, result), nil
		})

	multiplyTool := claude.Tool("multiply", "Multiply two numbers").
		Param("a", "number", "First number").
		Param("b", "number", "Second number").
		Required("a", "b").
		HandlerFunc(func(ctx context.Context, args map[string]interface{}) (string, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			result := a * b
			return fmt.Sprintf("%.2f × %.2f = %.2f", a, b, result), nil
		})

	divideTool := claude.Tool("divide", "Divide one number by another").
		Param("a", "number", "Dividend (number to divide)").
		Param("b", "number", "Divisor (number to divide by)").
		Required("a", "b").
		Handler(func(ctx context.Context, args map[string]interface{}) (*claude.ToolResult, error) {
			a := args["a"].(float64)
			b := args["b"].(float64)
			if b == 0 {
				return claude.ErrorResult(fmt.Errorf("cannot divide by zero")), nil
			}
			result := a / b
			return claude.TextResult(fmt.Sprintf("%.2f ÷ %.2f = %.2f", a, b, result)), nil
		})

	sqrtTool := claude.Tool("sqrt", "Calculate the square root of a number").
		Param("n", "number", "Number to find square root of").
		Required("n").
		Handler(func(ctx context.Context, args map[string]interface{}) (*claude.ToolResult, error) {
			n := args["n"].(float64)
			if n < 0 {
				return claude.ErrorResult(fmt.Errorf("cannot calculate square root of negative number")), nil
			}
			result := math.Sqrt(n)
			return claude.TextResult(fmt.Sprintf("√%.2f = %.4f", n, result)), nil
		})

	// Create an SDK MCP server with all calculator tools
	calculator := claude.CreateSDKMCPServer("calculator", "1.0.0",
		addTool, subtractTool, multiplyTool, divideTool, sqrtTool)

	fmt.Println("Calculator MCP Server created with tools:")
	for _, tool := range calculator.ListTools() {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	// Create client and register the MCP server
	client := claude.New(&claude.AgentOptions{
		PermissionMode: claude.PermissionModeBypassPermission,
		Interactive:    true,
		AllowedTools: []string{
			"mcp__calculator__add",
			"mcp__calculator__subtract",
			"mcp__calculator__multiply",
			"mcp__calculator__divide",
			"mcp__calculator__sqrt",
		},
	})
	defer client.Close()

	client.RegisterSDKMCPServer("calculator", calculator)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ask Claude to use the calculator
	fmt.Println("Asking Claude to perform calculations...")
	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: "Please calculate (15 + 7) * 3, then find the square root of the result. Show your work step by step using the calculator tools.",
	})
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Handle errors in the background
	go func() {
		for err := range resp.Errors {
			log.Printf("Error: %v", err)
		}
	}()

	// Print response
	for msg := range resp.Messages {
		if msg.Type == "content" || msg.Type == "text" {
			fmt.Println(msg.Content)
		}
	}
}
