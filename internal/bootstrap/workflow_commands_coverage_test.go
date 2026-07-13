package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

func TestWorkflowCommandsDryRunHappyPaths(t *testing.T) {
	testCases := []struct {
		name     string
		current  string
		messages []string
		args     []string
	}{
		{
			name:    "ticket start",
			current: "feature/ABC-123-add-export",
			args: []string{
				"workflow", "ticket", "start",
				"--family", "feature",
				"--key", "ABC",
				"--ticket", "123",
				"--slug", "add-export",
			},
		},
		{
			name:     "ticket publish",
			current:  "feature/ABC-123-add-export",
			messages: []string{"feat(ABC-123): add export"},
			args:     []string{"workflow", "ticket", "publish", "--branch", "feature/ABC-123-add-export"},
		},
		{
			name:    "hotfix start",
			current: "hotfix/ABC-999-payment-timeout",
			args: []string{
				"workflow", "hotfix", "start",
				"--key", "ABC",
				"--ticket", "999",
				"--slug", "payment-timeout",
				"--affected-line", "main",
			},
		},
		{
			name:     "hotfix publish",
			current:  "hotfix/ABC-999-payment-timeout",
			messages: []string{"fix(ABC-999): resolve payment timeout"},
			args: []string{
				"workflow", "hotfix", "publish",
				"--branch", "hotfix/ABC-999-payment-timeout",
				"--affected-line", "main",
			},
		},
		{
			name:    "hotfix propagation",
			current: "hotfix/ABC-999-payment-timeout",
			args: []string{
				"workflow", "hotfix", "propagate",
				"--source", "hotfix/ABC-999-payment-timeout",
				"--target-line", "develop",
				"--commit", strings.Repeat("a", 40),
				"--slug", "forward-port-payment-timeout",
			},
		},
		{
			name:    "scratch cleanup",
			current: "scratch/ABC-123-export-exploration",
			args:    []string{"workflow", "cleanup", "--branch", "scratch/ABC-123-export-exploration"},
		},
		{
			name:    "release cut",
			current: "feature/ABC-123-add-export",
			args:    []string{"workflow", "release", "cut", "--version", "2.8.0"},
		},
		{
			name:    "release stabilization",
			current: "fix/ABC-999-release-blocker",
			args: []string{
				"workflow", "release", "stabilize",
				"--release", "release/2.8.0",
				"--kind", "blocker",
				"--key", "ABC",
				"--ticket", "999",
				"--slug", "release-blocker",
			},
		},
		{
			name:     "release stabilization publish",
			current:  "fix/ABC-999-release-blocker",
			messages: []string{"fix(ABC-999): resolve release blocker"},
			args: []string{
				"workflow", "release", "publish-stabilization",
				"--branch", "fix/ABC-999-release-blocker",
				"--release", "release/2.8.0",
			},
		},
		{
			name:    "release promotion",
			current: "feature/ABC-123-add-export",
			args:    []string{"workflow", "release", "promote", "--release", "release/2.8.0"},
		},
		{
			name:    "release backmerge",
			current: "feature/ABC-123-add-export",
			args:    []string{"workflow", "release", "backmerge", "--release", "release/2.8.0"},
		},
		{
			name:    "support preparation",
			current: "feature/ABC-123-add-export",
			args:    []string{"workflow", "release", "support", "--version", "2.8"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			git := newCommandGit(t, testCase.current, testCase.messages)
			command := NewWithRuntime(BuildInfo{Version: "test"}, commandRuntime(git))
			args := append(
				[]string{"--interactive", "never", "--output", "json", "--yes", "--dry-run"},
				testCase.args...,
			)

			output, err := executeBootstrapCommand(t, command, args...)
			if err != nil {
				t.Fatalf("command %q error = %v; output=%q", testCase.name, err, output)
			}
			if !strings.Contains(output, `"ok":true`) {
				t.Fatalf("command %q output is not a successful JSON result: %q", testCase.name, output)
			}
		})
	}
}

func TestInteractiveTicketStartReportsRemoteRefresh(t *testing.T) {
	git := newBranchCommandGit(t, "feature/ABC-123-add-export")
	application := newBranchCommandApplication(git, nil, &commandHelperPrompt{}, "human")
	application.options.yes = true

	stdout, stderr, err := executeBranchCommand(
		t,
		newTicketStartCommand(application),
		context.Background(),
		"--family", "feature",
		"--key", "ABC",
		"--ticket", "123",
		"--slug", "add-export",
		"--scratch",
	)
	if err != nil {
		t.Fatalf("interactive ticket start error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("interactive ticket start stderr = %q", stderr)
	}
	for _, expected := range []string{
		"🟢 Remote references fetched and stale references pruned from origin before this operation.",
		"Ticket workflow start completed.",
		"officialBranch: feature/ABC-123-add-export",
		"scratchBranch: scratch/ABC-123-add-export-exploration",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("interactive ticket start output missing %q: %q", expected, stdout)
		}
	}
	if git.fetchCalls != 1 {
		t.Fatalf("interactive ticket start fetch calls = %d, want 1", git.fetchCalls)
	}
}

