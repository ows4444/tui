// Command genbidi generates internal/bidi/tables.go, the Unicode Bidi_Class,
// bracket and mirroring tables used by the UAX #9 reordering, from a pinned
// version of the Unicode Character Database. It uses only the standard library
// and is a development tool: no library package imports it.
//
//	go run ./internal/tools/genbidi -o internal/bidi/tables.go   # fetch and write
//	go run ./internal/tools/genbidi -dir path/to/ucd -o ...       # read local files instead
//	go run ./internal/tools/genbidi -check -o internal/bidi/tables.go
//	go run ./internal/tools/genbidi -print-hashes                 # SHA-256 of the fetched files
//
// Every source file is verified against a pinned SHA-256 before use, so the
// output is reproducible and a changed upstream file cannot change it silently.
// To move to a newer Unicode version, change unicodeVersion and run
// -print-hashes to obtain the new pins; CI regenerates the tables and fails when
// the committed file differs, so a version change cannot be forgotten.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	unicodeVersion = "17.0.0"
	baseURL        = "https://www.unicode.org/Public/" + unicodeVersion + "/ucd/"
)

const (
	derivedBidiClass = "extracted/DerivedBidiClass.txt"
	bidiBrackets     = "BidiBrackets.txt"
	bidiMirroring    = "BidiMirroring.txt"
)

// The UCD files the tables are built from, with their pinned SHA-256.
var sources = []struct{ path, sha256 string }{
	{derivedBidiClass, "4867b4b7f0731ed1bfcd34cc6251211ff1542541fce0734b6fbda139ee80b3a4"},
	{bidiBrackets, "dadbaf38a0d0246e5b805bf8725cb81b7c621f93d030595635f5ba2c2f179428"},
	{bidiMirroring, "a2f16fb873ab4fcdf3221cb1a8a85a134ddd6ed03603181823ff5206af3741ce"},
}

func main() {
	out := flag.String("o", "internal/bidi/tables.go", "output file")
	dir := flag.String("dir", "", "read the UCD files from this directory (same layout as the Unicode server) instead of fetching them")
	check := flag.Bool("check", false, "fail if the output file differs from the generated tables, without writing")
	printHashes := flag.Bool("print-hashes", false, "print the SHA-256 of each source file and exit")
	flag.Parse()
	if err := run(*out, *dir, *check, *printHashes); err != nil {
		fmt.Fprintln(os.Stderr, "genbidi:", err)
		os.Exit(1)
	}
}

