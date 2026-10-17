package main

import (
	"bufio"
	"fmt"
	"strings"
)

// command is one thing the launcher can run. With shell set it runs that
// shell command line; without, it plays its script, so the sample project runs
// the same everywhere and touches nothing.
type command struct {
	name   string
	desc   string
	shell  string
	script []string // lines the simulated run prints
	exit   int      // exit code of the simulated run
}

// sampleCommands is the project the launcher opens when given no file.
func sampleCommands() []command {
	return []command{
		{name: "build", desc: "Compile the project", script: []string{"go build ./...", "compiling 41 packages", "linking bin/app", "done"}},
		{name: "test", desc: "Run the tests with the race detector", script: []string{"=== RUN   TestParse", "--- PASS: TestParse (0.00s)", "=== RUN   TestRender", "--- PASS: TestRender (0.01s)", "=== RUN   TestMouse", "--- PASS: TestMouse (0.02s)", "PASS", "ok  \tapp\t0.412s"}},
		{name: "test-flaky", desc: "Run the tests that sometimes fail", script: []string{"=== RUN   TestRetry", "--- FAIL: TestRetry (0.30s)", "    retry_test.go:41: got 2 attempts, want 3", "FAIL"}, exit: 1},
		{name: "lint", desc: "Check the code with the linters", script: []string{"staticcheck ./...", "gosec ./...", "golangci-lint run", "0 issues"}},
		{name: "fmt", desc: "Format every source file", script: []string{"gofmt -l -w .", "3 files changed"}},
		{name: "bench", desc: "Run the benchmarks", script: []string{"BenchmarkFrame-8      18432     64211 ns/op      5 allocs/op", "BenchmarkTable-8      49120     24100 ns/op      1 allocs/op", "PASS"}},
		{name: "docs", desc: "Build the documentation site", script: []string{"reading 38 pages", "building search index", "writing site/", "done in 1.4s"}},
		{name: "clean", desc: "Remove build output", script: []string{"rm -rf bin/ dist/", "removed 2 directories"}},
		{name: "deploy-staging", desc: "Deploy the last build to staging", script: []string{"uploading bin/app (4.1 MB)", "restarting 3 instances", "instance 1 healthy", "instance 2 healthy", "instance 3 healthy", "staging is live"}},
		{name: "deploy-production", desc: "Deploy to production (fails: not signed off)", script: []string{"checking sign-off", "error: release 1.4.0 has no sign-off"}, exit: 2},
		{name: "release", desc: "Tag, build and publish a release", script: []string{"tagging v1.4.0", "building 5 targets", "signing archives", "publishing", "release v1.4.0 is out"}},
	}
}

// parseCommands reads a command file: one "name | description | shell command"
// per line, blank lines and lines starting with # ignored.
func parseCommands(text string) ([]command, error) {
	var out []command
	seen := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(text))
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("line %d: want \"name | description | command\"", n)
		}
		c := command{name: strings.TrimSpace(parts[0]), desc: strings.TrimSpace(parts[1]), shell: strings.TrimSpace(parts[2])}
		switch {
		case c.name == "" || c.shell == "":
			return nil, fmt.Errorf("line %d: the name and the command cannot be empty", n)
		case seen[c.name]:
			return nil, fmt.Errorf("line %d: %s appears twice", n, c.name)
		}
		seen[c.name] = true
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no commands found")
	}
	return out, sc.Err()
}
