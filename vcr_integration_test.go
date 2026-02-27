package claude

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// IntegrationTestConfig holds configuration for integration tests
type IntegrationTestConfig struct {
	CassetteName  string
	Provider      string
	Model         string
	Prompt        string
	Timeout       time.Duration
	APIKey        string
	BaseURL       string
	SkipRecording bool
}

// IntegrationTestHelper provides VCR-based integration testing
type IntegrationTestHelper struct {
	vcr      *VCRServer
	config   IntegrationTestConfig
	opts     *AgentOptions
	cleanup  func()
}

// NewIntegrationTestHelper creates a new integration test helper
func NewIntegrationTestHelper(config IntegrationTestConfig) *IntegrationTestHelper {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Model == "" {
		config.Model = "sonnet"
	}
	if config.Prompt == "" {
		config.Prompt = "What is 2+2? Just give me the number."
	}

	return &IntegrationTestHelper{
		config: config,
	}
}

// Setup prepares the test environment
func (h *IntegrationTestHelper) Setup(t *testing.T) {
	t.Helper()

	// Create cassette path
	cassetteDir := filepath.Join("testdata", "cassettes", "integration")
	if err := os.MkdirAll(cassetteDir, 0755); err != nil {
		t.Fatalf("Failed to create cassette directory: %v", err)
	}

	cassettePath := filepath.Join(cassetteDir, h.config.CassetteName)

	// Determine mode
	mode := VCRModeAuto
	if h.config.SkipRecording {
		mode = VCRModeReplay
	}

	// Determine base URL
	baseURL := h.config.BaseURL
	if baseURL == "" {
		baseURL = GetProviderBaseURL(h.config.Provider)
	}

	// Create VCR server
	h.vcr = NewVCRServer(cassettePath,
		WithVCRMode(mode),
		WithTargetURL(baseURL),
	)

	// Start VCR
	baseURL, err := h.vcr.Start()
	if err != nil {
		t.Fatalf("Failed to start VCR: %v", err)
	}

	// Get API key
	apiKey := h.config.APIKey
	if apiKey == "" {
		apiKey = GetProviderAPIKey(h.config.Provider)
	}

	// Create client options
	h.opts = &AgentOptions{
		Model:          GetProviderModel(h.config.Provider, h.config.Model),
		BaseURL:        baseURL,
		APIKey:         apiKey,
		PermissionMode: PermissionModeBypassPermission,
		Interactive:    false,
		TimeoutSecs:    int(h.config.Timeout.Seconds()),
	}

	// Save original env and set up cleanup
	h.cleanup = func() {
		h.vcr.Close()
	}
}

// Close cleans up resources
func (h *IntegrationTestHelper) Close() {
	if h.cleanup != nil {
		h.cleanup()
	}
}

// Client returns a new client configured for the test
func (h *IntegrationTestHelper) Client() *Client {
	return New(h.opts)
}

// Query runs a query and returns messages
func (h *IntegrationTestHelper) Query(t *testing.T) []MessageType {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), h.config.Timeout)
	defer cancel()

	messages, err := Query(ctx, h.config.Prompt, h.opts)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	return messages
}

// QueryWithContext runs a query with custom prompt and timeout
func (h *IntegrationTestHelper) QueryWithContext(t *testing.T, prompt string, timeout time.Duration) []MessageType {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	opts := &AgentOptions{
		Model:          h.opts.Model,
		BaseURL:        h.opts.BaseURL,
		APIKey:         h.opts.APIKey,
		PermissionMode: PermissionModeBypassPermission,
		Interactive:    false,
		TimeoutSecs:    int(timeout.Seconds()),
	}

	messages, err := Query(ctx, prompt, opts)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	return messages
}

// IsRecording returns true if the VCR is recording
func (h *IntegrationTestHelper) IsRecording() bool {
	return h.vcr.IsRecording()
}

// Provider utilities

// GetProviderBaseURL returns the base URL for a provider
func GetProviderBaseURL(provider string) string {
	switch strings.ToLower(provider) {
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

// GetProviderModel returns the model name for a provider and tier
func GetProviderModel(provider, tier string) string {
	tier = strings.ToLower(tier)

	switch strings.ToLower(provider) {
	case "minmax":
		switch tier {
		case "haiku", "small", "small-fast":
			return "minimax-abab6.5s-chat"
		case "opus", "big":
			return "minimax-abab6.5s-chat"
		default:
			return "minimax-abab6.5-chat"
		}
	case "glm":
		switch tier {
		case "haiku", "small", "small-fast":
			return "glm-3-turbo"
		case "opus", "big":
			return "glm-4-plus"
		default:
			return "glm-4"
		}
	case "anthropic":
		switch tier {
		case "haiku", "small", "small-fast":
			return "claude-3-5-haiku-20241022"
		case "opus", "big":
			return "claude-opus-4-20250514"
		default:
			return "claude-sonnet-4-20250514"
		}
	default:
		switch tier {
		case "haiku", "small", "small-fast":
			return "claude-3-5-haiku-20241022"
		case "opus", "big":
			return "claude-opus-4-20250514"
		default:
			return "claude-sonnet-4-20250514"
		}
	}
}

// GetProviderAPIKey returns the environment variable for a provider's API key
func GetProviderAPIKey(provider string) string {
	switch strings.ToLower(provider) {
	case "minmax":
		return os.Getenv("MINMAX_API_KEY")
	case "glm":
		return os.Getenv("GLM_API_ACCESS_ID")
	case "anthropic":
		fallthrough
	default:
		return os.Getenv("ANTHROPIC_API_KEY")
	}
}

// SetProviderAPIKey sets the API key for a provider
func SetProviderAPIKey(provider, key string) {
	switch strings.ToLower(provider) {
	case "minmax":
		os.Setenv("MINMAX_API_KEY", key)
	case "glm":
		os.Setenv("GLM_API_ACCESS_ID", key)
	case "anthropic":
		fallthrough
	default:
		os.Setenv("ANTHROPIC_API_KEY", key)
	}
}

// RecordingServer creates a mock server that simulates provider API responses
// for recording cassettes
func RecordingServer(t *testing.T, provider string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Generate response based on provider
		response := generateMockResponse(provider, r)
		w.Write([]byte(response))
	}))
}

