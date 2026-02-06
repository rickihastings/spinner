package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/rickihastings/spinner/internal/version"
	"github.com/spf13/cobra"
)

const (
	repoOwner = "rickihastings"
	repoName  = "spinner"
)

// updateCmd is the production update command
var updateCmd = NewUpdateCommand()

func init() {
	rootCmd.AddCommand(updateCmd)
}

// NewUpdateCommand creates a new update command.
// This command allows the CLI to self-update from GitHub releases.
func NewUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update spinner to the latest version",
		Long: `Update spinner to the latest version from GitHub releases.

This command checks for the latest release and updates the binary in place
if a newer version is available.`,
		RunE: runUpdate,
	}

	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	currentVersion := version.Version

	// Don't allow updates from dev builds
	if currentVersion == "dev" {
		return fmt.Errorf("cannot update a development build; please install a released version")
	}

	fmt.Printf("Current version: %s\n", currentVersion)
	fmt.Println("Checking for updates...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("failed to create GitHub source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return fmt.Errorf("failed to create updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug(repoOwner, repoName))
	if err != nil {
		return fmt.Errorf("failed to detect latest version: %w", err)
	}

	if !found {
		fmt.Println("No releases found.")
		return nil
	}

	if !latest.GreaterThan(currentVersion) {
		fmt.Printf("Already up to date (latest: %s)\n", latest.Version())
		return nil
	}

	fmt.Printf("New version available: %s\n", latest.Version())
	fmt.Printf("Release notes: %s\n", latest.ReleaseNotes)
	fmt.Println()
	fmt.Printf("Updating to %s...\n", latest.Version())

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	fmt.Printf("Successfully updated to %s\n", latest.Version())

	return nil
}
