package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

type reconciliationAlignmentGit struct {
	*releaseWhiteboxGit

	current         branch.BranchName
	currentErr      error
	targetExists    bool
	targetErr       error
	mergeErr        error
	validateErrors  []error
	mergedBase      branch.TargetBase
	mergedMessage   commitmsg.Message
	activeOperation string
	active          bool
	activeErr       error
	conflicts       bool
	conflictsErr    error
	continueErr     error
	continued       bool
	mergeMatches    bool
	mergeInspectErr error
	releaseRevision string
	developRevision string
	revisionErr     error
}

func (git *reconciliationAlignmentGit) CurrentBranch(context.Context, port.RepositoryIdentity) (branch.BranchName, error) {
	git.calls = append(git.calls, "current-branch")
	if git.currentErr != nil {
		return branch.BranchName{}, git.currentErr
	}
	return git.current, nil
}

func (git *reconciliationAlignmentGit) ValidateBranchRef(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	git.calls = append(git.calls, "validate-ref")
	if len(git.validateErrors) > 0 {
		err := git.validateErrors[0]
		git.validateErrors = git.validateErrors[1:]
		return err
	}
	return git.validateErr
}

func (git *reconciliationAlignmentGit) TargetBaseExists(
	context.Context,
	port.RepositoryIdentity,
	branch.TargetBase,
) (bool, error) {
	git.calls = append(git.calls, "target-base-exists")
	if git.targetErr != nil {
		return false, git.targetErr
	}
	return git.targetExists, nil
}

func (git *reconciliationAlignmentGit) Merge(
	_ context.Context,
	_ port.RepositoryIdentity,
	base branch.TargetBase,
	message commitmsg.Message,
) error {
	git.calls = append(git.calls, "merge")
	if git.mergeErr != nil {
		return git.mergeErr
	}
	git.mergedBase = base
	git.mergedMessage = message
	return nil
}

func (git *reconciliationAlignmentGit) ActiveOperation(context.Context, port.RepositoryIdentity) (string, bool, error) {
	git.calls = append(git.calls, "active-operation")
	return git.activeOperation, git.active, git.activeErr
}

func (git *reconciliationAlignmentGit) HasUnmergedConflicts(context.Context, port.RepositoryIdentity) (bool, error) {
	git.calls = append(git.calls, "unmerged-conflicts")
	return git.conflicts, git.conflictsErr
}

type reconciliationAlignmentPublisher struct {
	releaseWhiteboxPublisher
	preflightErr   error
	preflightCalls int
}

func (publisher *reconciliationAlignmentPublisher) Validate(context.Context, port.PullRequestPublication) error {
	publisher.preflightCalls++
	return publisher.preflightErr
}

func newReconciliationAlignmentGit(t *testing.T) *reconciliationAlignmentGit {
	t.Helper()

	worker := mustBranch("chore/GOV-20-align-release-reconciliation-base")
	release := mustBranch("release/1.0.1")
	releaseBase := mustBase("origin", release.String())
	base := newReleaseWhiteboxGit()
	base.clean = true
	base.missing = true
	base.publication = branch.PublicationUnpublished
	base.workflowBases = map[string]branch.TargetBase{worker.String(): releaseBase}

	return &reconciliationAlignmentGit{
		releaseWhiteboxGit: base,
		current:            worker,
		targetExists:       true,
		mergeMatches:       true,
		releaseRevision:    "release-sha",
		developRevision:    "develop-sha",
	}
}

type reconciliationAlignmentRecoveryGit struct {
	*reconciliationAlignmentGit
}

func newReconciliationAlignmentRecoveryGit(t *testing.T) *reconciliationAlignmentRecoveryGit {
	t.Helper()
	return &reconciliationAlignmentRecoveryGit{reconciliationAlignmentGit: newReconciliationAlignmentGit(t)}
}

func (git *reconciliationAlignmentRecoveryGit) ContinueMerge(context.Context, port.RepositoryIdentity) error {
	git.calls = append(git.calls, "continue-merge")
	if git.continueErr != nil {
		return git.continueErr
	}
	git.continued = true
	return nil
}

func (git *reconciliationAlignmentRecoveryGit) HeadIsMergeOf(
	context.Context,
	port.RepositoryIdentity,
	branch.TargetBase,
	branch.TargetBase,
) (bool, error) {
	git.calls = append(git.calls, "head-is-merge-of")
	return git.mergeMatches, git.mergeInspectErr
}

