package branchapp

import (
	"context"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

// SyncStrategy controls how a detected base delta is handled.
type SyncStrategy string

const (
	SyncCheck  SyncStrategy = "check"
	SyncAuto   SyncStrategy = "auto"
	SyncRebase SyncStrategy = "rebase"
	SyncMerge  SyncStrategy = "merge"
)

// Synchronizer centralizes base freshness and rewrite policy.
type Synchronizer struct {
	git       port.GitRepository
	validator *Service
	quality   port.QualityRunner
}

// NewSynchronizer creates a synchronization service.
func NewSynchronizer(git port.GitRepository, validator *Service, quality port.QualityRunner) *Synchronizer {
	return &Synchronizer{
		git:       git,
		validator: validator,
		quality:   quality,
	}
}

// SyncRequest describes a base-synchronization request.
type SyncRequest struct {
	Repository      port.RepositoryIdentity
	Name            branch.BranchName
	Base            *branch.TargetBase
	Strategy        SyncStrategy
	MergeMessage    *commitmsg.Message
	DryRun          bool
	SkipFetch       bool
	WorkflowManaged bool
}

// SyncResult describes the observed state and any mutation performed.
type SyncResult struct {
	Name               branch.BranchName
	Base               branch.TargetBase
	Publication        branch.PublicationState
	MissingBaseCommits bool
	Mutated            bool
	RecommendedAction  string
	Quality            *port.QualityResult
}

// Sync applies the requested policy-safe synchronization strategy.
func (synchronizer *Synchronizer) Sync(ctx context.Context, request SyncRequest) (SyncResult, error) {
	repository, err := normalizeRepository(request.Repository)
	if err != nil {
		return SyncResult{}, err
	}
	if err := contextError(ctx); err != nil {
		return SyncResult{}, err
	}
	if synchronizer.validator == nil {
		return SyncResult{}, internalDependencyError("branch validator")
	}
	if _, err := synchronizer.validator.Validate(ctx, ValidateRequest{Repository: repository, Name: request.Name}); err != nil {
		return SyncResult{}, err
	}
	if !request.Name.Family().IsOfficialWorkingBranch() {
		return SyncResult{}, unsupportedSyncFamily(request.Name)
	}

	baseInput, err := synchronizer.workflowBase(ctx, repository, request.Name, request.Base)
	if err != nil {
		return SyncResult{}, err
	}
	base, err := resolveSyncBase(request.Name, repository, baseInput, request.WorkflowManaged)
	if err != nil {
		return SyncResult{}, err
	}
	strategy := request.Strategy
	if strategy == "" {
		strategy = SyncCheck
	}
	if strategy != SyncCheck && strategy != SyncAuto && strategy != SyncRebase && strategy != SyncMerge {
		return SyncResult{}, invalidSyncStrategy(strategy)
	}

	clean, err := synchronizer.git.IsWorktreeClean(ctx, repository)
	if err != nil {
		return SyncResult{}, err
	}
	if !clean {
		return SyncResult{}, worktreeNotCleanForSync()
	}
	if !request.SkipFetch && !request.DryRun {
		if err := synchronizer.git.Fetch(ctx, repository); err != nil {
			return SyncResult{}, err
		}
	}

	publication, err := synchronizer.git.PublicationState(ctx, repository, request.Name)
	if err != nil {
		return SyncResult{}, err
	}
	if publication == branch.PublicationUnknown {
		return SyncResult{}, problem.New(problem.Details{
			Code:        problem.CodeBranchPublicationUnknown,
			Category:    problem.CategoryRepository,
			Field:       "branch publication state",
			Actual:      request.Name.String(),
			Expected:    "a known published or unpublished state",
			Rule:        "history rewrites are forbidden when publication state cannot be determined",
			Remediation: "fetch the remote successfully and resolve the branch tracking state",
		})
	}

	missing, err := synchronizer.git.HasMissingBaseCommits(ctx, repository, base)
	if err != nil {
		return SyncResult{}, err
	}
	result := SyncResult{
		Name:               request.Name,
		Base:               base,
		Publication:        publication,
		MissingBaseCommits: missing,
	}
	if !missing {
		if strategy == SyncRebase {
			return SyncResult{}, problem.New(problem.Details{
				Code:        problem.CodeRebaseNotRequired,
				Category:    problem.CategoryGovernance,
				Field:       "target base",
				Actual:      base.String(),
				Expected:    "a target base containing commits missing from the current branch",
				Rule:        "a rebase is performed only when the target base has advanced",
				Remediation: "continue without a rebase; the current branch already contains the target base",
			})
		}
		result.RecommendedAction = "none"
		return result, nil
	}

	switch strategy {
	case SyncCheck:
		result.RecommendedAction = recommendedAction(publication)
		return result, nil
	case SyncAuto:
		if publication == branch.PublicationPublished {
			result.RecommendedAction = "merge"
			return result, nil
		}
		if request.DryRun {
			result.RecommendedAction = "rebase"
			return result, nil
		}
		if err := synchronizer.git.Rebase(ctx, repository, base); err != nil {
			return SyncResult{}, err
		}
		quality, err := synchronizer.validateAfterMutation(ctx, repository, request.Name)
		if err != nil {
			return SyncResult{}, err
		}
		result.Quality = &quality
		result.Mutated = true
		result.RecommendedAction = "rebased"
		return result, nil
	case SyncRebase:
		if publication != branch.PublicationUnpublished {
			return SyncResult{}, rebaseAfterPublishForbidden(request.Name, base)
		}
		if request.DryRun {
			result.RecommendedAction = "rebase"
			return result, nil
		}
		if err := synchronizer.git.Rebase(ctx, repository, base); err != nil {
			return SyncResult{}, err
		}
		quality, err := synchronizer.validateAfterMutation(ctx, repository, request.Name)
		if err != nil {
			return SyncResult{}, err
		}
		result.Quality = &quality
		result.Mutated = true
		result.RecommendedAction = "rebased"
		return result, nil
	case SyncMerge:
		if publication != branch.PublicationPublished {
			return SyncResult{}, invalidMergeBeforePublish(request.Name)
		}
		if request.MergeMessage == nil {
			return SyncResult{}, problem.New(problem.Details{
				Code:        problem.CodeCommitHeaderInvalid,
				Category:    problem.CategoryGovernance,
				Field:       "merge message",
				Expected:    "a validated Conventional Commit message",
				Rule:        "published branch synchronization creates an explicit governed merge commit",
				Example:     "chore(ABC-123): merge origin/develop",
				Remediation: "supply a validated merge message matching the branch ticket",
			})
		}
		if request.DryRun {
			result.RecommendedAction = "merge"
			return result, nil
		}
		if err := synchronizer.git.Merge(ctx, repository, base, *request.MergeMessage); err != nil {
			return SyncResult{}, err
		}
		quality, err := synchronizer.validateAfterMutation(ctx, repository, request.Name)
		if err != nil {
			return SyncResult{}, err
		}
		result.Quality = &quality
		result.Mutated = true
		result.RecommendedAction = "merged"
		return result, nil
	default:
		return SyncResult{}, invalidSyncStrategy(strategy)
	}
}

// PrePushRequest describes the local governance data checked before a push.
type PrePushRequest struct {
	Repository port.RepositoryIdentity
	Name       branch.BranchName
	Base       *branch.TargetBase
}

// PrePushResult describes the freshness and publication state checked locally.
type PrePushResult struct {
	Name               branch.BranchName
	Base               *branch.TargetBase
	Publication        branch.PublicationState
	MissingBaseCommits bool
}

// ValidatePrePush validates an outgoing branch but never rewrites, merges, or
// otherwise mutates local branch history.
func (synchronizer *Synchronizer) ValidatePrePush(ctx context.Context, request PrePushRequest) (PrePushResult, error) {
	repository, err := normalizeRepository(request.Repository)
	if err != nil {
		return PrePushResult{}, err
	}
	if synchronizer.validator == nil {
		return PrePushResult{}, internalDependencyError("branch validator")
	}
	if _, err := synchronizer.validator.Validate(ctx, ValidateRequest{Repository: repository, Name: request.Name}); err != nil {
		return PrePushResult{}, err
	}
	if request.Name.Family().IsSharedLine() {
		return PrePushResult{}, problem.New(problem.Details{
			Code:        problem.CodeSharedLineMutationForbidden,
			Category:    problem.CategoryGovernance,
			Field:       "branch",
			Actual:      request.Name.String(),
			Expected:    "a pull request into a shared line",
			Rule:        "developers do not directly push main, develop, release, or support lines",
			Remediation: "push an official working branch and open a pull request",
		})
	}
	if request.Name.Family() == branch.FamilyScratch {
		return PrePushResult{
			Name:        request.Name,
			Publication: branch.PublicationUnknown,
		}, nil
	}
	if !request.Name.Family().IsOfficialWorkingBranch() {
		return PrePushResult{}, unsupportedSyncFamily(request.Name)
	}

	baseInput, err := synchronizer.workflowBase(ctx, repository, request.Name, request.Base)
	if err != nil {
		return PrePushResult{}, err
	}
	base, err := resolveSyncBase(request.Name, repository, baseInput, false)
	if err != nil {
		return PrePushResult{}, err
	}
	if err := synchronizer.git.Fetch(ctx, repository); err != nil {
		return PrePushResult{}, err
	}
	publication, err := synchronizer.git.PublicationState(ctx, repository, request.Name)
	if err != nil {
		return PrePushResult{}, err
	}
	if publication == branch.PublicationUnknown {
		return PrePushResult{}, problem.New(problem.Details{
			Code:        problem.CodeBranchPublicationUnknown,
			Category:    problem.CategoryRepository,
			Field:       "branch publication state",
			Actual:      request.Name.String(),
			Expected:    "a known published or unpublished state",
			Rule:        "pre-push validation must not infer branch history safety from an unknown state",
			Remediation: "fetch the remote successfully and resolve the branch tracking state",
		})
	}
	missing, err := synchronizer.git.HasMissingBaseCommits(ctx, repository, base)
	if err != nil {
		return PrePushResult{}, err
	}
	result := PrePushResult{
		Name:               request.Name,
		Base:               &base,
		Publication:        publication,
		MissingBaseCommits: missing,
	}
	if publication == branch.PublicationUnpublished && missing {
		return PrePushResult{}, problem.New(problem.Details{
			Code:        problem.CodeBranchBaseInvalid,
			Category:    problem.CategoryGovernance,
			Field:       "target base",
			Actual:      base.String(),
			Expected:    "an unpublished branch based on the latest target base",
			Rule:        "before the first push, a branch with missing base commits must be rebased",
			Example:     "git rebase " + base.String(),
			Remediation: "run branch sync-base --strategy rebase, rerun quality checks, then push again",
		})
	}
	return result, nil
}

// ValidatePush exposes pre-push validation as a small cross-use-case contract.
// It deliberately performs no history rewrite, merge, or push itself.
func (synchronizer *Synchronizer) ValidatePush(ctx context.Context, repository port.RepositoryIdentity, name branch.BranchName, base *branch.TargetBase) error {
	_, err := synchronizer.ValidatePrePush(ctx, PrePushRequest{
		Repository: repository,
		Name:       name,
		Base:       base,
	})
	return err
}

func (synchronizer *Synchronizer) runQuality(ctx context.Context, repository port.RepositoryIdentity) (port.QualityResult, error) {
	if synchronizer.quality == nil {
		return port.QualityResult{
			Status: port.QualityUnconfigured,
			Detail: "no quality runner is configured",
		}, nil
	}
	return synchronizer.quality.Run(ctx, repository)
}

func (synchronizer *Synchronizer) workflowBase(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	explicit *branch.TargetBase,
) (*branch.TargetBase, error) {
	if explicit != nil {
		return explicit, nil
	}
	if !name.Family().MayUseWorkflowBase() {
		return nil, nil
	}
	stored, found, err := synchronizer.git.WorkflowBase(ctx, repository, name)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &stored, nil
}

func (synchronizer *Synchronizer) validateAfterMutation(
	ctx context.Context,
	repository port.RepositoryIdentity,
	name branch.BranchName,
) (port.QualityResult, error) {
	if _, err := synchronizer.validator.Validate(ctx, ValidateRequest{
		Repository: repository,
		Name:       name,
	}); err != nil {
		return port.QualityResult{}, err
	}
	return synchronizer.runQuality(ctx, repository)
}

func recommendedAction(publication branch.PublicationState) string {
	if publication == branch.PublicationUnpublished {
		return "rebase"
	}
	return "merge"
}

func unsupportedSyncFamily(name branch.BranchName) error {
	return problem.New(problem.Details{
		Code:        problem.CodeBranchFamilyInvalid,
		Category:    problem.CategoryGovernance,
		Field:       "branch family",
		Actual:      name.Family().String(),
		Expected:    "an official working branch",
		Rule:        "base synchronization is defined for official published or unpublished working branches",
		Example:     "feature/ABC-123-add-export-button",
		Remediation: "use the matching workflow for release, support, hotfix, or scratch work",
	})
}

func invalidSyncStrategy(strategy SyncStrategy) error {
	return problem.New(problem.Details{
		Code:        problem.CodeInvalidInput,
		Category:    problem.CategoryUsage,
		Field:       "sync strategy",
		Actual:      string(strategy),
		Expected:    "check, auto, rebase, or merge",
		Rule:        "synchronization uses an explicit strategy",
		Example:     "rebase",
		Remediation: "choose a supported synchronization strategy",
	})
}

func worktreeNotCleanForSync() error {
	return problem.New(problem.Details{
		Code:        problem.CodeWorktreeNotClean,
		Category:    problem.CategoryRepository,
		Field:       "worktree",
		Expected:    "a clean working tree before rebase or merge",
		Rule:        "history operations must not risk uncommitted changes",
		Example:     "git status --porcelain returns no entries",
		Remediation: "commit, stash, or discard local changes before synchronizing",
	})
}

func rebaseAfterPublishForbidden(name branch.BranchName, base branch.TargetBase) error {
	return problem.New(problem.Details{
		Code:        problem.CodeRebaseAfterPublishForbidden,
		Category:    problem.CategoryGovernance,
		Field:       "branch",
		Actual:      name.String(),
		Expected:    "an unpublished official branch for a rebase",
		Rule:        "published official branches are append-only and synchronize with an explicit merge",
		Example:     "chore(ABC-123): merge " + base.String(),
		Remediation: "use --strategy merge with a governed merge message",
	})
}

func invalidMergeBeforePublish(name branch.BranchName) error {
	return problem.New(problem.Details{
		Code:        problem.CodeBranchBaseInvalid,
		Category:    problem.CategoryGovernance,
		Field:       "branch",
		Actual:      name.String(),
		Expected:    "a published official branch for a merge synchronization",
		Rule:        "unpublished branches rebase only when their target base has advanced",
		Remediation: "use --strategy rebase before the first push",
	})
}

func internalDependencyError(name string) error {
	return problem.New(problem.Details{
		Code:        problem.CodeInternal,
		Category:    problem.CategoryInternal,
		Field:       "dependency",
		Actual:      name,
		Expected:    "a configured application dependency",
		Rule:        "application services are composed with their required ports",
		Remediation: "fix the composition root",
	})
}
