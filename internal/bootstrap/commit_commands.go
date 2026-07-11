package bootstrap

import (
	"strings"

	commitapp "github.com/CyberT33N/git-governance/internal/application/commit"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
	"github.com/spf13/cobra"
)

func newCommitCommand(application *application) *cobra.Command {
	command := &cobra.Command{
		Use:   "commit",
		Short: "Create and validate governed commits",
	}
	command.AddCommand(
		newCommitCreateCommand(application),
		newCommitValidateCommand(application),
	)
	return command
}

func newCommitCreateCommand(application *application) *cobra.Command {
	var (
		typeRaw             string
		ticketRaw           string
		subject             string
		body                string
		breaking            bool
		breakingDescription string
		footerSpecs         []string
		stagePaths          []string
		push                bool
		baseRaw             string
	)
	command := &cobra.Command{
		Use:   "create",
		Short: "Create a governed commit from explicit staged paths",
		RunE: withWorkflowInputs(func(command *cobra.Command, inputs *workflowInputSummary) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			currentBranch, err := services.git.CurrentBranch(command.Context(), repository)
			if err != nil {
				return err
			}
			commitTicket, err := resolveCommitTicket(application, command, currentBranch, ticketRaw)
			if err != nil {
				return err
			}
			inputs.add("ticket", commitTicket.String())
			kind, err := resolveCommitType(application, command, currentBranch, typeRaw)
			if err != nil {
				return err
			}
			inputs.add("commit type", kind.String())
			if subject == "" {
				subject, err = application.requireInput(
					command.Context(),
					"",
					"Commit subject",
					"Enter one non-empty, unpadded line of at most 200 characters. Do not use control characters. Example: add export button.",
					func(value string) error {
						_, validationErr := commitmsg.NewHeader(kind, commitTicket, value, breaking)
						return validationErr
					},
				)
				if err != nil {
					return err
				}
			}
			inputs.add("commit subject", subject)
			header, err := commitmsg.NewHeader(kind, commitTicket, subject, breaking)
			if err != nil {
				return err
			}
			footers, err := parseFooterSpecs(footerSpecs)
			if err != nil {
				return err
			}
			if breaking {
				if breakingDescription == "" {
					breakingDescription, err = application.requireInput(
						command.Context(),
						"",
						"Breaking change impact",
						"Describe the incompatible public contract change and the concrete migration impact without leading or trailing whitespace. Example: clients must use the versioned export endpoint.",
						func(value string) error {
							_, validationErr := commitmsg.NewFooter("BREAKING CHANGE", value)
							return validationErr
						},
					)
					if err != nil {
						return err
					}
				}
				inputs.add("breaking change impact", breakingDescription)
				breakingFooter, err := commitmsg.NewFooter("BREAKING CHANGE", breakingDescription)
				if err != nil {
					return err
				}
				footers = append(footers, breakingFooter)
			}
			message, err := commitmsg.NewMessage(header, body, footers)
			if err != nil {
				return err
			}
			base, err := parseBase(baseRaw, repository.Remote)
			if err != nil {
				return err
			}
			if err := application.confirmMutation(command.Context(), "Create commit", "Create "+message.Header().String()+"?"); err != nil {
				return err
			}
			result, err := services.commits.Create(command.Context(), commitapp.CreateRequest{
				Repository: repository,
				Branch:     currentBranch,
				Message:    message,
				StagePaths: stagePaths,
				Base:       base,
				Push:       push,
				DryRun:     application.options.dryRun,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "commit.create",
				Summary:   commitCreationSummary(result),
				Fields: map[string]string{
					"branch":    result.Branch.String(),
					"message":   result.Message.Header().String(),
					"committed": boolString(result.Committed),
					"pushed":    boolString(result.Pushed),
					"dryRun":    boolString(result.DryRun),
					"plan":      commitPlanText(result.Plan),
				},
			})
		}),
	}
	command.Flags().StringVar(&typeRaw, "type", "", "commit type")
	command.Flags().StringVar(&ticketRaw, "ticket", "", "ticket ID; defaults to the current ticket branch")
	command.Flags().StringVar(&subject, "subject", "", "short commit subject")
	command.Flags().StringVar(&body, "body", "", "optional commit body")
	command.Flags().BoolVar(&breaking, "breaking", false, "mark an incompatible public contract change")
	command.Flags().StringVar(&breakingDescription, "breaking-description", "", "breaking change migration impact")
	command.Flags().StringSliceVar(&footerSpecs, "footer", nil, "footer as TOKEN=VALUE; repeatable")
	command.Flags().StringSliceVar(&stagePaths, "stage", nil, "explicit path to stage; repeatable")
	command.Flags().BoolVar(&push, "push", false, "validate and push after committing")
	command.Flags().StringVar(&baseRaw, "base", "", "explicit base for pre-push validation")
	return command
}

