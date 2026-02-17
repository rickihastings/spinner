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
	reRepoName        = regexp.MustCompile(`([^/:]+)(\.git)?$`)
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
	Passphrase    string
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

// extractRepoName extracts the repository name from a Git URL.
// Handles both SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git) formats.
func extractRepoName(repoURL string) string {
	matches := reRepoName.FindStringSubmatch(repoURL)
	if len(matches) > 1 {
		return strings.TrimSuffix(matches[1], ".git")
	}

	return "sandbox"
}

// generateContainerName generates a deterministic container name based on image, repo, and branch.
// Format: {image}-{repo} or {image}-{repo}-{branch}
func generateContainerName(config spinConfig) string {
	imagePart := sanitizeComponent(strings.ReplaceAll(config.Image, ":", "-"))
	repoPart := sanitizeComponent(extractRepoName(config.Repo))

	if config.Branch != "" {
		branchPart := sanitizeComponent(config.Branch)
		return fmt.Sprintf("%s-%s-%s", imagePart, repoPart, branchPart)
	}

	return fmt.Sprintf("%s-%s", imagePart, repoPart)
}

// buildDockerRunCommand builds the docker run command arguments.
// Returns docker args and a temp file path (or empty string if no temp file created).
// The caller MUST delete the temp file after docker run completes.
func buildDockerRunCommand(config spinConfig, containerName string, hasNpmrc bool) ([]string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get home directory: %w", err)
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

	// Pass SPINNER_SECRET_PASSPHRASE so startup.sh can decrypt the blob
	if config.Passphrase != "" {
		_, _ = fmt.Fprintf(tmpFile, "SPINNER_SECRET_PASSPHRASE=%s\n", config.Passphrase)
	}

	// Write custom environment variables
	for key, value := range config.EnvVars {
		_, _ = fmt.Fprintf(tmpFile, "%s=%s\n", key, value)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFilePath)

		return nil, "", fmt.Errorf("failed to close env file: %w", err)
	}

	// Write encrypted blob to host-mounted directory for container access
	if len(config.SecretBlob) > 0 {
		blobDir := filepath.Join(homeDir, ".spinner", containerName)
		if err := os.MkdirAll(blobDir, 0700); err != nil {
			return nil, "", fmt.Errorf("failed to create blob directory: %w", err)
		}

		blobPath := filepath.Join(blobDir, "secrets.enc")
		if err := os.WriteFile(blobPath, config.SecretBlob, 0600); err != nil {
			return nil, "", fmt.Errorf("failed to write secrets blob: %w", err)
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
		fmt.Sprintf("%s/.spinner/%s/logs:/logs", homeDir, containerName),
		"-v",
		fmt.Sprintf("%s/.spinner/%s/state:/state", homeDir, containerName),
	}

	// Mount encrypted secrets blob read-only into container
	if len(config.SecretBlob) > 0 {
		blobPath := filepath.Join(homeDir, ".spinner", containerName, "secrets.enc")
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/run/spinner/secrets.enc:ro", blobPath))
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
