package exec

import (
	"os"
	"testing"
)

func TestLoadConfig_Success(t *testing.T) {
	// Set required env vars
	_ = os.Setenv("PROMPT", "test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "10")
	_ = os.Setenv("BRANCH", "main")
	_ = os.Setenv("LOG_DIR", "/tmp/logs")
	_ = os.Setenv("CONTAINER_NAME", "test-container")

	defer clearEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.Prompt != "test prompt" {
		t.Errorf("expected prompt 'test prompt', got %s", config.Prompt)
	}

	if config.MaxIterations != 10 {
		t.Errorf("expected max iterations 10, got %d", config.MaxIterations)
	}

	if config.Branch != "main" {
		t.Errorf("expected branch 'main', got %s", config.Branch)
	}

	if config.LogDir != "/tmp/logs" {
		t.Errorf("expected log dir '/tmp/logs', got %s", config.LogDir)
	}

	if config.ContainerName != "test-container" {
		t.Errorf("expected container name 'test-container', got %s", config.ContainerName)
	}
}

func TestLoadConfig_MissingPrompt(t *testing.T) {
	_ = os.Setenv("MAX_ITERATIONS", "10")

	defer clearEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing PROMPT, got nil")
	}
}

func TestLoadConfig_MissingMaxIterations(t *testing.T) {
	_ = os.Setenv("PROMPT", "test prompt")

	defer clearEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for missing MAX_ITERATIONS, got nil")
	}
}

func TestLoadConfig_InvalidMaxIterations(t *testing.T) {
	_ = os.Setenv("PROMPT", "test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "not a number")

	defer clearEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for invalid MAX_ITERATIONS, got nil")
	}
}

func TestLoadConfig_ZeroMaxIterations(t *testing.T) {
	_ = os.Setenv("PROMPT", "test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "0")

	defer clearEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for MAX_ITERATIONS = 0, got nil")
	}
}

func TestLoadConfig_NegativeMaxIterations(t *testing.T) {
	_ = os.Setenv("PROMPT", "test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "-5")

	defer clearEnv()

	_, err := LoadConfig()
	if err == nil {
		t.Error("expected error for negative MAX_ITERATIONS, got nil")
	}
}

func TestLoadConfig_OptionalFieldsMissing(t *testing.T) {
	_ = os.Setenv("PROMPT", "test prompt")
	_ = os.Setenv("MAX_ITERATIONS", "5")

	defer clearEnv()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Optional fields should be empty strings, not errors
	if config.Branch != "" {
		t.Errorf("expected empty branch, got %s", config.Branch)
	}

	if config.LogDir != "" {
		t.Errorf("expected empty log dir, got %s", config.LogDir)
	}

	if config.ContainerName != "" {
		t.Errorf("expected empty container name, got %s", config.ContainerName)
	}
}

func clearEnv() {
	_ = os.Unsetenv("PROMPT")
	_ = os.Unsetenv("MAX_ITERATIONS")
	_ = os.Unsetenv("BRANCH")
	_ = os.Unsetenv("LOG_DIR")
	_ = os.Unsetenv("CONTAINER_NAME")
}
