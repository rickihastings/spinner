package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rickihastings/spinner/internal/util"
)

// Pre-compiled regexes for container name sanitization.
var (
	reInvalidChars    = regexp.MustCompile(`[^a-z0-9-_]`)
	reConsecutiveDash = regexp.MustCompile(`-+`)
)

// spinConfig contains configuration for spinning up a container.
type spinConfig struct {
	Image         string
	Repo          string
	Prompt        string
	Branch        string
	MaxIterations string
	Model         string
	EnvVars       map[string]string
	EnvFile       string
	SecretBlob    []byte
	SecretKey     []byte
	ExtraArgs     []string
}

// ContainerResult contains the result of a container operation.
type ContainerResult struct {
	Success       bool
	ContainerName string
	Error         string
}

// ContainerStatus represents the status of a Docker container.
type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
	StatusNone    ContainerStatus = "none"
)

// defaultMaxIterations is the default maximum number of iterations for the exec loop.
const defaultMaxIterations = "100"

// sanitizeComponent sanitizes a component for use in a Docker container name.
// Converts to lowercase, replaces invalid characters with hyphens,
// collapses consecutive hyphens, and trims leading/trailing hyphens.
func sanitizeComponent(input string) string {
	// Convert to lowercase
	result := strings.ToLower(input)

	// Replace invalid characters with hyphens
	result = reInvalidChars.ReplaceAllString(result, "-")

	// Collapse consecutive hyphens
	result = reConsecutiveDash.ReplaceAllString(result, "-")

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	return result
}

// generateContainerName generates a deterministic container name based on image, repo, and branch.
// Format: {image}-{repo} or {image}-{repo}-{branch}
func generateContainerName(config spinConfig) string {
	imagePart := sanitizeComponent(strings.ReplaceAll(config.Image, ":", "-"))
	repoPart := sanitizeComponent(util.ExtractRepoName(config.Repo))

	if config.Branch != "" {
		branchPart := sanitizeComponent(config.Branch)
		return fmt.Sprintf("%s-%s-%s", imagePart, repoPart, branchPart)
	}

	return fmt.Sprintf("%s-%s", imagePart, repoPart)
}

// inceptionMountBase is the container-local mount point for the outer state directory.
// Overridden in tests to avoid writing to /state on the host machine.
var inceptionMountBase = "/state"

