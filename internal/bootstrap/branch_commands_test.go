package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
	"github.com/spf13/cobra"
)

func TestBranchCommandTreeAndListReportContracts(t *testing.T) {
	application := newBranchCommandApplication(
		newBranchCommandGit(t, "feature/ABC-123-add-export"),
		nil,
		nil,
		"json",
	)
	command := newBranchCommand(application)
	children := make(map[string]bool)
	for _, child := range command.Commands() {
		children[child.Name()] = true
	}
	for _, expected := range []string{"list", "validate", "create", "sync-base"} {
		if !children[expected] {
			t.Fatalf("branch command children = %#v, missing %q", children, expected)
		}
	}

	stdout, stderr, err := executeBranchCommand(t, newBranchListCommand(application), context.Background())
	if err != nil {
		t.Fatalf("branch list error = %v", err)
	}
	if stderr != "" {
		t.Fatalf("branch list stderr = %q", stderr)
	}
	result := assertSingleUtilityJSONResult(t, stdout, "branch.list")
	fields := utilityJSONFields(t, result)
	if !strings.Contains(fields["feature"], "feature/<ticket>-<slug>") {
		t.Fatalf("feature family report = %q", fields["feature"])
	}
	families, ok := result["data"].([]any)
	if !ok || len(families) != len(branchapp.ListFamilies()) {
		t.Fatalf("branch list data = %#v", result["data"])
	}

	application.options.output = "human"
	humanOutput, _, err := executeBranchCommand(t, newBranchListCommand(application), context.Background())
	if err != nil {
		t.Fatalf("human branch list error = %v", err)
	}
	if !strings.Contains(humanOutput, "Supported branch families:") ||
		!strings.Contains(humanOutput, "feature") {
		t.Fatalf("human branch list output = %q", humanOutput)
	}
}

func TestBranchValidateCommandContracts(t *testing.T) {
	t.Run("validates explicit and current branches with a JSON report", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "json")

		stdout, stderr, err := executeBranchCommand(
			t,
			newBranchValidateCommand(application),
			context.Background(),
			"--branch", "feature/ABC-123-add-export",
		)
		if err != nil {
			t.Fatalf("explicit branch validation error = %v", err)
		}
		if stderr != "" {
			t.Fatalf("explicit branch validation stderr = %q", stderr)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, stdout, "branch.validate"))
		if fields["branch"] != "feature/ABC-123-add-export" || fields["family"] != "feature" {
			t.Fatalf("explicit validation fields = %#v", fields)
		}
		if git.currentCalls != 0 {
			t.Fatalf("explicit branch validation called CurrentBranch %d times", git.currentCalls)
		}
		if got := branchNames(git.validatedBranches); strings.Join(got, ",") != "feature/ABC-123-add-export" {
			t.Fatalf("validated branches = %v", got)
		}

		manualGit := newBranchCommandGit(t, "feature/ABC-123-add-export")
		manualApplication := newBranchCommandApplication(manualGit, nil, nil, "human")
		manualOutput, _, err := executeBranchCommand(
			t,
			newBranchValidateCommand(manualApplication),
			context.Background(),
		)
		if err != nil {
			t.Fatalf("manual branch validation error = %v", err)
		}
		if manualGit.currentCalls != 1 {
			t.Fatalf("manual validation CurrentBranch calls = %d, want 1", manualGit.currentCalls)
		}
		if !strings.Contains(manualOutput, "Branch is valid.") {
			t.Fatalf("manual validation output = %q", manualOutput)
		}
	})

	discoverErr := errors.New("repository discovery failed")
	currentErr := errors.New("current branch failed")
	validateErr := errors.New("branch reference validation failed")
	for _, testCase := range []struct {
		name      string
		arguments []string
		configure func(*branchCommandGit)
		wantErr   error
	}{
		{
			name: "preserves discovery failures",
			configure: func(git *branchCommandGit) {
				git.discoverErr = discoverErr
			},
			wantErr: discoverErr,
		},
		{
			name: "preserves current branch failures",
			configure: func(git *branchCommandGit) {
				git.currentErr = currentErr
			},
			wantErr: currentErr,
		},
		{
			name:      "preserves validation service failures",
			arguments: []string{"--branch", "feature/ABC-123-add-export"},
			configure: func(git *branchCommandGit) {
				git.validateErr = validateErr
			},
			wantErr: validateErr,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			git := newBranchCommandGit(t, "feature/ABC-123-add-export")
			testCase.configure(git)
			_, _, err := executeBranchCommand(
				t,
				newBranchValidateCommand(newBranchCommandApplication(git, nil, nil, "human")),
				context.Background(),
				testCase.arguments...,
			)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("branch validation error = %v, want %v", err, testCase.wantErr)
			}
		})
	}

	t.Run("stops validation when the command context is cancelled", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := executeBranchCommand(
			t,
			newBranchValidateCommand(newBranchCommandApplication(git, nil, nil, "human")),
			ctx,
			"--branch", "feature/ABC-123-add-export",
		)
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled validation error = %v, want context cancellation", err)
		}
		if len(git.discoverContexts) != 1 || git.discoverContexts[0] != ctx {
			t.Fatalf("discover contexts = %#v, want cancelled command context", git.discoverContexts)
		}
	})
}

