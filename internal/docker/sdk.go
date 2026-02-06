package docker

import (
	"context"
	"fmt"
	"sync"

	"github.com/docker/docker/client"
)

// sdkClient wraps the Docker SDK client with lazy initialization.
// It provides thread-safe access to the Docker client and handles
// connection lifecycle management.
type sdkClient struct {
	cli *client.Client
	mu  sync.Mutex
}

// newSDKClient creates a new sdkClient instance.
func newSDKClient() *sdkClient {
	return &sdkClient{}
}

// getClient returns the Docker SDK client, creating it lazily on first use.
// It detects the Docker host from the active Docker context (supporting Docker Desktop)
// and enables API version negotiation for compatibility across different Docker versions.
func (s *sdkClient) getClient(ctx context.Context) (*client.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cli == nil {
		host, err := detectDockerHost()
		if err != nil {
			return nil, fmt.Errorf("failed to detect Docker host: %w", err)
		}

		opts := []client.Opt{client.WithAPIVersionNegotiation()}
		if host != "" {
			opts = append(opts, client.WithHost(host))
		} else {
			opts = append(opts, client.FromEnv)
		}

		cli, err := client.NewClientWithOpts(opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create Docker client: %w", err)
		}

		s.cli = cli
	}

	return s.cli, nil
}

// close releases resources associated with the Docker client.
func (s *sdkClient) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cli != nil {
		err := s.cli.Close()
		s.cli = nil

		return err
	}

	return nil
}
