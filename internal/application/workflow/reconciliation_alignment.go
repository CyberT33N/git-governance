package workflow

import (
	"context"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
)

// AlignReleaseReconciliationBaseRequest describes a controlled merge of the
// current integration line into a release-derived reconciliation branch.
type AlignReleaseReconciliationBaseRequest struct {
	Repository        port.RepositoryIdentity
	Release           branch.BranchName
	Branch            branch.BranchName
	Push              bool
	CreatePullRequest bool
	Draft             bool
	DryRun            bool
}

// AlignReleaseReconciliationBaseResult records a preparation branch that is
// current with develop without mutating the delivered release line.
type AlignReleaseReconciliationBaseResult struct {
	Branch                branch.BranchName
	Release               branch.BranchName
	Develop               branch.BranchName
	PullRequest           port.PullRequest
	Evidence              port.ReleaseReconciliationEvidence
	MissingDevelopCommits bool
	Merged                bool
	Pushed                bool
	PublishedURL          string
	Quality               *port.QualityResult
	DryRun                bool
}

// AlignReleaseReconciliationBase merges develop into a ticket-bound,
// release-derived working branch. It preserves the delivered release ref and
// prepares a reviewed merge-commit pull request to develop.
func (service *ReleaseService) AlignReleaseReconciliationBase(
	ctx context.Context,
	request AlignReleaseReconciliationBaseRequest,
) (AlignReleaseReconciliationBaseResult, error) {
	if service.branches == nil || service.git == nil {
		return AlignReleaseReconciliationBaseResult{}, internalDependencyError("reconciliation-base alignment services")
	}
	if request.Release.Family() != branch.FamilyRelease {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires a release/<semver> line",
			"provide the delivered release line that must be reconciled with develop",
		)
	}
	if request.Branch.Family() != branch.FamilyChore {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires a chore preparation branch",
			"create a release-preparation branch before aligning the develop base",
		)
	}
	if request.CreatePullRequest && !request.Push {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"pull-request creation requires an explicit branch push",
			"set Push before requesting provider pull-request creation",
		)
	}
	repository, err := normalizeWorkflowRepository(request.Repository)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if _, err := service.branches.Validate(ctx, branchapp.ValidateRequest{
		Repository: repository,
		Name:       request.Branch,
	}); err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	current, err := service.git.CurrentBranch(ctx, repository)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if current.String() != request.Branch.String() {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment may mutate only the checked-out preparation branch",
			"switch to the requested reconciliation-preparation branch before retrying",
		)
	}
	releaseBase, err := branch.NewTargetBase(repository.Remote, request.Release)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	storedBase, found, err := service.git.WorkflowBase(ctx, repository, request.Branch)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if !found || storedBase.String() != releaseBase.String() {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires a branch created from the requested release line",
			"create the preparation branch with workflow release stabilize and use its recorded release base",
		)
	}
	develop := mustDevelop()
	developBase, _ := branch.NewTargetBase(repository.Remote, develop)
	result := AlignReleaseReconciliationBaseResult{
		Branch:      request.Branch,
		Release:     request.Release,
		Develop:     develop,
		PullRequest: reconciliationBaseAlignmentPullRequest(request.Branch, develop, request.Draft),
		DryRun:      request.DryRun,
	}
	if request.DryRun {
		exists, err := service.git.TargetBaseExists(ctx, repository, developBase)
		if err != nil {
			return AlignReleaseReconciliationBaseResult{}, err
		}
		if !exists {
			return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
				"the current develop target base must exist before reconciliation-base alignment",
				"fetch the selected remote and verify develop before retrying",
			)
		}
		missing, err := service.git.HasMissingBaseCommits(ctx, repository, developBase)
		if err != nil {
			return AlignReleaseReconciliationBaseResult{}, err
		}
		result.MissingDevelopCommits = missing
		return result, nil
	}
	if request.CreatePullRequest {
		if service.tickets == nil {
			return AlignReleaseReconciliationBaseResult{}, internalDependencyError("ticket publication service")
		}
		if err := service.tickets.PreflightPullRequest(ctx, repository, result.PullRequest); err != nil {
			return AlignReleaseReconciliationBaseResult{}, err
		}
	}
	clean, err := service.git.IsWorktreeClean(ctx, repository)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if !clean {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires a clean working tree",
			"commit, explicitly stash, or otherwise safely handle local changes before retrying",
		)
	}
	if err := service.git.Fetch(ctx, repository); err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	exists, err := service.git.TargetBaseExists(ctx, repository, developBase)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if !exists {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"the current develop target base must exist before reconciliation-base alignment",
			"fetch the selected remote and verify develop before retrying",
		)
	}
	missing, err := service.git.HasMissingBaseCommits(ctx, repository, developBase)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	result.MissingDevelopCommits = missing
	assessment, err := service.AssessReleaseBackmerge(ctx, AssessReleaseBackmergeRequest{
		Repository: repository,
		Release:    request.Release,
	})
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if assessment.Status != ReleaseBackmergeRequired {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires an effective release-only delta",
			"record the verified not-required reconciliation result instead of preparing a backmerge branch",
		)
	}
	result.Evidence = assessment.Evidence
	if !missing && !request.Push {
		return result, nil
	}
	if service.quality == nil {
		return AlignReleaseReconciliationBaseResult{}, internalDependencyError("quality runner")
	}
	if missing {
		message := reconciliationBaseAlignmentMergeMessage(request.Branch, request.Release)
		if err := service.git.Merge(ctx, repository, developBase, message); err != nil {
			return AlignReleaseReconciliationBaseResult{}, err
		}
		result.Merged = true
	}
	if _, err := service.branches.Validate(ctx, branchapp.ValidateRequest{
		Repository: repository,
		Name:       request.Branch,
	}); err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	quality, err := service.quality.Run(ctx, repository, port.QualityRequest{
		Families: []branch.Family{request.Branch.Family()},
	})
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	result.Quality = &quality
	if !request.Push {
		return result, nil
	}
	publication, err := service.git.PublicationState(ctx, repository, request.Branch)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	if publication == branch.PublicationUnknown {
		return AlignReleaseReconciliationBaseResult{}, invalidWorkflowInput(
			"reconciliation-base alignment requires a known branch publication state",
			"fetch the remote and resolve the branch tracking state before retrying",
		)
	}
	if err := service.git.Push(ctx, repository, request.Branch, publication == branch.PublicationUnpublished); err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	result.Pushed = true
	if !request.CreatePullRequest {
		return result, nil
	}
	publishedURL, err := service.tickets.PublishPullRequest(ctx, repository, result.PullRequest)
	if err != nil {
		return AlignReleaseReconciliationBaseResult{}, err
	}
	result.PublishedURL = publishedURL
	return result, nil
}

func reconciliationBaseAlignmentPullRequest(worker, develop branch.BranchName, draft bool) port.PullRequest {
	id, _ := worker.Ticket()
	slug, _ := worker.Slug()
	return port.PullRequest{
		Source: worker,
		Target: develop,
		Ticket: id,
		Title:  id.String() + ": " + slug.String(),
		Draft:  draft,
	}
}

func reconciliationBaseAlignmentMergeMessage(worker, release branch.BranchName) commitmsg.Message {
	id, _ := worker.Ticket()
	version, _ := release.ReleaseVersion()
	header, _ := commitmsg.NewHeader(
		commitmsg.TypeChore,
		id,
		"align release "+version.String()+" with develop for reconciliation",
		false,
	)
	message, _ := commitmsg.NewMessage(header, "", nil)
	return message
}
