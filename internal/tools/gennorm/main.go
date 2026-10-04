// Command gennorm generates avatar/nfc_tables.go, the data for Unicode
// Normalization Form C (UAX #15), and avatar/testdata/nfc_vectors.txt, a
// sample of the Unicode conformance test, from a pinned version of the Unicode
// Character Database. It uses only the standard library and is a development
// tool: no library package imports it. It follows internal/tools/gengrapheme.
//
//	go run ./internal/tools/gennorm                      # fetch and write both files
//	go run ./internal/tools/gennorm -dir path/to/ucd     # read local files instead
//	go run ./internal/tools/gennorm -check               # fail if either file is stale
//	go run ./internal/tools/gennorm -print-hashes        # SHA-256 of the fetched files
//
// Three tables are written: the canonical combining class of every rune that
// has one, every canonical decomposition, and the primary composites (the
// decompositions NFC may put back together). Hangul syllables are composed
// and decomposed by arithmetic and are in no table. Every source file is
// verified against a pinned SHA-256 before use, so the output is reproducible
// and a changed upstream file cannot change it silently. To move to a newer
// Unicode version, change unicodeVersion and run -print-hashes to obtain the
// new pins.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

	unicodeData = "UnicodeData.txt"
	normProps   = "DerivedNormalizationProps.txt"
	normTest    = "NormalizationTest.txt"
)

// The UCD files the outputs are built from, with their pinned SHA-256.
var sources = []struct{ path, sha256 string }{
	{unicodeData, "2e1efc1dcb59c575eedf5ccae60f95229f706ee6d031835247d843c11d96470c"},
	{normProps, "71fd6a206a2c0cdd41feb6b7f656aa31091db45e9cedc926985d718397f9e488"},
	{normTest, "5019ffd530751a741900c849c0e010332f142a3612234639bd200b82138a87db"},
}

func main() {
	out := flag.String("o", "avatar/nfc_tables.go", "output file for the tables")
	vectors := flag.String("vectors", "avatar/testdata/nfc_vectors.txt", "output file for the sampled conformance test")
	dir := flag.String("dir", "", "read the UCD files from this directory (same layout as the Unicode server) instead of fetching them")
	check := flag.Bool("check", false, "fail if an output file differs from what would be generated, without writing")
	printHashes := flag.Bool("print-hashes", false, "print the SHA-256 of each source file and exit")
	flag.Parse()

	if err := run(*out, *vectors, *dir, *check, *printHashes); err != nil {
		fmt.Fprintln(os.Stderr, "gennorm:", err)
		os.Exit(1)
	}
}

func run(out, vectors, dir string, check, printHashes bool) error {
	files := make(map[string][]byte, len(sources))
	for _, s := range sources {
		b, err := load(dir, s.path)
		if err != nil {
			return err
		}
		got := sha(b)
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
	tables, err := generate(files[unicodeData], files[normProps])
	if err != nil {
		return err
	}
	sample, err := sampleTest(files[normTest])
	if err != nil {
		return err
	}
	for _, f := range []struct {
		path string
		data []byte
	}{{out, tables}, {vectors, sample}} {
		if check {
			have, err := os.ReadFile(f.path) // #nosec G304 -- a dev tool reading the file named on its command line
			if err != nil {
				return err
			}
			if !bytes.Equal(have, f.data) {
				return fmt.Errorf("%s is out of date; run `go run ./internal/tools/gennorm` and commit it", f.path)
			}
			continue
		}
		if err := os.WriteFile(f.path, f.data, 0o644); err != nil { // #nosec G306 -- a committed source or test data file
			return err
		}
	}
	return nil
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

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

func parseRune(s string) (rune, error) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 16, 32)
	if err != nil || v > 0x10FFFF {
		return 0, fmt.Errorf("bad code point %q", s)
	}
	return rune(v), nil // #nosec G115 -- v is at most 0x10FFFF
}

// lines calls fn with the fields of each data line of a UCD file: the text
// before any "#", split at ";" and trimmed. Blank lines are skipped.
func lines(data []byte, name string, fn func(fields []string) error) error {
	for n, line := range strings.Split(string(data), "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, ";")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		if err := fn(fields); err != nil {
			return fmt.Errorf("%s:%d: %w", name, n+1, err)
		}
	}
	return nil
}

// decomp is a canonical decomposition of r into a, or into a then b; b is 0
// for a singleton.
type decomp struct{ r, a, b rune }

// classRange is a run of consecutive runes with one combining class.
type classRange struct {
	lo, hi rune
	ccc    uint8
}

