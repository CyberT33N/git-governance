package workflow

import (
	"context"
	"strings"
	"testing"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

type fakeGitRepository struct {
	hasCommits      bool
	clean           bool
	exists          bool
	publication     branch.PublicationState
	missing         bool
	messages        []string
	messageBatches  [][]string
	err             error
	calls           []string
	createdNames    []branch.BranchName
	createdBases    []branch.TargetBase
	createdSwitches []bool
	pushed          []branch.BranchName
	cherryPicked    []string
	releaseTags     []string
	workflowBases   map[string]branch.TargetBase
}

func (fake *fakeGitRepository) Discover(context.Context, string) (port.RepositoryIdentity, error) {
	fake.calls = append(fake.calls, "discover")
	return testRepository(), fake.err
}

func (fake *fakeGitRepository) Version(context.Context) (string, error) {
	fake.calls = append(fake.calls, "version")
	return "git version test", fake.err
}

func (fake *fakeGitRepository) RemoteURL(context.Context, port.RepositoryIdentity) (string, error) {
	fake.calls = append(fake.calls, "remote-url")
	return "https://example.invalid/repo.git", fake.err
}

func (fake *fakeGitRepository) ActiveOperation(context.Context, port.RepositoryIdentity) (string, bool, error) {
	fake.calls = append(fake.calls, "active-operation")
	return "", false, fake.err
}

func (fake *fakeGitRepository) HasCommits(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "has-commits")
	return fake.hasCommits, fake.err
}

func (fake *fakeGitRepository) IsWorktreeClean(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "worktree-clean")
	return fake.clean, fake.err
}

func (fake *fakeGitRepository) CurrentBranch(context.Context, port.RepositoryIdentity) (branch.BranchName, error) {
	fake.calls = append(fake.calls, "current-branch")
	return mustBranch("feature/ABC-123-add-export"), fake.err
}

func (fake *fakeGitRepository) ValidateBranchRef(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	fake.calls = append(fake.calls, "validate-ref")
	return fake.err
}

func (fake *fakeGitRepository) BranchExists(context.Context, port.RepositoryIdentity, branch.BranchName) (bool, error) {
	fake.calls = append(fake.calls, "branch-exists")
	return fake.exists, fake.err
}

func (fake *fakeGitRepository) OfficialBranchesForTicket(context.Context, port.RepositoryIdentity, ticket.ID) ([]branch.BranchName, error) {
	fake.calls = append(fake.calls, "official-branches-for-ticket")
	return nil, fake.err
}

func (fake *fakeGitRepository) Fetch(context.Context, port.RepositoryIdentity) error {
	fake.calls = append(fake.calls, "fetch")
	return fake.err
}

func (fake *fakeGitRepository) CreateBranch(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName, base branch.TargetBase, switchTo bool) error {
	fake.calls = append(fake.calls, "create-branch")
	fake.createdNames = append(fake.createdNames, name)
	fake.createdBases = append(fake.createdBases, base)
	fake.createdSwitches = append(fake.createdSwitches, switchTo)
	return fake.err
}

func (fake *fakeGitRepository) StoreWorkflowBase(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName, base branch.TargetBase) error {
	fake.calls = append(fake.calls, "store-workflow-base")
	if fake.workflowBases == nil {
		fake.workflowBases = make(map[string]branch.TargetBase)
	}
	fake.workflowBases[name.String()] = base
	return fake.err
}

func (fake *fakeGitRepository) ClearWorkflowBase(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName) error {
	fake.calls = append(fake.calls, "clear-workflow-base")
	delete(fake.workflowBases, name.String())
	return fake.err
}

func (fake *fakeGitRepository) WorkflowBase(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName) (branch.TargetBase, bool, error) {
	fake.calls = append(fake.calls, "workflow-base")
	base, found := fake.workflowBases[name.String()]
	return base, found, fake.err
}

