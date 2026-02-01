package semantic

import "testing"

func TestClickPlan_OrderTargets(t *testing.T) {
	plan := BuildClickPlan([]string{"a", "b"})
	if plan[0] != "a" {
		t.Fatalf("expected first target to be a")
	}
}
