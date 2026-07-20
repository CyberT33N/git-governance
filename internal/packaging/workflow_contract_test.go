package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTagApprovalExplicitlyDispatchesReleaseArtifacts(t *testing.T) {
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
		"actions: write",
		"actions/workflows/release.yml/dispatches",
		`\"inputs\":{\"tag\":\"${TAG}\"}`,
	} {
		if !strings.Contains(tagWorkflow, expected) {
			t.Fatalf("tag workflow does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"repository_dispatch",
	} {
		if strings.Contains(tagWorkflow, forbidden) {
			t.Fatalf("tag workflow must not contain %q", forbidden)
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

func TestTagApprovalArtifactDispatchUsesJobToken(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, "tag-approved-release.yml")
	dispatchStart := strings.Index(workflow, "- name: Dispatch artifact workflow for immutable tag")
	if dispatchStart == -1 {
		t.Fatal("tag workflow does not contain the artifact dispatch step")
	}

	dispatchStep := workflow[dispatchStart:]
	for _, expected := range []string{
		"GITHUB_TOKEN: ${{ github.token }}",
		`--header "Authorization: Bearer ${GITHUB_TOKEN}"`,
	} {
		if !strings.Contains(dispatchStep, expected) {
			t.Fatalf("artifact dispatch step does not contain %q", expected)
		}
	}
	if strings.Contains(dispatchStep, "GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}") {
		t.Fatal("artifact dispatch step must use the job token, not a repository secret")
	}
}

func TestProtectedLineWorkflowKeepsSharedLineMutationInCI(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, "create-protected-line.yml")
	for _, expected := range []string{
		"workflow_dispatch:",
		"run-name: Create ${{ inputs.kind }} line ${{ inputs.version }} (${{ inputs.request_id || 'manual' }})",
		"request_id:",
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

func TestReleaseControlWorkflowUsesEphemeralBrokerIdentity(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, "release-control.yml")
	for _, expected := range []string{
		"workflow_dispatch:",
		"broker-smoke",
		"release-cut",
		"environment: release",
		"id-token: write",
		"google-github-actions/auth@7c6bc770dae815cd3e89ee6cdf493a5fab2cc093",
		"token_format: id_token",
		"id_token_audience: ${{ vars.GCP_BROKER_URL }}",
		"GIT_GOVERNANCE_GITHUB_CREDENTIAL_BROKER_URL",
		"GIT_GOVERNANCE_WORKLOAD_IDENTITY_TOKEN",
		`--dispatch`,
		`"repository":"git-governance"`,
		`"repository":"not-approved"`,
		`test "$approved_status" = "200"`,
		`test "$rejected_status" = "403"`,
		`rm -f "$response"`,
	} {
		if !strings.Contains(workflow, expected) {
			t.Fatalf("release-control workflow does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{
		"GITHUB_RELEASE_APP_ID",
		"GITHUB_RELEASE_APP_INSTALLATION_ID",
		"echo \"$BROKER_ID_TOKEN\"",
		"cat \"$response\"",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("release-control workflow must not contain %q", forbidden)
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
