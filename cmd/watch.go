package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
)

// watchCmd is the production watch command using Provider
var watchCmd = NewWatchCommand(docker.NewDockerProvider(docker.NewRealDockerClient()))

func init() {
	rootCmd.AddCommand(watchCmd)
}
