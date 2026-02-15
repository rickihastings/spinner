package cmd

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestListCommand_MultiBackendListing(t *testing.T) {
	now := time.Now()
	started := now.Add(-2 * time.Hour)
	updated := now.Add(-5 * time.Minute)

	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:          "spinner-default-repo1",
			Status:        provider.InstanceStatusRunning,
			Backend:       "docker",
			Iteration:     5,
			MaxIterations: 100,
			AgentStatus:   "running",
			StartedAt:     &started,
			LastUpdated:   &updated,
		},
	}, nil)

	gcpMock := new(provider.MockProvider)
	gcpMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:          "spinner-prod-refactor",
			Status:        provider.InstanceStatusRunning,
			Backend:       "gcp",
			Iteration:     12,
			MaxIterations: 100,
			AgentStatus:   "rate_limited",
			StartedAt:     &started,
			LastUpdated:   &updated,
		},
	}, nil)

	f := provider.NewFactory()
	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return dockerMock, nil
	})
	f.Register(provider.BackendGCP, func() (provider.Provider, error) {
		return gcpMock, nil
	})

	// Set GCP config so it's not skipped
	t.Setenv("SPINNER_PROJECT", "test-proj")
	t.Setenv("SPINNER_ZONE", "us-central1-a")

	cmd := NewListCommand(f)
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "BACKEND")
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "spinner-default-repo1")
	assert.Contains(t, output, "spinner-prod-refactor")
	assert.Contains(t, output, "docker")
	assert.Contains(t, output, "gcp")
	assert.Contains(t, output, "5/100")
	assert.Contains(t, output, "12/100")
	assert.Contains(t, output, "running")
	assert.Contains(t, output, "rate_limited")
	dockerMock.AssertCalled(t, "List", mock.Anything)
	gcpMock.AssertCalled(t, "List", mock.Anything)
}

func TestListCommand_BackendUnavailableWarning(t *testing.T) {
	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{}, nil)

	f := provider.NewFactory()
	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return nil, fmt.Errorf("docker not running")
	})

	cmd := NewListCommand(f)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, stderr.String(), "docker: docker not running")
	assert.Contains(t, stdout.String(), "No instances found.")
}

func TestListCommand_NoInstances(t *testing.T) {
	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{}, nil)

	cmd := NewListCommand(testFactory(dockerMock))
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, b.String(), "No instances found.")
}

func TestListCommand_StaleWarning(t *testing.T) {
	now := time.Now()
	started := now.Add(-5 * time.Hour)
	staleUpdate := now.Add(-3 * time.Hour) // >2h ago = stale

	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:          "spinner-stale-instance",
			Status:        provider.InstanceStatusRunning,
			Backend:       "docker",
			Iteration:     88,
			MaxIterations: 100,
			AgentStatus:   "running",
			StartedAt:     &started,
			LastUpdated:   &staleUpdate,
		},
	}, nil)

	cmd := NewListCommand(testFactory(dockerMock))
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "spinner-stale-instance")
	assert.Contains(t, output, "⚠ stale")
}

func TestListCommand_StoppedInstanceNoStaleWarning(t *testing.T) {
	now := time.Now()
	started := now.Add(-5 * time.Hour)
	oldUpdate := now.Add(-3 * time.Hour) // >2h ago but instance is stopped

	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:          "spinner-stopped-old",
			Status:        provider.InstanceStatusStopped,
			Backend:       "docker",
			Iteration:     42,
			MaxIterations: 50,
			AgentStatus:   "completed",
			StartedAt:     &started,
			LastUpdated:   &oldUpdate,
		},
	}, nil)

	cmd := NewListCommand(testFactory(dockerMock))
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "spinner-stopped-old")
	assert.NotContains(t, output, "⚠ stale")
}

func TestListCommand_SortOrder(t *testing.T) {
	now := time.Now()
	started := now.Add(-1 * time.Hour)
	updated := now.Add(-5 * time.Minute)

	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:    "b-stopped",
			Status:  provider.InstanceStatusStopped,
			Backend: "docker",
		},
		{
			Name:          "a-running",
			Status:        provider.InstanceStatusRunning,
			Backend:       "docker",
			Iteration:     1,
			MaxIterations: 10,
			AgentStatus:   "running",
			StartedAt:     &started,
			LastUpdated:   &updated,
		},
	}, nil)

	cmd := NewListCommand(testFactory(dockerMock))
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := b.String()
	// Running instance should appear before stopped instance
	runningIdx := bytes.Index([]byte(output), []byte("a-running"))
	stoppedIdx := bytes.Index([]byte(output), []byte("b-stopped"))
	assert.Greater(t, stoppedIdx, runningIdx, "running instances should be listed before stopped")
}

func TestListCommand_BackendListError(t *testing.T) {
	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return(nil, fmt.Errorf("connection refused"))

	cmd := NewListCommand(testFactory(dockerMock))
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, stderr.String(), "docker: connection refused")
	assert.Contains(t, stdout.String(), "No instances found.")
}

func TestListCommand_GCPSkippedWhenNotConfigured(t *testing.T) {
	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{}, nil)

	gcpMock := new(provider.MockProvider)
	// GCP mock should NOT be called

	f := provider.NewFactory()
	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return dockerMock, nil
	})
	f.Register(provider.BackendGCP, func() (provider.Provider, error) {
		return gcpMock, nil
	})

	// Don't set any GCP env vars
	t.Setenv("SPINNER_PROJECT", "")
	t.Setenv("SPINNER_ZONE", "")

	cmd := NewListCommand(f)
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	gcpMock.AssertNotCalled(t, "List", mock.Anything)
}

func TestListCommand_MissingStateFields(t *testing.T) {
	dockerMock := new(provider.MockProvider)
	dockerMock.On("List", mock.Anything).Return([]provider.InstanceInfo{
		{
			Name:    "spinner-no-state",
			Status:  provider.InstanceStatusRunning,
			Backend: "docker",
			// No iteration, agent status, or timestamps
		},
	}, nil)

	cmd := NewListCommand(testFactory(dockerMock))
	b := new(bytes.Buffer)
	cmd.SetOut(b)
	cmd.SetErr(b)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	assert.NoError(t, err)
	output := b.String()
	assert.Contains(t, output, "spinner-no-state")
	// Dash placeholders for missing data
	assert.Contains(t, output, "-")
}

func TestFormatDuration(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		t        *time.Time
		expected string
	}{
		{"nil", nil, "-"},
		{"seconds", timePtr(now.Add(-30 * time.Second)), "30s ago"},
		{"minutes", timePtr(now.Add(-45 * time.Minute)), "45m ago"},
		{"hours", timePtr(now.Add(-3 * time.Hour)), "3h ago"},
		{"days", timePtr(now.Add(-48 * time.Hour)), "2d ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.t, now)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatIter(t *testing.T) {
	tests := []struct {
		current, max int
		expected     string
	}{
		{0, 0, "-"},
		{5, 100, "5/100"},
		{10, 0, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatIter(tt.current, tt.max)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