func run(out, dir string, check, printHashes bool) error {
	files := make(map[string][]byte, len(sources))
	for _, s := range sources {
		b, err := load(dir, s.path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		got := hex.EncodeToString(sum[:])
		if printHashes {
			fmt.Printf("%s  %s\n", got, s.path)
			continue
		}
		if got != s.sha256 {
			return fmt.Errorf("%s: SHA-256 is %s, pinned %s for Unicode %s; nothing written", s.path, got, s.sha256, unicodeVersion)
		}
		files[s.path] = b
	}
	if printHashes {
		return nil
	}
	src, err := generate(files)
	if err != nil {
		return err
	}
	return writeOrCheck(out, src, check)
}

// writeOrCheck writes src to out, or with check reports an error when out
// differs from src, without writing.
func writeOrCheck(out string, src []byte, check bool) error {
	if check {
		have, err := os.ReadFile(out) // #nosec G304 -- a dev tool reading the file named on its command line
		if err != nil {
			return err
		}
		if !bytes.Equal(have, src) {
			return fmt.Errorf("%s is out of date; run `go run ./internal/tools/genbidi -o %s` and commit it", out, out)
		}
		return nil
	}
	return os.WriteFile(out, src, 0o644) // #nosec G306 -- a committed Go source file
}

// load reads one source file from dir, or fetches it from unicode.org.
func load(dir, path string) ([]byte, error) {
	if dir != "" {
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(path))) // #nosec G304 -- a dev tool reading the directory named on its command line
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(baseURL + path) // #nosec G107 -- baseURL is a constant and path comes from the fixed sources table
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", baseURL+path, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// longNames maps the long Bidi_Class names of the @missing lines to the short
// names the explicit lines use.
var longNames = map[string]string{
	"Left_To_Right":       "L",
	"Right_To_Left":       "R",
	"Arabic_Letter":       "AL",
	"European_Terminator": "ET",
	"Boundary_Neutral":    "BN",
}

// shortNames are the Bidi_Class values a Go identifier of the same name exists
// for in package bidi.
var shortNames = map[string]bool{
	"L": true, "R": true, "AL": true, "EN": true, "ES": true, "ET": true, "AN": true, "CS": true,
	"NSM": true, "BN": true, "B": true, "S": true, "WS": true, "ON": true,
	"LRE": true, "LRO": true, "RLE": true, "RLO": true, "PDF": true, "LRI": true, "RLI": true, "FSI": true, "PDI": true,
}

// classRange is an inclusive rune range with its Bidi_Class.
type classRange struct {
	lo, hi rune
	class  string
}

// classes returns the runs of runes whose class is not L, sorted and
// non-overlapping. A rune not listed in an explicit line takes the class of the
// last "@missing" line covering it, which is how the file gives unassigned code
// points in right-to-left blocks R or AL and in the currency block ET.
func classes(data []byte) ([]classRange, error) {
	class := make([]string, 0x110000)
	for i := range class {
		class[i] = "L"
	}
	var explicit []classRange
	for n, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(line, "# @missing:"); ok {
			rng, val, ok := strings.Cut(rest, ";")
			if !ok {
				return nil, fmt.Errorf("%s:%d: bad @missing line", derivedBidiClass, n+1)
			}
			lo, hi, err := parseRange(strings.TrimSpace(rng))
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", derivedBidiClass, n+1, err)
			}
			short, ok := longNames[strings.TrimSpace(val)]
			if !ok {
				return nil, fmt.Errorf("%s:%d: unknown class %q", derivedBidiClass, n+1, strings.TrimSpace(val))
			}
			for r := lo; r <= hi; r++ {
				class[r] = short
			}
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		rng, val, ok := strings.Cut(line, ";")
		if !ok {
			return nil, fmt.Errorf("%s:%d: no ';' in %q", derivedBidiClass, n+1, line)
		}
		lo, hi, err := parseRange(strings.TrimSpace(rng))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %v", derivedBidiClass, n+1, err)
		}
		val = strings.TrimSpace(val)
		if !shortNames[val] {
			return nil, fmt.Errorf("%s:%d: unknown class %q", derivedBidiClass, n+1, val)
		}
		explicit = append(explicit, classRange{lo, hi, val})
	}
	for _, e := range explicit { // explicit lines win over every default
		for r := e.lo; r <= e.hi; r++ {
			class[r] = e.class
		}
	}
	var out []classRange
	for r := rune(0); r < 0x110000; r++ {
		c := class[r]
		if c == "L" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].hi == r-1 && out[n-1].class == c {
			out[n-1].hi = r
			continue
		}
		out = append(out, classRange{r, r, c})
	}
	return out, nil
}

// bracket is one line of BidiBrackets.txt.
type bracket struct {
	r, pair rune
	kind    byte // 'o' opening, 'c' closing
}

func brackets(data []byte) ([]bracket, error) {
	var out []bracket
	for n, line := range strings.Split(string(data), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Split(line, ";")
		if len(f) != 3 {
			return nil, fmt.Errorf("%s:%d: want 3 fields in %q", bidiBrackets, n+1, line)
		}
		r, err1 := strconv.ParseUint(strings.TrimSpace(f[0]), 16, 32)
		p, err2 := strconv.ParseUint(strings.TrimSpace(f[1]), 16, 32)
		k := strings.TrimSpace(f[2])
		if err1 != nil || err2 != nil || (k != "o" && k != "c") {
			return nil, fmt.Errorf("%s:%d: bad line %q", bidiBrackets, n+1, line)
		}
		out = append(out, bracket{rune(r), rune(p), k[0]}) // #nosec G115 -- parsed with bitSize 32
	}
	sort.Slice(out, func(i, j int) bool { return out[i].r < out[j].r })
	return out, nil
}