// generate builds the Go source of the tables from UnicodeData.txt and
// DerivedNormalizationProps.txt.
func generate(data, props []byte) ([]byte, error) {
	var classes []classRange
	var decomps []decomp
	err := lines(data, unicodeData, func(f []string) error {
		if len(f) < 6 {
			return fmt.Errorf("%d fields, want at least 6", len(f))
		}
		r, err := parseRune(f[0])
		if err != nil {
			return err
		}
		ccc, err := strconv.ParseUint(f[3], 10, 8)
		if err != nil {
			return fmt.Errorf("bad combining class %q", f[3])
		}
		if ccc != 0 {
			if n := len(classes); n > 0 && classes[n-1].hi == r-1 && classes[n-1].ccc == uint8(ccc) {
				classes[n-1].hi = r
			} else {
				classes = append(classes, classRange{r, r, uint8(ccc)})
			}
		}
		// A decomposition that starts with a "<tag>" is a compatibility one,
		// which NFC does not use.
		if f[5] == "" || strings.HasPrefix(f[5], "<") {
			return nil
		}
		parts := strings.Fields(f[5])
		if len(parts) > 2 {
			return fmt.Errorf("canonical decomposition of %d runes", len(parts))
		}
		d := decomp{r: r}
		if d.a, err = parseRune(parts[0]); err != nil {
			return err
		}
		if len(parts) == 2 {
			if d.b, err = parseRune(parts[1]); err != nil {
				return err
			}
		}
		decomps = append(decomps, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(classes) == 0 || len(decomps) == 0 {
		return nil, fmt.Errorf("%s: no combining classes or no decompositions found", unicodeData)
	}

	excluded := map[rune]bool{}
	err = lines(props, normProps, func(f []string) error {
		if len(f) < 2 || f[1] != "Full_Composition_Exclusion" {
			return nil
		}
		lo, hi, ok := strings.Cut(f[0], "..")
		a, err := parseRune(lo)
		if err != nil {
			return err
		}
		b := a
		if ok {
			if b, err = parseRune(hi); err != nil {
				return err
			}
		}
		for r := a; r <= b; r++ {
			excluded[r] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(excluded) == 0 {
		return nil, fmt.Errorf("%s: no Full_Composition_Exclusion found", normProps)
	}

	sort.Slice(decomps, func(i, j int) bool { return decomps[i].r < decomps[j].r })
	var composites []decomp
	for _, d := range decomps {
		if d.b != 0 && !excluded[d.r] {
			composites = append(composites, d)
		}
	}
	sort.Slice(composites, func(i, j int) bool {
		if composites[i].a != composites[j].a {
			return composites[i].a < composites[j].a
		}
		return composites[i].b < composites[j].b
	})
	for i := 1; i < len(composites); i++ {
		if composites[i].a == composites[i-1].a && composites[i].b == composites[i-1].b {
			return nil, fmt.Errorf("U+%04X and U+%04X compose from the same pair", composites[i-1].r, composites[i].r)
		}
	}

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by tools/gennorm; DO NOT EDIT.\n\npackage avatar\n\n")
	fmt.Fprintf(&b, "// The tables below are derived from the Unicode Character Database, version\n// %s (%s and %s).\n\n", unicodeVersion, unicodeData, normProps)
	fmt.Fprintf(&b, "// nfcClasses holds the canonical combining class of every rune whose class\n// is not zero, as sorted runs.\nvar nfcClasses = [...]struct {\n\tlo, hi rune\n\tccc    uint8\n}{\n")
	for _, c := range classes {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, %d},\n", c.lo, c.hi, c.ccc)
	}
	fmt.Fprintf(&b, "}\n\n// nfcDecomps holds every canonical decomposition, sorted by rune: the rune,\n// then the one or two runes it decomposes to (the second is 0 for one).\nvar nfcDecomps = [...][3]rune{\n")
	for _, d := range decomps {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, 0x%04X},\n", d.r, d.a, d.b)
	}
	fmt.Fprintf(&b, "}\n\n// nfcComposites holds the primary composites, sorted by pair: the two runes\n// that compose, then the rune they compose to.\nvar nfcComposites = [...][3]rune{\n")
	for _, d := range composites {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, 0x%04X},\n", d.a, d.b, d.r)
	}
	fmt.Fprintf(&b, "}\n")
	return format.Source(b.Bytes())
}

// Part 1 of the conformance test has a line for every rune that
// normalization changes, most of them Hangul syllables, and part 2 is long
// too; one line in sampleEvery of each is kept. The other parts are short and
// kept whole.
const sampleEvery = 8

// sampleTest reduces NormalizationTest.txt to a sample: the five columns of
// each kept line, without comments, under its part heading.
func sampleTest(data []byte) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# A sample of NormalizationTest.txt, Unicode %s. Generated by tools/gennorm; do not edit.\n", unicodeVersion)
	fmt.Fprintf(&b, "# Columns: source; NFC; NFD; NFKC; NFKD. One line in %d of parts 1 and 2 is kept.\n", sampleEvery)
	part, n, kept := "", 0, 0
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "@Part") {
			part, n = strings.Fields(line)[0], 0
			fmt.Fprintln(&b, part)
			continue
		}
		if j := strings.IndexByte(line, '#'); j >= 0 {
			line = line[:j]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cols := strings.Split(strings.TrimSuffix(line, ";"), ";")
		if len(cols) != 5 || part == "" {
			return nil, fmt.Errorf("%s:%d: %d columns in part %q, want 5", normTest, i+1, len(cols), part)
		}
		n++
		if (part == "@Part1" || part == "@Part2") && n%sampleEvery != 1 {
			continue
		}
		kept++
		fmt.Fprintln(&b, strings.Join(cols, ";"))
	}
	if kept == 0 {
		return nil, fmt.Errorf("%s: no test lines found", normTest)
	}
	return b.Bytes(), nil
}
