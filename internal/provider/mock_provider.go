package provider

import (
	"context"
	"io"

	"github.com/stretchr/testify/mock"
)

// MockProvider is a mock implementation of Provider for testing.
type MockProvider struct {
	mock.Mock
}

// Setup mocks the Setup method.
func (m *MockProvider) Setup(ctx context.Context, config SetupConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

// Create mocks the Create method.
func (m *MockProvider) Create(ctx context.Context, config CreateConfig) (*Instance, error) {
	args := m.Called(ctx, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Instance), args.Error(1)
}

// Start mocks the Start method.
func (m *MockProvider) Start(ctx context.Context, name string, config CreateConfig) (*Instance, error) {
	args := m.Called(ctx, name, config)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Instance), args.Error(1)
}

// Restart mocks the Restart method.
func (m *MockProvider) Restart(ctx context.Context, name string) (*Instance, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*Instance), args.Error(1)
}

// Stop mocks the Stop method.
func (m *MockProvider) Stop(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// Remove mocks the Remove method.
func (m *MockProvider) Remove(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

// Logs mocks the Logs method.
func (m *MockProvider) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Status mocks the Status method.
func (m *MockProvider) Status(ctx context.Context, name string) (InstanceStatus, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(InstanceStatus), args.Error(1)
}

// InstanceName mocks the InstanceName method.
func (m *MockProvider) InstanceName(config CreateConfig) string {
	args := m.Called(config)
	return args.String(0)
}

// WatchLogs mocks the WatchLogs method.
func (m *MockProvider) WatchLogs(ctx context.Context, name string, tailLines int, ch chan<- string) error {
	args := m.Called(ctx, name, tailLines, ch)
	return args.Error(0)
}

// WatchMetrics mocks the WatchMetrics method.
func (m *MockProvider) WatchMetrics(ctx context.Context, name string, ch chan<- ContainerMetrics) error {
	args := m.Called(ctx, name, ch)
	return args.Error(0)
}

// GetInstanceMetadata mocks the GetInstanceMetadata method.
func (m *MockProvider) GetInstanceMetadata(ctx context.Context, name string) (*InstanceMetadata, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*InstanceMetadata), args.Error(1)
}
