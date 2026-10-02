package archtest

import (
	"fmt"
	"slices"
	"testing"
)

// statelessKits are tier 4a: renderers with no state and no runtime. They may
// import tiers 0 to 2 and each other, never the root runtime (tier 3) or a
// runtime-coupled kit (tier 4b: focus, tuitest).
var statelessKits = []string{"widgets", "widgets/chart", "markdown"}

// runtimeKits are tier 4b.
var runtimeKits = []string{"focus", "tuitest"}

// kitViolations returns one line for each import of a stateless kit that
// reaches the root runtime or a runtime-coupled kit.
func kitViolations(pkgs map[string]map[string]bool) []string {
	var out []string
	for _, from := range statelessKits {
		for to := range pkgs[from] {
			switch {
			case to == "":
				out = append(out, fmt.Sprintf("%s (tier 4a) must not import the root package tui (tier 3)", from))
			case slices.Contains(runtimeKits, to):
				out = append(out, fmt.Sprintf("%s (tier 4a) must not import %s (tier 4b)", from, to))
			case tierOf(to) == 3:
				out = append(out, fmt.Sprintf("%s (tier 4a) must not import %s (tier 3)", from, to))
			}
		}
	}
	slices.Sort(out)
	return out
}

// rootViolations returns one line for each component package (tier 5) that the
// root package imports.
func rootViolations(pkgs map[string]map[string]bool) []string {
	var out []string
	for to := range pkgs[""] {
		if tierOf(to) == componentTier {
			out = append(out, fmt.Sprintf("tui (tier 3) must not import the component package %s (tier 5)", to))
		}
	}
	slices.Sort(out)
	return out
}

func TestStatelessKitsDoNotImportRuntime(t *testing.T) {
	pkgs := imports(t, "../..")
	for _, k := range statelessKits {
		if _, ok := pkgs[k]; !ok {
			t.Fatalf("stateless kit %s not found; is the list stale?", k)
		}
	}
	for _, v := range kitViolations(pkgs) {
		t.Error(v)
	}
}

func TestRootImportsNoComponent(t *testing.T) {
	pkgs := imports(t, "../..")
	if len(pkgs[""]) == 0 {
		t.Fatal("the root package has no recorded imports; the \"\" key is wrong")
	}
	for _, v := range rootViolations(pkgs) {
		t.Error(v)
	}
}

func TestKitTiersAreConsistent(t *testing.T) {
	for _, k := range append(slices.Clone(statelessKits), runtimeKits...) {
		if tierOf(k) != 4 {
			t.Errorf("%s is listed as a tier 4 kit but tierOf says %d", k, tierOf(k))
		}
	}
}

func TestKitRulesCatchViolations(t *testing.T) {
	got := kitViolations(map[string]map[string]bool{
		"widgets":       {"": true, "ansi": true},
		"widgets/chart": {"focus": true, "widgets": true},
		"markdown":      {"internal/render": true},
	})
	want := []string{
		"widgets (tier 4a) must not import the root package tui (tier 3)",
		"widgets/chart (tier 4a) must not import focus (tier 4b)",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
	got = rootViolations(map[string]map[string]bool{"": {"textinput": true, "ansi": true, "internal/render": true}})
	if want := []string{"tui (tier 3) must not import the component package textinput (tier 5)"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
