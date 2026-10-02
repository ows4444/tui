package highlight

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

var classNames = [...]string{"Plain", "Keyword", "Type", "String", "Number", "Comment", "Literal", "Key", "Variable"}

// goldenLangs maps a testdata basename to the language name passed to Lines.
// The criterion needs at least 14 highlighted languages, each with a golden.
var goldenLangs = map[string]string{
	"go": "go", "json": "json", "sh": "sh", "yaml": "yaml", "python": "python",
	"js": "javascript", "ts": "typescript", "rust": "rust", "c": "c", "cpp": "c++",
	"java": "java", "diff": "diff", "markdown": "markdown", "sql": "sql",
	"toml": "toml", "dockerfile": "dockerfile",
	"csharp": "c#", "kotlin": "kotlin", "swift": "swift", "php": "php", "ruby": "ruby",
	"lua": "lua", "html": "html", "xml": "xml", "css": "css", "ini": "ini", "makefile": "makefile",
}

// renderSpans prints one "line class text" row per span.
func renderSpans(lines [][]Span) string {
	var b strings.Builder
	for i, l := range lines {
		for _, sp := range l {
			fmt.Fprintf(&b, "%d %s %q\n", i+1, classNames[sp.Class], sp.Text)
		}
	}
	return b.String()
}

func TestGoldenLanguages(t *testing.T) {
	if len(goldenLangs) < 14 {
		t.Fatalf("%d golden languages, want at least 14", len(goldenLangs))
	}
	for name, lang := range goldenLangs {
		t.Run(name, func(t *testing.T) {
			if !Supported(lang) {
				t.Fatalf("%q not supported", lang)
			}
			src, err := os.ReadFile(filepath.Join("testdata", name+".src"))
			if err != nil {
				t.Fatal(err)
			}
			got := renderSpans(Lines(lang, string(src)))
			path := filepath.Join("testdata", name+".golden")
			if *update {
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Errorf("%s differs from %s (run with -update):\n%s", name, path, got)
			}
		})
	}
}

func TestLanguageAliases(t *testing.T) {
	for _, l := range []string{"js", "JavaScript", "jsx", "mjs", "ts", "tsx", "typescript", "rs", "rust", "c", "h", "cpp", "c++", "cc", "hpp", "cxx", "java",
		"diff", "patch", "md", "markdown", "sql", "toml", "dockerfile", "Docker",
		"cs", "C#", "csharp", "kt", "kts", "kotlin", "swift", "php", "rb", "ruby", "lua",
		"html", "htm", "xhtml", "xml", "svg", "xsd", "xsl", "plist", "css", "ini", "cfg", "conf",
		"properties", "editorconfig", "gitconfig", "make", "Makefile", "mk", "GNUmakefile"} {
		if !Supported(l) {
			t.Errorf("Supported(%q) = false", l)
		}
	}
}

// Every language partitions its input and survives malformed text.
func TestNewLanguagesPartitionInput(t *testing.T) {
	inputs := []string{"", "\n", "'", "\"", "`", "/*", "/* /*", "r#\"", "\"\"\"", "R\"(", "#", "[", "[[", "```", "*", "**a", "-- x", "\\", "@@", "é \"é\\", "0x", "1e+", "<", "$", "${", "/", "//", "r#", "'a", "`a",
		"<a", "<a b='", "<a b=", "</", "<!--", "<![CDATA[", "&", "&x", "&#;", "<script>", "<style>a{",
		"<script", "--[[", "--[=[", "[=[", "=begin", "$(", "$$", "$${", "@\"", "$@\"x\"\"", "#[",
		"<?php", "?>", "a{b:", "@media (", ":", "[s", "k=", "\tx # y", "a: b # c", "ifeq ("}
	for name, lang := range goldenLangs {
		for _, in := range inputs {
			var got []string
			for _, l := range Lines(lang, in) {
				var b strings.Builder
				for _, sp := range l {
					b.WriteString(sp.Text)
				}
				got = append(got, b.String())
			}
			if strings.Join(got, "\n") != in {
				t.Errorf("%s %q does not reassemble: %q", name, in, got)
			}
		}
	}
}