// mirror is an exact Bidi_Mirroring_Glyph pair; best-fit pairs are left out, as
// a substituted glyph that is not the mirror image would mislead.
type mirror struct{ r, glyph rune }

func mirrors(data []byte) ([]mirror, error) {
	var out []mirror
	for n, line := range strings.Split(string(data), "\n") {
		code, comment, _ := strings.Cut(line, "#")
		code = strings.TrimSpace(code)
		if code == "" || strings.Contains(comment, "[BEST FIT]") {
			continue
		}
		f := strings.Split(code, ";")
		if len(f) != 2 {
			return nil, fmt.Errorf("%s:%d: want 2 fields in %q", bidiMirroring, n+1, line)
		}
		r, err1 := strconv.ParseUint(strings.TrimSpace(f[0]), 16, 32)
		g, err2 := strconv.ParseUint(strings.TrimSpace(f[1]), 16, 32)
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("%s:%d: bad line %q", bidiMirroring, n+1, line)
		}
		out = append(out, mirror{rune(r), rune(g)}) // #nosec G115 -- parsed with bitSize 32
	}
	sort.Slice(out, func(i, j int) bool { return out[i].r < out[j].r })
	return out, nil
}

func parseRange(s string) (lo, hi rune, err error) {
	from, to, isRange := strings.Cut(s, "..")
	l, err := strconv.ParseUint(from, 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("bad code point %q", s)
	}
	h := l
	if isRange {
		if h, err = strconv.ParseUint(to, 16, 32); err != nil {
			return 0, 0, fmt.Errorf("bad code point %q", s)
		}
	}
	if h < l || h > 0x10FFFF {
		return 0, 0, fmt.Errorf("bad range %q", s)
	}
	return rune(l), rune(h), nil // #nosec G115 -- h is checked against 0x10FFFF and l <= h above
}

// generate builds the Go source for the tables from the three UCD files.
func generate(files map[string][]byte) ([]byte, error) {
	cls, err := classes(files[derivedBidiClass])
	if err != nil {
		return nil, err
	}
	br, err := brackets(files[bidiBrackets])
	if err != nil {
		return nil, err
	}
	mi, err := mirrors(files[bidiMirroring])
	if err != nil {
		return nil, err
	}
	if len(cls) == 0 || len(br) == 0 || len(mi) == 0 {
		return nil, errors.New("the source files produced an empty table; their format may have changed")
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by internal/tools/genbidi; DO NOT EDIT.\n//\n")
	fmt.Fprintf(&b, "// Unicode version: %s\n// Sources (SHA-256 verified):\n", unicodeVersion)
	for _, s := range sources {
		fmt.Fprintf(&b, "//   %s%s\n//     %s\n", baseURL, s.path, s.sha256)
	}
	fmt.Fprintf(&b, "\npackage bidi\n\n")
	fmt.Fprintf(&b, "// UnicodeVersion is the Unicode version the generated tables come from.\n")
	fmt.Fprintf(&b, "const UnicodeVersion = %q\n\n", unicodeVersion)
	fmt.Fprintf(&b, "// classRanges lists every rune whose Bidi_Class is not L, sorted and\n// non-overlapping; a rune in no range is L.\n")
	fmt.Fprintf(&b, "var classRanges = []classRange{\n")
	for _, r := range cls {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, %s},\n", r.lo, r.hi, r.class)
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "// bracketTable lists the paired brackets (Bidi_Paired_Bracket_Type), sorted by\n// rune: pair is the other bracket of the pair, kind 'o' or 'c'.\n")
	fmt.Fprintf(&b, "var bracketTable = []bracketEntry{\n")
	for _, r := range br {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, '%c'},\n", r.r, r.pair, r.kind)
	}
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "// mirrorTable lists the exact Bidi_Mirroring_Glyph pairs, sorted by rune.\n")
	fmt.Fprintf(&b, "var mirrorTable = []mirrorEntry{\n")
	for _, r := range mi {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X},\n", r.r, r.glyph)
	}
	fmt.Fprintf(&b, "}\n")
	return format.Source(b.Bytes())
}
