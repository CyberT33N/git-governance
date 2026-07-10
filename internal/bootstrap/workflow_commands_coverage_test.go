package bootstrap

import (
	"strings"
	"testing"
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
