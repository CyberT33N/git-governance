package bootstrap

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestAppCoverageBuildInfoErrorRenderingAndCompletionFailure(t *testing.T) {
	t.Run("uses a development version when build metadata is absent", func(t *testing.T) {
		command := NewWithRuntime(BuildInfo{}, commandRuntime(newCommandGit(t, "feature/ABC-123-add-export", nil)))
		if command.Version != "devel" {
			t.Fatalf("command version = %q, want devel", command.Version)
		}
	})

	t.Run("renders unclassified human errors as internal failures", func(t *testing.T) {
		command := NewWithRuntime(BuildInfo{Version: "test"}, commandRuntime(newCommandGit(t, "feature/ABC-123-add-export", nil)))
		stderr := &bytes.Buffer{}
		command.SetErr(stderr)
		if err := command.PersistentFlags().Set("color", "always"); err != nil {
			t.Fatal(err)
		}

		RenderError(command, errors.New("unexpected implementation failure"))
		output := stderr.String()
		if !strings.Contains(output, "Error [INTERNAL]") || !strings.Contains(output, "\x1b[") {
			t.Fatalf("RenderError() output = %q", output)
		}
	})

	t.Run("rejects unsupported completion shells", func(t *testing.T) {
		command := NewWithRuntime(BuildInfo{Version: "test"}, commandRuntime(newCommandGit(t, "feature/ABC-123-add-export", nil)))
		_, err := executeBootstrapCommand(t, command, "completion", "unsupported")
		if err == nil || !strings.Contains(err.Error(), "unsupported shell") {
			t.Fatalf("completion error = %v", err)
		}
	})

	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		shell := shell
		t.Run("generates "+shell+" completion", func(t *testing.T) {
			command := NewWithRuntime(BuildInfo{Version: "test"}, commandRuntime(newCommandGit(t, "feature/ABC-123-add-export", nil)))
			output, err := executeBootstrapCommand(t, command, "completion", shell)
			if err != nil || output == "" {
				t.Fatalf("completion %q = (%q, %v)", shell, output, err)
			}
		})
	}
}
