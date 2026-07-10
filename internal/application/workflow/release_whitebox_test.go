package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

type releaseWhiteboxGit struct {
	*fakeGitRepository

	validateErr       error
	hasCommitsErr     error
	cleanErr          error
	branchExistsErr   error
	createErr         error
	storeErr          error
	releaseTagsErr    error
	cherryPickErr     error
	deleteErr         error
	clearErr          error
	commitMessagesErr error
	fetchErrors       []error

	validateContexts    []context.Context
	hasCommitsContexts  []context.Context
	fetchContexts       []context.Context
	storeContexts       []context.Context
	cherryPickContexts  []context.Context
	deleteContexts      []context.Context
	clearContexts       []context.Context
	releaseTagRevisions []string
	deletedBranches     []branch.BranchName
	deleteForces        []bool
}

func newReleaseWhiteboxGit() *releaseWhiteboxGit {
	return &releaseWhiteboxGit{
		fakeGitRepository: &fakeGitRepository{
			hasCommits:  true,
			clean:       true,
			publication: branch.PublicationUnpublished,
			messages:    []string{"fix(ABC-999): resolve payment timeout"},
		},
	}
}

func (git *releaseWhiteboxGit) ValidateBranchRef(ctx context.Context, repository port.RepositoryIdentity, name branch.BranchName) error {
	git.validateContexts = append(git.validateContexts, ctx)
	if git.validateErr != nil {
		git.calls = append(git.calls, "validate-ref")
		return git.validateErr
	}
	return git.fakeGitRepository.ValidateBranchRef(ctx, repository, name)
}

func (git *releaseWhiteboxGit) HasCommits(ctx context.Context, repository port.RepositoryIdentity) (bool, error) {
	git.hasCommitsContexts = append(git.hasCommitsContexts, ctx)
	if git.hasCommitsErr != nil {
		git.calls = append(git.calls, "has-commits")
		return false, git.hasCommitsErr
	}
	return git.fakeGitRepository.HasCommits(ctx, repository)
}

func (git *releaseWhiteboxGit) IsWorktreeClean(ctx context.Context, repository port.RepositoryIdentity) (bool, error) {
	if git.cleanErr != nil {
		git.calls = append(git.calls, "worktree-clean")
		return false, git.cleanErr
	}
	return git.fakeGitRepository.IsWorktreeClean(ctx, repository)
}

func (git *releaseWhiteboxGit) BranchExists(ctx context.Context, repository port.RepositoryIdentity, name branch.BranchName) (bool, error) {
	if git.branchExistsErr != nil {
		git.calls = append(git.calls, "branch-exists")
		return false, git.branchExistsErr
	}
	return git.fakeGitRepository.BranchExists(ctx, repository, name)
}

func (git *releaseWhiteboxGit) Fetch(ctx context.Context, repository port.RepositoryIdentity) error {
	git.fetchContexts = append(git.fetchContexts, ctx)
	if len(git.fetchErrors) > 0 {
		err := git.fetchErrors[0]
		git.fetchErrors = git.fetchErrors[1:]
		if err != nil {
			git.calls = append(git.calls, "fetch")
			return err
		}
	}
	return git.fakeGitRepository.Fetch(ctx, repository)
}

func (git *releaseWhiteboxGit) CreateBranch(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	base branch.TargetBase,
	switchTo bool,
) error {
	if git.createErr != nil {
		git.calls = append(git.calls, "create-branch")
		return git.createErr
	}
	return git.fakeGitRepository.CreateBranch(ctx, repository, name, base, switchTo)
}

func (git *releaseWhiteboxGit) StoreWorkflowBase(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	base branch.TargetBase,
) error {
	git.storeContexts = append(git.storeContexts, ctx)
	if git.storeErr != nil {
		git.calls = append(git.calls, "store-workflow-base")
		return git.storeErr
	}
	return git.fakeGitRepository.StoreWorkflowBase(ctx, repository, name, base)
}

func (git *releaseWhiteboxGit) ReleaseTagsAt(
	ctx context.Context,
	repository port.RepositoryIdentity,
	revision string,
) ([]string, error) {
	git.releaseTagRevisions = append(git.releaseTagRevisions, revision)
	if git.releaseTagsErr != nil {
		git.calls = append(git.calls, "release-tags")
		return nil, git.releaseTagsErr
	}
	return git.fakeGitRepository.ReleaseTagsAt(ctx, repository, revision)
}

func (git *releaseWhiteboxGit) CherryPick(ctx context.Context, repository port.RepositoryIdentity, commitID string) error {
	git.cherryPickContexts = append(git.cherryPickContexts, ctx)
	if git.cherryPickErr != nil {
		git.calls = append(git.calls, "cherry-pick")
		return git.cherryPickErr
	}
	return git.fakeGitRepository.CherryPick(ctx, repository, commitID)
}

