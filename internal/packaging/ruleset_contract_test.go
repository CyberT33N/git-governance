package packaging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProtectedLineRulesetsAllowInitialCreation(t *testing.T) {
	t.Parallel()

	for _, fileName := range []string{"04-release.json", "05-support.json"} {
		fileName := fileName
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			contents, err := os.ReadFile(filepath.Join(
				repositoryRoot(t),
				"docs",
				"hosting-platforms",
				"github",
				"rulesets",
				fileName,
			))
			if err != nil {
				t.Fatal(err)
			}

			var ruleset struct {
				Rules []struct {
					Type       string `json:"type"`
					Parameters struct {
						DoNotEnforceOnCreate bool `json:"do_not_enforce_on_create"`
						RequiredStatusChecks []struct {
							Context string `json:"context"`
						} `json:"required_status_checks"`
					} `json:"parameters"`
				} `json:"rules"`
			}
			if err := json.Unmarshal(contents, &ruleset); err != nil {
				t.Fatal(err)
			}

			var statusRuleFound bool
			for _, rule := range ruleset.Rules {
				if rule.Type != "required_status_checks" {
					continue
				}
				statusRuleFound = true
				if !rule.Parameters.DoNotEnforceOnCreate {
					t.Fatal("required status checks must not be enforced when the protected line is first created")
				}
				if got, want := statusCheckContexts(rule.Parameters.RequiredStatusChecks), []string{
					"Quality gates (linux-amd64)",
					"Quality gates (macos-arm64)",
					"Quality gates (windows-amd64)",
				}; !equalStrings(got, want) {
					t.Fatalf("required status checks = %#v, want %#v", got, want)
				}
			}
			if !statusRuleFound {
				t.Fatal("ruleset does not define required status checks")
			}
		})
	}
}

func statusCheckContexts(checks []struct {
	Context string `json:"context"`
}) []string {
	contexts := make([]string, len(checks))
	for index, check := range checks {
		contexts[index] = check.Context
	}
	return contexts
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