func (fake *fakeGitRepository) SwitchBranch(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	fake.calls = append(fake.calls, "switch")
	return fake.err
}

func (fake *fakeGitRepository) PublicationState(context.Context, port.RepositoryIdentity, branch.BranchName) (branch.PublicationState, error) {
	fake.calls = append(fake.calls, "publication")
	return fake.publication, fake.err
}

func (fake *fakeGitRepository) HasMissingBaseCommits(context.Context, port.RepositoryIdentity, branch.TargetBase) (bool, error) {
	fake.calls = append(fake.calls, "missing-base")
	return fake.missing, fake.err
}

func (fake *fakeGitRepository) CommitMessagesSince(context.Context, port.RepositoryIdentity, branch.TargetBase) ([]string, error) {
	fake.calls = append(fake.calls, "commit-messages")
	if len(fake.messageBatches) > 0 {
		messages := fake.messageBatches[0]
		fake.messageBatches = fake.messageBatches[1:]
		return append([]string(nil), messages...), fake.err
	}
	return append([]string(nil), fake.messages...), fake.err
}

func (fake *fakeGitRepository) Rebase(context.Context, port.RepositoryIdentity, branch.TargetBase) error {
	fake.calls = append(fake.calls, "rebase")
	return fake.err
}

func (fake *fakeGitRepository) Merge(context.Context, port.RepositoryIdentity, branch.TargetBase, commitmsg.Message) error {
	fake.calls = append(fake.calls, "merge")
	return fake.err
}

func (fake *fakeGitRepository) CherryPick(_ context.Context, _ port.RepositoryIdentity, commitID string) error {
	fake.calls = append(fake.calls, "cherry-pick")
	fake.cherryPicked = append(fake.cherryPicked, commitID)
	return fake.err
}

func (fake *fakeGitRepository) DeleteLocalBranch(context.Context, port.RepositoryIdentity, branch.BranchName, bool) error {
	fake.calls = append(fake.calls, "delete-local-branch")
	return fake.err
}

func (fake *fakeGitRepository) ReleaseTagsAt(context.Context, port.RepositoryIdentity, string) ([]string, error) {
	fake.calls = append(fake.calls, "release-tags")
	return append([]string(nil), fake.releaseTags...), fake.err
}

func (fake *fakeGitRepository) HasStagedChanges(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "staged")
	return true, fake.err
}

func (fake *fakeGitRepository) Stage(context.Context, port.RepositoryIdentity, []string) error {
	fake.calls = append(fake.calls, "stage")
	return fake.err
}

func (fake *fakeGitRepository) Commit(context.Context, port.RepositoryIdentity, commitmsg.Message) error {
	fake.calls = append(fake.calls, "commit")
	return fake.err
}

func (fake *fakeGitRepository) Push(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName, _ bool) error {
	fake.calls = append(fake.calls, "push")
	fake.pushed = append(fake.pushed, name)
	return fake.err
}

func (fake *fakeGitRepository) InspectPushUpdate(context.Context, port.RepositoryIdentity, branch.TargetBase, string, string) (port.PushUpdateInspection, error) {
	fake.calls = append(fake.calls, "inspect-push")
	return port.PushUpdateInspection{}, fake.err
}

type fakeKeyPolicy struct {
	err error
}

func (fake *fakeKeyPolicy) ValidateKey(context.Context, port.RepositoryIdentity, ticket.Key) error {
	return fake.err
}

type fakeQualityRunner struct {
	calls int
	err   error
}

func (fake *fakeQualityRunner) Run(context.Context, port.RepositoryIdentity, port.QualityRequest) (port.QualityResult, error) {
	fake.calls++
	return port.QualityResult{Status: port.QualityPassed}, fake.err
}

type fakePublisher struct {
	calls   int
	request port.PullRequest
	result  port.PublishedPullRequest
	err     error
}

