package docker

import (
	"context"
	"testing"

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

	instance, err := p.Start(ctx, "test-container")

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

	_, err := p.Start(ctx, "test-container")

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