// buildDockerRunCommand builds the docker run command arguments.
// Returns docker args and a temp file path (or empty string if no temp file created).
// The caller MUST delete the temp file after docker run completes.
//
// In normal mode (running on the host), files are written to ~/.spinner/<containerName>/
// and the same paths are passed as Docker volume mount sources.
//
// In inception mode (running inside a spinner container, detected via SPINNER_HOST_STATE_DIR),
// files are written to the outer container's /state/<containerName>/ directory which is
// bind-mounted from the host. The host-side path (from SPINNER_HOST_STATE_DIR) is passed
// to Docker for volume mounts so the host Docker daemon can resolve them correctly.
func buildDockerRunCommand(config spinConfig, containerName string, hasNpmrc bool) ([]string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Determine write paths (where this process creates files) and host paths (what Docker
	// passes to the daemon for volume mounts). In inception mode these diverge: files are
	// written inside the container at /state/<name>/ but Docker needs the host-side path.
	var fileBase string // path in this process's filesystem for writing files

	var hostBase string // path the host Docker daemon uses for volume mounts

	if parentStateDir := os.Getenv("SPINNER_HOST_STATE_DIR"); parentStateDir != "" {
		// Inception mode: write into the outer container's bind-mounted /state/ dir.
		// The outer container's /state maps to parentStateDir on the host, so sibling
		// directories written there are accessible to the host Docker daemon.
		fileBase = filepath.Join(inceptionMountBase, containerName)
		hostBase = filepath.Join(parentStateDir, containerName)
	} else {
		fileBase = filepath.Join(homeDir, ".spinner", containerName)
		hostBase = fileBase
	}

	// Pre-create state and logs directories now so Docker doesn't create them as root.
	// Use 0777 because the container's spinner user (different UID) needs write access.
	// The parent directory (fileBase) is 0700 which restricts access on the host.
	for _, subDir := range []string{"state", "logs"} {
		if err := os.MkdirAll(filepath.Join(fileBase, subDir), 0777); err != nil {
			return nil, "", fmt.Errorf("failed to create %s directory: %w", subDir, err)
		}
	}

	// Create temp file for non-secret environment variables
	tmpFile, err := os.CreateTemp("", "spinner-env-")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create env file: %w", err)
	}

	tmpFilePath := tmpFile.Name()

	// Set permissions to 0600 (owner read/write only)
	if err := os.Chmod(tmpFilePath, 0600); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFilePath)

		return nil, "", fmt.Errorf("failed to set env file permissions: %w", err)
	}

	// Write non-secret built-in environment variables.
	// Tokens (GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN) travel via the encrypted blob.
	_, _ = fmt.Fprintf(tmpFile, "REPO_URL=%s\n", config.Repo)

	// Pass the host-side state directory path so the container can support inception:
	// if it runs `spinner spin` itself, it uses this to resolve host-accessible paths.
	_, _ = fmt.Fprintf(tmpFile, "SPINNER_HOST_STATE_DIR=%s\n", filepath.Join(hostBase, "state"))

	// Pass host git user config so commits are attributed correctly
	if name := gitConfigValue("user.name"); name != "" {
		_, _ = fmt.Fprintf(tmpFile, "GIT_USER_NAME=%s\n", name)
	}

	if email := gitConfigValue("user.email"); email != "" {
		_, _ = fmt.Fprintf(tmpFile, "GIT_USER_EMAIL=%s\n", email)
	}

	// Add branch if specified
	if config.Branch != "" {
		_, _ = fmt.Fprintf(tmpFile, "BRANCH=%s\n", config.Branch)
	}

	// Add model if specified
	if config.Model != "" {
		_, _ = fmt.Fprintf(tmpFile, "ANTHROPIC_MODEL=%s\n", config.Model)
	}

	// Add exec loop environment variables if prompt is provided
	if config.Prompt != "" {
		_, _ = fmt.Fprintf(tmpFile, "PROMPT=%s\n", config.Prompt)

		maxIterations := config.MaxIterations
		if maxIterations == "" {
			maxIterations = defaultMaxIterations
		}

		_, _ = fmt.Fprintf(tmpFile, "MAX_ITERATIONS=%s\n", maxIterations)
		_, _ = fmt.Fprintf(tmpFile, "LOG_DIR=/logs\n")
	}

	// Write custom environment variables
	for key, value := range config.EnvVars {
		_, _ = fmt.Fprintf(tmpFile, "%s=%s\n", key, value)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFilePath)

		return nil, "", fmt.Errorf("failed to close env file: %w", err)
	}

	// Write encrypted blob and key file
	if len(config.SecretBlob) > 0 {
		if err := os.MkdirAll(fileBase, 0700); err != nil {
			return nil, "", fmt.Errorf("failed to create blob directory: %w", err)
		}

		blobPath := filepath.Join(fileBase, "secrets.enc")
		if err := os.WriteFile(blobPath, config.SecretBlob, 0644); err != nil {
			return nil, "", fmt.Errorf("failed to write secrets blob: %w", err)
		}

		if len(config.SecretKey) > 0 {
			keyPath := filepath.Join(fileBase, "secrets.key")
			if err := os.WriteFile(keyPath, config.SecretKey, 0644); err != nil {
				return nil, "", fmt.Errorf("failed to write secrets key: %w", err)
			}
		}
	}

	dockerArgs := []string{
		"run",
		"-d",
		"--name",
		containerName,
		"--label",
		"spinner-managed=true",
		"--env-file",
		tmpFilePath,
		"-v",
		fmt.Sprintf("%s:/logs", filepath.Join(hostBase, "logs")),
		"-v",
		fmt.Sprintf("%s:/state", filepath.Join(hostBase, "state")),
	}

	// Mount encrypted secrets blob and key file read-only into container
	if len(config.SecretBlob) > 0 {
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/run/spinner/secrets.enc:ro", filepath.Join(hostBase, "secrets.enc")))

		if len(config.SecretKey) > 0 {
			dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/run/spinner/secrets.key:ro", filepath.Join(hostBase, "secrets.key")))
		}
	}

	// Add .npmrc mount if it exists
	if hasNpmrc {
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/spinner/.npmrc", npmrcPath))
	}

	// Add user's env file if specified
	if config.EnvFile != "" {
		dockerArgs = append(dockerArgs, "--env-file", config.EnvFile)
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/tmp/.env:ro", config.EnvFile))
	}

	// Append provider pass-through args before the image
	dockerArgs = append(dockerArgs, util.SplitArgs(config.ExtraArgs)...)

	// Add image (must be last)
	dockerArgs = append(dockerArgs, config.Image)

	return dockerArgs, tmpFilePath, nil
}

// dockerManagedRunFlags lists flags that Spinner manages in docker run commands.
// Provider args that conflict with these are rejected to avoid breaking Spinner's
// internal wiring.
var dockerManagedRunFlags = []string{
	"-d", "--detach",
	"--name",
	"--label",
	"--env-file",
}

// ValidateDockerRunArgs checks that provider args don't conflict with
// Spinner-managed docker run flags. Returns an error listing all conflicts.
func ValidateDockerRunArgs(args []string) error {
	var conflicts []string

	for _, arg := range args {
		for _, managed := range dockerManagedRunFlags {
			if arg == managed || strings.HasPrefix(arg, managed+"=") {
				conflicts = append(conflicts, arg)
			}
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("--provider-args conflicts with Spinner-managed docker run flags: %s", strings.Join(conflicts, ", "))
	}

	return nil
}

// dockerManagedBuildFlags lists flags that Spinner manages in docker build commands.
// Provider args that conflict with these are rejected to avoid breaking Spinner's
// internal wiring.
var dockerManagedBuildFlags = []string{
	"-t", "--tag",
}

// ValidateDockerBuildArgs checks that provider args don't conflict with
// Spinner-managed docker build flags. Returns an error listing all conflicts.
func ValidateDockerBuildArgs(args []string) error {
	var conflicts []string

	for _, arg := range args {
		for _, managed := range dockerManagedBuildFlags {
			if arg == managed || strings.HasPrefix(arg, managed+"=") {
				conflicts = append(conflicts, arg)
			}
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf("--provider-args conflicts with Spinner-managed docker build flags: %s", strings.Join(conflicts, ", "))
	}

	return nil
}

// gitConfigValue reads a git config value from the host machine.
// Returns empty string if the value is not set or git is not available.
func gitConfigValue(key string) string {
	out, err := exec.Command("git", "config", key).Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
