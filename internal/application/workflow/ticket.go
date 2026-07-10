// Package workflow composes branch and commit use cases into bounded,
// resumable Git workflows without spawning the CLI recursively.
package workflow

import (
	"context"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
)

// TicketService owns the bounded ticket start and publish workflows.
type TicketService struct {
	branches  *branchapp.Service
	sync      *branchapp.Synchronizer
	git       port.GitRepository
	quality   port.QualityRunner
	publisher port.PullRequestPublisher
}

// NewTicketService creates the ticket workflow service.
func NewTicketService(
	branches *branchapp.Service,
	sync *branchapp.Synchronizer,
	git port.GitRepository,
	quality port.QualityRunner,
	publisher port.PullRequestPublisher,
) *TicketService {
	return &TicketService{
		branches:  branches,
		sync:      sync,
		git:       git,
		quality:   quality,
		publisher: publisher,
	}
}

// StartTicketRequest describes normal ticket work from develop.
type StartTicketRequest struct {
	Repository    port.RepositoryIdentity
	Family        branch.Family
	Ticket        ticket.ID
	Slug          branch.Slug
	CreateScratch bool
	ScratchSlug   branch.Slug
	DryRun        bool
}

// StartTicketResult identifies the official and optional scratch branches.
type StartTicketResult struct {
	Official branchapp.CreateResult
	Scratch  *branchapp.CreateResult
	Active   branch.BranchName
}

// StartTicket creates one official regular ticket branch and, optionally, a
// private scratch branch from it. It ends at the active branch and deliberately
// does not continue into the development or pull-request phase.
func (service *TicketService) StartTicket(ctx context.Context, request StartTicketRequest) (StartTicketResult, error) {
	if service.branches == nil {
		return StartTicketResult{}, internalDependencyError("branch service")
	}
	if request.Family != branch.FamilyFeature &&
		request.Family != branch.FamilyFix &&
		request.Family != branch.FamilyDocs &&
		request.Family != branch.FamilyRefactor &&
		request.Family != branch.FamilyChore &&
		request.Family != branch.FamilyTest &&
		request.Family != branch.FamilyPerf {
		return StartTicketResult{}, invalidWorkflowInput(
			"regular ticket start accepts feature, fix, docs, refactor, chore, test, or perf",
			"select a regular ticket family or use the hotfix/release workflow",
		)
	}

	switchToOfficial := true
	official, err := service.branches.Create(ctx, branchapp.CreateRequest{
		Repository: request.Repository,
		Family:     request.Family,
		Ticket:     request.Ticket,
		Slug:       request.Slug,
		Switch:     &switchToOfficial,
		DryRun:     request.DryRun,
	})
	if err != nil {
		return StartTicketResult{}, err
	}
	result := StartTicketResult{
		Official: official,
		Active:   official.Name,
	}
	if !request.CreateScratch {
		return result, nil
	}

	scratchSlug := request.ScratchSlug
	if scratchSlug.String() == "" {
		scratchSlug, err = branch.ParseSlug(request.Slug.String() + "-exploration")
		if err != nil {
			return StartTicketResult{}, err
		}
	}
	// official.Name was just created by the branch service and is therefore a
	// canonical local branch name. NewLocalBase cannot reject that invariant.
	localBase, _ := branch.NewLocalBase(official.Name)
	switchToScratch := true
	scratch, err := service.branches.Create(ctx, branchapp.CreateRequest{
		Repository: request.Repository,
		Family:     branch.FamilyScratch,
		Ticket:     request.Ticket,
		Slug:       scratchSlug,
		Base:       &localBase,
		Switch:     &switchToScratch,
		DryRun:     request.DryRun,
		SkipFetch:  true,
	})
	if err != nil {
		return StartTicketResult{}, err
	}
	result.Scratch = &scratch
	result.Active = scratch.Name
	return result, nil
}

// PublishTicketRequest describes the handoff from completed local work to a
// push and provider-neutral pull request.
type PublishTicketRequest struct {
	Repository      port.RepositoryIdentity
	Branch          branch.BranchName
	Base            *branch.TargetBase
	Target          *branch.BranchName
	WorkflowManaged bool
	Push            bool
	Draft           bool
	DryRun          bool
}

// PublishTicketResult contains the push status and provider-neutral PR intent.
type PublishTicketResult struct {
	Branch              branch.BranchName
	Sync                branchapp.SyncResult
	Pushed              bool
	PullRequest         port.PullRequest
	PublishedURL        string
	DryRun              bool
	Quality             port.QualityResult
	PostMutationQuality *port.QualityResult
}