func (git *reconciliationAlignmentRecoveryGit) ResolveReconciliationBases(
	context.Context,
	port.RepositoryIdentity,
	branch.TargetBase,
	branch.TargetBase,
) (string, string, error) {
	git.calls = append(git.calls, "resolve-reconciliation-bases")
	return git.releaseRevision, git.developRevision, git.revisionErr
}

func newReconciliationAlignmentService(
	git port.GitRepository,
	quality port.QualityRunner,
	publisher port.PullRequestPublisher,
	lifecycle port.ReleaseLifecycleProvider,
) *ReleaseService {
	branches := branchapp.NewService(git, &fakeKeyPolicy{})
	tickets := NewTicketService(branches, branchapp.NewSynchronizer(git, branches, quality), git, quality, publisher)
	return NewReleaseService(branches, git, publisher).
		WithTicketService(tickets).
		WithQualityRunner(quality).
		WithReleaseLifecycleProvider(lifecycle)
}

func reconciliationAlignmentRequest() AlignReleaseReconciliationBaseRequest {
	return AlignReleaseReconciliationBaseRequest{
		Repository: testRepository(),
		Release:    mustBranch("release/1.0.1"),
		Branch:     mustBranch("chore/GOV-20-align-release-reconciliation-base"),
	}
}

func requiredReconciliationEvidence() port.ReleaseReconciliationEvidence {
	return port.ReleaseReconciliationEvidence{
		PromotionPullRequestURL: "https://example.invalid/pr/30",
		PromotionMergeCommit:    strings.Repeat("a", 40),
		Tag:                     "v1.0.1",
		ReleaseURL:              "https://example.invalid/releases/v1.0.1",
		EffectiveDelta:          true,
	}
}

