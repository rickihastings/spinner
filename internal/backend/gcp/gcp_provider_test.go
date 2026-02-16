package gcp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestProvider(client Client) *Provider {
	return NewGCPProvider(client, "test-project", "us-central1-a", "test-bucket")
}

func TestNewGCPProvider(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := NewGCPProvider(mockClient, "proj", "zone", "bucket")

	assert.Equal(t, "proj", p.project)
	assert.Equal(t, "zone", p.zone)
	assert.Equal(t, "bucket", p.bucket)
	assert.NotNil(t, p.client)
}

func TestProviderInstanceName(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/my-repo.git",
		Branch: "main",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	name := p.InstanceName(config)
	assert.Equal(t, "spinner-my-env-my-repo-main", name)
}

func TestProviderInstanceNameWithoutBranch(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	config := provider.CreateConfig{
		Repo: "https://github.com/user/my-repo.git",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	name := p.InstanceName(config)
	assert.Equal(t, "spinner-my-env-my-repo", name)
}

func TestProviderStatusRunning(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	running := "RUNNING"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name:   strPtr("test-vm"),
			Status: &running,
		}, nil)

	status, err := p.Status(context.Background(), "test-vm")
	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusRunning, status)
	mockClient.AssertExpectations(t)
}

func TestProviderStatusStopped(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	terminated := "TERMINATED"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name:   strPtr("test-vm"),
			Status: &terminated,
		}, nil)

	status, err := p.Status(context.Background(), "test-vm")
	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusStopped, status)
	mockClient.AssertExpectations(t)
}

func TestProviderStatusNotFound(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil, fmt.Errorf("googleapi: Error 404: The resource was not found"))

	status, err := p.Status(context.Background(), "test-vm")
	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusNone, status)
	mockClient.AssertExpectations(t)
}

func TestProviderStatusAPIError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil, fmt.Errorf("permission denied"))

	status, err := p.Status(context.Background(), "test-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get instance status")
	assert.Equal(t, provider.InstanceStatusNone, status)
	mockClient.AssertExpectations(t)
}

func TestProviderStartSuccess(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	fingerprint := "abc123"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name: strPtr("test-vm"),
			Metadata: &computepb.Metadata{
				Fingerprint: &fingerprint,
				Items: []*computepb.Items{
					{Key: strPtr("PROMPT"), Value: strPtr("old prompt")},
				},
			},
		}, nil)
	mockClient.On("SetMetadata", mock.Anything, "test-project", "us-central1-a", "test-vm", mock.Anything).
		Return(nil)
	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)

	config := provider.CreateConfig{Prompt: "new prompt"}
	instance, err := p.Start(context.Background(), "test-vm", config)
	assert.NoError(t, err)
	assert.Equal(t, "test-vm", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	mockClient.AssertExpectations(t)
}

