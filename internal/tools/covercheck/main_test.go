package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mod = "github.com/ows4444/tui/"

func TestBelowFloorIsNamed(t *testing.T) {
	profile := "mode: set\n" +
		mod + "widget/a.go:1.1,2.2 8 1\n" +
		mod + "widget/a.go:3.1,4.2 2 0\n" // 80%
	bad, _, err := check(strings.NewReader(profile), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "widget") || !strings.Contains(bad[0], "80.0%") {
		t.Fatalf("got %q, want widget at 80.0%%", bad)
	}
}

func TestAtOrAboveFloorPasses(t *testing.T) {
	profile := "mode: atomic\n" +
		mod + "widget/a.go:1.1,2.2 9 3\n" +
		mod + "widget/b.go:1.1,2.2 1 0\n" + // exactly 90%
		mod + "a.go:1.1,2.2 1 1\n" // root package
	bad, _, err := check(strings.NewReader(profile), nil)
	if err != nil || len(bad) != 0 {
		t.Fatalf("got %q, %v; want none", bad, err)
	}
}

func TestExcludedPathsAreIgnored(t *testing.T) {
	profile := "mode: set\n" +
		mod + "examples/demo/main.go:1.1,2.2 5 0\n" +
		mod + "internal/tools/covercheck/main.go:1.1,2.2 5 0\n" +
		mod + "internal/highlight/h.go:1.1,2.2 5 0\n" // not excluded
	bad, _, err := check(strings.NewReader(profile), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "internal/highlight") {
		t.Fatalf("got %q, want only internal/highlight", bad)
	}
	for _, b := range bad {
		if strings.Contains(b, "examples") || strings.Contains(b, "internal/tools") {
			t.Errorf("excluded path reported: %q", b)
		}
	}
}

func TestMalformedProfile(t *testing.T) {
	if _, _, err := check(strings.NewReader("mode: set\nnot a profile line\n"), nil); err == nil {
		t.Fatal("want an error")
	}
	if _, _, err := check(strings.NewReader("mode: set\nx/a.go:1.1,2.2 x y\n"), nil); err == nil {
		t.Fatal("want an error for non-numeric counts")
	}
}

func TestHelperMarkedPackageIsExcludedAndNamedWithItsReason(t *testing.T) {
	profile := "mode: set\n" +
		mod + "internal/testutil/t.go:1.1,2.2 5 0\n" +
		mod + "internal/highlight/h.go:1.1,2.2 5 0\n"
	marked := func(pkg string) (string, bool) {
		if pkg == "internal/testutil" {
			return "test support", true
		}
		return "", false
	}
	bad, skipped, err := check(strings.NewReader(profile), marked)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "internal/testutil") || !strings.Contains(skipped[0], "test support") {
		t.Fatalf("skipped %q, want internal/testutil and its reason named", skipped)
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "internal/highlight") {
		t.Fatalf("bad %q, want only internal/highlight", bad)
	}
}

// An exclusion without a reason fails the check and names the package.
func TestExclusionWithoutReasonFails(t *testing.T) {
	profile := "mode: set\n" + mod + "internal/testutil/t.go:1.1,2.2 5 0\n"
	bad, skipped, err := check(strings.NewReader(profile), func(string) (string, bool) { return "  ", true })
	if err != nil {
		t.Fatal(err)
	}
	if len(bad) != 1 || !strings.Contains(bad[0], "internal/testutil") || !strings.Contains(bad[0], "no reason") {
		t.Fatalf("bad %q, want a no-reason failure naming internal/testutil", bad)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped %q; a reasonless exclusion must not be honoured", skipped)
	}
}

// Coverage another package's tests give a package counts: the same block
// listed once per test binary is one block, covered if any copy ran.
func TestCoverageFromAnotherPackagesTestsIsCredited(t *testing.T) {
	profile := "mode: set\n" +
		mod + "internal/termio/t.go:1.1,2.2 5 0\n" + // its own tests: nothing
		mod + "internal/termio/t.go:3.1,4.2 5 0\n" +
		mod + "internal/termio/t.go:1.1,2.2 5 1\n" + // a dependant's tests ran it
		mod + "internal/termio/t.go:3.1,4.2 5 2\n"
	bad, _, err := check(strings.NewReader(profile), nil)
	if err != nil || len(bad) != 0 {
		t.Fatalf("got %q, %v; want the package credited to 100%%", bad, err)
	}
}

// A block repeated across binaries is counted once, not once per copy.
func TestRepeatedBlocksAreNotDoubleCounted(t *testing.T) {
	profile := "mode: set\n" +
		mod + "w/a.go:1.1,2.2 8 1\n" +
		mod + "w/a.go:1.1,2.2 8 1\n" +
		mod + "w/a.go:3.1,4.2 2 0\n" +
		mod + "w/a.go:3.1,4.2 2 0\n" // 8 of 10 statements, not 16 of 20 either way
	bad, _, _ := check(strings.NewReader(profile), nil)
	if len(bad) != 1 || !strings.Contains(bad[0], "80.0%") {
		t.Fatalf("got %q, want w at 80.0%%", bad)
	}
}

func TestHelperMarkerReason(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/h/h.go", "//covercheck:helper PTY harness for tests\npackage h\n")
	write("internal/bare/b.go", "//covercheck:helper\npackage bare\n")
	write("internal/plain/p.go", "package plain\n")
	write("internal/lookalike/l.go", "//covercheck:helperx not a marker\npackage lookalike\n")
	write("internal/intest/x_test.go", "//covercheck:helper in a test\npackage intest\n")
	write("widget/w.go", "//covercheck:helper outside internal\npackage widget\n")
	for pkg, want := range map[string]struct {
		reason string
		ok     bool
	}{
		"internal/h":         {"PTY harness for tests", true},
		"internal/bare":      {"", true}, // marked, but reasonless: check fails it
		"internal/plain":     {"", false},
		"internal/lookalike": {"", false},
		"internal/intest":    {"", false}, // marker only counts in non-test source
		"widget":             {"", false}, // marker only counts under internal/
	} {
		if reason, ok := hasHelperMarker(pkg); reason != want.reason || ok != want.ok {
			t.Errorf("hasHelperMarker(%q) = %q, %v; want %q, %v", pkg, reason, ok, want.reason, want.ok)
		}
	}
}
