package secret

import "github.com/stretchr/testify/mock"

// MockStore is a mock implementation of Store for testing.
type MockStore struct {
	mock.Mock
}

// Set mocks the Set method.
func (m *MockStore) Set(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

// Get mocks the Get method.
func (m *MockStore) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

// Delete mocks the Delete method.
func (m *MockStore) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

// List mocks the List method.
func (m *MockStore) List() ([]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
