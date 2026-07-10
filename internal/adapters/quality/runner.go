// Package quality runs explicitly configured repository-local quality gates.
package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

const (
	defaultConfigName = "git-governance.quality.json"
	currentSchema     = 1
	maxConfigBytes    = 1 << 20
	maxGateCount      = 32
	maxArgumentCount  = 64
	defaultTimeout    = 5 * time.Minute
)

var gateNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type commandRunner func(ctx context.Context, directory, executable string, arguments ...string) error

// Options configures a generic quality runner. It deliberately accepts command
// arrays, never shell command strings.
type Options struct {
	Path           string
	DefaultTimeout time.Duration
	ReadFile       func(string) ([]byte, error)
	Run            commandRunner
}

// Runner executes a trusted repository's explicitly declared local quality
// gates. Repositories with no config are reported as unconfigured, not passed.
type Runner struct {
	path           string
	defaultTimeout time.Duration
	readFile       func(string) ([]byte, error)
	run            commandRunner
}

// New creates a repository-local quality runner.
func New(options Options) *Runner {
	timeout := options.DefaultTimeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	readFile := options.ReadFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	run := options.Run
	if run == nil {
		run = runCommand
	}
	return &Runner{
		path:           options.Path,
		defaultTimeout: timeout,
		readFile:       readFile,
		run:            run,
	}
}

// Run loads and executes every declared gate. The config file is an explicit
// trust boundary: invoking project tooling may execute arbitrary project code,
// so the runner neither guesses commands nor interprets shell syntax.
func (runner *Runner) Run(ctx context.Context, repository port.RepositoryIdentity) (port.QualityResult, error) {
	if repository.Root == "" {
		return port.QualityResult{}, problem.New(problem.Details{
			Code:        problem.CodeRepositoryNotFound,
			Category:    problem.CategoryRepository,
			Field:       "repository",
			Expected:    "a discovered repository root for quality configuration",
			Rule:        "quality gates run only inside an explicit repository",
			Remediation: "run from a Git repository or pass --repo",
		})
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return port.QualityResult{}, cancelled(err)
	}

	path := runner.configPath(repository.Root)
	contents, err := runner.readFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return port.QualityResult{
			Status: port.QualityUnconfigured,
			Detail: "no repository-local quality configuration is present",
		}, nil
	}
	if err != nil {
		return port.QualityResult{}, unavailable(path, "read quality configuration", err)
	}
	if len(contents) > maxConfigBytes {
		return port.QualityResult{}, invalid(path, "quality configuration must not exceed 1 MiB", nil)
	}

	config, err := decode(path, contents)
	if err != nil {
		return port.QualityResult{}, err
	}
	result := port.QualityResult{
		Status: port.QualityPassed,
		Detail: "all configured repository-local quality gates passed",
		Gates:  make([]port.QualityGateResult, 0, len(config.Gates)),
	}
	for _, gate := range config.Gates {
		directory, err := resolveWorkingDirectory(repository.Root, gate.WorkingDirectory)
		if err != nil {
			return port.QualityResult{}, err
		}
		timeout, err := gateTimeout(gate.Timeout, runner.defaultTimeout)
		if err != nil {
			return port.QualityResult{}, invalid(path, "gate "+gate.Name+" has an invalid timeout", err)
		}
		gateContext, cancel := context.WithTimeout(ctx, timeout)
		err = runner.run(gateContext, directory, gate.Command, gate.Args...)
		cancel()
		if err != nil {
			return port.QualityResult{}, problem.Wrap(problem.Details{
				Code:        problem.CodeExternalCommandFailed,
				Category:    problem.CategoryExternal,
				Field:       "quality gate",
				Actual:      gate.Name,
				Expected:    "a successful configured quality command",
				Rule:        "each configured quality gate must pass before publication-affecting work continues",
				Example:     `{"name":"unit-tests","command":"go","args":["test","./..."],"timeout":"2m"}`,
				Remediation: "fix the reported project quality failure, adjust the trusted configuration, or use an explicitly documented skip policy",
			}, err)
		}
		result.Gates = append(result.Gates, port.QualityGateResult{Name: gate.Name})
	}
	return result, nil
}

func (runner *Runner) configPath(root string) string {
	if runner.path == "" {
		return filepath.Join(root, defaultConfigName)
	}
	if filepath.IsAbs(runner.path) {
		return filepath.Clean(runner.path)
	}
	return filepath.Join(root, filepath.Clean(runner.path))
}

type config struct {
	SchemaVersion int    `json:"schemaVersion"`
	Gates         []gate `json:"gates"`
}

type gate struct {
	Name             string   `json:"name"`
	Command          string   `json:"command"`
	Args             []string `json:"args"`
	Timeout          string   `json:"timeout"`
	WorkingDirectory string   `json:"workingDirectory,omitempty"`
}

