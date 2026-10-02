package main

import (
	"os"
	"strings"
	"testing"
)

// The CI workflow must keep a FreeBSD job that builds and runs the tests; a
// workflow edit that drops it fails here, though only a CI run shows it green.
func TestCIHasAFreeBSDJobThatRunsTheTests(t *testing.T) {
	raw, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	i := strings.Index(s, "\n  freebsd:")
	if i < 0 {
		t.Fatal("ci.yml has no freebsd job")
	}
	job := s[i+1:]
	// Cut the job off at the next top-level job (two spaces, a name, a colon).
	for _, line := range strings.SplitAfter(job, "\n")[1:] {
		if len(line) > 2 && strings.HasPrefix(line, "  ") && line[2] != ' ' && line[2] != '#' && strings.HasSuffix(strings.TrimSpace(line), ":") {
			job = job[:strings.Index(job, line)]
			break
		}
	}
	for _, want := range []string{"freebsd-vm", "go build ./...", "go test ./..."} {
		if !strings.Contains(job, want) {
			t.Errorf("the freebsd job lacks %q", want)
		}
	}
}
