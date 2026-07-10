package branchapp

import (
	"context"
	"strings"
	"testing"

	"github.com/CyberT33N/git-governance/internal/application/port"
	"github.com/CyberT33N/git-governance/internal/domain/branch"
	"github.com/CyberT33N/git-governance/internal/domain/problem"
)

func TestParsePrePushUpdates(t *testing.T) {
	t.Parallel()

	zero := strings.Repeat("0", 40)
	local := strings.Repeat("a", 40)
	remote := strings.Repeat("b", 40)
	input := strings.Join([]string{
		"refs/heads/feature/ABC-123-add-export " + local + " refs/heads/feature/ABC-123-add-export " + zero,
		"HEAD " + local + " refs/heads/main " + remote,
		"(delete) " + zero + " refs/heads/scratch/ABC-123-experiment " + remote,
		"refs/tags/v1.0.0 " + local + " refs/tags/v1.0.0 " + zero,
	}, "\n") + "\n"

	updates, err := ParsePrePushUpdates(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 4 {
		t.Fatalf("len(ParsePrePushUpdates()) = %d, want 4", len(updates))
	}

	if updates[0].Action != PushActionCreate || !updates[0].GovernedBranch || updates[0].Target.String() != "feature/ABC-123-add-export" {
		t.Fatalf("create update = %#v", updates[0])
	}
	if updates[1].Action != PushActionUpdate || updates[1].Target.String() != "main" || updates[1].LocalRef != "HEAD" {
		t.Fatalf("HEAD:main update = %#v", updates[1])
	}
	if updates[2].Action != PushActionDelete || updates[2].Target.String() != "scratch/ABC-123-experiment" {
		t.Fatalf("delete update = %#v", updates[2])
	}
	if updates[3].Action != PushActionOther || updates[3].GovernedBranch {
		t.Fatalf("tag update = %#v", updates[3])
	}
}

func TestParsePrePushUpdatesRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	validOID := strings.Repeat("a", 40)
	for _, testCase := range []struct {
		raw  string
		code problem.Code
	}{
		{raw: "too few fields", code: problem.CodeInvalidInput},
		{raw: "refs/heads/main " + validOID + " refs/heads/main invalid", code: problem.CodeInvalidInput},
		{raw: "refs/heads/main " + validOID + " main " + validOID, code: problem.CodeInvalidInput},
		{raw: "refs/heads/main " + validOID + " refs/heads/feat/ABC-123-invalid " + validOID, code: problem.CodeBranchNameInvalid},
		{raw: "(delete) " + validOID + " refs/heads/feature/ABC-123-add-export " + validOID, code: problem.CodeInvalidInput},
		{raw: "refs/heads/feature/ABC-123-add-export " + strings.Repeat("0", 40) + " refs/heads/feature/ABC-123-add-export " + validOID + "\n\n", code: problem.CodeInvalidInput},
	} {
		testCase := testCase
		t.Run(testCase.raw, func(t *testing.T) {
			t.Parallel()
			_, err := ParsePrePushUpdates(strings.NewReader(testCase.raw))
			assertProblemCode(t, err, testCase.code)
		})
	}
}

func TestValidatePrePushUpdatesBlocksActualSharedLineTargets(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{}
	synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
	update := pushUpdate(t, "HEAD", "main", PushActionUpdate)

	_, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{update}, nil)
	assertProblemCode(t, err, problem.CodeSharedLineMutationForbidden)
	if got := strings.Join(git.calls, ","); got != "fetch,validate-ref" {
		t.Fatalf("calls = %q, want fetch,validate-ref", got)
	}
}

func TestValidatePrePushUpdatesValidatesEveryBranchUpdate(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		inspections: []port.PushUpdateInspection{{
			FastForward:    true,
			CommitMessages: []string{"feat(ABC-123): add export"},
		}},
	}
	synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
	feature := pushUpdate(t, "refs/heads/feature/ABC-123-add-export", "feature/ABC-123-add-export", PushActionCreate)
	main := pushUpdate(t, "HEAD", "main", PushActionUpdate)

	_, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{feature, main}, nil)
	assertProblemCode(t, err, problem.CodeSharedLineMutationForbidden)
	if got := strings.Join(git.calls, ","); got != "fetch,validate-ref,inspect-push,validate-ref" {
		t.Fatalf("calls = %q", got)
	}
}

