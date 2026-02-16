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
