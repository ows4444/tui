// Command apilist prints the exported API of every public package in the
// module, one declaration per entry, so a change to the public surface shows
// up as a diff. It uses only the standard library.
//
//	go run ./internal/tools/apilist            # print the listing
//	go run ./internal/tools/apilist -check     # fail if api.txt is out of date
//	go run ./internal/tools/apilist -w         # rewrite api.txt
//	go run ./internal/tools/apilist -since-tag # fail if a symbol at the latest tag was removed or changed
//
// -since-tag compares the generated listing with api.txt as committed at the
// latest tag (nothing to do before the first tag). It reports lines that
// disappeared or changed, not additions; -allow-breaking reports without
// failing. -ref REV compares with api.txt at REV instead of the latest tag, for
// a repository with no tag yet. Added lines are listed and never fail. With -changelog FILE a removed or changed symbol is accepted when a
// "- BREAKING" bullet in the unreleased (first) section of FILE names its
// identifier as a whole word; any other fails. It cannot see a change that only adds lines but breaks callers,
// such as a new method on an exported interface.
//
// Packages under examples, internal, tools and testdata are not public and are
// skipped, as are package main and test files. Files are parsed regardless of
// build constraints, so platform-specific exports appear once, merged.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const listingFile = "api.txt"

func main() {
	check := flag.Bool("check", false, "fail if "+listingFile+" differs from the generated listing")
	write := flag.Bool("w", false, "write the listing to "+listingFile)
	ref := flag.String("ref", "", "with -since-tag, compare with "+listingFile+" at this git revision instead of the latest tag (needed when no tag exists yet)")
	since := flag.Bool("since-tag", false, "fail if an exported symbol in "+listingFile+" at the latest git tag was removed or changed")
	allow := flag.Bool("allow-breaking", false, "with -since-tag, report breaking changes but do not fail")
	changelog := flag.String("changelog", "", "with -since-tag, accept a breaking change only when a \"- BREAKING\" entry in the unreleased section of this file names its identifier")
	flag.Parse()

	got, err := listing(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "apilist:", err)
		os.Exit(1)
	}
	if *since {
		tag, changes, adds, err := sinceTag(runGit, string(got), *ref)
		if err != nil {
			fmt.Fprintln(os.Stderr, "apilist:", err)
			os.Exit(1)
		}
		if len(adds) > 0 {
			fmt.Printf("apilist: %d exported line(s) added since %s:\n", len(adds), tag)
			for _, a := range adds {
				fmt.Println("  " + a)
			}
		}
		switch {
		case tag == "":
			fmt.Println("apilist: no tag yet, nothing to compare against (use -ref REV to compare with a revision)")
		case len(changes) > 0:
			fmt.Fprintf(os.Stderr, "apilist: exported API removed or changed since %s:\n", tag)
			for _, c := range changes {
				fmt.Fprintln(os.Stderr, "  "+c)
			}
			if *changelog != "" {
				text, err := os.ReadFile(*changelog) // #nosec G304 G703 -- a dev tool reading the file named on its command line
				if err != nil {
					fmt.Fprintln(os.Stderr, "apilist:", err)
					os.Exit(1)
				}
				if open := unacknowledged(changes, string(text)); len(open) > 0 && !*allow {
					fmt.Fprintf(os.Stderr, "apilist: %d change(s) have no \"- BREAKING\" entry naming the identifier in %s:\n", len(open), *changelog)
					for _, c := range open {
						fmt.Fprintln(os.Stderr, "  "+c)
					}
					os.Exit(1)
				}
				fmt.Printf("apilist: every breaking change since %s is named in %s\n", tag, *changelog)
			} else if !*allow {
				fmt.Fprintln(os.Stderr, "apilist: mark the change breaking (-allow-breaking, or -changelog) if it is intended")
				os.Exit(1)
			}
		default:
			fmt.Printf("apilist: no exported symbol removed or changed since %s\n", tag)
		}
	}
	switch {
	case *write:
		if err := os.WriteFile(listingFile, got, 0o644); err != nil { // #nosec G306 -- a public, committed text file
			fmt.Fprintln(os.Stderr, "apilist:", err)
			os.Exit(1)
		}
	case *check:
		want, err := os.ReadFile(listingFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "apilist:", err)
			os.Exit(1)
		}
		if !bytes.Equal(got, want) {
			fmt.Fprintf(os.Stderr, "apilist: %s is out of date; run `go run ./internal/tools/apilist -w` and commit it\n", listingFile)
			fmt.Fprint(os.Stderr, firstDifference(string(want), string(got)))
			os.Exit(1)
		}
	default:
		if _, err := os.Stdout.Write(got); err != nil {
			fmt.Fprintln(os.Stderr, "apilist:", err)
			os.Exit(1)
		}
	}
}

