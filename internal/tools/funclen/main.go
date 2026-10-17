// Command funclen keeps long functions from getting longer. It reports every
// non-test function of the module over a line limit that is not on the allow
// list, and every listed function that has grown past its recorded length.
// The list is a ratchet: it holds the functions that were already long when the
// check was added, each with the length it may not exceed, and -w rewrites it
// from the code (so a function that was split shrinks its entry or drops out).
//
//	go run ./internal/tools/funclen          check
//	go run ./internal/tools/funclen -w       rewrite the allow list
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const allowFile = "internal/tools/funclen/allow.txt"

func main() {
	limit := flag.Int("limit", 150, "longest function, in lines, that needs no entry on the allow list")
	write := flag.Bool("w", false, "rewrite the allow list from the code")
	flag.Parse()

	lengths, err := measure(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "funclen:", err)
		os.Exit(2)
	}
	if *write {
		if err := os.WriteFile(allowFile, []byte(render(lengths, *limit)), 0o644); err != nil { // #nosec G306 -- a source file
			fmt.Fprintln(os.Stderr, "funclen:", err)
			os.Exit(2)
		}
		return
	}
	allow, err := readAllow(allowFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "funclen:", err)
		os.Exit(2)
	}
	var bad []string
	for name, n := range lengths {
		if n <= *limit {
			continue
		}
		cap, listed := allow[name]
		switch {
		case !listed:
			bad = append(bad, fmt.Sprintf("%s is %d lines (limit %d): split it", name, n, *limit))
		case n > cap:
			bad = append(bad, fmt.Sprintf("%s grew from %d to %d lines: split it, do not raise the list", name, cap, n))
		}
	}
	sort.Strings(bad)
	for _, b := range bad {
		fmt.Println(b)
	}
	if len(bad) > 0 {
		os.Exit(1)
	}
}

// measure returns the length in lines of every function and method outside
// test files, vendor and testdata, keyed "dir.Recv.Name".
func measure(root string) (map[string]int, error) {
	out := map[string]int{}
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			n := fset.Position(fd.End()).Line - fset.Position(fd.Pos()).Line + 1
			key := funcName(fd)
			if dir != "." {
				key = dir + "." + key
			}
			if n > out[key] { // two build-tagged files may define the same name
				out[key] = n
			}
		}
		return nil
	})
	return out, err
}

func funcName(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	t := fd.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	if ix, ok := t.(*ast.IndexExpr); ok {
		t = ix.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name + "." + fd.Name.Name
	}
	return fd.Name.Name
}

func render(lengths map[string]int, limit int) string {
	var names []string
	for name, n := range lengths {
		if n > limit {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("# Functions over the funclen limit that were already long, with the length\n")
	b.WriteString("# each may not exceed. Generated: go run ./internal/tools/funclen -w\n")
	for _, name := range names {
		fmt.Fprintf(&b, "%s %d\n", name, lengths[name])
	}
	return b.String()
}

func readAllow(path string) (map[string]int, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- a fixed path in the module
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.LastIndexByte(line, ' ')
		if i < 0 {
			return nil, fmt.Errorf("%s: bad line %q", path, line)
		}
		n, err := strconv.Atoi(line[i+1:])
		if err != nil {
			return nil, fmt.Errorf("%s: bad line %q", path, line)
		}
		out[line[:i]] = n
	}
	return out, nil
}
