package tui_test

import (
	"os"
	"regexp"
	"testing"
)

// benchSmoke finds the pattern of the benchmark smoke step in ci.yml.
var benchSmoke = regexp.MustCompile(`go test -run '\^\$' -bench '([^']+)' -benchtime=100x \./\.\.\.`)

// The benchmark smoke step in CI runs a View benchmark for avatar and for
// faces: its -bench pattern matches a benchmark that each package declares,
// and docs/testing.md quotes the same command.
func TestCIBenchSmokeRunsAvatarAndFaces(t *testing.T) {
	ci, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	found := benchSmoke.FindSubmatch(ci)
	if found == nil {
		t.Fatal("ci.yml has no benchmark smoke step of the expected form")
	}
	pattern, err := regexp.Compile(string(found[1]))
	if err != nil {
		t.Fatal(err)
	}
	decl := regexp.MustCompile(`(?m)^func (Benchmark\w+)\(`)
	for dir, file := range map[string]string{"avatar": "avatar/cache_test.go", "faces": "faces/bench_test.go"} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		matched := false
		for _, m := range decl.FindAllSubmatch(src, -1) {
			matched = matched || pattern.Match(m[1])
		}
		if !matched {
			t.Errorf("the CI pattern %q matches no benchmark in %s", found[1], dir)
		}
	}
	doc, err := os.ReadFile("docs/testing.md")
	if err != nil {
		t.Fatal(err)
	}
	if got := benchSmoke.FindSubmatch(doc); got == nil || string(got[1]) != string(found[1]) {
		t.Errorf("docs/testing.md does not quote CI's benchmark command with pattern %q", found[1])
	}
}
