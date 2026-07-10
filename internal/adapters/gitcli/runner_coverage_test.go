package gitcli

import (
	"context"
	"strings"
	"testing"
)

func TestExecRunnerCoverageFailureAndTruncationPaths(t *testing.T) {
	t.Run("captures a started command failure with its exit code", func(t *testing.T) {
		result := (execRunner{binary: "go", maxOutputBytes: 1 << 10}).run(
			context.Background(),
			"",
			nil,
			"tool",
			"definitely-not-a-go-tool",
		)
		if result.err == nil || result.exitCode <= 0 {
			t.Fatalf("run() = %#v, want a process failure with a positive exit code", result)
		}
	})

	t.Run("classifies a process that cannot be started", func(t *testing.T) {
		result := (execRunner{binary: "git-governance-definitely-missing-executable"}).run(
			context.Background(),
			"",
			nil,
			"version",
		)
		if result.err == nil || result.exitCode != -1 {
			t.Fatalf("run() = %#v, want an unstarted-process failure", result)
		}
	})

	t.Run("truncates bounded command output without losing the process result", func(t *testing.T) {
		result := (execRunner{binary: "go", maxOutputBytes: 1}).run(
			context.Background(),
			"",
			nil,
			"version",
		)
		if result.err != nil || result.exitCode != 0 || !result.truncated {
			t.Fatalf("run() = %#v, want successful truncated output", result)
		}
		if len(result.stdout) != 1 || strings.Contains(result.stdout, "\n") {
			t.Fatalf("truncated stdout = %q", result.stdout)
		}
	})
}

func TestNewBoundedBufferUsesDefaultLimitForNonPositiveInput(t *testing.T) {
	buffer := newBoundedBuffer(0)
	if buffer.limit != defaultMaxOutputBytes {
		t.Fatalf("buffer limit = %d, want %d", buffer.limit, defaultMaxOutputBytes)
	}
}