func decode(path string, contents []byte) (config, error) {
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var value config
	if err := decoder.Decode(&value); err != nil {
		return config{}, invalid(path, "quality configuration must contain valid JSON with known fields", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return config{}, invalid(path, "quality configuration must contain exactly one JSON document", nil)
	}
	if value.SchemaVersion != currentSchema {
		return config{}, invalid(path, "schemaVersion must equal 1", nil)
	}
	if len(value.Gates) == 0 || len(value.Gates) > maxGateCount {
		return config{}, invalid(path, "gates must contain between 1 and 32 entries", nil)
	}
	seen := make(map[string]struct{}, len(value.Gates))
	for _, gate := range value.Gates {
		if !gateNamePattern.MatchString(gate.Name) {
			return config{}, invalid(path, "gate names must use lowercase letters, digits, hyphens, or underscores", nil)
		}
		if _, found := seen[gate.Name]; found {
			return config{}, invalid(path, "gate names must be unique", nil)
		}
		seen[gate.Name] = struct{}{}
		if strings.TrimSpace(gate.Command) == "" || strings.ContainsAny(gate.Command, "\r\n") {
			return config{}, invalid(path, "each gate command must be a non-empty executable name or path", nil)
		}
		if len(gate.Args) > maxArgumentCount {
			return config{}, invalid(path, "each gate may contain at most 64 arguments", nil)
		}
		for _, argument := range gate.Args {
			if strings.ContainsAny(argument, "\x00\r\n") {
				return config{}, invalid(path, "gate arguments cannot contain NUL or line-control characters", nil)
			}
		}
		if _, err := resolveWorkingDirectory(".", gate.WorkingDirectory); err != nil {
			return config{}, err
		}
		if _, err := gateTimeout(gate.Timeout, defaultTimeout); err != nil {
			return config{}, invalid(path, "gate "+gate.Name+" has an invalid timeout", err)
		}
	}
	return value, nil
}

func resolveWorkingDirectory(root, relative string) (string, error) {
	if relative == "" {
		relative = "."
	}
	if filepath.IsAbs(relative) {
		return "", problem.New(problem.Details{
			Code:        problem.CodeConfigurationInvalid,
			Category:    problem.CategoryConfig,
			Field:       "quality gate workingDirectory",
			Actual:      relative,
			Expected:    "a path relative to the repository root",
			Rule:        "quality gate working directories cannot escape the selected repository",
			Remediation: "use . or a relative descendant path",
		})
	}
	clean := filepath.Clean(relative)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", problem.New(problem.Details{
			Code:        problem.CodeConfigurationInvalid,
			Category:    problem.CategoryConfig,
			Field:       "quality gate workingDirectory",
			Actual:      relative,
			Expected:    "a path inside the repository root",
			Rule:        "quality gate working directories cannot escape the selected repository",
			Remediation: "use . or a relative descendant path",
		})
	}
	return filepath.Join(root, clean), nil
}

func gateTimeout(raw string, fallback time.Duration) (time.Duration, error) {
	if raw == "" {
		return fallback, nil
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		return 0, errors.New("timeout must be a positive Go duration")
	}
	return timeout, nil
}

func runCommand(ctx context.Context, directory, executable string, arguments ...string) error {
	command := exec.CommandContext(ctx, executable, arguments...)
	command.Dir = directory
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func cancelled(cause error) error {
	return problem.Wrap(problem.Details{
		Code:        problem.CodeOperationCancelled,
		Category:    problem.CategoryCancelled,
		Field:       "quality gates",
		Expected:    "an active context",
		Rule:        "quality gate execution stops when the caller cancels its context",
		Remediation: "retry with an active context",
	}, cause)
}

func unavailable(path, action string, cause error) error {
	return problem.Wrap(problem.Details{
		Code:        problem.CodeConfigurationUnavailable,
		Category:    problem.CategoryConfig,
		Field:       "quality configuration",
		Actual:      path,
		Expected:    "an accessible repository-local quality configuration",
		Rule:        action,
		Remediation: "check the configuration path and filesystem permissions",
	}, cause)
}

func invalid(path, rule string, cause error) error {
	return problem.Wrap(problem.Details{
		Code:        problem.CodeConfigurationInvalid,
		Category:    problem.CategoryConfig,
		Field:       "quality configuration",
		Actual:      path,
		Expected:    "a valid git-governance.quality.json document",
		Rule:        rule,
		Example:     `{"schemaVersion":1,"gates":[{"name":"unit-tests","command":"go","args":["test","./..."],"timeout":"2m"}]}`,
		Remediation: "correct the repository-local quality configuration",
	}, cause)
}

var _ port.QualityRunner = (*Runner)(nil)
