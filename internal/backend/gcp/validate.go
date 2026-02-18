package gcp

import (
	"fmt"
	"strings"
)

// gcpManagedInstanceFlags lists gcloud compute instances create flags that
// Spinner manages internally. Provider args that conflict with these are
// rejected to avoid breaking Spinner's internal wiring (metadata, labels,
// image selection, etc.).
var gcpManagedInstanceFlags = []string{
	"--project",
	"--zone",
	"--image",
	"--image-project",
	"--metadata-from-file",
	"--metadata",
	"--labels",
	"--quiet",
	"--format",
}

// ValidateGCPInstanceArgs checks that provider args don't conflict with
// Spinner-managed gcloud compute instances create flags. Returns an error
// listing all conflicts.
func ValidateGCPInstanceArgs(args []string) error {
	var conflicts []string

	for _, arg := range args {
		for _, managed := range gcpManagedInstanceFlags {
			if arg == managed || strings.HasPrefix(arg, managed+"=") {
				conflicts = append(conflicts, arg)
			}
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("--provider-args conflicts with Spinner-managed gcloud flags: %s", strings.Join(conflicts, ", "))
	}

	return nil
}