func TestBranchCreateCommandContracts(t *testing.T) {
	t.Run("parses root and creation flags then creates a regular branch", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		command := NewWithRuntime(BuildInfo{Version: "test"}, commandRuntime(git))

		output, err := executeBootstrapCommand(
			t,
			command,
			"--interactive", "never",
			"--output", "json",
			"--yes",
			"branch", "create",
			"--family", "feature",
			"--key", "ABC",
			"--ticket", "123",
			"--slug", "add-export",
			"--switch=false",
		)
		if err != nil {
			t.Fatalf("regular branch creation error = %v", err)
		}
		if len(git.createdBranches) != 1 {
			t.Fatalf("created branches = %#v", git.createdBranches)
		}
		created := git.createdBranches[0]
		if created.name.String() != "feature/ABC-123-add-export" ||
			created.base.String() != "origin/develop" ||
			created.switchTo {
			t.Fatalf("created branch = %#v", created)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, output, "branch.create"))
		if fields["branch"] != created.name.String() ||
			fields["base"] != "origin/develop" ||
			fields["switched"] != "false" ||
			fields["dryRun"] != "false" ||
			!strings.Contains(fields["plan"], "create: feature/ABC-123-add-export from origin/develop") {
			t.Fatalf("regular creation fields = %#v", fields)
		}
	})

	t.Run("generates scratch dry-run plans without mutating Git", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "json")
		application.options.dryRun = true

		stdout, stderr, err := executeBranchCommand(
			t,
			newBranchCreateCommand(application),
			context.Background(),
			"--family", "scratch",
			"--key", "ABC",
			"--ticket", "123",
			"--slug", "exploration",
			"--base", "feature/ABC-123-add-export",
			"--switch=false",
		)
		if err != nil {
			t.Fatalf("scratch dry-run error = %v", err)
		}
		if stderr != "" {
			t.Fatalf("scratch dry-run stderr = %q", stderr)
		}
		if git.fetchCalls != 0 || len(git.createdBranches) != 0 {
			t.Fatalf("scratch dry-run mutated Git: fetches=%d creates=%#v", git.fetchCalls, git.createdBranches)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, stdout, "branch.create"))
		if fields["branch"] != "scratch/ABC-123-exploration" ||
			fields["base"] != "feature/ABC-123-add-export" ||
			fields["switched"] != "false" ||
			fields["dryRun"] != "true" ||
			fields["plan"] != "fetch: git fetch --prune origin; create: scratch/ABC-123-exploration from feature/ABC-123-add-export" {
			t.Fatalf("scratch dry-run fields = %#v", fields)
		}
	})

	t.Run("requires noninteractive consent and honors a declined prompt", func(t *testing.T) {
		arguments := []string{
			"--family", "feature",
			"--key", "ABC",
			"--ticket", "123",
			"--slug", "add-export",
		}
		noninteractiveGit := newBranchCommandGit(t, "feature/ABC-123-add-export")
		_, _, err := executeBranchCommand(
			t,
			newBranchCreateCommand(newBranchCommandApplication(noninteractiveGit, nil, nil, "human")),
			context.Background(),
			arguments...,
		)
		assertProblemCode(t, err, problem.CodeInvalidInput)
		if len(noninteractiveGit.createdBranches) != 0 {
			t.Fatalf("noninteractive creation mutated Git: %#v", noninteractiveGit.createdBranches)
		}

		prompt := &commandHelperPrompt{
			confirms: []commandHelperConfirmReply{{value: false}},
		}
		declinedGit := newBranchCommandGit(t, "feature/ABC-123-add-export")
		_, _, err = executeBranchCommand(
			t,
			newBranchCreateCommand(newBranchCommandApplication(declinedGit, nil, prompt, "human")),
			context.Background(),
			arguments...,
		)
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if len(prompt.confirmRequests) != 1 ||
			prompt.confirmRequests[0].Label != "Create branch" ||
			!strings.Contains(prompt.confirmRequests[0].Description, "feature/ABC-123-add-export") {
			t.Fatalf("creation confirmation request = %#v", prompt.confirmRequests)
		}
		if len(declinedGit.createdBranches) != 0 {
			t.Fatalf("declined creation mutated Git: %#v", declinedGit.createdBranches)
		}
	})

	t.Run("preserves report writer failures after a successful creation", func(t *testing.T) {
		writeErr := errors.New("output unavailable")
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "human")
		application.options.yes = true
		command := newBranchCreateCommand(application)
		command.SetOut(commandHelperFailingWriter{err: writeErr})
		command.SetErr(io.Discard)
		command.SetArgs([]string{
			"--family", "feature",
			"--key", "ABC",
			"--ticket", "123",
			"--slug", "add-export",
		})

		err := command.ExecuteContext(context.Background())
		assertProblemCode(t, err, problem.CodeExternalCommandFailed)
		if !errors.Is(err, writeErr) {
			t.Fatalf("creation report error = %v, want %v", err, writeErr)
		}
		if len(git.createdBranches) != 1 {
			t.Fatalf("creation before report failure = %#v", git.createdBranches)
		}
	})

	discoverErr := errors.New("repository discovery failed")
	fetchErr := errors.New("fetch failed")
	for _, testCase := range []struct {
		name      string
		arguments []string
		configure func(*branchCommandGit, *application)
		wantCode  problem.Code
		wantErr   error
	}{
		{
			name: "preserves discovery failures",
			configure: func(git *branchCommandGit, _ *application) {
				git.discoverErr = discoverErr
			},
			wantErr: discoverErr,
		},
		{
			name:      "rejects unsupported branch families",
			arguments: []string{"--family", "unknown"},
			wantCode:  problem.CodeBranchFamilyInvalid,
		},
		{
			name:      "rejects invalid ticket keys",
			arguments: []string{"--family", "feature", "--key", "invalid"},
			wantCode:  problem.CodeTicketKeyInvalid,
		},
		{
			name:      "rejects invalid ticket numbers",
			arguments: []string{"--family", "feature", "--key", "ABC", "--ticket", "012"},
			wantCode:  problem.CodeTicketNumberInvalid,
		},
		{
			name:      "rejects invalid branch slugs",
			arguments: []string{"--family", "feature", "--key", "ABC", "--ticket", "123", "--slug", "Add Export"},
			wantCode:  problem.CodeBranchSlugInvalid,
		},
		{
			name:      "rejects bases from another remote",
			arguments: []string{"--family", "feature", "--key", "ABC", "--ticket", "123", "--slug", "add-export", "--base", "upstream/develop"},
			wantCode:  problem.CodeBranchBaseInvalid,
		},
		{
			name:      "rejects non-ticket families after parsing inputs",
			arguments: []string{"--family", "main", "--key", "ABC", "--ticket", "123", "--slug", "add-export"},
			wantCode:  problem.CodeBranchFamilyInvalid,
		},
		{
			name:      "rejects remote scratch bases",
			arguments: []string{"--family", "scratch", "--key", "ABC", "--ticket", "123", "--slug", "exploration", "--base", "origin/feature/ABC-123-add-export"},
			wantCode:  problem.CodeBranchBaseInvalid,
		},
		{
			name:      "enforces matching scratch ticket bases",
			arguments: []string{"--family", "scratch", "--key", "ABC", "--ticket", "123", "--slug", "exploration", "--base", "feature/XYZ-999-other-ticket"},
			wantCode:  problem.CodeBranchBaseInvalid,
		},
		{
			name: "preserves branch creation service failures",
			arguments: []string{
				"--family", "feature",
				"--key", "ABC",
				"--ticket", "123",
				"--slug", "add-export",
			},
			configure: func(git *branchCommandGit, _ *application) {
				git.fetchErr = fetchErr
			},
			wantErr: fetchErr,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			git := newBranchCommandGit(t, "feature/ABC-123-add-export")
			application := newBranchCommandApplication(git, nil, nil, "human")
			application.options.yes = true
			if testCase.configure != nil {
				testCase.configure(git, application)
			}

			_, _, err := executeBranchCommand(
				t,
				newBranchCreateCommand(application),
				context.Background(),
				testCase.arguments...,
			)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("branch creation error = %v, want %v", err, testCase.wantErr)
				}
			} else {
				assertProblemCode(t, err, testCase.wantCode)
			}
			if len(git.createdBranches) != 0 {
				t.Fatalf("failed creation mutated Git: %#v", git.createdBranches)
			}
		})
	}

	t.Run("stops creation when the command context is cancelled", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "human")
		application.options.yes = true
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := executeBranchCommand(
			t,
			newBranchCreateCommand(application),
			ctx,
			"--family", "feature",
			"--key", "ABC",
			"--ticket", "123",
			"--slug", "add-export",
		)
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled creation error = %v, want context cancellation", err)
		}
		if len(git.createdBranches) != 0 {
			t.Fatalf("cancelled creation mutated Git: %#v", git.createdBranches)
		}
	})
}

