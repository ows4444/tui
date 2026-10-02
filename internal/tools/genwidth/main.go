// Command genwidth generates ansi/runewidth_tables.go, the terminal-column
// width tables, from a pinned version of the Unicode Character Database. It
// uses only the standard library and is a development tool: no library package
// imports it.
//
//	go run ./internal/tools/genwidth -o ansi/runewidth_tables.go   # fetch and write
//	go run ./internal/tools/genwidth -dir path/to/ucd -o ...        # read local files instead
//	go run ./internal/tools/genwidth -check -o ansi/runewidth_tables.go
//	go run ./internal/tools/genwidth -print-hashes                  # SHA-256 of the fetched files
//
// The tables classify a rune as zero width (general category Mn, Me or Cf),
// wide (East_Asian_Width W or F, or Emoji_Presentation) or, by omission,
// narrow. Every source file is verified against a pinned SHA-256 before use,
// so the output is reproducible and a changed upstream file cannot change it
// silently. To move to a newer Unicode version, change unicodeVersion and run
// -print-hashes to obtain the new pins.
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
	{"EastAsianWidth.txt", "ea7ce50f3444a050333448dffef1cadd9325af55cbb764b4a2280faf52170a33"},
	{"emoji/emoji-data.txt", "2cb2bb9455cda83e8481541ecf5b6dfda66a3bb89efa3fa7c5297eccf607b72b"},
	{"extracted/DerivedGeneralCategory.txt", "d62e5bab70ca74f099343f71224fa051cb1fdd61a1ab45c0488c44cfc0b6102e"},
}

const (
	eastAsianWidth  = "EastAsianWidth.txt"
	emojiData       = "emoji/emoji-data.txt"
	generalCategory = "extracted/DerivedGeneralCategory.txt"
)

func main() {
	out := flag.String("o", "ansi/runewidth_tables.go", "output file")
	dir := flag.String("dir", "", "read the UCD files from this directory (same layout as the Unicode server) instead of fetching them")
	check := flag.Bool("check", false, "fail if the output file differs from the generated tables, without writing")
	printHashes := flag.Bool("print-hashes", false, "print the SHA-256 of each source file and exit")
	flag.Parse()

	if err := run(*out, *dir, *check, *printHashes); err != nil {
		fmt.Fprintln(os.Stderr, "genwidth:", err)
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
			return fmt.Errorf("%s is out of date; run `go generate ./ansi` and commit it", out)
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

// span is an inclusive rune range.
type span struct{ lo, hi rune }

// generate builds the Go source for the tables from the three UCD files.
func generate(files map[string][]byte) ([]byte, error) {
	var wide, zero []span

	eaw, err := parse(files[eastAsianWidth], eastAsianWidth)
	if err != nil {
		return nil, err
	}
	for _, e := range eaw {
		if e.value == "W" || e.value == "F" {
			wide = append(wide, e.span)
		}
	}
	emoji, err := parse(files[emojiData], emojiData)
	if err != nil {
		return nil, err
	}
	for _, e := range emoji {
		if e.value == "Emoji_Presentation" {
			wide = append(wide, e.span)
		}
	}
	gc, err := parse(files[generalCategory], generalCategory)
	if err != nil {
		return nil, err
	}
	for _, e := range gc {
		if e.value == "Mn" || e.value == "Me" || e.value == "Cf" {
			zero = append(zero, e.span)
		}
	}
	if len(wide) == 0 || len(zero) == 0 {
		return nil, errors.New("the source files produced an empty table; their format may have changed")
	}
	ranges := classify(wide, zero)

	var b bytes.Buffer
	fmt.Fprintf(&b, "// Code generated by tools/genwidth; DO NOT EDIT.\n//\n")
	fmt.Fprintf(&b, "// Unicode version: %s\n// Sources (SHA-256 verified):\n", unicodeVersion)
	for _, s := range sources {
		fmt.Fprintf(&b, "//   %s%s\n//     %s\n", baseURL, s.path, s.sha256)
	}
	fmt.Fprintf(&b, "\npackage ansi\n\n")
	fmt.Fprintf(&b, "// unicodeVersion is the Unicode version the generated width tables come from.\n")
	fmt.Fprintf(&b, "const unicodeVersion = %q\n\n", unicodeVersion)
	fmt.Fprintf(&b, "// generatedWidthRanges lists every rune that does not take exactly one column,\n")
	fmt.Fprintf(&b, "// sorted and non-overlapping, with its width: 0 for general category Mn, Me or\n")
	fmt.Fprintf(&b, "// Cf, 2 for East_Asian_Width W or F or Emoji_Presentation. Where both apply\n")
	fmt.Fprintf(&b, "// (a nonspacing mark that is also East Asian Wide), the width is 0. A rune in\n")
	fmt.Fprintf(&b, "// no range is one column wide.\n")
	fmt.Fprintf(&b, "var generatedWidthRanges = []widthRange{\n")
	for _, r := range ranges {
		fmt.Fprintf(&b, "\t{0x%04X, 0x%04X, %d},\n", r.lo, r.hi, r.w)
	}
	fmt.Fprintf(&b, "}\n")
	return format.Source(b.Bytes())
}

// wspan is an inclusive rune range with a column width.
type wspan struct {
	lo, hi rune
	w      uint8
}

// classify gives every rune its width (zero wins over wide) and returns the
// runs of runes whose width is not one, sorted and non-overlapping. Adjacent
// runes of the same width are joined; adjacent runes of different widths stay
// separate ranges.
func classify(wide, zero []span) []wspan {
	width := make([]uint8, 0x110000)
	for i := range width {
		width[i] = 1
	}
	for _, s := range wide {
		for r := s.lo; r <= s.hi; r++ {
			width[r] = 2
		}
	}
	for _, s := range zero {
		for r := s.lo; r <= s.hi; r++ {
			width[r] = 0
		}
	}
	var out []wspan
	for r := rune(0); r < 0x110000; r++ {
		w := width[r]
		if w == 1 {
			continue
		}
		if n := len(out); n > 0 && out[n-1].hi == r-1 && out[n-1].w == w {
			out[n-1].hi = r
			continue
		}
		out = append(out, wspan{r, r, w})
	}
	return out
}

type entry struct {
	span  span
	value string
}

// parse reads the lines of a UCD property file: `range ; value # comment`,
// where range is `XXXX` or `XXXX..YYYY`. Comment and blank lines are skipped.
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
		fields := strings.Split(line, ";")
		if len(fields) < 2 {
			return nil, fmt.Errorf("%s:%d: no ';' in %q", name, n+1, line)
		}
		lo, hi, err := parseRange(strings.TrimSpace(fields[0]))
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %v", name, n+1, err)
		}
		out = append(out, entry{span{lo, hi}, strings.TrimSpace(fields[1])})
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
