package docker

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// spinConfig contains configuration for spinning up a container.
type spinConfig struct {
	Image         string
	Repo          string
	Prompt        string
	Branch        string
	MaxIterations string
	EnvVars       map[string]string
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

// escapeShellArg escapes a string for safe use as a shell argument.
// Wraps the string in single quotes and escapes any single quotes within.
func escapeShellArg(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
}

// convertSshToHttps converts SSH Git URLs to HTTPS format for GitHub PAT authentication.
// Example: git@github.com:user/repo.git -> https://github.com/user/repo.git
func convertSshToHttps(repoURL string) string {
	if strings.HasPrefix(repoURL, "git@github.com:") {
		return strings.Replace(repoURL, "git@github.com:", "https://github.com/", 1)
	}

	return repoURL
}

// sanitizeComponent sanitizes a component for use in a Docker container name.
// Converts to lowercase, replaces invalid characters with hyphens,
// collapses consecutive hyphens, and trims leading/trailing hyphens.
func sanitizeComponent(input string) string {
	// Convert to lowercase
	result := strings.ToLower(input)

	// Replace invalid characters with hyphens
	re := regexp.MustCompile(`[^a-z0-9-_]`)
	result = re.ReplaceAllString(result, "-")

	// Collapse consecutive hyphens
	re = regexp.MustCompile(`-+`)
	result = re.ReplaceAllString(result, "-")

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	return result
}

// extractRepoName extracts the repository name from a Git URL.
// Handles both SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git) formats.
func extractRepoName(repoURL string) string {
	re := regexp.MustCompile(`([^/:]+)(\.git)?$`)

	matches := re.FindStringSubmatch(repoURL)
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
	// Convert SSH URLs to HTTPS for GitHub PAT authentication
	repoURL := convertSshToHttps(config.Repo)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create temp file for all environment variables
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

	// Write built-in environment variables
	_, _ = fmt.Fprintf(tmpFile, "GITHUB_TOKEN=%s\n", os.Getenv("GITHUB_TOKEN"))
	_, _ = fmt.Fprintf(tmpFile, "CLAUDE_CODE_OAUTH_TOKEN=%s\n", os.Getenv("CLAUDE_CODE_OAUTH_TOKEN"))
	_, _ = fmt.Fprintf(tmpFile, "REPO_URL=%s\n", repoURL)

	// Add branch if specified
	if config.Branch != "" {
		_, _ = fmt.Fprintf(tmpFile, "BRANCH=%s\n", config.Branch)
	}

	// Add Ralph loop environment variables if prompt is provided
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

	dockerArgs := []string{
		"run",
		"-d",
		"--name",
		containerName,
		"--env-file",
		tmpFilePath,
		"-v",
		fmt.Sprintf("%s/.spinner/%s/logs:/logs", homeDir, containerName),
		"-v",
		fmt.Sprintf("%s/.spinner/%s/state:/state", homeDir, containerName),
	}

	// Add .npmrc mount if it exists
	if hasNpmrc {
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/spinner/.npmrc", npmrcPath))
	}

	// Add image
	dockerArgs = append(dockerArgs, config.Image)

	return dockerArgs, tmpFilePath, nil
}
