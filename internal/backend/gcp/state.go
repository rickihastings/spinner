package gcp

import (
	"context"
	"fmt"
)

// stateObjectPath returns the GCS object path for an instance's state file.
func stateObjectPath(instanceName string) string {
	return instanceName + "/state.json"
}

// readState downloads the state JSON from GCS for the given instance.
// Returns the raw JSON bytes, or an error if the read fails.
// If the state object does not exist, returns (nil, nil) to indicate a fresh start.
func readState(ctx context.Context, client Client, bucket, instanceName string) ([]byte, error) {
	object := stateObjectPath(instanceName)

	exists, err := client.ObjectExists(ctx, bucket, object)
	if err != nil {
		return nil, fmt.Errorf("failed to check state existence: %w", err)
	}

	if !exists {
		return nil, nil
	}

	data, err := client.ReadObject(ctx, bucket, object)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from gs://%s/%s: %w", bucket, object, err)
	}

	return data, nil
}
