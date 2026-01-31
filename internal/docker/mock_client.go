package docker

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of DockerClient for testing.
type MockDockerClient struct {
	mock.Mock
}

// BuildImage mocks the BuildImage method.
func (m *MockDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

// RunContainer mocks the RunContainer method.
func (m *MockDockerClient) RunContainer(ctx context.Context, config SpinConfig, containerName string, hasNpmrc bool) (ContainerResult, error) {
	callArgs := m.Called(ctx, config, containerName, hasNpmrc)
	return callArgs.Get(0).(ContainerResult), callArgs.Error(1)
}

// ImageExists mocks the ImageExists method.
func (m *MockDockerClient) ImageExists(ctx context.Context, image string) (bool, error) {
	args := m.Called(ctx, image)
	return args.Bool(0), args.Error(1)
}

// ContainerExists mocks the ContainerExists method.
func (m *MockDockerClient) ContainerExists(ctx context.Context, name string) (ContainerStatus, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerStatus), args.Error(1)
}

// RemoveContainer mocks the RemoveContainer method.
func (m *MockDockerClient) RemoveContainer(ctx context.Context, name string) (ContainerResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerResult), args.Error(1)
}

// RestartContainer mocks the RestartContainer method.
func (m *MockDockerClient) RestartContainer(ctx context.Context, name string) (ContainerResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerResult), args.Error(1)
}

// VerifyContainerStatus mocks the VerifyContainerStatus method.
func (m *MockDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerResult), args.Error(1)
}
