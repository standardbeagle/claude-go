package claude

import (
	"os"
	"strings"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	// Save and restore environment
	origEnv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, e := range origEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()

	t.Run("loads API key from environment", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvAnthropicAPIKey, "test-api-key")

		cfg := LoadConfigFromEnv()

		if cfg.APIKey != "test-api-key" {
			t.Errorf("expected APIKey 'test-api-key', got '%s'", cfg.APIKey)
		}
	})

	t.Run("prefers CLAUDE_CODE_API_KEY over ANTHROPIC_API_KEY", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvAnthropicAPIKey, "anthropic-key")
		os.Setenv(EnvClaudeCodeAPIKey, "claude-code-key")

		cfg := LoadConfigFromEnv()

		if cfg.APIKey != "claude-code-key" {
			t.Errorf("expected APIKey 'claude-code-key', got '%s'", cfg.APIKey)
		}
	})

	t.Run("loads base URL from environment", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvAnthropicBaseURL, "https://custom.api.com")

		cfg := LoadConfigFromEnv()

		if cfg.BaseURL != "https://custom.api.com" {
			t.Errorf("expected BaseURL 'https://custom.api.com', got '%s'", cfg.BaseURL)
		}
	})

	t.Run("detects Bedrock provider", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeUseBedrock, "true")

		cfg := LoadConfigFromEnv()

		if cfg.Provider != APIProviderBedrock {
			t.Errorf("expected Provider 'bedrock', got '%s'", cfg.Provider)
		}
	})

	t.Run("detects Vertex provider", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeUseVertex, "1")

		cfg := LoadConfigFromEnv()

		if cfg.Provider != APIProviderVertex {
			t.Errorf("expected Provider 'vertex', got '%s'", cfg.Provider)
		}
	})

	t.Run("loads Bedrock configuration", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeUseBedrock, "1")
		os.Setenv(EnvAWSRegion, "us-east-1")
		os.Setenv(EnvAWSProfile, "my-profile")
		os.Setenv(EnvBedrockCrossRegion, "true")

		cfg := LoadConfigFromEnv()

		if cfg.Bedrock == nil {
			t.Fatal("expected Bedrock config, got nil")
		}
		if cfg.Bedrock.Region != "us-east-1" {
			t.Errorf("expected Region 'us-east-1', got '%s'", cfg.Bedrock.Region)
		}
		if cfg.Bedrock.Profile != "my-profile" {
			t.Errorf("expected Profile 'my-profile', got '%s'", cfg.Bedrock.Profile)
		}
		if !cfg.Bedrock.CrossRegion {
			t.Error("expected CrossRegion true")
		}
	})

	t.Run("loads Vertex configuration", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeUseVertex, "1")
		os.Setenv(EnvVertexProject, "my-project")
		os.Setenv(EnvVertexRegion, "us-central1")

		cfg := LoadConfigFromEnv()

		if cfg.Vertex == nil {
			t.Fatal("expected Vertex config, got nil")
		}
		if cfg.Vertex.ProjectID != "my-project" {
			t.Errorf("expected ProjectID 'my-project', got '%s'", cfg.Vertex.ProjectID)
		}
		if cfg.Vertex.Region != "us-central1" {
			t.Errorf("expected Region 'us-central1', got '%s'", cfg.Vertex.Region)
		}
	})

	t.Run("loads model configuration", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeModel, "claude-3-opus")
		os.Setenv(EnvClaudeSmallFastModel, "claude-3-haiku")
		os.Setenv(EnvClaudeBigModel, "claude-4-opus")

		cfg := LoadConfigFromEnv()

		if cfg.Model != "claude-3-opus" {
			t.Errorf("expected Model 'claude-3-opus', got '%s'", cfg.Model)
		}
		if cfg.SmallFastModel != "claude-3-haiku" {
			t.Errorf("expected SmallFastModel 'claude-3-haiku', got '%s'", cfg.SmallFastModel)
		}
		if cfg.BigModel != "claude-4-opus" {
			t.Errorf("expected BigModel 'claude-4-opus', got '%s'", cfg.BigModel)
		}
	})

	t.Run("loads runtime limits", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeMaxTokens, "4096")
		os.Setenv(EnvClaudeCodeMaxTurns, "10")
		os.Setenv(EnvClaudeCodeTimeout, "120")
		os.Setenv(EnvClaudeCodeMaxBudgetUSD, "5.50")

		cfg := LoadConfigFromEnv()

		if cfg.MaxTokens != 4096 {
			t.Errorf("expected MaxTokens 4096, got %d", cfg.MaxTokens)
		}
		if cfg.MaxTurns != 10 {
			t.Errorf("expected MaxTurns 10, got %d", cfg.MaxTurns)
		}
		if cfg.TimeoutSecs != 120 {
			t.Errorf("expected TimeoutSecs 120, got %d", cfg.TimeoutSecs)
		}
		if cfg.MaxBudgetUSD != 5.50 {
			t.Errorf("expected MaxBudgetUSD 5.50, got %f", cfg.MaxBudgetUSD)
		}
	})

	t.Run("loads behavior flags", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvClaudeCodeDebug, "1")
		os.Setenv(EnvClaudeCodeVerbose, "true")
		os.Setenv(EnvClaudeCodeNoTelemetry, "yes")
		os.Setenv(EnvClaudeCodeSkipOAuth, "on")

		cfg := LoadConfigFromEnv()

		if !cfg.Debug {
			t.Error("expected Debug true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose true")
		}
		if !cfg.NoTelemetry {
			t.Error("expected NoTelemetry true")
		}
		if !cfg.SkipOAuth {
			t.Error("expected SkipOAuth true")
		}
	})

	t.Run("loads proxy configuration", func(t *testing.T) {
		os.Clearenv()
		os.Setenv(EnvHTTPProxy, "http://proxy:8080")
		os.Setenv(EnvHTTPSProxy, "https://proxy:8443")
		os.Setenv(EnvNoProxy, "localhost,127.0.0.1")

		cfg := LoadConfigFromEnv()

		if cfg.Proxy == nil {
			t.Fatal("expected Proxy config, got nil")
		}
		if cfg.Proxy.HTTPProxy != "http://proxy:8080" {
			t.Errorf("expected HTTPProxy 'http://proxy:8080', got '%s'", cfg.Proxy.HTTPProxy)
		}
		if cfg.Proxy.HTTPSProxy != "https://proxy:8443" {
			t.Errorf("expected HTTPSProxy 'https://proxy:8443', got '%s'", cfg.Proxy.HTTPSProxy)
		}
		if cfg.Proxy.NoProxy != "localhost,127.0.0.1" {
			t.Errorf("expected NoProxy 'localhost,127.0.0.1', got '%s'", cfg.Proxy.NoProxy)
		}
	})
}

