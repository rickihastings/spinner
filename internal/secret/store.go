package secret

import "errors"

// ErrNotFound is returned when a secret key does not exist in the store.
var ErrNotFound = errors.New("secret not found")

// Store is the interface for secret storage backends.
type Store interface {
	// Set stores a secret value under the given key.
	Set(key, value string) error

	// Get retrieves the secret value for the given key.
	// Returns ErrNotFound if the key does not exist.
	Get(key string) (string, error)

	// Delete removes the secret with the given key.
	// Returns ErrNotFound if the key does not exist.
	Delete(key string) error

	// List returns the names of all stored secret keys.
	List() ([]string, error)
}
