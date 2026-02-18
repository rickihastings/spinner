package secret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBlobRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	secrets := map[string]string{
		"GITHUB_TOKEN":            "ghp_abc123",
		"CLAUDE_CODE_OAUTH_TOKEN": "oauth_xyz",
		"CUSTOM_KEY":              "custom_value",
	}

	blob, err := EncryptBlob(secrets, "test-pass")
	if err != nil {
		t.Fatalf("EncryptBlob: %v", err)
	}

	if err := os.WriteFile(path, blob, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := DecryptBlob(path, "test-pass")
	if err != nil {
		t.Fatalf("DecryptBlob: %v", err)
	}

	for k, want := range secrets {
		if got[k] != want {
			t.Errorf("key %s: got %q, want %q", k, got[k], want)
		}
	}

	if len(got) != len(secrets) {
		t.Errorf("got %d keys, want %d", len(got), len(secrets))
	}
}

func TestBlobWrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	blob, err := EncryptBlob(map[string]string{"KEY": "val"}, "correct-pass")
	if err != nil {
		t.Fatalf("EncryptBlob: %v", err)
	}

	if err := os.WriteFile(path, blob, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = DecryptBlob(path, "wrong-pass")
	if err == nil {
		t.Fatal("DecryptBlob with wrong passphrase should fail")
	}
}

func TestBlobCorrupted(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	if err := os.WriteFile(path, []byte("corrupted-data-that-is-long-enough"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := DecryptBlob(path, "test-pass")
	if err == nil {
		t.Fatal("DecryptBlob on corrupted data should fail")
	}
}

func TestBlobCorruptedTooShort(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	if err := os.WriteFile(path, []byte("short"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := DecryptBlob(path, "test-pass")
	if err == nil {
		t.Fatal("DecryptBlob on too-short data should fail")
	}
}

func TestBlobEmptySecretsMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	blob, err := EncryptBlob(map[string]string{}, "test-pass")
	if err != nil {
		t.Fatalf("EncryptBlob: %v", err)
	}

	if err := os.WriteFile(path, blob, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := DecryptBlob(path, "test-pass")
	if err != nil {
		t.Fatalf("DecryptBlob: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %d keys, want 0", len(got))
	}
}

func TestBlobMissingFile(t *testing.T) {
	_, err := DecryptBlob("/nonexistent/path/secrets.enc", "test-pass")
	if err == nil {
		t.Fatal("DecryptBlob on missing file should fail")
	}
}

func TestBlobFreshSaltPerCall(t *testing.T) {
	secrets := map[string]string{"KEY": "value"}

	blob1, err := EncryptBlob(secrets, "test-pass")
	if err != nil {
		t.Fatalf("EncryptBlob (1): %v", err)
	}

	blob2, err := EncryptBlob(secrets, "test-pass")
	if err != nil {
		t.Fatalf("EncryptBlob (2): %v", err)
	}

	// Different salt means different ciphertext
	if string(blob1) == string(blob2) {
		t.Error("two encryptions of the same data should produce different blobs (fresh salt)")
	}
}

// --- Raw-key blob tests (EncryptBlobWithKey / DecryptBlobWithKeyFile) ---

func TestBlobWithKeyRoundTrip(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	keyPath := filepath.Join(dir, "secrets.key")

	secrets := map[string]string{
		"GITHUB_TOKEN":            "ghp_abc123",
		"CLAUDE_CODE_OAUTH_TOKEN": "oauth_xyz",
		"CUSTOM_KEY":              "custom_value",
	}

	key, blob, err := EncryptBlobWithKey(secrets)
	if err != nil {
		t.Fatalf("EncryptBlobWithKey: %v", err)
	}

	if len(key) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(key))
	}

	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		t.Fatalf("WriteFile blob: %v", err)
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatalf("WriteFile key: %v", err)
	}

	got, err := DecryptBlobWithKeyFile(blobPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptBlobWithKeyFile: %v", err)
	}

	for k, want := range secrets {
		if got[k] != want {
			t.Errorf("key %s: got %q, want %q", k, got[k], want)
		}
	}

	if len(got) != len(secrets) {
		t.Errorf("got %d keys, want %d", len(got), len(secrets))
	}
}

func TestBlobWithKeyWrongKey(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	wrongKeyPath := filepath.Join(dir, "wrong.key")

	_, blob, err := EncryptBlobWithKey(map[string]string{"KEY": "val"})
	if err != nil {
		t.Fatalf("EncryptBlobWithKey: %v", err)
	}

	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		t.Fatalf("WriteFile blob: %v", err)
	}

	// Write a different 32-byte key
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = 0xFF
	}

	if err := os.WriteFile(wrongKeyPath, wrongKey, 0600); err != nil {
		t.Fatalf("WriteFile wrong key: %v", err)
	}

	_, err = DecryptBlobWithKeyFile(blobPath, wrongKeyPath)
	if err == nil {
		t.Fatal("DecryptBlobWithKeyFile with wrong key should fail")
	}
}

