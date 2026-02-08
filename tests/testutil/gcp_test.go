package testutil

import (
	"testing"
)

func TestComputeConfigHash(t *testing.T) {
	tests := []struct {
		name     string
		project  string
		zone     string
		bucket   string
		wantSame bool
	}{
		{
			name:     "same config produces same hash",
			project:  "test-project",
			zone:     "us-central1-a",
			bucket:   "test-bucket",
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := ComputeConfigHash(tt.project, tt.zone, tt.bucket)
			hash2 := ComputeConfigHash(tt.project, tt.zone, tt.bucket)

			// Verify hash is 8 characters
			if len(hash1) != 8 {
				t.Errorf("ComputeConfigHash() produced hash of length %d, want 8", len(hash1))
			}

			// Verify determinism
			if hash1 != hash2 {
				t.Errorf("ComputeConfigHash() not deterministic: got %s and %s", hash1, hash2)
			}
		})
	}

	// Test that different configs produce different hashes
	hash1 := ComputeConfigHash("project1", "zone1", "bucket1")
	hash2 := ComputeConfigHash("project2", "zone1", "bucket1")
	hash3 := ComputeConfigHash("project1", "zone2", "bucket1")
	hash4 := ComputeConfigHash("project1", "zone1", "bucket2")

	if hash1 == hash2 {
		t.Errorf("Different projects produced same hash: %s", hash1)
	}

	if hash1 == hash3 {
		t.Errorf("Different zones produced same hash: %s", hash1)
	}

	if hash1 == hash4 {
		t.Errorf("Different buckets produced same hash: %s", hash1)
	}
}

func TestGenerateSharedImageName(t *testing.T) {
	tests := []struct {
		name    string
		version int
		project string
		zone    string
		bucket  string
		want    string
	}{
		{
			name:    "generates expected format",
			version: 1,
			project: "test-project",
			zone:    "us-central1-a",
			bucket:  "test-bucket",
			want:    "", // We'll verify format, not exact string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSharedImageName(tt.version, tt.project, tt.zone, tt.bucket)

			// Verify format: test-shared-v{version}-{hash}
			expectedPrefix := "test-shared-v1-"
			if len(got) < len(expectedPrefix)+8 {
				t.Errorf("GenerateSharedImageName() = %v, expected longer string", got)
			}

			if got[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("GenerateSharedImageName() = %v, expected prefix %v", got, expectedPrefix)
			}

			// Verify hash portion is 8 characters
			hashPart := got[len(expectedPrefix):]
			if len(hashPart) != 8 {
				t.Errorf("GenerateSharedImageName() hash part = %v, expected 8 characters", hashPart)
			}
		})
	}

	// Test that different versions produce different names
	name1 := GenerateSharedImageName(1, "project", "zone", "bucket")
	name2 := GenerateSharedImageName(2, "project", "zone", "bucket")

	if name1 == name2 {
		t.Errorf("Different versions produced same name: %s", name1)
	}

	// Test that different configs produce different names
	name3 := GenerateSharedImageName(1, "project1", "zone", "bucket")
	name4 := GenerateSharedImageName(1, "project2", "zone", "bucket")

	if name3 == name4 {
		t.Errorf("Different projects produced same name: %s", name3)
	}
}