func (git *releaseWhiteboxGit) CommitMessagesSince(
	ctx context.Context,
	repository port.RepositoryIdentity,
	base branch.TargetBase,
) ([]string, error) {
	if git.commitMessagesErr != nil {
		git.calls = append(git.calls, "commit-messages")
		return nil, git.commitMessagesErr
	}
	return git.fakeGitRepository.CommitMessagesSince(ctx, repository, base)
}

func (git *releaseWhiteboxGit) DeleteLocalBranch(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	force bool,
) error {
	git.deleteContexts = append(git.deleteContexts, ctx)
	git.deletedBranches = append(git.deletedBranches, name)
	git.deleteForces = append(git.deleteForces, force)
	if git.deleteErr != nil {
		git.calls = append(git.calls, "delete-local-branch")
		return git.deleteErr
	}
	return git.fakeGitRepository.DeleteLocalBranch(ctx, repository, name, force)
}

func (git *releaseWhiteboxGit) ClearWorkflowBase(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
) error {
	git.clearContexts = append(git.clearContexts, ctx)
	if git.clearErr != nil {
		git.calls = append(git.calls, "clear-workflow-base")
		return git.clearErr
	}
	return git.fakeGitRepository.ClearWorkflowBase(ctx, repository, name)
}

type releaseWhiteboxPublisher struct {
	result   port.PublishedPullRequest
	err      error
	contexts []context.Context
	requests []port.PullRequest
}

func (publisher *releaseWhiteboxPublisher) Publish(ctx context.Context, request port.PullRequest) (port.PublishedPullRequest, error) {
	publisher.contexts = append(publisher.contexts, ctx)
	publisher.requests = append(publisher.requests, request)
	return publisher.result, publisher.err
}

func newReleaseWhiteboxService(git port.GitRepository, publisher port.PullRequestPublisher) *ReleaseService {
	branches := branchapp.NewService(git, &fakeKeyPolicy{})
	sync := branchapp.NewSynchronizer(git, branches, nil)
	tickets := NewTicketService(branches, sync, git, nil, publisher)
	return NewReleaseService(branches, git, publisher).WithTicketService(tickets)
}

func newReleaseWhiteboxServiceWithoutTickets(git port.GitRepository) *ReleaseService {
	branches := branchapp.NewService(git, &fakeKeyPolicy{})
	return NewReleaseService(branches, git, nil)
}

func releaseHotfixRequest() StartHotfixRequest {
	return StartHotfixRequest{
		Repository:   testRepository(),
		Ticket:       mustTicket("ABC-999"),
		Slug:         mustSlug("payment-timeout"),
		AffectedLine: mustBranch("main"),
	}
}

func releaseStabilizationRequest() CreateReleaseStabilizationRequest {
	return CreateReleaseStabilizationRequest{
		Repository: testRepository(),
		Release:    mustBranch("release/2.8.0"),
		Ticket:     mustTicket("ABC-999"),
		Slug:       mustSlug("release-blocker"),
		Kind:       ReleaseStabilizationBlocker,
	}
}

func releasePropagationRequest() PropagateHotfixRequest {
	return PropagateHotfixRequest{
		Repository: testRepository(),
		Source:     mustBranch("hotfix/ABC-999-payment-timeout"),
		TargetLine: mustBranch("main"),
		CommitID:   strings.Repeat("a", 40),
		Slug:       mustSlug("forward-port-payment-timeout"),
	}
}

func assertReleaseErrorIs(t *testing.T, got error, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}

func assertReleaseNoCall(t *testing.T, calls []string, forbidden string) {
	t.Helper()
	if countCall(calls, forbidden) != 0 {
		t.Fatalf("calls %v include forbidden %q", calls, forbidden)
	}
}

