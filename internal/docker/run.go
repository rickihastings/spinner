package docker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rickihastings/spinner/internal/prerequisites"
)

// SpinConfig contains configuration for spinning up a container.
type SpinConfig struct {
	Image         string
	Repo          string
	Prompt        string
	Branch        string
	MaxIterations string
	Recreate      bool
}

// ValidationResult contains the result of prerequisite validation.
type ValidationResult struct {
	Valid    bool
	Error    string
	Warnings []string
	HasNpmrc bool
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

// ReuseAction represents the action taken when handling an existing container.
type ReuseAction string

const (
	ActionCreated   ReuseAction = "created"
	ActionReused    ReuseAction = "reused"
	ActionRestarted ReuseAction = "restarted"
)

// DefaultMaxIterations is the default maximum number of iterations for the exec loop.
const DefaultMaxIterations = "100"

// ReuseResult contains the status and action taken for container reuse.
type ReuseResult struct {
	Status ContainerStatus
	Action ReuseAction
}

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

// ValidatePrerequisitesWithClient validates prerequisites using a provided DockerClient.
func ValidatePrerequisitesWithClient(ctx context.Context, client DockerClient, config SpinConfig) ValidationResult {
	warnings := []string{}

	// Check if repo is a valid git URL
	isValidGitUrl := strings.HasPrefix(config.Repo, "http://") ||
		strings.HasPrefix(config.Repo, "https://") ||
		strings.HasPrefix(config.Repo, "git@")
	if !isValidGitUrl {
		return ValidationResult{
			Valid:    false,
			Error:    "Repository must be a valid git URL (https://, http://, or git@)",
			Warnings: warnings,
			HasNpmrc: false,
		}
	}

	// Check if Docker image exists
	exists, err := client.ImageExists(ctx, config.Image)
	if err != nil || !exists {
		return ValidationResult{
			Valid:    false,
			Error:    fmt.Sprintf("Docker image '%s' not found", config.Image),
			Warnings: warnings,
			HasNpmrc: false,
		}
	}

	// Check environment variables (GITHUB_TOKEN, CLAUDE_CODE_OAUTH_TOKEN)
	if err := prerequisites.CheckEnvironmentVariables(); err != nil {
		return ValidationResult{
			Valid:    false,
			Error:    err.Error(),
			Warnings: warnings,
			HasNpmrc: false,
		}
	}

	// Check ~/.npmrc
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	npmrcPath := filepath.Join(homeDir, ".npmrc")

	hasNpmrc := false
	if _, err := os.Stat(npmrcPath); err == nil {
		hasNpmrc = true
	}

	return ValidationResult{
		Valid:    true,
		Warnings: warnings,
		HasNpmrc: hasNpmrc,
	}
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

// GenerateContainerName generates a deterministic container name based on image, repo, and branch.
// Format: {image}-{repo} or {image}-{repo}-{branch}
func GenerateContainerName(config SpinConfig) string {
	imagePart := sanitizeComponent(strings.ReplaceAll(config.Image, ":", "-"))
	repoPart := sanitizeComponent(extractRepoName(config.Repo))

	if config.Branch != "" {
		branchPart := sanitizeComponent(config.Branch)
		return fmt.Sprintf("%s-%s-%s", imagePart, repoPart, branchPart)
	}

	return fmt.Sprintf("%s-%s", imagePart, repoPart)
}

// BuildDockerRunCommand builds the docker run command arguments.
func BuildDockerRunCommand(config SpinConfig, containerName string, hasNpmrc bool) ([]string, error) {
	// Convert SSH URLs to HTTPS for GitHub PAT authentication
	repoURL := convertSshToHttps(config.Repo)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dockerArgs := []string{
		"run",
		"-d",
		"--name",
		containerName,
		"-e",
		fmt.Sprintf("GITHUB_TOKEN=%s", os.Getenv("GITHUB_TOKEN")),
		"-e",
		fmt.Sprintf("CLAUDE_CODE_OAUTH_TOKEN=%s", os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")),
		"-e",
		fmt.Sprintf("REPO_URL=%s", repoURL),
		"-v",
		fmt.Sprintf("%s/.spinner/%s/logs:/logs", homeDir, containerName),
		"-v",
		fmt.Sprintf("%s/.spinner/%s/state:/state", homeDir, containerName),
	}

	// Add Ralph loop environment variables if prompt is provided
	if config.Prompt != "" {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("PROMPT=%s", escapeShellArg(config.Prompt)))

		maxIterations := config.MaxIterations
		if maxIterations == "" {
			maxIterations = DefaultMaxIterations
		}

		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("MAX_ITERATIONS=%s", maxIterations))

		// Add branch if specified
		if config.Branch != "" {
			dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("BRANCH=%s", escapeShellArg(config.Branch)))
		}
	}

	// Add .npmrc mount if it exists
	if hasNpmrc {
		npmrcPath := filepath.Join(homeDir, ".npmrc")
		dockerArgs = append(dockerArgs, "-v", fmt.Sprintf("%s:/home/spinner/.npmrc", npmrcPath))
	}

	// Add image
	dockerArgs = append(dockerArgs, config.Image)

	return dockerArgs, nil
}
