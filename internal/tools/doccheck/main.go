// Command doccheck lists exported declarations that have no doc comment and
// exits non-zero if there are any. It uses only the standard library.
//
//	go run ./internal/tools/doccheck          # check the module in the current directory
//	go run ./internal/tools/doccheck ./path   # check another root
//
// Packages under examples, internal, tools and testdata are skipped, as are
// package main and test files. A declaration counts as documented when it, or
// the const/var/type group it sits in, carries a comment. Methods count only
// when both the method and its receiver type are exported.
//
// It also fails when a package, internal ones included, has no package comment
// in any of its files (test files count; examples and testdata are skipped).
//
// It also fails when the "Experimental:" list in the root doc.go and the
// packages whose comment says "Stability: experimental." disagree.
//
// It also fails when a markdown file or a workflow names a ./path, or an import
// path under the module, that does not exist (CHANGELOG.md, the audit report and
// the agent prompts are skipped: they name paths that were removed).
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	missing, err := check(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccheck:", err)
		os.Exit(1)
	}
	for _, m := range missing {
		fmt.Fprintln(os.Stderr, m)
	}
	bare, err := packageComments(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccheck:", err)
		os.Exit(1)
	}
	for _, m := range bare {
		fmt.Fprintln(os.Stderr, m)
	}
	unstable, err := stability(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccheck:", err)
		os.Exit(1)
	}
	for _, m := range unstable {
		fmt.Fprintln(os.Stderr, m)
	}
	stale, err := stalePaths(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "doccheck:", err)
		os.Exit(1)
	}
	for _, m := range stale {
		fmt.Fprintln(os.Stderr, m)
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "doccheck: %d exported identifier(s) without a doc comment\n", len(missing))
	}
	if len(bare) > 0 {
		fmt.Fprintf(os.Stderr, "doccheck: %d package(s) without a package comment\n", len(bare))
	}
	if len(unstable) > 0 {
		fmt.Fprintf(os.Stderr, "doccheck: %d stability disagreement(s) between doc.go and package comments\n", len(unstable))
	}
	if len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "doccheck: %d reference(s) to a path that does not exist\n", len(stale))
	}
	if len(missing)+len(bare)+len(unstable)+len(stale) > 0 {
		os.Exit(1)
	}
}

// skipDir reports whether a directory holds no public package.
func skipDir(name string) bool {
	switch name {
	case "examples", "internal", "tools", "testdata", "vendor":
		return true
	}
	return strings.HasPrefix(name, ".")
}

// check returns "file:line: name" for every undocumented exported identifier
// under root, in walk then source order.
func check(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // #nosec G703 -- a dev tool walking the tree named on its command line
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		if f.Name.Name == "main" {
			return nil
		}
		for _, name := range undocumented(f) {
			out = append(out, fmt.Sprintf("%s:%d: %s", path, fset.Position(name.pos).Line, name.name))
		}
		return nil
	})
	return out, err
}

type finding struct {
	pos  token.Pos
	name string
}

// undocumented returns the exported declarations in f without a comment.
func undocumented(f *ast.File) []finding {
	var out []finding
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() || d.Doc != nil {
				continue
			}
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := receiverName(d.Recv.List[0].Type)
				if !ast.IsExported(recv) {
					continue
				}
				name = recv + "." + name
			}
			out = append(out, finding{d.Pos(), name})
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() && d.Doc == nil && s.Doc == nil {
						out = append(out, finding{s.Pos(), s.Name.Name})
					}
				case *ast.ValueSpec:
					if d.Doc != nil || s.Doc != nil {
						continue
					}
					for _, n := range s.Names {
						if n.IsExported() {
							out = append(out, finding{n.Pos(), n.Name})
						}
					}
				}
			}
		}
	}
	return out
}

// receiverName returns the base type name of a method receiver.
func receiverName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return receiverName(t.X)
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

// modulePath is the import path prefix whose references stalePaths verifies.
const modulePath = "github.com/ows4444/tui"

// quotesPaths names the markdown files that mention removed paths on purpose:
// the changelog and audit reports record what was deleted, and prompt.md and
// role.md are agent prompts, not documentation of the tree.
var quotesPaths = map[string]bool{"CHANGELOG.md": true, "TUI_AUDIT.md": true, "TUI_AUDIT_2026-10-01.md": true, "prompt.md": true, "role.md": true}

// pathRef matches a ./relative path or a module import path inside text. The
// leading class keeps it from matching the middle of a longer path or URL.
var pathRef = regexp.MustCompile("(?:^|[\\s`'\"(=:])(\\./[A-Za-z0-9_][A-Za-z0-9_./-]*|" + regexp.QuoteMeta(modulePath) + "/[A-Za-z0-9_][A-Za-z0-9_./-]*)")