func TestBranchSyncBaseCommandContracts(t *testing.T) {
	t.Run("checks the current branch and reports JSON without quality fields", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "json")

		stdout, stderr, err := executeBranchCommand(
			t,
			newBranchSyncBaseCommand(application),
			context.Background(),
			"--base", "develop",
		)
		if err != nil {
			t.Fatalf("manual sync check error = %v", err)
		}
		if stderr != "" {
			t.Fatalf("manual sync check stderr = %q", stderr)
		}
		if git.currentCalls != 1 || git.fetchCalls != 1 || len(git.rebasedBases) != 0 || len(git.mergedBranches) != 0 {
			t.Fatalf(
				"sync check calls = current:%d fetch:%d rebase:%v merge:%v",
				git.currentCalls,
				git.fetchCalls,
				git.rebasedBases,
				git.mergedBranches,
			)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, stdout, "branch.sync-base"))
		if fields["branch"] != "feature/ABC-123-add-export" ||
			fields["base"] != "origin/develop" ||
			fields["publication"] != string(branch.PublicationUnpublished) ||
			fields["missingBaseCommits"] != "false" ||
			fields["mutated"] != "false" ||
			fields["recommendedAction"] != "none" {
			t.Fatalf("sync check fields = %#v", fields)
		}
		if _, found := fields["qualityStatus"]; found {
			t.Fatalf("non-mutating sync unexpectedly reported quality: %#v", fields)
		}
	})

	t.Run("merges a published branch and reports post-mutation quality", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		git.publication = branch.PublicationPublished
		git.missingBaseCommits = true
		quality := &branchCommandQuality{
			result: port.QualityResult{
				Status: port.QualityPassed,
				Detail: "all branch gates passed",
			},
		}
		application := newBranchCommandApplication(git, quality, nil, "json")
		application.options.yes = true

		stdout, _, err := executeBranchCommand(
			t,
			newBranchSyncBaseCommand(application),
			context.Background(),
			"--branch", "feature/ABC-123-add-export",
			"--base", "origin/develop",
			"--strategy", "merge",
			"--merge-message", "chore(ABC-123): merge origin/develop",
		)
		if err != nil {
			t.Fatalf("merge sync error = %v", err)
		}
		if len(git.mergedBranches) != 1 || git.mergedBranches[0].base.String() != "origin/develop" {
			t.Fatalf("merge calls = %#v", git.mergedBranches)
		}
		if quality.calls != 1 {
			t.Fatalf("quality calls = %d, want 1", quality.calls)
		}
		fields := utilityJSONFields(t, assertSingleUtilityJSONResult(t, stdout, "branch.sync-base"))
		if fields["mutated"] != "true" ||
			fields["recommendedAction"] != "merged" ||
			fields["qualityStatus"] != string(port.QualityPassed) ||
			fields["qualityDetail"] != "all branch gates passed" {
			t.Fatalf("merge sync fields = %#v", fields)
		}
	})

	t.Run("honors a declined rebase confirmation", func(t *testing.T) {
		prompt := &commandHelperPrompt{
			confirms: []commandHelperConfirmReply{{value: false}},
		}
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, prompt, "human")

		_, _, err := executeBranchCommand(
			t,
			newBranchSyncBaseCommand(application),
			context.Background(),
			"--branch", "feature/ABC-123-add-export",
			"--strategy", "rebase",
		)
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if len(prompt.confirmRequests) != 1 ||
			prompt.confirmRequests[0].Label != "Synchronize branch base" ||
			!strings.Contains(prompt.confirmRequests[0].Description, "rebase") {
			t.Fatalf("sync confirmation request = %#v", prompt.confirmRequests)
		}
		if len(git.rebasedBases) != 0 {
			t.Fatalf("declined sync rebased %v", git.rebasedBases)
		}
	})

	discoverErr := errors.New("repository discovery failed")
	currentErr := errors.New("current branch failed")
	fetchErr := errors.New("fetch failed")
	for _, testCase := range []struct {
		name      string
		arguments []string
		configure func(*branchCommandGit)
		wantCode  problem.Code
		wantErr   error
	}{
		{
			name: "preserves discovery failures",
			configure: func(git *branchCommandGit) {
				git.discoverErr = discoverErr
			},
			wantErr: discoverErr,
		},
		{
			name: "preserves current branch failures",
			configure: func(git *branchCommandGit) {
				git.currentErr = currentErr
			},
			wantErr: currentErr,
		},
		{
			name:      "rejects bases from another remote",
			arguments: []string{"--branch", "feature/ABC-123-add-export", "--base", "upstream/develop"},
			wantCode:  problem.CodeBranchBaseInvalid,
		},
		{
			name:      "rejects malformed merge messages before confirmation",
			arguments: []string{"--branch", "feature/ABC-123-add-export", "--strategy", "merge", "--merge-message", "not a commit message"},
			wantCode:  problem.CodeCommitHeaderInvalid,
		},
		{
			name:      "preserves synchronization service failures",
			arguments: []string{"--branch", "feature/ABC-123-add-export"},
			configure: func(git *branchCommandGit) {
				git.fetchErr = fetchErr
			},
			wantErr: fetchErr,
		},
		{
			name:      "returns invalid strategy errors from the synchronizer",
			arguments: []string{"--branch", "feature/ABC-123-add-export", "--strategy", "rewrite"},
			wantCode:  problem.CodeInvalidInput,
		},
	} {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			git := newBranchCommandGit(t, "feature/ABC-123-add-export")
			if testCase.configure != nil {
				testCase.configure(git)
			}
			_, _, err := executeBranchCommand(
				t,
				newBranchSyncBaseCommand(newBranchCommandApplication(git, nil, nil, "human")),
				context.Background(),
				testCase.arguments...,
			)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("sync error = %v, want %v", err, testCase.wantErr)
				}
			} else {
				assertProblemCode(t, err, testCase.wantCode)
			}
		})
	}

	t.Run("stops synchronization when the command context is cancelled", func(t *testing.T) {
		git := newBranchCommandGit(t, "feature/ABC-123-add-export")
		application := newBranchCommandApplication(git, nil, nil, "human")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := executeBranchCommand(
			t,
			newBranchSyncBaseCommand(application),
			ctx,
			"--branch", "feature/ABC-123-add-export",
		)
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled sync error = %v, want context cancellation", err)
		}
	})
}

