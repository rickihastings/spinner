package provider

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	assert.NotNil(t, f)
	assert.Empty(t, f.Available())
}

func TestFactory_RegisterAndCreate(t *testing.T) {
	f := NewFactory()
	mock := &MockProvider{}

	f.Register("docker", func() (Provider, error) {
		return mock, nil
	})

	p, err := f.Create("docker")
	require.NoError(t, err)
	assert.Equal(t, mock, p)
}

func TestFactory_CreateUnknownBackend(t *testing.T) {
	f := NewFactory()
	f.Register("docker", func() (Provider, error) {
		return &MockProvider{}, nil
	})

	_, err := f.Create("gcp")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown backend: "gcp"`)
	assert.Contains(t, err.Error(), "docker")
}

func TestFactory_CreateConstructorError(t *testing.T) {
	f := NewFactory()
	f.Register("broken", func() (Provider, error) {
		return nil, errors.New("connection failed")
	})

	_, err := f.Create("broken")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestFactory_Available(t *testing.T) {
	f := NewFactory()
	f.Register("gcp", func() (Provider, error) { return nil, nil })
	f.Register("docker", func() (Provider, error) { return nil, nil })
	f.Register("aws", func() (Provider, error) { return nil, nil })

	available := f.Available()
	assert.Equal(t, []string{"aws", "docker", "gcp"}, available)
}

func TestFactory_AvailableEmpty(t *testing.T) {
	f := NewFactory()
	assert.Empty(t, f.Available())
}

func TestFactory_RegisterOverwrite(t *testing.T) {
	f := NewFactory()
	mock1 := &MockProvider{}
	mock2 := &MockProvider{}

	f.Register("docker", func() (Provider, error) { return mock1, nil })
	f.Register("docker", func() (Provider, error) { return mock2, nil })

	p, err := f.Create("docker")
	require.NoError(t, err)
	assert.Equal(t, mock2, p)
}