// stalePaths returns "file:line: path" for every ./path or module import path
// that a *.md file or a .github workflow names and that does not exist under
// root, in walk then source order. "./..." patterns and the files in
// quotesPaths are skipped.
func stalePaths(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // #nosec G703 -- a dev tool walking the tree named on its command line
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		inWorkflow := strings.Contains(filepath.ToSlash(path), ".github/workflows/") && (ext == ".yml" || ext == ".yaml")
		if (ext != ".md" && !inWorkflow) || quotesPaths[d.Name()] {
			return nil
		}
		raw, err := os.ReadFile(path) // #nosec G304 G122 -- a dev tool reading the tree named on its command line
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, m := range pathRef.FindAllStringSubmatch(line, -1) {
				ref := strings.TrimRight(m[1], ".,;:)/")
				if strings.Contains(ref, "...") {
					continue
				}
				rel := strings.TrimPrefix(strings.TrimPrefix(ref, modulePath), "/")
				rel = strings.TrimPrefix(rel, "./")
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil { // #nosec G703 -- a dev tool checking paths named in the tree it was pointed at
					out = append(out, fmt.Sprintf("%s:%d: %s", path, i+1, ref))
				}
			}
		}
		return nil
	})
	return out, err
}

// packageComments returns "dir: package name" for every package under root
// whose files carry no package comment. Unlike check it descends into
// internal, and it counts test files, because a test-only package has no other
// place for one. examples, testdata, vendor and dot directories are skipped.
func packageComments(root string) ([]string, error) {
	documented := map[string]bool{}
	names := map[string]string{}
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // #nosec G703 -- a dev tool walking the tree named on its command line
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "examples", "testdata", "vendor":
				return filepath.SkipDir
			}
			if path != root && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		dir := filepath.Dir(path)
		if _, seen := names[dir]; !seen {
			dirs = append(dirs, dir)
			names[dir] = f.Name.Name
		}
		if f.Doc != nil && strings.TrimSpace(f.Doc.Text()) != "" {
			documented[dir] = true
		}
		return nil
	})
	var out []string
	for _, dir := range dirs {
		if !documented[dir] {
			out = append(out, fmt.Sprintf("%s: package %s has no package comment", dir, names[dir]))
		}
	}
	return out, err
}

// experimentalMarker is the sentence a package comment carries to say the
// package is experimental; the root doc.go lists the same packages after
// "Experimental:".
const experimentalMarker = "stability: experimental"

// stability returns one line for each package that the root doc.go lists as
// experimental without saying so in its package comment, and for each package
// that says so without being listed. It reports nothing when root has no
// doc.go. Packages under internal, examples, testdata and vendor are skipped.
func stability(root string) ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(root, "doc.go")) // #nosec G304 G703 -- a dev tool reading the tree named on its command line
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	f, err := parser.ParseFile(token.NewFileSet(), "doc.go", raw, parser.ParseComments|parser.PackageClauseOnly)
	if err != nil {
		return nil, err
	}
	listed := map[string]bool{}
	if f.Doc != nil {
		for _, para := range strings.Split(f.Doc.Text(), "\n\n") {
			rest, ok := strings.CutPrefix(strings.TrimSpace(para), "Experimental:")
			if !ok {
				continue
			}
			for _, name := range strings.Split(strings.Join(strings.Fields(rest), " "), ",") {
				if name = strings.Trim(strings.TrimSpace(name), "."); name != "" {
					listed[name] = true
				}
			}
		}
	}
	marked := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error { // #nosec G703 -- a dev tool walking the tree named on its command line
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (skipDir(d.Name()) || d.Name() == "internal") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Dir(path) == filepath.Clean(root) || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		pf, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		if pf.Doc != nil && strings.Contains(strings.ToLower(strings.Join(strings.Fields(pf.Doc.Text()), " ")), experimentalMarker) {
			rel, _ := filepath.Rel(root, filepath.Dir(path))
			marked[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	var out []string
	for name := range listed {
		if !marked[name] {
			out = append(out, fmt.Sprintf("%s: listed as experimental in doc.go but its package comment lacks %q", name, experimentalMarker))
		}
	}
	for name := range marked {
		if !listed[name] {
			out = append(out, fmt.Sprintf("%s: package comment says %q but doc.go does not list it", name, experimentalMarker))
		}
	}
	sort.Strings(out)
	return out, err
}
