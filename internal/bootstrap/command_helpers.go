package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"

	branchapp "github.com/CyberT33N/git-governance/internal/application/branch"
	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/commitmsg"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
	"github.com/CyberT33N/git-governance/internal/domain/ticket"
	"github.com/spf13/cobra"
)

const maxCommitMessageBytes = 1 << 20

func (application *application) validateOptions() error {
	switch application.options.interactive {
	case "auto", "always", "never":
	default:
		return invalidOption("interactive", application.options.interactive, "auto, always, or never")
	}
	if _, err := application.outputFormat(); err != nil {
		return err
	}
	switch application.options.color {
	case "auto", "always", "never":
	default:
		return invalidOption("color", application.options.color, "auto, always, or never")
	}
	if application.options.timeout <= 0 {
		return invalidOption("timeout", application.options.timeout.String(), "a positive duration")
	}
	if application.options.interactive == "always" {
		if application.options.output == "json" {
			return invalidOption("interactive", application.options.interactive, "auto or never when --output json is selected")
		}
		if !application.inputIsTerminal() || !application.outputIsTerminal() {
			return problem.New(problem.Details{
				Code:        problem.CodeInvalidInput,
				Category:    problem.CategoryUsage,
				Field:       "interactive",
				Actual:      "always",
				Expected:    "an interactive terminal connected to both standard input and standard output",
				Rule:        "--interactive always must fail early when no terminal is available",
				Remediation: "run from a terminal, use --interactive auto, or supply all required flags with --interactive never",
			})
		}
	}
	return nil
}

func (application *application) report(command *cobra.Command, result port.Report) error {
	return application.reporter(command.OutOrStdout()).Report(command.Context(), result)
}

func (application *application) discover(ctx context.Context, service services) (port.RepositoryIdentity, error) {
	identity, err := service.git.Discover(ctx, application.options.repository)
	if err != nil {
		return port.RepositoryIdentity{}, err
	}
	identity.Remote = application.options.remote
	return identity, nil
}

func (application *application) confirmMutation(ctx context.Context, label, description string) error {
	if application.options.dryRun || application.options.yes {
		return nil
	}
	if !application.promptAvailable() {
		return problem.New(problem.Details{
			Code:        problem.CodeInvalidInput,
			Category:    problem.CategoryUsage,
			Field:       "confirmation",
			Expected:    "--yes for non-interactive mutations",
			Rule:        "mutating Git operations require explicit confirmation",
			Example:     "--yes",
			Remediation: "pass --yes or run in an interactive terminal",
		})
	}
	confirmed, err := application.prompt().Confirm(ctx, port.ConfirmRequest{
		Label:       label,
		Description: description,
		Default:     false,
	})
	if err != nil {
		return err
	}
	if confirmed {
		return nil
	}
	return problem.New(problem.Details{
		Code:        problem.CodeOperationCancelled,
		Category:    problem.CategoryCancelled,
		Field:       "confirmation",
		Expected:    "an explicit yes",
		Rule:        "the user declined the mutation",
		Remediation: "rerun the command and confirm the planned operation",
	})
}

func (application *application) resolveKey(ctx context.Context, service services, raw string) (ticket.Key, error) {
	if raw != "" {
		return ticket.ParseKey(raw)
	}
	defaultValue := ""
	preferences, err := service.preferences.List(ctx)
	if err == nil && preferences.DefaultKey != nil {
		defaultValue = preferences.DefaultKey.String()
	}
	if !application.promptAvailable() {
		return ticket.Key{}, missingInput("ticket key")
	}
	value, err := application.prompt().Input(ctx, port.InputRequest{
		Label:       "Ticket key",
		Description: "A ticket key uses uppercase letters and digits and begins with a letter.",
		Default:     defaultValue,
		Required:    true,
	})
	if err != nil {
		return ticket.Key{}, err
	}
	return ticket.ParseKey(value)
}

func (application *application) resolveNumber(ctx context.Context, raw string) (ticket.Number, error) {
	if raw != "" {
		return ticket.ParseNumber(raw)
	}
	value, err := application.requireInput(ctx, "", "Ticket number", "Enter the positive numeric ticket identifier.")
	if err != nil {
		return ticket.Number{}, err
	}
	return ticket.ParseNumber(value)
}

func (application *application) resolveSlug(ctx context.Context, raw string, label string) (branch.Slug, error) {
	if raw != "" {
		return branch.ParseSlug(raw)
	}
	value, err := application.requireInput(ctx, "", label, "Use lowercase words separated by single hyphens.")
	if err != nil {
		return branch.Slug{}, err
	}
	return branch.ParseSlug(value)
}

func (application *application) resolveFamily(ctx context.Context, raw string, includeSpecial bool) (branch.Family, error) {
	if raw != "" {
		return branch.ParseFamily(raw)
	}
	if !application.promptAvailable() {
		return "", missingInput("branch family")
	}

	families := branchapp.ListFamilies()
	options := make([]port.SelectOption, 0, len(families))
	for _, family := range families {
		if !includeSpecial && !family.DirectlyCreatable {
			continue
		}
		options = append(options, port.SelectOption{
			Value:       family.Family.String(),
			Label:       family.Label,
			Description: family.Description,
		})
	}
	value, err := application.prompt().Select(ctx, port.SelectRequest{
		Label:       "Branch family",
		Description: "Select the branch family that matches the work. The list command explains every family.",
		Options:     options,
		Default:     branch.FamilyFeature.String(),
	})
	if err != nil {
		return "", err
	}
	return branch.ParseFamily(value)
}