func TestValidatePrePushUpdatesRunsQualityOnceForAllGovernedFamilies(t *testing.T) {
	t.Parallel()

	git := &fakeGitRepository{
		inspections: []port.PushUpdateInspection{
			{FastForward: true, CommitMessages: []string{"feat(ABC-123): add export"}},
			{FastForward: true, CommitMessages: []string{"docs(ABC-124): update guide"}},
		},
	}
	quality := &fakeQualityRunner{}
	synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), quality)

	result, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
		pushUpdate(t, "refs/heads/feature/ABC-123-add-export", "feature/ABC-123-add-export", PushActionCreate),
		pushUpdate(t, "refs/heads/docs/ABC-124-update-guide", "docs/ABC-124-update-guide", PushActionCreate),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if quality.calls != 1 || len(quality.requests) != 1 {
		t.Fatalf("quality calls = %d, requests = %#v", quality.calls, quality.requests)
	}
	families := quality.requests[0].Families
	if len(families) != 2 || families[0] != branch.FamilyFeature || families[1] != branch.FamilyDocs {
		t.Fatalf("quality families = %v", families)
	}
	if result.Quality.Status != port.QualityPassed {
		t.Fatalf("quality result = %#v", result.Quality)
	}
}

func TestValidatePrePushUpdatesEnforcesFirstPushAndForcePushPolicy(t *testing.T) {
	t.Parallel()

	t.Run("stale first push", func(t *testing.T) {
		git := &fakeGitRepository{
			inspections: []port.PushUpdateInspection{{
				MissingBaseCommits: true,
				FastForward:        true,
				CommitMessages:     []string{"feat(ABC-123): add export"},
			}},
		}
		synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
			pushUpdate(t, "refs/heads/feature/ABC-123-add-export", "feature/ABC-123-add-export", PushActionCreate),
		}, nil)
		assertProblemCode(t, err, problem.CodeBranchBaseInvalid)
	})

	t.Run("non-fast-forward published update", func(t *testing.T) {
		git := &fakeGitRepository{
			inspections: []port.PushUpdateInspection{{
				FastForward:    false,
				CommitMessages: []string{"feat(ABC-123): add export"},
			}},
		}
		synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
			pushUpdate(t, "refs/heads/feature/ABC-123-add-export", "feature/ABC-123-add-export", PushActionUpdate),
		}, nil)
		assertProblemCode(t, err, problem.CodeForcePushForbidden)
	})
}

func TestValidatePrePushUpdatesHandlesDeletionAndNonBranchRefs(t *testing.T) {
	t.Parallel()

	t.Run("working branch deletion is classified", func(t *testing.T) {
		git := &fakeGitRepository{}
		synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		result, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
			pushUpdate(t, "(delete)", "feature/ABC-123-add-export", PushActionDelete),
		}, nil)
		if err != nil || len(result.Updates) != 1 || result.Updates[0].Update.Action != PushActionDelete {
			t.Fatalf("ValidatePrePushUpdates() = (%#v, %v)", result, err)
		}
		if result.Quality.Status != port.QualitySkipped {
			t.Fatalf("deletion quality = %#v", result.Quality)
		}
	})

	t.Run("shared line deletion is forbidden", func(t *testing.T) {
		git := &fakeGitRepository{}
		synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		_, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
			pushUpdate(t, "(delete)", "main", PushActionDelete),
		}, nil)
		assertProblemCode(t, err, problem.CodeSharedLineMutationForbidden)
	})

	t.Run("tag ref is explicitly skipped", func(t *testing.T) {
		git := &fakeGitRepository{}
		synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
		result, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{{
			LocalRef:       "refs/tags/v1.0.0",
			LocalObjectID:  strings.Repeat("a", 40),
			RemoteRef:      "refs/tags/v1.0.0",
			RemoteObjectID: strings.Repeat("0", 40),
			Action:         PushActionOther,
		}}, nil)
		if err != nil || len(result.Updates) != 1 || !result.Updates[0].Skipped {
			t.Fatalf("ValidatePrePushUpdates() = (%#v, %v)", result, err)
		}
		if result.Quality.Status != port.QualitySkipped {
			t.Fatalf("tag quality = %#v", result.Quality)
		}
	})
}

