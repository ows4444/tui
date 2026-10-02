// Command covercheck reads a Go cover profile and fails when any library
// package has less than 90% statement coverage. It uses only the standard
// library and is not imported by any library package.
//
//	go test -coverprofile=cover.out -coverpkg=./... ./...
//	go run ./internal/tools/covercheck cover.out
//
// The floor is per package, not total. Excluded from the check, because they
// are not library code: everything under examples and tools, and any package
// under internal/ whose non-test source carries a "//covercheck:helper <reason>"
// line (a test-support package with no logic of its own to cover). The reason
// is mandatory: a marker without one fails the check. Each excluded package is
// named on stdout with its reason.
//
// Coverage is credited across packages. Run the tests with -coverpkg=./... so
// an adapter exercised only by another package's tests (internal/termio,
// internal/termio) is credited; a block that appears once per test binary is
// counted once and is covered if any copy ran.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	floor      = 90.0
	modulePath = "github.com/ows4444/tui"
)

// excluded lists the path prefixes, relative to the module root, that the
// floor does not apply to.
var excluded = []string{"examples", "internal/tools"}

// helperMarker, at the start of a line in a non-test .go file of a package
// under internal/, followed by the reason, marks that package as test-support
// code.
const helperMarker = "//covercheck:helper"

// hasHelperMarker reports whether the package at rel, a directory relative to
// the working directory (the module root), is under internal/ and carries the
// helper marker, and the reason written after it (empty when none is given).
func hasHelperMarker(rel string) (reason string, marked bool) {
	if !strings.HasPrefix(rel, "internal/") {
		return "", false
	}
	files, _ := filepath.Glob(filepath.Join(rel, "*.go"))
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name) // #nosec G304 G703 -- a dev tool reading the module's own sources
		if err != nil {
			continue
		}
		for _, l := range strings.Split(string(b), "\n") {
			if rest, ok := strings.CutPrefix(strings.TrimSpace(l), helperMarker); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
				return strings.TrimSpace(rest), true
			}
		}
	}
	return "", false
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: covercheck cover.out")
		os.Exit(2)
	}
	f, err := os.Open(os.Args[1]) // #nosec G304 G703 -- a dev tool reading the profile named on its command line
	if err != nil {
		fmt.Fprintln(os.Stderr, "covercheck:", err)
		os.Exit(1)
	}
	defer f.Close()
	bad, skipped, err := check(f, hasHelperMarker)
	if err != nil {
		fmt.Fprintln(os.Stderr, "covercheck:", err)
		os.Exit(1)
	}
	for _, s := range skipped {
		fmt.Println(s)
	}
	for _, b := range bad {
		fmt.Fprintln(os.Stderr, b)
	}
	if len(bad) > 0 {
		os.Exit(1)
	}
}

// isExcluded reports whether the package at rel (relative to the module root)
// is outside the floor.
func isExcluded(rel string) bool {
	for _, e := range excluded {
		if rel == e || strings.HasPrefix(rel, e+"/") {
			return true
		}
	}
	return false
}

// block is one statement block of the profile, keyed by file and range.
type block struct {
	file, span string
}

// check returns one line per included package below the floor, sorted by
// package, each naming the package and its percentage, and one line per
// package skipped because helperOf reports it as test-support code, with its
// reason. A package marked without a reason is reported in bad instead and is
// not skipped.
func check(r io.Reader, helperOf func(pkg string) (reason string, marked bool)) (bad, skipped []string, err error) {
	type stats struct{ stmts, covered int }
	blocks := map[block]*stats{}
	var order []block
	reasons := map[string]string{}
	helper := map[string]bool{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}
		// file:startLine.startCol,endLine.endCol numStmts count
		colon := strings.LastIndex(line, ":")
		if colon < 0 {
			return nil, nil, fmt.Errorf("malformed profile line %q", line)
		}
		fields := strings.Fields(line[colon+1:])
		if len(fields) != 3 {
			return nil, nil, fmt.Errorf("malformed profile line %q", line)
		}
		stmts, err1 := strconv.Atoi(fields[1])
		count, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil {
			return nil, nil, fmt.Errorf("malformed profile line %q", line)
		}
		file := strings.TrimPrefix(line[:colon], modulePath+"/")
		pkg := path.Dir(file)
		if isExcluded(pkg) {
			continue
		}
		if _, ok := helper[pkg]; !ok {
			reason, marked := "", false
			if helperOf != nil {
				reason, marked = helperOf(pkg)
			}
			helper[pkg], reasons[pkg] = marked, strings.TrimSpace(reason)
		}
		if helper[pkg] {
			continue
		}
		b := block{file, fields[0]}
		st := blocks[b]
		if st == nil {
			st = &stats{stmts: stmts}
			blocks[b] = st
			order = append(order, b)
		}
		if count > 0 {
			st.covered = st.stmts
		}
	}
	if err := sc.Err(); err != nil {
		return nil, nil, err
	}
	total := map[string]int{}
	covered := map[string]int{}
	for _, b := range order {
		pkg := path.Dir(b.file)
		total[pkg] += blocks[b].stmts
		covered[pkg] += blocks[b].covered
	}
	for pkg, h := range helper {
		switch {
		case !h:
		case reasons[pkg] == "":
			bad = append(bad, fmt.Sprintf("covercheck: package %s is excluded by %s but gives no reason", pkg, helperMarker))
		default:
			skipped = append(skipped, fmt.Sprintf("covercheck: excluded %s (%s: %s)", pkg, helperMarker, reasons[pkg]))
		}
	}
	for pkg, n := range total {
		if n == 0 {
			continue
		}
		if pct := 100 * float64(covered[pkg]) / float64(n); pct < floor {
			bad = append(bad, fmt.Sprintf("covercheck: package %s is at %.1f%%, below the %.0f%% floor", pkg, pct, floor))
		}
	}
	sort.Strings(bad)
	sort.Strings(skipped)
	return bad, skipped, nil
}
