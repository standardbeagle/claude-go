// +build ignore

// This script is used to record VCR cassettes for integration tests.
// Run with: go run scripts/record_cassettes.go [--provider minmax|glm]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/standardbeagle/claude-go"
)

const (
	// Default cassette directory
	defaultCassetteDir = "testdata/cassettes/integration"
)

var (
	provider   = flag.String("provider", "anthropic", "API provider (anthropic, minmax, glm)")
	model      = flag.String("model", "sonnet", "Model to use (haiku, sonnet, opus)")
	cassette   = flag.String("cassette", "", "Cassette name (required)")
	apiKey     = flag.String("api-key", "", "API key (required)")
	baseURL    = flag.String("base-url", "", "Base URL for API")
	timeout    = flag.Duration("timeout", 60*time.Second, "Request timeout")
	prompt     = flag.String("prompt", "What is 2+2? Just give me the number.", "Prompt to send")
)

func main() {
	flag.Parse()

	if *cassette == "" {
		fmt.Fprintf(os.Stderr, "Error: --cassette is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if *apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: --api-key is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Determine model based on provider
	modelName := getModelName(*provider, *model)

	// Determine base URL
	baseURLStr := getBaseURL(*provider, *baseURL)

	fmt.Printf("Recording cassette: %s\n", *cassette)
	fmt.Printf("Provider: %s\n", *provider)
	fmt.Printf("Model: %s\n", modelName)
	fmt.Printf("Base URL: %s\n", baseURLStr)

	// Create client with provider settings
	opts := &claude.AgentOptions{
		Model:        modelName,
		BaseURL:      baseURLStr,
		APIKey:       *apiKey,
		PermissionMode: claude.PermissionModeBypassPermission,
		Interactive:  false,
		TimeoutSecs:  int(timeout.Seconds()),
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Run the query and collect messages
	messages, err := claude.Query(ctx, *prompt, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nReceived %d messages:\n", len(messages))
	for i, msg := range messages {
		fmt.Printf("  [%d] %s\n", i+1, msg.Summary())
	}

	fmt.Println("\nCassette recording complete!")
	fmt.Printf("Cassette saved to: %s/%s.yaml\n", defaultCassetteDir, *cassette)
}

func getModelName(provider, model string) string {
	switch provider {
	case "minmax":
		switch model {
		case "haiku":
			return "minimax-abab6.5s-chat"
		case "sonnet":
			return "minimax-abab6.5-chat"
		case "opus":
			return "minimax-abab6.5s-chat"
		default:
			return "minimax-abab6.5-chat"
		}
	case "glm":
		switch model {
		case "haiku":
			return "glm-3-turbo"
		case "sonnet":
			return "glm-4"
		case "opus":
			return "glm-4-plus"
		default:
			return "glm-4"
		}
	case "anthropic":
		fallthrough
	default:
		switch model {
		case "haiku":
			return "claude-3-5-haiku-20241022"
		case "sonnet":
			return "claude-sonnet-4-20250514"
		case "opus":
			return "claude-opus-4-20250514"
		default:
			return "claude-sonnet-4-20250514"
		}
	}
}

func getBaseURL(provider, baseURL string) string {
	if baseURL != "" {
		return baseURL
	}

	switch provider {
	case "minmax":
		return "https://api.minimax.chat/v1/openai/chat/completions"
	case "glm":
		return "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	case "anthropic":
		fallthrough
	default:
		return "https://api.anthropic.com/v1"
	}
}

func init() {
	// Ensure cassette directory exists
	_ = os.MkdirAll(defaultCassetteDir, 0755)
}

// Save cassette path for reference
func cassettePath(name string) string {
	return filepath.Join(defaultCassetteDir, name)
}
