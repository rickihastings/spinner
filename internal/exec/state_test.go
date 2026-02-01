package exec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadState_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state.Status != StatusRunning {
		t.Errorf("expected status %s, got %s", StatusRunning, state.Status)
	}

	if state.Iteration != 0 {
		t.Errorf("expected iteration 0, got %d", state.Iteration)
	}

	if state.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}

	if state.LastUpdated.IsZero() {
		t.Error("expected LastUpdated to be set")
	}
}

func TestLoadState_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create a state file
	originalState := &State{
		Branch:      "test-branch",
		Iteration:   5,
		Status:      StatusCompleted,
		StartedAt:   time.Now().Add(-1 * time.Hour),
		LastUpdated: time.Now().Add(-30 * time.Minute),
		CompletedAt: time.Now().Add(-15 * time.Minute),
	}

	data, err := json.Marshal(originalState)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	// Load the state
	state, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if state.Branch != originalState.Branch {
		t.Errorf("expected branch %s, got %s", originalState.Branch, state.Branch)
	}

	if state.Iteration != originalState.Iteration {
		t.Errorf("expected iteration %d, got %d", originalState.Iteration, state.Iteration)
	}

	if state.Status != originalState.Status {
		t.Errorf("expected status %s, got %s", originalState.Status, state.Status)
	}
}

func TestLoadState_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON
	if err := os.WriteFile(statePath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	_, err := LoadState(statePath)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestLoadState_EmptyStatus(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create state with empty status
	invalidState := map[string]interface{}{
		"branch":       "test",
		"iteration":    1,
		"status":       "",
		"started_at":   time.Now(),
		"last_updated": time.Now(),
	}

	data, err := json.Marshal(invalidState)
	if err != nil {
		t.Fatalf("failed to marshal state: %v", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		t.Fatalf("failed to write state file: %v", err)
	}

	_, err = LoadState(statePath)
	if err == nil {
		t.Error("expected error for empty status, got nil")
	}
}

func TestSaveState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "subdir", "state.json")

	state := &State{
		Branch:      "main",
		Iteration:   3,
		Status:      StatusRunning,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	if err := SaveState(statePath, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file was not created")
	}

	// Load and verify
	loaded, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("failed to load saved state: %v", err)
	}

	if loaded.Branch != state.Branch {
		t.Errorf("expected branch %s, got %s", state.Branch, loaded.Branch)
	}

	if loaded.Iteration != state.Iteration {
		t.Errorf("expected iteration %d, got %d", state.Iteration, loaded.Iteration)
	}

	if loaded.Status != state.Status {
		t.Errorf("expected status %s, got %s", state.Status, loaded.Status)
	}
}

func TestSaveState_UpdatesTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state := &State{
		Branch:      "main",
		Iteration:   1,
		Status:      StatusRunning,
		StartedAt:   time.Now(),
		LastUpdated: time.Now().Add(-1 * time.Hour),
	}

	oldTimestamp := state.LastUpdated

	// Wait a bit to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	if err := SaveState(statePath, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	if !state.LastUpdated.After(oldTimestamp) {
		t.Error("expected LastUpdated to be updated")
	}
}

func TestSaveState_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	state := &State{
		Branch:      "main",
		Iteration:   1,
		Status:      StatusRunning,
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	if err := SaveState(statePath, state); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify temp file is cleaned up
	tmpPath := statePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}
}

func TestState_AllStatuses(t *testing.T) {
	statuses := []Status{
		StatusRunning,
		StatusCompleted,
		StatusRateLimited,
		StatusError,
		StatusAuthError,
	}

	tmpDir := t.TempDir()

	for _, status := range statuses {
		statePath := filepath.Join(tmpDir, string(status)+".json")

		state := &State{
			Branch:      "test",
			Iteration:   1,
			Status:      status,
			StartedAt:   time.Now(),
			LastUpdated: time.Now(),
		}

		if err := SaveState(statePath, state); err != nil {
			t.Fatalf("SaveState failed for status %s: %v", status, err)
		}

		loaded, err := LoadState(statePath)
		if err != nil {
			t.Fatalf("LoadState failed for status %s: %v", status, err)
		}

		if loaded.Status != status {
			t.Errorf("expected status %s, got %s", status, loaded.Status)
		}
	}
}