func TestBlobWithKeyMissingKeyFile(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")

	if err := os.WriteFile(blobPath, []byte("dummy"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := DecryptBlobWithKeyFile(blobPath, "/nonexistent/secrets.key")
	if err == nil {
		t.Fatal("DecryptBlobWithKeyFile with missing key file should fail")
	}
}

func TestBlobWithKeyMissingBlobFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "secrets.key")

	if err := os.WriteFile(keyPath, make([]byte, 32), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := DecryptBlobWithKeyFile("/nonexistent/secrets.enc", keyPath)
	if err == nil {
		t.Fatal("DecryptBlobWithKeyFile with missing blob file should fail")
	}
}

func TestBlobWithKeyFreshKeyPerCall(t *testing.T) {
	secrets := map[string]string{"KEY": "value"}

	key1, blob1, err := EncryptBlobWithKey(secrets)
	if err != nil {
		t.Fatalf("EncryptBlobWithKey (1): %v", err)
	}

	key2, blob2, err := EncryptBlobWithKey(secrets)
	if err != nil {
		t.Fatalf("EncryptBlobWithKey (2): %v", err)
	}

	// Different key each call
	if string(key1) == string(key2) {
		t.Error("two calls should produce different keys")
	}

	// Different blob each call
	if string(blob1) == string(blob2) {
		t.Error("two encryptions should produce different blobs")
	}
}

func TestBlobWithKeyInvalidKeyLength(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	keyPath := filepath.Join(dir, "secrets.key")

	if err := os.WriteFile(blobPath, []byte("some-blob-data-long-enough-for-nonce"), 0600); err != nil {
		t.Fatalf("WriteFile blob: %v", err)
	}

	// Write a key that's too short
	if err := os.WriteFile(keyPath, []byte("short-key"), 0600); err != nil {
		t.Fatalf("WriteFile key: %v", err)
	}

	_, err := DecryptBlobWithKeyFile(blobPath, keyPath)
	if err == nil {
		t.Fatal("DecryptBlobWithKeyFile with invalid key length should fail")
	}
}

func TestBlobWithKeyEmptySecretsMap(t *testing.T) {
	dir := t.TempDir()
	blobPath := filepath.Join(dir, "secrets.enc")
	keyPath := filepath.Join(dir, "secrets.key")

	key, blob, err := EncryptBlobWithKey(map[string]string{})
	if err != nil {
		t.Fatalf("EncryptBlobWithKey: %v", err)
	}

	if err := os.WriteFile(blobPath, blob, 0600); err != nil {
		t.Fatalf("WriteFile blob: %v", err)
	}

	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatalf("WriteFile key: %v", err)
	}

	got, err := DecryptBlobWithKeyFile(blobPath, keyPath)
	if err != nil {
		t.Fatalf("DecryptBlobWithKeyFile: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("got %d keys, want 0", len(got))
	}
}