func TestReleaseWhiteboxStartHotfixBoundaries(t *testing.T) {
	t.Run("requires a composed branch service", func(t *testing.T) {
		_, err := (&ReleaseService{}).StartHotfix(context.Background(), releaseHotfixRequest())
		assertProblemCode(t, err, problem.CodeInternal)
	})

	t.Run("rejects absent and non-shared affected lines before Git", func(t *testing.T) {
		for _, affected := range []branch.BranchName{{}, mustBranch("feature/ABC-999-payment-timeout")} {
			git := newReleaseWhiteboxGit()
			request := releaseHotfixRequest()
			request.AffectedLine = affected

			_, err := newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), request)
			assertProblemCode(t, err, problem.CodeInvalidInput)
			if len(git.calls) != 0 {
				t.Fatalf("invalid input called Git: %v", git.calls)
			}
		}
	})

	t.Run("rejects missing repositories and invalid remotes before creation", func(t *testing.T) {
		for _, repository := range []port.RepositoryIdentity{
			{},
			{Root: testRepository().Root, Remote: "invalid remote"},
		} {
			git := newReleaseWhiteboxGit()
			request := releaseHotfixRequest()
			request.Repository = repository

			_, err := newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), request)
			if repository.Root == "" {
				assertProblemCode(t, err, problem.CodeRepositoryNotFound)
			} else {
				assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
			}
			if len(git.calls) != 0 {
				t.Fatalf("invalid repository called Git: %v", git.calls)
			}
		}
	})

	t.Run("honors a cancelled context before adapter interactions", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		git := newReleaseWhiteboxGit()

		_, err := newReleaseWhiteboxService(git, nil).StartHotfix(ctx, releaseHotfixRequest())
		assertProblemCode(t, err, problem.CodeOperationCancelled)
		if len(git.calls) != 0 {
			t.Fatalf("cancelled operation called Git: %v", git.calls)
		}
	})

	t.Run("propagates validation and branch creation failures", func(t *testing.T) {
		validationFailure := errors.New("validate hotfix ref")
		git := newReleaseWhiteboxGit()
		git.validateErr = validationFailure
		_, err := newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), releaseHotfixRequest())
		assertReleaseErrorIs(t, err, validationFailure)
		assertReleaseNoCall(t, git.calls, "create-branch")

		createFailure := errors.New("create hotfix")
		git = newReleaseWhiteboxGit()
		git.createErr = createFailure
		_, err = newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), releaseHotfixRequest())
		assertReleaseErrorIs(t, err, createFailure)
		assertReleaseNoCall(t, git.calls, "store-workflow-base")
	})

	t.Run("preflights the metadata dependency before local mutation", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		branches := branchapp.NewService(git, &fakeKeyPolicy{})
		service := NewReleaseService(branches, nil, nil)

		_, err := service.StartHotfix(context.Background(), releaseHotfixRequest())
		assertProblemCode(t, err, problem.CodeInternal)
		assertReleaseNoCall(t, git.calls, "create-branch")
	})

	t.Run("returns a plan without mutations during dry run", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		request := releaseHotfixRequest()
		request.DryRun = true

		result, err := newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || result.Name.String() != "hotfix/ABC-999-payment-timeout" || result.Base.String() != "origin/main" {
			t.Fatalf("StartHotfix() = %#v", result)
		}
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "store-workflow-base")
		assertReleaseNoCall(t, git.calls, "push")
	})

	t.Run("stores the hotfix provenance and forwards context", func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "hotfix")
		git := newReleaseWhiteboxGit()

		result, err := newReleaseWhiteboxService(git, nil).StartHotfix(ctx, releaseHotfixRequest())
		if err != nil {
			t.Fatal(err)
		}
		if stored := git.workflowBases[result.Name.String()]; stored.String() != "origin/main" {
			t.Fatalf("stored base = %q, want origin/main", stored)
		}
		if len(git.validateContexts) == 0 || git.validateContexts[0] != ctx {
			t.Fatalf("validation contexts = %v, want %v", git.validateContexts, ctx)
		}
		if len(git.storeContexts) != 1 || git.storeContexts[0] != ctx {
			t.Fatalf("store contexts = %v, want %v", git.storeContexts, ctx)
		}

		storeFailure := errors.New("store workflow base")
		git = newReleaseWhiteboxGit()
		git.storeErr = storeFailure
		_, err = newReleaseWhiteboxService(git, nil).StartHotfix(context.Background(), releaseHotfixRequest())
		assertReleaseErrorIs(t, err, storeFailure)
	})
}

func TestReleaseWhiteboxCutReleaseCreatesOnlyCIIntent(t *testing.T) {
	version := mustReleaseVersion(t, "2.8.0")

	t.Run("rejects an absent version and missing Git adapter", func(t *testing.T) {
		_, err := (&ReleaseService{}).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
		})
		assertProblemCode(t, err, problem.CodeBranchNameInvalid)

		_, err = (&ReleaseService{}).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeInternal)
	})

	t.Run("propagates validation and source resolution errors", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		_, err := newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: port.RepositoryIdentity{},
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		validationFailure := errors.New("validate release intent")
		git = newReleaseWhiteboxGit()
		git.validateErr = validationFailure
		_, err = newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, validationFailure)

		git = newReleaseWhiteboxGit()
		_, err = newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: port.RepositoryIdentity{Root: testRepository().Root, Remote: "bad remote"},
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
	})

	t.Run("plans dry runs without fetching or mutating a shared line", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		result, err := newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
			DryRun:     true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || result.Intent.Workflow != "create-protected-line.yml" ||
			result.Intent.Kind != "release" || result.Intent.Branch.String() != "release/2.8.0" ||
			result.Intent.Source.String() != "origin/develop" || result.Intent.Inputs["version"] != "2.8.0" ||
			len(result.Plan) != 2 {
			t.Fatalf("CutRelease() = %#v", result)
		}
		assertReleaseNoCall(t, git.calls, "fetch")
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "push")
	})

	t.Run("propagates fetch and commit inspection failures", func(t *testing.T) {
		fetchFailure := errors.New("fetch release source")
		git := newReleaseWhiteboxGit()
		git.fetchErrors = []error{fetchFailure}
		_, err := newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, fetchFailure)

		commitInspectionFailure := errors.New("inspect release source")
		git = newReleaseWhiteboxGit()
		git.hasCommitsErr = commitInspectionFailure
		_, err = newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, commitInspectionFailure)

		git = newReleaseWhiteboxGit()
		git.hasCommits = false
		_, err = newReleaseWhiteboxService(git, nil).CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeRepositoryHasNoCommits)
	})

	t.Run("forwards context to read-only adapters and never pushes", func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "release")
		git := newReleaseWhiteboxGit()

		result, err := newReleaseWhiteboxService(git, nil).CutRelease(ctx, CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(git.validateContexts) != 1 || git.validateContexts[0] != ctx ||
			len(git.fetchContexts) != 1 || git.fetchContexts[0] != ctx ||
			len(git.hasCommitsContexts) != 1 || git.hasCommitsContexts[0] != ctx {
			t.Fatalf("release contexts were not forwarded: validate=%v fetch=%v commits=%v",
				git.validateContexts, git.fetchContexts, git.hasCommitsContexts)
		}
		if result.Intent.Branch.Family() != branch.FamilyRelease {
			t.Fatalf("release intent branch = %q", result.Intent.Branch)
		}
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "push")
	})
}

