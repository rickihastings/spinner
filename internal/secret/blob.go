package secret

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// EncryptBlob encrypts a map of secrets into a binary blob using AES-256-GCM
// with Argon2id key derivation. Each call generates a fresh salt and nonce.
// The blob format is identical to the store file format: salt + nonce + ciphertext.
func EncryptBlob(secrets map[string]string, passphrase string) ([]byte, error) {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return nil, fmt.Errorf("encoding secrets: %w", err)
	}

	return encrypt(plaintext, passphrase)
}

// DecryptBlob reads an encrypted blob from path and decrypts it into a map of secrets.
func DecryptBlob(path, passphrase string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("secrets blob not found: %s", path)
		}

		return nil, fmt.Errorf("reading secrets blob: %w", err)
	}

	plaintext, err := decrypt(data, passphrase)
	if err != nil {
		return nil, fmt.Errorf("decrypting secrets blob: %w", err)
	}

	var secrets map[string]string
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("parsing secrets blob: %w", err)
	}

	return secrets, nil
}
