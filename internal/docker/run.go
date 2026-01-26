package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type SpinConfig struct {
	Image         string
	Repo          string
	Prompt        string
	Branch        string
	MaxIterations string
	Recreate      bool
}

type ValidationResult struct {
	Valid    bool
	Error    string
	Warnings []string
	HasNpmrc bool
}

type ContainerResult struct {
	Success       bool
	ContainerName string
	Error         string
}

type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
	StatusNone    ContainerStatus = "none"
)

type ReuseAction string

const (
	ActionCreated   ReuseAction = "created"
	ActionReused    ReuseAction = "reused"
	ActionRestarted ReuseAction = "restarted"
)

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
func convertSshToHttps(repoUrl string) string {
	if strings.HasPrefix(repoUrl, "git@github.com:") {
		return strings.Replace(repoUrl, "git@github.com:", "https://github.com/", 1)
	}
	return repoUrl
}

// ValidatePrerequisites validates prerequisites for spinning up a container.
// Checks: Docker image exists, valid git repo URL, GITHUB_TOKEN is set, CLAUDE_CODE_OAUTH_TOKEN is set, npmrc availability.
func ValidatePrerequisites(config SpinConfig) ValidationResult {
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
	cmd := exec.Command("docker", "image", "inspect", config.Image)
	if err := cmd.Run(); err != nil {
		return ValidationResult{
			Valid:    false,
			Error:    fmt.Sprintf("Docker image '%s' not found", config.Image),
			Warnings: warnings,
			HasNpmrc: false,
		}
	}

	// Check GITHUB_TOKEN
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return ValidationResult{
			Valid:    false,
			Error:    "GITHUB_TOKEN environment variable not set. Please set GITHUB_TOKEN before running spin.",
			Warnings: warnings,
			HasNpmrc: false,
		}
	}

	// Check CLAUDE_CODE_OAUTH_TOKEN
	claudeToken := os.Getenv("CLAUDE_CODE_OAUTH_TOKEN")
	if claudeToken == "" {
		return ValidationResult{
			Valid:    false,
			Error:    "CLAUDE_CODE_OAUTH_TOKEN environment variable not set. Please set CLAUDE_CODE_OAUTH_TOKEN before running spin.",
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
	} else {
		warnings = append(warnings, "~/.npmrc not found, npm will use default registry")
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
func extractRepoName(repoUrl string) string {
	re := regexp.MustCompile(`([^/:]+)(\.git)?$`)
	matches := re.FindStringSubmatch(repoUrl)
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
func BuildDockerRunCommand(config SpinConfig, containerName string, hasNpmrc bool) []string {
	// Convert SSH URLs to HTTPS for GitHub PAT authentication
	repoUrl := convertSshToHttps(config.Repo)

	homeDir, _ := os.UserHomeDir()

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
		fmt.Sprintf("REPO_URL=%s", repoUrl),
		"-v",
		fmt.Sprintf("%s/.spinner/%s/logs:/logs", homeDir, containerName),
	}

	// Add Ralph loop environment variables if prompt is provided
	if config.Prompt != "" {
		dockerArgs = append(dockerArgs, "-e", fmt.Sprintf("PROMPT=%s", escapeShellArg(config.Prompt)))

		maxIterations := config.MaxIterations
		if maxIterations == "" {
			maxIterations = "100"
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

	return dockerArgs
}

// ExecuteDockerRun executes the docker run command.
func ExecuteDockerRun(dockerArgs []string, containerName string) ContainerResult {
	homeDir, _ := os.UserHomeDir()
	logsDir := filepath.Join(homeDir, ".spinner", containerName, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Failed to create logs directory: %s", err.Error()),
		}
	}

	cmd := exec.Command("docker", dockerArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container may have started but clone failed
		// Try to get the git error message from container logs
		logsCmd := exec.Command("docker", "logs", containerName)
		if logsOutput, logsErr := logsCmd.CombinedOutput(); logsErr == nil {
			return ContainerResult{
				Success:       false,
				ContainerName: containerName,
				Error:         fmt.Sprintf("Git clone failed: %s", strings.TrimSpace(string(logsOutput))),
			}
		}
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         strings.TrimSpace(string(output)),
		}
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}
}

// VerifyContainerStatus verifies that the container is running.
func VerifyContainerStatus(containerName string) ContainerResult {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         "Failed to verify container status",
		}
	}

	status := strings.TrimSpace(string(output))
	if status != "running" {
		// Get logs to show what went wrong
		logsCmd := exec.Command("docker", "logs", containerName)
		logsOutput, _ := logsCmd.CombinedOutput()
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         fmt.Sprintf("Container exited. Logs: %s", strings.TrimSpace(string(logsOutput))),
		}
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}
}

// CheckContainerExists checks if a container exists and returns its status.
// Returns 'running' if container exists and is running,
// 'stopped' if container exists but is not running,
// 'none' if container does not exist.
func CheckContainerExists(containerName string) ContainerStatus {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Container doesn't exist
		return StatusNone
	}

	status := strings.TrimSpace(string(output))
	if status == "running" {
		return StatusRunning
	}
	return StatusStopped
}

// RestartContainer restarts a stopped container.
func RestartContainer(containerName string) ContainerResult {
	cmd := exec.Command("docker", "start", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         strings.TrimSpace(string(output)),
		}
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}
}

// RemoveContainer removes a container, forcing removal if it's running.
func RemoveContainer(containerName string) ContainerResult {
	cmd := exec.Command("docker", "rm", "-f", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ContainerResult{
			Success:       false,
			ContainerName: containerName,
			Error:         strings.TrimSpace(string(output)),
		}
	}

	return ContainerResult{
		Success:       true,
		ContainerName: containerName,
	}
}
