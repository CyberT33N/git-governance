package workflow

import (
	"context"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
)

// AlignReleasePromotionBaseRequest describes an explicit, release-preparation
// merge of the current main line into a release stabilization branch.
type AlignReleasePromotionBaseRequest struct {
	Repository        port.RepositoryIdentity
	Release           branch.BranchName
	Branch            branch.BranchName
	Push              bool
	CreatePullRequest bool
	Draft             bool
	DryRun            bool
}

// AlignReleasePromotionBaseResult records the controlled alignment and its
// optional publication back to the frozen release line.
type AlignReleasePromotionBaseResult struct {
	Branch             branch.BranchName
	Release            branch.BranchName
	Main               branch.BranchName
	PullRequest        port.PullRequest
	MissingMainCommits bool
	Merged             bool
	Pushed             bool
	PublishedURL       string
	Quality            *port.QualityResult
	DryRun             bool
}

// AlignReleasePromotionBase merges main into a release-preparation working
// branch so a strict main ruleset can validate the exact promotion result. It
// never mutates the frozen release line directly; publication remains a PR.
func (service *ReleaseService) AlignReleasePromotionBase(
	ctx context.Context,
	request AlignReleasePromotionBaseRequest,
) (AlignReleasePromotionBaseResult, error) {
	if service.branches == nil || service.git == nil {
		return AlignReleasePromotionBaseResult{}, internalDependencyError("promotion-base alignment services")
	}
	if request.Release.Family() != branch.FamilyRelease {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment requires a release/<semver> line",
			"provide the frozen release line whose promotion is out of date",
		)
	}
	if request.Branch.Family() != branch.FamilyChore {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment requires a chore release-preparation branch",
			"create a release-prep stabilization branch before aligning its promotion base",
		)
	}
	if request.CreatePullRequest && !request.Push {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"pull-request creation requires an explicit branch push",
			"set Push before requesting provider pull-request creation",
		)
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if _, err := service.branches.Validate(ctx, branchValidationRequest(repository, request.Branch)); err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	current, err := service.git.CurrentBranch(ctx, repository)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if current.String() != request.Branch.String() {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment may mutate only the checked-out stabilization branch",
			"switch to the requested release-preparation branch before retrying",
		)
	}
	releaseBase, err := branch.NewTargetBase(repository.Remote, request.Release)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	storedBase, found, err := service.git.WorkflowBase(ctx, repository, request.Branch)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if !found || storedBase.String() != releaseBase.String() {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment requires a branch created from the requested release line",
			"create the release-preparation branch with workflow release stabilize and use its recorded release base",
		)
	}
	main := mustMain()
	// releaseBase above already established that the selected remote is valid,
	// and main is a fixed, valid branch taxonomy member.
	mainBase, _ := branch.NewTargetBase(repository.Remote, main)
	result := AlignReleasePromotionBaseResult{
		Branch:      request.Branch,
		Release:     request.Release,
		Main:        main,
		PullRequest: promotionBaseAlignmentPullRequest(request.Branch, request.Release, request.Draft),
		DryRun:      request.DryRun,
	}
	if request.DryRun {
		exists, err := service.git.TargetBaseExists(ctx, repository, mainBase)
		if err != nil {
			return AlignReleasePromotionBaseResult{}, err
		}
		if !exists {
			return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
				"the current main target base must exist before promotion-base alignment",
				"fetch the selected remote and verify main before retrying",
			)
		}
		missing, err := service.git.HasMissingBaseCommits(ctx, repository, mainBase)
		if err != nil {
			return AlignReleasePromotionBaseResult{}, err
		}
		result.MissingMainCommits = missing
		return result, nil
	}
	if request.CreatePullRequest {
		if service.tickets == nil {
			return AlignReleasePromotionBaseResult{}, internalDependencyError("ticket publication service")
		}
		if err := service.tickets.PreflightPullRequest(ctx, repository, result.PullRequest); err != nil {
			return AlignReleasePromotionBaseResult{}, err
		}
	}
	clean, err := service.git.IsWorktreeClean(ctx, repository)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if !clean {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment requires a clean working tree",
			"commit, explicitly stash, or otherwise safely handle local changes before retrying",
		)
	}
	if err := service.git.Fetch(ctx, repository); err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	exists, err := service.git.TargetBaseExists(ctx, repository, mainBase)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if !exists {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"the current main target base must exist before promotion-base alignment",
			"fetch the selected remote and verify main before retrying",
		)
	}
	missing, err := service.git.HasMissingBaseCommits(ctx, repository, mainBase)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	result.MissingMainCommits = missing
	if !missing && !request.Push {
		return result, nil
	}
	if service.quality == nil {
		return AlignReleasePromotionBaseResult{}, internalDependencyError("quality runner")
	}
	if missing {
		message := promotionBaseAlignmentMergeMessage(request.Branch, request.Release)
		if err := service.git.Merge(ctx, repository, mainBase, message); err != nil {
			return AlignReleasePromotionBaseResult{}, err
		}
		result.Merged = true
	}
	if _, err := service.branches.Validate(ctx, branchValidationRequest(repository, request.Branch)); err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	quality, err := service.quality.Run(ctx, repository, port.QualityRequest{
		Families: []branch.Family{request.Branch.Family()},
	})
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	result.Quality = &quality
	if !request.Push {
		return result, nil
	}
	publication, err := service.git.PublicationState(ctx, repository, request.Branch)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	if publication == branch.PublicationUnknown {
		return AlignReleasePromotionBaseResult{}, invalidWorkflowInput(
			"promotion-base alignment requires a known branch publication state",
			"fetch the remote and resolve the branch tracking state before retrying",
		)
	}
	if err := service.git.Push(ctx, repository, request.Branch, publication == branch.PublicationUnpublished); err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	result.Pushed = true
	if !request.CreatePullRequest {
		return result, nil
	}
	publishedURL, err := service.tickets.PublishPullRequest(ctx, repository, result.PullRequest)
	if err != nil {
		return AlignReleasePromotionBaseResult{}, err
	}
	result.PublishedURL = publishedURL
	return result, nil
}

func branchValidationRequest(repository port.RepositoryIdentity, name branch.BranchName) branchapp.ValidateRequest {
	return branchapp.ValidateRequest{Repository: repository, Name: name}
}

func promotionBaseAlignmentPullRequest(worker, release branch.BranchName, draft bool) port.PullRequest {
	id, _ := worker.Ticket()
	slug, _ := worker.Slug()
	return port.PullRequest{
		Source: worker,
		Target: release,
		Ticket: id,
		Title:  id.String() + ": " + slug.String(),
		Draft:  draft,
	}
}

func promotionBaseAlignmentMergeMessage(worker, release branch.BranchName) commitmsg.Message {
	id, _ := worker.Ticket()
	version, _ := release.ReleaseVersion()
	// The caller validates the ticket-scoped chore worker and semantic release
	// line before reaching this construction, so these value-object operations
	// are total for the supplied arguments.
	header, _ := commitmsg.NewHeader(
		commitmsg.TypeChore,
		id,
		"align release "+version.String()+" with main for promotion",
		false,
	)
	message, _ := commitmsg.NewMessage(header, "", nil)
	return message
}
