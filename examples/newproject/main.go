// Command newproject asks a few questions and writes a project skeleton, with
// no Model of its own: every question is one call of package prompt.
//
//	newproject            # asks, then creates ./<name>
//	newproject -dry-run   # asks, then only says what it would create
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/ows4444/tui/prompt"
)

type answers struct {
	Name     string
	Language string
	Features []string
	Confirm  bool
}

var languages = []string{"Go", "Rust", "Python", "TypeScript"}

var features = []string{"README", "LICENSE (MIT)", "CI workflow", "Dockerfile", ".gitignore"}

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// validName is the rule Input applies to the project name.
func validName(s string) error {
	if !nameRE.MatchString(s) {
		return errors.New("lower-case letters, digits and '-', starting with a letter")
	}
	return nil
}

// ask runs the five questions. cfg returns the prompt.Config for question i
// (0 to 4), so a test can give each its own input and output.
func ask(cfg func(i int) prompt.Config) (answers, error) {
	var a answers
	var err error

	c := cfg(0)
	c.Validate, c.Placeholder = validName, "my-project"
	if a.Name, err = prompt.Input("Project name:", c); err != nil {
		return a, err
	}
	li, err := prompt.Select("Language:", languages, cfg(1))
	if err != nil {
		return a, err
	}
	a.Language = languages[li]
	picked, err := prompt.MultiSelect("Include (space ticks):", features, cfg(2))
	if err != nil {
		return a, err
	}
	for _, i := range picked {
		a.Features = append(a.Features, features[i])
	}
	a.Confirm, err = prompt.Confirm(fmt.Sprintf("Create %s (%s)?", a.Name, a.Language), true, cfg(3))
	return a, err
}

// files is what the answers turn into: path and contents.
func files(a answers) map[string]string {
	out := map[string]string{}
	has := func(f string) bool {
		for _, x := range a.Features {
			if x == f {
				return true
			}
		}
		return false
	}
	if has("README") {
		out["README.md"] = "# " + a.Name + "\n\nA " + a.Language + " project.\n"
	}
	if has("LICENSE (MIT)") {
		out["LICENSE"] = "MIT License\n\nCopyright (c) " + a.Name + " authors\n"
	}
	if has("CI workflow") {
		out[".github/workflows/ci.yml"] = "name: CI\non: [push, pull_request]\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n"
	}
	if has("Dockerfile") {
		out["Dockerfile"] = "FROM scratch\n"
	}
	if has(".gitignore") {
		out[".gitignore"] = "*.log\n.DS_Store\n"
	}
	return out
}

// create writes files under root/name, or only lists them when dry is set.
func create(root string, a answers, dry bool, out io.Writer) error {
	base := filepath.Join(root, a.Name)
	fs := files(a)
	if len(fs) == 0 {
		fmt.Fprintf(out, "%s: nothing ticked, nothing to create\n", a.Name)
		return nil
	}
	var names []string
	for n := range fs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintln(out, filepath.Join(a.Name, n))
		if dry {
			continue
		}
		p := filepath.Join(base, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { // #nosec G301 -- a project directory, world-readable like any source tree
			return err
		}
		if err := os.WriteFile(p, []byte(fs[n]), 0o644); err != nil { // #nosec G306 -- a source file of a new project
			return err
		}
	}
	return nil
}

func main() {
	dry := flag.Bool("dry-run", false, "only list what would be created")
	flag.Parse()
	a, err := ask(func(int) prompt.Config { return prompt.Config{} })
	switch {
	case errors.Is(err, prompt.ErrCanceled):
		fmt.Fprintln(os.Stderr, "canceled")
		os.Exit(130)
	case err != nil:
		fmt.Fprintln(os.Stderr, "newproject:", err)
		os.Exit(1)
	case !a.Confirm:
		fmt.Fprintln(os.Stderr, "not created")
		return
	}
	if err := create(".", a, *dry, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "newproject:", err)
		os.Exit(1)
	}
}
