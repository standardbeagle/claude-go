package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/standardbeagle/claude-go"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_runner.go <test_type>")
		fmt.Println("Test types: basic, multiturn, files, concurrent, complex")
		os.Exit(1)
	}

	testType := os.Args[1]

	switch testType {
	case "basic":
		runBasicTest()
	case "multiturn":
		runMultiTurnTest()
	case "files":
		runFileTest()
	case "concurrent":
		runConcurrentTest()
	case "complex":
		runComplexTest()
	default:
		fmt.Printf("Unknown test type: %s\n", testType)
		os.Exit(1)
	}
}

func runBasicTest() {
	fmt.Println("Running basic test...")

	client := claude.New(&claude.Options{
		PermissionMode: "bypassPermissions",
		Debug:          false,
		Verbose:        false,
		Interactive:    false, // Use -p mode for simple single query
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: "What is the capital of France? Just give me the city name.",
	})
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	go func() {
		for err := range resp.Errors {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	fmt.Printf("Session ID: %s\n", resp.SessionID)

	timeout := time.After(20 * time.Second)
	for {
		select {
		case msg, ok := <-resp.Messages:
			if !ok {
				fmt.Println("Messages channel closed")
				return
			}

			fmt.Printf("[%s] %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.Type,
				msg.Content)

			if msg.Type == "content" && msg.Content != "" {
				fmt.Println("✓ Basic test completed successfully")
				return
			}

		case <-timeout:
			fmt.Println("✗ Test timed out")
			return
		}
	}
}

func runMultiTurnTest() {
	fmt.Println("Running multi-turn conversation test...")

	client := claude.New(&claude.Options{
		PermissionMode: "bypassPermissions",
		Debug:          false,
		Verbose:        false,
		Interactive:    true, // Use interactive mode for multi-turn
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// First message
	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: "My favorite color is blue. Remember this.",
		// Let the library generate a UUID
	})
	if err != nil {
		log.Fatalf("Initial query failed: %v", err)
	}

	session, exists := client.GetSession(resp.SessionID)
	if !exists {
		log.Fatal("Session not found")
	}

	go func() {
		for err := range resp.Errors {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	fmt.Printf("Session ID: %s\n", resp.SessionID)

	// Wait for first response
	waitForContentMessage(resp.Messages, "First turn")

	// Second turn
	fmt.Println("\nSending second message...")
	err = session.SendMessage("What is my favorite color?")
	if err != nil {
		log.Fatalf("Failed to send second message: %v", err)
	}

	response := waitForContentMessage(resp.Messages, "Second turn")
	if response != "" {
		fmt.Printf("✓ Multi-turn test completed. Claude remembered: %s\n", response)
	}

	session.Close()
}

func runFileTest() {
	fmt.Println("Running file operations test...")

	tempDir, err := os.MkdirTemp("", "claude-test-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("Using temp directory: %s\n", tempDir)

	client := claude.New(&claude.Options{
		PermissionMode:   "bypassPermissions",
		Debug:            false,
		Verbose:          false,
		Interactive:      true, // Use interactive mode for file operations
		WorkingDirectory: tempDir,
		AddDirectories:   []string{tempDir},
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Create file
	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: fmt.Sprintf("Create a file called 'hello.txt' in %s with the content 'Hello World from test!'", tempDir),
	})
	if err != nil {
		log.Fatalf("File creation query failed: %v", err)
	}

	go func() {
		for err := range resp.Errors {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	fmt.Printf("Session ID: %s\n", resp.SessionID)

	// Wait for file creation
	waitForContentMessage(resp.Messages, "File creation")

	// Check if file exists
	filePath := filepath.Join(tempDir, "hello.txt")
	time.Sleep(2 * time.Second)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("✗ File was not created")
		return
	}

	fmt.Println("✓ File created successfully")

	// Read file
	session, exists := client.GetSession(resp.SessionID)
	if !exists {
		log.Fatal("Session not found")
	}

	fmt.Println("\nReading file...")
	err = session.SendMessage(fmt.Sprintf("Read the file %s and tell me what it contains", filePath))
	if err != nil {
		log.Fatalf("Failed to send read message: %v", err)
	}

	readResponse := waitForContentMessage(resp.Messages, "File reading")
	if readResponse != "" {
		fmt.Printf("✓ File read test completed: %s\n", readResponse)
	}

	session.Close()
}

func runConcurrentTest() {
	fmt.Println("Running concurrent sessions test...")

	client := claude.New(&claude.Options{
		PermissionMode: "bypassPermissions",
		Debug:          false,
		Verbose:        false,
		Interactive:    false, // Use -p mode for concurrent single queries
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	questions := []string{
		"What is 10 + 15?",
		"What is the square root of 64?",
		"What is 7 * 8?",
	}

	for i, question := range questions {
		go func(id int, q string) {
			resp, err := client.Query(ctx, &claude.QueryRequest{
				Prompt: q,
			})
			if err != nil {
				fmt.Printf("Session %d failed: %v\n", id, err)
				return
			}

			go func() {
				for err := range resp.Errors {
					fmt.Printf("Session %d error: %v\n", id, err)
				}
			}()

			fmt.Printf("Session %d started: %s\n", id, resp.SessionID)

			response := waitForContentMessage(resp.Messages, fmt.Sprintf("Session %d", id))
			if response != "" {
				fmt.Printf("✓ Session %d completed: %s\n", id, response)
			}
		}(i+1, question)
	}

	time.Sleep(45 * time.Second)
	fmt.Println("✓ Concurrent test completed")
}

func runComplexTest() {
	fmt.Println("Running complex multi-turn with files test...")

	tempDir, err := os.MkdirTemp("", "claude-complex-*")
	if err != nil {
		log.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Printf("Using temp directory: %s\n", tempDir)

	client := claude.New(&claude.Options{
		PermissionMode:   "bypassPermissions",
		Debug:            false,
		Verbose:          false,
		Interactive:      true, // Use interactive mode for complex operations
		WorkingDirectory: tempDir,
		AddDirectories:   []string{tempDir},
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := client.Query(ctx, &claude.QueryRequest{
		Prompt: "I want to create some files and work with them. Are you ready to help?",
	})
	if err != nil {
		log.Fatalf("Initial query failed: %v", err)
	}

	session, exists := client.GetSession(resp.SessionID)
	if !exists {
		log.Fatal("Session not found")
	}

	go func() {
		for err := range resp.Errors {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	fmt.Printf("Session ID: %s\n", resp.SessionID)

	// Initial response
	waitForContentMessage(resp.Messages, "Initial")

	// Create Python file
	fmt.Println("\nCreating Python file...")
	err = session.SendMessage(fmt.Sprintf("Create a Python file called 'math_utils.py' in %s with functions for add, subtract, and multiply", tempDir))
	if err != nil {
		log.Fatalf("Failed to send Python file creation message: %v", err)
	}
	waitForContentMessage(resp.Messages, "Python file creation")

	// Create README
	fmt.Println("\nCreating README...")
	err = session.SendMessage(fmt.Sprintf("Create a README.md file in %s that describes the math_utils.py file", tempDir))
	if err != nil {
		log.Fatalf("Failed to send README creation message: %v", err)
	}
	waitForContentMessage(resp.Messages, "README creation")

	// Read and analyze
	fmt.Println("\nReading files...")
	err = session.SendMessage("Read both files and tell me what they contain")
	if err != nil {
		log.Fatalf("Failed to send read message: %v", err)
	}
	readResponse := waitForContentMessage(resp.Messages, "File reading")

	// Verify files exist
	pythonPath := filepath.Join(tempDir, "math_utils.py")
	readmePath := filepath.Join(tempDir, "README.md")

	time.Sleep(2 * time.Second)

	filesCreated := 0
	if _, err := os.Stat(pythonPath); err == nil {
		filesCreated++
		fmt.Println("✓ Python file created")
	}
	if _, err := os.Stat(readmePath); err == nil {
		filesCreated++
		fmt.Println("✓ README file created")
	}

	if filesCreated == 2 {
		fmt.Printf("✓ Complex test completed successfully. Files created and read: %s\n", readResponse[:100]+"...")
	} else {
		fmt.Printf("✗ Complex test partially failed. Only %d/2 files created\n", filesCreated)
	}

	session.Close()
}

func waitForContentMessage(messages <-chan *claude.Message, context string) string {
	timeout := time.After(30 * time.Second)

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				fmt.Printf("Channel closed for %s\n", context)
				return ""
			}

			fmt.Printf("[%s] %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.Type,
				truncate(msg.Content, 100))

			if msg.Type == "content" && msg.Content != "" {
				return msg.Content
			}

		case <-timeout:
			fmt.Printf("Timeout waiting for %s\n", context)
			return ""
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
