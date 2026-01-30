package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
)

// spinCmd is the production spin command using RealDockerClient
var spinCmd = NewSpinCommand(docker.NewRealDockerClient())

func init() {
	rootCmd.AddCommand(spinCmd)
}