func TestReleaseWhiteboxPrepareSupportProvenance(t *testing.T) {
	version := mustSupportVersion(t, "2.8")

	t.Run("rejects malformed requests and missing dependencies", func(t *testing.T) {
		_, err := (&ReleaseService{}).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
		})
		assertProblemCode(t, err, problem.CodeBranchNameInvalid)

		git := newReleaseWhiteboxGit()
		_, err = newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: port.RepositoryIdentity{},
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		_, err = (&ReleaseService{}).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeInternal)
	})

	t.Run("stops on source fetch, tag lookup, and protected-line fetch failures", func(t *testing.T) {
		fetchFailure := errors.New("fetch main")
		git := newReleaseWhiteboxGit()
		git.fetchErrors = []error{fetchFailure}
		_, err := newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, fetchFailure)

		tagFailure := errors.New("read release tags")
		git = newReleaseWhiteboxGit()
		git.releaseTagsErr = tagFailure
		_, err = newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, tagFailure)

		sharedLineFetchFailure := errors.New("fetch support source")
		git = newReleaseWhiteboxGit()
		git.releaseTags = []string{"v2.8.0"}
		git.fetchErrors = []error{nil, sharedLineFetchFailure}
		_, err = newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertReleaseErrorIs(t, err, sharedLineFetchFailure)
	})

	t.Run("requires a matching main release tag", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		git.releaseTags = []string{"v2.9.0", "not-a-version"}

		_, err := newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
		assertReleaseNoCall(t, git.calls, "validate-ref")
	})

	t.Run("skips tag inspection only for dry runs", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		result, err := newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
			DryRun:     true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || result.Intent.Kind != "support" || result.Intent.Source.String() != "origin/main" {
			t.Fatalf("PrepareSupport() = %#v", result)
		}
		assertReleaseNoCall(t, git.calls, "fetch")
		assertReleaseNoCall(t, git.calls, "release-tags")
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "push")
	})

	t.Run("requires commits after provenance is established", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		git.releaseTags = []string{"v2.8.0"}
		git.hasCommits = false

		_, err := newReleaseWhiteboxService(git, nil).PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		assertProblemCode(t, err, problem.CodeRepositoryHasNoCommits)
	})

	t.Run("creates a CI-owned intent from the tagged main line", func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "support")
		git := newReleaseWhiteboxGit()
		git.releaseTags = []string{"v2.8.0"}

		result, err := newReleaseWhiteboxService(git, nil).PrepareSupport(ctx, PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Intent.Branch.String() != "support/2.8" || result.Intent.Source.String() != "origin/main" ||
			result.Intent.Inputs["kind"] != "support" || countCall(git.calls, "fetch") != 2 ||
			len(git.releaseTagRevisions) != 1 || git.releaseTagRevisions[0] != "origin/main" {
			t.Fatalf("PrepareSupport() = %#v, calls=%v, revisions=%v", result, git.calls, git.releaseTagRevisions)
		}
		if len(git.fetchContexts) != 2 || git.fetchContexts[0] != ctx || git.fetchContexts[1] != ctx {
			t.Fatalf("fetch contexts = %v, want %v", git.fetchContexts, ctx)
		}
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "push")
	})
}