func TestReleaseReconciliationBaseAlignment(t *testing.T) {
	t.Run("plans a dry run without provider verification merge quality or publication", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		quality := &fakeQualityRunner{}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}

		request := reconciliationAlignmentRequest()
		request.DryRun = true
		result, err := newReconciliationAlignmentService(git, quality, nil, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || !result.MissingDevelopCommits || result.Merged || result.Pushed ||
			result.PullRequest.Source.String() != request.Branch.String() ||
			result.PullRequest.Target.String() != "develop" {
			t.Fatalf("dry-run result = %#v", result)
		}
		if quality.calls != 0 || len(lifecycle.reconciles) != 0 ||
			strings.Contains(strings.Join(git.calls, ","), "fetch") ||
			strings.Contains(strings.Join(git.calls, ","), "merge") ||
			strings.Contains(strings.Join(git.calls, ","), "push") {
			t.Fatalf("dry-run mutated workflow state: calls=%v quality=%d reconciles=%d",
				git.calls, quality.calls, len(lifecycle.reconciles))
		}
	})

	t.Run("merges current develop, validates quality, and publishes the reconciliation PR", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		quality := &fakeQualityRunner{}
		publisher := &reconciliationAlignmentPublisher{
			releaseWhiteboxPublisher: releaseWhiteboxPublisher{
				result: port.PublishedPullRequest{URL: "https://example.invalid/pr/reconciliation"},
			},
		}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}
		request := reconciliationAlignmentRequest()
		request.Push = true
		request.CreatePullRequest = true
		request.Draft = true

		result, err := newReconciliationAlignmentService(git, quality, publisher, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if !result.MissingDevelopCommits || !result.Merged || !result.Pushed ||
			result.PublishedURL != "https://example.invalid/pr/reconciliation" || result.Quality == nil ||
			result.Quality.Status != port.QualityPassed || !result.PullRequest.Draft ||
			!result.Evidence.EffectiveDelta {
			t.Fatalf("alignment result = %#v", result)
		}
		if git.mergedBase.String() != "origin/develop" ||
			git.mergedMessage.String() != "chore(GOV-20): align release 1.0.1 with develop for reconciliation" {
			t.Fatalf("merge = (%q, %q)", git.mergedBase, git.mergedMessage)
		}
		if quality.calls != 1 || len(git.pushed) != 1 || git.pushed[0] != request.Branch ||
			publisher.preflightCalls != 1 || len(publisher.requests) != 1 ||
			publisher.requests[0].Target.String() != "develop" || len(lifecycle.reconciles) != 1 {
			t.Fatalf("quality=%d pushed=%v preflight=%d publications=%#v reconciles=%#v",
				quality.calls, git.pushed, publisher.preflightCalls, publisher.requests, lifecycle.reconciles)
		}
	})

	t.Run("returns a validated unpublished alignment after merging develop", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		quality := &fakeQualityRunner{}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}

		result, err := newReconciliationAlignmentService(git, quality, nil, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		if err != nil || !result.Merged || result.Pushed || result.Quality == nil || quality.calls != 1 {
			t.Fatalf("unpublished alignment = (%#v, %v), quality=%d", result, err, quality.calls)
		}
	})

	t.Run("returns a safe no-op when develop is already an ancestor", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		git.missing = false
		quality := &fakeQualityRunner{}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}

		result, err := newReconciliationAlignmentService(git, quality, nil, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		if err != nil {
			t.Fatal(err)
		}
		if result.MissingDevelopCommits || result.Merged || result.Quality != nil || quality.calls != 0 ||
			len(lifecycle.reconciles) != 1 {
			t.Fatalf("no-op result = %#v quality=%d reconciles=%d", result, quality.calls, len(lifecycle.reconciles))
		}
	})

	t.Run("runs quality before an explicit push of an already aligned branch", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		git.missing = false
		quality := &fakeQualityRunner{}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}
		request := reconciliationAlignmentRequest()
		request.Push = true

		result, err := newReconciliationAlignmentService(git, quality, nil, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Merged || !result.Pushed || result.Quality == nil || quality.calls != 1 ||
			len(git.pushed) != 1 {
			t.Fatalf("already-aligned push = %#v quality=%d pushed=%v", result, quality.calls, git.pushed)
		}
	})

	t.Run("continues a resolved merge before quality and publication", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		git.active = true
		git.activeOperation = "merge"
		git.missing = false
		quality := &fakeQualityRunner{}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}
		request := reconciliationAlignmentRequest()
		request.Resume = true
		request.Push = true

		result, err := newReconciliationAlignmentService(git, quality, nil, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), request)
		if err != nil || !result.Resumed || !result.Merged || result.Prepared || !result.Pushed ||
			!git.continued || quality.calls != 1 {
			t.Fatalf("resumed alignment = (%#v, %v), continued=%t quality=%d", result, err, git.continued, quality.calls)
		}
	})

	t.Run("validates and publishes a prepared merge without local workflow metadata", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		git.missing = false
		git.workflowBases = nil
		quality := &fakeQualityRunner{}
		publisher := &reconciliationAlignmentPublisher{
			releaseWhiteboxPublisher: releaseWhiteboxPublisher{
				result: port.PublishedPullRequest{URL: "https://example.invalid/pr/prepared"},
			},
		}
		lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}
		request := reconciliationAlignmentRequest()
		request.Prepared = true
		request.Push = true
		request.CreatePullRequest = true

		result, err := newReconciliationAlignmentService(git, quality, publisher, lifecycle).
			AlignReleaseReconciliationBase(context.Background(), request)
		if err != nil || !result.Prepared || !result.Merged || result.Resumed || !result.Pushed ||
			result.PublishedURL != "https://example.invalid/pr/prepared" || quality.calls != 1 {
			t.Fatalf("prepared alignment = (%#v, %v), quality=%d", result, err, quality.calls)
		}
		if !strings.Contains(strings.Join(git.calls, ","), "head-is-merge-of") {
			t.Fatalf("prepared alignment did not inspect merge provenance: %v", git.calls)
		}
	})
}