func (fake *fakePublisher) Publish(_ context.Context, request port.PullRequest) (port.PublishedPullRequest, error) {
	fake.calls++
	fake.request = request
	return fake.result, fake.err
}

func TestStartTicketCreatesOfficialAndScratchWithoutSecondFetch(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{hasCommits: true, clean: true}
	service := newTicketService(git, nil, nil)
	result, err := service.StartTicket(context.Background(), StartTicketRequest{
		Repository:    testRepository(),
		Family:        branch.FamilyFeature,
		Ticket:        mustTicket("ABC-123"),
		Slug:          mustSlug("add-export"),
		CreateScratch: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Official.Name.String() != "feature/ABC-123-add-export" || result.Scratch == nil {
		t.Fatalf("StartTicket() = %#v", result)
	}
	if result.Scratch.Name.String() != "scratch/ABC-123-add-export-exploration" || result.Active.String() != result.Scratch.Name.String() {
		t.Fatalf("scratch result = %#v", result)
	}
	if len(git.createdBases) != 2 || git.createdBases[0].String() != "origin/develop" || git.createdBases[1].String() != "feature/ABC-123-add-export" {
		t.Fatalf("created bases = %v", git.createdBases)
	}
	if countCall(git.calls, "fetch") != 1 {
		t.Fatalf("fetch calls = %v", git.calls)
	}
}

func TestStartTicketRejectsNonRegularFamily(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{}
	service := newTicketService(git, nil, nil)
	_, err := service.StartTicket(context.Background(), StartTicketRequest{
		Repository: testRepository(),
		Family:     branch.FamilyHotfix,
		Ticket:     mustTicket("ABC-123"),
		Slug:       mustSlug("payment-timeout"),
	})
	assertProblemCode(t, err, problem.CodeInvalidInput)
	if len(git.calls) != 0 {
		t.Fatalf("invalid workflow must not invoke Git: %v", git.calls)
	}
}

func TestPublishTicketValidatesSeriesAndBuildsPullRequest(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		clean:       true,
		publication: branch.PublicationUnpublished,
		messages:    []string{"feat(ABC-123): add export"},
	}
	quality := &fakeQualityRunner{}
	publisher := &fakePublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/1"}}
	service := newTicketService(git, quality, publisher)
	name := mustBranch("feature/ABC-123-add-export")
	result, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     name,
		Push:       true,
		Draft:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Pushed || result.PullRequest.Source.String() != name.String() || result.PullRequest.Target.String() != "develop" || !result.PullRequest.Draft {
		t.Fatalf("PublishTicket() = %#v", result)
	}
	if result.PublishedURL != "https://example.invalid/pr/1" || publisher.calls != 1 || quality.calls != 1 {
		t.Fatalf("publish result = %#v, publisher=%d, quality=%d", result, publisher.calls, quality.calls)
	}
	expected := "validate-ref,fetch,commit-messages,validate-ref,worktree-clean,publication,missing-base,push"
	if got := strings.Join(git.calls, ","); got != expected {
		t.Fatalf("calls = %q, want %q", got, expected)
	}
}

func TestPublishTicketStopsOnInvalidSeries(t *testing.T) {
	t.Parallel()

	t.Run("no commits", func(t *testing.T) {
		git := &fakeGitRepository{clean: true, messages: nil}
		quality := &fakeQualityRunner{}
		service := newTicketService(git, quality, nil)
		_, err := service.PublishTicket(context.Background(), PublishTicketRequest{
			Repository: testRepository(),
			Branch:     mustBranch("feature/ABC-123-add-export"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
		if quality.calls != 0 {
			t.Fatal("quality must not run without a commit series")
		}
	})

	t.Run("mixed ticket", func(t *testing.T) {
		git := &fakeGitRepository{clean: true, messages: []string{"feat(ABC-124): unrelated work"}}
		service := newTicketService(git, nil, nil)
		_, err := service.PublishTicket(context.Background(), PublishTicketRequest{
			Repository: testRepository(),
			Branch:     mustBranch("feature/ABC-123-add-export"),
		})
		assertProblemCode(t, err, problem.CodeCommitTicketMismatch)
	})
}

func TestPublishTicketRevalidatesAfterRebase(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		clean:       true,
		publication: branch.PublicationUnpublished,
		missing:     true,
		messageBatches: [][]string{
			{"feat(ABC-123): add export"},
			{"feat(ABC-123): add export"},
		},
	}
	quality := &fakeQualityRunner{}
	service := newTicketService(git, quality, nil)

	result, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     mustBranch("feature/ABC-123-add-export"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Sync.Mutated || quality.calls != 2 {
		t.Fatalf("PublishTicket() = %#v, quality calls=%d", result, quality.calls)
	}
	if countCall(git.calls, "commit-messages") != 2 {
		t.Fatalf("commit-series validation calls = %v", git.calls)
	}
	if countCall(git.calls, "validate-ref") != 3 {
		t.Fatalf("branch/policy validation calls = %v", git.calls)
	}
}

func TestPublishTicketBlocksPushWhenPostRebaseValidationFails(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		clean:       true,
		publication: branch.PublicationUnpublished,
		missing:     true,
		messageBatches: [][]string{
			{"feat(ABC-123): add export"},
			{"feat(ABC-124): invalid after rebase"},
		},
	}
	service := newTicketService(git, &fakeQualityRunner{}, nil)
	_, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     mustBranch("feature/ABC-123-add-export"),
		Push:       true,
	})
	assertProblemCode(t, err, problem.CodeCommitTicketMismatch)
	if strings.Contains(strings.Join(git.calls, ","), "push") {
		t.Fatalf("post-rebase validation failure must stop before push: %v", git.calls)
	}
}

func TestPublishHotfixTargetsAffectedLine(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		clean:       true,
		publication: branch.PublicationUnpublished,
		messages:    []string{"fix(ABC-999): resolve payment timeout"},
	}
	service := newTicketService(git, nil, nil)
	name := mustBranch("hotfix/ABC-999-payment-timeout")
	affected := mustBase("origin", "main")

	result, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     name,
		Base:       &affected,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PullRequest.Target.String() != "main" {
		t.Fatalf("hotfix target = %q, want main", result.PullRequest.Target.String())
	}
}

