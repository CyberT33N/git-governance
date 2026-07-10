package terminal

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

func TestLineInputUsesDefaultAndRequiredValidation(t *testing.T) {
	t.Parallel()

	t.Run("default", func(t *testing.T) {
		output := &bytes.Buffer{}
		prompt := New(Options{
			Input:  strings.NewReader("\n"),
			Output: output,
		})
		actual, err := prompt.Input(context.Background(), port.InputRequest{
			Label:       "Ticket key",
			Description: "Enter the ticket namespace.",
			Default:     "ABC",
			Required:    true,
		})
		if err != nil || actual != "ABC" {
			t.Fatalf("Input() = (%q, %v)", actual, err)
		}
		if !strings.Contains(output.String(), "Ticket key [ABC]:") {
			t.Fatalf("output = %q", output.String())
		}
	})

	t.Run("required retry", func(t *testing.T) {
		output := &bytes.Buffer{}
		prompt := New(Options{
			Input:  strings.NewReader("\nABC\n"),
			Output: output,
		})
		actual, err := prompt.Input(context.Background(), port.InputRequest{
			Label:    "Ticket key",
			Required: true,
		})
		if err != nil || actual != "ABC" {
			t.Fatalf("Input() = (%q, %v)", actual, err)
		}
		if !strings.Contains(output.String(), "A value is required.") {
			t.Fatalf("output = %q", output.String())
		}
	})
}

func TestLineSelectAndConfirm(t *testing.T) {
	t.Parallel()

	options := []port.SelectOption{
		{Value: "feature", Label: "Feature", Description: "New product capability."},
		{Value: "fix", Label: "Fix", Description: "Correct a defect."},
	}

	t.Run("selection retries", func(t *testing.T) {
		output := &bytes.Buffer{}
		prompt := New(Options{
			Input:  strings.NewReader("9\n2\n"),
			Output: output,
		})
		actual, err := prompt.Select(context.Background(), port.SelectRequest{
			Label:       "Branch family",
			Description: "Choose a branch family.",
			Options:     options,
			Default:     "feature",
		})
		if err != nil || actual != "fix" {
			t.Fatalf("Select() = (%q, %v)", actual, err)
		}
		if !strings.Contains(output.String(), "Choose a listed option number.") {
			t.Fatalf("output = %q", output.String())
		}
	})

	t.Run("selection default", func(t *testing.T) {
		prompt := New(Options{
			Input:  strings.NewReader("\n"),
			Output: &bytes.Buffer{},
		})
		actual, err := prompt.Select(context.Background(), port.SelectRequest{
			Label:   "Branch family",
			Options: options,
			Default: "fix",
		})
		if err != nil || actual != "fix" {
			t.Fatalf("Select() = (%q, %v)", actual, err)
		}
	})

	t.Run("confirm values", func(t *testing.T) {
		prompt := New(Options{
			Input:  strings.NewReader("wrong\nyes\n"),
			Output: &bytes.Buffer{},
		})
		actual, err := prompt.Confirm(context.Background(), port.ConfirmRequest{Label: "Continue", Default: false})
		if err != nil || !actual {
			t.Fatalf("Confirm() = (%t, %v)", actual, err)
		}
	})
}

func TestColorModeControlsPlainFallbackAndLineStyling(t *testing.T) {
	t.Parallel()

	plain := New(Options{Color: "never"})
	if plain.useForms {
		t.Fatal("color=never must use the plain line-oriented prompt instead of a styled form")
	}

	output := &bytes.Buffer{}
	prompt := New(Options{
		Input:  strings.NewReader("ABC\n"),
		Output: output,
		Color:  "always",
	})
	if _, err := prompt.Input(context.Background(), port.InputRequest{Label: "Ticket key", Required: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "\x1b[36mTicket key\x1b[0m") {
		t.Fatalf("color=always prompt output = %q", output.String())
	}
}

func TestPromptInputFailures(t *testing.T) {
	t.Parallel()

	t.Run("empty options", func(t *testing.T) {
		_, err := New(Options{Input: strings.NewReader(""), Output: &bytes.Buffer{}}).Select(context.Background(), port.SelectRequest{Label: "Family"})
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("closed input", func(t *testing.T) {
		_, err := New(Options{Input: strings.NewReader(""), Output: &bytes.Buffer{}}).Input(context.Background(), port.InputRequest{Label: "Key"})
		assertProblemCode(t, err, problem.CodeOperationCancelled)
	})

	t.Run("missing label", func(t *testing.T) {
		_, err := New(Options{Input: strings.NewReader("value\n"), Output: &bytes.Buffer{}}).Input(context.Background(), port.InputRequest{})
		assertProblemCode(t, err, problem.CodeInvalidInput)
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := New(Options{Input: strings.NewReader("value\n"), Output: &bytes.Buffer{}}).Input(ctx, port.InputRequest{Label: "Key"})
		assertProblemCode(t, err, problem.CodeOperationCancelled)
	})
}

func TestFormAdapterAccessibleMode(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	prompt := New(Options{
		Input:      strings.NewReader("1\n"),
		Output:     output,
		Accessible: true,
		UseForms:   true,
	})
	actual, err := prompt.Select(context.Background(), port.SelectRequest{
		Label: "Branch family",
		Options: []port.SelectOption{
			{Value: "feature", Label: "Feature"},
			{Value: "fix", Label: "Fix"},
		},
	})
	if err != nil || actual != "feature" {
		t.Fatalf("form Select() = (%q, %v); output=%q", actual, err, output.String())
	}
}

func TestFormInputAndConfirmAccessibleMode(t *testing.T) {
	t.Parallel()

	t.Run("input", func(t *testing.T) {
		output := &bytes.Buffer{}
		prompt := New(Options{
			Input:      strings.NewReader("ABC\n"),
			Output:     output,
			Accessible: true,
			UseForms:   true,
		})
		actual, err := prompt.Input(context.Background(), port.InputRequest{
			Label:    "Ticket key",
			Required: true,
		})
		if err != nil || actual != "ABC" {
			t.Fatalf("form Input() = (%q, %v); output=%q", actual, err, output.String())
		}
	})

	t.Run("confirm", func(t *testing.T) {
		output := &bytes.Buffer{}
		prompt := New(Options{
			Input:      strings.NewReader("y\n"),
			Output:     output,
			Accessible: true,
			UseForms:   true,
		})
		actual, err := prompt.Confirm(context.Background(), port.ConfirmRequest{
			Label:   "Continue",
			Default: false,
		})
		if err != nil || !actual {
			t.Fatalf("form Confirm() = (%t, %v); output=%q", actual, err, output.String())
		}
	})
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
