// Command gengrapheme generates ansi/graphemebreak_tables.go, the extended
// grapheme cluster break data (UAX #29), from a pinned version of the Unicode
// Character Database. It uses only the standard library and is a development
// tool: no library package imports it. It follows internal/tools/genwidth.
//
//	go run ./internal/tools/gengrapheme -o ansi/graphemebreak_tables.go   # fetch and write
//	go run ./internal/tools/gengrapheme -dir path/to/ucd -o ...           # read local files instead
//	go run ./internal/tools/gengrapheme -check -o ansi/graphemebreak_tables.go
//	go run ./internal/tools/gengrapheme -print-hashes                     # SHA-256 of the fetched files
//
// Three tables are written: the Grapheme_Cluster_Break property, the
// Extended_Pictographic emoji property (rule GB11) and the Indic_Conjunct_Break
// property (rule GB9c). Every source file is verified against a pinned SHA-256
// before use, so the output is reproducible and a changed upstream file cannot
// change it silently. To move to a newer Unicode version, change
// unicodeVersion and run -print-hashes to obtain the new pins.
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

// The UCD files the tables are built from, with their pinned SHA-256.
var sources = []struct{ path, sha256 string }{
	{"auxiliary/GraphemeBreakProperty.txt", "d6b51d1d2ae5c33b451b7ed994b48f1f4dc62b2272a5831e7fd418514a6bae89"},
	{"emoji/emoji-data.txt", "2cb2bb9455cda83e8481541ecf5b6dfda66a3bb89efa3fa7c5297eccf607b72b"},
	{"DerivedCoreProperties.txt", "24c7fed1195c482faaefd5c1e7eb821c5ee1fb6de07ecdbaa64b56a99da22c08"},
}

const (
	graphemeBreak = "auxiliary/GraphemeBreakProperty.txt"
	emojiData     = "emoji/emoji-data.txt"
	derivedCore   = "DerivedCoreProperties.txt"
)

// gbNames maps a Grapheme_Cluster_Break value to its Go constant in package
// ansi. A value not listed here makes generation fail, so a new value in a
// newer UCD is noticed instead of dropped.
var gbNames = map[string]string{
	"CR": "gbCR", "LF": "gbLF", "Control": "gbControl", "Extend": "gbExtend",
	"ZWJ": "gbZWJ", "Regional_Indicator": "gbRegionalIndicator", "Prepend": "gbPrepend",
	"SpacingMark": "gbSpacingMark", "L": "gbL", "V": "gbV", "T": "gbT", "LV": "gbLV", "LVT": "gbLVT",
}

// incbNames maps an Indic_Conjunct_Break value (other than None) to its Go
// constant.
var incbNames = map[string]string{
	"Consonant": "incbConsonant", "Linker": "incbLinker", "Extend": "incbExtend",
}

