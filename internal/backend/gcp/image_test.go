package gcp

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// withFastPolling overrides bakePollInterval and bakeTimeout for fast tests
// and restores them when done.
func withFastPolling(t *testing.T) {
	t.Helper()

	origPoll := bakePollInterval
	origTimeout := bakeTimeout
	bakePollInterval = 10 * time.Millisecond
	bakeTimeout = 2 * time.Second

	t.Cleanup(func() {
		bakePollInterval = origPoll
		bakeTimeout = origTimeout
	})
}

func TestBakeImageSuccess(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	config := bakeConfig{
		ImageName:     "test-image",
		Project:       "test-project",
		Zone:          "us-central1-a",
		MachineType:   "e2-standard-2",
		DiskSizeGB:    30,
		StartupScript: "#!/bin/bash\necho hello",
	}

	// Expect CreateInstance
	mockClient.On("CreateInstance", mock.Anything, instanceConfig{
		Name:         "spinner-bake-test-image",
		Project:      "test-project",
		Zone:         "us-central1-a",
		MachineType:  "e2-standard-2",
		ImageProject: bakeBaseImageProject,
		ImageName:    bakeBaseImage,
		DiskSizeGB:   30,
		Network:      "default",
		ExternalIP:   true,
		Metadata: map[string]string{
			"startup-script":  "#!/bin/bash\necho hello",
			"LOCAL_BUILD":     os.Getenv("LOCAL_BUILD"),
			"STATE_BUCKET":    "",
			"SPINNER_VERSION": "",
		},
		Labels: map[string]string{
			"spinner-managed": "true",
			"spinner-purpose": "bake",
			"spinner-image":   "test-image",
		},
		Scopes: []string{
			"https://www.googleapis.com/auth/devstorage.read_write",
			"https://www.googleapis.com/auth/logging.write",
		},
	}).Return(nil)

	// GetInstance returns RUNNING first, then TERMINATED
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(&GCPInstance{Name: "spinner-bake-test-image", Status: "RUNNING"}, nil).Once()
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(&GCPInstance{Name: "spinner-bake-test-image", Status: "TERMINATED"}, nil).Once()

	// Expect CreateImage
	mockClient.On("CreateImage", mock.Anything, "test-project", imageConfig{
		Name:        "test-image",
		SourceDisk:  "projects/test-project/zones/us-central1-a/disks/spinner-bake-test-image",
		Description: "Spinner baked image: test-image",
		Labels: map[string]string{
			"spinner-managed": "true",
			"spinner-image":   "test-image",
		},
	}).Return(nil)

	// Expect cleanup DeleteInstance
	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(nil)

	err := bakeImage(context.Background(), mockClient, config)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestBakeImageCreateInstanceError(t *testing.T) {
	mockClient := &MockGCPClient{}

	config := bakeConfig{
		ImageName:     "test-image",
		Project:       "test-project",
		Zone:          "us-central1-a",
		MachineType:   "e2-standard-2",
		DiskSizeGB:    30,
		StartupScript: "#!/bin/bash\necho hello",
	}

	mockClient.On("CreateInstance", mock.Anything, mock.Anything).
		Return(fmt.Errorf("quota exceeded"))

	err := bakeImage(context.Background(), mockClient, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create bake VM")
	assert.Contains(t, err.Error(), "quota exceeded")
	mockClient.AssertExpectations(t)
}

func TestBakeImageCreateImageError(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	config := bakeConfig{
		ImageName:     "test-image",
		Project:       "test-project",
		Zone:          "us-central1-a",
		MachineType:   "e2-standard-2",
		DiskSizeGB:    30,
		StartupScript: "#!/bin/bash\necho hello",
	}

	mockClient.On("CreateInstance", mock.Anything, mock.Anything).Return(nil)

	// VM terminates immediately
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(&GCPInstance{Name: "spinner-bake-test-image", Status: "TERMINATED"}, nil)

	// CreateImage fails
	mockClient.On("CreateImage", mock.Anything, "test-project", mock.Anything).
		Return(fmt.Errorf("image creation failed"))

	// Cleanup should still happen
	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(nil)

	err := bakeImage(context.Background(), mockClient, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create image")

	// Verify cleanup was called even though CreateImage failed
	mockClient.AssertCalled(t, "DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image")
}

func TestBakeImageWaitError(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	config := bakeConfig{
		ImageName:     "test-image",
		Project:       "test-project",
		Zone:          "us-central1-a",
		MachineType:   "e2-standard-2",
		DiskSizeGB:    30,
		StartupScript: "#!/bin/bash\necho hello",
	}

	mockClient.On("CreateInstance", mock.Anything, mock.Anything).Return(nil)

	// VM enters unexpected state
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(&GCPInstance{Name: "spinner-bake-test-image", Status: "SUSPENDED"}, nil)

	// Cleanup should still happen
	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(nil)

	err := bakeImage(context.Background(), mockClient, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bake failed")
	mockClient.AssertCalled(t, "DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image")
}

func TestBakeImageCleanupErrorDoesNotOverride(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	config := bakeConfig{
		ImageName:     "test-image",
		Project:       "test-project",
		Zone:          "us-central1-a",
		MachineType:   "e2-standard-2",
		DiskSizeGB:    30,
		StartupScript: "#!/bin/bash\necho hello",
	}

	mockClient.On("CreateInstance", mock.Anything, mock.Anything).Return(nil)

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(&GCPInstance{Name: "spinner-bake-test-image", Status: "TERMINATED"}, nil)

	mockClient.On("CreateImage", mock.Anything, "test-project", mock.Anything).Return(nil)

	// Cleanup fails — should not affect the overall result
	mockClient.On("DeleteInstance", mock.Anything, "test-project", "us-central1-a", "spinner-bake-test-image").
		Return(fmt.Errorf("delete failed"))

	err := bakeImage(context.Background(), mockClient, config)
	assert.NoError(t, err)
}

func TestWaitForVMTerminatedSuccess(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "TERMINATED"}, nil)

	err := waitForVMTerminated(context.Background(), mockClient, "test-project", "us-central1-a", "test-vm")
	assert.NoError(t, err)
}

func TestWaitForVMTerminatedTransitionsToTerminated(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "RUNNING"}, nil).Once()
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "STOPPING"}, nil).Once()
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "TERMINATED"}, nil).Once()

	err := waitForVMTerminated(context.Background(), mockClient, "test-project", "us-central1-a", "test-vm")
	assert.NoError(t, err)
}