// firstDifference describes the first line where want and got disagree.
func firstDifference(want, got string) string {
	w, g := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return fmt.Sprintf("first difference at line %d:\n  committed: %q\n  generated: %q\n", i+1, a, b)
		}
	}
	return ""
}

// listing returns the API listing for the module rooted at root.
func listing(root string) ([]byte, error) {
	modPath, err := modulePath(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	var dirs []string
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		name := d.Name()
		if p != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") ||
			name == "examples" || name == "internal" || name == "tools" || name == "testdata") {
			return filepath.SkipDir
		}
		dirs = append(dirs, p)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)

	var out bytes.Buffer
	out.WriteString("# Public API of " + modPath + ". Generated by `go run ./internal/tools/apilist -w`; do not edit.\n")
	for _, dir := range dirs {
		decls, pkgName, err := exported(dir)
		if err != nil {
			return nil, err
		}
		if len(decls) == 0 {
			continue
		}
		rel, _ := filepath.Rel(root, dir)
		path := modPath
		if rel != "." {
			path += "/" + filepath.ToSlash(rel)
		}
		fmt.Fprintf(&out, "\n## %s (package %s)\n\n", path, pkgName)
		for _, d := range decls {
			out.WriteString(d + "\n")
		}
	}
	return out.Bytes(), nil
}

func modulePath(gomod string) (string, error) {
	b, err := os.ReadFile(gomod) // #nosec G304 -- a dev tool reading the module file under the directory it is run in
	if err != nil {
		return "", err
	}
	for _, l := range strings.Split(string(b), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(l), "module "); ok {
			return strings.TrimSpace(rest), nil
		}
	}
	return "", fmt.Errorf("no module line in %s", gomod)
}

// exported returns the printed exported declarations of the non-test files in
// dir, sorted, with duplicates from build-constrained files merged.
func exported(dir string) ([]string, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}
	fset := token.NewFileSet()
	seen := map[string]bool{}
	var decls []string
	pkgName := ""
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, n), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, "", err
		}
		if f.Name.Name == "main" {
			return nil, "", nil
		}
		pkgName = f.Name.Name
		ast.FileExports(f) // drops unexported declarations, fields and methods
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
				continue
			}
			if fn, ok := d.(*ast.FuncDecl); ok && onUnexportedType(fn) {
				continue // a method of an unexported type is not named API
			}
			stripComments(d)
			var b bytes.Buffer
			if err := printer.Fprint(&b, fset, d); err != nil {
				return nil, "", err
			}
			s := b.String()
			if !seen[s] {
				seen[s] = true
				decls = append(decls, s)
			}
		}
	}
	sort.Strings(decls)
	return decls, pkgName, nil
}

// onUnexportedType reports whether fn is a method whose receiver type is not
// exported, however its own name is spelled.
func onUnexportedType(fn *ast.FuncDecl) bool {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	t := fn.Recv.List[0].Type
	for {
		switch x := t.(type) {
		case *ast.StarExpr:
			t = x.X
		case *ast.ParenExpr:
			t = x.X
		case *ast.IndexExpr: // generic receiver T[K]
			t = x.X
		case *ast.IndexListExpr:
			t = x.X
		case *ast.Ident:
			return !x.IsExported()
		default:
			return false
		}
	}
}

func stripComments(n ast.Node) {
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			x.Doc = nil
		case *ast.FuncDecl:
			x.Doc = nil
			x.Body = nil // the signature is the API, not the implementation
		case *ast.ValueSpec:
			x.Doc, x.Comment = nil, nil
		case *ast.TypeSpec:
			x.Doc, x.Comment = nil, nil
		case *ast.Field:
			x.Doc, x.Comment = nil, nil
		}
		return true
	})
}
