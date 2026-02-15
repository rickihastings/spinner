package docker

import (
	"time"
)

// LogEvent represents a single log entry from a container.
type LogEvent struct {
	// Timestamp of the log entry
	Timestamp time.Time

	// Stream indicates the source stream ("stdout" or "stderr")
	Stream string

	// Message is the log content
	Message string

	// Error is set if there was an error reading the log
	Error error
}

// buildEvent represents a build progress event during image building.
type buildEvent struct {
	// Stream contains build output text
	Stream string

	// Status contains status messages (e.g., "Pulling from library/ubuntu")
	Status string

	// Progress contains progress information (e.g., "[=====>    ] 50%")
	Progress string

	// Error is set if there was a build error
	Error string

	// ErrorDetail provides detailed error information
	ErrorDetail *buildErrorDetail

	// Aux contains auxiliary data (e.g., image ID after successful build)
	Aux *buildAux
}

// buildErrorDetail provides detailed error information for build failures.
type buildErrorDetail struct {
	Code    int
	Message string
}

// buildAux contains auxiliary build data.
type buildAux struct {
	ID string
}

// isError returns true if this build event represents an error.
func (e *buildEvent) isError() bool {
	return e.Error != "" || e.ErrorDetail != nil
}

// LogStreamOptions configures log streaming behavior.
type LogStreamOptions struct {
	// Follow keeps the stream open and follows new log output
	Follow bool

	// Timestamps prepends timestamps to each log line
	Timestamps bool

	// Tail limits output to the last N lines ("all" for everything)
	Tail string

	// Since shows logs since this timestamp (RFC3339 format or relative duration)
	Since string

	// Until shows logs until this timestamp
	Until string

	// Stdout includes stdout output
	Stdout bool

	// Stderr includes stderr output
	Stderr bool
}

// defaultLogStreamOptions returns sensible defaults for log streaming.
func defaultLogStreamOptions() LogStreamOptions {
	return LogStreamOptions{
		Follow:     false,
		Timestamps: false,
		Tail:       "all",
		Stdout:     true,
		Stderr:     true,
	}
}

// ContainerListEntry represents a container returned by ListContainers.
// This replaces the Docker SDK's container.Summary type.
type ContainerListEntry struct {
	ID     string
	Names  []string
	Image  string
	State  string
	Labels map[string]string
}
