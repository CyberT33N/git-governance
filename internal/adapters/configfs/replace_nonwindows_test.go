//go:build !windows

package configfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNonWindowsConfigurationReplacement(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("current"), defaultFileMode); err != nil {
		t.Fatal(err)
	}
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
	if err := recoverConfiguration(path); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(actual) != "replacement" {
		t.Fatalf("replacement content = %q", actual)
	}
}
