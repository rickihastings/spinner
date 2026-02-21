package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	// aesKeyLen is the length of the random AES-256 key for blob transport encryption.
	aesKeyLen = 32
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

// EncryptBlobWithKey encrypts a map of secrets using a random 32-byte AES-256-GCM key.
// No Argon2id key derivation — the random key is already full entropy.
// Returns the key and the encrypted blob (nonce + ciphertext).
func EncryptBlobWithKey(secrets map[string]string) (key []byte, blob []byte, err error) {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding secrets: %w", err)
	}

	key = make([]byte, aesKeyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, nil, fmt.Errorf("generating key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// blob = nonce + ciphertext
	blob = make([]byte, 0, len(nonce)+len(ciphertext))
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)

	return key, blob, nil
}

// DecryptBlobWithKeyFile reads an encrypted blob and key file, then decrypts
// the blob into a map of secrets. The key file must contain a raw 32-byte key.
// The blob format is: nonce (12 bytes) + ciphertext.
func DecryptBlobWithKeyFile(blobPath, keyPath string) (map[string]string, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("secrets key file not found: %s", keyPath)
		}

		return nil, fmt.Errorf("reading secrets key: %w", err)
	}

	data, err := os.ReadFile(blobPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("secrets blob not found: %s", blobPath)
		}

		return nil, fmt.Errorf("reading secrets blob: %w", err)
	}

	return decryptDataWithKey(data, key)
}

// decryptDataWithKey decrypts a blob (nonce+ciphertext format) using a raw AES-256 key.
func decryptDataWithKey(data, key []byte) (map[string]string, error) {
	if len(key) != aesKeyLen {
		return nil, fmt.Errorf("invalid key length: expected %d bytes, got %d", aesKeyLen, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	if len(data) < gcm.NonceSize()+1 {
		return nil, errors.New("corrupted secrets blob: data too short")
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: wrong key or corrupted data")
	}

	var secrets map[string]string
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, fmt.Errorf("parsing secrets blob: %w", err)
	}

	return secrets, nil
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