func parseTicketParts(keyRaw, numberRaw string) (ticket.ID, error) {
	key, err := ticket.ParseKey(keyRaw)
	if err != nil {
		return ticket.ID{}, err
	}
	number, err := ticket.ParseNumber(numberRaw)
	if err != nil {
		return ticket.ID{}, err
	}
	return ticket.NewID(key, number), nil
}

func parseBase(raw, remote string) (*branch.TargetBase, error) {
	if raw == "" {
		return nil, nil
	}
	nameRaw := raw
	prefix := remote + "/"
	if strings.HasPrefix(raw, prefix) {
		nameRaw = strings.TrimPrefix(raw, prefix)
	} else if strings.Contains(raw, "/") {
		return nil, problem.New(problem.Details{
			Code:        problem.CodeBranchBaseInvalid,
			Category:    problem.CategoryUsage,
			Field:       "base",
			Actual:      raw,
			Expected:    "a canonical branch name or " + prefix + "<branch>",
			Rule:        "explicit bases use the selected remote only",
			Example:     prefix + "develop",
			Remediation: "pass a branch on the selected --remote",
		})
	}
	name, err := branch.ParseName(nameRaw)
	if err != nil {
		return nil, err
	}
	base, err := branch.NewTargetBase(remote, name)
	if err != nil {
		return nil, err
	}
	return &base, nil
}

func (application *application) resolveScratchBase(ctx context.Context, raw, remote string) (*branch.TargetBase, error) {
	if raw == "" {
		value, err := application.requireInput(
			ctx,
			"",
			"Official ticket branch base",
			"Scratch branches are private exploration only. Select the local official ticket branch that owns the same ticket; never use scratch as a pull-request source.",
		)
		if err != nil {
			return nil, err
		}
		raw = value
	}
	if strings.HasPrefix(raw, remote+"/") {
		return nil, problem.New(problem.Details{
			Code:        problem.CodeBranchBaseInvalid,
			Category:    problem.CategoryGovernance,
			Field:       "scratch base",
			Actual:      raw,
			Expected:    "a local official ticket branch name",
			Rule:        "scratch branches start from the local official branch, not a remote-tracking ref",
			Example:     "feature/ABC-123-add-export",
			Remediation: "provide the checked-out official ticket branch without the remote prefix",
		})
	}
	base, err := branch.ParseLocalBase(raw)
	if err != nil {
		return nil, err
	}
	name := base.Branch()
	if !name.Family().IsOfficialWorkingBranch() {
		return nil, problem.New(problem.Details{
			Code:        problem.CodeBranchBaseInvalid,
			Category:    problem.CategoryGovernance,
			Field:       "scratch base",
			Actual:      raw,
			Expected:    "a local feature, fix, docs, refactor, chore, test, perf, or hotfix branch",
			Rule:        "scratch branches are private exploration of an official ticket branch",
			Example:     "feature/ABC-123-add-export",
			Remediation: "select the official branch for the same ticket or use workflow ticket start",
		})
	}
	return &base, nil
}

func currentOrSpecified(ctx context.Context, service services, raw string, repository port.RepositoryIdentity) (branch.BranchName, error) {
	if raw != "" {
		return branch.ParseName(raw)
	}
	return service.git.CurrentBranch(ctx, repository)
}

func readCommitMessage(path, inline string) (string, error) {
	if inline != "" {
		return inline, nil
	}
	if path == "" {
		return "", missingInput("commit message")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", problem.Wrap(problem.Details{
			Code:        problem.CodeConfigurationUnavailable,
			Category:    problem.CategoryUsage,
			Field:       "message file",
			Actual:      path,
			Expected:    "a readable commit message file",
			Rule:        "commit validation reads the exact message supplied by Git",
			Remediation: "provide --message or an existing --message-file",
		}, err)
	}
	if info.Size() > maxCommitMessageBytes {
		return "", problem.New(problem.Details{
			Code:        problem.CodeCommitHeaderInvalid,
			Category:    problem.CategoryUsage,
			Field:       "message file",
			Actual:      path,
			Expected:    fmt.Sprintf("at most %d bytes", maxCommitMessageBytes),
			Rule:        "commit message input is size-bounded",
			Remediation: "reduce the commit message file size",
		})
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", problem.Wrap(problem.Details{
			Code:        problem.CodeConfigurationUnavailable,
			Category:    problem.CategoryUsage,
			Field:       "message file",
			Actual:      path,
			Expected:    "a readable commit message file",
			Rule:        "commit validation reads the exact message supplied by Git",
			Remediation: "check the path and file permissions",
		}, err)
	}
	return string(contents), nil
}

func parseFooterSpec(raw string) (commitmsg.Footer, error) {
	token, value, found := strings.Cut(raw, "=")
	if !found {
		return commitmsg.Footer{}, problem.New(problem.Details{
			Code:        problem.CodeCommitDescriptionInvalid,
			Category:    problem.CategoryUsage,
			Field:       "footer",
			Actual:      raw,
			Expected:    "TOKEN=VALUE",
			Rule:        "CLI footer flags use an equals separator",
			Example:     "--footer Refs=#123",
			Remediation: "provide one footer token and one non-empty value",
		})
	}
	return commitmsg.NewFooter(token, value)
}

func invalidOption(field, actual, expected string) error {
	return problem.New(problem.Details{
		Code:        problem.CodeInvalidInput,
		Category:    problem.CategoryUsage,
		Field:       field,
		Actual:      actual,
		Expected:    expected,
		Rule:        "CLI option values must use a supported value",
		Remediation: "use --help to see supported values",
	})
}
