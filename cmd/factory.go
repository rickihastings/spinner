package cmd

import (
	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/provider"
)

// defaultFactory is the production provider factory with all backends registered.
var defaultFactory = newDefaultFactory()

func newDefaultFactory() *provider.Factory {
	f := provider.NewFactory()

	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return docker.NewDockerProvider(docker.NewRealDockerClient()), nil
	})

	return f
}
