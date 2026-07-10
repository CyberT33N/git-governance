package commitapp

import (
	"context"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

type fakeGitRepository struct {
	staged      bool
	publication branch.PublicationState
	err         error
	calls       []string
	stagedPaths []string
	committed   commitmsg.Message
	pushedName  branch.BranchName
	setUpstream bool
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
	return true, fake.err
}

func (fake *fakeGitRepository) IsWorktreeClean(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "worktree-clean")
	return true, fake.err
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
	return false, fake.err
}

func (fake *fakeGitRepository) OfficialBranchesForTicket(context.Context, port.RepositoryIdentity, ticket.ID) ([]branch.BranchName, error) {
	fake.calls = append(fake.calls, "official-branches-for-ticket")
	return nil, fake.err
}

func (fake *fakeGitRepository) Fetch(context.Context, port.RepositoryIdentity) error {
	fake.calls = append(fake.calls, "fetch")
	return fake.err
}

func (fake *fakeGitRepository) CreateBranch(context.Context, port.RepositoryIdentity, branch.BranchName, branch.TargetBase, bool) error {
	fake.calls = append(fake.calls, "create-branch")
	return fake.err
}

func (fake *fakeGitRepository) StoreWorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName, branch.TargetBase) error {
	fake.calls = append(fake.calls, "store-workflow-base")
	return fake.err
}

func (fake *fakeGitRepository) ClearWorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	fake.calls = append(fake.calls, "clear-workflow-base")
	return fake.err
}

func (fake *fakeGitRepository) WorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName) (branch.TargetBase, bool, error) {
	fake.calls = append(fake.calls, "workflow-base")
	return branch.TargetBase{}, false, fake.err
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
	return false, fake.err
}

func (fake *fakeGitRepository) CommitMessagesSince(context.Context, port.RepositoryIdentity, branch.TargetBase) ([]string, error) {
	fake.calls = append(fake.calls, "commit-messages")
	return nil, fake.err
}

func (fake *fakeGitRepository) Rebase(context.Context, port.RepositoryIdentity, branch.TargetBase) error {
	fake.calls = append(fake.calls, "rebase")
	return fake.err
}

func (fake *fakeGitRepository) Merge(context.Context, port.RepositoryIdentity, branch.TargetBase, commitmsg.Message) error {
	fake.calls = append(fake.calls, "merge")
	return fake.err
}

func (fake *fakeGitRepository) CherryPick(context.Context, port.RepositoryIdentity, string) error {
	fake.calls = append(fake.calls, "cherry-pick")
	return fake.err
}

func (fake *fakeGitRepository) DeleteLocalBranch(context.Context, port.RepositoryIdentity, branch.BranchName, bool) error {
	fake.calls = append(fake.calls, "delete-local-branch")
	return fake.err
}

func (fake *fakeGitRepository) ReleaseTagsAt(context.Context, port.RepositoryIdentity, string) ([]string, error) {
	fake.calls = append(fake.calls, "release-tags")
	return nil, fake.err
}

func (fake *fakeGitRepository) HasStagedChanges(context.Context, port.RepositoryIdentity) (bool, error) {
	fake.calls = append(fake.calls, "has-staged")
	return fake.staged, fake.err
}

func (fake *fakeGitRepository) Stage(_ context.Context, _ port.RepositoryIdentity, paths []string) error {
	fake.calls = append(fake.calls, "stage")
	fake.stagedPaths = append([]string(nil), paths...)
	return fake.err
}

func (fake *fakeGitRepository) Commit(_ context.Context, _ port.RepositoryIdentity, message commitmsg.Message) error {
	fake.calls = append(fake.calls, "commit")
	fake.committed = message
	return fake.err
}

func (fake *fakeGitRepository) Push(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName, setUpstream bool) error {
	fake.calls = append(fake.calls, "push")
	fake.pushedName = name
	fake.setUpstream = setUpstream
	return fake.err
}

func (fake *fakeGitRepository) InspectPushUpdate(context.Context, port.RepositoryIdentity, branch.TargetBase, string, string) (port.PushUpdateInspection, error) {
	fake.calls = append(fake.calls, "inspect-push")
	return port.PushUpdateInspection{}, fake.err
}

type fakeKeyPolicy struct {
	err  error
	keys []string
}

func (fake *fakeKeyPolicy) ValidateKey(_ context.Context, _ port.RepositoryIdentity, key ticket.Key) error {
	fake.keys = append(fake.keys, key.String())
	return fake.err
}

type fakePushValidator struct {
	err   error
	calls int
	name  branch.BranchName
	base  *branch.TargetBase
}

func (fake *fakePushValidator) ValidatePush(_ context.Context, _ port.RepositoryIdentity, name branch.BranchName, base *branch.TargetBase) error {
	fake.calls++
	fake.name = name
	fake.base = base
	return fake.err
}

func TestValidateCommit(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{}
	keys := &fakeKeyPolicy{}
	service := NewService(git, keys, nil)
	request := validRequest()

	actual, err := service.Validate(context.Background(), ValidateRequest{
		Repository: request.Repository,
		Branch:     request.Branch,
		Message:    request.Message,
	})
	if err != nil {
		t.Fatal(err)
	}
	if actual.Message.String() != request.Message.String() || strings.Join(keys.keys, ",") != "ABC" {
		t.Fatalf("Validate() = %#v, keys=%v", actual, keys.keys)
	}
	if strings.Join(git.calls, ",") != "validate-ref" {
		t.Fatalf("calls = %v", git.calls)
	}
}

