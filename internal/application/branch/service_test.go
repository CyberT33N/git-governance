package branchapp

import (
	"context"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	domainbranch "github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

type fakeGitRepository struct {
	hasCommits    bool
	clean         bool
	exists        bool
	publication   domainbranch.PublicationState
	missingBase   bool
	err           error
	inspectionErr error
	inspections   []port.PushUpdateInspection
	official      []domainbranch.BranchName
	workflowBase  *domainbranch.TargetBase
	calls         []string
	createdName   domainbranch.BranchName
	createdBase   domainbranch.TargetBase
	createdSwitch bool
	mergedMessage commitmsg.Message
}

func (fake *fakeGitRepository) Discover(context.Context, string) (port.RepositoryIdentity, error) {
	fake.calls = append(fake.calls, "discover")
	return port.RepositoryIdentity{Root: "C:/repo", Remote: "origin"}, fake.err
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

func (fake *fakeGitRepository) CurrentBranch(context.Context, port.RepositoryIdentity) (domainbranch.BranchName, error) {
	fake.calls = append(fake.calls, "current-branch")
	return mustBranch("feature/ABC-123-add-export"), fake.err
}

func (fake *fakeGitRepository) ValidateBranchRef(context.Context, port.RepositoryIdentity, domainbranch.BranchName) error {
	fake.calls = append(fake.calls, "validate-ref")
	return fake.err
}

func (fake *fakeGitRepository) BranchExists(context.Context, port.RepositoryIdentity, domainbranch.BranchName) (bool, error) {
	fake.calls = append(fake.calls, "branch-exists")
	return fake.exists, fake.err
}

func (fake *fakeGitRepository) OfficialBranchesForTicket(_ context.Context, _ port.RepositoryIdentity, _ ticket.ID) ([]domainbranch.BranchName, error) {
	fake.calls = append(fake.calls, "official-branches-for-ticket")
	return append([]domainbranch.BranchName(nil), fake.official...), fake.err
}

func (fake *fakeGitRepository) Fetch(context.Context, port.RepositoryIdentity) error {
	fake.calls = append(fake.calls, "fetch")
	return fake.err
}

func (fake *fakeGitRepository) CreateBranch(_ context.Context, _ port.RepositoryIdentity, name domainbranch.BranchName, base domainbranch.TargetBase, switchTo bool) error {
	fake.calls = append(fake.calls, "create-branch")
	fake.createdName = name
	fake.createdBase = base
	fake.createdSwitch = switchTo
	return fake.err
}

func (fake *fakeGitRepository) StoreWorkflowBase(_ context.Context, _ port.RepositoryIdentity, _ domainbranch.BranchName, base domainbranch.TargetBase) error {
	fake.calls = append(fake.calls, "store-workflow-base")
	copy := base
	fake.workflowBase = &copy
	return fake.err
}

func (fake *fakeGitRepository) WorkflowBase(context.Context, port.RepositoryIdentity, domainbranch.BranchName) (domainbranch.TargetBase, bool, error) {
	fake.calls = append(fake.calls, "workflow-base")
	if fake.workflowBase == nil {
		return domainbranch.TargetBase{}, false, fake.err
	}
	return *fake.workflowBase, true, fake.err
}

func (fake *fakeGitRepository) SwitchBranch(context.Context, port.RepositoryIdentity, domainbranch.BranchName) error {
	fake.calls = append(fake.calls, "switch")
	return fake.err
}

func (fake *fakeGitRepository) PublicationState(context.Context, port.RepositoryIdentity, domainbranch.BranchName) (domainbranch.PublicationState, error) {
	fake.calls = append(fake.calls, "publication-state")
	return fake.publication, fake.err
}

func (fake *fakeGitRepository) HasMissingBaseCommits(context.Context, port.RepositoryIdentity, domainbranch.TargetBase) (bool, error) {
	fake.calls = append(fake.calls, "missing-base")
	return fake.missingBase, fake.err
}

func (fake *fakeGitRepository) CommitMessagesSince(context.Context, port.RepositoryIdentity, domainbranch.TargetBase) ([]string, error) {
	fake.calls = append(fake.calls, "commit-messages")
	return nil, fake.err
}

func (fake *fakeGitRepository) Rebase(context.Context, port.RepositoryIdentity, domainbranch.TargetBase) error {
	fake.calls = append(fake.calls, "rebase")
	return fake.err
}

func (fake *fakeGitRepository) Merge(_ context.Context, _ port.RepositoryIdentity, _ domainbranch.TargetBase, message commitmsg.Message) error {
	fake.calls = append(fake.calls, "merge")
	fake.mergedMessage = message
	return fake.err
}

func (fake *fakeGitRepository) CherryPick(context.Context, port.RepositoryIdentity, string) error {
	fake.calls = append(fake.calls, "cherry-pick")
	return fake.err
}

func (fake *fakeGitRepository) DeleteLocalBranch(context.Context, port.RepositoryIdentity, domainbranch.BranchName, bool) error {
	fake.calls = append(fake.calls, "delete-local-branch")
	return fake.err
}

func (fake *fakeGitRepository) DeleteRemoteBranch(context.Context, port.RepositoryIdentity, domainbranch.BranchName) error {
	fake.calls = append(fake.calls, "delete-remote-branch")
	return fake.err
}

func (fake *fakeGitRepository) ReleaseTagsAt(context.Context, port.RepositoryIdentity, string) ([]string, error) {
	fake.calls = append(fake.calls, "release-tags")
	return nil, fake.err
}

func (fake *fakeGitRepository) HasStagedChanges(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "staged")
	return false, fake.err
}

func (fake *fakeGitRepository) Stage(context.Context, port.RepositoryIdentity, []string) error {
	fake.calls = append(fake.calls, "stage")
	return fake.err
}

func (fake *fakeGitRepository) Commit(context.Context, port.RepositoryIdentity, commitmsg.Message) error {
	fake.calls = append(fake.calls, "commit")
	return fake.err
}

func (fake *fakeGitRepository) Push(context.Context, port.RepositoryIdentity, domainbranch.BranchName, bool) error {
	fake.calls = append(fake.calls, "push")
	return fake.err
}

func (fake *fakeGitRepository) InspectPushUpdate(context.Context, port.RepositoryIdentity, domainbranch.TargetBase, string, string) (port.PushUpdateInspection, error) {
	fake.calls = append(fake.calls, "inspect-push")
	if fake.inspectionErr != nil {
		return port.PushUpdateInspection{}, fake.inspectionErr
	}
	if len(fake.inspections) == 0 {
		return port.PushUpdateInspection{}, nil
	}
	inspection := fake.inspections[0]
	fake.inspections = fake.inspections[1:]
	return inspection, nil
}

type fakeKeyPolicy struct {
	err  error
	keys []string
}

func (fake *fakeKeyPolicy) ValidateKey(_ context.Context, _ port.RepositoryIdentity, key ticket.Key) error {
	fake.keys = append(fake.keys, key.String())
	return fake.err
}

type fakeQualityRunner struct {
	calls int
	err   error
}

func (fake *fakeQualityRunner) Run(context.Context, port.RepositoryIdentity) (port.QualityResult, error) {
	fake.calls++
	return port.QualityResult{Status: port.QualityPassed}, fake.err
}

func TestListFamiliesIsCompleteAndIndependent(t *testing.T) {
	t.Parallel()

	actual := ListFamilies()
	if len(actual) != 13 {
		t.Fatalf("ListFamilies() length = %d, want 13", len(actual))
	}
	if actual[0].Family != domainbranch.FamilyMain || actual[len(actual)-1].Family != domainbranch.FamilyScratch {
		t.Fatalf("ListFamilies() = %#v", actual)
	}
	actual[0].Label = "changed"
	if ListFamilies()[0].Label == "changed" {
		t.Fatal("ListFamilies returned shared backing storage")
	}
}

func TestValidateAllowsSharedLinesButChecksRefAndPolicy(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{}
	keys := &fakeKeyPolicy{}
	service := NewService(git, keys)

	main := mustBranch("main")
	if _, err := service.Validate(context.Background(), ValidateRequest{Repository: testRepository(), Name: main}); err != nil {
		t.Fatalf("Validate(main) error = %v", err)
	}
	if got := strings.Join(git.calls, ","); got != "validate-ref" {
		t.Fatalf("calls = %q", got)
	}

	feature := mustBranch("feature/ABC-123-add-export")
	git.calls = nil
	if _, err := service.Validate(context.Background(), ValidateRequest{Repository: testRepository(), Name: feature}); err != nil {
		t.Fatalf("Validate(feature) error = %v", err)
	}
	if got := strings.Join(keys.keys, ","); got != "ABC" {
		t.Fatalf("policy keys = %q", got)
	}
	if got := strings.Join(git.calls, ","); got != "validate-ref" {
		t.Fatalf("calls = %q", got)
	}
}

func TestCreateRegularBranch(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{hasCommits: true, clean: true}
	keys := &fakeKeyPolicy{}
	service := NewService(git, keys)
	request := regularRequest()

	actual, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Name.String() != "feature/ABC-123-add-export" || actual.Base.String() != "origin/develop" || !actual.Switched {
		t.Fatalf("Create() = %#v", actual)
	}
	if git.createdName.String() != actual.Name.String() || git.createdBase.String() != "origin/develop" || !git.createdSwitch {
		t.Fatalf("create call = (%q, %q, %t)", git.createdName, git.createdBase, git.createdSwitch)
	}
	expectedCalls := "validate-ref,has-commits,worktree-clean,fetch,branch-exists,official-branches-for-ticket,create-branch"
	if got := strings.Join(git.calls, ","); got != expectedCalls {
		t.Fatalf("calls = %q, want %q", got, expectedCalls)
	}
}

func TestCreateDryRunNeverMutates(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{hasCommits: true, clean: true}
	service := NewService(git, &fakeKeyPolicy{})
	request := regularRequest()
	request.DryRun = true
	switchTo := false
	request.Switch = &switchTo

	actual, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !actual.DryRun || actual.Switched || len(actual.Plan) != 2 {
		t.Fatalf("Create() = %#v", actual)
	}
	if got := strings.Join(git.calls, ","); got != "validate-ref,has-commits,branch-exists,official-branches-for-ticket" {
		t.Fatalf("dry-run calls = %q", got)
	}
}

func TestCreateStopsBeforeMutationOnInvalidState(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		git     *fakeGitRepository
		request CreateRequest
		code    problem.Code
		calls   string
	}{
		{
			name:    "no commits",
			git:     &fakeGitRepository{hasCommits: false},
			request: regularRequest(),
			code:    problem.CodeRepositoryHasNoCommits,
			calls:   "validate-ref,has-commits",
		},
		{
			name:    "branch exists",
			git:     &fakeGitRepository{hasCommits: true, clean: true, exists: true},
			request: regularRequest(),
			code:    problem.CodeBranchAlreadyExists,
			calls:   "validate-ref,has-commits,worktree-clean,fetch,branch-exists",
		},
		{
			name:    "dirty worktree",
			git:     &fakeGitRepository{hasCommits: true, clean: false},
			request: regularRequest(),
			code:    problem.CodeWorktreeNotClean,
			calls:   "validate-ref,has-commits,worktree-clean",
		},
		{
			name:    "hotfix outside workflow",
			git:     &fakeGitRepository{},
			request: hotfixRequest(false),
			code:    problem.CodeBranchFamilyInvalid,
			calls:   "",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(testCase.git, &fakeKeyPolicy{})
			_, err := service.Create(context.Background(), testCase.request)
			assertProblemCode(t, err, testCase.code)
			if got := strings.Join(testCase.git.calls, ","); got != testCase.calls {
				t.Fatalf("calls = %q, want %q", got, testCase.calls)
			}
		})
	}
}