func TestReleaseWhiteboxReleaseStabilizationBoundaries(t *testing.T) {
	t.Run("requires branches and a release line", func(t *testing.T) {
		_, err := (&ReleaseService{}).CreateReleaseStabilization(context.Background(), releaseStabilizationRequest())
		assertProblemCode(t, err, problem.CodeInternal)

		git := newReleaseWhiteboxGit()
		request := releaseStabilizationRequest()
		request.Release = mustBranch("main")
		_, err = newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
		assertProblemCode(t, err, problem.CodeInvalidInput)
		assertReleaseNoCall(t, git.calls, "validate-ref")
	})

	t.Run("rejects unsupported kinds, repositories, and remotes before creation", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		request := releaseStabilizationRequest()
		request.Kind = "feature"
		_, err := newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
		assertProblemCode(t, err, problem.CodeInvalidInput)

		git = newReleaseWhiteboxGit()
		request = releaseStabilizationRequest()
		request.Repository = port.RepositoryIdentity{}
		_, err = newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		git = newReleaseWhiteboxGit()
		request = releaseStabilizationRequest()
		request.Repository = port.RepositoryIdentity{Root: testRepository().Root, Remote: "bad remote"}
		_, err = newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
	})

	t.Run("honors cancellation and propagates creation validation failures", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		git := newReleaseWhiteboxGit()
		_, err := newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(ctx, releaseStabilizationRequest())
		assertProblemCode(t, err, problem.CodeOperationCancelled)

		validationFailure := errors.New("validate stabilization ref")
		git = newReleaseWhiteboxGit()
		git.validateErr = validationFailure
		_, err = newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), releaseStabilizationRequest())
		assertReleaseErrorIs(t, err, validationFailure)
	})

	t.Run("uses only a dry-run plan when requested", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		switchToBranch := false
		request := releaseStabilizationRequest()
		request.Switch = &switchToBranch
		request.DryRun = true

		result, err := newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || result.Switched || result.Base.String() != "origin/release/2.8.0" {
			t.Fatalf("CreateReleaseStabilization() = %#v", result)
		}
		assertReleaseNoCall(t, git.calls, "create-branch")
		assertReleaseNoCall(t, git.calls, "store-workflow-base")
	})

	t.Run("preflights the metadata dependency before local mutation", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		branches := branchapp.NewService(git, &fakeKeyPolicy{})
		service := NewReleaseService(branches, nil, nil)

		_, err := service.CreateReleaseStabilization(context.Background(), releaseStabilizationRequest())
		assertProblemCode(t, err, problem.CodeInternal)
		assertReleaseNoCall(t, git.calls, "create-branch")
	})

	t.Run("maps each permitted kind and records the release provenance", func(t *testing.T) {
		for _, testCase := range []struct {
			kind   ReleaseStabilizationKind
			family branch.Family
		}{
			{kind: ReleaseStabilizationBlocker, family: branch.FamilyFix},
			{kind: ReleaseStabilizationDocs, family: branch.FamilyDocs},
			{kind: ReleaseStabilizationPrep, family: branch.FamilyChore},
		} {
			t.Run(string(testCase.kind), func(t *testing.T) {
				git := newReleaseWhiteboxGit()
				request := releaseStabilizationRequest()
				request.Kind = testCase.kind

				result, err := newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				if result.Name.Family() != testCase.family || git.workflowBases[result.Name.String()].String() != "origin/release/2.8.0" {
					t.Fatalf("CreateReleaseStabilization() = %#v, bases=%v", result, git.workflowBases)
				}
			})
		}
	})

	t.Run("propagates provenance storage failures", func(t *testing.T) {
		storeFailure := errors.New("store stabilization base")
		git := newReleaseWhiteboxGit()
		git.storeErr = storeFailure

		_, err := newReleaseWhiteboxService(git, nil).CreateReleaseStabilization(context.Background(), releaseStabilizationRequest())
		assertReleaseErrorIs(t, err, storeFailure)
	})
}

