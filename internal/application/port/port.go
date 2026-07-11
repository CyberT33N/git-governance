// Package port contains outbound contracts owned by the application layer.
// Adapters implement these contracts; domain packages never import them.
package port

import (
	"context"

	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

// RepositoryIdentity identifies the local repository and its selected remote.
type RepositoryIdentity struct {
	Root   string
	Remote string
}

// PushUpdateInspection contains Git-derived facts about one outgoing branch
// update. The application supplies the exact object IDs received from the
// pre-push hook; the adapter never infers them from the checked-out branch.
type PushUpdateInspection struct {
	MissingBaseCommits bool
	FastForward        bool
	CommitMessages     []string
}

// GitRepository is the explicit boundary for Git process operations. It is not
// a generic persistence repository.
type GitRepository interface {
	Discover(ctx context.Context, directory string) (RepositoryIdentity, error)
	Version(ctx context.Context) (string, error)
	RemoteURL(ctx context.Context, repository RepositoryIdentity) (string, error)
	ActiveOperation(ctx context.Context, repository RepositoryIdentity) (string, bool, error)
	HasCommits(ctx context.Context, repository RepositoryIdentity) (bool, error)
	IsWorktreeClean(ctx context.Context, repository RepositoryIdentity) (bool, error)
	CurrentBranch(ctx context.Context, repository RepositoryIdentity) (branch.BranchName, error)
	ValidateBranchRef(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) error
	BranchExists(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) (bool, error)
	OfficialBranchesForTicket(ctx context.Context, repository RepositoryIdentity, id ticket.ID) ([]branch.BranchName, error)
	Fetch(ctx context.Context, repository RepositoryIdentity) error
	TargetBaseExists(ctx context.Context, repository RepositoryIdentity, base branch.TargetBase) (bool, error)
	CreateBranch(ctx context.Context, repository RepositoryIdentity, name branch.BranchName, base branch.TargetBase, switchTo bool) error
	StoreWorkflowBase(ctx context.Context, repository RepositoryIdentity, name branch.BranchName, base branch.TargetBase) error
	ClearWorkflowBase(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) error
	WorkflowBase(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) (branch.TargetBase, bool, error)
	SwitchBranch(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) error
	PublicationState(ctx context.Context, repository RepositoryIdentity, name branch.BranchName) (branch.PublicationState, error)
	HasMissingBaseCommits(ctx context.Context, repository RepositoryIdentity, base branch.TargetBase) (bool, error)
	CommitMessagesSince(ctx context.Context, repository RepositoryIdentity, base branch.TargetBase) ([]string, error)
	Rebase(ctx context.Context, repository RepositoryIdentity, base branch.TargetBase) error
	Merge(ctx context.Context, repository RepositoryIdentity, base branch.TargetBase, message commitmsg.Message) error
	CherryPick(ctx context.Context, repository RepositoryIdentity, commitID string) error
	DeleteLocalBranch(ctx context.Context, repository RepositoryIdentity, name branch.BranchName, force bool) error
	ReleaseTagsAt(ctx context.Context, repository RepositoryIdentity, revision string) ([]string, error)
	HasStagedChanges(ctx context.Context, repository RepositoryIdentity) (bool, error)
	Stage(ctx context.Context, repository RepositoryIdentity, paths []string) error
	Commit(ctx context.Context, repository RepositoryIdentity, message commitmsg.Message) error
	Push(ctx context.Context, repository RepositoryIdentity, name branch.BranchName, setUpstream bool) error
	InspectPushUpdate(
		ctx context.Context,
		repository RepositoryIdentity,
		base branch.TargetBase,
		localObjectID string,
		remoteObjectID string,
	) (PushUpdateInspection, error)
}

// KeyPolicy validates a syntactically valid key against the active local
// policy. The first implementation only checks syntax; a bundle adapter can
// add repository authorization later.
type KeyPolicy interface {
	ValidateKey(ctx context.Context, repository RepositoryIdentity, key ticket.Key) error
}

// PolicyStatus describes the active local policy mode without exposing policy
// implementation details to diagnostics consumers.
type PolicyStatus struct {
	Mode          string
	BundlePresent bool
	BundleFresh   bool
	Detail        string
}

// PolicyInspector reports read-only status for the active policy adapter.
type PolicyInspector interface {
	Status(ctx context.Context, repository RepositoryIdentity) (PolicyStatus, error)
}

// ToolInspector performs bounded read-only diagnostics for external tools and
// repository-local configuration files.
type ToolInspector interface {
	Platform() (operatingSystem string, architecture string)
	Version(ctx context.Context, executable string) (string, error)
	FileExists(path string) (bool, error)
}

// Preferences are user-scoped UX preferences, never organizational policy or
// secrets.
type Preferences struct {
	SchemaVersion int
	KnownKeys     []ticket.Key
	DefaultKey    *ticket.Key
	Accessible    bool
}

// PreferencesStore persists user-scoped preferences.
type PreferencesStore interface {
	Load(ctx context.Context) (Preferences, error)
	Save(ctx context.Context, preferences Preferences) error
}

// Prompt is the inbound interactive terminal boundary.
type Prompt interface {
	Input(ctx context.Context, request InputRequest) (string, error)
	Select(ctx context.Context, request SelectRequest) (string, error)
	Confirm(ctx context.Context, request ConfirmRequest) (bool, error)
}

// InputValidator validates one interactive input candidate. It returns an
// error that the terminal adapter can render before asking again.
type InputValidator func(string) error

// InputRequest describes an explanatory text input.
type InputRequest struct {
	Label       string
	Description string
	Default     string
	Required    bool
	Validate    InputValidator
	Sensitive   bool
}

// SelectRequest describes an explanatory single-value choice.
type SelectRequest struct {
	Label       string
	Description string
	Options     []SelectOption
	Default     string
}

// SelectOption is a stable machine value and a human-facing explanation.
type SelectOption struct {
	Value       string
	Label       string
	Description string
}

// ConfirmRequest describes a consequential action requiring explicit consent.
type ConfirmRequest struct {
	Label       string
	Description string
	Default     bool
}

// Report is the delivery-neutral result of a use case.
type Report struct {
	Operation string
	Summary   string
	Fields    map[string]string
	Data      any
	Problem   *problem.Problem
}

// Reporter renders application results as human or machine output.
type Reporter interface {
	Report(ctx context.Context, result Report) error
}

// QualityStatus classifies whether repository-local quality gates were
// configured and successfully executed.
type QualityStatus string

const (
	QualityUnconfigured QualityStatus = "unconfigured"
	QualitySkipped      QualityStatus = "skipped"
	QualityPassed       QualityStatus = "passed"
)

// QualityGateResult records one successfully completed configured gate without
// exposing its process output, which may contain sensitive project data.
type QualityGateResult struct {
	Name string
}

// QualityResult reports the actual quality-gate outcome. An unconfigured
// repository is not reported as a passed suite.
type QualityResult struct {
	Status QualityStatus
	Detail string
	Gates  []QualityGateResult
}

// QualityRequest identifies the branch families relevant to one
// publication-affecting operation. The runner evaluates each configured gate
// once against this complete set, so a multi-ref push cannot duplicate work.
type QualityRequest struct {
	Families []branch.Family
}

// QualityRunner runs repository-defined local quality gates before a
// publication-affecting operation.
type QualityRunner interface {
	Run(ctx context.Context, repository RepositoryIdentity, request QualityRequest) (QualityResult, error)
}

// PullRequest describes a provider-neutral pull request intent.
type PullRequest struct {
	Source branch.BranchName
	Target branch.BranchName
	Ticket ticket.ID
	Title  string
	Draft  bool
}

// PublishedPullRequest represents an optional provider-specific result.
type PublishedPullRequest struct {
	URL string
}

// PullRequestPublisher is optional: the initial product emits provider-neutral
// pull request data, while later adapters can publish it to a configured host.
type PullRequestPublisher interface {
	Publish(ctx context.Context, request PullRequest) (PublishedPullRequest, error)
}