func TestConfigToAgentOptions(t *testing.T) {
	t.Run("converts basic fields", func(t *testing.T) {
		cfg := &Config{
			Model:        "claude-3-opus",
			MaxTurns:     10,
			MaxBudgetUSD: 5.0,
			Debug:        true,
			Verbose:      true,
			CLIPath:      "/usr/local/bin/claude",
		}

		opts := cfg.ToAgentOptions()

		if opts.Model != "claude-3-opus" {
			t.Errorf("expected Model 'claude-3-opus', got '%s'", opts.Model)
		}
		if opts.MaxTurns != 10 {
			t.Errorf("expected MaxTurns 10, got %d", opts.MaxTurns)
		}
		if opts.MaxBudgetUSD != 5.0 {
			t.Errorf("expected MaxBudgetUSD 5.0, got %f", opts.MaxBudgetUSD)
		}
		if !opts.Debug {
			t.Error("expected Debug true")
		}
		if !opts.Verbose {
			t.Error("expected Verbose true")
		}
		if opts.CLIPath != "/usr/local/bin/claude" {
			t.Errorf("expected CLIPath '/usr/local/bin/claude', got '%s'", opts.CLIPath)
		}
	})

	t.Run("sets environment for API key", func(t *testing.T) {
		cfg := &Config{
			APIKey: "test-key",
		}

		opts := cfg.ToAgentOptions()

		if opts.Environment[EnvAnthropicAPIKey] != "test-key" {
			t.Errorf("expected env %s='test-key', got '%s'", EnvAnthropicAPIKey, opts.Environment[EnvAnthropicAPIKey])
		}
	})

	t.Run("sets environment for Bedrock", func(t *testing.T) {
		cfg := &Config{
			Provider: APIProviderBedrock,
			Bedrock: &BedrockConfig{
				Region:        "us-west-2",
				CrossRegion:   true,
				PromptCaching: true,
			},
		}

		opts := cfg.ToAgentOptions()

		if opts.Environment[EnvClaudeCodeUseBedrock] != "1" {
			t.Error("expected CLAUDE_CODE_USE_BEDROCK=1")
		}
		if opts.Environment[EnvAWSRegion] != "us-west-2" {
			t.Errorf("expected AWS_REGION='us-west-2', got '%s'", opts.Environment[EnvAWSRegion])
		}
		if opts.Environment[EnvBedrockCrossRegion] != "1" {
			t.Error("expected BEDROCK_CROSS_REGION=1")
		}
		if opts.Environment[EnvBedrockPromptCaching] != "1" {
			t.Error("expected BEDROCK_PROMPT_CACHING=1")
		}
	})

	t.Run("sets environment for Vertex", func(t *testing.T) {
		cfg := &Config{
			Provider: APIProviderVertex,
			Vertex: &VertexConfig{
				ProjectID: "my-project",
				Region:    "us-central1",
			},
		}

		opts := cfg.ToAgentOptions()

		if opts.Environment[EnvClaudeCodeUseVertex] != "1" {
			t.Error("expected CLAUDE_CODE_USE_VERTEX=1")
		}
		if opts.Environment[EnvVertexProject] != "my-project" {
			t.Errorf("expected project 'my-project', got '%s'", opts.Environment[EnvVertexProject])
		}
		if opts.Environment[EnvVertexRegion] != "us-central1" {
			t.Errorf("expected region 'us-central1', got '%s'", opts.Environment[EnvVertexRegion])
		}
	})
}

