package secret

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_AllTokensFromStore(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("gh-token-123", nil)
	store.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("claude-token-456", nil)

	secrets, err := Resolve(store, nil)
	require.NoError(t, err)
	assert.Equal(t, "gh-token-123", secrets["GITHUB_TOKEN"])
	assert.Equal(t, "claude-token-456", secrets["CLAUDE_CODE_OAUTH_TOKEN"])
	assert.Len(t, secrets, 2)
	store.AssertExpectations(t)
}

func TestResolve_MissingBuiltInToken(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("", fmt.Errorf("%w: GITHUB_TOKEN", ErrNotFound))

	secrets, err := Resolve(store, nil)
	assert.Error(t, err)
	assert.Nil(t, secrets)
	assert.Contains(t, err.Error(), "GITHUB_TOKEN")
	assert.Contains(t, err.Error(), "not found in store")
	assert.Contains(t, err.Error(), "spinner secret set")
	store.AssertExpectations(t)
}

func TestResolve_NoEnvFallback(t *testing.T) {
	// Even if GITHUB_TOKEN is in the environment, Resolve should not use it
	t.Setenv("GITHUB_TOKEN", "env-value-should-be-ignored")

	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("", fmt.Errorf("%w: GITHUB_TOKEN", ErrNotFound))

	secrets, err := Resolve(store, nil)
	assert.Error(t, err)
	assert.Nil(t, secrets)
	assert.Contains(t, err.Error(), "not found in store")
	store.AssertExpectations(t)
}

func TestResolve_CustomKeyFromStore(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("gh-token", nil)
	store.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("claude-token", nil)
	store.On("Get", "NPM_TOKEN").Return("npm-secret-123", nil)

	secrets, err := Resolve(store, []string{"NPM_TOKEN"})
	require.NoError(t, err)
	assert.Equal(t, "npm-secret-123", secrets["NPM_TOKEN"])
	assert.Len(t, secrets, 3)
	store.AssertExpectations(t)
}

func TestResolve_CustomKeyNotFound(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("gh-token", nil)
	store.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("claude-token", nil)
	store.On("Get", "NPM_TOKEN").Return("", fmt.Errorf("%w: NPM_TOKEN", ErrNotFound))

	secrets, err := Resolve(store, []string{"NPM_TOKEN"})
	assert.Error(t, err)
	assert.Nil(t, secrets)
	assert.Contains(t, err.Error(), "NPM_TOKEN")
	assert.Contains(t, err.Error(), "not found in store")
	assert.Contains(t, err.Error(), "spinner secret set")
	store.AssertExpectations(t)
}

func TestResolve_StoreError(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("", fmt.Errorf("decryption failed"))

	secrets, err := Resolve(store, nil)
	assert.Error(t, err)
	assert.Nil(t, secrets)
	assert.Contains(t, err.Error(), "reading secret")
	store.AssertExpectations(t)
}

func TestResolve_DuplicateCustomAndBuiltIn(t *testing.T) {
	// If a custom key duplicates a built-in key, it should not be resolved twice
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("gh-token", nil)
	store.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("claude-token", nil)

	secrets, err := Resolve(store, []string{"GITHUB_TOKEN"})
	require.NoError(t, err)
	assert.Equal(t, "gh-token", secrets["GITHUB_TOKEN"])
	assert.Len(t, secrets, 2)
	// GITHUB_TOKEN should only be called once (built-in), not again for custom
	store.AssertNumberOfCalls(t, "Get", 2)
}

func TestResolve_MultipleCustomKeys(t *testing.T) {
	store := new(MockStore)
	store.On("Get", "GITHUB_TOKEN").Return("gh-token", nil)
	store.On("Get", "CLAUDE_CODE_OAUTH_TOKEN").Return("claude-token", nil)
	store.On("Get", "NPM_TOKEN").Return("npm-123", nil)
	store.On("Get", "API_KEY").Return("api-456", nil)

	secrets, err := Resolve(store, []string{"NPM_TOKEN", "API_KEY"})
	require.NoError(t, err)
	assert.Len(t, secrets, 4)
	assert.Equal(t, "npm-123", secrets["NPM_TOKEN"])
	assert.Equal(t, "api-456", secrets["API_KEY"])
	store.AssertExpectations(t)
}