func pushUpdate(t *testing.T, localRef, target string, action PushAction) PushUpdate {
	t.Helper()
	name := mustBranch(target)
	localObjectID := strings.Repeat("a", 40)
	remoteObjectID := strings.Repeat("b", 40)
	if action == PushActionCreate {
		remoteObjectID = strings.Repeat("0", 40)
	}
	if action == PushActionDelete {
		localObjectID = strings.Repeat("0", 40)
	}
	return PushUpdate{
		LocalRef:       localRef,
		LocalObjectID:  localObjectID,
		RemoteRef:      "refs/heads/" + target,
		RemoteObjectID: remoteObjectID,
		Target:         name,
		Action:         action,
		GovernedBranch: true,
	}
}

func TestValidatePrePushUpdatesUsesExplicitHotfixBase(t *testing.T) {
	t.Parallel()

	base := mustBase("origin", "main")
	git := &fakeGitRepository{
		inspections: []port.PushUpdateInspection{{
			FastForward:    true,
			CommitMessages: []string{"fix(ABC-999): resolve timeout"},
		}},
	}
	synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
	result, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
		pushUpdate(t, "refs/heads/hotfix/ABC-999-payment-timeout", "hotfix/ABC-999-payment-timeout", PushActionCreate),
	}, &base)
	if err != nil || len(result.Updates) != 1 || result.Updates[0].Base == nil || result.Updates[0].Base.String() != "origin/main" {
		t.Fatalf("ValidatePrePushUpdates() = (%#v, %v)", result, err)
	}
}

func TestValidatePrePushUpdatesUsesStoredWorkflowBase(t *testing.T) {
	t.Parallel()

	base := mustBase("origin", "main")
	git := &fakeGitRepository{
		workflowBase: &base,
		inspections: []port.PushUpdateInspection{{
			FastForward:    true,
			CommitMessages: []string{"fix(ABC-999): resolve timeout"},
		}},
	}
	synchronizer := NewSynchronizer(git, NewService(git, &fakeKeyPolicy{}), nil)
	result, err := synchronizer.ValidatePrePushUpdates(context.Background(), testRepository(), []PushUpdate{
		pushUpdate(t, "refs/heads/hotfix/ABC-999-payment-timeout", "hotfix/ABC-999-payment-timeout", PushActionCreate),
	}, nil)
	if err != nil || len(result.Updates) != 1 || result.Updates[0].Base == nil || result.Updates[0].Base.String() != "origin/main" {
		t.Fatalf("ValidatePrePushUpdates() = (%#v, %v)", result, err)
	}
}

func TestValidateCommitSeries(t *testing.T) {
	t.Parallel()

	name := mustBranch("feature/ABC-123-add-export")
	if err := ValidateCommitSeries(name, []string{"feat(ABC-123): add export", "test(ABC-123): cover export"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommitSeries(name, nil); err == nil {
		t.Fatal("ValidateCommitSeries accepted no commits")
	}
	err := ValidateCommitSeries(name, []string{"feat(ABC-124): unrelated"})
	assertProblemCode(t, err, problem.CodeCommitTicketMismatch)
}

func TestPushUpdateParserAcceptsSHA256(t *testing.T) {
	t.Parallel()

	oid := strings.Repeat("a", 64)
	input := "refs/heads/feature/ABC-123-add-export " + oid + " refs/heads/feature/ABC-123-add-export " + strings.Repeat("0", 64)
	updates, err := ParsePrePushUpdates(strings.NewReader(input))
	if err != nil || len(updates) != 1 || updates[0].LocalObjectID != oid {
		t.Fatalf("ParsePrePushUpdates() = (%#v, %v)", updates, err)
	}
}

func TestPushUpdateResultRetainsTargetFamily(t *testing.T) {
	t.Parallel()

	update := pushUpdate(t, "HEAD", "feature/ABC-123-add-export", PushActionUpdate)
	if update.Target.Family() != branch.FamilyFeature {
		t.Fatalf("target family = %q", update.Target.Family())
	}
}
