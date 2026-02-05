package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
)

// setupCmd is the production setup command using Provider
var setupCmd = NewSetupCommand(docker.NewDockerProvider(docker.NewRealDockerClient()))

func init() {
	rootCmd.AddCommand(setupCmd)
}
