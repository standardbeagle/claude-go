package claude

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBasicQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Read GLM config from environment (same as bashrc alias)
	baseURL := os.Getenv("ANTHROPIC_BASE_URL")
	authToken := os.Getenv("ANTHROPIC_AUTH_TOKEN")
	model := os.Getenv("ANTHROPIC_MODEL")

	if baseURL == "" || authToken == "" {
		t.Skip("Set ANTHROPIC_BASE_URL and ANTHROPIC_AUTH_TOKEN to run this test")
	}

	client := New(&Options{
		PermissionMode: "bypassPermissions",
		Debug:          false,
		Verbose:        false,
		BaseURL:        baseURL,
		AccessToken:    authToken,
		Model:          model,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Query(ctx, &QueryRequest{
		Prompt: "What is 2+2? Just give me the number.",
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	var messages []*Message
	var errors []error

	go func() {
		for err := range resp.Errors {
			errors = append(errors, err)
		}
	}()

	timeout := time.After(20 * time.Second)
	messageReceived := false

	for {
		select {
		case msg, ok := <-resp.Messages:
			if !ok {
				if !messageReceived {
					t.Fatal("No messages received before channel closed")
				}
				goto checkResult
			}

			messages = append(messages, msg)
			if msg.Type == "content" || msg.Type == "text" {
				messageReceived = true
				t.Logf("Received message: %s", msg.Content)
				if strings.Contains(msg.Content, "4") {
					goto checkResult
				}
			}

		case <-timeout:
			t.Fatal("Test timed out waiting for response")
		}
	}

checkResult:
	if len(errors) > 0 {
		t.Logf("Errors received: %v", errors)
	}

	if !messageReceived {
		t.Fatal("No content message received")
	}

	hasCorrectAnswer := false
	for _, msg := range messages {
		if strings.Contains(msg.Content, "4") {
			hasCorrectAnswer = true
			break
		}
	}

	if !hasCorrectAnswer {
		t.Fatal("Response did not contain expected answer '4'")
	}
}

func TestMultiTurnConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	client := New(&Options{
		PermissionMode: "bypassPermissions",
		Debug:          false,
		Verbose:        false,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start conversation
	resp, err := client.Query(ctx, &QueryRequest{
		Prompt:    "My name is TestUser. Remember this for our conversation.",
		SessionID: "test-multi-turn",
	})
	if err != nil {
		t.Fatalf("Initial query failed: %v", err)
	}

	session, exists := client.GetSession("test-multi-turn")
	if !exists {
		t.Fatal("Session not found after creation")
	}

	// Collect errors
	go func() {
		for err := range resp.Errors {
			t.Logf("Session error: %v", err)
		}
	}()

	// Wait for initial response
	waitForResponse(t, resp.Messages, 15*time.Second)

	// Second turn
	err = session.SendMessage("What is my name?")
	if err != nil {
		t.Fatalf("Failed to send second message: %v", err)
	}

	nameResponse := waitForResponse(t, resp.Messages, 15*time.Second)
	if !strings.Contains(strings.ToLower(nameResponse), "testuser") {
		t.Errorf("Claude did not remember the name. Response: %s", nameResponse)
	}

	// Third turn
	err = session.SendMessage("What is 5 * 7?")
	if err != nil {
		t.Fatalf("Failed to send third message: %v", err)
	}

	mathResponse := waitForResponse(t, resp.Messages, 15*time.Second)
	if !strings.Contains(mathResponse, "35") {
		t.Errorf("Claude did not answer math question correctly. Response: %s", mathResponse)
	}

	session.Close()
}

func TestFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "claude-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := New(&Options{
		PermissionMode:   "bypassPermissions",
		Debug:            false,
		Verbose:          false,
		WorkingDirectory: tempDir,
		AddDirectories:   []string{tempDir},
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Test file creation
	resp, err := client.Query(ctx, &QueryRequest{
		Prompt: fmt.Sprintf("Create a file called 'test.txt' in the directory %s with the content 'Hello from Claude Go test!'", tempDir),
	})
	if err != nil {
		t.Fatalf("File creation query failed: %v", err)
	}

	// Collect errors
	go func() {
		for err := range resp.Errors {
			t.Logf("File creation error: %v", err)
		}
	}()

	// Wait for file creation response
	createResponse := waitForResponse(t, resp.Messages, 30*time.Second)
	t.Logf("File creation response: %s", createResponse)

	// Verify file was created
	testFilePath := filepath.Join(tempDir, "test.txt")
	time.Sleep(2 * time.Second) // Give file system time to sync

	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Get session for follow-up
	session, exists := client.GetSession(resp.SessionID)
	if !exists {
		t.Fatal("Session not found")
	}

	// Test file reading
	err = session.SendMessage(fmt.Sprintf("Read the contents of the file %s and tell me what it says", testFilePath))
	if err != nil {
		t.Fatalf("Failed to send file read message: %v", err)
	}

	readResponse := waitForResponse(t, resp.Messages, 30*time.Second)
	t.Logf("File read response: %s", readResponse)

	if !strings.Contains(readResponse, "Hello from Claude Go test!") {
		t.Errorf("File content not found in response: %s", readResponse)
	}

	// Test file modification
	err = session.SendMessage(fmt.Sprintf("Append the text '\\nSecond line added by test.' to the file %s", testFilePath))
	if err != nil {
		t.Fatalf("Failed to send file modify message: %v", err)
	}

	modifyResponse := waitForResponse(t, resp.Messages, 30*time.Second)
	t.Logf("File modify response: %s", modifyResponse)

	// Verify modification
	content, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if !strings.Contains(string(content), "Second line added by test.") {
		t.Errorf("File was not modified correctly. Content: %s", string(content))
	}

	session.Close()
}

func TestConcurrentFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "claude-concurrent-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := New(&Options{
		PermissionMode:   "bypassPermissions",
		Debug:            false,
		Verbose:          false,
		WorkingDirectory: tempDir,
		AddDirectories:   []string{tempDir},
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	numSessions := 3

	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(sessionNum int) {
			defer wg.Done()

			fileName := fmt.Sprintf("concurrent_test_%d.txt", sessionNum)
			content := fmt.Sprintf("Content from session %d", sessionNum)

			resp, err := client.Query(ctx, &QueryRequest{
				Prompt: fmt.Sprintf("Create a file called '%s' with the content '%s' in directory %s", fileName, content, tempDir),
			})
			if err != nil {
				t.Errorf("Session %d query failed: %v", sessionNum, err)
				return
			}

			// Collect errors
			go func() {
				for err := range resp.Errors {
					t.Logf("Session %d error: %v", sessionNum, err)
				}
			}()

			// Wait for file creation
			response := waitForResponseWithTimeout(t, resp.Messages, 45*time.Second, fmt.Sprintf("Session %d", sessionNum))
			t.Logf("Session %d creation response: %s", sessionNum, response)

			// Verify file was created
			filePath := filepath.Join(tempDir, fileName)
			time.Sleep(2 * time.Second) // Give file system time to sync

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("Session %d: File %s was not created", sessionNum, fileName)
				return
			}

			// Read back the file
			session, exists := client.GetSession(resp.SessionID)
			if !exists {
				t.Errorf("Session %d not found", sessionNum)
				return
			}

			err = session.SendMessage(fmt.Sprintf("Read the file %s and tell me its content", filePath))
			if err != nil {
				t.Errorf("Session %d failed to send read message: %v", sessionNum, err)
				session.Close()
				return
			}

			readResponse := waitForResponseWithTimeout(t, resp.Messages, 30*time.Second, fmt.Sprintf("Session %d read", sessionNum))

			if !strings.Contains(readResponse, content) {
				t.Errorf("Session %d: Expected content '%s' not found in response: %s", sessionNum, content, readResponse)
			}

			session.Close()
		}(i)
	}

	wg.Wait()

	// Verify all files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	if len(files) != numSessions {
		t.Errorf("Expected %d files, found %d", numSessions, len(files))
	}
}

func TestComplexMultiTurnWithFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	tempDir, err := os.MkdirTemp("", "claude-complex-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	client := New(&Options{
		PermissionMode:   "bypassPermissions",
		Debug:            false,
		Verbose:          false,
		WorkingDirectory: tempDir,
		AddDirectories:   []string{tempDir},
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	resp, err := client.Query(ctx, &QueryRequest{
		Prompt:    "I'm going to ask you to create several files and then work with them. Are you ready?",
		SessionID: "complex-session",
	})
	if err != nil {
		t.Fatalf("Initial query failed: %v", err)
	}

	session, exists := client.GetSession("complex-session")
	if !exists {
		t.Fatal("Session not found")
	}

	go func() {
		for err := range resp.Errors {
			t.Logf("Session error: %v", err)
		}
	}()

	// Step 1: Initial response
	waitForResponse(t, resp.Messages, 20*time.Second)

	// Step 2: Create a Go file
	err = session.SendMessage(fmt.Sprintf("Create a Go file called 'calculator.go' in %s with a function that adds two integers", tempDir))
	if err != nil {
		t.Fatalf("Failed to send Go file creation message: %v", err)
	}
	waitForResponse(t, resp.Messages, 30*time.Second)

	// Step 3: Create a test file
	err = session.SendMessage(fmt.Sprintf("Now create a test file called 'calculator_test.go' in %s that tests the add function", tempDir))
	if err != nil {
		t.Fatalf("Failed to send test file creation message: %v", err)
	}
	waitForResponse(t, resp.Messages, 30*time.Second)

	// Step 4: Read and verify files
	err = session.SendMessage("Read both files and tell me what functions they contain")
	if err != nil {
		t.Fatalf("Failed to send read message: %v", err)
	}
	waitForResponse(t, resp.Messages, 30*time.Second)

	// Verify files exist
	calculatorPath := filepath.Join(tempDir, "calculator.go")
	testPath := filepath.Join(tempDir, "calculator_test.go")

	time.Sleep(2 * time.Second)

	if _, err := os.Stat(calculatorPath); os.IsNotExist(err) {
		t.Fatal("calculator.go was not created")
	}

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatal("calculator_test.go was not created")
	}

	// Step 5: Modify the calculator
	err = session.SendMessage("Add a subtract function to calculator.go")
	if err != nil {
		t.Fatalf("Failed to send modify message: %v", err)
	}
	waitForResponse(t, resp.Messages, 30*time.Second)

	// Step 6: Final verification
	err = session.SendMessage("Read calculator.go again and list all the functions it now contains")
	if err != nil {
		t.Fatalf("Failed to send final read message: %v", err)
	}
	finalResponse := waitForResponse(t, resp.Messages, 30*time.Second)

	if !strings.Contains(strings.ToLower(finalResponse), "subtract") {
		t.Errorf("Subtract function not found in final response: %s", finalResponse)
	}

	session.Close()
}

// Helper function to wait for a response message
func waitForResponse(t *testing.T, messages <-chan *Message, timeout time.Duration) string {
	return waitForResponseWithTimeout(t, messages, timeout, "")
}

func waitForResponseWithTimeout(t *testing.T, messages <-chan *Message, timeout time.Duration, context string) string {
	timer := time.After(timeout)
	var content strings.Builder

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				if content.Len() > 0 {
					return content.String()
				}
				t.Fatalf("%s: Message channel closed without receiving content", context)
			}

			if msg.Type == "content" || msg.Type == "text" {
				content.WriteString(msg.Content)
				if strings.TrimSpace(msg.Content) != "" {
					return content.String()
				}
			}

		case <-timer:
			if content.Len() > 0 {
				return content.String()
			}
			t.Fatalf("%s: Timeout waiting for response", context)
		}
	}
}
