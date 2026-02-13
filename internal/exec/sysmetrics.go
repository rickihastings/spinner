package exec

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// procStatPath and procMeminfoPath are the default paths to /proc files.
// They are variables so tests can override them.
var (
	procStatPath    = "/proc/stat"
	procMeminfoPath = "/proc/meminfo"
)

// collectSystemMetrics reads CPU and memory usage from /proc.
// Returns (cpuPercent, memoryPercent). Gracefully returns 0 on any error.
func collectSystemMetrics() (float64, float64) {
	cpu := collectCPUPercent()
	mem := collectMemoryPercent()

	return cpu, mem
}

// collectCPUPercent reads /proc/stat twice with a brief delay to compute
// CPU usage as a percentage (0-100). Returns 0 on any error.
func collectCPUPercent() float64 {
	idle1, total1, err := readCPUSample()
	if err != nil {
		return 0
	}

	time.Sleep(100 * time.Millisecond)

	idle2, total2, err := readCPUSample()
	if err != nil {
		return 0
	}

	totalDelta := total2 - total1
	idleDelta := idle2 - idle1

	if totalDelta == 0 {
		return 0
	}

	return float64(totalDelta-idleDelta) / float64(totalDelta) * 100.0
}

// readCPUSample reads the aggregate CPU line from /proc/stat and returns
// (idle, total) jiffies.
func readCPUSample() (uint64, uint64, error) {
	data, err := os.ReadFile(procStatPath)
	if err != nil {
		return 0, 0, err
	}

	return parseProcStat(string(data))
}

// parseProcStat extracts (idle, total) jiffies from /proc/stat content.
// The first line is "cpu  user nice system idle iowait irq softirq steal ...".
func parseProcStat(content string) (uint64, uint64, error) {
	lines := strings.SplitN(content, "\n", 2)
	if len(lines) == 0 {
		return 0, 0, os.ErrNotExist
	}

	line := lines[0]
	if !strings.HasPrefix(line, "cpu ") {
		return 0, 0, os.ErrNotExist
	}

	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, 0, os.ErrNotExist
	}

	// fields: cpu user nice system idle [iowait irq softirq steal ...]
	var total uint64

	var idle uint64

	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, err
		}

		total += val

		if i == 4 { // idle is the 4th value (index 4 in fields)
			idle = val
		}
	}

	return idle, total, nil
}

// collectMemoryPercent reads /proc/meminfo and computes memory usage
// as a percentage: (MemTotal - MemAvailable) / MemTotal * 100.
// Returns 0 on any error.
func collectMemoryPercent() float64 {
	data, err := os.ReadFile(procMeminfoPath)
	if err != nil {
		return 0
	}

	return parseProcMeminfo(string(data))
}

// parseProcMeminfo extracts memory usage percentage from /proc/meminfo content.
func parseProcMeminfo(content string) float64 {
	var memTotal, memAvailable uint64

	var foundTotal, foundAvailable bool

	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMeminfoValue(line)
			foundTotal = true
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMeminfoValue(line)
			foundAvailable = true
		}

		if foundTotal && foundAvailable {
			break
		}
	}

	if !foundTotal || !foundAvailable || memTotal == 0 {
		return 0
	}

	return float64(memTotal-memAvailable) / float64(memTotal) * 100.0
}

// parseMeminfoValue extracts the numeric kB value from a /proc/meminfo line
// like "MemTotal:       16384000 kB".
func parseMeminfoValue(line string) uint64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}

	val, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}

	return val
}