func TestValidateCommitRejectsMismatchesAndSharedLines(t *testing.T) {
	t.Parallel()

	t.Run("ticket mismatch", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{}, nil)
		message := mustMessage("feat(ABC-124): add export")
		_, err := service.Validate(context.Background(), ValidateRequest{
			Repository: testRepository(),
			Branch:     mustBranch("feature/ABC-123-add-export"),
			Message:    message,
		})
		assertProblemCode(t, err, problem.CodeCommitTicketMismatch)
	})

	t.Run("shared line", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{}, nil)
		_, err := service.Validate(context.Background(), ValidateRequest{
			Repository: testRepository(),
			Branch:     mustBranch("develop"),
			Message:    mustMessage("feat(ABC-123): add export"),
		})
		assertProblemCode(t, err, problem.CodeSharedLineMutationForbidden)
		if len(git.calls) != 0 {
			t.Fatalf("shared line must stop before Git calls: %v", git.calls)
		}
	})
}

func TestCreateCommitWithExplicitStage(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{staged: true}
	keys := &fakeKeyPolicy{}
	service := NewService(git, keys, nil)
	request := validRequest()
	request.StagePaths = []string{"cmd/git-governance/main.go", "README.md"}

	actual, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if !actual.Committed || actual.Pushed {
		t.Fatalf("Create() = %#v", actual)
	}
	if got := strings.Join(git.stagedPaths, ","); got != "cmd/git-governance/main.go,README.md" {
		t.Fatalf("StagePaths = %q", got)
	}
	if git.committed.String() != request.Message.String() {
		t.Fatalf("committed message = %q", git.committed.String())
	}
	if got := strings.Join(git.calls, ","); got != "validate-ref,stage,has-staged,commit" {
		t.Fatalf("calls = %q", got)
	}
}

func TestCreateCommitDryRunAndNoStagedChanges(t *testing.T) {
	t.Parallel()

	t.Run("dry run avoids mutation", func(t *testing.T) {
		git := &fakeGitRepository{}
		service := NewService(git, &fakeKeyPolicy{}, nil)
		request := validRequest()
		request.StagePaths = []string{"README.md"}
		request.DryRun = true
		actual, err := service.Create(context.Background(), request)
		if err != nil || !actual.DryRun || actual.Committed || len(actual.Plan) != 2 {
			t.Fatalf("Create() = (%#v, %v)", actual, err)
		}
		if got := strings.Join(git.calls, ","); got != "validate-ref" {
			t.Fatalf("calls = %q", got)
		}
	})

	t.Run("no staged changes", func(t *testing.T) {
		git := &fakeGitRepository{staged: false}
		service := NewService(git, &fakeKeyPolicy{}, nil)
		_, err := service.Create(context.Background(), validRequest())
		assertProblemCode(t, err, problem.CodeInvalidInput)
		if got := strings.Join(git.calls, ","); got != "validate-ref,has-staged" {
			t.Fatalf("calls = %q", got)
		}
	})
}

func TestCreateCommitPushesOnlyAfterValidation(t *testing.T) {
	t.Parallel()

	t.Run("first push configures upstream", func(t *testing.T) {
		git := &fakeGitRepository{
			staged:      true,
			publication: branch.PublicationUnpublished,
		}
		validator := &fakePushValidator{}
		service := NewService(git, &fakeKeyPolicy{}, validator)
		request := validRequest()
		request.Push = true
		base := mustBase("origin", "develop")
		request.Base = &base

		actual, err := service.Create(context.Background(), request)
		if err != nil || !actual.Committed || !actual.Pushed || validator.calls != 1 {
			t.Fatalf("Create() = (%#v, %v), validation calls=%d", actual, err, validator.calls)
		}
		if !git.setUpstream || git.pushedName.String() != request.Branch.String() {
			t.Fatalf("push = (%q, upstream=%t)", git.pushedName, git.setUpstream)
		}
		if got := strings.Join(git.calls, ","); got != "validate-ref,has-staged,commit,publication,push" {
			t.Fatalf("calls = %q", got)
		}
	})

	t.Run("validation failure stops before publication and push", func(t *testing.T) {
		git := &fakeGitRepository{staged: true}
		validator := &fakePushValidator{err: errorsForTest("pre-push failed")}
		service := NewService(git, &fakeKeyPolicy{}, validator)
		request := validRequest()
		request.Push = true
		_, err := service.Create(context.Background(), request)
		if err == nil {
			t.Fatal("Create() unexpectedly succeeded")
		}
		if strings.Contains(strings.Join(git.calls, ","), "push") || strings.Contains(strings.Join(git.calls, ","), "publication") {
			t.Fatalf("push state must not be queried after validation failure: %v", git.calls)
		}
	})

	t.Run("unknown publication blocks push", func(t *testing.T) {
		git := &fakeGitRepository{staged: true, publication: branch.PublicationUnknown}
		validator := &fakePushValidator{}
		service := NewService(git, &fakeKeyPolicy{}, validator)
		request := validRequest()
		request.Push = true
		_, err := service.Create(context.Background(), request)
		assertProblemCode(t, err, problem.CodeBranchPublicationUnknown)
		if strings.Contains(strings.Join(git.calls, ","), "push") {
			t.Fatalf("unknown publication must not push: %v", git.calls)
		}
	})
}

func validRequest() CreateRequest {
	return CreateRequest{
		Repository: testRepository(),
		Branch:     mustBranch("feature/ABC-123-add-export"),
		Message:    mustMessage("feat(ABC-123): add export"),
	}
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

func mustMessage(raw string) commitmsg.Message {
	value, err := commitmsg.Parse(raw)
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

type errorsForTest string

func (value errorsForTest) Error() string {
	return string(value)
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
var _ PushValidator = (*fakePushValidator)(nil)
