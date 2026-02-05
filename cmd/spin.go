package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
)

// spinCmd is the production spin command using Provider
var spinCmd = NewSpinCommand(docker.NewDockerProvider(docker.NewRealDockerClient()))

func init() {
	rootCmd.AddCommand(spinCmd)
}
