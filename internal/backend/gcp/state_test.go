package gcp

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateObjectPath(t *testing.T) {
	assert.Equal(t, "my-instance/state.json", stateObjectPath("my-instance"))
	assert.Equal(t, "spinner-env-repo-main/state.json", stateObjectPath("spinner-env-repo-main"))
}

func TestReadState_Success(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	stateJSON := []byte(`{"status":"running","iteration":3}`)

	mockClient.On("ObjectExists", ctx, bucket, "test-instance/state.json").Return(true, nil)
	mockClient.On("ReadObject", ctx, bucket, "test-instance/state.json").Return(stateJSON, nil)

	data, err := readState(ctx, mockClient, bucket, instance)
	assert.NoError(t, err)
	assert.Equal(t, stateJSON, data)
	mockClient.AssertExpectations(t)
}

func TestReadState_NotFound(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	mockClient.On("ObjectExists", ctx, bucket, "test-instance/state.json").Return(false, nil)

	data, err := readState(ctx, mockClient, bucket, instance)
	assert.NoError(t, err)
	assert.Nil(t, data)
	mockClient.AssertExpectations(t)
}

func TestReadState_ExistsCheckError(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	mockClient.On("ObjectExists", ctx, bucket, "test-instance/state.json").
		Return(false, fmt.Errorf("network error"))

	data, err := readState(ctx, mockClient, bucket, instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check state existence")
	assert.Nil(t, data)
	mockClient.AssertExpectations(t)
}

func TestReadState_ReadError(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	mockClient.On("ObjectExists", ctx, bucket, "test-instance/state.json").Return(true, nil)
	mockClient.On("ReadObject", ctx, bucket, "test-instance/state.json").
		Return(nil, fmt.Errorf("permission denied"))

	data, err := readState(ctx, mockClient, bucket, instance)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read state")
	assert.Contains(t, err.Error(), "gs://test-bucket/test-instance/state.json")
	assert.Nil(t, data)
	mockClient.AssertExpectations(t)
}

func TestWriteState_Success(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	stateJSON := []byte(`{"status":"completed","iteration":5}`)

	mockClient.On("WriteObject", ctx, bucket, "test-instance/state.json", stateJSON).Return(nil)

	err := writeState(ctx, mockClient, bucket, instance, stateJSON)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestWriteState_Error(t *testing.T) {
	mockClient := &MockGCPClient{}
	ctx := context.Background()
	bucket := "test-bucket"
	instance := "test-instance"

	stateJSON := []byte(`{"status":"running","iteration":1}`)

	mockClient.On("WriteObject", ctx, bucket, "test-instance/state.json", stateJSON).
		Return(fmt.Errorf("bucket not found"))

	err := writeState(ctx, mockClient, bucket, instance, stateJSON)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write state")
	assert.Contains(t, err.Error(), "gs://test-bucket/test-instance/state.json")
	mockClient.AssertExpectations(t)
}
