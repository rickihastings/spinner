package gcp

import (
	"context"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/stretchr/testify/mock"
)

// MockGCPClient is a mock implementation of Client for testing.
type MockGCPClient struct {
	mock.Mock
}

// CreateInstance mocks the CreateInstance method.
func (m *MockGCPClient) CreateInstance(ctx context.Context, config instanceConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

// GetInstance mocks the GetInstance method.
func (m *MockGCPClient) GetInstance(ctx context.Context, project, zone, name string) (*computepb.Instance, error) {
	args := m.Called(ctx, project, zone, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*computepb.Instance), args.Error(1)
}

// SetMetadata mocks the SetMetadata method.
func (m *MockGCPClient) SetMetadata(ctx context.Context, project, zone, name string, metadata *computepb.Metadata) error {
	args := m.Called(ctx, project, zone, name, metadata)
	return args.Error(0)
}

// StartInstance mocks the StartInstance method.
func (m *MockGCPClient) StartInstance(ctx context.Context, project, zone, name string) error {
	args := m.Called(ctx, project, zone, name)
	return args.Error(0)
}

// StopInstance mocks the StopInstance method.
func (m *MockGCPClient) StopInstance(ctx context.Context, project, zone, name string) error {
	args := m.Called(ctx, project, zone, name)
	return args.Error(0)
}

// ResetInstance mocks the ResetInstance method.
func (m *MockGCPClient) ResetInstance(ctx context.Context, project, zone, name string) error {
	args := m.Called(ctx, project, zone, name)
	return args.Error(0)
}

// DeleteInstance mocks the DeleteInstance method.
func (m *MockGCPClient) DeleteInstance(ctx context.Context, project, zone, name string) error {
	args := m.Called(ctx, project, zone, name)
	return args.Error(0)
}

// CreateImage mocks the CreateImage method.
func (m *MockGCPClient) CreateImage(ctx context.Context, project string, config imageConfig) error {
	args := m.Called(ctx, project, config)
	return args.Error(0)
}

// GetImage mocks the GetImage method.
func (m *MockGCPClient) GetImage(ctx context.Context, project, name string) (*computepb.Image, error) {
	args := m.Called(ctx, project, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*computepb.Image), args.Error(1)
}

// DeleteImage mocks the DeleteImage method.
func (m *MockGCPClient) DeleteImage(ctx context.Context, project, name string) error {
	args := m.Called(ctx, project, name)
	return args.Error(0)
}

// GetSerialPortOutput mocks the GetSerialPortOutput method.
func (m *MockGCPClient) GetSerialPortOutput(ctx context.Context, project, zone, name string, start int64) (*serialPortOutput, error) {
	args := m.Called(ctx, project, zone, name, start)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*serialPortOutput), args.Error(1)
}

// WriteObject mocks the WriteObject method.
func (m *MockGCPClient) WriteObject(ctx context.Context, bucket, object string, data []byte) error {
	args := m.Called(ctx, bucket, object, data)
	return args.Error(0)
}

// ReadObject mocks the ReadObject method.
func (m *MockGCPClient) ReadObject(ctx context.Context, bucket, object string) ([]byte, error) {
	args := m.Called(ctx, bucket, object)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]byte), args.Error(1)
}

// ReadObjectRange mocks the ReadObjectRange method.
func (m *MockGCPClient) ReadObjectRange(ctx context.Context, bucket, object string, offset int64) ([]byte, error) {
	args := m.Called(ctx, bucket, object, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]byte), args.Error(1)
}

// ObjectSize mocks the ObjectSize method.
func (m *MockGCPClient) ObjectSize(ctx context.Context, bucket, object string) (int64, error) {
	args := m.Called(ctx, bucket, object)
	return args.Get(0).(int64), args.Error(1)
}

// ObjectExists mocks the ObjectExists method.
func (m *MockGCPClient) ObjectExists(ctx context.Context, bucket, object string) (bool, error) {
	args := m.Called(ctx, bucket, object)
	return args.Bool(0), args.Error(1)
}

// Close mocks the Close method.
func (m *MockGCPClient) Close() error {
	args := m.Called()
	return args.Error(0)
}