func TestReleaseReconciliationBaseAlignmentRejectsUnsafeInputsBeforeMerge(t *testing.T) {
	testCases := []struct {
		name      string
		configure func(*reconciliationAlignmentGit, *AlignReleaseReconciliationBaseRequest, *ReleaseService)
		code      problem.Code
	}{
		{
			name: "missing dependencies",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, service *ReleaseService) {
				*service = ReleaseService{}
				_ = request
			},
			code: problem.CodeInternal,
		},
		{
			name: "non-release line",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				request.Release = mustBranch("main")
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "non-chore worker",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				request.Branch = mustBranch("fix/GOV-20-align-release-reconciliation-base")
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "pull request without push",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				request.CreatePullRequest = true
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "missing repository",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				request.Repository = port.RepositoryIdentity{}
			},
			code: problem.CodeRepositoryNotFound,
		},
		{
			name: "invalid remote",
			configure: func(_ *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				request.Repository.Remote = "invalid remote"
			},
			code: problem.CodeBranchBaseInvalid,
		},
		{
			name: "invalid worker reference",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.validateErrors = []error{errors.New("validate worker")}
			},
		},
		{
			name: "wrong checked-out branch",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.current = mustBranch("chore/GOV-20-another-release-prep")
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "current branch lookup failure",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.currentErr = errors.New("read current branch")
			},
		},
		{
			name: "workflow base lookup failure",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.workflowBaseErr = errors.New("read workflow base")
			},
		},
		{
			name: "missing workflow base",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.workflowBases = nil
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "mismatched workflow base",
			configure: func(git *reconciliationAlignmentGit, request *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.workflowBases[request.Branch.String()] = mustBase("origin", "release/1.0.0")
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "missing develop target",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.targetExists = false
			},
			code: problem.CodeInvalidInput,
		},
		{
			name: "develop target lookup failure",
			configure: func(git *reconciliationAlignmentGit, _ *AlignReleaseReconciliationBaseRequest, _ *ReleaseService) {
				git.targetErr = errors.New("inspect develop")
			},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			quality := &fakeQualityRunner{}
			lifecycle := &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}
			service := newReconciliationAlignmentService(git, quality, nil, lifecycle)
			request := reconciliationAlignmentRequest()
			testCase.configure(git, &request, service)

			_, err := service.AlignReleaseReconciliationBase(context.Background(), request)
			if testCase.code != "" {
				assertProblemCode(t, err, testCase.code)
			} else if err == nil {
				t.Fatal("expected dependency error")
			}
			if strings.Contains(strings.Join(git.calls, ","), "merge") {
				t.Fatalf("unsafe request merged: calls=%v", git.calls)
			}
		})
	}
}