func TestInteractiveTicketPublishFromScratchConfirmsSquashTransfer(t *testing.T) {
	source := "scratch/ABC-123-export-exploration"
	target, err := branch.ParseName("feature/ABC-123-add-export")
	if err != nil {
		t.Fatal(err)
	}
	git := newBranchCommandGit(t, source)
	git.messages = []string{"feat(ABC-123): add export"}
	git.officialBranches = []branch.BranchName{target}
	git.localBranches = map[string]bool{
		source:          true,
		target.String(): true,
	}
	prompt := &commandHelperPrompt{
		confirms: []commandHelperConfirmReply{{value: true}},
	}
	application := newBranchCommandApplication(git, nil, prompt, "human")

	stdout, stderr, err := executeBranchCommand(
		t,
		newTicketPublishCommand(application),
		context.Background(),
		"--message", "feat(ABC-123): add export",
	)
	if err != nil {
		t.Fatalf("scratch ticket publish error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("scratch ticket publish stderr = %q", stderr)
	}
	if len(prompt.confirmRequests) != 1 ||
		prompt.confirmRequests[0].Label != "Publish ticket workflow from scratch" ||
		!strings.Contains(prompt.confirmRequests[0].Description, source) ||
		!strings.Contains(prompt.confirmRequests[0].Description, target.String()) ||
		!strings.Contains(prompt.confirmRequests[0].Description, "Squash-merge") {
		t.Fatalf("scratch publish confirmation = %#v", prompt.confirmRequests)
	}
	for _, expected := range []string{
		"Ticket publish workflow completed.",
		"branch: " + target.String(),
		"scratchBranch: " + source,
		"squashMerged: true",
	} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("scratch ticket publish output missing %q: %q", expected, stdout)
		}
	}
	if len(git.squashedBranches) != 1 || git.squashedBranches[0].String() != source ||
		len(git.committedMessages) != 1 || git.committedMessages[0].Header().String() != "feat(ABC-123): add export" {
		t.Fatalf(
			"scratch ticket publish calls = squashed:%#v committed:%#v",
			git.squashedBranches,
			git.committedMessages,
		)
	}
}

func TestTicketPublishScratchInputContracts(t *testing.T) {
	source := "scratch/ABC-123-export-exploration"
	target, err := branch.ParseName("feature/ABC-123-add-export")
	if err != nil {
		t.Fatal(err)
	}
	newGit := func() *branchCommandGit {
		git := newBranchCommandGit(t, source)
		git.messages = []string{"feat(ABC-123): add export"}
		git.officialBranches = []branch.BranchName{target}
		git.localBranches = map[string]bool{
			source:          true,
			target.String(): true,
		}
		return git
	}

	t.Run("accepts an explicit target in a dry-run", func(t *testing.T) {
		git := newGit()
		application := newBranchCommandApplication(git, nil, nil, "json")
		application.options.yes = true
		application.options.dryRun = true
		stdout, _, err := executeBranchCommand(
			t,
			newTicketPublishCommand(application),
			context.Background(),
			"--target", target.String(),
			"--message", "feat(ABC-123): add export",
		)
		if err != nil {
			t.Fatalf("explicit scratch target dry-run error = %v", err)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, stdout, "workflow.ticket.publish"))
		if fields["branch"] != target.String() ||
			fields["scratchBranch"] != source ||
			fields["squashMerged"] != "false" {
			t.Fatalf("explicit scratch target fields = %#v", fields)
		}
	})

	t.Run("rejects invalid scratch inputs before confirmation", func(t *testing.T) {
		git := newGit()
		_, _, err := executeBranchCommand(
			t,
			newTicketPublishCommand(newBranchCommandApplication(git, nil, nil, "human")),
			context.Background(),
			"--target", "not-a-branch",
		)
		assertProblemCode(t, err, problem.CodeBranchNameInvalid)

		git = newGit()
		git.localBranches[source] = false
		_, _, err = executeBranchCommand(
			t,
			newTicketPublishCommand(newBranchCommandApplication(git, nil, nil, "human")),
			context.Background(),
			"--message", "feat(ABC-123): add export",
		)
		assertProblemCode(t, err, problem.CodeScratchSourceBranchMissing)

		git = newGit()
		_, _, err = executeBranchCommand(
			t,
			newTicketPublishCommand(newBranchCommandApplication(git, nil, nil, "human")),
			context.Background(),
			"--message", "not a Conventional Commit",
		)
		assertProblemCode(t, err, problem.CodeCommitHeaderInvalid)
	})

	t.Run("rejects scratch-only options for official branches", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		_, _, err := executeBranchCommand(
			t,
			newTicketPublishCommand(newBranchCommandApplication(git, nil, nil, "human")),
			context.Background(),
			"--message", "feat(ABC-123): add export",
		)
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})
}