// generateMockResponse generates a mock response for a provider
func generateMockResponse(provider string, r *http.Request) string {
	switch strings.ToLower(provider) {
	case "minmax":
		return `{"id":"minimax-test-123","object":"chat.completion","created":1234567890,"model":"minimax-abab6.5-chat","choices":[{"index":0,"message":{"role":"assistant","content":"4"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	case "glm":
		return `{"id":"glm-test-123","object":"chat.completion","created":1234567890,"model":"glm-4","choices":[{"index":0,"message":{"role":"assistant","content":"4"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	case "anthropic":
		fallthrough
	default:
		return `{"id":"test-123","object":"chat.completion","created":1234567890,"model":"claude-sonnet-4-20250514","choices":[{"index":0,"message":{"role":"assistant","content":"4"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`
	}
}

// VCRTestCase defines a test case for VCR-based testing
type VCRTestCase struct {
	Name        string
	Prompt      string
	Setup       func(*IntegrationTestConfig)
	Assert      func(*testing.T, []MessageType)
	SkipRecording bool
}

// RunVCRTestCases runs a set of VCR test cases
func RunVCRTestCases(t *testing.T, provider, cassetteName string, cases []VCRTestCase) {
	t.Helper()

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			config := IntegrationTestConfig{
				CassetteName:  cassetteName + "_" + tc.Name,
				Provider:      provider,
				SkipRecording: tc.SkipRecording,
			}

			if tc.Setup != nil {
				tc.Setup(&config)
			}

			helper := NewIntegrationTestHelper(config)
			defer helper.Close()

			helper.Setup(t)
			defer helper.Close()

			// Run query
			prompt := tc.Prompt
			if prompt == "" {
				prompt = "What is 2+2? Just give me the number."
			}

			messages := helper.QueryWithContext(t, prompt, 30*time.Second)

			// Assert
			if tc.Assert != nil {
				tc.Assert(t, messages)
			}
		})
	}
}

// SkipIfNoAPIKey skips the test if no API key is available for the provider
func SkipIfNoAPIKey(t *testing.T, provider string) {
	t.Helper()

	key := GetProviderAPIKey(provider)
	if key == "" && !testing.Short() {
		t.Skipf("No API key available for provider %s. Set %s_API_KEY environment variable.",
			provider, strings.ToUpper(provider))
	}
}

// CreateVCRCassetteForCLI creates a cassette by running actual CLI commands
// This is used for initial cassette generation
func CreateVCRCassetteForCLI(t *testing.T, cassetteName, provider, prompt string) {
	t.Helper()

	SkipIfNoAPIKey(t, provider)

	// Create cassette directory
	cassetteDir := filepath.Join("testdata", "cassettes", "integration")
	if err := os.MkdirAll(cassetteDir, 0755); err != nil {
		t.Fatalf("Failed to create cassette directory: %v", err)
	}

	cassettePath := filepath.Join(cassetteDir, cassetteName)

	// Get provider settings
	baseURL := GetProviderBaseURL(provider)
	apiKey := GetProviderAPIKey(provider)
	model := GetProviderModel(provider, "sonnet")

	// Create VCR server in record mode
	vcr := NewVCRServer(cassettePath,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(baseURL),
	)

	baseURL, err := vcr.Start()
	if err != nil {
		t.Fatalf("Failed to start VCR: %v", err)
	}
	defer vcr.Close()

	// Set environment for CLI
	envBaseURL := os.Getenv("ANTHROPIC_BASE_URL")
	envAPIKey := os.Getenv("ANTHROPIC_API_KEY")

	os.Setenv("ANTHROPIC_BASE_URL", baseURL)
	os.Setenv("ANTHROPIC_API_KEY", apiKey)

	defer func() {
		if envBaseURL == "" {
			os.Unsetenv("ANTHROPIC_BASE_URL")
		} else {
			os.Setenv("ANTHROPIC_BASE_URL", envBaseURL)
		}
		if envAPIKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", envAPIKey)
		}
	}()

	// Create client and run query
	opts := &AgentOptions{
		Model:          model,
		BaseURL:        baseURL,
		APIKey:         apiKey,
		PermissionMode: PermissionModeBypassPermission,
		Interactive:    false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if prompt == "" {
		prompt = "What is 2+2? Just give me the number."
	}

	messages, err := Query(ctx, prompt, opts)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	t.Logf("Recorded %d messages for cassette %s", len(messages), cassetteName)
}