func newCommitValidateCommand(application *application) *cobra.Command {
	var (
		messageFile string
		messageRaw  string
		branchRaw   string
	)
	command := &cobra.Command{
		Use:   "validate",
		Short: "Validate a complete commit message for the current branch",
		RunE: func(command *cobra.Command, _ []string) error {
			services := application.services()
			repository, err := application.discover(command.Context(), services)
			if err != nil {
				return err
			}
			currentBranch, err := currentOrSpecified(command.Context(), services, branchRaw, repository)
			if err != nil {
				return err
			}
			raw, err := readCommitMessage(messageFile, messageRaw)
			if err != nil {
				return err
			}
			message, err := commitmsg.Parse(raw)
			if err != nil {
				return err
			}
			result, err := services.commits.Validate(command.Context(), commitapp.ValidateRequest{
				Repository: repository,
				Branch:     currentBranch,
				Message:    message,
			})
			if err != nil {
				return err
			}
			return application.report(command, port.Report{
				Operation: "commit.validate",
				Summary:   "Commit message is valid.",
				Fields: map[string]string{
					"branch":   result.Branch.String(),
					"message":  result.Message.Header().String(),
					"breaking": boolString(result.Message.IsBreaking()),
				},
			})
		},
	}
	command.Flags().StringVar(&messageFile, "message-file", "", "file containing the complete commit message")
	command.Flags().StringVar(&messageRaw, "message", "", "complete commit message")
	command.Flags().StringVar(&branchRaw, "branch", "", "branch name; defaults to the current branch")
	return command
}

func resolveCommitTicket(application *application, command *cobra.Command, current branch.BranchName, raw string) (ticket.ID, error) {
	if raw != "" {
		return ticket.ParseID(raw)
	}
	if fromBranch, ok := current.Ticket(); ok {
		return fromBranch, nil
	}
	if !application.promptAvailable() {
		return ticket.ID{}, missingInput("ticket")
	}
	key, err := application.resolveKey(command.Context(), application.services(), "")
	if err != nil {
		return ticket.ID{}, err
	}
	number, err := application.resolveNumber(command.Context(), "")
	if err != nil {
		return ticket.ID{}, err
	}
	return ticket.NewID(key, number), nil
}

func resolveCommitType(application *application, command *cobra.Command, current branch.BranchName, raw string) (commitmsg.Type, error) {
	if raw != "" {
		return commitmsg.ParseType(raw)
	}
	defaultType := defaultCommitType(current.Family())
	if !application.promptAvailable() {
		return defaultType, nil
	}
	types := commitmsg.Types()
	options := make([]port.SelectOption, 0, len(types))
	for _, kind := range types {
		options = append(options, port.SelectOption{
			Value:       kind.String(),
			Label:       kind.String(),
			Description: commitTypeDescription(kind),
		})
	}
	value, err := application.prompt().Select(command.Context(), port.SelectRequest{
		Label:       "Commit type",
		Description: "Select the semantic type of this change. Branch and commit taxonomies are intentionally separate.",
		Options:     options,
		Default:     defaultType.String(),
	})
	if err != nil {
		return "", err
	}
	return commitmsg.ParseType(value)
}

func defaultCommitType(family branch.Family) commitmsg.Type {
	switch family {
	case branch.FamilyFeature:
		return commitmsg.TypeFeat
	case branch.FamilyFix, branch.FamilyHotfix:
		return commitmsg.TypeFix
	case branch.FamilyDocs:
		return commitmsg.TypeDocs
	case branch.FamilyRefactor:
		return commitmsg.TypeRefactor
	case branch.FamilyChore:
		return commitmsg.TypeChore
	case branch.FamilyTest:
		return commitmsg.TypeTest
	case branch.FamilyPerf:
		return commitmsg.TypePerf
	default:
		return commitmsg.TypeChore
	}
}

func commitTypeDescription(kind commitmsg.Type) string {
	switch kind {
	case commitmsg.TypeFeat:
		return "New product functionality."
	case commitmsg.TypeFix:
		return "A defect correction."
	case commitmsg.TypeDocs:
		return "Documentation-only change."
	case commitmsg.TypeRefactor:
		return "Internal restructuring without a feature or fix."
	case commitmsg.TypeTest:
		return "Test work."
	case commitmsg.TypePerf:
		return "Measured performance improvement."
	case commitmsg.TypeBuild:
		return "Build system or dependency change."
	case commitmsg.TypeCI:
		return "Continuous-integration configuration."
	case commitmsg.TypeStyle:
		return "Formatting with no semantic effect."
	case commitmsg.TypeRevert:
		return "A deliberate revert with a commit reference."
	default:
		return "Maintenance or tooling work."
	}
}

func parseFooterSpecs(values []string) ([]commitmsg.Footer, error) {
	footers := make([]commitmsg.Footer, 0, len(values))
	for _, value := range values {
		footer, err := parseFooterSpec(value)
		if err != nil {
			return nil, err
		}
		footers = append(footers, footer)
	}
	return footers, nil
}

func commitCreationSummary(result commitapp.CreateResult) string {
	if result.DryRun {
		return "Commit creation plan generated."
	}
	if result.Pushed {
		return "Commit created and pushed."
	}
	return "Commit created."
}

func commitPlanText(steps []commitapp.PlanStep) string {
	parts := make([]string, 0, len(steps))
	for _, step := range steps {
		parts = append(parts, step.Action+": "+step.Detail)
	}
	return strings.Join(parts, "; ")
}