func TestBranchCommandReportHelpers(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		result branchapp.CreateResult
		want   string
	}{
		{
			name:   "dry run",
			result: branchapp.CreateResult{DryRun: true},
			want:   "Branch creation plan generated.",
		},
		{
			name: "created branch",
			want: "Branch created.",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := branchCreationSummary(testCase.result); got != testCase.want {
				t.Fatalf("branchCreationSummary(%#v) = %q, want %q", testCase.result, got, testCase.want)
			}
		})
	}

	plan := []branchapp.PlanStep{
		{Action: "fetch", Detail: "git fetch --prune origin"},
		{Action: "create", Detail: "feature/ABC-123-add-export from origin/develop"},
	}
	if got := planText(plan); got != "fetch: git fetch --prune origin; create: feature/ABC-123-add-export from origin/develop" {
		t.Fatalf("planText() = %q", got)
	}
	if got := planText(nil); got != "" {
		t.Fatalf("empty planText() = %q", got)
	}
	if got := boolString(true); got != "true" {
		t.Fatalf("boolString(true) = %q", got)
	}
	if got := boolString(false); got != "false" {
		t.Fatalf("boolString(false) = %q", got)
	}
}

func TestBranchCommandFlagParsingFailure(t *testing.T) {
	command := NewWithRuntime(
		BuildInfo{Version: "test"},
		commandRuntime(newBranchCommandGit(t, "feature/ABC-123-add-export")),
	)
	_, err := executeBootstrapCommand(
		t,
		command,
		"branch", "create",
		"--switch=not-a-boolean",
	)
	assertProblemCode(t, err, problem.CodeInvalidInput)
}