func TestConfigMerge(t *testing.T) {
	t.Run("other takes precedence", func(t *testing.T) {
		base := &Config{
			APIKey: "base-key",
			Model:  "base-model",
			Debug:  false,
		}

		other := &Config{
			APIKey: "other-key",
			Debug:  true,
		}

		merged := base.Merge(other)

		if merged.APIKey != "other-key" {
			t.Errorf("expected APIKey 'other-key', got '%s'", merged.APIKey)
		}
		if merged.Model != "base-model" {
			t.Errorf("expected Model 'base-model', got '%s'", merged.Model)
		}
		if !merged.Debug {
			t.Error("expected Debug true")
		}
	})

	t.Run("nil other returns base", func(t *testing.T) {
		base := &Config{
			APIKey: "base-key",
		}

		merged := base.Merge(nil)

		if merged.APIKey != "base-key" {
			t.Errorf("expected APIKey 'base-key', got '%s'", merged.APIKey)
		}
	})
}

func TestNewConfigFromOptions(t *testing.T) {
	t.Run("extracts basic fields", func(t *testing.T) {
		opts := &AgentOptions{
			Model:        "claude-3-opus",
			MaxTurns:     10,
			MaxBudgetUSD: 5.0,
			Debug:        true,
			Verbose:      true,
			CLIPath:      "/usr/local/bin/claude",
		}

		cfg := NewConfigFromOptions(opts)

		if cfg.Model != "claude-3-opus" {
			t.Errorf("expected Model 'claude-3-opus', got '%s'", cfg.Model)
		}
		if cfg.MaxTurns != 10 {
			t.Errorf("expected MaxTurns 10, got %d", cfg.MaxTurns)
		}
		if cfg.MaxBudgetUSD != 5.0 {
			t.Errorf("expected MaxBudgetUSD 5.0, got %f", cfg.MaxBudgetUSD)
		}
		if !cfg.Debug {
			t.Error("expected Debug true")
		}
		if !cfg.Verbose {
			t.Error("expected Verbose true")
		}
		if cfg.CLIPath != "/usr/local/bin/claude" {
			t.Errorf("expected CLIPath '/usr/local/bin/claude', got '%s'", cfg.CLIPath)
		}
	})

	t.Run("extracts environment variables", func(t *testing.T) {
		opts := &AgentOptions{
			Environment: map[string]string{
				EnvAnthropicAPIKey:      "test-key",
				EnvAnthropicBaseURL:     "https://custom.api.com",
				EnvClaudeCodeUseBedrock: "1",
			},
		}

		cfg := NewConfigFromOptions(opts)

		if cfg.APIKey != "test-key" {
			t.Errorf("expected APIKey 'test-key', got '%s'", cfg.APIKey)
		}
		if cfg.BaseURL != "https://custom.api.com" {
			t.Errorf("expected BaseURL 'https://custom.api.com', got '%s'", cfg.BaseURL)
		}
		if cfg.Provider != APIProviderBedrock {
			t.Errorf("expected Provider 'bedrock', got '%s'", cfg.Provider)
		}
	})

	t.Run("nil options returns empty config", func(t *testing.T) {
		cfg := NewConfigFromOptions(nil)

		if cfg.Model != "" {
			t.Errorf("expected empty Model, got '%s'", cfg.Model)
		}
	})
}

