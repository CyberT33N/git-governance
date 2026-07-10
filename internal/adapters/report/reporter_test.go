package report

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

func TestHumanReportSuccessAndQuiet(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	reporter := New(Options{Writer: buffer, Format: FormatHuman})
	err := reporter.Report(context.Background(), port.Report{
		Operation: "branch.create",
		Summary:   "Branch created.",
		Fields: map[string]string{
			"branch": "feature/ABC-123-add-export",
			"base":   "origin/develop",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := "Branch created.\nbase: origin/develop\nbranch: feature/ABC-123-add-export\n"
	if buffer.String() != expected {
		t.Fatalf("human output = %q, want %q", buffer.String(), expected)
	}

	quiet := &bytes.Buffer{}
	if err := New(Options{Writer: quiet, Quiet: true}).Report(context.Background(), port.Report{Summary: "hidden"}); err != nil {
		t.Fatal(err)
	}
	if quiet.Len() != 0 {
		t.Fatalf("quiet output = %q", quiet.String())
	}
}

func TestHumanReportAppliesExplicitColor(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	reporter := New(Options{Writer: buffer, Format: FormatHuman, Color: true})
	if err := reporter.Report(context.Background(), port.Report{
		Summary: "Branch created.",
		Fields:  map[string]string{"branch": "feature/ABC-123-add-export"},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "\x1b[32mBranch created.\x1b[0m") {
		t.Fatalf("colored summary = %q", buffer.String())
	}
	if !strings.Contains(buffer.String(), "\x1b[36mbranch\x1b[0m") {
		t.Fatalf("colored field = %q", buffer.String())
	}
}

func TestHumanProblemReport(t *testing.T) {
	t.Parallel()

	buffer := &bytes.Buffer{}
	value := problem.New(problem.Details{
		Code:        problem.CodeTicketKeyInvalid,
		Category:    problem.CategoryGovernance,
		Field:       "ticket key",
		Actual:      "abc",
		Expected:    "uppercase letters",
		Rule:        "ticket keys must be uppercase",
		Example:     "ABC",
		Remediation: "use uppercase",
	})
	if err := New(Options{Writer: buffer}).Report(context.Background(), port.Report{Problem: value}); err != nil {
		t.Fatal(err)
	}
	output := buffer.String()
	for _, expected := range []string{
		"Error [TICKET_KEY_INVALID]",
		"Field: ticket key",
		"Actual value:",
		"abc",
		"What is wrong?",
		"Expected:",
		"Valid example:",
		"How to fix it:",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("human problem output missing %q: %q", expected, output)
		}
	}
}

func TestJSONReportContracts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		reporter := New(Options{Writer: buffer, Format: FormatJSON})
		if err := reporter.Report(context.Background(), port.Report{
			Operation: "branch.validate",
			Summary:   "valid",
			Fields:    map[string]string{"branch": "feature/ABC-123-add-export"},
		}); err != nil {
			t.Fatal(err)
		}
		expected := "{\"schemaVersion\":1,\"ok\":true,\"operation\":\"branch.validate\",\"summary\":\"valid\",\"fields\":{\"branch\":\"feature/ABC-123-add-export\"}}\n"
		if buffer.String() != expected {
			t.Fatalf("JSON output = %q, want %q", buffer.String(), expected)
		}
	})

	t.Run("sensitive problem omits actual", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		value := problem.New(problem.Details{
			Code:            problem.CodeConfigurationInvalid,
			Category:        problem.CategoryConfig,
			Actual:          "secret",
			SensitiveActual: true,
		})
		if err := New(Options{Writer: buffer, Format: FormatJSON}).Report(context.Background(), port.Report{Problem: value}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buffer.String(), "secret") || strings.Contains(buffer.String(), `"actual"`) {
			t.Fatalf("sensitive JSON output = %q", buffer.String())
		}
	})

	t.Run("sensitive human problem omits actual", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		value := problem.New(problem.Details{
			Code:            problem.CodeConfigurationInvalid,
			Category:        problem.CategoryConfig,
			Actual:          "secret",
			SensitiveActual: true,
		})
		if err := New(Options{Writer: buffer, Format: FormatHuman}).Report(context.Background(), port.Report{Problem: value}); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(buffer.String(), "secret") || strings.Contains(buffer.String(), "Actual value:") {
			t.Fatalf("sensitive human output = %q", buffer.String())
		}
	})
}

func TestReportFailureAndCancellation(t *testing.T) {
	t.Parallel()

	t.Run("writer failure", func(t *testing.T) {
		err := New(Options{Writer: failingWriter{}}).Report(context.Background(), port.Report{Summary: "output"})
		assertProblemCode(t, err, problem.CodeExternalCommandFailed)
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := New(Options{Writer: &bytes.Buffer{}}).Report(ctx, port.Report{})
		assertProblemCode(t, err, problem.CodeOperationCancelled)
	})
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
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