func executeBranchCommand(
	t *testing.T,
	command *cobra.Command,
	ctx context.Context,
	arguments ...string,
) (string, string, error) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(arguments)
	err := command.ExecuteContext(ctx)
	return stdout.String(), stderr.String(), err
}

func newBranchCommandApplication(
	git port.GitRepository,
	quality port.QualityRunner,
	prompt port.Prompt,
	output string,
) *application {
	if quality == nil {
		quality = &branchCommandQuality{
			result: port.QualityResult{
				Status: port.QualityUnconfigured,
				Detail: "quality is not configured",
			},
		}
	}
	runtime := commandRuntime(git)
	runtime.Quality = quality
	runtime.PromptFactory = func(bool, string) port.Prompt {
		return prompt
	}
	runtime.InputIsTerminal = func() bool {
		return prompt != nil
	}
	runtime.OutputIsTerminal = func() bool {
		return prompt != nil
	}
	interactive := "never"
	if prompt != nil {
		interactive = "auto"
	}
	return newApplication(runtime, &appOptions{
		interactive: interactive,
		output:      output,
		color:       "never",
		remote:      "origin",
		repository:  "C:/repo",
		timeout:     time.Second,
	})
}

func branchNames(names []branch.BranchName) []string {
	result := make([]string, 0, len(names))
	for _, name := range names {
		result = append(result, name.String())
	}
	return result
}

