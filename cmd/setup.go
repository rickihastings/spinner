package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
)

// setupCmd is the production setup command using RealDockerClient
var setupCmd = NewSetupCommand(docker.NewRealDockerClient())

func init() {
	rootCmd.AddCommand(setupCmd)
}