func TestPublishHotfixUsesStoredWorkflowBase(t *testing.T) {
	t.Parallel()

	name := mustBranch("hotfix/ABC-999-payment-timeout")
	base := mustBase("origin", "main")
	git := &fakeGitRepository{
		clean:       true,
		publication: branch.PublicationUnpublished,
		messages:    []string{"fix(ABC-999): resolve payment timeout"},
		workflowBases: map[string]branch.TargetBase{
			name.String(): base,
		},
	}
	service := newTicketService(git, nil, nil)
	result, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     name,
	})
	if err != nil || result.PullRequest.Target.String() != "main" {
		t.Fatalf("PublishTicket() = (%#v, %v)", result, err)
	}
}

func TestPublishHotfixRejectsIncorrectAffectedLine(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{clean: true, messages: []string{"fix(ABC-999): resolve payment timeout"}}
	service := newTicketService(git, nil, nil)
	invalidBase := mustBase("origin", "develop")
	_, err := service.PublishTicket(context.Background(), PublishTicketRequest{
		Repository: testRepository(),
		Branch:     mustBranch("hotfix/ABC-999-payment-timeout"),
		Base:       &invalidBase,
	})
	assertProblemCode(t, err, problem.CodeInvalidInput)
}

func TestReleaseWorkflows(t *testing.T) {
	t.Parallel()

	t.Run("hotfix starts from main", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true}
		service := newReleaseService(git, nil)
		result, err := service.StartHotfix(context.Background(), StartHotfixRequest{
			Repository:   testRepository(),
			Ticket:       mustTicket("ABC-999"),
			Slug:         mustSlug("payment-timeout"),
			AffectedLine: mustBranch("main"),
		})
		if err != nil || result.Name.String() != "hotfix/ABC-999-payment-timeout" || result.Base.String() != "origin/main" {
			t.Fatalf("StartHotfix() = (%#v, %v)", result, err)
		}
		if stored, found := git.workflowBases[result.Name.String()]; !found || stored.String() != "origin/main" {
			t.Fatalf("hotfix workflow base metadata = %#v", git.workflowBases)
		}
	})

	t.Run("hotfix rejects develop", func(t *testing.T) {
		service := newReleaseService(&fakeGitRepository{}, nil)
		_, err := service.StartHotfix(context.Background(), StartHotfixRequest{
			Repository:   testRepository(),
			Ticket:       mustTicket("ABC-999"),
			Slug:         mustSlug("payment-timeout"),
			AffectedLine: mustBranch("develop"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("release cut starts from develop", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true}
		service := newReleaseService(git, nil)
		version := mustReleaseVersion(t, "2.8.0")
		result, err := service.CutRelease(context.Background(), CutReleaseRequest{
			Repository: testRepository(),
			Version:    version,
		})
		if err != nil || result.Intent.Branch.String() != "release/2.8.0" || result.Intent.Source.String() != "origin/develop" {
			t.Fatalf("CutRelease() = (%#v, %v)", result, err)
		}
		if result.Intent.Workflow != "create-protected-line.yml" || result.Intent.Kind != "release" {
			t.Fatalf("release intent = %#v", result.Intent)
		}
		if strings.Contains(strings.Join(git.calls, ","), "create-branch") {
			t.Fatalf("release intent must not create a local shared line: %v", git.calls)
		}
	})

	t.Run("support starts from main", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true, releaseTags: []string{"v2.8.0"}}
		service := newReleaseService(git, nil)
		version := mustSupportVersion(t, "2.8")
		result, err := service.PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    version,
		})
		if err != nil || result.Intent.Branch.String() != "support/2.8" || result.Intent.Source.String() != "origin/main" {
			t.Fatalf("PrepareSupport() = (%#v, %v)", result, err)
		}
		if result.Intent.Workflow != "create-protected-line.yml" || result.Intent.Kind != "support" {
			t.Fatalf("support intent = %#v", result.Intent)
		}
	})
}