func TestReleaseWhiteboxPromotionAndBackmergePublication(t *testing.T) {
	release := mustBranch("release/2.8.0")

	t.Run("promotion validates inputs and supports no-publisher paths", func(t *testing.T) {
		_, err := (&ReleaseService{}).PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    mustBranch("main"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)

		_, err = (&ReleaseService{}).PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: port.RepositoryIdentity{},
			Release:    release,
		})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		result, err := (&ReleaseService{}).PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    release,
			DryRun:     true,
		})
		if err != nil || !result.DryRun || result.PublishedURL != "" || result.PullRequest.Target.String() != "main" {
			t.Fatalf("dry-run promotion = (%#v, %v)", result, err)
		}

		result, err = (&ReleaseService{}).PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    release,
		})
		if err != nil || result.PublishedURL != "" {
			t.Fatalf("unpublished promotion = (%#v, %v)", result, err)
		}
	})

	t.Run("promotion propagates publisher errors and emits a complete PR intent", func(t *testing.T) {
		publishFailure := errors.New("publish promotion")
		publisher := &releaseWhiteboxPublisher{err: publishFailure}
		_, err := (&ReleaseService{publisher: publisher}).PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    release,
		})
		assertReleaseErrorIs(t, err, publishFailure)

		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "promotion")
		publisher = &releaseWhiteboxPublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/promotion"}}
		result, err := (&ReleaseService{publisher: publisher}).PrepareReleasePromotion(ctx, PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    release,
			Draft:      true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.PublishedURL == "" || result.PullRequest.Source.String() != "release/2.8.0" ||
			result.PullRequest.Target.String() != "main" || result.PullRequest.Title != "Release 2.8.0 into main" ||
			!result.PullRequest.Draft || len(publisher.contexts) != 1 || publisher.contexts[0] != ctx {
			t.Fatalf("promotion result = %#v, publisher=%#v", result, publisher)
		}
	})

	t.Run("backmerge validates inputs and supports no-publisher paths", func(t *testing.T) {
		_, err := (&ReleaseService{}).PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
			Repository: testRepository(),
			Release:    mustBranch("main"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)

		_, err = (&ReleaseService{}).PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
			Repository: port.RepositoryIdentity{},
			Release:    release,
		})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		result, err := (&ReleaseService{}).PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
			Repository: testRepository(),
			Release:    release,
			DryRun:     true,
		})
		if err != nil || !result.DryRun || result.PublishedURL != "" || result.PullRequest.Target.String() != "develop" {
			t.Fatalf("dry-run backmerge = (%#v, %v)", result, err)
		}

		result, err = (&ReleaseService{}).PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
			Repository: testRepository(),
			Release:    release,
		})
		if err != nil || result.PublishedURL != "" {
			t.Fatalf("unpublished backmerge = (%#v, %v)", result, err)
		}
	})

	t.Run("backmerge propagates publisher errors and emits a complete PR intent", func(t *testing.T) {
		publishFailure := errors.New("publish backmerge")
		publisher := &releaseWhiteboxPublisher{err: publishFailure}
		_, err := (&ReleaseService{publisher: publisher}).PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
			Repository: testRepository(),
			Release:    release,
		})
		assertReleaseErrorIs(t, err, publishFailure)

		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "backmerge")
		publisher = &releaseWhiteboxPublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/backmerge"}}
		result, err := (&ReleaseService{publisher: publisher}).PrepareReleaseBackmerge(ctx, PrepareReleaseBackmergeRequest{
			Repository: testRepository(),
			Release:    release,
			Draft:      true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.PublishedURL == "" || result.PullRequest.Source.String() != "release/2.8.0" ||
			result.PullRequest.Target.String() != "develop" || result.PullRequest.Title != "Backmerge release 2.8.0 into develop" ||
			!result.PullRequest.Draft || len(publisher.contexts) != 1 || publisher.contexts[0] != ctx {
			t.Fatalf("backmerge result = %#v, publisher=%#v", result, publisher)
		}
	})
}

