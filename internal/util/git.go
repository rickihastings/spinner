package util

import "strings"

// ConvertSshToHttps converts SSH Git URLs to HTTPS format for GitHub PAT authentication.
// Example: git@github.com:user/repo.git -> https://github.com/user/repo.git
func ConvertSshToHttps(repoURL string) string {
	if strings.HasPrefix(repoURL, "git@github.com:") {
		return strings.Replace(repoURL, "git@github.com:", "https://github.com/", 1)
	}

	return repoURL
}
