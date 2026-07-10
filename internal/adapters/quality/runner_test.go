package quality

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

type recordedCommand struct {
	directory  string
	executable string
	arguments  []string
}

func TestRunReportsUnconfiguredRepository(t *testing.T) {
	t.Parallel()

	runner := New(Options{})
	result, err := runner.Run(context.Background(), port.RepositoryIdentity{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != port.QualityUnconfigured || len(result.Gates) != 0 {
		t.Fatalf("Run() = %#v", result)
	}
}

func TestRunExecutesConfiguredArgumentArrays(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	configPath := filepath.Join(root, defaultConfigName)
	writeConfig(t, configPath, `{
  "schemaVersion": 1,
  "gates": [
    {"name":"unit-tests","command":"go","args":["test","./..."],"timeout":"2m"},
    {"name":"lint","command":"tool","args":["check"],"workingDirectory":"tools","timeout":"30s"}
  ]
}`)
	var calls []recordedCommand
	runner := New(Options{
		Run: func(_ context.Context, directory, executable string, arguments ...string) error {
			calls = append(calls, recordedCommand{
				directory:  directory,
				executable: executable,
				arguments:  append([]string(nil), arguments...),
			})
			return nil
		},
	})

	result, err := runner.Run(context.Background(), port.RepositoryIdentity{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != port.QualityPassed || len(result.Gates) != 2 {
		t.Fatalf("Run() = %#v", result)
	}
	if len(calls) != 2 ||
		calls[0].directory != root ||
		calls[0].executable != "go" ||
		strings.Join(calls[0].arguments, ",") != "test,./..." ||
		calls[1].directory != filepath.Join(root, "tools") {
		t.Fatalf("quality calls = %#v", calls)
	}
}

func TestRunRejectsUnsafeOrInvalidConfigurations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		contents string
	}{
		{
			name:     "unknown field",
			contents: `{"schemaVersion":1,"gates":[{"name":"test","command":"go","unknown":true}]}`,
		},
		{
			name:     "empty gates",
			contents: `{"schemaVersion":1,"gates":[]}`,
		},
		{
			name:     "escaping directory",
			contents: `{"schemaVersion":1,"gates":[{"name":"test","command":"go","workingDirectory":"../outside"}]}`,
		},
		{
			name:     "shell control argument",
			contents: "{\"schemaVersion\":1,\"gates\":[{\"name\":\"test\",\"command\":\"go\",\"args\":[\"test\\n./...\"]}]}",
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			writeConfig(t, filepath.Join(root, defaultConfigName), testCase.contents)
			_, err := New(Options{}).Run(context.Background(), port.RepositoryIdentity{Root: root})
			assertProblemCode(t, err, problem.CodeConfigurationInvalid)
		})
	}
}

func TestRunClassifiesGateFailureAndCancellation(t *testing.T) {
	t.Parallel()

	t.Run("gate failure", func(t *testing.T) {
		root := t.TempDir()
		writeConfig(t, filepath.Join(root, defaultConfigName), `{"schemaVersion":1,"gates":[{"name":"test","command":"go"}]}`)
		_, err := New(Options{
			Run: func(context.Context, string, string, ...string) error {
				return errors.New("exit status 1")
			},
		}).Run(context.Background(), port.RepositoryIdentity{Root: root})
		assertProblemCode(t, err, problem.CodeExternalCommandFailed)
	})

	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := New(Options{}).Run(ctx, port.RepositoryIdentity{Root: t.TempDir()})
		assertProblemCode(t, err, problem.CodeOperationCancelled)
	})
}

func TestInternalTimeoutAndPathHelpers(t *testing.T) {
	t.Parallel()

	if timeout, err := gateTimeout("", time.Second); err != nil || timeout != time.Second {
		t.Fatalf("gateTimeout default = (%s, %v)", timeout, err)
	}
	if _, err := gateTimeout("zero", time.Second); err == nil {
		t.Fatal("gateTimeout accepted invalid value")
	}
	outside, err := filepath.Abs("outside")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveWorkingDirectory(".", outside); err == nil {
		t.Fatal("resolveWorkingDirectory accepted absolute path")
	}
}

func FuzzDecodeQualityConfiguration(f *testing.F) {
	for _, seed := range []string{
		`{"schemaVersion":1,"gates":[{"name":"unit-tests","command":"go","args":["test","./..."],"timeout":"2m"}]}`,
		`{"schemaVersion":1,"gates":[]}`,
		`{"schemaVersion":1,"gates":[{"name":"test","command":"go","workingDirectory":"../outside"}]}`,
		`{`,
		"",
		"\x00",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = decode("fuzz-quality.json", []byte(raw))
	})
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertProblemCode(t *testing.T, err error, expected problem.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected problem code %q, got nil", expected)
	}
	actual, ok := problem.As(err)
	if !ok {
		t.Fatalf("error %T does not carry a problem: %v", err, err)
	}
	if actual.Code != expected {
		t.Fatalf("problem code = %q, want %q", actual.Code, expected)
	}
}