func TestCreateSpecialBranchesRequireCorrectBases(t *testing.T) {
	t.Parallel()

	t.Run("workflow hotfix from main", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true}
		service := NewService(git, &fakeKeyPolicy{})
		actual, err := service.Create(context.Background(), hotfixRequest(true))
		if err != nil {
			t.Fatal(err)
		}
		if actual.Base.String() != "origin/main" {
			t.Fatalf("hotfix base = %q", actual.Base.String())
		}
	})

	t.Run("scratch requires official base", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{})
		request := regularRequest()
		request.Family = domainbranch.FamilyScratch
		request.Base = nil
		_, err := service.Create(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
	})

	t.Run("scratch uses matching local official base", func(t *testing.T) {
		git := &fakeGitRepository{hasCommits: true, clean: true}
		service := NewService(git, &fakeKeyPolicy{})
		request := regularRequest()
		request.Family = domainbranch.FamilyScratch
		request.Slug = mustSlug("exploration")
		base, err := domainbranch.NewLocalBase(mustBranch("feature/ABC-123-add-export"))
		if err != nil {
			t.Fatal(err)
		}
		request.Base = &base
		actual, err := service.Create(context.Background(), request)
		if err != nil || actual.Base.String() != "feature/ABC-123-add-export" {
			t.Fatalf("Create() = (%#v, %v)", actual, err)
		}
	})

	t.Run("scratch rejects a different ticket base", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{})
		request := regularRequest()
		request.Family = domainbranch.FamilyScratch
		request.Slug = mustSlug("exploration")
		base, err := domainbranch.NewLocalBase(mustBranch("feature/ABC-124-other-ticket"))
		if err != nil {
			t.Fatal(err)
		}
		request.Base = &base
		_, err = service.Create(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
	})

	t.Run("scratch rejects a remote tracking base", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{})
		request := regularRequest()
		request.Family = domainbranch.FamilyScratch
		request.Slug = mustSlug("exploration")
		base := mustBase("origin", "feature/ABC-123-add-export")
		request.Base = &base

		_, err := service.Create(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
		if len(git.calls) != 0 {
			t.Fatalf("remote scratch base must stop before Git calls: %v", git.calls)
		}
	})
}

