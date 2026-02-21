// Package version provides version information for the spinner CLI.
// These variables are set at build time using ldflags.
package version

// Build-time variables (set via -ldflags)
var (
	// Version is the semantic version (e.g., "1.0.0")
	Version = "dev"

	// Commit is the git commit SHA
	Commit = "unknown"

	// Date is the build date
	Date = "unknown"
)

// Info returns a formatted version string.
func Info() string {
	return Version
}

// IsRelease reports whether this is a release build (version set via ldflags).
func IsRelease() bool {
	return Version != "dev"
}

// Tag returns the version as a git tag (e.g. "v1.0.0").
func Tag() string {
	return "v" + Version
}
