package policy

import (
	"context"
	"path/filepath"

	"github.com/CyberT33N/git-governance/internal/application/port"
)

// DoctorService performs read-only local diagnostics.
type DoctorService struct {
	git    port.GitRepository
	store  port.PreferencesStore
	policy port.PolicyInspector
	tools  port.ToolInspector
}

// NewDoctorService creates a read-only diagnostics service.
func NewDoctorService(git port.GitRepository, store port.PreferencesStore) *DoctorService {
	return NewDoctorServiceWithDependencies(git, store, nil, nil)
}

// NewDoctorServiceWithDependencies creates a diagnostics service with optional
// policy and host-tool inspectors.
func NewDoctorServiceWithDependencies(
	git port.GitRepository,
	store port.PreferencesStore,
	policy port.PolicyInspector,
	tools port.ToolInspector,
) *DoctorService {
	return &DoctorService{
		git:    git,
		store:  store,
		policy: policy,
		tools:  tools,
	}
}

// Check is one diagnostic outcome.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// DoctorResult is a non-mutating environment snapshot.
type DoctorResult struct {
	Repository port.RepositoryIdentity `json:"repository"`
	Checks     []Check                 `json:"checks"`
}

// Run checks the repository and configuration without installing, repairing,
// fetching, or otherwise mutating anything.
func (service *DoctorService) Run(ctx context.Context, directory string) (DoctorResult, error) {
	result := DoctorResult{Checks: make([]Check, 0, 9)}
	if service.git == nil {
		result.Checks = append(result.Checks, Check{
			Name:   "git repository",
			OK:     false,
			Detail: "Git adapter is not configured",
		})
	} else {
		version, err := service.git.Version(ctx)
		if err != nil {
			result.Checks = append(result.Checks, Check{
				Name:   "Git version",
				OK:     false,
				Detail: err.Error(),
			})
		} else {
			result.Checks = append(result.Checks, Check{
				Name:   "Git version",
				OK:     true,
				Detail: version,
			})
		}

		repository, err := service.git.Discover(ctx, directory)
		if err != nil {
			result.Checks = append(result.Checks, Check{
				Name:   "git repository",
				OK:     false,
				Detail: err.Error(),
			})
		} else {
			result.Repository = repository
			result.appendRepositoryChecks(ctx, service.git, repository)
		}
	}

	result.appendToolChecks(ctx, service.tools)
	result.appendPolicyCheck(ctx, service.policy)
	result.appendConfigurationCheck(ctx, service.store)
	return result, nil
}

func (result *DoctorResult) appendRepositoryChecks(ctx context.Context, git port.GitRepository, repository port.RepositoryIdentity) {
	result.Checks = append(result.Checks, Check{
		Name:   "git repository",
		OK:     true,
		Detail: repository.Root,
	})

	hasCommits, commitErr := git.HasCommits(ctx, repository)
	if commitErr != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "repository history",
			OK:     false,
			Detail: commitErr.Error(),
		})
	} else {
		detail := "repository has at least one commit"
		if !hasCommits {
			detail = "repository has no commits; branch creation is unavailable"
		}
		result.Checks = append(result.Checks, Check{
			Name:   "repository history",
			OK:     hasCommits,
			Detail: detail,
		})
	}

	if _, err := git.RemoteURL(ctx, repository); err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "selected remote",
			OK:     false,
			Detail: err.Error(),
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "selected remote",
			OK:     true,
			Detail: repository.Remote + " is configured",
		})
	}

	operation, active, err := git.ActiveOperation(ctx, repository)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "Git operation state",
			OK:     false,
			Detail: err.Error(),
		})
	} else if active {
		result.Checks = append(result.Checks, Check{
			Name:   "Git operation state",
			OK:     false,
			Detail: operation + " is in progress; complete or abort it before governed mutations",
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "Git operation state",
			OK:     true,
			Detail: "no merge, rebase, or cherry-pick is in progress",
		})
	}

}

func (result *DoctorResult) appendToolChecks(ctx context.Context, tools port.ToolInspector) {
	if tools == nil {
		result.Checks = append(result.Checks, Check{
			Name:   "runtime platform",
			OK:     false,
			Detail: "tool inspector is not configured",
		})
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook executable",
			OK:     false,
			Detail: "tool inspector is not configured",
		})
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook configuration",
			OK:     false,
			Detail: "tool inspector is not configured",
		})
		return
	}
	operatingSystem, architecture := tools.Platform()
	result.Checks = append(result.Checks, Check{
		Name:   "runtime platform",
		OK:     operatingSystem != "" && architecture != "",
		Detail: operatingSystem + "/" + architecture,
	})
	version, err := tools.Version(ctx, "lefthook")
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook executable",
			OK:     false,
			Detail: err.Error(),
		})
	} else {
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook executable",
			OK:     true,
			Detail: version,
		})
	}
	if result.Repository.Root == "" {
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook configuration",
			OK:     false,
			Detail: "repository root is unavailable",
		})
		return
	}
	exists, err := tools.FileExists(filepath.Join(result.Repository.Root, "lefthook.yml"))
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "Lefthook configuration",
			OK:     false,
			Detail: err.Error(),
		})
		return
	}
	result.Checks = append(result.Checks, Check{
		Name:   "Lefthook configuration",
		OK:     exists,
		Detail: lefthookConfigurationDetail(exists),
	})
}

func (result *DoctorResult) appendPolicyCheck(ctx context.Context, policy port.PolicyInspector) {
	if policy == nil {
		result.Checks = append(result.Checks, Check{
			Name:   "local policy",
			OK:     false,
			Detail: "policy inspector is not configured",
		})
		return
	}
	status, err := policy.Status(ctx, result.Repository)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "local policy",
			OK:     false,
			Detail: err.Error(),
		})
		return
	}
	result.Checks = append(result.Checks, Check{
		Name:   "local policy",
		OK:     true,
		Detail: status.Detail,
	})
}

func (result *DoctorResult) appendConfigurationCheck(ctx context.Context, store port.PreferencesStore) {
	if store == nil {
		result.Checks = append(result.Checks, Check{
			Name:   "user configuration",
			OK:     false,
			Detail: "preferences store is not configured",
		})
		return
	}
	preferences, err := store.Load(ctx)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:   "user configuration",
			OK:     false,
			Detail: err.Error(),
		})
		return
	}
	result.Checks = append(result.Checks, Check{
		Name:   "user configuration",
		OK:     true,
		Detail: configurationDetail(preferences),
	})
}

func configurationDetail(preferences port.Preferences) string {
	if preferences.DefaultKey == nil {
		return "configuration is readable; no default ticket key is set"
	}
	return "configuration is readable; default ticket key is " + preferences.DefaultKey.String()
}

func lefthookConfigurationDetail(exists bool) string {
	if exists {
		return "lefthook.yml is present"
	}
	return "lefthook.yml is not present"
}
