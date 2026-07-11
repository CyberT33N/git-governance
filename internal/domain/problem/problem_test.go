package problem

import (
	"errors"
	"strings"
	"testing"
)

func TestExitCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{name: "nil", err: nil, expected: ExitSuccess},
		{name: "untyped", err: errors.New("unexpected"), expected: ExitInternal},
		{name: "usage", err: New(Details{Code: CodeInvalidInput, Category: CategoryUsage}), expected: ExitUsage},
		{name: "governance", err: New(Details{Code: CodeTicketKeyInvalid, Category: CategoryGovernance}), expected: ExitGovernance},
		{name: "policy", err: New(Details{Code: CodePolicyBundleStale, Category: CategoryPolicy}), expected: ExitGovernance},
		{name: "repository", err: New(Details{Code: CodeRepositoryNotFound, Category: CategoryRepository}), expected: ExitRepository},
		{name: "git", err: New(Details{Code: CodeGitCommandFailed, Category: CategoryGit}), expected: ExitGit},
		{name: "config", err: New(Details{Code: CodeConfigurationInvalid, Category: CategoryConfig}), expected: ExitConfig},
		{name: "external", err: New(Details{Code: CodeExternalCommandFailed, Category: CategoryExternal}), expected: ExitExternal},
		{name: "cancelled", err: New(Details{Code: CodeOperationCancelled, Category: CategoryCancelled}), expected: ExitCancelled},
		{name: "internal", err: New(Details{Code: CodeInternal, Category: CategoryInternal}), expected: ExitInternal},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if actual := ExitCode(testCase.err); actual != testCase.expected {
				t.Fatalf("ExitCode() = %d, want %d", actual, testCase.expected)
			}
		})
	}
}

func TestProblemDefaultsAndCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("adapter failed")
	err := Wrap(Details{}, cause)

	if err.Code != CodeInternal {
		t.Fatalf("Code = %q, want %q", err.Code, CodeInternal)
	}
	if err.Category != CategoryInternal {
		t.Fatalf("Category = %q, want %q", err.Category, CategoryInternal)
	}
	if !errors.Is(err, cause) {
		t.Fatal("wrapped problem does not preserve the cause")
	}

	actual, ok := As(err)
	if !ok || actual != err {
		t.Fatalf("As() = (%v, %t), want original problem and true", actual, ok)
	}
}

func TestProblemErrorDoesNotExposeActualValue(t *testing.T) {
	t.Parallel()

	cause := errors.New("upstream returned sensitive-token-value")
	err := New(Details{
		Code:            CodeConfigurationInvalid,
		Category:        CategoryConfig,
		Field:           "token",
		Actual:          "sensitive-token-value",
		SensitiveActual: true,
		Rule:            "configuration value is invalid",
	})
	err = Wrap(err.Details, cause)

	if strings.Contains(err.Error(), err.Actual) {
		t.Fatalf("Error() exposed actual value: %q", err.Error())
	}
	if strings.Contains(err.Error(), cause.Error()) {
		t.Fatalf("Error() exposed causal diagnostic: %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("Error() must remain unwrap-compatible with its cause")
	}
	if !strings.Contains(err.Error(), string(CodeConfigurationInvalid)) {
		t.Fatalf("Error() = %q, want code", err.Error())
	}
}

func TestAsRejectsUntypedError(t *testing.T) {
	t.Parallel()

	actual, ok := As(errors.New("plain"))
	if actual != nil || ok {
		t.Fatalf("As() = (%v, %t), want (nil, false)", actual, ok)
	}
}

func TestWithWorkflowInputsPreservesClassifiedFailures(t *testing.T) {
	t.Parallel()

	baseCause := errors.New("Git failed")
	base := Wrap(Details{
		Code:     CodeGitCommandFailed,
		Category: CategoryGit,
		Field:    "git operation",
	}, baseCause)
	inputs := []WorkflowInput{
		{Field: "ticket number", Value: "1"},
		{Field: "branch description", Value: "add-export"},
	}

	enriched := WithWorkflowInputs(base, inputs)
	actual, ok := As(enriched)
	if !ok {
		t.Fatalf("WithWorkflowInputs() result = %T, want classified problem", enriched)
	}
	if actual == base || len(actual.WorkflowInputs) != 2 {
		t.Fatalf("workflow input result = %#v", actual)
	}
	if !errors.Is(enriched, base) || !errors.Is(enriched, baseCause) {
		t.Fatal("workflow input enrichment did not preserve the error chain")
	}
	inputs[0].Value = "mutated"
	if actual.WorkflowInputs[0].Value != "1" {
		t.Fatalf("workflow inputs alias caller memory: %#v", actual.WorkflowInputs)
	}

	if got := WithWorkflowInputs(nil, inputs); got != nil {
		t.Fatalf("nil input error = %v", got)
	}
	if got := WithWorkflowInputs(base, nil); got != base {
		t.Fatalf("empty input list returned %v, want original problem", got)
	}
	plain := errors.New("plain")
	enrichedPlain := WithWorkflowInputs(plain, inputs)
	plainProblem, ok := As(enrichedPlain)
	if !ok || plainProblem.Code != CodeInternal || !errors.Is(enrichedPlain, plain) {
		t.Fatalf("untyped input error = %#v", enrichedPlain)
	}
}

func TestProblemErrorAndUnwrapZeroPaths(t *testing.T) {
	t.Parallel()

	var nilProblem *Problem
	if nilProblem.Error() != "" || nilProblem.Unwrap() != nil {
		t.Fatal("nil problem must expose empty error text and nil cause")
	}

	expectedOnly := New(Details{
		Code:     CodeInvalidInput,
		Category: CategoryUsage,
		Expected: "a valid value",
	})
	if got := expectedOnly.Error(); !strings.Contains(got, "expected a valid value") {
		t.Fatalf("Error() = %q", got)
	}
	ruleOnly := New(Details{
		Code:     CodeInvalidInput,
		Category: CategoryUsage,
		Rule:     "must be valid",
	})
	if got := ruleOnly.Error(); !strings.Contains(got, "must be valid") {
		t.Fatalf("Error() = %q", got)
	}
	if expectedOnly.Unwrap() != nil {
		t.Fatal("problem without a cause must unwrap to nil")
	}
}
