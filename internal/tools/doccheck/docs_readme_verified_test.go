package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// A terminal or screen reader may be called verified only with a recorded
// human_review check, which a test cannot see. Until then the README must mark
// each of them unverified; a row that says "verified (human_review #N)" is the
// one way to change that, and names the check.
var verifiedRow = regexp.MustCompile(`\bverified \(human_review #\d+\)`)

func TestReadmeMarksTerminalsWithoutAHumanReviewUnverified(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	i := strings.Index(s, "## Terminal probe results")
	if i < 0 {
		t.Fatal("README has no terminal probe results section")
	}
	rest := s[i:]
	if j := strings.Index(rest[3:], "\n## "); j >= 0 {
		rest = rest[:j+3]
	}
	rows := 0
	for _, l := range strings.Split(rest, "\n") {
		if !strings.HasPrefix(l, "| ") || strings.HasPrefix(l, "| Terminal |") {
			continue
		}
		rows++
		if !strings.Contains(l, "unverified") && !verifiedRow.MatchString(l) {
			t.Errorf("row has neither \"unverified\" nor \"verified (human_review #N)\": %s", l)
		}
	}
	if rows < 5 {
		t.Errorf("found only %d terminal rows; the scan is broken", rows)
	}
}

func TestReadmeMarksScreenReadersUnverified(t *testing.T) {
	bullet := readmeBullet(t, "Accessibility.")
	for _, sr := range []string{"VoiceOver", "NVDA", "Orca"} {
		if !strings.Contains(bullet, sr) {
			t.Errorf("README accessibility bullet does not name %s", sr)
		}
	}
	if !strings.Contains(bullet, "unverified") && !verifiedRow.MatchString(bullet) {
		t.Error("README accessibility bullet must say the screen readers are unverified")
	}
}
