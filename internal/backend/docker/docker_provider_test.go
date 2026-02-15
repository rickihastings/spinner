package docker

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestDockerProvider_Setup(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("BuildImage", ctx, BuildConfig{
		Name:      "test-env",
		BaseImage: "ubuntu:22.04",
	}).Return(nil)

	err := p.Setup(ctx, provider.SetupConfig{
		Name:    "test-env",
		Options: map[string]string{"base-image": "ubuntu:22.04"},
	})

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDockerProvider_Setup_Error(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("BuildImage", ctx, mock.Anything).Return(assert.AnError)

	err := p.Setup(ctx, provider.SetupConfig{
		Name:    "test-env",
		Options: map[string]string{},
	})

	assert.Error(t, err)
}

func TestDockerProvider_Setup_NonExistentDockerfile(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	err := p.Setup(ctx, provider.SetupConfig{
		Name:    "test-env",
		Options: map[string]string{"dockerfile": "/nonexistent/path/Dockerfile"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dockerfile not found at path")
	client.AssertNotCalled(t, "BuildImage")
}

func TestDockerProvider_InstanceName(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)

	name := p.InstanceName(provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
	})

	assert.Equal(t, "spinner-test-repo", name)
}

func TestDockerProvider_InstanceName_WithBranch(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)

	name := p.InstanceName(provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Branch:  "main",
		Options: map[string]string{"image": "spinner:test"},
	})

	assert.Equal(t, "spinner-test-repo-main", name)
}

func TestDockerProvider_Create_ImageNotFound(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:missing").Return(false, nil)

	_, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:missing"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDockerProvider_Create_Success(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)

	instance, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "spinner-test-repo", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_Create_RunFails(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: false, ContainerName: "spinner-test-repo", Error: "port conflict"}, nil,
	)

	_, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port conflict")
}

func TestDockerProvider_Create_VerifyFails(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "spinner-test-repo").Return(
		ContainerResult{Success: false, ContainerName: "spinner-test-repo", Error: "exited early"}, nil,
	)

	_, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exited early")
}