func TestCreateRejectsAnotherOfficialRegularBranchForTheSameTicket(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		hasCommits: true,
		clean:      true,
		official:   []domainbranch.BranchName{mustBranch("fix/ABC-123-existing-ticket-work")},
	}
	service := NewService(git, &fakeKeyPolicy{})

	_, err := service.Create(context.Background(), regularRequest())
	assertProblemCode(t, err, problem.CodeTicketBranchAlreadyExists)
	if strings.Contains(strings.Join(git.calls, ","), "create-branch") {
		t.Fatalf("duplicate ticket must stop before branch creation: %v", git.calls)
	}
}

func TestCreateAllowsWorkflowManagedBranchesToReuseTicketAcrossActiveLines(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		hasCommits: true,
		clean:      true,
		official:   []domainbranch.BranchName{mustBranch("feature/ABC-123-existing-ticket-work")},
	}
	service := NewService(git, &fakeKeyPolicy{})
	base := mustBase("origin", "release/2.8.0")
	request := CreateRequest{
		Repository:      testRepository(),
		Family:          domainbranch.FamilyFix,
		Ticket:          mustTicket("ABC-123"),
		Slug:            mustSlug("release-blocker"),
		Base:            &base,
		WorkflowManaged: true,
	}

	result, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name.String() != "fix/ABC-123-release-blocker" {
		t.Fatalf("Create() branch = %q", result.Name.String())
	}
	if strings.Contains(strings.Join(git.calls, ","), "official-branches-for-ticket") {
		t.Fatalf("workflow-managed branch must not apply regular-ticket exclusivity: %v", git.calls)
	}
}

