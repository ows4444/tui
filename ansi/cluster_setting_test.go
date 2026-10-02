package ansi

import "testing"

// The default comes from TUI_NO_CLUSTERS when first read, not at import, and an
// explicit Store always wins.
func TestClusterSettingResolvesEnvironmentLazily(t *testing.T) {
	t.Setenv("TUI_NO_CLUSTERS", "1")
	var off clusterSetting
	if off.Load() {
		t.Error("TUI_NO_CLUSTERS=1: clusters on, want off")
	}
	t.Setenv("TUI_NO_CLUSTERS", "")
	var on clusterSetting
	if !on.Load() {
		t.Error("TUI_NO_CLUSTERS empty: clusters off, want on")
	}
	off.Store(true)
	if !off.Load() {
		t.Error("Store(true) did not override the environment default")
	}
	// Once decided, the environment is not consulted again.
	t.Setenv("TUI_NO_CLUSTERS", "1")
	if !on.Load() {
		t.Error("decided setting changed when the environment did")
	}
}