func TestDockerProvider_Start_Success(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("StartContainer", ctx, "test-container").Return(
		ContainerResult{Success: true, ContainerName: "test-container"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "test-container").Return(
		ContainerResult{Success: true, ContainerName: "test-container"}, nil,
	)

	config := provider.CreateConfig{Prompt: "Fix the bug"}
	instance, err := p.Start(ctx, "test-container", config)

	assert.NoError(t, err)
	assert.Equal(t, "test-container", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_Start_Fails(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("StartContainer", ctx, "test-container").Return(
		ContainerResult{Success: false, ContainerName: "test-container", Error: "no such container"}, nil,
	)

	config := provider.CreateConfig{Prompt: "Fix the bug"}
	_, err := p.Start(ctx, "test-container", config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such container")
}

func TestDockerProvider_Restart_Success(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("StopContainer", ctx, "test-container").Return(nil)
	client.On("StartContainer", ctx, "test-container").Return(
		ContainerResult{Success: true, ContainerName: "test-container"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "test-container").Return(
		ContainerResult{Success: true, ContainerName: "test-container"}, nil,
	)

	instance, err := p.Restart(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, "test-container", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_Restart_StopFails(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("StopContainer", ctx, "test-container").Return(assert.AnError)

	_, err := p.Restart(ctx, "test-container")

	assert.Error(t, err)
}

func TestDockerProvider_Stop_Success(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("StopContainer", ctx, "test-container").Return(nil)

	err := p.Stop(ctx, "test-container")

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDockerProvider_Remove_Success(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("RemoveContainer", ctx, "test-container").Return(
		ContainerResult{Success: true, ContainerName: "test-container"}, nil,
	)

	err := p.Remove(ctx, "test-container")

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestDockerProvider_Remove_Fails(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("RemoveContainer", ctx, "test-container").Return(
		ContainerResult{Success: false, Error: "container in use"}, nil,
	)

	err := p.Remove(ctx, "test-container")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "container in use")
}

func TestDockerProvider_Logs_Success(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("LogsContainer", ctx, "test-container").Return([]byte("log output"), nil)

	reader, err := p.Logs(ctx, "test-container")

	assert.NoError(t, err)

	defer func() { _ = reader.Close() }()

	buf := make([]byte, 10)
	n, _ := reader.Read(buf)
	assert.Equal(t, "log output", string(buf[:n]))
	client.AssertExpectations(t)
}

func TestDockerProvider_Status_Running(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ContainerExists", ctx, "test-container").Return(StatusRunning, nil)

	status, err := p.Status(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusRunning, status)
}

func TestDockerProvider_Status_Stopped(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ContainerExists", ctx, "test-container").Return(StatusStopped, nil)

	status, err := p.Status(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusStopped, status)
}

func TestDockerProvider_Status_None(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ContainerExists", ctx, "test-container").Return(StatusNone, nil)

	status, err := p.Status(ctx, "test-container")

	assert.NoError(t, err)
	assert.Equal(t, provider.InstanceStatusNone, status)
}

func TestDockerProvider_Create_WithEnvVars(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)

	instance, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
		EnvVars: map[string]string{
			"NPM_TOKEN":  "npm-secret-123",
			"MY_API_KEY": "api-secret-456",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "spinner-test-repo", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_Create_WithEmptyEnvVars(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)

	instance, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
		EnvVars: map[string]string{},
	})

	assert.NoError(t, err)
	assert.Equal(t, "spinner-test-repo", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_Create_WithNilEnvVars(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "test-token")

	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ImageExists", ctx, "spinner:test").Return(true, nil)
	client.On("RunContainer", ctx, mock.Anything, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)
	client.On("VerifyContainerStatus", ctx, "spinner-test-repo").Return(
		ContainerResult{Success: true, ContainerName: "spinner-test-repo"}, nil,
	)

	instance, err := p.Create(ctx, provider.CreateConfig{
		Repo:    "https://github.com/user/repo.git",
		Options: map[string]string{"image": "spinner:test"},
		EnvVars: nil,
	})

	assert.NoError(t, err)
	assert.Equal(t, "spinner-test-repo", instance.Name)
	assert.Equal(t, provider.InstanceStatusRunning, instance.Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_List_ContainersWithLabels(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(
		[]container.Summary{
			{
				Names:  []string{"/spinner-test-repo"},
				Image:  "spinner:test",
				State:  "running",
				Labels: map[string]string{"spinner-managed": "true"},
			},
			{
				Names:  []string{"/spinner-test-other"},
				Image:  "spinner:test",
				State:  "exited",
				Labels: map[string]string{"spinner-managed": "true"},
			},
		}, nil,
	)
	client.On("ContainerEnvVars", ctx, "spinner-test-repo").Return(map[string]string{}, nil)
	client.On("ContainerEnvVars", ctx, "spinner-test-other").Return(map[string]string{}, nil)

	instances, err := p.List(ctx)

	assert.NoError(t, err)
	assert.Len(t, instances, 2)

	assert.Equal(t, "spinner-test-repo", instances[0].Name)
	assert.Equal(t, provider.InstanceStatusRunning, instances[0].Status)
	assert.Equal(t, "docker", instances[0].Backend)
	assert.Equal(t, "spinner:test", instances[0].Image)

	assert.Equal(t, "spinner-test-other", instances[1].Name)
	assert.Equal(t, provider.InstanceStatusStopped, instances[1].Status)
	client.AssertExpectations(t)
}

func TestDockerProvider_List_NoContainers(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(
		[]container.Summary{}, nil,
	)

	instances, err := p.List(ctx)

	assert.NoError(t, err)
	assert.Empty(t, instances)
	client.AssertExpectations(t)
}

func TestDockerProvider_List_DockerUnavailable(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	client.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(
		nil, assert.AnError,
	)

	instances, err := p.List(ctx)

	assert.Error(t, err)
	assert.Nil(t, instances)
	assert.Contains(t, err.Error(), "failed to list containers")
	client.AssertExpectations(t)
}

func TestDockerProvider_List_StateEnrichment(t *testing.T) {
	client := new(MockDockerClient)
	p := NewDockerProvider(client)
	ctx := context.Background()

	// Create a temp state directory that mimics ~/.spinner/<name>/state/
	homeDir, err := os.UserHomeDir()
	assert.NoError(t, err)

	containerName := "spinner-test-stateful"
	stateDir := filepath.Join(homeDir, ".spinner", containerName, "state")
	assert.NoError(t, os.MkdirAll(stateDir, 0755))

	defer func() { _ = os.RemoveAll(filepath.Join(homeDir, ".spinner", containerName)) }()

	startedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	lastUpdated := time.Date(2026, 1, 15, 12, 30, 0, 0, time.UTC)

	stateData, _ := json.Marshal(map[string]interface{}{
		"iteration":    42,
		"status":       "running",
		"branch":       "feature-branch",
		"started_at":   startedAt,
		"last_updated": lastUpdated,
	})

	assert.NoError(t, os.WriteFile(filepath.Join(stateDir, "state.json"), stateData, 0644))

	client.On("ListContainers", ctx, map[string]string{"spinner-managed": "true"}).Return(
		[]container.Summary{
			{
				Names:  []string{"/" + containerName},
				Image:  "spinner:test",
				State:  "running",
				Labels: map[string]string{"spinner-managed": "true"},
			},
		}, nil,
	)
	client.On("ContainerEnvVars", ctx, containerName).Return(map[string]string{}, nil)

	instances, err := p.List(ctx)

	assert.NoError(t, err)
	assert.Len(t, instances, 1)

	info := instances[0]
	assert.Equal(t, containerName, info.Name)
	assert.Equal(t, 42, info.Iteration)
	assert.Equal(t, "running", info.AgentStatus)
	assert.Equal(t, "feature-branch", info.Branch)
	assert.NotNil(t, info.StartedAt)
	assert.Equal(t, startedAt, *info.StartedAt)
	assert.NotNil(t, info.LastUpdated)
	assert.Equal(t, lastUpdated, *info.LastUpdated)
	client.AssertExpectations(t)
}

func TestWriteConfigOverrides_WritesModelFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	containerName := "test-container"
	config := provider.CreateConfig{
		Prompt:        "Fix the bug",
		MaxIterations: "50",
		Model:         "claude-sonnet-4-5-20250929",
	}

	err := writeConfigOverrides(containerName, config)
	assert.NoError(t, err)

	stateDir := filepath.Join(tmpHome, ".spinner", containerName, "state")

	// Verify prompt.txt
	promptData, err := os.ReadFile(filepath.Join(stateDir, "prompt.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "Fix the bug", string(promptData))

	// Verify max-iterations.txt
	maxIterData, err := os.ReadFile(filepath.Join(stateDir, "max-iterations.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "50", string(maxIterData))

	// Verify model.txt
	modelData, err := os.ReadFile(filepath.Join(stateDir, "model.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-5-20250929", string(modelData))
}

func TestWriteConfigOverrides_EmptyModel(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	containerName := "test-container"
	config := provider.CreateConfig{
		Prompt: "Fix the bug",
		Model:  "",
	}

	err := writeConfigOverrides(containerName, config)
	assert.NoError(t, err)

	stateDir := filepath.Join(tmpHome, ".spinner", containerName, "state")

	// model.txt should exist but be empty (startup.sh skips empty values)
	modelData, err := os.ReadFile(filepath.Join(stateDir, "model.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "", string(modelData))
}

func TestWriteConfigOverrides_DefaultMaxIterations(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	containerName := "test-container"
	config := provider.CreateConfig{
		Prompt:        "Fix the bug",
		MaxIterations: "",
	}

	err := writeConfigOverrides(containerName, config)
	assert.NoError(t, err)

	stateDir := filepath.Join(tmpHome, ".spinner", containerName, "state")

	// When MaxIterations is empty, should use defaultMaxIterations
	maxIterData, err := os.ReadFile(filepath.Join(stateDir, "max-iterations.txt"))
	assert.NoError(t, err)
	assert.Equal(t, defaultMaxIterations, string(maxIterData))
}
