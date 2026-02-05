package provider

import (
	"context"
	"io"
)

// InstanceStatus represents the lifecycle state of an instance.
type InstanceStatus string

const (
	InstanceStatusNone    InstanceStatus = "none"
	InstanceStatusRunning InstanceStatus = "running"
	InstanceStatusStopped InstanceStatus = "stopped"
)

// ContainerState represents the execution state of a container
type ContainerState string

const (
	// StateRunning indicates the container is currently running
	StateRunning ContainerState = "running"
	// StateStopped indicates the container has been stopped
	StateStopped ContainerState = "stopped"
	// StateExited indicates the container has exited
	StateExited ContainerState = "exited"
	// StateUnknown indicates the container state cannot be determined
	StateUnknown ContainerState = "unknown"
)

// ContainerMetrics represents resource usage metrics for a container
type ContainerMetrics struct {
	// State is the current execution state of the container
	State ContainerState

	// CPUPercent is the CPU usage as a percentage (0-100+)
	CPUPercent float64

	// MemoryUsed is the current memory usage in bytes
	MemoryUsed uint64

	// MemoryLimit is the memory limit in bytes (0 if unlimited)
	MemoryLimit uint64

	// MemoryPercent is the memory usage as a percentage (0-100)
	MemoryPercent float64

	// Error contains any error that occurred during metrics collection
	Error error
}

// SetupConfig holds configuration for provisioning a named environment.
// Name is universal. Options carries backend-specific keys: e.g. "base-image"
// and "dockerfile" for Docker, "machine-type" and "zone" for GCP.
type SetupConfig struct {
	Name    string
	Options map[string]string
}

// CreateConfig holds configuration for creating an instance from a provisioned
// environment. Repo, Prompt, Branch, and MaxIterations are universal —
// every backend needs to know what code to run and how. Options carries
// backend-specific keys: e.g. "image" for Docker.
type CreateConfig struct {
	Repo          string
	Prompt        string
	Branch        string
	MaxIterations string
	Options       map[string]string
}

// Instance represents an execution environment instance.
type Instance struct {
	Name   string
	Status InstanceStatus
}

// Provider is the backend-agnostic interface for managing isolated execution
// environments. Each backend (Docker, VMs, cloud instances) implements this
// interface independently. The rest of the system depends only on Provider.
type Provider interface {
	// Setup provisions a named environment (e.g. builds a Docker image).
	Setup(ctx context.Context, config SetupConfig) error

	// Create creates a new instance from a provisioned environment.
	// Returns an error if the instance already exists or the environment
	// has not been set up.
	Create(ctx context.Context, config CreateConfig) (*Instance, error)

	// Start starts a stopped instance.
	Start(ctx context.Context, name string) (*Instance, error)

	// Restart restarts a running instance (stop then start).
	Restart(ctx context.Context, name string) (*Instance, error)

	// Stop stops a running instance without removing it.
	Stop(ctx context.Context, name string) error

	// Remove destroys an instance and reclaims resources.
	Remove(ctx context.Context, name string) error

	// Logs returns the output logs from an instance.
	Logs(ctx context.Context, name string) (io.ReadCloser, error)

	// Status returns the current lifecycle status of an instance.
	Status(ctx context.Context, name string) (InstanceStatus, error)

	// InstanceName returns the deterministic instance name for the given
	// configuration. This lets callers check Status before deciding whether
	// to Create.
	InstanceName(config CreateConfig) string

	// WatchLogs streams raw log lines from an instance.
	// First emits the last `tailLines` lines (for initial display),
	// then continues streaming new lines in real-time.
	// Sends raw log lines (strings) to the channel.
	// Closes the channel when the stream ends or context is cancelled.
	WatchLogs(ctx context.Context, name string, tailLines int, ch chan<- string) error

	// WatchMetrics streams real-time resource metrics for an instance.
	// Polls metrics at regular intervals and sends ContainerMetrics to the channel.
	// Closes the channel when the stream ends or context is cancelled.
	// Returns an error if the instance doesn't exist or metrics cannot be collected.
	WatchMetrics(ctx context.Context, name string, ch chan<- ContainerMetrics) error
}
