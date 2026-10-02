package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// readmeBullet returns the README "Platforms and limitations" bullet that
// starts with the bold label.
func readmeBullet(t *testing.T, label string) string {
	t.Helper()
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	i := strings.Index(s, "- **"+label+"**")
	if i < 0 {
		t.Fatalf("README has no %q bullet", label)
	}
	rest := s[i+2:]
	if j := strings.Index(rest, "\n- **"); j >= 0 {
		rest = rest[:j]
	}
	return strings.Join(strings.Fields(rest), " ")
}

// TestReadmeColourClaimsMatchDetection fails when the README's colour
// statement contradicts NewProgram's default detection: it must not claim
// truecolor by default, must name every variable ansi/profile.go reads, and
// must not say detection needs an opt-in.
func TestReadmeColourClaimsMatchDetection(t *testing.T) {
	bullet := readmeBullet(t, "Colour.")

	// The default is detection, not truecolor: with nothing set and a
	// non-terminal output, the profile is not TrueColor.
	if got := ansi.DetectColorProfileFor(&strings.Builder{}, func(string) string { return "" }); got == ansi.TrueColor {
		t.Fatal("test premise broken: default detection is truecolor")
	}
	lower := strings.ToLower(bullet)
	for _, bad := range []string{"truecolor by default", "true colour by default", "to also honour"} {
		if strings.Contains(lower, bad) {
			t.Errorf("README colour bullet contains %q, which contradicts default detection", bad)
		}
	}

	src, err := os.ReadFile("ansi/profile.go")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`getenv\("([A-Z_]+)"\)`).FindAllStringSubmatch(string(src), -1) {
		seen[m[1]] = true
	}
	if len(seen) < 6 {
		t.Fatalf("found only %d variables in ansi/profile.go; the scan is broken", len(seen))
	}
	for v := range seen {
		if !strings.Contains(bullet, v) {
			t.Errorf("README colour bullet does not mention %s, which ansi/profile.go reads", v)
		}
	}
}

// TestReadmeUnicodeClaimsMatchClusterDefault fails when the README says
// grapheme clusters are (not) segmented against what ansi.Width does by
// default.
func TestReadmeUnicodeClaimsMatchClusterDefault(t *testing.T) {
	if os.Getenv("TUI_NO_CLUSTERS") != "" {
		t.Skip("cluster handling disabled in this environment; default differs")
	}
	bullet := strings.ToLower(readmeBullet(t, "Unicode."))
	family := "\U0001F468‍\U0001F469‍\U0001F467"
	segmented := ansi.Width(family) == 2
	says := strings.Contains(bullet, "grapheme cluster") &&
		!strings.Contains(bullet, "not** segmented") && !strings.Contains(bullet, "not segmented")
	if segmented != says {
		t.Fatalf("family emoji is segmented by default = %v, but README claims segmented = %v", segmented, says)
	}
}

// TestReadmeListsUnsupportedPlatforms fails when the README Platforms
// bullet does not name solaris and illumos as unsupported.
func TestReadmeListsUnsupportedPlatforms(t *testing.T) {
	b := strings.ToLower(readmeBullet(t, "Platforms."))
	for _, want := range []string{"illumos", "solaris", "unsupported"} {
		if !strings.Contains(b, want) {
			t.Errorf("README Platforms bullet does not mention %q", want)
		}
	}
}
