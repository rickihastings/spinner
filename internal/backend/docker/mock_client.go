package docker

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of Client for testing.
type MockDockerClient struct {
	mock.Mock
}

// BuildImage mocks the BuildImage method.
func (m *MockDockerClient) BuildImage(ctx context.Context, config BuildConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

// RunContainer mocks the RunContainer method.
func (m *MockDockerClient) RunContainer(ctx context.Context, args []string, containerName string) (ContainerResult, error) {
	callArgs := m.Called(ctx, args, containerName)
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

// StartContainer mocks the StartContainer method.
func (m *MockDockerClient) StartContainer(ctx context.Context, name string) (ContainerResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerResult), args.Error(1)
}

// StopContainer mocks the StopContainer method.
func (m *MockDockerClient) StopContainer(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// LogsContainer mocks the LogsContainer method.
func (m *MockDockerClient) LogsContainer(ctx context.Context, name string) ([]byte, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]byte), args.Error(1)
}

// VerifyContainerStatus mocks the VerifyContainerStatus method.
func (m *MockDockerClient) VerifyContainerStatus(ctx context.Context, name string) (ContainerResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(ContainerResult), args.Error(1)
}

// StreamContainerLogs mocks the StreamContainerLogs method.
func (m *MockDockerClient) StreamContainerLogs(ctx context.Context, name string, opts LogStreamOptions) (<-chan LogEvent, error) {
	args := m.Called(ctx, name, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(<-chan LogEvent), args.Error(1)
}

// ListContainers mocks the ListContainers method.
func (m *MockDockerClient) ListContainers(ctx context.Context, filterLabels map[string]string) ([]ContainerListEntry, error) {
	args := m.Called(ctx, filterLabels)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]ContainerListEntry), args.Error(1)
}

// ContainerEnvVars mocks the ContainerEnvVars method.
func (m *MockDockerClient) ContainerEnvVars(ctx context.Context, name string) (map[string]string, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(map[string]string), args.Error(1)
}

// Ensure MockDockerClient implements Client interface.
var _ Client = (*MockDockerClient)(nil)
