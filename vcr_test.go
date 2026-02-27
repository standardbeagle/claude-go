package claude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVCRServer_BasicRecordReplay(t *testing.T) {
	// Create temp directory for cassettes
	tempDir, err := os.MkdirTemp("", "vcr-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "test_cassette")

	// Create a mock target server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response": "hello from mock server", "path": "` + r.URL.Path + `"}`))
	}))
	defer mockServer.Close()

	// Phase 1: Record
	t.Run("record", func(t *testing.T) {
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

		// Make a request through VCR
		resp, err := http.Get(baseURL + "/v1/messages")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "hello from mock server") {
			t.Errorf("Unexpected response: %s", body)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})

	// Verify cassette was created
	cassettePath = cassettePath + ".yaml"
	if _, err := os.Stat(cassettePath); os.IsNotExist(err) {
		t.Fatalf("Cassette file was not created at %s", cassettePath)
	}

	// Phase 2: Replay (without mock server)
	t.Run("replay", func(t *testing.T) {
		// Remove .yaml for VCR path
		cassettePathNoExt := strings.TrimSuffix(cassettePath, ".yaml")

		vcr := NewVCRServer(cassettePathNoExt,
			WithVCRMode(VCRModeReplay),
			// Target URL doesn't matter in replay mode
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
		resp, err := http.Get(baseURL + "/v1/messages")
		if err != nil {
			t.Fatalf("Replay request failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "hello from mock server") {
			t.Errorf("Replayed response doesn't match: %s", body)
		}

		if err := vcr.Close(); err != nil {
			t.Errorf("Failed to close VCR: %v", err)
		}
	})
}

func TestVCRServer_AutoMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vcr-auto-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "auto_cassette")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"auto": "test"}`))
	}))
	defer mockServer.Close()

	// First run - should record (cassette doesn't exist)
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

		resp, err := http.Get(baseURL + "/test")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		resp.Body.Close()

		vcr.Close()
	})

	// Second run - should replay (cassette exists)
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

		resp, err := http.Get(baseURL + "/test")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "auto") {
			t.Errorf("Unexpected response: %s", body)
		}

		vcr.Close()
	})
}

func TestVCRServer_PassthroughMode(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"passthrough": "test"}`))
	}))
	defer mockServer.Close()

	vcr := NewVCRServer("unused",
		WithVCRMode(VCRModePassthrough),
		WithTargetURL(mockServer.URL),
	)

	baseURL, err := vcr.Start()
	if err != nil {
		t.Fatalf("Failed to start VCR: %v", err)
	}
	defer vcr.Close()

	// In passthrough mode, baseURL should be the target URL
	if baseURL != mockServer.URL {
		t.Errorf("Expected baseURL %s, got %s", mockServer.URL, baseURL)
	}
}

func TestVCRTestHelper_Setup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vcr-helper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Save original env
	origBaseURL := os.Getenv(EnvAnthropicBaseURL)
	defer func() {
		if origBaseURL == "" {
			os.Unsetenv(EnvAnthropicBaseURL)
		} else {
			os.Setenv(EnvAnthropicBaseURL, origBaseURL)
		}
	}()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"helper": "test"}`))
	}))
	defer mockServer.Close()

	// Create helper with custom cassette path
	cassetteName := filepath.Join(tempDir, "helper_cassette")
	helper := &VCRTestHelper{
		vcr:     NewVCRServer(cassetteName, WithVCRMode(VCRModeRecord), WithTargetURL(mockServer.URL)),
		origEnv: make(map[string]string),
		envVars: []string{EnvAnthropicBaseURL},
	}

	cleanup, err := helper.Setup()
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer cleanup()

	// Check environment was set
	baseURL := os.Getenv(EnvAnthropicBaseURL)
	if baseURL == "" {
		t.Error("ANTHROPIC_BASE_URL was not set")
	}

	if baseURL != helper.BaseURL() {
		t.Errorf("Environment doesn't match helper: env=%s, helper=%s", baseURL, helper.BaseURL())
	}

	// Make a request
	resp, err := http.Get(baseURL + "/test")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	// Cleanup and verify env restored
	cleanup()

	restoredURL := os.Getenv(EnvAnthropicBaseURL)
	if restoredURL != origBaseURL {
		t.Errorf("Environment not restored: expected %q, got %q", origBaseURL, restoredURL)
	}
}

func TestVCRServer_SanitizesCredentials(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "vcr-sanitize-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	cassettePath := filepath.Join(tempDir, "sanitize_cassette")

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't echo back the API key - just return a simple response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer mockServer.Close()

	// Record with API key - use the recorder directly instead of through proxy
	// This tests the sanitization hook properly
	rec, err := NewVCRRecorder(cassettePath,
		WithVCRMode(VCRModeRecord),
		WithTargetURL(mockServer.URL),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	client := rec.HTTPClient()

	req, _ := http.NewRequest("GET", mockServer.URL+"/test", nil)
	req.Header.Set("X-Api-Key", "sk-ant-secret-key-12345")
	req.Header.Set("Authorization", "Bearer secret-token")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	resp.Body.Close()

	rec.Close()

	// Read cassette and verify credentials are redacted in headers
	cassetteFile := cassettePath + ".yaml"
	content, err := os.ReadFile(cassetteFile)
	if err != nil {
		t.Fatalf("Failed to read cassette: %v", err)
	}

	contentStr := string(content)

	// The raw API key should NOT appear in the headers section
	// We check that the key doesn't appear as a header value
	if strings.Contains(contentStr, "- sk-ant-secret-key-12345") {
		t.Errorf("API key was not sanitized from cassette headers. Content:\n%s", contentStr)
	}

	if strings.Contains(contentStr, "- Bearer secret-token") {
		t.Errorf("Authorization token was not sanitized from cassette headers. Content:\n%s", contentStr)
	}

	// Verify [REDACTED] placeholder is present
	if !strings.Contains(contentStr, "[REDACTED]") {
		t.Errorf("Expected [REDACTED] placeholder in cassette. Content:\n%s", contentStr)
	}

	// Verify the headers section has X-Api-Key with redacted value
	if !strings.Contains(contentStr, "X-Api-Key:") {
		t.Errorf("Expected X-Api-Key header to be present. Content:\n%s", contentStr)
	}
}
