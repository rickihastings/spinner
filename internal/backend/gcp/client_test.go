package gcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestMockGCPClientImplementsInterface verifies that MockGCPClient satisfies
// the Client interface at compile time.
func TestMockGCPClientImplementsInterface(t *testing.T) {
	var _ Client = (*MockGCPClient)(nil)
}

// TestRealGCPClientImplementsInterface verifies that RealGCPClient satisfies
// the Client interface at compile time.
func TestRealGCPClientImplementsInterface(t *testing.T) {
	var _ Client = (*RealGCPClient)(nil)
}

func TestMockCreateInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	config := instanceConfig{
		Name:        "test-instance",
		Project:     "test-project",
		Zone:        "us-central1-a",
		MachineType: "e2-standard-2",
		ImageName:   "spinner-test",
	}

	mockClient.On("CreateInstance", mock.Anything, config).Return(nil)

	err := mockClient.CreateInstance(context.Background(), config)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockCreateInstanceError(t *testing.T) {
	mockClient := &MockGCPClient{}

	config := instanceConfig{
		Name:    "test-instance",
		Project: "test-project",
		Zone:    "us-central1-a",
	}

	mockClient.On("CreateInstance", mock.Anything, config).
		Return(fmt.Errorf("quota exceeded"))

	err := mockClient.CreateInstance(context.Background(), config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
	mockClient.AssertExpectations(t)
}

func TestMockGetInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	expected := &GCPInstance{
		Name:   "test-instance",
		Status: "RUNNING",
	}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-instance").
		Return(expected, nil)

	instance, err := mockClient.GetInstance(context.Background(), "test-project", "us-central1-a", "test-instance")
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", instance.Name)
	assert.Equal(t, "RUNNING", instance.Status)
	mockClient.AssertExpectations(t)
}

func TestMockGetInstanceNotFound(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "nonexistent").
		Return(nil, fmt.Errorf("instance not found"))

	instance, err := mockClient.GetInstance(context.Background(), "test-project", "us-central1-a", "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, instance)
	assert.Contains(t, err.Error(), "not found")
	mockClient.AssertExpectations(t)
}

func TestMockStartInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("StartInstance", mock.Anything, "test-project", "us-central1-a", "test-instance").
		Return(nil)

	err := mockClient.StartInstance(context.Background(), "test-project", "us-central1-a", "test-instance")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockStopInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("StopInstance", mock.Anything, "test-project", "us-central1-a", "test-instance").
		Return(nil)

	err := mockClient.StopInstance(context.Background(), "test-project", "us-central1-a", "test-instance")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockResetInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("ResetInstance", mock.Anything, "test-project", "us-central1-a", "test-instance").
		Return(nil)

	err := mockClient.ResetInstance(context.Background(), "test-project", "us-central1-a", "test-instance")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockDeleteInstance(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "test-instance").
		Return(nil)

	err := mockClient.DeleteInstance(context.Background(), "test-project", "us-central1-a", "test-instance")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockCreateImage(t *testing.T) {
	mockClient := &MockGCPClient{}

	config := imageConfig{
		Name:       "spinner-test",
		SourceDisk: "projects/test-project/zones/us-central1-a/disks/temp-bake-vm",
	}

	mockClient.On("CreateImage", mock.Anything, "test-project", config).Return(nil)

	err := mockClient.CreateImage(context.Background(), "test-project", config)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockGetImage(t *testing.T) {
	mockClient := &MockGCPClient{}

	expected := &GCPImage{
		Name: "spinner-test",
	}

	mockClient.On("GetImage", mock.Anything, "test-project", "spinner-test").
		Return(expected, nil)

	image, err := mockClient.GetImage(context.Background(), "test-project", "spinner-test")
	assert.NoError(t, err)
	assert.Equal(t, "spinner-test", image.Name)
	mockClient.AssertExpectations(t)
}

func TestMockDeleteImage(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("DeleteImage", mock.Anything, "test-project", "spinner-test").Return(nil)

	err := mockClient.DeleteImage(context.Background(), "test-project", "spinner-test")
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockGetSerialPortOutput(t *testing.T) {
	mockClient := &MockGCPClient{}

	expected := &serialPortOutput{
		Contents: "SPINNER_BAKE_COMPLETE",
		Next:     1024,
	}

	mockClient.On("GetSerialPortOutput", mock.Anything, "test-project", "us-central1-a", "temp-vm", int64(0)).
		Return(expected, nil)

	output, err := mockClient.GetSerialPortOutput(context.Background(), "test-project", "us-central1-a", "temp-vm", 0)
	assert.NoError(t, err)
	assert.Equal(t, "SPINNER_BAKE_COMPLETE", output.Contents)
	assert.Equal(t, int64(1024), output.Next)
	mockClient.AssertExpectations(t)
}

func TestMockWriteObject(t *testing.T) {
	mockClient := &MockGCPClient{}

	data := []byte(`{"status":"running"}`)
	mockClient.On("WriteObject", mock.Anything, "my-bucket", "instance/state.json", data).
		Return(nil)

	err := mockClient.WriteObject(context.Background(), "my-bucket", "instance/state.json", data)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestMockReadObject(t *testing.T) {
	mockClient := &MockGCPClient{}

	expected := []byte(`{"status":"completed"}`)
	mockClient.On("ReadObject", mock.Anything, "my-bucket", "instance/state.json").
		Return(expected, nil)

	data, err := mockClient.ReadObject(context.Background(), "my-bucket", "instance/state.json")
	assert.NoError(t, err)
	assert.Equal(t, expected, data)
	mockClient.AssertExpectations(t)
}

func TestMockReadObjectRange(t *testing.T) {
	mockClient := &MockGCPClient{}

	expected := []byte("new log content")
	mockClient.On("ReadObjectRange", mock.Anything, "my-bucket", "instance/logs/raw.log", int64(100)).
		Return(expected, nil)

	data, err := mockClient.ReadObjectRange(context.Background(), "my-bucket", "instance/logs/raw.log", 100)
	assert.NoError(t, err)
	assert.Equal(t, expected, data)
	mockClient.AssertExpectations(t)
}

func TestMockObjectSize(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("ObjectSize", mock.Anything, "my-bucket", "instance/logs/raw.log").
		Return(int64(2048), nil)

	size, err := mockClient.ObjectSize(context.Background(), "my-bucket", "instance/logs/raw.log")
	assert.NoError(t, err)
	assert.Equal(t, int64(2048), size)
	mockClient.AssertExpectations(t)
}

func TestMockObjectExists(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("ObjectExists", mock.Anything, "my-bucket", "instance/state.json").
		Return(true, nil)

	exists, err := mockClient.ObjectExists(context.Background(), "my-bucket", "instance/state.json")
	assert.NoError(t, err)
	assert.True(t, exists)
	mockClient.AssertExpectations(t)
}

func TestMockObjectExistsNotFound(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("ObjectExists", mock.Anything, "my-bucket", "nonexistent").
		Return(false, nil)

	exists, err := mockClient.ObjectExists(context.Background(), "my-bucket", "nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
	mockClient.AssertExpectations(t)
}

func TestMockClose(t *testing.T) {
	mockClient := &MockGCPClient{}

	mockClient.On("Close").Return(nil)

	err := mockClient.Close()
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("strPtr", func(t *testing.T) {
		s := strPtr("test")
		assert.NotNil(t, s)
		assert.Equal(t, "test", *s)
	})

	t.Run("boolPtr", func(t *testing.T) {
		b := boolPtr(true)
		assert.NotNil(t, b)
		assert.True(t, *b)

		b2 := boolPtr(false)
		assert.NotNil(t, b2)
		assert.False(t, *b2)
	})

	t.Run("int64Ptr", func(t *testing.T) {
		i := int64Ptr(42)
		assert.NotNil(t, i)
		assert.Equal(t, int64(42), *i)
	})
}