// PublishTicket validates the complete local commit series, runs quality
// gates, synchronizes the base safely, and emits a pull request intent. It
// stops at the pull request boundary.
func (service *TicketService) PublishTicket(ctx context.Context, request PublishTicketRequest) (PublishTicketResult, error) {
	if service.branches == nil || service.sync == nil || service.git == nil {
		return PublishTicketResult{}, internalDependencyError("ticket workflow services")
	}
	if request.Branch.IsZero() || !request.Branch.Family().IsOfficialWorkingBranch() {
		return PublishTicketResult{}, invalidWorkflowInput(
			"ticket publish requires an official ticket branch",
			"run this workflow from feature, fix, docs, refactor, chore, test, perf, or hotfix work",
		)
	}

	repository := request.Repository
	if repository.Remote == "" {
		repository.Remote = "origin"
	}
	if repository.Root == "" {
		return PublishTicketResult{}, repositoryRequired()
	}

	validation, err := service.branches.Validate(ctx, branchapp.ValidateRequest{
		Repository: repository,
		Name:       request.Branch,
	})
	if err != nil {
		return PublishTicketResult{}, err
	}
	baseInput := request.Base
	if baseInput == nil && validation.Name.Family().MayUseWorkflowBase() {
		storedBase, found, err := service.git.WorkflowBase(ctx, repository, validation.Name)
		if err != nil {
			return PublishTicketResult{}, err
		}
		if found {
			baseInput = &storedBase
		}
	}
	base, err := resolveTicketBase(validation.Name, repository, baseInput, request.WorkflowManaged)
	if err != nil {
		return PublishTicketResult{}, err
	}
	target, err := resolvePullRequestTarget(validation.Name, base, request.Target, request.WorkflowManaged)
	if err != nil {
		return PublishTicketResult{}, err
	}

	if !request.DryRun {
		if err := service.git.Fetch(ctx, repository); err != nil {
			return PublishTicketResult{}, err
		}
	}
	if err := service.validateCommitSeries(ctx, repository, validation.Name, base); err != nil {
		return PublishTicketResult{}, err
	}
	quality := port.QualityResult{
		Status: port.QualitySkipped,
		Detail: "quality gates are not executed during dry-run",
	}
	if !request.DryRun {
		if service.quality == nil {
			quality = port.QualityResult{
				Status: port.QualityUnconfigured,
				Detail: "no quality runner is configured",
			}
		} else {
			quality, err = service.quality.Run(ctx, repository, port.QualityRequest{
				Families: []branch.Family{validation.Name.Family()},
			})
			if err != nil {
				return PublishTicketResult{}, err
			}
		}
	}

	syncResult, err := service.sync.Sync(ctx, branchapp.SyncRequest{
		Repository:      repository,
		Name:            validation.Name,
		Base:            &base,
		Strategy:        branchapp.SyncAuto,
		DryRun:          request.DryRun,
		SkipFetch:       true,
		WorkflowManaged: request.WorkflowManaged,
	})
	if err != nil {
		return PublishTicketResult{}, err
	}
	if syncResult.Mutated {
		if err := service.validateCommitSeries(ctx, repository, validation.Name, base); err != nil {
			return PublishTicketResult{}, err
		}
	}

	branchTicket, _ := validation.Name.Ticket()
	branchSlug, _ := validation.Name.Slug()
	pullRequest := port.PullRequest{
		Source: validation.Name,
		Target: target,
		Ticket: branchTicket,
		Title:  branchTicket.String() + ": " + branchSlug.String(),
		Draft:  request.Draft,
	}
	result := PublishTicketResult{
		Branch:      validation.Name,
		Sync:        syncResult,
		PullRequest: pullRequest,
		DryRun:      request.DryRun,
		Quality:     quality,
	}
	if syncResult.Quality != nil {
		result.PostMutationQuality = syncResult.Quality
	}
	if request.DryRun {
		return result, nil
	}
	if request.Push {
		if err := service.git.Push(ctx, repository, validation.Name, syncResult.Publication == branch.PublicationUnpublished); err != nil {
			return PublishTicketResult{}, err
		}
		result.Pushed = true
	}
	if request.Push && service.publisher != nil {
		published, err := service.publisher.Publish(ctx, pullRequest)
		if err != nil {
			return PublishTicketResult{}, err
		}
		result.PublishedURL = published.URL
	}
	return result, nil
}

