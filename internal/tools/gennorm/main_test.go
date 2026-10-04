package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	fakeData = "0041;LATIN CAPITAL LETTER A;Lu;0;L;;;;;N;;;;0061;\n" +
		"00C5;LATIN CAPITAL LETTER A WITH RING ABOVE;Lu;0;L;0041 030A;;;;N;;;;00E5;\n" +
		"00B5;MICRO SIGN;Ll;0;L;<compat> 03BC;;;;N;;;;;\n" +
		"0300;COMBINING GRAVE ACCENT;Mn;230;NSM;;;;;N;;;;;\n" +
		"0301;COMBINING ACUTE ACCENT;Mn;230;NSM;;;;;N;;;;;\n" +
		"030A;COMBINING RING ABOVE;Mn;230;NSM;;;;;N;;;;;\n" +
		"0323;COMBINING DOT BELOW;Mn;220;NSM;;;;;N;;;;;\n" +
		"0958;DEVANAGARI LETTER QA;Lo;0;L;0915 093C;;;;N;;;;;\n" +
		"212B;ANGSTROM SIGN;Lu;0;L;00C5;;;;N;;;;00E5;\n"
	fakeProps = "# comment\n0958..095F  ; Full_Composition_Exclusion # Lo\n212B ; Full_Composition_Exclusion\n00C5 ; NFD_QC; N\n"
	fakeTest  = "# header\n@Part0 # Specific cases\n1E0A;1E0A;0044 0307;1E0A;0044 0307; # comment\n" +
		"@Part1 # Character by character test\n" +
		"00C0;00C0;0041 0300;00C0;0041 0300; # one\n00C1;00C1;0041 0301;00C1;0041 0301; # two\n"
)

func TestGenerateBuildsTheThreeTables(t *testing.T) {
	a, err := generate([]byte(fakeData), []byte(fakeProps))
	if err != nil {
		t.Fatal(err)
	}
	b, err := generate([]byte(fakeData), []byte(fakeProps))
	if err != nil || !bytes.Equal(a, b) {
		t.Fatalf("two runs differ (err %v)", err)
	}
	src := string(a)
	for _, want := range []string{
		"{0x0300, 0x0301, 230},",    // consecutive runes of one class are merged
		"{0x030A, 0x030A, 230},",    // a gap starts a new run
		"{0x0323, 0x0323, 220},",    // another class
		"{0x00C5, 0x0041, 0x030A},", // a decomposition
		"{0x212B, 0x00C5, 0x0000},", // a singleton
		"{0x0958, 0x0915, 0x093C},", // an excluded pair still decomposes
		"{0x0041, 0x030A, 0x00C5},", // and the one primary composite
		"version\n// " + unicodeVersion,
	} {
		if !strings.Contains(src, want) {
			t.Errorf("output lacks %q:\n%s", want, src)
		}
	}
	if strings.Contains(src, "0x03BC") {
		t.Error("a compatibility decomposition leaked into the tables")
	}
	if strings.Contains(src, "{0x0915, 0x093C, 0x0958}") {
		t.Error("an excluded composite is in the composites table")
	}
}

func TestGenerateRejectsBadInput(t *testing.T) {
	for name, in := range map[string][2]string{
		"no data":         {"", fakeProps},
		"no exclusions":   {fakeData, "0041 ; NFD_QC; N\n"},
		"short line":      {"0041;A;Lu\n", fakeProps},
		"bad code point":  {"ZZZZ;A;Lu;0;L;;\n", fakeProps},
		"bad class":       {"0041;A;Lu;x;L;;\n", fakeProps},
		"long decomp":     {"0041;A;Lu;0;L;0042 0043 0044;\n0300;G;Mn;230;NSM;;\n", fakeProps},
		"bad decomp":      {"0041;A;Lu;0;L;004G;\n0300;G;Mn;230;NSM;;\n", fakeProps},
		"bad second":      {"0041;A;Lu;0;L;0042 004G;\n0300;G;Mn;230;NSM;;\n", fakeProps},
		"bad exclusion":   {fakeData, "09ZZ ; Full_Composition_Exclusion\n"},
		"bad range end":   {fakeData, "0958..09ZZ ; Full_Composition_Exclusion\n"},
		"duplicate pairs": {fakeData + "E000;X;Lu;0;L;0041 030A;\n", fakeProps},
	} {
		if _, err := generate([]byte(in[0]), []byte(in[1])); err == nil {
			t.Errorf("%s: generate accepted it", name)
		}
	}
}

func TestSampleTestKeepsShortPartsWhole(t *testing.T) {
	var long strings.Builder
	long.WriteString(fakeTest)
	for i := 0; i < 2*sampleEvery; i++ {
		long.WriteString("00C2;00C2;0041 0302;00C2;0041 0302;\n")
	}
	got, err := sampleTest([]byte(long.String()))
	if err != nil {
		t.Fatal(err)
	}
	out := string(got)
	if !strings.Contains(out, "@Part0\n1E0A;1E0A;0044 0307;1E0A;0044 0307\n@Part1\n00C0;00C0;0041 0300;00C0;0041 0300\n") {
		t.Errorf("unexpected sample:\n%s", out)
	}
	// Part 1 has 2+2*sampleEvery lines; lines 1, 1+sampleEvery and
	// 1+2*sampleEvery are kept.
	if n := strings.Count(out, "\n00C"); n != 3 {
		t.Errorf("%d lines of part 1 kept, want 3:\n%s", n, out)
	}
	if strings.Contains(out, "comment") || strings.Contains(out, "00C1") {
		t.Errorf("a comment or an unsampled line was kept:\n%s", out)
	}
	for name, in := range map[string]string{
		"empty":           "# nothing\n",
		"before any part": "00C0;00C0;0041 0300;00C0;0041 0300;\n",
		"four columns":    "@Part0\n00C0;00C0;0041 0300;00C0;\n",
	} {
		if _, err := sampleTest([]byte(in)); err == nil {
			t.Errorf("%s: sampleTest accepted it", name)
		}
	}
}

// run reads the files from a directory, refuses one whose hash is not the
// pinned one, and with the pins met writes both outputs and then passes its
// own -check.
func TestRunVerifiesHashesAndChecks(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string]string{unicodeData: fakeData, normProps: fakeProps, normTest: fakeTest} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out, vec := filepath.Join(dir, "tables.go"), filepath.Join(dir, "vectors.txt")
	if err := run(out, vec, dir, false, false); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("run accepted files that do not match the pins: %v", err)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("run wrote the tables despite a hash mismatch")
	}
	if err := run(out, vec, dir, false, true); err != nil {
		t.Errorf("-print-hashes failed: %v", err)
	}
	if err := run(out, vec, filepath.Join(dir, "missing"), false, false); err == nil {
		t.Error("run accepted a missing directory")
	}

	saved := sources
	t.Cleanup(func() { sources = saved })
	sources = nil
	for _, name := range []string{unicodeData, normProps, normTest} {
		b, _ := os.ReadFile(filepath.Join(dir, name))
		sources = append(sources, struct{ path, sha256 string }{name, sha(b)})
	}
	if err := run(out, vec, dir, true, false); err == nil {
		t.Error("-check passed with no output files")
	}
	if err := run(out, vec, dir, false, false); err != nil {
		t.Fatal(err)
	}
	if err := run(out, vec, dir, true, false); err != nil {
		t.Errorf("-check failed on files just written: %v", err)
	}
	if err := os.WriteFile(vec, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(out, vec, dir, true, false); err == nil || !strings.Contains(err.Error(), "out of date") {
		t.Errorf("-check passed on a stale file: %v", err)
	}
}
