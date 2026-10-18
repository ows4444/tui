package main

import (
	"strconv"
	"strings"
)

// paneTabs are the tabs of the three panes: explorer, editor, output.
var paneTabs = [3][]string{
	{"Files", "Outline"},
	{"main.go", "notes.md", "todo.txt"},
	{"Output", "Problems", "Log"},
}

var paneNames = [3]string{"Explorer", "Editor", "Panel"}

// content is the text of one tab. It is made up, fixed and long enough to
// scroll.
func content(pane, tab int) []string {
	switch {
	case pane == 0 && tab == 0:
		return []string{"▾ workspace", "  ▾ cmd", "      main.go", "  ▾ internal", "    ▸ cache", "    ▸ config", "    ▸ server", "      doc.go", "  ▾ testdata", "      golden.txt", "      sample.json", "    README.md", "    go.mod", "    go.sum", "    Makefile", "    LICENSE", "    CHANGELOG.md", "    .gitignore"}
	case pane == 0:
		return []string{"func main", "func run", "func parseFlags", "func load", "type Config", "type Server", "func (s *Server) Start", "func (s *Server) Stop", "func (s *Server) handle", "func (s *Server) route", "var ErrClosed"}
	case pane == 1 && tab == 0:
		return numbered([]string{"package main", "", "import (", `	"flag"`, `	"fmt"`, `	"os"`, ")", "", "// run starts the server and waits for it to stop.", "func run(addr string) error {", "	s := newServer(addr)", "	if err := s.Start(); err != nil {", "		return fmt.Errorf(\"start: %w\", err)", "	}", "	defer s.Stop()", "	return s.Wait()", "}", "", "func main() {", `	addr := flag.String("addr", ":8080", "listen address")`, "	flag.Parse()", "	if err := run(*addr); err != nil {", `		fmt.Fprintln(os.Stderr, "error:", err)`, "		os.Exit(1)", "	}", "}"})
	case pane == 1 && tab == 1:
		return []string{"# Notes", "", "- The server starts on :8080 unless -addr says otherwise.", "- Stop waits for open requests, then closes the listener.", "- The cache is cleared on every restart; that is on purpose.", "", "## Open questions", "", "1. Should Stop give up after a timeout?", "2. Where do the logs go when stdout is closed?", "3. Is the route table safe to read while it is rebuilt?", "", "## Later", "", "- Metrics on /debug.", "- A health check that looks at the cache."}
	case pane == 1:
		return []string{"[x] write the server", "[x] add the config loader", "[ ] time out Stop", "[ ] log to a file", "[ ] document the flags", "[ ] tag a release"}
	case pane == 2 && tab == 0:
		var out []string
		for i := 1; i <= 30; i++ {
			out = append(out, "build step "+strconv.Itoa(i)+" of 30: ok")
		}
		return append(out, "done in 2.4s")
	case pane == 2 && tab == 1:
		return []string{"main.go:12:5  declared and not used: s", "server.go:48:2  missing return", "config.go:7:1  exported type Config should have a comment", "3 problems"}
	}
	return []string{"10:00:01 info  listening on :8080", "10:00:07 info  GET /health 200 1ms", "10:00:09 warn  slow route /report 412ms", "10:00:15 info  GET /health 200 1ms", "10:00:21 error cache: write failed: disk full", "10:00:21 info  retrying in 5s"}
}

func numbered(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		n := strconv.Itoa(i + 1)
		out[i] = "      "[:max(0, 4-len(n))] + n + "  " + strings.ReplaceAll(l, "\t", "    ")
	}
	return out
}
