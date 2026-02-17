package secret

import (
	"errors"
	"fmt"
)

// builtInKeys are the tokens that are always resolved from the secret store.
// They are treated identically to custom keys — they must exist in the store.
var builtInKeys = []string{
	"GITHUB_TOKEN",
	"CLAUDE_CODE_OAUTH_TOKEN",
}

// Resolve resolves all required secrets from the store: built-in tokens plus any
// custom keys. All keys must exist in the store — there is no environment variable
// fallback. Returns a map of key→value for all resolved secrets.
func Resolve(store Store, customKeys []string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Resolve built-in keys
	for _, key := range builtInKeys {
		val, err := store.Get(key)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, fmt.Errorf("secret %q not found in store — run: spinner secret set %s", key, key)
			}

			return nil, fmt.Errorf("reading secret %q: %w", key, err)
		}

		secrets[key] = val
	}

	// Resolve custom keys
	for _, key := range customKeys {
		// Skip if already resolved as a built-in key
		if _, ok := secrets[key]; ok {
			continue
		}

		val, err := store.Get(key)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, fmt.Errorf("secret %q not found in store — run: spinner secret set %s", key, key)
			}

			return nil, fmt.Errorf("reading secret %q: %w", key, err)
		}

		secrets[key] = val
	}

	return secrets, nil
}