func main() {
	out := flag.String("o", "ansi/graphemebreak_tables.go", "output file")
	dir := flag.String("dir", "", "read the UCD files from this directory (same layout as the Unicode server) instead of fetching them")
	check := flag.Bool("check", false, "fail if the output file differs from the generated tables, without writing")
	printHashes := flag.Bool("print-hashes", false, "print the SHA-256 of each source file and exit")
	flag.Parse()

	if err := run(*out, *dir, *check, *printHashes); err != nil {
		fmt.Fprintln(os.Stderr, "gengrapheme:", err)
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
	if check {
		have, err := os.ReadFile(out) // #nosec G304 -- a dev tool reading the file named on its command line
		if err != nil {
			return err
		}
		if !bytes.Equal(have, src) {
			return fmt.Errorf("%s is out of date; run `go run ./internal/tools/gengrapheme -o %s` and commit it", out, out)
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

// entry is one line of a UCD property file: an inclusive range and its
// semicolon-separated fields after the range, trimmed.
type entry struct {
	lo, hi rune
	fields []string
}

// parse reads the lines of a UCD property file: `range ; field [; field] #
// comment`, where range is `XXXX` or `XXXX..YYYY`. Comment and blank lines are
// skipped.
func parse(data []byte, name string) ([]entry, error) {
	var out []entry
	for n, line := range strings.Split(string(data), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ";")
		if len(parts) < 2 {
			return nil, fmt.Errorf("%s:%d: no ';' in %q", name, n+1, line)
		}
		lo, hi, err := parseRange(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %v", name, n+1, err)
		}
		fields := make([]string, len(parts)-1)
		for i, p := range parts[1:] {
			fields[i] = strings.TrimSpace(p)
		}
		out = append(out, entry{lo, hi, fields})
	}
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

// span is one inclusive rune range with a value: a Go constant name, or empty
// for a plain range.
type span struct {
	lo, hi rune
	name   string
}

// merge sorts spans, checks they do not overlap, and joins neighbours that
// carry the same name.
func merge(in []span, table string) ([]span, error) {
	sort.Slice(in, func(i, j int) bool { return in[i].lo < in[j].lo })
	var out []span
	for _, s := range in {
		if n := len(out); n > 0 {
			last := &out[n-1]
			if s.lo <= last.hi {
				return nil, fmt.Errorf("%s: ranges %04X..%04X and %04X..%04X overlap", table, last.lo, last.hi, s.lo, s.hi)
			}
			if s.lo == last.hi+1 && s.name == last.name {
				last.hi = s.hi
				continue
			}
		}
		out = append(out, s)
	}
	return out, nil
}

// generate builds the Go source for the tables from the three UCD files.
func generate(files map[string][]byte) ([]byte, error) {
	gb, err := parse(files[graphemeBreak], graphemeBreak)
	if err != nil {
		return nil, err
	}
	var gbSpans []span
	for _, e := range gb {
		name, ok := gbNames[e.fields[0]]
		if !ok {
			return nil, fmt.Errorf("%s: unknown Grapheme_Cluster_Break value %q", graphemeBreak, e.fields[0])
		}
		gbSpans = append(gbSpans, span{e.lo, e.hi, name})
	}
	emoji, err := parse(files[emojiData], emojiData)
	if err != nil {
		return nil, err
	}
	var pict []span
	for _, e := range emoji {
		if e.fields[0] == "Extended_Pictographic" {
			pict = append(pict, span{e.lo, e.hi, ""})
		}
	}
	core, err := parse(files[derivedCore], derivedCore)
	if err != nil {
		return nil, err
	}
	var incb []span
	for _, e := range core {
		if len(e.fields) < 2 || e.fields[0] != "InCB" {
			continue
		}
		name, ok := incbNames[e.fields[1]]
		if !ok {
			return nil, fmt.Errorf("%s: unknown InCB value %q", derivedCore, e.fields[1])
		}
		incb = append(incb, span{e.lo, e.hi, name})
	}
	if len(gbSpans) == 0 || len(pict) == 0 || len(incb) == 0 {
		return nil, errors.New("a source file produced an empty table; its format may have changed")
	}
	gbSpans, err = merge(gbSpans, "Grapheme_Cluster_Break")
	if err != nil {
		return nil, err
	}
	pict, err = merge(pict, "Extended_Pictographic")
	if err != nil {
		return nil, err
	}
	incb, err = merge(incb, "Indic_Conjunct_Break")
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by tools/gengrapheme; DO NOT EDIT.\n//\n")
	fmt.Fprintf(&b, "// Unicode version: %s\n// Sources (SHA-256 verified):\n", unicodeVersion)
	for _, s := range sources {
		fmt.Fprintf(&b, "//   %s%s\n//     %s\n", baseURL, s.path, s.sha256)
	}
	fmt.Fprintf(&b, "\npackage ansi\n\n")
	table := func(doc, name, typ string, spans []span, withValue bool) {
		fmt.Fprintf(&b, "%s\nvar %s = []%s{\n", doc, name, typ)
		for _, s := range spans {
			if withValue {
				fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, %s},\n", s.lo, s.hi, s.name)
			} else {
				fmt.Fprintf(&b, "\t{0x%04X, 0x%04X},\n", s.lo, s.hi)
			}
		}
		fmt.Fprintf(&b, "}\n\n")
	}
	table("// generatedGraphemeBreak lists every rune with a Grapheme_Cluster_Break value\n// other than Other, sorted and non-overlapping. A rune in no range is Other.",
		"generatedGraphemeBreak", "gbRange", gbSpans, true)
	table("// generatedExtPict lists every Extended_Pictographic rune (UAX #29 rule GB11),\n// sorted and non-overlapping.",
		"generatedExtPict", "runeRange", pict, false)
	table("// generatedIndicConjunctBreak lists every rune whose Indic_Conjunct_Break is\n// Consonant, Linker or Extend (UAX #29 rule GB9c), sorted and non-overlapping.\n// A rune in no range has the value None.",
		"generatedIndicConjunctBreak", "incbRange", incb, true)
	return format.Source(b.Bytes())
}
