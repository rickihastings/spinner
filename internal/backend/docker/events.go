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

// BuildEvent represents a build progress event during image building.
type BuildEvent struct {
	// Stream contains build output text
	Stream string

	// Status contains status messages (e.g., "Pulling from library/ubuntu")
	Status string

	// Progress contains progress information (e.g., "[=====>    ] 50%")
	Progress string

	// Error is set if there was a build error
	Error string

	// ErrorDetail provides detailed error information
	ErrorDetail *BuildErrorDetail

	// Aux contains auxiliary data (e.g., image ID after successful build)
	Aux *BuildAux
}

// BuildErrorDetail provides detailed error information for build failures.
type BuildErrorDetail struct {
	Code    int
	Message string
}

// BuildAux contains auxiliary build data.
type BuildAux struct {
	ID string
}

// IsError returns true if this build event represents an error.
func (e *BuildEvent) IsError() bool {
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

// DefaultLogStreamOptions returns sensible defaults for log streaming.
func DefaultLogStreamOptions() LogStreamOptions {
	return LogStreamOptions{
		Follow:     false,
		Timestamps: false,
		Tail:       "all",
		Stdout:     true,
		Stderr:     true,
	}
}