func TestSyncPolicy(t *testing.T) {
	t.Parallel()

	name := mustBranch("feature/ABC-123-add-export")
	base := mustBase("origin", "develop")
	message := mustMessage(t, "chore(ABC-123): merge origin/develop")

	t.Run("up to date check", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationUnpublished,
			clean:       true,
			missingBase: false,
		}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		actual, err := sync.Sync(context.Background(), SyncRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
			Strategy:   SyncCheck,
		})
		if err != nil || actual.RecommendedAction != "none" || actual.Mutated {
			t.Fatalf("Sync() = (%#v, %v)", actual, err)
		}
	})

	t.Run("unpublished auto rebase", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationUnpublished,
			clean:       true,
			missingBase: true,
		}
		quality := &fakeQualityRunner{}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), quality)
		actual, err := sync.Sync(context.Background(), SyncRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
			Strategy:   SyncAuto,
		})
		if err != nil || !actual.Mutated || actual.RecommendedAction != "rebased" || quality.calls != 1 {
			t.Fatalf("Sync() = (%#v, %v), quality=%d", actual, err, quality.calls)
		}
		if !strings.Contains(strings.Join(git.calls, ","), "rebase") {
			t.Fatalf("calls = %v", git.calls)
		}
	})

	t.Run("published auto recommends merge", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationPublished,
			clean:       true,
			missingBase: true,
		}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		actual, err := sync.Sync(context.Background(), SyncRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
			Strategy:   SyncAuto,
		})
		if err != nil || actual.Mutated || actual.RecommendedAction != "merge" {
			t.Fatalf("Sync() = (%#v, %v)", actual, err)
		}
		if strings.Contains(strings.Join(git.calls, ","), "merge") {
			t.Fatalf("auto must not merge published branch: %v", git.calls)
		}
	})

	t.Run("published rebase is forbidden", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationPublished,
			clean:       true,
			missingBase: true,
		}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := sync.Sync(context.Background(), SyncRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
			Strategy:   SyncRebase,
		})
		assertProblemCode(t, err, problem.CodeRebaseAfterPublishForbidden)
		if strings.Contains(strings.Join(git.calls, ","), "rebase") {
			t.Fatalf("published branch must not rebase: %v", git.calls)
		}
	})

	t.Run("published merge", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationPublished,
			clean:       true,
			missingBase: true,
		}
		quality := &fakeQualityRunner{}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), quality)
		actual, err := sync.Sync(context.Background(), SyncRequest{
			Repository:   testRepository(),
			Name:         name,
			Base:         &base,
			Strategy:     SyncMerge,
			MergeMessage: &message,
		})
		if err != nil || !actual.Mutated || actual.RecommendedAction != "merged" || quality.calls != 1 {
			t.Fatalf("Sync() = (%#v, %v), quality=%d", actual, err, quality.calls)
		}
		if git.mergedMessage.String() != message.String() {
			t.Fatalf("merge message = %q", git.mergedMessage.String())
		}
	})
}

