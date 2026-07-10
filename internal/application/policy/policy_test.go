package policy

import (
	"context"
	"errors"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

type memoryStore struct {
	preferences port.Preferences
	err         error
}

func (store *memoryStore) Load(context.Context) (port.Preferences, error) {
	return store.preferences, store.err
}

func (store *memoryStore) Save(_ context.Context, preferences port.Preferences) error {
	if store.err != nil {
		return store.err
	}
	store.preferences = preferences
	return nil
}

type doctorGit struct {
	discoverErr error
	commits     bool
	commitErr   error
}

type doctorTools struct {
	version string
	exists  bool
	err     error
}

func (doctorTools) Platform() (string, string) {
	return "windows", "amd64"
}

func (tools doctorTools) Version(context.Context, string) (string, error) {
	if tools.err != nil {
		return "", tools.err
	}
	return tools.version, nil
}

func (tools doctorTools) FileExists(string) (bool, error) {
	if tools.err != nil {
		return false, tools.err
	}
	return tools.exists, nil
}

func (git doctorGit) Discover(context.Context, string) (port.RepositoryIdentity, error) {
	if git.discoverErr != nil {
		return port.RepositoryIdentity{}, git.discoverErr
	}
	return port.RepositoryIdentity{Root: "C:/repo", Remote: "origin"}, nil
}

func (doctorGit) Version(context.Context) (string, error) {
	return "git version test", nil
}

func (doctorGit) RemoteURL(context.Context, port.RepositoryIdentity) (string, error) {
	return "https://example.invalid/repo.git", nil
}

func (doctorGit) ActiveOperation(context.Context, port.RepositoryIdentity) (string, bool, error) {
	return "", false, nil
}

func (git doctorGit) HasCommits(context.Context, port.RepositoryIdentity) (bool, error) {
	return git.commits, git.commitErr
}

func (doctorGit) IsWorktreeClean(context.Context, port.RepositoryIdentity) (bool, error) {
	return true, nil
}

func (doctorGit) CurrentBranch(context.Context, port.RepositoryIdentity) (branch.BranchName, error) {
	return branch.ParseName("feature/ABC-123-add-export")
}

func (doctorGit) ValidateBranchRef(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	return nil
}

func (doctorGit) BranchExists(context.Context, port.RepositoryIdentity, branch.BranchName) (bool, error) {
	return false, nil
}

func (doctorGit) OfficialBranchesForTicket(context.Context, port.RepositoryIdentity, ticket.ID) ([]branch.BranchName, error) {
	return nil, nil
}

func (doctorGit) Fetch(context.Context, port.RepositoryIdentity) error {
	return nil
}

func (doctorGit) CreateBranch(context.Context, port.RepositoryIdentity, branch.BranchName, branch.TargetBase, bool) error {
	return nil
}

func (doctorGit) StoreWorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName, branch.TargetBase) error {
	return nil
}

func (doctorGit) ClearWorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	return nil
}

func (doctorGit) WorkflowBase(context.Context, port.RepositoryIdentity, branch.BranchName) (branch.TargetBase, bool, error) {
	return branch.TargetBase{}, false, nil
}

func (doctorGit) SwitchBranch(context.Context, port.RepositoryIdentity, branch.BranchName) error {
	return nil
}

func (doctorGit) PublicationState(context.Context, port.RepositoryIdentity, branch.BranchName) (branch.PublicationState, error) {
	return branch.PublicationUnpublished, nil
}

func (doctorGit) HasMissingBaseCommits(context.Context, port.RepositoryIdentity, branch.TargetBase) (bool, error) {
	return false, nil
}

func (doctorGit) CommitMessagesSince(context.Context, port.RepositoryIdentity, branch.TargetBase) ([]string, error) {
	return nil, nil
}

func (doctorGit) Rebase(context.Context, port.RepositoryIdentity, branch.TargetBase) error {
	return nil
}

func (doctorGit) Merge(context.Context, port.RepositoryIdentity, branch.TargetBase, commitmsg.Message) error {
	return nil
}

func (doctorGit) CherryPick(context.Context, port.RepositoryIdentity, string) error {
	return nil
}

func (doctorGit) DeleteLocalBranch(context.Context, port.RepositoryIdentity, branch.BranchName, bool) error {
	return nil
}

func (doctorGit) ReleaseTagsAt(context.Context, port.RepositoryIdentity, string) ([]string, error) {
	return nil, nil
}

func (doctorGit) HasStagedChanges(context.Context, port.RepositoryIdentity) (bool, error) {
	return false, nil
}

func (doctorGit) Stage(context.Context, port.RepositoryIdentity, []string) error {
	return nil
}

