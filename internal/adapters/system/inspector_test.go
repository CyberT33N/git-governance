package system

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestVersionUsesResolvedExecutableAndFirstOutputLine(t *testing.T) {
	t.Parallel()

	var (
		resolved  string
		arguments []string
	)
	inspector := New(Options{
		Timeout: time.Second,
		LookPath: func(executable string) (string, error) {
			if executable != "lefthook" {
				t.Fatalf("LookPath(%q)", executable)
			}
			return "C:/tools/lefthook.exe", nil
		},
		Run: func(_ context.Context, executable string, values ...string) ([]byte, error) {
			resolved = executable
			arguments = append([]string(nil), values...)
			return []byte("lefthook version 2.1.8\nextra output"), nil
		},
	})

	version, err := inspector.Version(context.Background(), "lefthook")
	if err != nil {
		t.Fatal(err)
	}
	if version != "lefthook version 2.1.8" || resolved != "C:/tools/lefthook.exe" || len(arguments) != 1 || arguments[0] != "--version" {
		t.Fatalf("Version() = (%q, %q, %v)", version, resolved, arguments)
	}
	operatingSystem, architecture := inspector.Platform()
	if operatingSystem == "" || architecture == "" {
		t.Fatalf("Platform() = (%q, %q)", operatingSystem, architecture)
	}
}

func TestFileExistsAndFailures(t *testing.T) {
	t.Parallel()

	t.Run("exists", func(t *testing.T) {
		inspector := New(Options{
			Stat: func(string) (os.FileInfo, error) {
				return fakeFileInfo{}, nil
			},
		})
		exists, err := inspector.FileExists("lefthook.yml")
		if err != nil || !exists {
			t.Fatalf("FileExists() = (%t, %v)", exists, err)
		}
	})

	t.Run("missing", func(t *testing.T) {
		inspector := New(Options{
			Stat: func(string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
		})
		exists, err := inspector.FileExists("missing")
		if err != nil || exists {
			t.Fatalf("FileExists() = (%t, %v)", exists, err)
		}
	})

	t.Run("version failure", func(t *testing.T) {
		inspector := New(Options{
			LookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
		})
		if _, err := inspector.Version(context.Background(), "lefthook"); err == nil {
			t.Fatal("Version() unexpectedly succeeded")
		}
	})
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "test" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }
