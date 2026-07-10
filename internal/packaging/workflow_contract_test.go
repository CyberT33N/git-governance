package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTagApprovalCreatesExactlyOneReleaseTrigger(t *testing.T) {
	t.Parallel()

	tagWorkflow := readWorkflow(t, "tag-approved-release.yml")
	for _, expected := range []string{
		"pull_request:",
		"branches:",
		"- main",
		"types:",
		"- closed",
		"startsWith(github.event.pull_request.head.ref, 'release/')",
		`git push origin "refs/tags/${TAG}"`,
	} {
		if !strings.Contains(tagWorkflow, expected) {
			t.Fatalf("tag workflow does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"Dispatch the artifact workflow",
		"actions: write",
		"actions/workflows/release.yml/dispatches",
	} {
		if strings.Contains(tagWorkflow, forbidden) {
			t.Fatalf("tag workflow must not contain %q; the tag push is the single release trigger", forbidden)
		}
	}

	releaseWorkflow := readWorkflow(t, "release.yml")
	for _, expected := range []string{
		"push:",
		"tags:",
		`- "v*"`,
		"workflow_dispatch:",
		`ref: ${{ github.event_name == 'workflow_dispatch' && inputs.tag || github.ref }}`,
	} {
		if !strings.Contains(releaseWorkflow, expected) {
			t.Fatalf("release workflow does not contain %q", expected)
		}
	}
}

func TestProtectedLineWorkflowKeepsSharedLineMutationInCI(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, "create-protected-line.yml")
	for _, expected := range []string{
		"workflow_dispatch:",
		"environment: release",
		"source=\"origin/develop\"",
		"source=\"origin/main\"",
		`git push origin "${SOURCE}:refs/heads/${TARGET}"`,
	} {
		if !strings.Contains(workflow, expected) {
			t.Fatalf("protected-line workflow does not contain %q", expected)
		}
	}
}

func readWorkflow(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(repositoryRoot(t), ".github", "workflows", name)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