func TestProviderStartError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	fingerprint := "abc123"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name: strPtr("test-vm"),
			Metadata: &computepb.Metadata{
				Fingerprint: &fingerprint,
				Items:       []*computepb.Items{},
			},
		}, nil)
	mockClient.On("SetMetadata", mock.Anything, "test-project", "us-central1-a", "test-vm", mock.Anything).
		Return(nil)
	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(fmt.Errorf("instance start failed"))

	config := provider.CreateConfig{Prompt: "Fix the bug"}
	instance, err := p.Start(context.Background(), "test-vm", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start instance")
	assert.Nil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderStopSuccess(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("StopInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)

	err := p.Stop(context.Background(), "test-vm")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestProviderStopError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("StopInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(fmt.Errorf("failed to stop instance"))

	err := p.Stop(context.Background(), "test-vm")
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestProviderRestartSuccess(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	fingerprint := "abc123"

	mockClient.On("StopInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name: strPtr("test-vm"),
			Metadata: &computepb.Metadata{
				Fingerprint: &fingerprint,
				Items:       []*computepb.Items{},
			},
		}, nil)
	mockClient.On("SetMetadata", mock.Anything, "test-project", "us-central1-a", "test-vm", mock.Anything).
		Return(nil)
	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)

	instance, err := p.Restart(context.Background(), "test-vm")
	assert.NoError(t, err)
	assert.Equal(t, "test-vm", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	mockClient.AssertExpectations(t)
}

func TestProviderRestartStopError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("StopInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(fmt.Errorf("stop failed"))

	instance, err := p.Restart(context.Background(), "test-vm")
	assert.Error(t, err)
	assert.Nil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderRemoveSuccess(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)
	mockClient.On("DeleteObjectsWithPrefix", mock.Anything, "test-bucket", "test-vm/").
		Return(nil)

	err := p.Remove(context.Background(), "test-vm")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestProviderRemoveError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(fmt.Errorf("delete failed"))

	err := p.Remove(context.Background(), "test-vm")
	assert.Error(t, err)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateImageNotFound(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("GetImage", mock.Anything, "test-project", "nonexistent").
		Return(nil, fmt.Errorf("image not found"))

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "nonexistent",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.Contains(t, err.Error(), "run setup first")
	assert.Nil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateInstanceError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance fails
	mockClient.On("CreateInstance", mock.Anything, mock.Anything).
		Return(fmt.Errorf("quota exceeded"))

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create instance")
	assert.Nil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateSuccess(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance succeeds - use mock.MatchedBy to verify key fields
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		return config.Name == "spinner-my-env-repo" &&
			config.Project == "test-project" &&
			config.Zone == "us-central1-a" &&
			config.ImageName == "my-env" &&
			config.ImageProject == "test-project" &&
			config.MachineType == "e2-standard-2" &&
			config.DiskSizeGB == 30 &&
			config.ExternalIP == true &&
			config.Labels["spinner-managed"] == "true" &&
			config.Labels["spinner-image"] == "my-env" &&
			config.Metadata["REPO_URL"] == "https://github.com/user/repo.git" &&
			config.Metadata["PROMPT"] == "Fix the bug" &&
			config.Metadata["MAX_ITERATIONS"] == "100" &&
			config.Metadata["SPINNER_LOG_BUCKET"] == "test-bucket" &&
			config.Metadata["SPINNER_STATE_BUCKET"] == "test-bucket" &&
			config.Metadata["SPINNER_INSTANCE_NAME"] == "spinner-my-env-repo"
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.Equal(t, "spinner-my-env-repo", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithModel(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		return config.Metadata["ANTHROPIC_MODEL"] == "claude-sonnet-4-5-20250929"
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Model:  "claude-sonnet-4-5-20250929",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithoutModel(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		return config.Metadata["ANTHROPIC_MODEL"] == ""
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderStartWithModel(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	fingerprint := "abc123"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name: strPtr("test-vm"),
			Metadata: &computepb.Metadata{
				Fingerprint: &fingerprint,
				Items: []*computepb.Items{
					{Key: strPtr("PROMPT"), Value: strPtr("old prompt")},
				},
			},
		}, nil)
	mockClient.On("SetMetadata", mock.Anything, "test-project", "us-central1-a", "test-vm",
		mock.MatchedBy(func(m *computepb.Metadata) bool {
			// Verify ANTHROPIC_MODEL was added to metadata
			for _, item := range m.Items {
				if item.GetKey() == "ANTHROPIC_MODEL" && item.GetValue() == "claude-opus-4-6" {
					return true
				}
			}

			return false
		})).Return(nil)
	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)

	config := provider.CreateConfig{
		Prompt: "new prompt",
		Model:  "claude-opus-4-6",
	}
	instance, err := p.Start(context.Background(), "test-vm", config)
	assert.NoError(t, err)
	assert.Equal(t, "test-vm", instance.Name)
	mockClient.AssertExpectations(t)
}

func TestProviderStartWithoutModelDoesNotUpdateMetadata(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	fingerprint := "abc123"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{
			Name: strPtr("test-vm"),
			Metadata: &computepb.Metadata{
				Fingerprint: &fingerprint,
				Items: []*computepb.Items{
					{Key: strPtr("PROMPT"), Value: strPtr("old prompt")},
				},
			},
		}, nil)
	mockClient.On("SetMetadata", mock.Anything, "test-project", "us-central1-a", "test-vm",
		mock.MatchedBy(func(m *computepb.Metadata) bool {
			// Verify ANTHROPIC_MODEL was NOT added when model is empty
			for _, item := range m.Items {
				if item.GetKey() == "ANTHROPIC_MODEL" {
					return false
				}
			}

			return true
		})).Return(nil)
	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil)

	config := provider.CreateConfig{
		Prompt: "new prompt",
	}
	instance, err := p.Start(context.Background(), "test-vm", config)
	assert.NoError(t, err)
	assert.Equal(t, "test-vm", instance.Name)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithCustomOptions(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance with custom machine type and disk size
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		return config.MachineType == "n2-standard-4" &&
			config.DiskSizeGB == 50
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:          "https://github.com/user/repo.git",
		Prompt:        "Fix the bug",
		Branch:        "feature",
		MaxIterations: "50",
		Options: map[string]string{
			"image":        "my-env",
			"machine-type": "n2-standard-4",
			"disk-size":    "50",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateNoBucket(t *testing.T) {
	mockClient := &MockGCPClient{}
	// Provider with no bucket
	p := NewGCPProvider(mockClient, "test-project", "us-central1-a", "")

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// Verify no log/state bucket metadata when bucket is empty
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		_, hasLogBucket := config.Metadata["SPINNER_LOG_BUCKET"]
		_, hasStateBucket := config.Metadata["SPINNER_STATE_BUCKET"]

		return !hasLogBucket && !hasStateBucket
	})).Return(nil)

	config := provider.CreateConfig{
		Repo: "https://github.com/user/repo.git",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "not found in message",
			err:      fmt.Errorf("googleapi: Error 404: The resource was not found"),
			expected: true,
		},
		{
			name:     "notFound code",
			err:      fmt.Errorf("rpc error: code = NotFound"),
			expected: true,
		},
		{
			name:     "404 status",
			err:      fmt.Errorf("failed to get instance: 404"),
			expected: true,
		},
		{
			name:     "permission denied",
			err:      fmt.Errorf("permission denied"),
			expected: false,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("connection timeout"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNotFoundError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProviderLogs_ReadsFromGCS(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	ctx := context.Background()
	logData := []byte("iteration 1 output\n")
	mockClient.On("ReadObject", ctx, "test-bucket", "test-vm/logs/raw.log").Return(logData, nil)

	reader, err := p.Logs(ctx, "test-vm")
	assert.NoError(t, err)
	assert.NotNil(t, reader)

	data, _ := io.ReadAll(reader)
	assert.Equal(t, "iteration 1 output\n", string(data))

	_ = reader.Close()

	mockClient.AssertExpectations(t)
}

func TestProviderLogs_NoBucket(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := NewGCPProvider(mockClient, "proj", "zone", "")

	_, err := p.Logs(context.Background(), "test-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no state bucket")
}

func TestProviderWatchLogs_NoBucket(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := NewGCPProvider(mockClient, "proj", "zone", "")

	ch := make(chan string)
	err := p.WatchLogs(context.Background(), "test-vm", 10, ch)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no state bucket")
}

func TestProviderWatchMetrics_StoppedVM(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	terminated := "TERMINATED"
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&computepb.Instance{Status: &terminated}, nil)

	ch := make(chan provider.ContainerMetrics, 10)
	err := p.WatchMetrics(context.Background(), "test-vm", ch)
	assert.NoError(t, err)

	select {
	case m := <-ch:
		assert.Equal(t, provider.StateStopped, m.State)
	default:
		t.Fatal("expected metrics in channel")
	}

	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithCustomEnvVars(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance with custom env vars - verify they have SPINNER_ENV_ prefix
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		return config.Metadata["SPINNER_ENV_NPM_TOKEN"] == "abc123" &&
			config.Metadata["SPINNER_ENV_API_KEY"] == "secret456"
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
		EnvVars: map[string]string{
			"NPM_TOKEN": "abc123",
			"API_KEY":   "secret456",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithEmptyEnvVars(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance with empty env vars map - verify no SPINNER_ENV_ keys
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		// Verify no metadata keys start with SPINNER_ENV_
		for key := range config.Metadata {
			if len(key) >= 12 && key[:12] == "SPINNER_ENV_" {
				return false
			}
		}

		return true
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
		EnvVars: map[string]string{},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateCustomEnvVarsNoCollision(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance - verify custom env vars don't overwrite internal metadata
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		// Internal keys should be present and unchanged
		if config.Metadata["REPO_URL"] != "https://github.com/user/repo.git" {
			return false
		}

		if config.Metadata["PROMPT"] != "Fix the bug" {
			return false
		}

		// Custom var should be prefixed
		if config.Metadata["SPINNER_ENV_MY_VAR"] != "value" {
			return false
		}

		// Verify internal keys are NOT prefixed
		_, hasPrefixedRepo := config.Metadata["SPINNER_ENV_REPO_URL"]

		return !hasPrefixedRepo
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
		EnvVars: map[string]string{
			"MY_VAR": "value",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreatePassesGitUserConfig(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance - verify git user config is passed in metadata
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		_, hasName := config.Metadata["GIT_USER_NAME"]
		_, hasEmail := config.Metadata["GIT_USER_EMAIL"]

		return hasName && hasEmail
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:   "https://github.com/user/repo.git",
		Prompt: "Fix the bug",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestGCPGitConfigValue_InvalidKey(t *testing.T) {
	result := gcpGitConfigValue("nonexistent.key.that.doesnt.exist")
	assert.Empty(t, result, "should return empty for non-existent git config key")
}

func TestProviderCreateWithEnvFile(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Create temp env file
	tmpFile := t.TempDir() + "/.env"
	envContent := "DATABASE_URL=postgres://localhost\nAPI_KEY=secret123"
	err := os.WriteFile(tmpFile, []byte(envContent), 0644)
	assert.NoError(t, err)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance - verify env file is base64-encoded in metadata
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		envFileB64, hasEnvFile := config.Metadata["SPINNER_ENV_FILE"]
		if !hasEnvFile {
			return false
		}

		// Decode and verify content
		decoded, err := base64.StdEncoding.DecodeString(envFileB64)
		if err != nil {
			return false
		}

		return string(decoded) == envContent
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Prompt:  "Fix the bug",
		EnvFile: tmpFile,
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderListWithInstances(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	running := "RUNNING"
	terminated := "TERMINATED"

	mockClient.On("ListInstances", mock.Anything, "test-project", "us-central1-a", "labels.spinner-managed=true").
		Return([]*computepb.Instance{
			{
				Name:   strPtr("spinner-default-my-repo"),
				Status: &running,
				Labels: map[string]string{
					"spinner-managed": "true",
					"spinner-image":   "default",
					"spinner-repo":    "my-repo",
				},
				Metadata: &computepb.Metadata{
					Items: []*computepb.Items{
						{Key: strPtr("ANTHROPIC_MODEL"), Value: strPtr("claude-sonnet-4-5-20250929")},
						{Key: strPtr("MAX_ITERATIONS"), Value: strPtr("100")},
						{Key: strPtr("BRANCH"), Value: strPtr("main")},
					},
				},
			},
			{
				Name:   strPtr("spinner-default-other-repo"),
				Status: &terminated,
				Labels: map[string]string{
					"spinner-managed": "true",
					"spinner-image":   "default",
					"spinner-repo":    "other-repo",
				},
			},
		}, nil)

	// State enrichment for first instance
	mockClient.On("ObjectExists", mock.Anything, "test-bucket", "spinner-default-my-repo/state.json").
		Return(true, nil)
	mockClient.On("ReadObject", mock.Anything, "test-bucket", "spinner-default-my-repo/state.json").
		Return([]byte(`{"iteration":5,"status":"running","branch":"feature","started_at":"2026-02-15T10:00:00Z","last_updated":"2026-02-15T11:00:00Z"}`), nil)

	// State enrichment for second instance - no state file
	mockClient.On("ObjectExists", mock.Anything, "test-bucket", "spinner-default-other-repo/state.json").
		Return(false, nil)

	instances, err := p.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, instances, 2)

	// First instance: running with full state
	assert.Equal(t, "spinner-default-my-repo", instances[0].Name)
	assert.Equal(t, provider.InstanceStatusRunning, instances[0].Status)
	assert.Equal(t, "gcp", instances[0].Backend)
	assert.Equal(t, "default", instances[0].Image)
	assert.Equal(t, "my-repo", instances[0].Repo)
	assert.Equal(t, "claude-sonnet-4-5-20250929", instances[0].Agent)
	assert.Equal(t, 100, instances[0].MaxIterations)
	assert.Equal(t, 5, instances[0].Iteration)
	assert.Equal(t, "running", instances[0].AgentStatus)
	// State file branch overrides metadata branch
	assert.Equal(t, "feature", instances[0].Branch)
	assert.NotNil(t, instances[0].StartedAt)
	assert.NotNil(t, instances[0].LastUpdated)

	// Second instance: stopped without state
	assert.Equal(t, "spinner-default-other-repo", instances[1].Name)
	assert.Equal(t, provider.InstanceStatusStopped, instances[1].Status)
	assert.Equal(t, "gcp", instances[1].Backend)
	assert.Equal(t, "default", instances[1].Image)
	assert.Equal(t, "other-repo", instances[1].Repo)
	assert.Equal(t, 0, instances[1].Iteration)
	assert.Empty(t, instances[1].AgentStatus)

	mockClient.AssertExpectations(t)
}

func TestProviderListNoInstances(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("ListInstances", mock.Anything, "test-project", "us-central1-a", "labels.spinner-managed=true").
		Return([]*computepb.Instance{}, nil)

	instances, err := p.List(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, instances)
	mockClient.AssertExpectations(t)
}

func TestProviderListAPIError(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	mockClient.On("ListInstances", mock.Anything, "test-project", "us-central1-a", "labels.spinner-managed=true").
		Return(nil, fmt.Errorf("permission denied"))

	instances, err := p.List(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to list instances")
	assert.Nil(t, instances)
	mockClient.AssertExpectations(t)
}

func TestProviderListNoBucket(t *testing.T) {
	mockClient := &MockGCPClient{}
	// Provider with no bucket configured
	p := NewGCPProvider(mockClient, "test-project", "us-central1-a", "")

	running := "RUNNING"
	mockClient.On("ListInstances", mock.Anything, "test-project", "us-central1-a", "labels.spinner-managed=true").
		Return([]*computepb.Instance{
			{
				Name:   strPtr("spinner-default-repo"),
				Status: &running,
				Labels: map[string]string{
					"spinner-managed": "true",
					"spinner-image":   "default",
				},
				Metadata: &computepb.Metadata{
					Items: []*computepb.Items{
						{Key: strPtr("MAX_ITERATIONS"), Value: strPtr("50")},
					},
				},
			},
		}, nil)

	instances, err := p.List(context.Background())
	assert.NoError(t, err)
	assert.Len(t, instances, 1)

	// Instance visible but without execution state
	assert.Equal(t, "spinner-default-repo", instances[0].Name)
	assert.Equal(t, provider.InstanceStatusRunning, instances[0].Status)
	assert.Equal(t, 50, instances[0].MaxIterations)
	assert.Equal(t, 0, instances[0].Iteration)
	assert.Empty(t, instances[0].AgentStatus)

	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithEnvFileNotFound(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	config := provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Prompt:  "Fix the bug",
		EnvFile: "/nonexistent/.env",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read env file")
	assert.Nil(t, instance)
	mockClient.AssertExpectations(t)
}

func TestProviderCreateWithEmptyEnvFile(t *testing.T) {
	mockClient := &MockGCPClient{}
	p := newTestProvider(mockClient)

	// Image exists
	mockClient.On("GetImage", mock.Anything, "test-project", "my-env").
		Return(&computepb.Image{Name: strPtr("my-env")}, nil)

	// CreateInstance - verify no SPINNER_ENV_FILE metadata when EnvFile is empty
	mockClient.On("CreateInstance", mock.Anything, mock.MatchedBy(func(config instanceConfig) bool {
		_, hasEnvFile := config.Metadata["SPINNER_ENV_FILE"]
		return !hasEnvFile
	})).Return(nil)

	config := provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Prompt:  "Fix the bug",
		EnvFile: "",
		Options: map[string]string{
			"image": "my-env",
		},
	}

	instance, err := p.Create(context.Background(), config)
	assert.NoError(t, err)
	assert.NotNil(t, instance)
	mockClient.AssertExpectations(t)
}
