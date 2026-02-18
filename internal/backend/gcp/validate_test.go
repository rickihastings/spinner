package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateGCPInstanceArgs_AllowsValidArgs(t *testing.T) {
	args := []string{
		"--machine-type=n2-standard-4",
		"--boot-disk-size=50GB",
		"--service-account=sa@proj.iam.gserviceaccount.com",
		"--accelerator=type=nvidia-tesla-t4,count=1",
		"--network=custom-vpc",
	}

	err := ValidateGCPInstanceArgs(args)
	assert.NoError(t, err)
}

func TestValidateGCPInstanceArgs_RejectsProjectFlag(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{"--project=other-project"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project=other-project")
	assert.Contains(t, err.Error(), "conflicts with Spinner-managed")
}

func TestValidateGCPInstanceArgs_RejectsZoneFlag(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{"--zone=us-west1-b"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--zone=us-west1-b")
}

func TestValidateGCPInstanceArgs_RejectsImageFlag(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{"--image=some-image"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--image=some-image")
}

func TestValidateGCPInstanceArgs_RejectsMetadataFlags(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{"metadata-from-file", "--metadata-from-file=key=file"},
		{"metadata", "--metadata=key=value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGCPInstanceArgs([]string{tt.arg})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.arg)
		})
	}
}

func TestValidateGCPInstanceArgs_RejectsLabelsFlag(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{"--labels=env=prod"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--labels=env=prod")
}

func TestValidateGCPInstanceArgs_RejectsMultipleConflicts(t *testing.T) {
	args := []string{
		"--machine-type=n2-standard-4",
		"--project=other",
		"--zone=us-west1-b",
	}

	err := ValidateGCPInstanceArgs(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--project=other")
	assert.Contains(t, err.Error(), "--zone=us-west1-b")
	// machine-type should NOT be in conflicts
	assert.NotContains(t, err.Error(), "--machine-type")
}

func TestValidateGCPInstanceArgs_EmptyArgs(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{})
	assert.NoError(t, err)
}

func TestValidateGCPInstanceArgs_NilArgs(t *testing.T) {
	err := ValidateGCPInstanceArgs(nil)
	assert.NoError(t, err)
}

func TestValidateGCPInstanceArgs_BareFlag(t *testing.T) {
	err := ValidateGCPInstanceArgs([]string{"--quiet"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--quiet")
}