type branchCreateCall struct {
	name     branch.BranchName
	base     branch.TargetBase
	switchTo bool
}

type branchMergeCall struct {
	base    branch.TargetBase
	message commitmsg.Message
}

type branchCommandGit struct {
	*commandGit

	discoverErr           error
	currentErr            error
	validateErr           error
	hasCommitsErr         error
	worktreeCleanErr      error
	branchExistsErr       error
	officialBranchesErr   error
	fetchErr              error
	createErr             error
	publicationErr        error
	missingBaseCommitsErr error
	rebaseErr             error
	mergeErr              error

	hasCommits         bool
	worktreeClean      bool
	branchExists       bool
	officialBranches   []branch.BranchName
	publication        branch.PublicationState
	missingBaseCommits bool

	discoverContexts  []context.Context
	currentContexts   []context.Context
	currentCalls      int
	validatedBranches []branch.BranchName
	fetchCalls        int
	createdBranches   []branchCreateCall
	rebasedBases      []branch.TargetBase
	mergedBranches    []branchMergeCall
}

func newBranchCommandGit(t *testing.T, current string) *branchCommandGit {
	t.Helper()
	return &branchCommandGit{
		commandGit:    newCommandGit(t, current, nil),
		hasCommits:    true,
		worktreeClean: true,
		publication:   branch.PublicationUnpublished,
	}
}

func (git *branchCommandGit) Discover(ctx context.Context, directory string) (port.RepositoryIdentity, error) {
	git.discoverContexts = append(git.discoverContexts, ctx)
	if git.discoverErr != nil {
		return port.RepositoryIdentity{}, git.discoverErr
	}
	return git.commandGit.Discover(ctx, directory)
}

