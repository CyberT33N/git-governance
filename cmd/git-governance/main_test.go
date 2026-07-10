package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewCommandUsesBuildMetadata(t *testing.T) {
	previousVersion, previousCommit, previousDate := version, commit, date
	t.Cleanup(func() {
		version, commit, date = previousVersion, previousCommit, previousDate
	})
	version, commit, date = "1.2.3", "abc123", "2026-07-10T12:00:00Z"

	command := newCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)
	command.SetArgs([]string{"--version"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"git-governance 1.2.3", "commit: abc123", "built: 2026-07-10T12:00:00Z"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("version output missing %q: %q", expected, output.String())
		}
	}
}
