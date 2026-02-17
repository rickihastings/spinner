package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/crypto/argon2"
)

const (
	saltLen       = 16
	nonceLen      = 12
	argonTime     = 1
	argonMemory   = 64 * 1024
	argonThreads  = 4
	argonKeyLen   = 32
	fileMode      = 0600
	envStorePath  = "SPINNER_SECRET_STORE"
	envPassphrase = "SPINNER_SECRET_PASSPHRASE"
)

// PassphraseFunc is a function that returns the passphrase for encrypting/decrypting secrets.
type PassphraseFunc func() (string, error)

// EncryptedFileStore implements Store using an AES-256-GCM encrypted file
// with Argon2id key derivation.
type EncryptedFileStore struct {
	path       string
	passphrase PassphraseFunc
}

// NewEncryptedFileStore creates a new EncryptedFileStore.
// If path is empty, it uses SPINNER_SECRET_STORE env var or defaults to ~/.spinner/secrets.enc.
func NewEncryptedFileStore(path string, passphrase PassphraseFunc) *EncryptedFileStore {
	if path == "" {
		path = DefaultStorePath()
	}

	return &EncryptedFileStore{
		path:       path,
		passphrase: passphrase,
	}
}

// DefaultStorePath returns the default store file path, checking SPINNER_SECRET_STORE
// env var first, then falling back to ~/.spinner/secrets.enc.
func DefaultStorePath() string {
	if p := os.Getenv(envStorePath); p != "" {
		return p
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".spinner", "secrets.enc")
	}

	return filepath.Join(home, ".spinner", "secrets.enc")
}

// Set stores a secret value under the given key.
func (s *EncryptedFileStore) Set(key, value string) error {
	secrets, err := s.load()
	if err != nil {
		return err
	}

	secrets[key] = value

	return s.save(secrets)
}

// Get retrieves the secret value for the given key.
func (s *EncryptedFileStore) Get(key string) (string, error) {
	secrets, err := s.load()
	if err != nil {
		return "", err
	}

	val, ok := secrets[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	return val, nil
}

// Delete removes the secret with the given key.
func (s *EncryptedFileStore) Delete(key string) error {
	secrets, err := s.load()
	if err != nil {
		return err
	}

	if _, ok := secrets[key]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, key)
	}

	delete(secrets, key)

	return s.save(secrets)
}

// List returns the names of all stored secret keys, sorted alphabetically.
func (s *EncryptedFileStore) List() ([]string, error) {
	secrets, err := s.load()
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(secrets))
	for k := range secrets {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys, nil
}

// load reads and decrypts the store file, returning the secrets map.
// Returns an empty map if the file does not exist.
func (s *EncryptedFileStore) load() (map[string]string, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]string), nil
		}

		return nil, fmt.Errorf("reading secret store: %w", err)
	}

	passphrase, err := s.passphrase()
	if err != nil {
		return nil, fmt.Errorf("getting passphrase: %w", err)
	}

	plaintext, err := decrypt(data, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypting secret store: %w", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("parsing secret store: %w", err)
	}

	return secrets, nil
}

// save encrypts and writes the secrets map to the store file atomically.
func (s *EncryptedFileStore) save(secrets map[string]string) error {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("encoding secrets: %w", err)
	}

	passphrase, err := s.passphrase()
	if err != nil {
		return fmt.Errorf("getting passphrase: %w", err)
	}

	ciphertext, err := encrypt(plaintext, passphrase)
	if err != nil {
		return fmt.Errorf("encrypting secret store: %w", err)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("creating secret store directory: %w", err)
	}

	// Atomic write: write to temp file, then rename.
	tmpFile := s.path + ".tmp"
	if err := os.WriteFile(tmpFile, ciphertext, fileMode); err != nil {
		return fmt.Errorf("writing secret store: %w", err)
	}

	if err := os.Rename(tmpFile, s.path); err != nil {
		_ = os.Remove(tmpFile) // best-effort cleanup
		return fmt.Errorf("saving secret store: %w", err)
	}

	return nil
}

// deriveKey uses Argon2id to derive an AES-256 key from a passphrase and salt.
func deriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey([]byte(passphrase), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
}

// encrypt encrypts plaintext using AES-256-GCM with Argon2id key derivation.
// Returns: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func encrypt(plaintext []byte, passphrase string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	key := deriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// salt + nonce + ciphertext
	result := make([]byte, 0, saltLen+nonceLen+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// decrypt decrypts data that was encrypted with encrypt().
// Expects: salt (16 bytes) + nonce (12 bytes) + ciphertext.
func decrypt(data []byte, passphrase string) ([]byte, error) {
	minLen := saltLen + nonceLen + 1 // at least 1 byte of ciphertext
	if len(data) < minLen {
		return nil, errors.New("corrupted secret store: data too short")
	}

	salt := data[:saltLen]
	nonce := data[saltLen : saltLen+nonceLen]
	ciphertext := data[saltLen+nonceLen:]

	key := deriveKey(passphrase, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong passphrase or corrupted data")
	}

	return plaintext, nil
}