func TestBuildCLIEnv(t *testing.T) {
	t.Run("includes SDK entrypoint", func(t *testing.T) {
		opts := &AgentOptions{}
		env := BuildCLIEnv(opts)

		found := false
		for _, e := range env {
			if e == "CLAUDE_CODE_ENTRYPOINT=sdk-go" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected CLAUDE_CODE_ENTRYPOINT=sdk-go in environment")
		}
	})

	t.Run("includes API key", func(t *testing.T) {
		opts := &AgentOptions{
			APIKey: "test-api-key",
		}
		env := BuildCLIEnv(opts)

		found := false
		for _, e := range env {
			if e == EnvAnthropicAPIKey+"=test-api-key" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected ANTHROPIC_API_KEY=test-api-key in environment")
		}
	})

	t.Run("includes Bedrock settings", func(t *testing.T) {
		opts := &AgentOptions{
			Provider: APIProviderBedrock,
			Bedrock: &BedrockConfig{
				Region: "us-east-1",
			},
		}
		env := BuildCLIEnv(opts)

		foundBedrock := false
		foundRegion := false
		for _, e := range env {
			if e == EnvClaudeCodeUseBedrock+"=1" {
				foundBedrock = true
			}
			if e == EnvAWSRegion+"=us-east-1" {
				foundRegion = true
			}
		}
		if !foundBedrock {
			t.Error("expected CLAUDE_CODE_USE_BEDROCK=1 in environment")
		}
		if !foundRegion {
			t.Error("expected AWS_REGION=us-east-1 in environment")
		}
	})

	t.Run("includes custom environment variables", func(t *testing.T) {
		opts := &AgentOptions{
			Environment: map[string]string{
				"CUSTOM_VAR": "custom-value",
			},
		}
		env := BuildCLIEnv(opts)

		found := false
		for _, e := range env {
			if e == "CUSTOM_VAR=custom-value" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected CUSTOM_VAR=custom-value in environment")
		}
	})
}

func TestBuildCLIArgs(t *testing.T) {
	t.Run("includes model flag", func(t *testing.T) {
		opts := &AgentOptions{
			Model: "claude-3-opus",
		}
		args := BuildCLIArgs(opts)

		foundModel := false
		for i, arg := range args {
			if arg == "--model" && i+1 < len(args) && args[i+1] == "claude-3-opus" {
				foundModel = true
				break
			}
		}
		if !foundModel {
			t.Error("expected --model claude-3-opus in args")
		}
	})

	t.Run("includes max-tokens flag", func(t *testing.T) {
		opts := &AgentOptions{
			MaxTokens: 4096,
		}
		args := BuildCLIArgs(opts)

		foundMaxTokens := false
		for i, arg := range args {
			if arg == "--max-tokens" && i+1 < len(args) && args[i+1] == "4096" {
				foundMaxTokens = true
				break
			}
		}
		if !foundMaxTokens {
			t.Error("expected --max-tokens 4096 in args")
		}
	})

	t.Run("includes output format", func(t *testing.T) {
		opts := &AgentOptions{
			OutputFormat: "json",
		}
		args := BuildCLIArgs(opts)

		foundFormat := false
		for i, arg := range args {
			if arg == "--output-format" && i+1 < len(args) && args[i+1] == "json" {
				foundFormat = true
				break
			}
		}
		if !foundFormat {
			t.Error("expected --output-format json in args")
		}
	})

	t.Run("defaults output format to stream-json", func(t *testing.T) {
		opts := &AgentOptions{}
		args := BuildCLIArgs(opts)

		foundFormat := false
		for i, arg := range args {
			if arg == "--output-format" && i+1 < len(args) && args[i+1] == "stream-json" {
				foundFormat = true
				break
			}
		}
		if !foundFormat {
			t.Error("expected --output-format stream-json in args")
		}
	})

	t.Run("includes debug and verbose flags", func(t *testing.T) {
		opts := &AgentOptions{
			Debug:   true,
			Verbose: true,
		}
		args := BuildCLIArgs(opts)

		foundDebug := false
		foundVerbose := false
		for _, arg := range args {
			if arg == "--debug" {
				foundDebug = true
			}
			if arg == "--verbose" {
				foundVerbose = true
			}
		}
		if !foundDebug {
			t.Error("expected --debug in args")
		}
		if !foundVerbose {
			t.Error("expected --verbose in args")
		}
	})

	t.Run("includes permission mode", func(t *testing.T) {
		opts := &AgentOptions{
			PermissionMode: PermissionModeBypassPermission,
		}
		args := BuildCLIArgs(opts)

		found := false
		for _, arg := range args {
			if arg == "--dangerously-skip-permissions" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected --dangerously-skip-permissions in args")
		}
	})
}
