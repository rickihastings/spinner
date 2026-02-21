package util

import (
	"regexp"
	"strings"
)

var reRepoName = regexp.MustCompile(`([^/:]+)(\.git)?$`)

// ConvertSshToHttps converts SSH Git URLs to HTTPS format for GitHub PAT authentication.
// Example: git@github.com:user/repo.git -> https://github.com/user/repo.git
func ConvertSshToHttps(repoURL string) string {
	if strings.HasPrefix(repoURL, "git@github.com:") {
		return strings.Replace(repoURL, "git@github.com:", "https://github.com/", 1)
	}

	return repoURL
}

// ExtractRepoName extracts the repository name from a Git URL.
// Handles both SSH (git@github.com:user/repo.git) and HTTPS (https://github.com/user/repo.git) formats.
func ExtractRepoName(repoURL string) string {
	matches := reRepoName.FindStringSubmatch(repoURL)
	if len(matches) > 1 {
		return strings.TrimSuffix(matches[1], ".git")
	}

	return "sandbox"
}
