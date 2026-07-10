package workflow

import (
	"context"
	"regexp"
	"strings"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

// ReleaseService owns the bounded hotfix, release, support, and release
// backmerge workflows.
type ReleaseService struct {
	branches  *branchapp.Service
	git       port.GitRepository
	publisher port.PullRequestPublisher
	tickets   *TicketService
}

var commitIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)

// NewReleaseService creates a release workflow service.
func NewReleaseService(branches *branchapp.Service, git port.GitRepository, publisher port.PullRequestPublisher) *ReleaseService {
	return &ReleaseService{
		branches:  branches,
		git:       git,
		publisher: publisher,
	}
}

// WithTicketService wires publication behavior into release workflows without
// making the release service depend on the CLI delivery layer.
func (service *ReleaseService) WithTicketService(tickets *TicketService) *ReleaseService {
	service.tickets = tickets
	return service
}

// StartHotfixRequest describes the affected line and ticket for a hotfix.
type StartHotfixRequest struct {
	Repository   port.RepositoryIdentity
	Ticket       ticket.ID
	Slug         branch.Slug
	AffectedLine branch.BranchName
	DryRun       bool
}

// StartHotfix creates a hotfix directly from the active line that contains the
// defect.
func (service *ReleaseService) StartHotfix(ctx context.Context, request StartHotfixRequest) (branchapp.CreateResult, error) {
	if service.branches == nil {
		return branchapp.CreateResult{}, internalDependencyError("branch service")
	}
	if request.AffectedLine.IsZero() {
		return branchapp.CreateResult{}, invalidWorkflowInput(
			"an affected main, release, or support line is required",
			"select the line that actually contains the defect",
		)
	}
	affectedFamily := request.AffectedLine.Family()
	if affectedFamily != branch.FamilyMain && affectedFamily != branch.FamilyRelease && affectedFamily != branch.FamilySupport {
		return branchapp.CreateResult{}, invalidWorkflowInput(
			"a hotfix starts from main, release/<semver>, or support/<major.minor>",
			"do not start a hotfix from develop or a regular ticket branch",
		)
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	base, err := branch.NewTargetBase(repository.Remote, request.AffectedLine)
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	switchToBranch := true
	result, err := service.branches.Create(ctx, branchapp.CreateRequest{
		Repository:      repository,
		Family:          branch.FamilyHotfix,
		Ticket:          request.Ticket,
		Slug:            request.Slug,
		Base:            &base,
		Switch:          &switchToBranch,
		DryRun:          request.DryRun,
		WorkflowManaged: true,
	})
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	if !request.DryRun {
		if service.git == nil {
			return branchapp.CreateResult{}, internalDependencyError("Git repository")
		}
		if err := service.git.StoreWorkflowBase(ctx, repository, result.Name, base); err != nil {
			return branchapp.CreateResult{}, err
		}
	}
	return result, nil
}

// CutReleaseRequest describes an intentional release cut from develop.
type CutReleaseRequest struct {
	Repository port.RepositoryIdentity
	Version    branch.SemanticVersion
	DryRun     bool
}

// SharedLineIntent describes a privileged CI operation that creates a remote
// protected release or support line. The local CLI never pushes shared lines.
type SharedLineIntent struct {
	Workflow string            `json:"workflow"`
	Kind     string            `json:"kind"`
	Branch   branch.BranchName `json:"branch"`
	Source   branch.TargetBase `json:"source"`
	Inputs   map[string]string `json:"inputs"`
}

// SharedLineIntentResult contains the prepared CI/hosting operation and the
// read-only validation plan that produced it.
type SharedLineIntentResult struct {
	Intent SharedLineIntent     `json:"intent"`
	DryRun bool                 `json:"dryRun"`
	Plan   []branchapp.PlanStep `json:"plan"`
}

// CutRelease creates release/<semver> directly from origin/develop. It does
// not tag, publish artifacts, or merge into main; those are separate release
// approval and pipeline responsibilities.
func (service *ReleaseService) CutRelease(ctx context.Context, request CutReleaseRequest) (SharedLineIntentResult, error) {
	name, err := branch.NewReleaseBranch(request.Version)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	develop := mustDevelop()
	return service.prepareSharedLine(
		ctx,
		request.Repository,
		name,
		develop,
		"release",
		request.Version.String(),
		request.DryRun,
	)
}

// PrepareSupportRequest describes a support-line creation from a released
// main-line version.
type PrepareSupportRequest struct {
	Repository port.RepositoryIdentity
	Version    branch.SupportVersion
	DryRun     bool
}

// PrepareSupport creates support/<major.minor> directly from origin/main only
// when that main revision carries a matching released version tag.
func (service *ReleaseService) PrepareSupport(ctx context.Context, request PrepareSupportRequest) (SharedLineIntentResult, error) {
	name, err := branch.NewSupportBranch(request.Version)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	main := mustMain()
	if !request.DryRun {
		if service.git == nil {
			return SharedLineIntentResult{}, internalDependencyError("Git repository")
		}
		if err := service.git.Fetch(ctx, repository); err != nil {
			return SharedLineIntentResult{}, err
		}
		tags, err := service.git.ReleaseTagsAt(ctx, repository, repository.Remote+"/"+main.String())
		if err != nil {
			return SharedLineIntentResult{}, err
		}
		if !hasMatchingSupportReleaseTag(tags, request.Version) {
			return SharedLineIntentResult{}, invalidWorkflowInput(
				"support lines can be created only from a released main revision with a matching v<major.minor.patch> tag",
				"release and tag the matching version on main before creating its support line",
			)
		}
	}
	return service.prepareSharedLine(
		ctx,
		repository,
		name,
		main,
		"support",
		request.Version.String(),
		request.DryRun,
	)
}

// ReleaseStabilizationKind constrains change categories allowed after a release
// line has been cut.
type ReleaseStabilizationKind string

const (
	ReleaseStabilizationBlocker ReleaseStabilizationKind = "blocker"
	ReleaseStabilizationDocs    ReleaseStabilizationKind = "docs"
	ReleaseStabilizationPrep    ReleaseStabilizationKind = "release-prep"
)

// CreateReleaseStabilizationRequest describes an explicitly permitted short
// working branch from a frozen release line.
type CreateReleaseStabilizationRequest struct {
	Repository port.RepositoryIdentity
	Release    branch.BranchName
	Ticket     ticket.ID
	Slug       branch.Slug
	Kind       ReleaseStabilizationKind
	Switch     *bool
	DryRun     bool
}

// CreateReleaseStabilization creates a controlled fix, docs, or chore branch
// from origin/release/<semver>. New features and refactors are deliberately
// not expressible through this workflow.
func (service *ReleaseService) CreateReleaseStabilization(ctx context.Context, request CreateReleaseStabilizationRequest) (branchapp.CreateResult, error) {
	if service.branches == nil {
		return branchapp.CreateResult{}, internalDependencyError("branch service")
	}
	if request.Release.Family() != branch.FamilyRelease {
		return branchapp.CreateResult{}, invalidWorkflowInput(
			"release stabilization requires a release/<semver> line",
			"select the frozen release line that contains the blocker or release task",
		)
	}
	family, err := stabilizationFamily(request.Kind)
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	base, err := branch.NewTargetBase(repository.Remote, request.Release)
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	result, err := service.branches.Create(ctx, branchapp.CreateRequest{
		Repository:      repository,
		Family:          family,
		Ticket:          request.Ticket,
		Slug:            request.Slug,
		Base:            &base,
		Switch:          request.Switch,
		DryRun:          request.DryRun,
		WorkflowManaged: true,
	})
	if err != nil {
		return branchapp.CreateResult{}, err
	}
	if !request.DryRun {
		if service.git == nil {
			return branchapp.CreateResult{}, internalDependencyError("Git repository")
		}
		if err := service.git.StoreWorkflowBase(ctx, repository, result.Name, base); err != nil {
			return branchapp.CreateResult{}, err
		}
	}
	return result, nil
}

// PrepareReleasePromotionRequest describes a provider-neutral release-to-main
// pull request after release stabilization and approval.
type PrepareReleasePromotionRequest struct {
	Repository port.RepositoryIdentity
	Release    branch.BranchName
	Draft      bool
	DryRun     bool
}

// PrepareReleasePromotionResult exposes the release-to-main pull request
// intent and optional provider result.
type PrepareReleasePromotionResult struct {
	PullRequest  port.PullRequest
	PublishedURL string
	DryRun       bool
}

// PrepareReleasePromotion prepares release/<semver> -> main. It does not tag,
// merge, or publish artifacts; those remain protected CI and hosting actions.
func (service *ReleaseService) PrepareReleasePromotion(ctx context.Context, request PrepareReleasePromotionRequest) (PrepareReleasePromotionResult, error) {
	if request.Release.Family() != branch.FamilyRelease {
		return PrepareReleasePromotionResult{}, invalidWorkflowInput(
			"release promotion requires a release/<semver> branch",
			"select the frozen release line approved for promotion",
		)
	}
	if _, err := normalizeWorkflowRepository(request.Repository); err != nil {
		return PrepareReleasePromotionResult{}, err
	}
	version, _ := request.Release.ReleaseVersion()
	result := PrepareReleasePromotionResult{
		PullRequest: port.PullRequest{
			Source: request.Release,
			Target: mustMain(),
			Title:  "Release " + version.String() + " into main",
			Draft:  request.Draft,
		},
		DryRun: request.DryRun,
	}
	if request.DryRun || service.publisher == nil {
		return result, nil
	}
	published, err := service.publisher.Publish(ctx, result.PullRequest)
	if err != nil {
		return PrepareReleasePromotionResult{}, err
	}
	result.PublishedURL = published.URL
	return result, nil
}

// PrepareReleaseBackmergeRequest describes provider-neutral release backmerge
// preparation after a release has been approved.
type PrepareReleaseBackmergeRequest struct {
	Repository port.RepositoryIdentity
	Release    branch.BranchName
	Draft      bool
	DryRun     bool
}

// PrepareReleaseBackmergeResult exposes the PR intent and optional published
// URL. The workflow never directly mutates develop.
type PrepareReleaseBackmergeResult struct {
	PullRequest  port.PullRequest
	PublishedURL string
	DryRun       bool
}

// PrepareReleaseBackmerge prepares release/<semver> -> develop.
func (service *ReleaseService) PrepareReleaseBackmerge(ctx context.Context, request PrepareReleaseBackmergeRequest) (PrepareReleaseBackmergeResult, error) {
	if request.Release.Family() != branch.FamilyRelease {
		return PrepareReleaseBackmergeResult{}, invalidWorkflowInput(
			"release backmerge requires a release/<semver> branch",
			"select the completed release branch to merge back into develop",
		)
	}
	if _, err := normalizeWorkflowRepository(request.Repository); err != nil {
		return PrepareReleaseBackmergeResult{}, err
	}
	releaseVersion, _ := request.Release.ReleaseVersion()
	pullRequest := port.PullRequest{
		Source: request.Release,
		Target: mustDevelop(),
		Title:  "Backmerge release " + releaseVersion.String() + " into develop",
		Draft:  request.Draft,
	}
	result := PrepareReleaseBackmergeResult{
		PullRequest: pullRequest,
		DryRun:      request.DryRun,
	}
	if request.DryRun || service.publisher == nil {
		return result, nil
	}
	published, err := service.publisher.Publish(ctx, pullRequest)
	if err != nil {
		return PrepareReleaseBackmergeResult{}, err
	}
	result.PublishedURL = published.URL
	return result, nil
}

// PropagateHotfixRequest describes an explicit forward-port or backport of one
// already-reviewed hotfix commit into another active line.
type PropagateHotfixRequest struct {
	Repository port.RepositoryIdentity
	Source     branch.BranchName
	TargetLine branch.BranchName
	CommitID   string
	Slug       branch.Slug
	Push       bool
	Draft      bool
	DryRun     bool
}

// PropagateHotfixResult describes the derived fix branch, cherry-pick, and
// provider-neutral pull request intent.
type PropagateHotfixResult struct {
	Branch       branchapp.CreateResult
	CherryPicked bool
	Publication  PublishTicketResult
}

// PropagateHotfix creates a short-lived fix branch from the target line,
// cherry-picks the requested commit with -x, and prepares the resulting pull
// request. The workflow never assumes that a hotfix automatically reaches
// another active line.
func (service *ReleaseService) PropagateHotfix(ctx context.Context, request PropagateHotfixRequest) (PropagateHotfixResult, error) {
	if service.branches == nil || service.git == nil || service.tickets == nil {
		return PropagateHotfixResult{}, internalDependencyError("hotfix propagation services")
	}
	if request.Source.Family() != branch.FamilyHotfix {
		return PropagateHotfixResult{}, invalidWorkflowInput(
			"hotfix propagation requires a hotfix/<ticket>-<slug> source branch",
			"select the reviewed hotfix branch that contains the commit to propagate",
		)
	}
	switch request.TargetLine.Family() {
	case branch.FamilyMain, branch.FamilyDevelop, branch.FamilyRelease, branch.FamilySupport:
	default:
		return PropagateHotfixResult{}, invalidWorkflowInput(
			"hotfix propagation targets main, develop, release/<semver>, or support/<major.minor>",
			"select the active line that also needs the reviewed hotfix",
		)
	}
	if !commitIDPattern.MatchString(request.CommitID) {
		return PropagateHotfixResult{}, invalidWorkflowInput(
			"a 7 to 64 character hexadecimal commit ID is required",
			"provide the reviewed source commit SHA to cherry-pick",
		)
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return PropagateHotfixResult{}, err
	}
	sourceTicket, _ := request.Source.Ticket()
	slug := request.Slug
	if slug.String() == "" {
		sourceSlug, _ := request.Source.Slug()
		slug, err = branch.ParseSlug("forward-port-" + sourceSlug.String())
		if err != nil {
			return PropagateHotfixResult{}, err
		}
	}
	base, err := branch.NewTargetBase(repository.Remote, request.TargetLine)
	if err != nil {
		return PropagateHotfixResult{}, err
	}
	switchToBranch := true
	created, err := service.branches.Create(ctx, branchapp.CreateRequest{
		Repository:      repository,
		Family:          branch.FamilyFix,
		Ticket:          sourceTicket,
		Slug:            slug,
		Base:            &base,
		Switch:          &switchToBranch,
		DryRun:          request.DryRun,
		WorkflowManaged: true,
	})
	if err != nil {
		return PropagateHotfixResult{}, err
	}
	result := PropagateHotfixResult{Branch: created}
	if request.DryRun {
		result.Publication = PublishTicketResult{
			Branch: created.Name,
			PullRequest: port.PullRequest{
				Source: created.Name,
				Target: request.TargetLine,
				Ticket: sourceTicket,
				Title:  sourceTicket.String() + ": " + slug.String(),
				Draft:  request.Draft,
			},
			DryRun: true,
		}
		return result, nil
	}
	if err := service.git.StoreWorkflowBase(ctx, repository, created.Name, base); err != nil {
		return PropagateHotfixResult{}, err
	}
	if err := service.git.CherryPick(ctx, repository, request.CommitID); err != nil {
		return PropagateHotfixResult{}, err
	}
	result.CherryPicked = true
	target := request.TargetLine
	publication, err := service.tickets.PublishTicket(ctx, PublishTicketRequest{
		Repository:      repository,
		Branch:          created.Name,
		Base:            &base,
		Target:          &target,
		WorkflowManaged: true,
		Push:            request.Push,
		Draft:           request.Draft,
	})
	if err != nil {
		return PropagateHotfixResult{}, err
	}
	result.Publication = publication
	return result, nil
}

// CleanupBranchRequest describes a local cleanup. Remote branch retention and
// deletion remain hosting or CI responsibilities.
type CleanupBranchRequest struct {
	Repository port.RepositoryIdentity
	Branch     branch.BranchName
	DryRun     bool
}

// CleanupBranchResult records the local cleanup and metadata removal outcome.
type CleanupBranchResult struct {
	Branch          branch.BranchName
	DeletedLocal    bool
	MetadataCleared bool
	DryRun          bool
}

// CleanupBranch removes a local private scratch branch. It never deletes remote
// branches or official working branches because their lifecycle belongs to
// hosting and CI automation.
func (service *ReleaseService) CleanupBranch(ctx context.Context, request CleanupBranchRequest) (CleanupBranchResult, error) {
	if service.git == nil {
		return CleanupBranchResult{}, internalDependencyError("Git repository")
	}
	family := request.Branch.Family()
	if family != branch.FamilyScratch {
		return CleanupBranchResult{}, invalidWorkflowInput(
			"cleanup accepts only a private scratch branch",
			"let GitHub, GitLab, or CI own every official branch lifecycle; use the CLI only to delete a local scratch branch",
		)
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return CleanupBranchResult{}, err
	}
	result := CleanupBranchResult{
		Branch: request.Branch,
		DryRun: request.DryRun,
	}
	if request.DryRun {
		return result, nil
	}
	if err := service.git.DeleteLocalBranch(ctx, repository, request.Branch, true); err != nil {
		return CleanupBranchResult{}, err
	}
	result.DeletedLocal = true
	if err := service.git.ClearWorkflowBase(ctx, repository, request.Branch); err != nil {
		return CleanupBranchResult{}, err
	}
	result.MetadataCleared = true
	return result, nil
}

func stabilizationFamily(kind ReleaseStabilizationKind) (branch.Family, error) {
	switch kind {
	case ReleaseStabilizationBlocker:
		return branch.FamilyFix, nil
	case ReleaseStabilizationDocs:
		return branch.FamilyDocs, nil
	case ReleaseStabilizationPrep:
		return branch.FamilyChore, nil
	default:
		return "", invalidWorkflowInput(
			"release stabilization kind must be blocker, docs, or release-prep",
			"select only a change category allowed on a frozen release line",
		)
	}
}

func hasMatchingSupportReleaseTag(tags []string, version branch.SupportVersion) bool {
	for _, tag := range tags {
		raw := strings.TrimPrefix(tag, "v")
		semantic, err := branch.ParseSemanticVersion(raw)
		if err == nil && strings.HasPrefix(semantic.String(), version.String()+".") {
			return true
		}
	}
	return false
}

func (service *ReleaseService) prepareSharedLine(
	ctx context.Context,
	identity port.RepositoryIdentity,
	name branch.BranchName,
	baseName branch.BranchName,
	lineKind string,
	version string,
	dryRun bool,
) (SharedLineIntentResult, error) {
	if service.git == nil {
		return SharedLineIntentResult{}, internalDependencyError("Git repository")
	}
	repository, err := normalizeWorkflowRepository(identity)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	if err := service.git.ValidateBranchRef(ctx, repository, name); err != nil {
		return SharedLineIntentResult{}, err
	}
	base, err := branch.NewTargetBase(repository.Remote, baseName)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	result := SharedLineIntentResult{
		Intent: SharedLineIntent{
			Workflow: "create-protected-line.yml",
			Kind:     lineKind,
			Branch:   name,
			Source:   base,
			Inputs: map[string]string{
				"kind":    lineKind,
				"version": version,
			},
		},
		DryRun: dryRun,
		Plan: []branchapp.PlanStep{
			{Action: "fetch", Detail: "git fetch --prune " + repository.Remote},
			{Action: "dispatch", Detail: "authorized CI creates " + name.String() + " from " + base.String()},
		},
	}
	if dryRun {
		return result, nil
	}
	if err := service.git.Fetch(ctx, repository); err != nil {
		return SharedLineIntentResult{}, err
	}
	hasCommits, err := service.git.HasCommits(ctx, repository)
	if err != nil {
		return SharedLineIntentResult{}, err
	}
	if !hasCommits {
		return SharedLineIntentResult{}, problem.New(problem.Details{
			Code:        problem.CodeRepositoryHasNoCommits,
			Category:    problem.CategoryRepository,
			Field:       "repository",
			Expected:    "at least one commit before preparing a protected shared line",
			Rule:        "release and support lines do not implicitly bootstrap repositories",
			Remediation: "create an explicit initial commit before requesting the protected line",
		})
	}
	return result, nil
}

func normalizeWorkflowRepository(repository port.RepositoryIdentity) (port.RepositoryIdentity, error) {
	if repository.Root == "" {
		return port.RepositoryIdentity{}, repositoryRequired()
	}
	if repository.Remote == "" {
		repository.Remote = "origin"
	}
	return repository, nil
}

func mustMain() branch.BranchName {
	name, err := branch.ParseName("main")
	if err != nil {
		panic(err)
	}
	return name
}