func TestReleaseReconciliationBaseAlignmentFailureAndCleanupPaths(t *testing.T) {
	t.Run("dry runs reject unavailable and unreadable develop bases", func(t *testing.T) {
		testCases := []struct {
			name      string
			configure func(*reconciliationAlignmentGit)
			code      problem.Code
		}{
			{name: "target lookup", configure: func(git *reconciliationAlignmentGit) { git.targetErr = errors.New("target") }},
			{name: "target absent", configure: func(git *reconciliationAlignmentGit) { git.targetExists = false }, code: problem.CodeInvalidInput},
			{name: "missing-base lookup", configure: func(git *reconciliationAlignmentGit) { git.missingErr = errors.New("missing") }},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				git := newReconciliationAlignmentGit(t)
				testCase.configure(git)
				request := reconciliationAlignmentRequest()
				request.DryRun = true
				_, err := newReconciliationAlignmentService(
					git,
					&fakeQualityRunner{},
					nil,
					&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
				).AlignReleaseReconciliationBase(context.Background(), request)
				if testCase.code != "" {
					assertProblemCode(t, err, testCase.code)
				} else if err == nil {
					t.Fatal("expected dry-run failure")
				}
			})
		}
	})

	t.Run("fails before mutation when pull request preflight is unavailable", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		request := reconciliationAlignmentRequest()
		request.Push = true
		request.CreatePullRequest = true

		_, err := newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), request)
		assertProblemCode(t, err, problem.CodeExternalCommandFailed)
		if strings.Contains(strings.Join(git.calls, ","), "merge") {
			t.Fatalf("missing publisher merged: %v", git.calls)
		}
	})

	t.Run("fails before mutation when ticket publication composition is absent", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		branches := branchapp.NewService(git, &fakeKeyPolicy{})
		service := NewReleaseService(branches, git, nil).
			WithQualityRunner(&fakeQualityRunner{}).
			WithReleaseLifecycleProvider(&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()})
		request := reconciliationAlignmentRequest()
		request.Push = true
		request.CreatePullRequest = true

		_, err := service.AlignReleaseReconciliationBase(context.Background(), request)
		assertProblemCode(t, err, problem.CodeInternal)
		if strings.Contains(strings.Join(git.calls, ","), "merge") {
			t.Fatalf("missing publication service merged: %v", git.calls)
		}
	})

	t.Run("fails before mutation when the publisher preflight rejects the PR", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		preflightErr := errors.New("publisher preflight")
		publisher := &reconciliationAlignmentPublisher{preflightErr: preflightErr}
		request := reconciliationAlignmentRequest()
		request.Push = true
		request.CreatePullRequest = true

		_, err := newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			publisher,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), request)
		if !errors.Is(err, preflightErr) {
			t.Fatalf("preflight error = %v, want %v", err, preflightErr)
		}
		if strings.Contains(strings.Join(git.calls, ","), "merge") {
			t.Fatalf("preflight failure merged: %v", git.calls)
		}
	})

	t.Run("propagates worktree fetch missing-base and merge failures", func(t *testing.T) {
		testCases := []struct {
			name      string
			configure func(*reconciliationAlignmentGit)
		}{
			{name: "worktree lookup", configure: func(git *reconciliationAlignmentGit) { git.cleanErr = errors.New("status") }},
			{name: "dirty worktree", configure: func(git *reconciliationAlignmentGit) { git.clean = false }},
			{name: "fetch", configure: func(git *reconciliationAlignmentGit) { git.fetchErrors = []error{errors.New("fetch")} }},
			{name: "missing base", configure: func(git *reconciliationAlignmentGit) { git.missingErr = errors.New("missing base") }},
			{name: "merge", configure: func(git *reconciliationAlignmentGit) { git.mergeErr = errors.New("merge") }},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				git := newReconciliationAlignmentGit(t)
				testCase.configure(git)
				_, err := newReconciliationAlignmentService(
					git,
					&fakeQualityRunner{},
					nil,
					&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
				).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
				if err == nil {
					t.Fatal("expected workflow failure")
				}
			})
		}
	})

	t.Run("classifies a reconciliation merge conflict without mutating a shared line", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		mergeErr := errors.New("merge conflict")
		git.mergeErr = mergeErr
		git.conflicts = true
		_, err := newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		assertProblemCode(t, err, problem.CodeMergeConflict)
		if !errors.Is(err, mergeErr) {
			t.Fatalf("conflict error = %v, want %v", err, mergeErr)
		}

		git = newReconciliationAlignmentRecoveryGit(t)
		inspectErr := errors.New("inspect conflicts")
		git.mergeErr = mergeErr
		git.conflictsErr = inspectErr
		_, err = newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		if !errors.Is(err, inspectErr) {
			t.Fatalf("conflict inspection error = %v, want %v", err, inspectErr)
		}

		git = newReconciliationAlignmentRecoveryGit(t)
		revisionErr := errors.New("resolve revisions")
		git.mergeErr = mergeErr
		git.conflicts = true
		git.revisionErr = revisionErr
		_, err = newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		if !errors.Is(err, revisionErr) {
			t.Fatalf("revision error = %v, want %v", err, revisionErr)
		}

		withoutInspector := newReconciliationAlignmentGit(t)
		withoutInspector.mergeErr = mergeErr
		withoutInspector.conflicts = true
		_, err = newReconciliationAlignmentService(
			withoutInspector,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		assertProblemCode(t, err, problem.CodeInternal)
	})

	t.Run("rejects unverified no-delta reconciliation before merge", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		evidence := requiredReconciliationEvidence()
		evidence.EffectiveDelta = false

		_, err := newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: evidence},
		).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
		assertProblemCode(t, err, problem.CodeInvalidInput)
		if strings.Contains(strings.Join(git.calls, ","), "merge") {
			t.Fatalf("no-delta reconciliation merged: %v", git.calls)
		}
	})

	t.Run("preserves lifecycle validation quality publication and provider failures", func(t *testing.T) {
		t.Run("lifecycle", func(t *testing.T) {
			lifecycleErr := errors.New("verify release delivery")
			_, err := newReconciliationAlignmentService(
				newReconciliationAlignmentGit(t),
				&fakeQualityRunner{},
				nil,
				&releaseWhiteboxLifecycle{verifyErr: lifecycleErr},
			).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
			if !errors.Is(err, lifecycleErr) {
				t.Fatalf("lifecycle error = %v, want %v", err, lifecycleErr)
			}
		})

		t.Run("post-merge validation", func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			git.validateErrors = []error{nil, errors.New("revalidate")}
			_, err := newReconciliationAlignmentService(
				git,
				&fakeQualityRunner{},
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
			if err == nil {
				t.Fatal("expected revalidation failure")
			}
		})

		t.Run("quality", func(t *testing.T) {
			qualityErr := errors.New("quality")
			_, err := newReconciliationAlignmentService(
				newReconciliationAlignmentGit(t),
				&fakeQualityRunner{err: qualityErr},
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
			if !errors.Is(err, qualityErr) {
				t.Fatalf("quality error = %v, want %v", err, qualityErr)
			}
		})

		t.Run("missing quality runner", func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			_, err := newReconciliationAlignmentService(
				git,
				nil,
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), reconciliationAlignmentRequest())
			assertProblemCode(t, err, problem.CodeInternal)
		})

		t.Run("publication lookup", func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			git.publicationErr = errors.New("publication")
			request := reconciliationAlignmentRequest()
			request.Push = true
			_, err := newReconciliationAlignmentService(
				git,
				&fakeQualityRunner{},
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), request)
			if err == nil {
				t.Fatal("expected publication failure")
			}
		})

		t.Run("unknown publication", func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			git.publication = branch.PublicationUnknown
			request := reconciliationAlignmentRequest()
			request.Push = true
			_, err := newReconciliationAlignmentService(
				git,
				&fakeQualityRunner{},
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), request)
			assertProblemCode(t, err, problem.CodeInvalidInput)
		})

		t.Run("push", func(t *testing.T) {
			git := newReconciliationAlignmentGit(t)
			git.pushErr = errors.New("push")
			request := reconciliationAlignmentRequest()
			request.Push = true
			_, err := newReconciliationAlignmentService(
				git,
				&fakeQualityRunner{},
				nil,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), request)
			if err == nil {
				t.Fatal("expected push failure")
			}
		})

		t.Run("publisher", func(t *testing.T) {
			publishErr := errors.New("publish")
			publisher := &reconciliationAlignmentPublisher{
				releaseWhiteboxPublisher: releaseWhiteboxPublisher{err: publishErr},
			}
			request := reconciliationAlignmentRequest()
			request.Push = true
			request.CreatePullRequest = true
			_, err := newReconciliationAlignmentService(
				newReconciliationAlignmentGit(t),
				&fakeQualityRunner{},
				publisher,
				&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
			).AlignReleaseReconciliationBase(context.Background(), request)
			if !errors.Is(err, publishErr) {
				t.Fatalf("publisher error = %v, want %v", err, publishErr)
			}
		})
	})
}

