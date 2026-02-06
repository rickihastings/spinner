package provider

import (
	"fmt"
	"sort"
	"strings"
)

// BackendConstructor creates a Provider for a specific backend.
type BackendConstructor func() (Provider, error)

// Factory holds registered backend constructors and creates providers by name.
type Factory struct {
	backends map[string]BackendConstructor
}

// NewFactory creates an empty Factory.
func NewFactory() *Factory {
	return &Factory{backends: make(map[string]BackendConstructor)}
}

// Register adds a backend constructor under the given name.
func (f *Factory) Register(name string, ctor BackendConstructor) {
	f.backends[name] = ctor
}

// Create builds a Provider for the named backend. Returns an error if
// the backend is not registered.
func (f *Factory) Create(name string) (Provider, error) {
	ctor, ok := f.backends[name]
	if !ok {
		return nil, fmt.Errorf("unknown backend: %q (available: %s)",
			name, strings.Join(f.Available(), ", "))
	}

	return ctor()
}

// Available returns the sorted list of registered backend names.
func (f *Factory) Available() []string {
	names := make([]string, 0, len(f.backends))
	for name := range f.backends {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
