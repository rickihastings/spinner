package secret

import (
	"errors"
	"fmt"
	"strings"
)

// builtInEntry represents a required secret. For OR-groups (multiple alternatives),
// the keys are tried in order and the first one found wins. At least one must be present.
type builtInEntry struct {
	alternatives []string
}

// builtInEntries are the tokens that are always resolved from the secret store.
// Single-key entries are required. OR-group entries require at least one alternative.
var builtInEntries = []builtInEntry{
	{alternatives: []string{"GITHUB_TOKEN"}},
	{alternatives: []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"}},
}

// Resolve resolves all required secrets from the store: built-in tokens plus any
// custom keys. All keys must exist in the store — there is no environment variable
// fallback. Returns a map of key→value for all resolved secrets.
func Resolve(store Store, customKeys []string) (map[string]string, error) {
	secrets := make(map[string]string)

	// Resolve built-in entries
	for _, entry := range builtInEntries {
		if err := resolveEntry(store, entry, secrets); err != nil {
			return nil, err
		}
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

// resolveEntry resolves a single built-in entry. For OR-groups, tries each alternative
// in order and stores the first one found. Returns an error if none are found or a
// non-ErrNotFound error occurs.
func resolveEntry(store Store, entry builtInEntry, secrets map[string]string) error {
	for _, key := range entry.alternatives {
		val, err := store.Get(key)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}

			return fmt.Errorf("reading secret %q: %w", key, err)
		}

		secrets[key] = val

		return nil
	}

	// None of the alternatives were found
	if len(entry.alternatives) == 1 {
		key := entry.alternatives[0]

		return fmt.Errorf("secret %q not found in store — run: spinner secret set %s", key, key)
	}

	// OR-group: build a helpful error listing all alternatives
	var hints []string
	for _, key := range entry.alternatives {
		hints = append(hints, fmt.Sprintf("  spinner secret set %s", key))
	}

	return fmt.Errorf("claude auth secret not found — set one with:\n%s", strings.Join(hints, "\nor:\n"))
}
