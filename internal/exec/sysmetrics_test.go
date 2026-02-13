package exec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProcStat_Valid(t *testing.T) {
	content := "cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0\ncpu0  1393280 32966 572056 13343292 6130 0 17875 0 0 0\n"

	idle, total, err := parseProcStat(content)
	assert.NoError(t, err)
	// idle = 46828483 (4th value after "cpu")
	assert.Equal(t, uint64(46828483), idle)
	// total = sum of all values = 10132153+290696+3084719+46828483+16683+0+25195+0+0+0 = 60377929
	assert.Equal(t, uint64(60377929), total)
}

func TestParseProcStat_MinimalFields(t *testing.T) {
	content := "cpu  100 200 300 400\n"

	idle, total, err := parseProcStat(content)
	assert.NoError(t, err)
	assert.Equal(t, uint64(400), idle)
	assert.Equal(t, uint64(1000), total)
}

func TestParseProcStat_Empty(t *testing.T) {
	_, _, err := parseProcStat("")
	assert.Error(t, err)
}

func TestParseProcStat_NoCPULine(t *testing.T) {
	content := "intr 123456789\n"
	_, _, err := parseProcStat(content)
	assert.Error(t, err)
}

func TestParseProcStat_TooFewFields(t *testing.T) {
	content := "cpu  100 200 300\n"
	_, _, err := parseProcStat(content)
	assert.Error(t, err)
}

func TestParseProcMeminfo_Valid(t *testing.T) {
	content := `MemTotal:       16384000 kB
MemFree:         2000000 kB
MemAvailable:    8192000 kB
Buffers:          500000 kB
`

	percent := parseProcMeminfo(content)
	// (16384000 - 8192000) / 16384000 * 100 = 50.0
	assert.InDelta(t, 50.0, percent, 0.01)
}

func TestParseProcMeminfo_HighUsage(t *testing.T) {
	content := `MemTotal:       16000000 kB
MemFree:          100000 kB
MemAvailable:    2400000 kB
`

	percent := parseProcMeminfo(content)
	// (16000000 - 2400000) / 16000000 * 100 = 85.0
	assert.InDelta(t, 85.0, percent, 0.01)
}

func TestParseProcMeminfo_MissingMemTotal(t *testing.T) {
	content := `MemFree:         2000000 kB
MemAvailable:    8192000 kB
`

	percent := parseProcMeminfo(content)
	assert.Equal(t, 0.0, percent)
}

func TestParseProcMeminfo_MissingMemAvailable(t *testing.T) {
	content := `MemTotal:       16384000 kB
MemFree:         2000000 kB
`

	percent := parseProcMeminfo(content)
	assert.Equal(t, 0.0, percent)
}

func TestParseProcMeminfo_Empty(t *testing.T) {
	percent := parseProcMeminfo("")
	assert.Equal(t, 0.0, percent)
}

func TestParseProcMeminfo_ZeroTotal(t *testing.T) {
	content := `MemTotal:       0 kB
MemAvailable:   0 kB
`

	percent := parseProcMeminfo(content)
	assert.Equal(t, 0.0, percent)
}

func TestParseMeminfoValue(t *testing.T) {
	tests := []struct {
		line     string
		expected uint64
	}{
		{"MemTotal:       16384000 kB", 16384000},
		{"MemAvailable:   8192000 kB", 8192000},
		{"MemFree:", 0},
		{"", 0},
	}

	for _, tt := range tests {
		val := parseMeminfoValue(tt.line)
		assert.Equal(t, tt.expected, val, "line: %s", tt.line)
	}
}

func TestCollectSystemMetrics_NonLinux(t *testing.T) {
	// Override paths to non-existent files to simulate non-Linux
	origStat := procStatPath
	origMeminfo := procMeminfoPath

	defer func() {
		procStatPath = origStat
		procMeminfoPath = origMeminfo
	}()

	procStatPath = "/nonexistent/proc/stat"
	procMeminfoPath = "/nonexistent/proc/meminfo"

	cpu, mem := collectSystemMetrics()
	assert.Equal(t, 0.0, cpu)
	assert.Equal(t, 0.0, mem)
}
