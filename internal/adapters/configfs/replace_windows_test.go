//go:build windows

package configfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWindowsConfigurationReplacementRecovery(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	backup := path + ".bak"
	if err := os.WriteFile(backup, []byte("recovered"), defaultFileMode); err != nil {
		t.Fatal(err)
	}
	if err := recoverConfiguration(path); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, path, "recovered")
	assertNotExists(t, backup)

	if err := os.WriteFile(path, []byte("current"), defaultFileMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup, []byte("stale"), defaultFileMode); err != nil {
		t.Fatal(err)
	}
	if err := recoverConfiguration(path); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, path, "current")
	assertNotExists(t, backup)

	temporary, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := temporary.WriteString("replacement"); err != nil {
		t.Fatal(err)
	}
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	if err := replaceConfiguration(path, temporary.Name()); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, path, "replacement")
	assertNotExists(t, backup)
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()

	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != expected {
		t.Fatalf("content at %s = %q, want %q", path, actual, expected)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s still exists or could not be inspected: %v", path, err)
	}
}
