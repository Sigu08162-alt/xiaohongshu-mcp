package semantic

import "testing"

func TestLoadConfig_Defaults(t *testing.T) {
	_, err := LoadConfig("../../configs/semantic_scan.yaml")
	if err != nil {
		t.Fatalf("expected config to load, got %v", err)
	}
}
