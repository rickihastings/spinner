package gcp

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/rickihastings/spinner/internal/provider"
)

const (
	// instancePrefix is the prefix for all spinner-managed GCP instances.
	instancePrefix = "spinner"

	// maxInstanceNameLength is the maximum length for GCP instance names.
	maxInstanceNameLength = 63

	// hashSuffixLength is the length of the hash suffix used for truncated names.
	// 8 hex chars = 4 bytes of hash = ~4 billion combinations.
	hashSuffixLength = 8
)

// invalidCharsRegex matches characters that are not allowed in GCP instance names.
var invalidCharsRegex = regexp.MustCompile(`[^a-z0-9-]`)

// consecutiveHyphensRegex matches runs of multiple hyphens.
var consecutiveHyphensRegex = regexp.MustCompile(`-{2,}`)

// GenerateInstanceName generates a deterministic GCP instance name from image, repo, and branch.
// Format: spinner-{image}-{repo}[-{branch}]
//
// GCP instance names must:
//   - Be lowercase
//   - Start with a letter
//   - Contain only [a-z0-9-]
//   - Be at most 63 characters
//   - Not end with a hyphen
//
// If the generated name exceeds 63 characters, it is truncated and a hash suffix
// is appended for uniqueness.
func GenerateInstanceName(image, repo, branch string) string {
	repoPart := extractRepoName(repo)

	var raw string
	if branch != "" {
		raw = fmt.Sprintf("%s-%s-%s-%s", instancePrefix, image, repoPart, branch)
	} else {
		raw = fmt.Sprintf("%s-%s-%s", instancePrefix, image, repoPart)
	}

	name := sanitizeGCPName(raw)

	if len(name) > maxInstanceNameLength {
		name = truncateWithHash(name, raw)
	}

	return name
}

// extractRepoName extracts the repository name from a Git URL.
// Handles SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git).
func extractRepoName(repoURL string) string {
	re := regexp.MustCompile(`([^/:]+)(\.git)?$`)
	matches := re.FindStringSubmatch(repoURL)

	if len(matches) > 1 {
		return strings.TrimSuffix(matches[1], ".git")
	}

	return "repo"
}

// sanitizeGCPName converts a string to a valid GCP instance name component.
// Lowercases, replaces invalid characters with hyphens, collapses consecutive
// hyphens, and trims leading/trailing hyphens.
func sanitizeGCPName(input string) string {
	result := strings.ToLower(input)
	result = invalidCharsRegex.ReplaceAllString(result, "-")
	result = consecutiveHyphensRegex.ReplaceAllString(result, "-")
	result = strings.Trim(result, "-")

	// Ensure name starts with a letter
	if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') {
		result = "s" + result
	}

	return result
}

// truncateWithHash truncates a name to fit within maxInstanceNameLength,
// appending a hash suffix for uniqueness. The hash is derived from the
// original (unsanitized) name to ensure determinism even after truncation.
func truncateWithHash(sanitized, original string) string {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(original)))
	suffix := hash[:hashSuffixLength]

	// Truncate: max - 1 (hyphen) - hashSuffixLength
	maxPrefix := maxInstanceNameLength - 1 - hashSuffixLength
	prefix := sanitized

	if len(prefix) > maxPrefix {
		prefix = prefix[:maxPrefix]
	}

	prefix = strings.TrimRight(prefix, "-")

	return prefix + "-" + suffix
}

// MapVMStatus maps a GCP VM status string to a provider.InstanceStatus.
//
// Mapping:
//   - RUNNING, PROVISIONING, STAGING -> InstanceStatusRunning
//   - TERMINATED, STOPPED, SUSPENDED -> InstanceStatusStopped
//   - STOPPING, SUSPENDING          -> InstanceStatusStopped (transitioning to stopped)
//   - Unknown / not found           -> InstanceStatusNone
func MapVMStatus(status string) provider.InstanceStatus {
	switch VMStatus(status) {
	case VMStatusRunning, VMStatusProvisioning, VMStatusStaging:
		return provider.InstanceStatusRunning
	case VMStatusTerminated, VMStatusStopped, VMStatusSuspended,
		VMStatusStopping, VMStatusSuspending:
		return provider.InstanceStatusStopped
	default:
		return provider.InstanceStatusNone
	}
}

// SanitizeLabel sanitizes a string for use as a GCP resource label value.
// Label values: lowercase, max 63 chars, [a-z0-9_-], must start with lowercase letter.
func SanitizeLabel(input string) string {
	result := strings.ToLower(input)
	// Replace invalid characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9_-]`)
	result = re.ReplaceAllString(result, "-")

	result = strings.Trim(result, "-_")

	if len(result) > 63 {
		result = result[:63]
		result = strings.TrimRight(result, "-_")
	}

	// Ensure starts with lowercase letter
	if len(result) > 0 && (result[0] < 'a' || result[0] > 'z') {
		result = "l" + result
	}

	if result == "" {
		result = "unknown"
	}

	return result
}
