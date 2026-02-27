package claude

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Provider-specific constants
const (
	// MinMax API endpoints
	MinMaxDefaultBaseURL = "https://api.minimax.chat/v1/openai/chat/completions"

	// GLM (ChatGLM) API endpoints
	GLMDefaultBaseURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
)

// TestMinMaxProvider_VCR tests minmax provider with VCR recording/replay
func TestMinMaxProvider_VCR(t *testing.T) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", "minmax-vcr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "minmax_provider")

	// Create a mock minmax-compatible API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers for minmax
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", r.Header.Get("Content-Type"))
		}

		// Return a minmax-compatible response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"id":      "chatcmpl-" + generateUUID(t),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "minmax-chat",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from MinMax provider!",
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
	}))
	defer mockServer.Close()

	// Phase 1: Record interactions
	t.Run("record_minmax", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeRecord),
			WithTargetURL(mockServer.URL),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR: %v", err)
		}

		if !vcr.IsRecording() {
			t.Error("Expected VCR to be recording")
		}

		// Make request through VCR
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"minmax-chat","messages":[{"role":"user","content":"Hello"}]}`))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})

	// Verify cassette was created
	cassetteFile := cassettePath + ".yaml"
	if _, err := os.Stat(cassetteFile); os.IsNotExist(err) {
		t.Fatalf("Cassette file was not created at %s", cassetteFile)
	}

	// Phase 2: Replay interactions
	t.Run("replay_minmax", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeReplay),
			WithTargetURL("http://should-not-be-called.invalid"),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR for replay: %v", err)
		}

		if vcr.IsRecording() {
			t.Error("Expected VCR to NOT be recording in replay mode")
		}

		// Make the same request - should be replayed
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"minmax-chat","messages":[{"role":"user","content":"Hello"}]}`))
		if err != nil {
			t.Fatalf("Replay request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})
}

// TestGLMProvider_VCR tests GLM (ChatGLM) provider with VCR recording/replay
func TestGLMProvider_VCR(t *testing.T) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", "glm-vcr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "glm_provider")

	// Create a mock GLM-compatible API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers for GLM
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain application/json, got: %s", contentType)
		}

		// Return a GLM-compatible response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"id":      "glm-" + generateUUID(t),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "glm-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from GLM provider!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     15,
				"completion_tokens": 25,
				"total_tokens":      40,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Phase 1: Record interactions
	t.Run("record_glm", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeRecord),
			WithTargetURL(mockServer.URL),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR: %v", err)
		}

		if !vcr.IsRecording() {
			t.Error("Expected VCR to be recording")
		}

		// Make request through VCR
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"glm-4","messages":[{"role":"user","content":"Hello"}]}`))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})

	// Verify cassette was created
	cassetteFile := cassettePath + ".yaml"
	if _, err := os.Stat(cassetteFile); os.IsNotExist(err) {
		t.Fatalf("Cassette file was not created at %s", cassetteFile)
	}

	// Phase 2: Replay interactions
	t.Run("replay_glm", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeReplay),
			WithTargetURL("http://should-not-be-called.invalid"),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR for replay: %v", err)
		}

		if vcr.IsRecording() {
			t.Error("Expected VCR to NOT be recording in replay mode")
		}

		// Make the same request - should be replayed
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"glm-4","messages":[{"role":"user","content":"Hello"}]}`))
		if err != nil {
			t.Fatalf("Replay request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})
}

// TestMinMaxProvider_Integration tests minmax provider integration with VCR
func TestMinMaxProvider_Integration(t *testing.T) {
	testProviderIntegration(t, "minmax", MinMaxDefaultBaseURL)
}

// TestGLMProvider_Integration tests GLM provider integration with VCR
func TestGLMProvider_Integration(t *testing.T) {
	testProviderIntegration(t, "glm", GLMDefaultBaseURL)
}

// testProviderIntegration is a helper function for testing provider integration
func testProviderIntegration(t *testing.T, providerName, defaultBaseURL string) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", providerName+"-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, providerName+"_integration")

	// Create a mock server that simulates provider API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"id":      providerName + "-test-" + generateUUID(t),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   providerName + "-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Test response from " + providerName,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     5,
				"completion_tokens": 10,
				"total_tokens":      15,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Save original environment
	origBaseURL := os.Getenv(EnvAnthropicBaseURL)
	origAPIKey := os.Getenv(EnvAnthropicAPIKey)
	defer func() {
		if origBaseURL == "" {
			os.Unsetenv(EnvAnthropicBaseURL)
		} else {
			os.Setenv(EnvAnthropicBaseURL, origBaseURL)
		}
		if origAPIKey == "" {
			os.Unsetenv(EnvAnthropicAPIKey)
		} else {
			os.Setenv(EnvAnthropicAPIKey, origAPIKey)
		}
	}()

	// Test with auto mode (records first, then replays)
	t.Run("auto_mode_record", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeAuto),
			WithTargetURL(mockServer.URL),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR: %v", err)
		}

		// Set environment to use VCR
		os.Setenv(EnvAnthropicBaseURL, baseURL)
		os.Setenv(EnvAnthropicAPIKey, "test-api-key")

		// Make request
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"test-model","messages":[{"role":"user","content":"Test"}]}`))
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		vcr.Close()
	})

	// Verify cassette exists
	cassetteFile := cassettePath + ".yaml"
	if _, err := os.Stat(cassetteFile); os.IsNotExist(err) {
		t.Fatalf("Cassette file was not created at %s", cassetteFile)
	}

	// Test replay
	t.Run("auto_mode_replay", func(t *testing.T) {
		vcr := NewVCRServer(cassettePath,
			WithVCRMode(VCRModeAuto),
			WithTargetURL("http://should-not-be-called.invalid"),
		)

		baseURL, err := vcr.Start()
		if err != nil {
			t.Fatalf("Failed to start VCR for replay: %v", err)
		}

		// Set environment to use VCR
		os.Setenv(EnvAnthropicBaseURL, baseURL)

		// Make the same request - should be replayed
		resp, err := http.Post(baseURL+"/chat/completions", "application/json",
			strings.NewReader(`{"model":"test-model","messages":[{"role":"user","content":"Test"}]}`))
		if err != nil {
			t.Fatalf("Replay request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}

		vcr.Close()
	})
}

// TestVCRTestHelper_MinMax tests VCRTestHelper with minmax provider
func TestVCRTestHelper_MinMax(t *testing.T) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", "minmax-helper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassetteName := filepath.Join(tempDir, "minmax_helper")

	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"id":      "minmax-helper-" + generateUUID(t),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "minimax-abab6.5s-chat",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "MinMax VCR Helper Test Response",
					},
					"finish_reason": "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create VCRTestHelper with custom settings
	helper := NewVCRTestHelper(cassetteName,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(mockServer.URL),
	)

	cleanup, err := helper.Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Verify environment was set
	baseURL := os.Getenv(EnvAnthropicBaseURL)
	if baseURL == "" {
		t.Error("ANTHROPIC_BASE_URL was not set")
	}

	if baseURL != helper.BaseURL() {
		t.Errorf("Environment doesn't match helper: env=%s, helper=%s", baseURL, helper.BaseURL())
	}

	// Make a request
	resp, err := http.Post(baseURL+"/chat/completions", "application/json",
		strings.NewReader(`{"model":"minimax-abab6.5s-chat","messages":[{"role":"user","content":"Test"}]}`))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()
}

// TestVCRTestHelper_GLM tests VCRTestHelper with GLM provider
func TestVCRTestHelper_GLM(t *testing.T) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", "glm-helper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassetteName := filepath.Join(tempDir, "glm_helper")

	// Create mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"id":      "glm-helper-" + generateUUID(t),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "glm-4-plus",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "GLM VCR Helper Test Response",
					},
					"finish_reason": "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create VCRTestHelper with custom settings
	helper := NewVCRTestHelper(cassetteName,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(mockServer.URL),
	)

	cleanup, err := helper.Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Verify environment was set
	baseURL := os.Getenv(EnvAnthropicBaseURL)
	if baseURL == "" {
		t.Error("ANTHROPIC_BASE_URL was not set")
	}

	if baseURL != helper.BaseURL() {
		t.Errorf("Environment doesn't match helper: env=%s, helper=%s", baseURL, helper.BaseURL())
	}

	// Make a request
	resp, err := http.Post(baseURL+"/chat/completions", "application/json",
		strings.NewReader(`{"model":"glm-4-plus","messages":[{"role":"user","content":"Test"}]}`))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()
}

// TestMinMaxProvider_CredentialSanitization tests that API credentials are properly sanitized
func TestMinMaxProvider_CredentialSanitization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "minmax-sanitize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "minmax_sanitize")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer mockServer.Close()

	// Record with API key
	rec, err := NewVCRRecorder(cassettePath,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(mockServer.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	client := rec.HTTPClient()

	req, _ := http.NewRequest("POST", mockServer.URL+"/chat/completions", strings.NewReader(`{"model":"test"}`))
	req.Header.Set("X-Api-Key", "minimax-secret-key-12345")
	req.Header.Set("Authorization", "Bearer minimax-bearer-token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	rec.Close()

	// Read cassette and verify credentials are redacted
	cassetteFile := cassettePath + ".yaml"
	content, err := os.ReadFile(cassetteFile)
	if err != nil {
		t.Fatalf("Failed to read cassette: %v", err)
	}

	contentStr := string(content)

	// Verify sanitization
	if strings.Contains(contentStr, "minimax-secret-key-12345") {
		t.Errorf("API key was not sanitized from cassette")
	}

	if strings.Contains(contentStr, "minimax-bearer-token") {
		t.Errorf("Bearer token was not sanitized from cassette")
	}
}

// TestGLMProvider_CredentialSanitization tests that GLM credentials are properly sanitized
func TestGLMProvider_CredentialSanitization(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glm-sanitize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "glm_sanitize")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer mockServer.Close()

	// Record with credentials
	rec, err := NewVCRRecorder(cassettePath,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(mockServer.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	client := rec.HTTPClient()

	req, _ := http.NewRequest("POST", mockServer.URL+"/chat/completions", strings.NewReader(`{"model":"test"}`))
	req.Header.Set("X-Api-Key", "glm-access-id-12345")
	req.Header.Set("Authorization", "Bearer glm-secret-key-67890")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	rec.Close()

	// Read cassette and verify credentials are redacted
	cassetteFile := cassettePath + ".yaml"
	content, err := os.ReadFile(cassetteFile)
	if err != nil {
		t.Fatalf("Failed to read cassette: %v", err)
	}

	contentStr := string(content)

	// Verify sanitization
	if strings.Contains(contentStr, "glm-access-id-12345") {
		t.Errorf("Access ID was not sanitized from cassette")
	}

	if strings.Contains(contentStr, "glm-secret-key-67890") {
		t.Errorf("Secret key was not sanitized from cassette")
	}
}

// TestProviderAutoModeReplay tests auto mode for providers
func TestProviderAutoModeReplay(t *testing.T) {
	testCases := []struct {
		name         string
		providerName string
	}{
		{"minmax", "minmax"},
		{"glm", "glm"},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_auto_mode", func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", tc.providerName+"-auto-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			cassettePath := filepath.Join(tempDir, tc.providerName+"_auto")

			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := map[string]interface{}{
					"id":      tc.providerName + "-auto-" + generateUUID(t),
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   tc.providerName + "-model",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]interface{}{
								"role":    "assistant",
								"content": "Auto mode response from " + tc.providerName,
							},
							"finish_reason": "stop",
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer mockServer.Close()

			// First run - should record
			t.Run("first_run_records", func(t *testing.T) {
				vcr := NewVCRServer(cassettePath,
					WithVCRMode(VCRModeAuto),
					WithTargetURL(mockServer.URL),
				)

				baseURL, err := vcr.Start()
				if err != nil {
					t.Fatalf("Failed to start VCR: %v", err)
				}

				// In auto mode with no cassette, should record
				if !vcr.IsRecording() {
					t.Error("Expected recording in auto mode with no cassette")
				}

				resp, err := http.Post(baseURL+"/chat/completions", "application/json",
					strings.NewReader(`{"model":"test"}`))
				if err != nil {
					t.Fatalf("Request failed: %v", err)
				}
				resp.Body.Close()

				vcr.Close()
			})

			// Second run - should replay
			t.Run("second_run_replays", func(t *testing.T) {
				vcr := NewVCRServer(cassettePath,
					WithVCRMode(VCRModeAuto),
					WithTargetURL("http://should-not-be-called.invalid"),
				)

				baseURL, err := vcr.Start()
				if err != nil {
					t.Fatalf("Failed to start VCR: %v", err)
				}

				// In auto mode with existing cassette, should replay
				if vcr.IsRecording() {
					t.Error("Expected replay in auto mode with existing cassette")
				}

				resp, err := http.Post(baseURL+"/chat/completions", "application/json",
					strings.NewReader(`{"model":"test"}`))
				if err != nil {
					t.Fatalf("Replay request failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status 200, got: %d", resp.StatusCode)
				}

				vcr.Close()
			})
		})
	}
}

// generateUUID generates a simple UUID-like string for test responses
func generateUUID(t *testing.T) string {
	t.Helper()
	// Use time-based UUID for simplicity in tests
	return strings.ReplaceAll(time.Now().Format(time.RFC3339Nano), ".", "")[:16]
}

// TestProviderEndpointConfiguration tests endpoint configuration for providers
func TestProviderEndpointConfiguration(t *testing.T) {
	testCases := []struct {
		name           string
		provider       string
		expectedSuffix string
	}{
		{"minmax_completions", "minmax", "/chat/completions"},
		{"glm_completions", "glm", "/chat/completions"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify that our provider constants have the correct endpoint suffix
			if tc.provider == "minmax" && !strings.HasSuffix(MinMaxDefaultBaseURL, tc.expectedSuffix) {
				t.Errorf("MinMax base URL should end with %s, got: %s", tc.expectedSuffix, MinMaxDefaultBaseURL)
			}
			if tc.provider == "glm" && !strings.HasSuffix(GLMDefaultBaseURL, tc.expectedSuffix) {
				t.Errorf("GLM base URL should end with %s, got: %s", tc.expectedSuffix, GLMDefaultBaseURL)
			}
		})
	}
}

// TestVCRRecorderProviderIntegration tests VCRRecorder with provider endpoints
func TestVCRRecorderProviderIntegration(t *testing.T) {
	// This test verifies the VCRRecorder works correctly with provider-like endpoints
	testCases := []struct {
		name         string
		cassetteName string
	}{
		{"minmax", "minmax_recorder_test"},
		{"glm", "glm_recorder_test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", tc.cassetteName+"-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			cassettePath := filepath.Join(tempDir, tc.cassetteName)

			// Create mock server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				response := map[string]interface{}{
					"id":      tc.name + "-recorder-" + generateUUID(t),
					"object":  "chat.completion",
					"created": time.Now().Unix(),
					"model":   tc.name + "-model",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]interface{}{
								"role":    "assistant",
								"content": "VCR Recorder test response",
							},
							"finish_reason": "stop",
						},
					},
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer mockServer.Close()

			// Create recorder
			rec, err := NewVCRRecorder(cassettePath,
				WithVCRMode(VCRModeRecord),
				WithTargetURL(mockServer.URL),
			)
			if err != nil {
				t.Fatalf("Failed to create recorder: %v", err)
			}

			// Make request
			client := rec.HTTPClient()
			req, _ := http.NewRequest("POST", mockServer.URL+"/chat/completions",
				strings.NewReader(`{"model":"test","messages":[{"role":"user","content":"test"}]}`))
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()

			if !rec.IsRecording() {
				t.Error("Expected recorder to be in recording mode")
			}

			rec.Close()

			// Verify cassette was created
			cassetteFile := cassettePath + ".yaml"
			if _, err := os.Stat(cassetteFile); os.IsNotExist(err) {
				t.Fatalf("Cassette file was not created at %s", cassetteFile)
			}
		})
	}
}

// TestProviderWithContextCancellation tests provider behavior with context cancellation
func TestProviderWithContextCancellation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cancel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "cancel_test")

	// Create a slow mock server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"message":{"content":"slow response"}}]}`))
	}))
	defer slowServer.Close()

	// Record the slow response
	rec, err := NewVCRRecorder(cassettePath,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(slowServer.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	client := rec.HTTPClient()
	req, _ := http.NewRequest("GET", slowServer.URL+"/test", nil)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	rec.Close()

	// Now test replay with cancellation
	t.Run("replay_with_context_cancel", func(t *testing.T) {
		rec, err := NewVCRRecorder(cassettePath,
			WithVCRMode(VCRModeReplay),
			WithTargetURL("http://should-not-be-called.invalid"),
		)
		if err != nil {
			t.Fatalf("Failed to create recorder: %v", err)
		}

		client := rec.HTTPClient()
		req, _ := http.NewRequest("GET", "http://should-not-be-called.invalid/test", nil)

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// The request should return immediately (replayed response has no latency)
		done := make(chan struct{})
		go func() {
			resp, err := client.Do(req.WithContext(ctx))
			if err == nil {
				resp.Body.Close()
			}
			close(done)
		}()

		select {
		case <-done:
			// Success - request completed immediately
			if rec.IsRecording() {
				t.Error("Recorder should not be recording in replay mode")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Request did not complete within timeout")
		}

		rec.Close()
	})
}
