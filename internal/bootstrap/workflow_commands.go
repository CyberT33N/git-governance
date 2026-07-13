package bootstrap

import (
	"context"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/application/workflow"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
	"github.com/spf13/cobra"
)

func newWorkflowCommand(application *application) *cobra.Command {
	command := &cobra.Command{
		Use:   "workflow",
		Short: "Run bounded governed Git workflows",
	}
	command.AddCommand(
		newTicketWorkflowCommand(application),
		newHotfixWorkflowCommand(application),
		newReleaseWorkflowCommand(application),
		newCleanupWorkflowCommand(application),
	)
	return command
}

func newTicketWorkflowCommand(application *application) *cobra.Command {
	command := &cobra.Command{
		Use:   "ticket",
		Short: "Start and publish regular ticket work",
	}
	command.AddCommand(
		newTicketStartCommand(application),
		newTicketPublishCommand(application),
	)
	return command
}

func newTicketStartCommand(application *application) *cobra.Command {
	var (
		familyRaw     string
		keyRaw        string
		numberRaw     string
		slugRaw       string
		createScratch bool
		scratchSlug   string
	)
	command := &cobra.Command{
		Use:   "start",
		Short: "Create a regular ticket branch and optionally a private scratch branch",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			family, err := application.resolveFamily(command.Context(), familyRaw, false)
			if err != nil {
				return err
			}
			inputs.add("branch family", family.String())
			key, err := application.resolveKey(command.Context(), services, keyRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket key", key.String())
			number, err := application.resolveNumber(command.Context(), numberRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket number", number.String())
			id := ticket.NewID(key, number)
			slug, err := application.resolveSlug(command.Context(), slugRaw, "Branch description")
			if err != nil {
				return err
			}
			inputs.add("branch description", slug.String())
			if !createScratch && application.promptAvailable() && !application.options.yes {
				createScratch, err = application.prompt().Confirm(command.Context(), port.ConfirmRequest{
					Label:       "Create a private scratch branch?",
					Description: "Scratch branches are for uncertain exploration only. Do not open a pull request from them; move stable work to the official ticket branch.",
					Default:     false,
				})
				if err != nil {
					return err
				}
			}
			inputs.add("create scratch branch", boolString(createScratch))
			var parsedScratchSlug branch.Slug
			if scratchSlug != "" {
				parsedScratchSlug, err = branch.ParseSlug(scratchSlug)
				if err != nil {
					return err
				}
				inputs.add("scratch branch description", parsedScratchSlug.String())
			}
			if err := application.confirmMutation(command.Context(), "Start ticket workflow", "Create the official ticket branch and any selected scratch branch?"); err != nil {
				return err
			}
			result, err := services.tickets.StartTicket(command.Context(), workflow.StartTicketRequest{
				Repository:    repository,
				Family:        family,
				Ticket:        id,
				Slug:          slug,
				CreateScratch: createScratch,
				ScratchSlug:   parsedScratchSlug,
				DryRun:        application.options.dryRun,
			})
			if err != nil {
				return err
			}
			fields := map[string]string{
				"officialBranch": result.Official.Name.String(),
				"activeBranch":   result.Active.String(),
				"dryRun":         boolString(application.options.dryRun),
			}
			if result.Scratch != nil {
				fields["scratchBranch"] = result.Scratch.Name.String()
			}
			return application.report(command, port.Report{
				Operation: "workflow.ticket.start",
				Summary: application.withInteractiveFetchSummary(
					"Ticket workflow start completed.",
					repository.Remote,
					fetchCompleted(result.Official.DryRun, result.Official.Plan),
				),
				Fields: fields,
			})
		}),
	}
	command.Flags().StringVar(&familyRaw, "family", "", "regular ticket branch family")
	command.Flags().StringVar(&keyRaw, "key", "", "ticket key")
	command.Flags().StringVar(&numberRaw, "ticket", "", "ticket number")
	command.Flags().StringVar(&slugRaw, "slug", "", "kebab-case branch description")
	command.Flags().BoolVar(&createScratch, "scratch", false, "create a private scratch branch")
	command.Flags().StringVar(&scratchSlug, "scratch-slug", "", "optional scratch branch slug")
	return command
}