func TestReleaseReconciliationBaseAlignmentRecoveryGuards(t *testing.T) {
	t.Run("rejects mutually exclusive resume and prepared modes", func(t *testing.T) {
		git := newReconciliationAlignmentGit(t)
		request := reconciliationAlignmentRequest()
		request.Resume = true
		request.Prepared = true
		_, err := newReconciliationAlignmentService(git, &fakeQualityRunner{}, nil, &releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()}).
			AlignReleaseReconciliationBase(context.Background(), request)
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("requires an active resolved merge before resuming", func(t *testing.T) {
		testCases := []struct {
			name      string
			configure func(*reconciliationAlignmentGit)
			code      problem.Code
		}{
			{
				name: "active operation lookup failure",
				configure: func(git *reconciliationAlignmentGit) {
					git.activeErr = errors.New("active operation")
				},
			},
			{
				name: "no active merge",
				configure: func(git *reconciliationAlignmentGit) {
					git.active = false
				},
				code: problem.CodeInvalidInput,
			},
			{
				name: "different active operation",
				configure: func(git *reconciliationAlignmentGit) {
					git.active = true
					git.activeOperation = "rebase"
				},
				code: problem.CodeInvalidInput,
			},
			{
				name: "conflict inspection failure",
				configure: func(git *reconciliationAlignmentGit) {
					git.active = true
					git.activeOperation = "merge"
					git.conflictsErr = errors.New("conflicts")
				},
			},
			{
				name: "unresolved conflicts",
				configure: func(git *reconciliationAlignmentGit) {
					git.active = true
					git.activeOperation = "merge"
					git.conflicts = true
				},
				code: problem.CodeInvalidInput,
			},
			{
				name: "missing merge continuator",
				configure: func(git *reconciliationAlignmentGit) {
					git.active = true
					git.activeOperation = "merge"
				},
				code: problem.CodeInternal,
			},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				git := newReconciliationAlignmentGit(t)
				testCase.configure(git)
				request := reconciliationAlignmentRequest()
				request.Resume = true
				_, err := newReconciliationAlignmentService(
					git,
					&fakeQualityRunner{},
					nil,
					&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
				).AlignReleaseReconciliationBase(context.Background(), request)
				if testCase.code != "" {
					assertProblemCode(t, err, testCase.code)
				} else if err == nil {
					t.Fatal("expected recovery failure")
				}
			})
		}
	})

	t.Run("propagates merge continuation failure and rejects a stale resumed target", func(t *testing.T) {
		git := newReconciliationAlignmentRecoveryGit(t)
		git.active = true
		git.activeOperation = "merge"
		git.continueErr = errors.New("continue merge")
		request := reconciliationAlignmentRequest()
		request.Resume = true
		_, err := newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), request)
		if !errors.Is(err, git.continueErr) {
			t.Fatalf("continue error = %v, want %v", err, git.continueErr)
		}

		git = newReconciliationAlignmentRecoveryGit(t)
		git.active = true
		git.activeOperation = "merge"
		git.missing = true
		_, err = newReconciliationAlignmentService(
			git,
			&fakeQualityRunner{},
			nil,
			&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
		).AlignReleaseReconciliationBase(context.Background(), request)
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("requires exact prepared merge provenance", func(t *testing.T) {
		testCases := []struct {
			name   string
			newGit func() port.GitRepository
			code   problem.Code
		}{
			{
				name: "prepared branch misses develop",
				newGit: func() port.GitRepository {
					git := newReconciliationAlignmentRecoveryGit(t)
					git.workflowBases = nil
					git.missing = true
					return git
				},
				code: problem.CodeInvalidInput,
			},
			{
				name: "missing merge inspector",
				newGit: func() port.GitRepository {
					git := newReconciliationAlignmentGit(t)
					git.workflowBases = nil
					git.missing = false
					return git
				},
				code: problem.CodeInternal,
			},
			{
				name: "merge inspector failure",
				newGit: func() port.GitRepository {
					git := newReconciliationAlignmentRecoveryGit(t)
					git.workflowBases = nil
					git.missing = false
					git.mergeInspectErr = errors.New("inspect merge")
					return git
				},
			},
			{
				name: "merge topology mismatch",
				newGit: func() port.GitRepository {
					git := newReconciliationAlignmentRecoveryGit(t)
					git.workflowBases = nil
					git.missing = false
					git.mergeMatches = false
					return git
				},
				code: problem.CodeInvalidInput,
			},
		}
		for _, testCase := range testCases {
			testCase := testCase
			t.Run(testCase.name, func(t *testing.T) {
				git := testCase.newGit()
				request := reconciliationAlignmentRequest()
				request.Prepared = true
				_, err := newReconciliationAlignmentService(
					git,
					&fakeQualityRunner{},
					nil,
					&releaseWhiteboxLifecycle{evidence: requiredReconciliationEvidence()},
				).AlignReleaseReconciliationBase(context.Background(), request)
				if testCase.code != "" {
					assertProblemCode(t, err, testCase.code)
				} else if err == nil {
					t.Fatal("expected prepared recovery failure")
				}
			})
		}
	})
}