func TestPrepareReleaseBackmerge(t *testing.T) {
	t.Parallel()

	publisher := &fakePublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/backmerge"}}
	service := newReleaseService(&fakeGitRepository{}, publisher)
	result, err := service.PrepareReleaseBackmerge(context.Background(), PrepareReleaseBackmergeRequest{
		Repository: testRepository(),
		Release:    mustBranch("release/2.8.0"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PullRequest.Source.String() != "release/2.8.0" || result.PullRequest.Target.String() != "develop" || result.PublishedURL == "" {
		t.Fatalf("PrepareReleaseBackmerge() = %#v", result)
	}
	if publisher.calls != 1 {
		t.Fatal("publisher was not invoked")
	}
}

func TestReleaseStabilizationConstrainsFamiliesAndBases(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		kind   ReleaseStabilizationKind
		family branch.Family
	}{
		{ReleaseStabilizationBlocker, branch.FamilyFix},
		{ReleaseStabilizationDocs, branch.FamilyDocs},
		{ReleaseStabilizationPrep, branch.FamilyChore},
	} {
		testCase := testCase
		t.Run(string(testCase.kind), func(t *testing.T) {
			t.Parallel()
			git := &fakeGitRepository{hasCommits: true, clean: true}
			service := newReleaseService(git, nil)
			result, err := service.CreateReleaseStabilization(context.Background(), CreateReleaseStabilizationRequest{
				Repository: testRepository(),
				Release:    mustBranch("release/2.8.0"),
				Ticket:     mustTicket("ABC-999"),
				Slug:       mustSlug("release-blocker-timeout"),
				Kind:       testCase.kind,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Name.Family() != testCase.family || result.Base.String() != "origin/release/2.8.0" {
				t.Fatalf("CreateReleaseStabilization() = %#v", result)
			}
		})
	}

	service := newReleaseService(&fakeGitRepository{}, nil)
	_, err := service.CreateReleaseStabilization(context.Background(), CreateReleaseStabilizationRequest{
		Repository: testRepository(),
		Release:    mustBranch("release/2.8.0"),
		Ticket:     mustTicket("ABC-999"),
		Slug:       mustSlug("new-feature"),
		Kind:       "feature",
	})
	assertProblemCode(t, err, problem.CodeInvalidInput)
}

func TestReleasePromotionAndSupportProvenance(t *testing.T) {
	t.Parallel()

	t.Run("release promotion targets main", func(t *testing.T) {
		publisher := &fakePublisher{result: port.PublishedPullRequest{URL: "https://example.invalid/pr/release"}}
		service := newReleaseService(&fakeGitRepository{}, publisher)
		result, err := service.PrepareReleasePromotion(context.Background(), PrepareReleasePromotionRequest{
			Repository: testRepository(),
			Release:    mustBranch("release/2.8.0"),
		})
		if err != nil || result.PullRequest.Target.String() != "main" || result.PublishedURL == "" {
			t.Fatalf("PrepareReleasePromotion() = (%#v, %v)", result, err)
		}
	})

	t.Run("support requires matching release tag", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true, releaseTags: []string{"v2.9.0"}}
		service := newReleaseService(git, nil)
		_, err := service.PrepareSupport(context.Background(), PrepareSupportRequest{
			Repository: testRepository(),
			Version:    mustSupportVersion(t, "2.8"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})
}

func TestCleanupBranchLimitsDeletionToPermittedLocalBranches(t *testing.T) {
	t.Parallel()

	t.Run("official branches are never local cleanup targets", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := newReleaseService(git, nil)
		_, err := service.CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     mustBranch("hotfix/ABC-999-payment-timeout"),
		})
		assertProblemCode(t, err, problem.CodeInvalidInput)
		if len(git.calls) != 0 {
			t.Fatalf("official cleanup must not call Git: %v", git.calls)
		}
	})

	t.Run("scratch cleanup is private and direct", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := newReleaseService(git, nil)
		result, err := service.CleanupBranch(context.Background(), CleanupBranchRequest{
			Repository: testRepository(),
			Branch:     mustBranch("scratch/ABC-123-experiment"),
		})
		if err != nil || !result.DeletedLocal || !result.MetadataCleared {
			t.Fatalf("CleanupBranch() = (%#v, %v)", result, err)
		}
	})

	t.Run("shared and release lines are protected", func(t *testing.T) {
		service := newReleaseService(&fakeGitRepository{}, nil)
		for _, name := range []string{"develop", "release/2.8.0", "support/2.8"} {
			_, err := service.CleanupBranch(context.Background(), CleanupBranchRequest{
				Repository: testRepository(),
				Branch:     mustBranch(name),
			})
			assertProblemCode(t, err, problem.CodeInvalidInput)
		}
	})
}

func TestPropagateHotfixUsesCherryPickProvenanceAndTargetLine(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		hasCommits:  true,
		clean:       true,
		publication: branch.PublicationUnpublished,
		messages:    []string{"fix(ABC-999): resolve payment timeout"},
	}
	service := newReleaseService(git, nil)
	commitID := strings.Repeat("a", 40)

	result, err := service.PropagateHotfix(context.Background(), PropagateHotfixRequest{
		Repository: testRepository(),
		Source:     mustBranch("hotfix/ABC-999-payment-timeout"),
		TargetLine: mustBranch("support/2.7"),
		CommitID:   commitID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.CherryPicked || len(git.cherryPicked) != 1 || git.cherryPicked[0] != commitID {
		t.Fatalf("cherry-pick result = %#v, commits=%v", result, git.cherryPicked)
	}
	if result.Branch.Name.String() != "fix/ABC-999-forward-port-payment-timeout" {
		t.Fatalf("propagation branch = %q", result.Branch.Name.String())
	}
	if result.Publication.PullRequest.Target.String() != "support/2.7" {
		t.Fatalf("propagation target = %q", result.Publication.PullRequest.Target.String())
	}
}

func TestPropagateHotfixRejectsInvalidRequestsBeforeMutation(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{}
	service := newReleaseService(git, nil)
	_, err := service.PropagateHotfix(context.Background(), PropagateHotfixRequest{
		Repository: testRepository(),
		Source:     mustBranch("feature/ABC-999-payment-timeout"),
		TargetLine: mustBranch("develop"),
		CommitID:   strings.Repeat("a", 40),
	})
	assertProblemCode(t, err, problem.CodeInvalidInput)
	if len(git.calls) != 0 {
		t.Fatalf("invalid propagation must not call Git: %v", git.calls)
	}
}

func newTicketService(git *fakeGitRepository, quality port.QualityRunner, publisher port.PullRequestPublisher) *TicketService {
	keys := &fakeKeyPolicy{}
	branches := branchapp.NewService(git, keys)
	sync := branchapp.NewSynchronizer(git, branches, quality)
	return NewTicketService(branches, sync, git, quality, publisher)
}

func newReleaseService(git *fakeGitRepository, publisher port.PullRequestPublisher) *ReleaseService {
	branches := branchapp.NewService(git, &fakeKeyPolicy{})
	sync := branchapp.NewSynchronizer(git, branches, nil)
	tickets := NewTicketService(branches, sync, git, nil, publisher)
	return NewReleaseService(branches, git, publisher).WithTicketService(tickets)
}

func testRepository() port.RepositoryIdentity {
	return port.RepositoryIdentity{Root: "C:/repo", Remote: "origin"}
}

func mustBranch(raw string) branch.BranchName {
	value, err := branch.ParseName(raw)
	if err != nil {
		panic(err)
	}
	return value
}

func mustBase(remote, raw string) branch.TargetBase {
	base, err := branch.NewTargetBase(remote, mustBranch(raw))
	if err != nil {
		panic(err)
	}
	return base
}

func mustTicket(raw string) ticket.ID {
	value, err := ticket.ParseID(raw)
	if err != nil {
		panic(err)
	}
	return value
}

func mustSlug(raw string) branch.Slug {
	value, err := branch.ParseSlug(raw)
	if err != nil {
		panic(err)
	}
	return value
}

func mustReleaseVersion(t *testing.T, raw string) branch.SemanticVersion {
	t.Helper()
	value, err := branch.ParseSemanticVersion(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func mustSupportVersion(t *testing.T, raw string) branch.SupportVersion {
	t.Helper()
	value, err := branch.ParseSupportVersion(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func countCall(calls []string, expected string) int {
	count := 0
	for _, actual := range calls {
		if actual == expected {
			count++
		}
	}
	return count
}

func assertProblemCode(t *testing.T, err error, expected problem.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected problem code %q, got nil", expected)
	}
	actual, ok := problem.As(err)
	if !ok {
		t.Fatalf("error %T does not carry a problem: %v", err, err)
	}
	if actual.Code != expected {
		t.Fatalf("problem code = %q, want %q", actual.Code, expected)
	}
}

var _ port.GitRepository = (*fakeGitRepository)(nil)
var _ port.KeyPolicy = (*fakeKeyPolicy)(nil)
var _ port.QualityRunner = (*fakeQualityRunner)(nil)
var _ port.PullRequestPublisher = (*fakePublisher)(nil)
