package docker

import (
	"testing"

	"github.com/rickihastings/spinner/internal/provider"
	"github.com/stretchr/testify/assert"
)

func TestParseCPUPercent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected float64
	}{
		{
			name:     "normal percentage",
			input:    "1.23%",
			expected: 1.23,
		},
		{
			name:     "zero percentage",
			input:    "0.00%",
			expected: 0.0,
		},
		{
			name:     "high percentage",
			input:    "156.78%",
			expected: 156.78,
		},
		{
			name:     "with whitespace",
			input:    "  42.5%  ",
			expected: 42.5,
		},
		{
			name:     "invalid value",
			input:    "abc%",
			expected: 0.0,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCPUPercent(tt.input)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestParseMemUsage(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedUsed  uint64
		expectedLimit uint64
	}{
		{
			name:          "MiB values",
			input:         "100MiB / 1024MiB",
			expectedUsed:  100 * 1024 * 1024,
			expectedLimit: 1024 * 1024 * 1024,
		},
		{
			name:          "GiB limit",
			input:         "256MiB / 1.94GiB",
			expectedUsed:  256 * 1024 * 1024,
			expectedLimit: parseMemoryValue("1.94GiB"),
		},
		{
			name:          "KiB values",
			input:         "512KiB / 1024KiB",
			expectedUsed:  512 * 1024,
			expectedLimit: 1024 * 1024,
		},
		{
			name:          "invalid format",
			input:         "not a memory string",
			expectedUsed:  0,
			expectedLimit: 0,
		},
		{
			name:          "empty string",
			input:         "",
			expectedUsed:  0,
			expectedLimit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, limit := parseMemUsage(tt.input)
			assert.Equal(t, tt.expectedUsed, used)
			assert.Equal(t, tt.expectedLimit, limit)
		})
	}
}

func TestParseMemoryValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
	}{
		{
			name:     "GiB value",
			input:    "2GiB",
			expected: 2 * 1024 * 1024 * 1024,
		},
		{
			name:     "MiB value",
			input:    "512MiB",
			expected: 512 * 1024 * 1024,
		},
		{
			name:     "KiB value",
			input:    "1024KiB",
			expected: 1024 * 1024,
		},
		{
			name:     "B value",
			input:    "4096B",
			expected: 4096,
		},
		{
			name:     "fractional GiB",
			input:    "1.5GiB",
			expected: uint64(1.5 * 1024 * 1024 * 1024),
		},
		{
			name:     "fractional MiB",
			input:    "100.5MiB",
			expected: uint64(100.5 * 1024 * 1024),
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "invalid value",
			input:    "notamemory",
			expected: 0,
		},
		{
			name:     "with whitespace",
			input:    " 256MiB ",
			expected: 256 * 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMemoryValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapDockerStateToMetrics(t *testing.T) {
	tests := []struct {
		name     string
		state    *dockerInspectState
		expected provider.ContainerState
	}{
		{
			name:     "nil state",
			state:    nil,
			expected: provider.StateUnknown,
		},
		{
			name:     "running",
			state:    &dockerInspectState{Running: true},
			expected: provider.StateRunning,
		},
		{
			name:     "stopped with zero exit code",
			state:    &dockerInspectState{Running: false, ExitCode: 0},
			expected: provider.StateStopped,
		},
		{
			name:     "exited with non-zero exit code",
			state:    &dockerInspectState{Running: false, ExitCode: 1},
			expected: provider.StateExited,
		},
		{
			name:     "dead container",
			state:    &dockerInspectState{Running: false, Dead: true},
			expected: provider.StateExited,
		},
		{
			name:     "OOM killed container",
			state:    &dockerInspectState{Running: false, OOMKilled: true},
			expected: provider.StateExited,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapDockerStateToMetrics(tt.state)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDockerStatsJSON_Parsing(t *testing.T) {
	// Verify the struct correctly maps docker stats JSON output fields
	stats := dockerStatsJSON{
		CPUPerc:  "5.23%",
		MemUsage: "100MiB / 1.94GiB",
	}

	cpuPercent := parseCPUPercent(stats.CPUPerc)
	assert.InDelta(t, 5.23, cpuPercent, 0.001)

	memUsed, memLimit := parseMemUsage(stats.MemUsage)
	assert.Equal(t, uint64(100*1024*1024), memUsed)
	assert.Equal(t, parseMemoryValue("1.94GiB"), memLimit)
}

func TestDockerInspectState_Fields(t *testing.T) {
	state := dockerInspectState{
		Running:   true,
		Dead:      false,
		OOMKilled: false,
		ExitCode:  0,
	}

	assert.True(t, state.Running)
	assert.False(t, state.Dead)
	assert.False(t, state.OOMKilled)
	assert.Equal(t, 0, state.ExitCode)
}