func (service *TicketService) validateCommitSeries(ctx context.Context, repository port.RepositoryIdentity, name branch.BranchName, base branch.TargetBase) error {
	messages, err := service.git.CommitMessagesSince(ctx, repository, base)
	if err != nil {
		return err
	}
	return branchapp.ValidateCommitSeries(name, messages)
}

func resolveTicketBase(
	name branch.BranchName,
	repository port.RepositoryIdentity,
	explicit *branch.TargetBase,
	workflowManaged bool,
) (branch.TargetBase, error) {
	if base, found, err := name.Family().DefaultTargetBase(repository.Remote); err != nil {
		return branch.TargetBase{}, err
	} else if found {
		if explicit != nil && explicit.String() != base.String() {
			if workflowManaged && isWorkflowManagedTicketBase(name.Family(), *explicit) {
				return *explicit, nil
			}
			return branch.TargetBase{}, invalidWorkflowInput(
				"regular ticket publication uses origin/develop unless a dedicated workflow selected another active line",
				"remove --base or use the dedicated stabilization or propagation workflow",
			)
		}
		return base, nil
	}
	if explicit != nil {
		if name.Family() == branch.FamilyHotfix && !isHotfixBase(*explicit) {
			return branch.TargetBase{}, invalidWorkflowInput(
				"a hotfix publish target must be main, release/<semver>, or support/<major.minor>",
				"provide the same active line from which the hotfix branch was created",
			)
		}
		return *explicit, nil
	}
	return branch.TargetBase{}, invalidWorkflowInput(
		"this ticket branch family requires an explicit target base",
		"provide --base for the actual hotfix target line",
	)
}

func resolvePullRequestTarget(
	name branch.BranchName,
	base branch.TargetBase,
	explicit *branch.BranchName,
	workflowManaged bool,
) (branch.BranchName, error) {
	target := mustDevelop()
	if name.Family() == branch.FamilyHotfix ||
		(workflowManaged && name.Family() == branch.FamilyFix && base.Branch().Family() != branch.FamilyDevelop) {
		target = base.Branch()
	}
	if explicit != nil && explicit.String() != target.String() {
		return branch.BranchName{}, invalidWorkflowInput(
			"the pull request target must match the branch workflow target line",
			"remove --target or supply the actual affected or integration line",
		)
	}
	return target, nil
}

func isHotfixBase(base branch.TargetBase) bool {
	switch base.Branch().Family() {
	case branch.FamilyMain, branch.FamilyRelease, branch.FamilySupport:
		return true
	default:
		return false
	}
}

func isSharedLineBase(base branch.TargetBase) bool {
	switch base.Branch().Family() {
	case branch.FamilyMain, branch.FamilyDevelop, branch.FamilyRelease, branch.FamilySupport:
		return true
	default:
		return false
	}
}

func isWorkflowManagedTicketBase(family branch.Family, base branch.TargetBase) bool {
	switch family {
	case branch.FamilyFix:
		return isSharedLineBase(base)
	case branch.FamilyDocs, branch.FamilyChore:
		return base.Branch().Family() == branch.FamilyRelease
	default:
		return false
	}
}

func mustDevelop() branch.BranchName {
	// This literal is part of the product's fixed branch taxonomy and is
	// independently validated by the branch domain tests.
	name, _ := branch.ParseName("develop")
	return name
}

func invalidWorkflowInput(rule, remediation string) error {
	return problem.New(problem.Details{
		Code:        problem.CodeInvalidInput,
		Category:    problem.CategoryGovernance,
		Field:       "workflow",
		Expected:    "a valid workflow request",
		Rule:        rule,
		Remediation: remediation,
	})
}

func repositoryRequired() error {
	return problem.New(problem.Details{
		Code:        problem.CodeRepositoryNotFound,
		Category:    problem.CategoryRepository,
		Field:       "repository",
		Expected:    "a discovered local Git repository",
		Rule:        "workflow operations require a repository root",
		Remediation: "run from a Git repository or pass --repo",
	})
}

func internalDependencyError(name string) error {
	return problem.New(problem.Details{
		Code:        problem.CodeInternal,
		Category:    problem.CategoryInternal,
		Field:       "dependency",
		Actual:      name,
		Expected:    "a configured workflow dependency",
		Rule:        "workflow services are composed with required use cases and adapters",
		Remediation: "fix the composition root",
	})
}
