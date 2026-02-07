package exec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// status represents the current state of the execution loop.
type status string

const (
	statusRunning     status = "running"
	statusCompleted   status = "completed"
	statusRateLimited status = "rate_limited"
	statusError       status = "error"
	statusAuthError   status = "auth_error"
)

// State represents the persistent state of the execution loop.
type State struct {
	Branch       string    `json:"branch"`
	Iteration    int       `json:"iteration"`
	Status       status    `json:"status"`
	LastUpdated  time.Time `json:"last_updated"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// LoadState loads the state from a JSON file.
// If the file does not exist, it returns a new state with default values.
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default state if file doesn't exist
			return &State{
				Status:      statusRunning,
				Iteration:   0,
				StartedAt:   time.Now(),
				LastUpdated: time.Now(),
			}, nil
		}

		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Validate required fields
	if state.Status == "" {
		return nil, fmt.Errorf("invalid state: status is empty")
	}

	return &state, nil
}

// saveState saves the state to a JSON file using atomic write.
func saveState(path string, state *State) error {
	// Update timestamp
	state.LastUpdated = time.Now()

	// Marshal state to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Write to temp file first for atomic operation
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, path); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temp state file: %w", err)
	}

	return nil
}