func TestReleaseWhiteboxPropagateHotfixBoundaries(t *testing.T) {
	t.Run("requires every composed workflow dependency", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		_, err := NewReleaseService(nil, git, nil).WithTicketService(&TicketService{}).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertProblemCode(t, err, problem.CodeInternal)

		branches := branchapp.NewService(git, &fakeKeyPolicy{})
		_, err = NewReleaseService(branches, nil, nil).WithTicketService(&TicketService{}).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertProblemCode(t, err, problem.CodeInternal)

		_, err = newReleaseWhiteboxServiceWithoutTickets(git).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertProblemCode(t, err, problem.CodeInternal)
	})

	t.Run("rejects source, target, commit, and repository inputs before mutations", func(t *testing.T) {
		for _, mutate := range []func(*PropagateHotfixRequest){
			func(request *PropagateHotfixRequest) { request.Source = mustBranch("feature/ABC-999-payment-timeout") },
			func(request *PropagateHotfixRequest) { request.TargetLine = mustBranch("feature/ABC-998-another-line") },
			func(request *PropagateHotfixRequest) { request.CommitID = "not-a-sha" },
			func(request *PropagateHotfixRequest) { request.Repository = port.RepositoryIdentity{} },
			func(request *PropagateHotfixRequest) {
				request.Repository = port.RepositoryIdentity{Root: testRepository().Root, Remote: "bad remote"}
			},
		} {
			git := newReleaseWhiteboxGit()
			request := releasePropagationRequest()
			mutate(&request)

			_, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), request)
			if err == nil {
				t.Fatal("invalid propagation succeeded")
			}
			assertReleaseNoCall(t, git.calls, "create-branch")
			assertReleaseNoCall(t, git.calls, "store-workflow-base")
			assertReleaseNoCall(t, git.calls, "cherry-pick")
		}
	})

	t.Run("fails safely when the derived forward-port slug is invalid", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		request := releasePropagationRequest()
		request.Source = mustBranch("hotfix/ABC-999-" + strings.Repeat("a", 100))
		request.Slug = branch.Slug{}

		_, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchSlugInvalid)
		assertReleaseNoCall(t, git.calls, "create-branch")
	})

	t.Run("propagates working-branch creation failures", func(t *testing.T) {
		createFailure := errors.New("create forward-port branch")
		git := newReleaseWhiteboxGit()
		git.createErr = createFailure

		_, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertReleaseErrorIs(t, err, createFailure)
		assertReleaseNoCall(t, git.calls, "store-workflow-base")
		assertReleaseNoCall(t, git.calls, "cherry-pick")
	})

	t.Run("supports every active target line in dry-run mode", func(t *testing.T) {
		for _, target := range []string{"main", "develop", "release/2.8.0", "support/2.8"} {
			t.Run(target, func(t *testing.T) {
				git := newReleaseWhiteboxGit()
				request := releasePropagationRequest()
				request.TargetLine = mustBranch(target)
				request.DryRun = true

				result, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				if !result.Branch.DryRun || !result.Publication.DryRun ||
					result.Publication.PullRequest.Target.String() != target ||
					result.Publication.PullRequest.Source.String() != result.Branch.Name.String() {
					t.Fatalf("dry-run propagation = %#v", result)
				}
				assertReleaseNoCall(t, git.calls, "create-branch")
				assertReleaseNoCall(t, git.calls, "store-workflow-base")
				assertReleaseNoCall(t, git.calls, "cherry-pick")
				assertReleaseNoCall(t, git.calls, "push")
			})
		}
	})

	t.Run("derives a default propagation slug", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		request := releasePropagationRequest()
		request.Slug = branch.Slug{}
		request.TargetLine = mustBranch("support/2.8")
		request.DryRun = true

		result, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		if result.Branch.Name.String() != "fix/ABC-999-forward-port-payment-timeout" {
			t.Fatalf("derived branch = %q", result.Branch.Name)
		}
	})

	t.Run("propagates metadata, cherry-pick, and publication failures", func(t *testing.T) {
		storeFailure := errors.New("store propagation base")
		git := newReleaseWhiteboxGit()
		git.storeErr = storeFailure
		_, err := newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertReleaseErrorIs(t, err, storeFailure)
		assertReleaseNoCall(t, git.calls, "cherry-pick")

		cherryPickFailure := errors.New("cherry-pick propagation")
		git = newReleaseWhiteboxGit()
		git.cherryPickErr = cherryPickFailure
		_, err = newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertReleaseErrorIs(t, err, cherryPickFailure)

		publicationFailure := errors.New("validate propagated commit series")
		git = newReleaseWhiteboxGit()
		git.commitMessagesErr = publicationFailure
		_, err = newReleaseWhiteboxService(git, nil).PropagateHotfix(context.Background(), releasePropagationRequest())
		assertReleaseErrorIs(t, err, publicationFailure)
	})

	t.Run("pushes only the derived working branch and publishes its PR", func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "propagation")
		git := newReleaseWhiteboxGit()
		publisher := &releaseWhiteboxPublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/forward-port"}}
		request := releasePropagationRequest()
		request.Push = true

		result, err := newReleaseWhiteboxService(git, publisher).PropagateHotfix(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if !result.CherryPicked || !result.Publication.Pushed || result.Publication.PublishedURL == "" ||
			len(git.pushed) != 1 || git.pushed[0].String() != result.Branch.Name.String() ||
			git.pushed[0].Family() != branch.FamilyFix || len(publisher.requests) != 1 ||
			publisher.requests[0].Target.String() != "main" {
			t.Fatalf("propagation = %#v, pushes=%v, publications=%#v", result, git.pushed, publisher.requests)
		}
		if len(git.storeContexts) != 1 || git.storeContexts[0] != ctx ||
			len(git.cherryPickContexts) != 1 || git.cherryPickContexts[0] != ctx ||
			len(publisher.contexts) != 1 || publisher.contexts[0] != ctx {
			t.Fatalf("propagation contexts were not forwarded")
		}
		if git.pushed[0].String() == request.TargetLine.String() {
			t.Fatalf("protected target line %q was pushed directly", request.TargetLine)
		}
	})
}

