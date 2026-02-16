package secret

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func passphraseFunc(p string) PassphraseFunc {
	return func() (string, error) { return p, nil }
}

func TestRoundTripSetGet(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	if err := store.Set("MY_TOKEN", "secret-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := store.Get("MY_TOKEN")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "secret-value" {
		t.Errorf("got %q, want %q", val, "secret-value")
	}
}

func TestMultipleKeys(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	secrets := map[string]string{
		"KEY_A": "value-a",
		"KEY_B": "value-b",
		"KEY_C": "value-c",
	}

	for k, v := range secrets {
		if err := store.Set(k, v); err != nil {
			t.Fatalf("Set(%s): %v", k, err)
		}
	}

	for k, want := range secrets {
		got, err := store.Get(k)
		if err != nil {
			t.Fatalf("Get(%s): %v", k, err)
		}
		if got != want {
			t.Errorf("Get(%s) = %q, want %q", k, got, want)
		}
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	if err := store.Set("TO_DELETE", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := store.Delete("TO_DELETE"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Get("TO_DELETE")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete: got err %v, want ErrNotFound", err)
	}
}

func TestDeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	err := store.Delete("NONEXISTENT")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete nonexistent: got err %v, want ErrNotFound", err)
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	if err := store.Set("BETA", "b"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Set("ALPHA", "a"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	keys, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("List returned %d keys, want 2", len(keys))
	}
	// List should return sorted keys
	if keys[0] != "ALPHA" || keys[1] != "BETA" {
		t.Errorf("List = %v, want [ALPHA BETA]", keys)
	}
}

func TestWrongPassphrase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	store := NewEncryptedFileStore(path, passphraseFunc("correct-pass"))

	if err := store.Set("KEY", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	wrongStore := NewEncryptedFileStore(path, passphraseFunc("wrong-pass"))
	_, err := wrongStore.Get("KEY")
	if err == nil {
		t.Fatal("Get with wrong passphrase should fail")
	}
}

func TestCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	if err := os.WriteFile(path, []byte("corrupted-data-that-is-long-enough"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	store := NewEncryptedFileStore(path, passphraseFunc("test-pass"))
	_, err := store.Get("KEY")
	if err == nil {
		t.Fatal("Get on corrupted file should fail")
	}
}

func TestMissingFileReturnsEmptyStore(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "nonexistent.enc"), passphraseFunc("test-pass"))

	keys, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List on missing file returned %d keys, want 0", len(keys))
	}

	_, err = store.Get("MISSING")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get on missing file: got %v, want ErrNotFound", err)
	}
}

func TestAtomicWriteSafety(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")
	store := NewEncryptedFileStore(path, passphraseFunc("test-pass"))

	// Write initial value
	if err := store.Set("KEY", "initial"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Verify no temp file remains
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); !errors.Is(err, os.ErrNotExist) {
		t.Error("temp file should not exist after successful write")
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions: got %o, want 0600", perm)
	}
}

func TestCustomStorePathViaEnv(t *testing.T) {
	dir := t.TempDir()
	customPath := filepath.Join(dir, "custom", "store.enc")

	t.Setenv(envStorePath, customPath)

	got := DefaultStorePath()
	if got != customPath {
		t.Errorf("DefaultStorePath() = %q, want %q", got, customPath)
	}

	// Verify the store can be created and used at the custom path
	store := NewEncryptedFileStore("", passphraseFunc("test-pass"))
	if err := store.Set("KEY", "value"); err != nil {
		t.Fatalf("Set at custom path: %v", err)
	}

	val, err := store.Get("KEY")
	if err != nil {
		t.Fatalf("Get at custom path: %v", err)
	}
	if val != "value" {
		t.Errorf("Get = %q, want %q", val, "value")
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	// Set one key so the file exists
	if err := store.Set("EXISTS", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	_, err := store.Get("MISSING")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get missing key: got %v, want ErrNotFound", err)
	}
}

func TestOverwriteExistingKey(t *testing.T) {
	dir := t.TempDir()
	store := NewEncryptedFileStore(filepath.Join(dir, "secrets.enc"), passphraseFunc("test-pass"))

	if err := store.Set("KEY", "original"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Set("KEY", "updated"); err != nil {
		t.Fatalf("Set overwrite: %v", err)
	}

	val, err := store.Get("KEY")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "updated" {
		t.Errorf("Get = %q, want %q", val, "updated")
	}
}

func TestPassphraseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.enc")

	// First create a valid store
	store := NewEncryptedFileStore(path, passphraseFunc("pass"))
	if err := store.Set("KEY", "val"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Now try with a failing passphrase function
	errStore := NewEncryptedFileStore(path, func() (string, error) {
		return "", errors.New("passphrase unavailable")
	})
	_, err := errStore.Get("KEY")
	if err == nil {
		t.Fatal("Get with failing passphrase should error")
	}
}