func TestWaitForVMTerminatedUnexpectedState(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "SUSPENDED"}, nil)

	err := waitForVMTerminated(context.Background(), mockClient, "test-project", "us-central1-a", "test-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected state")
	assert.Contains(t, err.Error(), "SUSPENDED")
}

func TestWaitForVMTerminatedGetInstanceError(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(nil, fmt.Errorf("API error"))

	err := waitForVMTerminated(context.Background(), mockClient, "test-project", "us-central1-a", "test-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check bake VM status")
}

func TestWaitForVMTerminatedContextCancelled(t *testing.T) {
	withFastPolling(t)

	mockClient := &MockGCPClient{}

	// Return RUNNING forever — the context cancellation should stop the loop
	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "RUNNING"}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := waitForVMTerminated(ctx, mockClient, "test-project", "us-central1-a", "test-vm")
	assert.Error(t, err)
}

func TestWaitForVMTerminatedTimeout(t *testing.T) {
	origPoll := bakePollInterval
	origTimeout := bakeTimeout
	bakePollInterval = 10 * time.Millisecond
	bakeTimeout = 50 * time.Millisecond

	t.Cleanup(func() {
		bakePollInterval = origPoll
		bakeTimeout = origTimeout
	})

	mockClient := &MockGCPClient{}

	mockClient.On("GetInstance", mock.Anything, "test-project", "us-central1-a", "test-vm").
		Return(&GCPInstance{Name: "test-vm", Status: "RUNNING"}, nil)

	err := waitForVMTerminated(context.Background(), mockClient, "test-project", "us-central1-a", "test-vm")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

func TestBakeConfigFields(t *testing.T) {
	config := bakeConfig{
		ImageName:     "my-image",
		Project:       "my-project",
		Zone:          "us-east1-b",
		MachineType:   "n2-standard-4",
		DiskSizeGB:    50,
		StartupScript: "#!/bin/bash\necho bake",
	}

	assert.Equal(t, "my-image", config.ImageName)
	assert.Equal(t, "my-project", config.Project)
	assert.Equal(t, "us-east1-b", config.Zone)
	assert.Equal(t, "n2-standard-4", config.MachineType)
	assert.Equal(t, int64(50), config.DiskSizeGB)
	assert.Equal(t, "#!/bin/bash\necho bake", config.StartupScript)
}

func TestBakeVMNameFormat(t *testing.T) {
	assert.Equal(t, "spinner-bake-", bakeVMPrefix)
	assert.Equal(t, "spinner-bake-my-image", bakeVMPrefix+"my-image")
}