func (doctorGit) Commit(context.Context, port.RepositoryIdentity, commitmsg.Message) error {
	return nil
}

func (doctorGit) Push(context.Context, port.RepositoryIdentity, branch.BranchName, bool) error {
	return nil
}

func (doctorGit) InspectPushUpdate(context.Context, port.RepositoryIdentity, branch.TargetBase, string, string) (port.PushUpdateInspection, error) {
	return port.PushUpdateInspection{}, nil
}

func TestSyntaxOnlyKeyPolicy(t *testing.T) {
	t.Parallel()

	key := mustKey(t, "PLATFORM2")
	if err := (SyntaxOnlyKeyPolicy{}).ValidateKey(context.Background(), port.RepositoryIdentity{}, key); err != nil {
		t.Fatal(err)
	}
	if err := (SyntaxOnlyKeyPolicy{}).ValidateKey(context.Background(), port.RepositoryIdentity{}, ticket.Key{}); err == nil {
		t.Fatal("zero ticket key was accepted")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := (SyntaxOnlyKeyPolicy{}).ValidateKey(ctx, port.RepositoryIdentity{}, key)
	assertProblemCode(t, err, problem.CodeOperationCancelled)
}

func TestDescriptionIsSelfContained(t *testing.T) {
	t.Parallel()

	actual := Describe()
	if actual.SchemaVersion != schemaVersion || actual.KeyPolicy != "syntax-only" {
		t.Fatalf("Describe() = %#v", actual)
	}
	if len(actual.BranchFamilies) != 13 || len(actual.CommitTypes) != 11 {
		t.Fatalf("Describe() counts = %d families, %d types", len(actual.BranchFamilies), len(actual.CommitTypes))
	}
	if actual.Limits.TicketKeyMaximumLength != 32 || actual.Limits.CommitSubjectMaximumRunes != 200 {
		t.Fatalf("Describe() limits = %#v", actual.Limits)
	}
}

func TestPreferencesService(t *testing.T) {
	t.Parallel()

	store := &memoryStore{preferences: port.Preferences{SchemaVersion: 1}}
	service := NewPreferencesService(store)
	abc := mustKey(t, "ABC")
	platform := mustKey(t, "PLATFORM2")

	actual, err := service.AddKey(context.Background(), abc)
	if err != nil || len(actual.KnownKeys) != 1 {
		t.Fatalf("AddKey() = (%#v, %v)", actual, err)
	}
	actual, err = service.AddKey(context.Background(), abc)
	if err != nil || len(actual.KnownKeys) != 1 {
		t.Fatalf("duplicate AddKey() = (%#v, %v)", actual, err)
	}
	actual, err = service.SetDefaultKey(context.Background(), platform)
	if err != nil || actual.DefaultKey == nil || actual.DefaultKey.String() != "PLATFORM2" || len(actual.KnownKeys) != 2 {
		t.Fatalf("SetDefaultKey() = (%#v, %v)", actual, err)
	}
	actual, err = service.RemoveKey(context.Background(), platform)
	if err != nil || actual.DefaultKey != nil || len(actual.KnownKeys) != 1 {
		t.Fatalf("RemoveKey() = (%#v, %v)", actual, err)
	}
	_, err = service.RemoveKey(context.Background(), platform)
	assertProblemCode(t, err, problem.CodeInvalidInput)
}

func TestDoctorIsReadOnly(t *testing.T) {
	t.Parallel()

	store := &memoryStore{preferences: port.Preferences{SchemaVersion: 1}}
	tools := doctorTools{version: "lefthook 2.1.8", exists: true}
	result, err := NewDoctorServiceWithDependencies(doctorGit{commits: true}, store, SyntaxOnlyKeyPolicy{}, tools).Run(context.Background(), "C:/repo")
	if err != nil {
		t.Fatal(err)
	}
	if result.Repository.Root != "C:/repo" || len(result.Checks) != 10 {
		t.Fatalf("Doctor.Run() = %#v", result)
	}
	for _, check := range result.Checks {
		if !check.OK {
			t.Fatalf("unexpected failed doctor check: %#v", check)
		}
	}

	result, err = NewDoctorServiceWithDependencies(doctorGit{discoverErr: errors.New("not a repository")}, store, SyntaxOnlyKeyPolicy{}, tools).Run(context.Background(), "C:/missing")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Checks) == 0 || result.Checks[1].OK {
		t.Fatalf("failed discovery doctor result = %#v", result)
	}
}

func mustKey(t *testing.T, raw string) ticket.Key {
	t.Helper()
	actual, err := ticket.ParseKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	return actual
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

var _ port.GitRepository = doctorGit{}
var _ port.PreferencesStore = (*memoryStore)(nil)
var _ port.ToolInspector = doctorTools{}
