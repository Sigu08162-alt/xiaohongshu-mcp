package semantic

import "testing"

func TestFingerprint_DeltaAboveThreshold(t *testing.T) {
	before := Fingerprint{VisibleCount: 100}
	after := Fingerprint{VisibleCount: 160}

	if !before.Delta(after).Above(0.5) {
		t.Fatalf("expected delta above threshold")
	}
}

func TestFingerprint_ChangedMetricCount(t *testing.T) {
	before := Fingerprint{
		VisibleCount:   100,
		ButtonCount:    10,
		InputCount:     5,
		ContainerCount: 20,
	}
	after := Fingerprint{
		VisibleCount:   120,
		ButtonCount:    10,
		InputCount:     7,
		ContainerCount: 20,
	}

	if got := before.Delta(after).ChangedMetricCount(0.1); got < 1 {
		t.Fatalf("expected at least one metric change, got %d", got)
	}
}
