package cmd

import (
	"context"

	"github.com/rickihastings/spinner/internal/docker"
	"github.com/rickihastings/spinner/internal/gcp"
	"github.com/rickihastings/spinner/internal/provider"
	"github.com/spf13/viper"
)

// defaultFactory is the production provider factory with all backends registered.
var defaultFactory = newDefaultFactory()

func newDefaultFactory() *provider.Factory {
	f := provider.NewFactory()

	f.Register(provider.BackendDocker, func() (provider.Provider, error) {
		return docker.NewDockerProvider(docker.NewRealDockerClient()), nil
	})

	f.Register(provider.BackendGCP, func() (provider.Provider, error) {
		project := viper.GetString(flagProject)
		zone := viper.GetString(flagZone)
		bucket := viper.GetString(flagStateBucket)

		client, err := gcp.NewRealGCPClient(context.Background())
		if err != nil {
			return nil, err
		}

		return gcp.NewGCPProvider(client, project, zone, bucket), nil
	})

	return f
}
