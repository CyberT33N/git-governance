package bootstrap

import (
	"strings"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/spf13/cobra"
)

func newBranchCommand(application *application) *cobra.Command {
	command := &cobra.Command{
		Use:   "branch",
		Short: "List, validate, create, and synchronize governed branches",
	}
	command.AddCommand(
		newBranchListCommand(application),
		newBranchValidateCommand(application),
		newBranchCreateCommand(application),
		newBranchSyncBaseCommand(application),
	)
	return command
}

func newBranchListCommand(application *application) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every supported branch family",
		RunE: func(command *cobra.Command, _ []string) error {
			families := branchapp.ListFamilies()
			fields := make(map[string]string, len(families))
			for _, family := range families {
				fields[family.Family.String()] = family.Pattern + " — " + family.Description
			}
			return application.report(command, port.Report{
				Operation: "branch.list",
				Summary:   "Supported branch families:",
				Fields:    fields,
				Data:      families,
			})
		},
	}
}

func newBranchValidateCommand(application *application) *cobra.Command {
	var nameRaw string
	command := &cobra.Command{
		Use:   "validate",
		Short: "Validate a branch name and its local Git reference",
		RunE: func(command *cobra.Command, _ []string) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, nameRaw, repository)
			if err != nil {
				return err
			}
			result, err := services.branches.Validate(command.Context(), branchapp.ValidateRequest{
				Repository: repository,
				Name:       name,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "branch.validate",
				Summary:   "Branch is valid.",
				Fields: map[string]string{
					"branch": result.Name.String(),
					"family": result.Name.Family().String(),
				},
			})
		},
	}
	command.Flags().StringVar(&nameRaw, "branch", "", "branch name; defaults to the current branch")
	return command
}

func newBranchCreateCommand(application *application) *cobra.Command {
	var (
		familyRaw string
		keyRaw    string
		numberRaw string
		slugRaw   string
		baseRaw   string
		switchTo  bool
	)
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a regular governed branch",
		RunE: func(command *cobra.Command, _ []string) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			family, err := application.resolveFamily(command.Context(), familyRaw, false)
			if err != nil {
				return err
			}
			key, err := application.resolveKey(command.Context(), services, keyRaw)
			if err != nil {
				return err
			}
			number, err := application.resolveNumber(command.Context(), numberRaw)
			if err != nil {
				return err
			}
			id, err := parseTicketParts(key.String(), number.String())
			if err != nil {
				return err
			}
			slug, err := application.resolveSlug(command.Context(), slugRaw, "Branch description")
			if err != nil {
				return err
			}
			var base *branch.TargetBase
			if family == branch.FamilyScratch {
				base, err = application.resolveScratchBase(command.Context(), baseRaw, repository.Remote)
			} else {
				base, err = parseBase(baseRaw, repository.Remote)
			}
			if err != nil {
				return err
			}
			name, err := branch.NewTicketBranch(family, id, slug)
			if err != nil {
				return err
			}
			if err := application.confirmMutation(command.Context(), "Create branch", "Create "+name.String()+" from the canonical target base?"); err != nil {
				return err
			}
			result, err := services.branches.Create(command.Context(), branchapp.CreateRequest{
				Repository: repository,
				Family:     family,
				Ticket:     id,
				Slug:       slug,
				Base:       base,
				Switch:     &switchTo,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "branch.create",
				Summary:   branchCreationSummary(result),
				Fields: map[string]string{
					"branch":   result.Name.String(),
					"base":     result.Base.String(),
					"switched": boolString(result.Switched),
					"dryRun":   boolString(result.DryRun),
					"plan":     planText(result.Plan),
				},
			})
		},
	}
	command.Flags().StringVar(&familyRaw, "family", "", "branch family")
	command.Flags().StringVar(&keyRaw, "key", "", "ticket key")
	command.Flags().StringVar(&numberRaw, "ticket", "", "ticket number")
	command.Flags().StringVar(&slugRaw, "slug", "", "kebab-case branch description")
	command.Flags().StringVar(&baseRaw, "base", "", "explicit base for eligible branch families")
	command.Flags().BoolVar(&switchTo, "switch", true, "switch to the branch after creating it")
	return command
}

func newBranchSyncBaseCommand(application *application) *cobra.Command {
	var (
		nameRaw      string
		baseRaw      string
		strategyRaw  string
		mergeMessage string
	)
	command := &cobra.Command{
		Use:   "sync-base",
		Short: "Check, rebase, or merge the current branch target base safely",
		RunE: func(command *cobra.Command, _ []string) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			name, err := currentOrSpecified(command.Context(), services, nameRaw, repository)
			if err != nil {
				return err
			}
			base, err := parseBase(baseRaw, repository.Remote)
			if err != nil {
				return err
			}
			strategy := branchapp.SyncStrategy(strategyRaw)
			var parsedMergeMessage *commitmsg.Message
			if mergeMessage != "" {
				message, err := commitmsg.Parse(mergeMessage)
				if err != nil {
					return err
				}
				parsedMergeMessage = &message
			}
			if strategy == branchapp.SyncRebase || strategy == branchapp.SyncMerge {
				if err := application.confirmMutation(command.Context(), "Synchronize branch base", "Apply "+strategyRaw+" to "+name.String()+" if policy permits?"); err != nil {
					return err
				}
			}
			result, err := services.sync.Sync(command.Context(), branchapp.SyncRequest{
				Repository:   repository,
				Name:         name,
				Base:         base,
				Strategy:     strategy,
				MergeMessage: parsedMergeMessage,
				DryRun:       application.options.dryRun,
			})
			if err != nil {
				return err
			}
			fields := map[string]string{
				"branch":             result.Name.String(),
				"base":               result.Base.String(),
				"publication":        string(result.Publication),
				"missingBaseCommits": boolString(result.MissingBaseCommits),
				"mutated":            boolString(result.Mutated),
				"recommendedAction":  result.RecommendedAction,
			}
			if result.Quality != nil {
				fields["qualityStatus"] = string(result.Quality.Status)
				fields["qualityDetail"] = result.Quality.Detail
			}
			return application.report(command, port.Report{
				Operation: "branch.sync-base",
				Summary:   "Branch base synchronization checked.",
				Fields:    fields,
			})
		},
	}
	command.Flags().StringVar(&nameRaw, "branch", "", "branch name; defaults to the current branch")
	command.Flags().StringVar(&baseRaw, "base", "", "explicit remote target base")
	command.Flags().StringVar(&strategyRaw, "strategy", string(branchapp.SyncCheck), "check, auto, rebase, or merge")
	command.Flags().StringVar(&mergeMessage, "merge-message", "", "full governed merge commit message for --strategy merge")
	return command
}

func branchCreationSummary(result branchapp.CreateResult) string {
	if result.DryRun {
		return "Branch creation plan generated."
	}
	return "Branch created."
}

func planText(steps []branchapp.PlanStep) string {
	parts := make([]string, 0, len(steps))
	for _, step := range steps {
		parts = append(parts, step.String())
	}
	return strings.Join(parts, "; ")
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