func newTicketPublishCommand(application *application) *cobra.Command {
	var (
		branchRaw         string
		baseRaw           string
		scratchTargetRaw  string
		scratchMessageRaw string
		scratchFamilyRaw  string
		scratchSubjectRaw string
		push              bool
		draft             bool
	)
	command := &cobra.Command{
		Use:   "publish",
		Short: "Validate, synchronize, optionally push, and prepare a ticket pull request",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, branchRaw, repository)
			if err != nil {
				return err
			}
			inputs.add("ticket branch", name.String())
			var (
				scratchTarget  *branch.BranchName
				scratchMessage *commitmsg.Message
			)
			if name.Family() == branch.FamilyScratch {
				var explicitTarget *branch.BranchName
				if scratchTargetRaw != "" {
					target, err := branch.ParseName(scratchTargetRaw)
					if err != nil {
						return err
					}
					explicitTarget = &target
				}
				target, err := services.scratch.ResolveTarget(command.Context(), repository, name, explicitTarget)
				if err != nil {
					return err
				}
				scratchTarget = &target
				inputs.add("official ticket branch", target.String())
				message, err := application.resolveScratchMergeMessage(
					command.Context(),
					scratchMessageRaw,
					scratchFamilyRaw,
					scratchSubjectRaw,
					target,
				)
				if err != nil {
					return err
				}
				scratchMessage = &message
				inputs.add("squash commit family", message.Header().Type().String())
				inputs.add("squash commit description", message.Header().Subject())
			} else if scratchTargetRaw != "" || scratchMessageRaw != "" || scratchFamilyRaw != "" || scratchSubjectRaw != "" {
				return invalidOption("scratch transfer", "configured", "--target, --message, --type, and --subject are only supported when publishing from scratch")
			}
			base, err := parseBase(baseRaw, repository.Remote)
			if err != nil {
				return err
			}
			if base != nil {
				inputs.add("target base", base.String())
			}
			label := "Publish ticket workflow"
			description := "Validate the commit series, synchronize safely, and optionally push the branch?"
			if scratchTarget != nil && scratchMessage != nil {
				label = "Publish ticket workflow from scratch"
				description = "You are on private scratch branch " + name.String() +
					". Squash-merge it into " + scratchTarget.String() + " as " +
					scratchMessage.Header().String() +
					", then validate, synchronize safely, and optionally push the official branch?"
			}
			if err := application.confirmMutation(command.Context(), label, description); err != nil {
				return err
			}
			result, err := services.tickets.PublishTicket(command.Context(), workflow.PublishTicketRequest{
				Repository:     repository,
				Branch:         name,
				Base:           base,
				ScratchTarget:  scratchTarget,
				ScratchMessage: scratchMessage,
				Draft:          draft,
				DryRun:         application.options.dryRun,
			})
			if err != nil {
				if isScratchMergeConflict(err) && application.promptAvailable() && scratchTarget != nil && scratchMessage != nil {
					scratchMerge, resumeErr := application.resumeScratchMergeAfterConflict(
						command.Context(),
						services,
						repository,
						name,
						*scratchTarget,
						*scratchMessage,
					)
					if resumeErr != nil {
						return resumeErr
					}
					result, err = services.tickets.PublishTicket(command.Context(), workflow.PublishTicketRequest{
						Repository: repository,
						Branch:     *scratchTarget,
						Base:       base,
						Draft:      draft,
						DryRun:     application.options.dryRun,
					})
					if err == nil {
						result.ScratchMerge = &scratchMerge
					}
				}
			}
			if err != nil {
				if !application.promptAvailable() || !isRebaseConflict(err) {
					return err
				}
				resumeBranch := name
				if scratchTarget != nil {
					resumeBranch = *scratchTarget
				}
				result, err = application.resumeTicketPublishAfterRebaseConflict(
					command.Context(),
					services,
					repository,
					resumeBranch,
					base,
					draft,
				)
				if err != nil {
					return err
				}
				if scratchTarget != nil && scratchMessage != nil {
					result.ScratchMerge = &branchapp.ScratchMergeResult{
						Source:    name,
						Target:    *scratchTarget,
						Message:   *scratchMessage,
						Committed: true,
					}
				}
			}
			if err := application.reportTicketSynchronization(command, result.Sync, result.DryRun); err != nil {
				return err
			}
			if err := application.completeTicketPublishInteraction(
				command.Context(),
				services,
				repository,
				&result,
				push,
			); err != nil {
				return err
			}
			fields := map[string]string{
				"branch":               result.Branch.String(),
				"pushed":               boolString(result.Pushed),
				"syncAction":           result.Sync.RecommendedAction,
				"pullRequestSource":    result.PullRequest.Source.String(),
				"pullRequestTarget":    result.PullRequest.Target.String(),
				"pullRequestTitle":     result.PullRequest.Title,
				"publishedPullRequest": result.PublishedURL,
			}
			switch {
			case result.PublishedURL != "":
				fields["pullRequestPublication"] = "created"
			case services.tickets.HasPullRequestPublisher():
				fields["pullRequestPublication"] = "not requested"
			default:
				fields["pullRequestPublication"] = "intent-only; no hosting-provider adapter is configured"
			}
			if result.ScratchMerge != nil {
				fields["scratchBranch"] = result.ScratchMerge.Source.String()
				fields["squashMerged"] = boolString(result.ScratchMerge.Committed)
				fields["squashCommit"] = result.ScratchMerge.Message.Header().String()
			}
			addQualityFields(fields, result)
			return application.report(command, port.Report{
				Operation: "workflow.ticket.publish",
				Summary: application.withInteractiveFetchSummary(
					"Ticket publish workflow completed.",
					repository.Remote,
					!result.DryRun,
				),
				Fields: fields,
				Data:   result.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&branchRaw, "branch", "", "ticket branch; defaults to the current branch")
	command.Flags().StringVar(&baseRaw, "base", "", "explicit base for hotfix ticket publication")
	command.Flags().StringVar(&scratchTargetRaw, "target", "", "optional local official target when publishing from scratch")
	command.Flags().StringVar(&scratchFamilyRaw, "type", "", "commit family for a scratch squash transfer")
	command.Flags().StringVar(&scratchSubjectRaw, "subject", "", "commit description for a scratch squash transfer")
	command.Flags().StringVar(&scratchMessageRaw, "message", "", "complete commit message compatibility input for a scratch squash transfer")
	command.Flags().BoolVar(&push, "push", false, "push the branch after validation")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func (application *application) reportTicketSynchronization(
	command *cobra.Command,
	result branchapp.SyncResult,
	dryRun bool,
) error {
	if dryRun || !application.promptAvailable() {
		return nil
	}
	summary := "Target-base synchronization completed without a rebase."
	switch result.RecommendedAction {
	case "rebased":
		summary = "Rebase completed successfully; the official branch is synchronized with its target base."
	case "none":
		summary = "No rebase was performed because the target base has no commits missing from the branch."
	case "merge":
		summary = "No rebase was performed because the branch is already published; a controlled merge is required if its target base advanced."
	}
	return application.report(command, port.Report{
		Operation: "workflow.ticket.publish.sync",
		Summary:   summary,
		Fields: map[string]string{
			"branch":     result.Name.String(),
			"targetBase": result.Base.String(),
			"syncAction": result.RecommendedAction,
		},
	})
}

func (application *application) resumeTicketPublishAfterRebaseConflict(
	ctx context.Context,
	services services,
	repository port.RepositoryIdentity,
	name branch.BranchName,
	base *branch.TargetBase,
	draft bool,
) (workflow.PublishTicketResult, error) {
	for {
		action, err := application.prompt().Select(ctx, port.SelectRequest{
			Label: "Rebase conflict requires resolution",
			Description: "Git paused the rebase because conflicts remain. Resolve every conflict, stage the resolutions, then select Retry. " +
				"Selecting Cancel leaves the Git rebase untouched.",
			Options: []port.SelectOption{
				{Value: "retry", Label: "Retry", Description: "Continue the resolved rebase and resume this ticket publication."},
				{Value: "cancel", Label: "Cancel", Description: "Leave the rebase paused for manual resolution."},
			},
			Default: "retry",
		})
		if err != nil {
			return workflow.PublishTicketResult{}, err
		}
		if action == "cancel" {
			return workflow.PublishTicketResult{}, problem.New(problem.Details{
				Code:        problem.CodeOperationCancelled,
				Category:    problem.CategoryCancelled,
				Field:       "rebase retry",
				Expected:    "Retry after resolving the rebase conflicts",
				Rule:        "the workflow leaves unresolved Git conflicts for explicit user resolution",
				Remediation: "resolve and stage the conflicts, then rerun ticket publish to resume the paused rebase",
			})
		}
		result, err := services.tickets.ResumeTicketPublish(ctx, workflow.ResumeTicketPublishRequest{
			Repository: repository,
			Branch:     name,
			Base:       base,
			Draft:      draft,
		})
		if err == nil {
			return result, nil
		}
		if !isRebaseConflict(err) {
			return workflow.PublishTicketResult{}, err
		}
	}
}

func (application *application) resumeScratchMergeAfterConflict(
	ctx context.Context,
	services services,
	repository port.RepositoryIdentity,
	source, target branch.BranchName,
	message commitmsg.Message,
) (branchapp.ScratchMergeResult, error) {
	for {
		action, err := application.prompt().Select(ctx, port.SelectRequest{
			Label: "Scratch merge conflict requires resolution",
			Description: "Git paused the scratch squash transfer because conflicts remain. Resolve every conflict, stage the resolutions, then select Retry. " +
				"Selecting Cancel leaves the squash transfer untouched.",
			Options: []port.SelectOption{
				{Value: "retry", Label: "Retry", Description: "Commit the resolved squash transfer and continue ticket publication."},
				{Value: "cancel", Label: "Cancel", Description: "Leave the unresolved scratch transfer for manual resolution."},
			},
			Default: "retry",
		})
		if err != nil {
			return branchapp.ScratchMergeResult{}, err
		}
		if action == "cancel" {
			return branchapp.ScratchMergeResult{}, problem.New(problem.Details{
				Code:        problem.CodeOperationCancelled,
				Category:    problem.CategoryCancelled,
				Field:       "scratch merge retry",
				Expected:    "Retry after resolving the scratch merge conflicts",
				Rule:        "the workflow leaves unresolved Git conflicts for explicit user resolution",
				Remediation: "resolve and stage the conflicts, then rerun ticket publish to resume the scratch transfer",
			})
		}
		result, err := services.scratch.Resume(ctx, branchapp.ScratchMergeRequest{
			Repository: repository,
			Source:     source,
			Target:     &target,
			Message:    message,
		})
		if err == nil {
			return result, nil
		}
		if !isScratchMergeConflict(err) {
			return branchapp.ScratchMergeResult{}, err
		}
	}
}

func (application *application) completeTicketPublishInteraction(
	ctx context.Context,
	services services,
	repository port.RepositoryIdentity,
	result *workflow.PublishTicketResult,
	requestedPush bool,
) error {
	if result == nil || result.DryRun {
		return nil
	}
	push := requestedPush
	if application.promptAvailable() && !application.options.yes {
		confirmed, err := application.prompt().Confirm(ctx, port.ConfirmRequest{
			Label:       "Push official ticket branch",
			Description: "Push " + result.Branch.String() + " after the completed synchronization? The first push configures its matching upstream branch.",
			Default:     requestedPush,
		})
		if err != nil {
			return err
		}
		push = confirmed
	}
	if !push {
		return nil
	}
	base := result.Sync.Base
	if err := services.tickets.PushPreparedTicket(ctx, repository, result.Branch, &base); err != nil {
		return err
	}
	result.Pushed = true
	if !services.tickets.HasPullRequestPublisher() {
		return nil
	}

	createPullRequest := requestedPush
	if application.promptAvailable() && !application.options.yes {
		confirmed, err := application.prompt().Confirm(ctx, port.ConfirmRequest{
			Label: "Create pull request",
			Description: "Create the pull request from " + result.PullRequest.Source.String() +
				" to " + result.PullRequest.Target.String() + " now?",
			Default: false,
		})
		if err != nil {
			return err
		}
		createPullRequest = confirmed
	}
	if !createPullRequest {
		return nil
	}
	publishedURL, err := services.tickets.PublishPullRequest(ctx, result.PullRequest)
	if err != nil {
		return err
	}
	result.PublishedURL = publishedURL
	return nil
}

func isRebaseConflict(err error) bool {
	typed, ok := problem.As(err)
	return ok && typed.Code == problem.CodeRebaseConflict
}

func isScratchMergeConflict(err error) bool {
	typed, ok := problem.As(err)
	return ok && typed.Code == problem.CodeScratchMergeConflict
}

func newHotfixWorkflowCommand(application *application) *cobra.Command {
	var (
		keyRaw      string
		numberRaw   string
		slugRaw     string
		affectedRaw string
	)
	command := &cobra.Command{
		Use:   "hotfix",
		Short: "Start a hotfix from the active line that contains the defect",
	}
	start := &cobra.Command{
		Use:   "start",
		Short: "Create a hotfix branch from main, release, or support",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			key, err := application.resolveKey(command.Context(), services, keyRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket key", key.String())
			number, err := application.resolveNumber(command.Context(), numberRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket number", number.String())
			slug, err := application.resolveSlug(command.Context(), slugRaw, "Hotfix description")
			if err != nil {
				return err
			}
			inputs.add("hotfix description", slug.String())
			affected, err := application.resolveAffectedLine(command.Context(), affectedRaw)
			if err != nil {
				return err
			}
			inputs.add("affected line", affected.String())
			if err := application.confirmMutation(command.Context(), "Start hotfix", "Create a hotfix from "+affected.String()+"?"); err != nil {
				return err
			}
			result, err := services.releases.StartHotfix(command.Context(), workflow.StartHotfixRequest{
				Repository:   repository,
				Ticket:       ticket.NewID(key, number),
				Slug:         slug,
				AffectedLine: affected,
				DryRun:       application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.hotfix.start",
				Summary: application.withInteractiveFetchSummary(
					"Hotfix branch created.",
					repository.Remote,
					fetchCompleted(result.DryRun, result.Plan),
				),
				Fields: map[string]string{
					"branch": result.Name.String(),
					"base":   result.Base.String(),
					"dryRun": boolString(result.DryRun),
				},
			})
		}),
	}
	start.Flags().StringVar(&keyRaw, "key", "", "ticket key")
	start.Flags().StringVar(&numberRaw, "ticket", "", "ticket number")
	start.Flags().StringVar(&slugRaw, "slug", "", "kebab-case hotfix description")
	start.Flags().StringVar(&affectedRaw, "affected-line", "", "main, release/<semver>, or support/<major.minor>")
	command.AddCommand(
		start,
		newHotfixPublishCommand(application),
		newHotfixPropagateCommand(application),
	)
	return command
}

func newHotfixPublishCommand(application *application) *cobra.Command {
	var (
		branchRaw   string
		affectedRaw string
		push        bool
		draft       bool
	)
	command := &cobra.Command{
		Use:   "publish",
		Short: "Validate, publish, and prepare a pull request for a hotfix",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, branchRaw, repository)
			if err != nil {
				return err
			}
			inputs.add("hotfix branch", name.String())
			if name.Family() != branch.FamilyHotfix {
				return invalidOption("branch", name.String(), "a hotfix/<ticket>-<slug> branch")
			}
			affected, err := application.resolveAffectedLine(command.Context(), affectedRaw)
			if err != nil {
				return err
			}
			inputs.add("affected line", affected.String())
			base, err := branch.NewTargetBase(repository.Remote, affected)
			if err != nil {
				return err
			}
			if err := application.confirmMutation(
				command.Context(),
				"Publish hotfix",
				"Validate the hotfix and prepare its pull request for "+affected.String()+"?",
			); err != nil {
				return err
			}
			result, err := services.tickets.PublishTicket(command.Context(), workflow.PublishTicketRequest{
				Repository: repository,
				Branch:     name,
				Base:       &base,
				Push:       push,
				Draft:      draft,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			fields := map[string]string{
				"branch":               result.Branch.String(),
				"affectedLine":         affected.String(),
				"pushed":               boolString(result.Pushed),
				"pullRequestSource":    result.PullRequest.Source.String(),
				"pullRequestTarget":    result.PullRequest.Target.String(),
				"pullRequestTitle":     result.PullRequest.Title,
				"publishedPullRequest": result.PublishedURL,
			}
			addQualityFields(fields, result)
			return application.report(command, port.Report{
				Operation: "workflow.hotfix.publish",
				Summary: application.withInteractiveFetchSummary(
					"Hotfix publish workflow completed.",
					repository.Remote,
					!result.DryRun,
				),
				Fields: fields,
				Data:   result.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&branchRaw, "branch", "", "hotfix branch; defaults to the current branch")
	command.Flags().StringVar(&affectedRaw, "affected-line", "", "main, release/<semver>, or support/<major.minor>")
	command.Flags().BoolVar(&push, "push", false, "push the hotfix branch after validation")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func newHotfixPropagateCommand(application *application) *cobra.Command {
	var (
		sourceRaw string
		targetRaw string
		commitID  string
		slugRaw   string
		push      bool
		draft     bool
	)
	command := &cobra.Command{
		Use:   "propagate",
		Short: "Forward-port or backport one reviewed hotfix commit",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			source, err := currentOrSpecified(command.Context(), services, sourceRaw, repository)
			if err != nil {
				return err
			}
			inputs.add("source branch", source.String())
			if source.Family() != branch.FamilyHotfix {
				return invalidOption("source", source.String(), "a hotfix/<ticket>-<slug> branch")
			}
			target, err := application.resolvePropagationTarget(command.Context(), targetRaw)
			if err != nil {
				return err
			}
			inputs.add("target line", target.String())
			commitID, err = application.resolveReviewedCommit(command.Context(), commitID)
			if err != nil {
				return err
			}
			inputs.add("reviewed source commit", commitID)
			var slug branch.Slug
			if slugRaw != "" {
				slug, err = branch.ParseSlug(slugRaw)
				if err != nil {
					return err
				}
				inputs.add("branch description", slug.String())
			}
			if err := application.confirmMutation(
				command.Context(),
				"Propagate hotfix",
				"Create a controlled fix branch from "+target.String()+" and cherry-pick "+commitID+" with -x?",
			); err != nil {
				return err
			}
			result, err := services.releases.PropagateHotfix(command.Context(), workflow.PropagateHotfixRequest{
				Repository: repository,
				Source:     source,
				TargetLine: target,
				CommitID:   commitID,
				Slug:       slug,
				Push:       push,
				Draft:      draft,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.hotfix.propagate",
				Summary: application.withInteractiveFetchSummary(
					"Hotfix propagation workflow completed.",
					repository.Remote,
					fetchCompleted(result.Branch.DryRun, result.Branch.Plan) || !result.Publication.DryRun,
				),
				Fields: map[string]string{
					"source":               source.String(),
					"target":               target.String(),
					"branch":               result.Branch.Name.String(),
					"cherryPicked":         boolString(result.CherryPicked),
					"pullRequestSource":    result.Publication.PullRequest.Source.String(),
					"pullRequestTarget":    result.Publication.PullRequest.Target.String(),
					"publishedPullRequest": result.Publication.PublishedURL,
				},
				Data: result.Publication.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&sourceRaw, "source", "", "hotfix source branch; defaults to the current branch")
	command.Flags().StringVar(&targetRaw, "target-line", "", "main, develop, release/<semver>, or support/<major.minor>")
	command.Flags().StringVar(&commitID, "commit", "", "reviewed source commit SHA")
	command.Flags().StringVar(&slugRaw, "slug", "", "optional kebab-case propagation branch description")
	command.Flags().BoolVar(&push, "push", false, "push the propagation branch after validation")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func newReleaseWorkflowCommand(application *application) *cobra.Command {
	command := &cobra.Command{
		Use:   "release",
		Short: "Cut and reconcile governed release lines",
	}
	command.AddCommand(
		newReleaseCutCommand(application),
		newReleaseStabilizeCommand(application),
		newReleasePublishStabilizationCommand(application),
		newReleasePromotionCommand(application),
		newReleaseBackmergeCommand(application),
		newSupportPrepareCommand(application),
	)
	return command
}

func newCleanupWorkflowCommand(application *application) *cobra.Command {
	var branchRaw string
	command := &cobra.Command{
		Use:   "cleanup",
		Short: "Delete a local private scratch branch without deleting a remote branch",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, branchRaw, repository)
			if err != nil {
				return err
			}
			inputs.add("scratch branch", name.String())
			if err := application.confirmMutation(
				command.Context(),
				"Clean up scratch branch",
				"Delete "+name.String()+" locally? Official branch lifecycle and remote deletion remain GitHub, GitLab, or CI responsibilities.",
			); err != nil {
				return err
			}
			result, err := services.releases.CleanupBranch(command.Context(), workflow.CleanupBranchRequest{
				Repository: repository,
				Branch:     name,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.cleanup",
				Summary:   "Branch cleanup completed.",
				Fields: map[string]string{
					"branch":          result.Branch.String(),
					"deletedLocal":    boolString(result.DeletedLocal),
					"metadataCleared": boolString(result.MetadataCleared),
					"dryRun":          boolString(result.DryRun),
				},
			})
		}),
	}
	command.Flags().StringVar(&branchRaw, "branch", "", "local scratch branch; defaults to the current branch")
	return command
}

func addQualityFields(fields map[string]string, result workflow.PublishTicketResult) {
	fields["qualityStatus"] = string(result.Quality.Status)
	fields["qualityDetail"] = result.Quality.Detail
	if result.PostMutationQuality != nil {
		fields["postMutationQualityStatus"] = string(result.PostMutationQuality.Status)
		fields["postMutationQualityDetail"] = result.PostMutationQuality.Detail
	}
}

func newReleaseCutCommand(application *application) *cobra.Command {
	var versionRaw string
	command := &cobra.Command{
		Use:   "cut",
		Short: "Prepare a protected CI request for release/<semver> from develop",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			version, err := application.resolveReleaseVersion(command.Context(), versionRaw)
			if err != nil {
				return err
			}
			inputs.add("release version", version.String())
			if err := application.confirmMutation(command.Context(), "Cut release", "Create release/"+version.String()+" from origin/develop?"); err != nil {
				return err
			}
			result, err := services.releases.CutRelease(command.Context(), workflow.CutReleaseRequest{
				Repository: repository,
				Version:    version,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.release.cut",
				Summary: application.withInteractiveFetchSummary(
					"Protected release-line creation intent prepared.",
					repository.Remote,
					fetchCompleted(result.DryRun, result.Plan),
				),
				Fields: map[string]string{
					"branch":   result.Intent.Branch.String(),
					"base":     result.Intent.Source.String(),
					"workflow": result.Intent.Workflow,
					"dryRun":   boolString(result.DryRun),
				},
				Data: result.Intent,
			})
		}),
	}
	command.Flags().StringVar(&versionRaw, "version", "", "release semantic version")
	return command
}

func newReleaseStabilizeCommand(application *application) *cobra.Command {
	var (
		releaseRaw string
		kindRaw    string
		keyRaw     string
		numberRaw  string
		slugRaw    string
		switchTo   bool
	)
	command := &cobra.Command{
		Use:   "stabilize",
		Short: "Create a permitted stabilization branch from a frozen release line",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			release, err := application.resolveReleaseLine(
				command.Context(),
				releaseRaw,
				"Release line",
				"Enter release/<semantic-version> for the frozen line that contains this stabilization task. Examples: release/2.8.0, release/2.8.0-rc.1.",
			)
			if err != nil {
				return err
			}
			inputs.add("release line", release.String())
			kind, err := application.resolveStabilizationKind(command.Context(), kindRaw)
			if err != nil {
				return err
			}
			inputs.add("stabilization kind", string(kind))
			key, err := application.resolveKey(command.Context(), services, keyRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket key", key.String())
			number, err := application.resolveNumber(command.Context(), numberRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket number", number.String())
			slug, err := application.resolveSlug(command.Context(), slugRaw, "Stabilization description")
			if err != nil {
				return err
			}
			inputs.add("stabilization description", slug.String())
			if err := application.confirmMutation(
				command.Context(),
				"Create release stabilization branch",
				"Create a "+kindRaw+" stabilization branch from "+release.String()+"?",
			); err != nil {
				return err
			}
			result, err := services.releases.CreateReleaseStabilization(command.Context(), workflow.CreateReleaseStabilizationRequest{
				Repository: repository,
				Release:    release,
				Ticket:     ticket.NewID(key, number),
				Slug:       slug,
				Kind:       kind,
				Switch:     &switchTo,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.release.stabilize",
				Summary: application.withInteractiveFetchSummary(
					"Release stabilization branch created.",
					repository.Remote,
					fetchCompleted(result.DryRun, result.Plan),
				),
				Fields: map[string]string{
					"release": release.String(),
					"branch":  result.Name.String(),
					"base":    result.Base.String(),
					"kind":    string(kind),
					"dryRun":  boolString(result.DryRun),
				},
			})
		}),
	}
	command.Flags().StringVar(&releaseRaw, "release", "", "release/<semver> line")
	command.Flags().StringVar(&kindRaw, "kind", "", "blocker, docs, or release-prep")
	command.Flags().StringVar(&keyRaw, "key", "", "ticket key")
	command.Flags().StringVar(&numberRaw, "ticket", "", "ticket number")
	command.Flags().StringVar(&slugRaw, "slug", "", "kebab-case stabilization description")
	command.Flags().BoolVar(&switchTo, "switch", true, "switch to the stabilization branch after creating it")
	return command
}

func newReleasePublishStabilizationCommand(application *application) *cobra.Command {
	var (
		branchRaw  string
		releaseRaw string
		push       bool
		draft      bool
	)
	command := &cobra.Command{
		Use:   "publish-stabilization",
		Short: "Validate and prepare a stabilization pull request for its release line",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, branchRaw, repository)
			if err != nil {
				return err
			}
			inputs.add("stabilization branch", name.String())
			switch name.Family() {
			case branch.FamilyFix, branch.FamilyDocs, branch.FamilyChore:
			default:
				return invalidOption("branch", name.String(), "a release stabilization fix, docs, or chore branch")
			}
			release, err := application.resolveReleaseLine(
				command.Context(),
				releaseRaw,
				"Release line",
				"Enter the release/<semantic-version> line from which the stabilization branch was created. Example: release/2.8.0.",
			)
			if err != nil {
				return err
			}
			inputs.add("release line", release.String())
			base, err := branch.NewTargetBase(repository.Remote, release)
			if err != nil {
				return err
			}
			if err := application.confirmMutation(
				command.Context(),
				"Publish release stabilization",
				"Validate the stabilization branch and prepare its pull request for "+release.String()+"?",
			); err != nil {
				return err
			}
			result, err := services.tickets.PublishTicket(command.Context(), workflow.PublishTicketRequest{
				Repository:      repository,
				Branch:          name,
				Base:            &base,
				WorkflowManaged: true,
				Push:            push,
				Draft:           draft,
				DryRun:          application.options.dryRun,
			})
			if err != nil {
				return err
			}
			fields := map[string]string{
				"branch":               result.Branch.String(),
				"release":              release.String(),
				"pushed":               boolString(result.Pushed),
				"pullRequestSource":    result.PullRequest.Source.String(),
				"pullRequestTarget":    result.PullRequest.Target.String(),
				"publishedPullRequest": result.PublishedURL,
			}
			addQualityFields(fields, result)
			return application.report(command, port.Report{
				Operation: "workflow.release.publish-stabilization",
				Summary: application.withInteractiveFetchSummary(
					"Release stabilization pull request prepared.",
					repository.Remote,
					!result.DryRun,
				),
				Fields: fields,
				Data:   result.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&branchRaw, "branch", "", "stabilization branch; defaults to the current branch")
	command.Flags().StringVar(&releaseRaw, "release", "", "release/<semver> target line")
	command.Flags().BoolVar(&push, "push", false, "push the stabilization branch after validation")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func newReleasePromotionCommand(application *application) *cobra.Command {
	var (
		releaseRaw string
		draft      bool
	)
	command := &cobra.Command{
		Use:   "promote",
		Short: "Prepare the release/<semver> to main pull request",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			release, err := application.resolveReleaseLine(
				command.Context(),
				releaseRaw,
				"Release branch",
				"Enter the approved release/<semantic-version> branch to promote to main. Example: release/2.8.0.",
			)
			if err != nil {
				return err
			}
			inputs.add("release branch", release.String())
			result, err := services.releases.PrepareReleasePromotion(command.Context(), workflow.PrepareReleasePromotionRequest{
				Repository: repository,
				Release:    release,
				Draft:      draft,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.release.promote",
				Summary:   "Release promotion pull request prepared.",
				Fields: map[string]string{
					"source": result.PullRequest.Source.String(),
					"target": result.PullRequest.Target.String(),
					"title":  result.PullRequest.Title,
					"url":    result.PublishedURL,
				},
				Data: result.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&releaseRaw, "release", "", "release/<semver> branch")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func newReleaseBackmergeCommand(application *application) *cobra.Command {
	var releaseRaw string
	var draft bool
	command := &cobra.Command{
		Use:   "backmerge",
		Short: "Prepare the release/<semver> to develop pull request",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			release, err := application.resolveReleaseLine(
				command.Context(),
				releaseRaw,
				"Release branch",
				"Enter the completed release/<semantic-version> branch to backmerge into develop. Example: release/2.8.0.",
			)
			if err != nil {
				return err
			}
			inputs.add("release branch", release.String())
			result, err := services.releases.PrepareReleaseBackmerge(command.Context(), workflow.PrepareReleaseBackmergeRequest{
				Repository: repository,
				Release:    release,
				Draft:      draft,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.release.backmerge",
				Summary:   "Release backmerge pull request prepared.",
				Fields: map[string]string{
					"source": result.PullRequest.Source.String(),
					"target": result.PullRequest.Target.String(),
					"title":  result.PullRequest.Title,
					"url":    result.PublishedURL,
				},
				Data: result.PullRequest,
			})
		}),
	}
	command.Flags().StringVar(&releaseRaw, "release", "", "release/<semver> branch")
	command.Flags().BoolVar(&draft, "draft", false, "mark the pull request intent as a draft")
	return command
}

func newSupportPrepareCommand(application *application) *cobra.Command {
	var versionRaw string
	command := &cobra.Command{
		Use:   "support",
		Short: "Prepare a protected CI request for support/<major.minor> from main",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			version, err := application.resolveSupportVersion(command.Context(), versionRaw)
			if err != nil {
				return err
			}
			inputs.add("support version", version.String())
			if err := application.confirmMutation(command.Context(), "Create support line", "Create support/"+version.String()+" from origin/main?"); err != nil {
				return err
			}
			result, err := services.releases.PrepareSupport(command.Context(), workflow.PrepareSupportRequest{
				Repository: repository,
				Version:    version,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "workflow.release.support",
				Summary: application.withInteractiveFetchSummary(
					"Protected support-line creation intent prepared.",
					repository.Remote,
					fetchCompleted(result.DryRun, result.Plan),
				),
				Fields: map[string]string{
					"branch":   result.Intent.Branch.String(),
					"base":     result.Intent.Source.String(),
					"workflow": result.Intent.Workflow,
					"dryRun":   boolString(result.DryRun),
				},
				Data: result.Intent,
			})
		}),
	}
	command.Flags().StringVar(&versionRaw, "version", "", "support major.minor version")
	return command
}