func TestValidatePrePush(t *testing.T) {
	t.Parallel()

	name := mustBranch("feature/ABC-123-add-export")
	base := mustBase("origin", "develop")

	t.Run("first push with missing base is blocked without rebase", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationUnpublished,
			missingBase: true,
		}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := sync.ValidatePrePush(context.Background(), PrePushRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
		})
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
		if strings.Contains(strings.Join(git.calls, ","), "rebase") {
			t.Fatalf("pre-push must never rebase: %v", git.calls)
		}
	})

	t.Run("published branch allows validation", func(t *testing.T) {
		git := &fakeGitRepository{
			publication: domainbranch.PublicationPublished,
			missingBase: true,
		}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		actual, err := sync.ValidatePrePush(context.Background(), PrePushRequest{
			Repository: testRepository(),
			Name:       name,
			Base:       &base,
		})
		if err != nil || actual.Publication != domainbranch.PublicationPublished || !actual.MissingBaseCommits {
			t.Fatalf("ValidatePrePush() = (%#v, %v)", actual, err)
		}
	})

	t.Run("shared line is blocked", func(t *testing.T) {
		git := &fakeGitRepository{}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := sync.ValidatePrePush(context.Background(), PrePushRequest{
			Repository: testRepository(),
			Name:       mustBranch("develop"),
		})
		assertProblemCode(t, err, problem.CodeSharedLineMutationForbidden)
	})

	t.Run("scratch has no first-push base rule", func(t *testing.T) {
		git := &fakeGitRepository{}
		sync := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		actual, err := sync.ValidatePrePush(context.Background(), PrePushRequest{
			Repository: testRepository(),
			Name:       mustBranch("scratch/ABC-123-experiment"),
		})
		if err != nil || actual.Publication != domainbranch.PublicationUnknown {
			t.Fatalf("ValidatePrePush() = (%#v, %v)", actual, err)
		}
	})
}

func regularRequest() CreateRequest {
	return CreateRequest{
		Repository: testRepository(),
		Family:     domainbranch.FamilyFeature,
		Ticket:     mustTicket("ABC-123"),
		Slug:       mustSlug("add-export"),
	}
}

func hotfixRequest(workflowManaged bool) CreateRequest {
	base := mustBase("origin", "main")
	return CreateRequest{
		Repository:      testRepository(),
		Family:          domainbranch.FamilyHotfix,
		Ticket:          mustTicket("ABC-123"),
		Slug:            mustSlug("payment-timeout"),
		Base:            &base,
		WorkflowManaged: workflowManaged,
	}
}

func testRepository() port.RepositoryIdentity {
	return port.RepositoryIdentity{Root: "C:/repo", Remote: "origin"}
}

func mustBranch(raw string) domainbranch.BranchName {
	name, err := domainbranch.ParseName(raw)
	if err != nil {
		panic(err)
	}
	return name
}

func mustBase(remote, raw string) domainbranch.TargetBase {
	base, err := domainbranch.NewTargetBase(remote, mustBranch(raw))
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

func mustSlug(raw string) domainbranch.Slug {
	value, err := domainbranch.ParseSlug(raw)
	if err != nil {
		panic(err)
	}
	return value
}

func mustMessage(t *testing.T, raw string) commitmsg.Message {
	t.Helper()
	value, err := commitmsg.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
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