func (git *branchCommandGit) CurrentBranch(
	ctx context.Context,
	repository port.RepositoryIdentity,
) (branch.BranchName, error) {
	git.currentCalls++
	git.currentContexts = append(git.currentContexts, ctx)
	if git.currentErr != nil {
		return branch.BranchName{}, git.currentErr
	}
	return git.commandGit.CurrentBranch(ctx, repository)
}

func (git *branchCommandGit) ValidateBranchRef(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
) error {
	git.validatedBranches = append(git.validatedBranches, name)
	if git.validateErr != nil {
		return git.validateErr
	}
	return git.commandGit.ValidateBranchRef(ctx, repository, name)
}

func (git *branchCommandGit) HasCommits(
	ctx context.Context,
	repository port.RepositoryIdentity,
) (bool, error) {
	if git.hasCommitsErr != nil {
		return false, git.hasCommitsErr
	}
	return git.hasCommits, nil
}

func (git *branchCommandGit) IsWorktreeClean(
	ctx context.Context,
	repository port.RepositoryIdentity,
) (bool, error) {
	if git.worktreeCleanErr != nil {
		return false, git.worktreeCleanErr
	}
	return git.worktreeClean, nil
}

func (git *branchCommandGit) BranchExists(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
) (bool, error) {
	if git.branchExistsErr != nil {
		return false, git.branchExistsErr
	}
	return git.branchExists, nil
}

func (git *branchCommandGit) OfficialBranchesForTicket(
	ctx context.Context,
	repository port.RepositoryIdentity,
	id ticket.ID,
) ([]branch.BranchName, error) {
	if git.officialBranchesErr != nil {
		return nil, git.officialBranchesErr
	}
	return append([]branch.BranchName(nil), git.officialBranches...), nil
}

func (git *branchCommandGit) Fetch(ctx context.Context, repository port.RepositoryIdentity) error {
	git.fetchCalls++
	if git.fetchErr != nil {
		return git.fetchErr
	}
	return git.commandGit.Fetch(ctx, repository)
}

func (git *branchCommandGit) CreateBranch(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	base branch.TargetBase,
	switchTo bool,
) error {
	if git.createErr != nil {
		return git.createErr
	}
	git.createdBranches = append(git.createdBranches, branchCreateCall{
		name:     name,
		base:     base,
		switchTo: switchTo,
	})
	return nil
}

func (git *branchCommandGit) PublicationState(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
) (branch.PublicationState, error) {
	if git.publicationErr != nil {
		return branch.PublicationUnknown, git.publicationErr
	}
	return git.publication, nil
}

func (git *branchCommandGit) HasMissingBaseCommits(
	ctx context.Context,
	repository port.RepositoryIdentity,
	base branch.TargetBase,
) (bool, error) {
	if git.missingBaseCommitsErr != nil {
		return false, git.missingBaseCommitsErr
	}
	return git.missingBaseCommits, nil
}

func (git *branchCommandGit) Rebase(
	ctx context.Context,
	repository port.RepositoryIdentity,
	base branch.TargetBase,
) error {
	if git.rebaseErr != nil {
		return git.rebaseErr
	}
	git.rebasedBases = append(git.rebasedBases, base)
	return nil
}

func (git *branchCommandGit) Merge(
	ctx context.Context,
	repository port.RepositoryIdentity,
	base branch.TargetBase,
	message commitmsg.Message,
) error {
	if git.mergeErr != nil {
		return git.mergeErr
	}
	git.mergedBranches = append(git.mergedBranches, branchMergeCall{
		base:    base,
		message: message,
	})
	return nil
}

var _ port.GitRepository = (*branchCommandGit)(nil)

type branchCommandQuality struct {
	result   port.QualityResult
	err      error
	calls    int
	contexts []context.Context
	requests []port.QualityRequest
}

func (runner *branchCommandQuality) Run(
	ctx context.Context,
	repository port.RepositoryIdentity,
	request port.QualityRequest,
) (port.QualityResult, error) {
	runner.calls++
	runner.contexts = append(runner.contexts, ctx)
	cloned := request
	cloned.Families = append([]branch.Family(nil), request.Families...)
	runner.requests = append(runner.requests, cloned)
	if runner.err != nil {
		return port.QualityResult{}, runner.err
	}
	return runner.result, nil
}

var _ port.QualityRunner = (*branchCommandQuality)(nil)