func TestReconciliationBaseAlignmentHelpers(t *testing.T) {
	worker := mustBranch("chore/GOV-20-align-release-reconciliation-base")
	release := mustBranch("release/1.0.1")
	message := reconciliationBaseAlignmentMergeMessage(worker, release)
	if message.String() != "chore(GOV-20): align release 1.0.1 with develop for reconciliation" {
		t.Fatalf("merge message = %q", message)
	}
	pullRequest := reconciliationBaseAlignmentPullRequest(worker, mustBranch("develop"), true)
	if pullRequest.Source != worker || pullRequest.Target.String() != "develop" || pullRequest.Ticket.String() != "GOV-20" ||
		pullRequest.Title != "GOV-20: align-release-reconciliation-base" || !pullRequest.Draft {
		t.Fatalf("pull request = %#v", pullRequest)
	}
	cause := problem.New(problem.Details{
		Code:       problem.CodeGitCommandFailed,
		Category:   problem.CategoryGit,
		Context:    "action=merge the target base",
		Diagnostic: "CONFLICT (content): runtime.go",
	})
	conflict := reconciliationMergeConflictProblem(
		worker,
		release,
		mustBase("origin", "develop"),
		"release-sha",
		"develop-sha",
		cause,
	)
	assertProblemCode(t, conflict, problem.CodeMergeConflict)
	value, ok := problem.As(conflict)
	if !ok || value.Context != cause.Context || value.Diagnostic != cause.Diagnostic {
		t.Fatalf("conflict manifest = %#v", value)
	}
}
