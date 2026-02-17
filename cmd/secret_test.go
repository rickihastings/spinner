package cmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/rickihastings/spinner/internal/secret"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// testStoreFactory returns a storeFactory that always returns the given mock.
func testStoreFactory(m *secret.MockStore) storeFactory {
	return func() secret.Store {
		return m
	}
}

func TestSecretSetCommand_WithValueFlag(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "MY_TOKEN", "supersecret").Return(nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", "supersecret"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "MY_TOKEN" stored`)
	mockStore.AssertExpectations(t)
}

func TestSecretSetCommand_EmptyValue(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", ""})

	// Provide empty input for the prompt path too
	// Since --value is "" the command will try to prompt; mock readPassword
	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte(""), nil
	}

	defer func() { readPassword = origReadPassword }()

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret value must not be empty")
	mockStore.AssertNotCalled(t, "Set", mock.Anything, mock.Anything)
}

func TestSecretSetCommand_WithPromptedValue(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "API_KEY", "prompted-value").Return(nil)

	origReadPassword := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("prompted-value"), nil
	}

	defer func() { readPassword = origReadPassword }()

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "API_KEY"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "API_KEY" stored`)
	mockStore.AssertExpectations(t)
}

func TestSecretSetCommand_StoreError(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Set", "MY_TOKEN", "val").Return(fmt.Errorf("disk full"))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set", "MY_TOKEN", "--value", "val"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestSecretSetCommand_MissingKeyArg(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"set"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}

func TestSecretListCommand_WithKeys(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return([]string{"API_KEY", "GITHUB_TOKEN"}, nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "API_KEY")
	assert.Contains(t, buf.String(), "GITHUB_TOKEN")
	mockStore.AssertExpectations(t)
}

func TestSecretListCommand_Empty(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return([]string{}, nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No secrets stored")
	mockStore.AssertExpectations(t)
}

func TestSecretListCommand_StoreError(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("List").Return(nil, fmt.Errorf("corrupt store"))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"list"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "corrupt store")
}

func TestSecretDeleteCommand_Success(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Delete", "OLD_TOKEN").Return(nil)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "OLD_TOKEN"})

	err := cmd.Execute()

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), `Secret "OLD_TOKEN" deleted`)
	mockStore.AssertExpectations(t)
}

func TestSecretDeleteCommand_NotFound(t *testing.T) {
	mockStore := new(secret.MockStore)
	mockStore.On("Delete", "NONEXISTENT").Return(fmt.Errorf("%w: NONEXISTENT", secret.ErrNotFound))

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete", "NONEXISTENT"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestSecretDeleteCommand_MissingKeyArg(t *testing.T) {
	mockStore := new(secret.MockStore)

	cmd := newSecretCommand(testStoreFactory(mockStore))
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"delete"})

	err := cmd.Execute()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "accepts 1 arg(s)")
}