func TestReleaseWhiteboxCleanupBranchBoundaries(t *testing.T) {
	scratch := mustBranch("scratch/ABC-999-cleanup")

	t.Run("requires Git and accepts only private scratch branches", func(t *testing.T) {
		_, err := (&ReleaseService{}).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     scratch,
		})
		assertProblemCode(t, err, problem.CodeInternal)

		git := newReleaseWhiteboxGit()
		_, err = newReleaseWhiteboxService(git, nil).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     mustBranch("release/2.8.0"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
		assertReleaseNoCall(t, git.calls, "delete-local-branch")
	})

	t.Run("rejects missing repositories before deletion", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		_, err := newReleaseWhiteboxService(git, nil).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: port.RepositoryIdentity{},
			Branch:     scratch,
		})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)
		assertReleaseNoCall(t, git.calls, "delete-local-branch")
	})

	t.Run("does not mutate during a dry run", func(t *testing.T) {
		git := newReleaseWhiteboxGit()
		result, err := newReleaseWhiteboxService(git, nil).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     scratch,
			DryRun:     true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.DryRun || result.DeletedLocal || result.MetadataCleared {
			t.Fatalf("dry-run cleanup = %#v", result)
		}
		assertReleaseNoCall(t, git.calls, "delete-local-branch")
		assertReleaseNoCall(t, git.calls, "clear-workflow-base")
	})

	t.Run("propagates deletion and metadata cleanup failures", func(t *testing.T) {
		deleteFailure := errors.New("delete scratch")
		git := newReleaseWhiteboxGit()
		git.deleteErr = deleteFailure
		_, err := newReleaseWhiteboxService(git, nil).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     scratch,
		})
		assertReleaseErrorIs(t, err, deleteFailure)
		assertReleaseNoCall(t, git.calls, "clear-workflow-base")

		clearFailure := errors.New("clear scratch metadata")
		git = newReleaseWhiteboxGit()
		git.clearErr = clearFailure
		_, err = newReleaseWhiteboxService(git, nil).CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     scratch,
		})
		assertReleaseErrorIs(t, err, clearFailure)
		if countCall(git.calls, "delete-local-branch") != 1 {
			t.Fatalf("metadata failure did not follow local deletion: %v", git.calls)
		}
	})

	t.Run("deletes only the local scratch branch and its metadata", func(t *testing.T) {
		type contextKey struct{}
		ctx := context.WithValue(context.Background(), contextKey{}, "cleanup")
		git := newReleaseWhiteboxGit()
		git.workflowBases = map[string]branch.TargetBase{scratch.String(): mustBase("origin", "develop")}

		result, err := newReleaseWhiteboxService(git, nil).CleanupBranch(ctx, CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     scratch,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.DeletedLocal || !result.MetadataCleared || len(git.deletedBranches) != 1 ||
			git.deletedBranches[0].String() != scratch.String() || !git.deleteForces[0] ||
			len(git.deleteContexts) != 1 || git.deleteContexts[0] != ctx ||
			len(git.clearContexts) != 1 || git.clearContexts[0] != ctx {
			t.Fatalf("cleanup result = %#v, deleted=%v", result, git.deletedBranches)
		}
		if _, found := git.workflowBases[scratch.String()]; found {
			t.Fatalf("workflow metadata for %q was retained", scratch)
		}
		assertReleaseNoCall(t, git.calls, "push")
	})
}

func TestReleaseWhiteboxHelperBranches(t *testing.T) {
	t.Run("stabilization kinds map to the only permitted families", func(t *testing.T) {
		for _, testCase := range []struct {
			kind   ReleaseStabilizationKind
			family branch.Family
		}{
			{ReleaseStabilizationBlocker, branch.FamilyFix},
			{ReleaseStabilizationDocs, branch.FamilyDocs},
			{ReleaseStabilizationPrep, branch.FamilyChore},
		} {
			family, err := stabilizationFamily(testCase.kind)
			if err != nil || family != testCase.family {
				t.Fatalf("stabilizationFamily(%q) = (%q, %v)", testCase.kind, family, err)
			}
		}
		_, err := stabilizationFamily("feature")
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("matches only released support line tags", func(t *testing.T) {
		version := mustSupportVersion(t, "2.8")
		if hasMatchingSupportReleaseTag([]string{"invalid", "v2.9.0", "2.8.0"}, version) != true {
			t.Fatal("matching support tag was not recognized")
		}
		if hasMatchingSupportReleaseTag([]string{"vv2.8.0", "v2.8"}, version) {
			t.Fatal("invalid tags matched a support line")
		}
		if hasMatchingSupportReleaseTag(nil, version) {
			t.Fatal("empty tag set matched a support line")
		}
	})

	t.Run("normalizes the default remote and preserves explicit input", func(t *testing.T) {
		_, err := normalizeWorkflowRepository(port.RepositoryIdentity{})
		assertProblemCode(t, err, problem.CodeRepositoryNotFound)

		normalized, err := normalizeWorkflowRepository(port.RepositoryIdentity{Root: testRepository().Root})
		if err != nil || normalized.Remote != "origin" {
			t.Fatalf("default normalization = (%#v, %v)", normalized, err)
		}

		normalized, err = normalizeWorkflowRepository(port.RepositoryIdentity{Root: testRepository().Root, Remote: "upstream"})
		if err != nil || normalized.Remote != "upstream" {
			t.Fatalf("explicit normalization = (%#v, %v)", normalized, err)
		}
	})

	if mustMain().String() != "main" {
		t.Fatalf("mustMain() = %q", mustMain())
	}
}

var _ port.GitRepository = (*releaseWhiteboxGit)(nil)
var _ port.PullRequestPublisher = (*releaseWhiteboxPublisher)(nil)
