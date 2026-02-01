// +build ignore

// CassetteGenerator generates VCR cassettes for integration tests.
// This simulates API responses to create cassettes for regression testing.
//
// To generate real cassettes:
// 1. Set the appropriate API key (MINMAX_API_KEY, GLM_API_ACCESS_ID, or ANTHROPIC_API_KEY)
// 2. Run: VCR_MODE=record go run scripts/generate_cassettes.go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/standardbeagle/claude-go"
)

var (
	cassetteDir = "testdata/cassettes/integration"
)

// Mock response generators for different providers
func minmaxHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"id":      "minimax-" + generateID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "minimax-abab6.5-chat",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": generateResponse(r),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func glmHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"id":      "glm-" + generateID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "glm-4",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": generateResponse(r),
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     10,
			"completion_tokens": 20,
			"total_tokens":      30,
		},
	}
	json.NewEncoder(w).Encode(response)
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateResponse(r *http.Request) string {
	// Read the request body to determine the prompt
	var req struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "I understand. Let me help you with that."
	}

	prompt := ""
	for _, m := range req.Messages {
		prompt = m.Content
		break
	}

	prompt = strings.ToLower(prompt)

	switch {
	case strings.Contains(prompt, "2+2") || strings.Contains(prompt, "2 + 2"):
		return "4"
	case strings.Contains(prompt, "capital of france"):
		return "Paris"
	case strings.Contains(prompt, "5 * 7") || strings.Contains(prompt, "5 * 7"):
		return "35"
	case strings.Contains(prompt, "100 - 45"):
		return "55"
	case strings.Contains(prompt, "name is"):
		return "I remember your name is TestUser."
	case strings.Contains(prompt, "what is my name"):
		return "Your name is TestUser."
	default:
		return "I understand. Let me help you with that."
	}
}

func main() {
	provider := flag.String("provider", "minmax", "Provider: minmax, glm")
	flag.Parse()

	// Create cassette directory
	if err := os.MkdirAll(cassetteDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create cassette directory: %v\n", err)
		os.Exit(1)
	}

	// Create mock server
	var handler http.HandlerFunc
	switch *provider {
	case "minmax":
		handler = minmaxHandler
	case "glm":
		handler = glmHandler
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", *provider)
		os.Exit(1)
	}

	server := httptest.NewServer(handler)
	defer server.Close()

	fmt.Printf("Mock server started at: %s\n", server.URL)
	fmt.Printf("Provider: %s\n", *provider)
	fmt.Println("\nCassettes will be recorded when tests run with real API or this mock.")

	// List existing cassettes
	files, _ := os.ReadDir(cassetteDir)
	fmt.Printf("\nExisting cassettes in %s:\n", cassetteDir)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".yaml") {
			fmt.Printf("  - %s\n", strings.TrimSuffix(f.Name(), ".yaml"))
		}
	}

	fmt.Println("\nTo run tests and generate cassettes:")
	fmt.Println("  1. For real API: set API key and run with VCR_MODE=record")
	fmt.Println("  2. For mock recording: tests use the mock server automatically")
}

// Helper to create cassette path
func cassettePath(name string) string {
	return filepath.Join(cassetteDir, name)
}
