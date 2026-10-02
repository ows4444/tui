package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDocsListEveryGlyphReaderAndLexer keeps docs/theming.md and the
// CodeBlockLang doc comment in step with the code: every non-test file that
// reads theme.Glyphs must be named in the theming guide, and every language
// name the highlighter accepts must be in CodeBlockLang's comment.
func TestDocsListEveryGlyphReaderAndLexer(t *testing.T) {
	raw, err := os.ReadFile("docs/theming.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := strings.ToLower(string(raw))
	glyphRef := regexp.MustCompile(`\bGlyphs\b`)

	readers := map[string]bool{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		switch e.Name() {
		case "examples", "tools", "internal", "testdata", "docs", "bench", "theme":
			continue
		}
		files, _ := filepath.Glob(filepath.Join(e.Name(), "*.go"))
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if !glyphRef.Match(src) {
				continue
			}
			if e.Name() == "widgets" {
				readers[strings.TrimSuffix(filepath.Base(f), ".go")] = true
			} else {
				readers[e.Name()] = true
			}
		}
	}
	if len(readers) == 0 {
		t.Fatal("found no glyph readers; the scan is broken")
	}
	for name := range readers {
		if !strings.Contains(doc, name) {
			t.Errorf("docs/theming.md does not list %q, which reads theme.Glyphs", name)
		}
	}

	f, err := parser.ParseFile(token.NewFileSet(), "internal/highlight/highlight.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	comment, err := os.ReadFile("widgets/codeblock_lang.go")
	if err != nil {
		t.Fatal(err)
	}
	var langs int
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "lexerFor" {
			return true
		}
		ast.Inspect(fd, func(n ast.Node) bool {
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, e := range cc.List {
				if bl, ok := e.(*ast.BasicLit); ok {
					name, _ := strconv.Unquote(bl.Value)
					langs++
					if !strings.Contains(string(comment), `"`+name+`"`) {
						t.Errorf("CodeBlockLang's doc comment does not list lexer language %q", name)
					}
				}
			}
			return true
		})
		return false
	})
	if langs == 0 {
		t.Fatal("found no lexer languages; the scan is broken")
	}
}
