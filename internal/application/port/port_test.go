package port

import "testing"

func TestQualityStatusValuesRemainStable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status QualityStatus
		want   string
	}{
		{status: QualityUnconfigured, want: "unconfigured"},
		{status: QualitySkipped, want: "skipped"},
		{status: QualityPassed, want: "passed"},
	}

	for _, testCase := range testCases {
		if string(testCase.status) != testCase.want {
			t.Errorf("QualityStatus %q = %q, want %q", testCase.status, testCase.status, testCase.want)
		}
	}
}
