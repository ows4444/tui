// Command benchgate compares two `go test -bench -benchmem` outputs and fails
// when allocations or bytes per operation grew past a threshold.
//
// It gates on allocs/op and B/op only: those are the same on any machine, so a
// baseline recorded on a laptop is valid on a CI runner. ns/op is machine
// dependent, so it is not gated. Standard library only.
//
//	benchgate -base bench/baseline.txt -new new.txt
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// sample holds the per-run measurements of one benchmark.
type sample struct{ allocs, bytes []float64 }

const (
	bytesSlack = 64 // B/op noise from map growth and pooled buffers
)

func main() {
	basePath := flag.String("base", "bench/baseline.txt", "baseline benchmark output")
	newPath := flag.String("new", "", "new benchmark output")
	pct := flag.Float64("max-pct", 10, "allowed growth in allocs/op and B/op, in percent")
	flag.Parse()
	if *newPath == "" {
		fmt.Fprintln(os.Stderr, "benchgate: -new is required")
		os.Exit(2)
	}
	base, err := load(*basePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchgate:", err)
		os.Exit(2)
	}
	cur, err := load(*newPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "benchgate:", err)
		os.Exit(2)
	}
	problems, notes := compare(base, cur, *pct)
	for _, n := range notes {
		fmt.Println("note:", n)
	}
	for _, p := range problems {
		fmt.Println("FAIL:", p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
	fmt.Printf("benchgate: %d benchmarks within %.0f%% of the baseline\n", len(base), *pct)
}

func load(path string) (map[string]sample, error) {
	f, err := os.Open(path) // #nosec G304 -- a dev tool reading the files named on its command line
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("%s: no benchmark results (was -benchmem set?)", path)
	}
	return m, nil
}

// parse reads benchmark lines, keyed by "package.Name" with the -N GOMAXPROCS
// suffix removed. Lines without B/op and allocs/op are ignored.
func parse(r io.Reader) (map[string]sample, error) {
	out := map[string]sample{}
	pkg := ""
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 2 && f[0] == "pkg:" {
			pkg = f[1]
			continue
		}
		if len(f) < 8 || !strings.HasPrefix(f[0], "Benchmark") {
			continue
		}
		var bytes, allocs float64
		var haveB, haveA bool
		for i := 2; i+1 < len(f); i += 2 {
			v, err := strconv.ParseFloat(f[i], 64)
			if err != nil {
				continue
			}
			switch f[i+1] {
			case "B/op":
				bytes, haveB = v, true
			case "allocs/op":
				allocs, haveA = v, true
			}
		}
		if !haveB || !haveA {
			continue
		}
		name := f[0]
		if i := strings.LastIndex(name, "-"); i > 0 {
			if _, err := strconv.Atoi(name[i+1:]); err == nil {
				name = name[:i]
			}
		}
		key := pkg + "." + name
		s := out[key]
		s.allocs = append(s.allocs, allocs)
		s.bytes = append(s.bytes, bytes)
		out[key] = s
	}
	return out, sc.Err()
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// grew reports whether cur exceeds base by more than pct percent plus slack.
func grew(base, cur, pct, slack float64) bool {
	return cur > base*(1+pct/100)+slack
}

// compare returns the regressions and informational notes, both sorted.
func compare(base, cur map[string]sample, pct float64) (problems, notes []string) {
	for name, b := range base {
		c, ok := cur[name]
		if !ok {
			problems = append(problems, fmt.Sprintf("%s is in the baseline but was not run", name))
			continue
		}
		ba, ca := median(b.allocs), median(c.allocs)
		bb, cb := median(b.bytes), median(c.bytes)
		if grew(ba, ca, pct, 0) {
			problems = append(problems, fmt.Sprintf("%s allocs/op %.0f -> %.0f", name, ba, ca))
		}
		if grew(bb, cb, pct, bytesSlack) {
			problems = append(problems, fmt.Sprintf("%s B/op %.0f -> %.0f", name, bb, cb))
		}
	}
	for name := range cur {
		if _, ok := base[name]; !ok {
			notes = append(notes, name+" is new; add it to the baseline")
		}
	}
	sort.Strings(problems)
	sort.Strings(notes)
	return problems, notes
}
