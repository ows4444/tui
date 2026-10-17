// Command tui scaffolds a program built on github.com/ows4444/tui, and checks
// theme files:
//
//	go run github.com/ows4444/tui/cmd/tui new myapp
//	go run github.com/ows4444/tui/cmd/tui new -module example.com/me/myapp myapp
//	go run github.com/ows4444/tui/cmd/tui theme check mytheme.json
//	go run github.com/ows4444/tui/cmd/tui theme list
//	name=$(go run github.com/ows4444/tui/cmd/tui prompt input "Your name")
//	go run github.com/ows4444/tui/cmd/tui keys        (what the terminal sends, live)
//	go run github.com/ows4444/tui/cmd/tui log -f app.log
//	go run github.com/ows4444/tui/cmd/tui colors nord
//	go run github.com/ows4444/tui/cmd/tui borders
//	go run github.com/ows4444/tui/cmd/tui easings out-bounce
//
// It writes a directory holding go.mod, main.go (a small Model), main_test.go
// (a headless test with tuitest) and a README. It writes no dependency version
// of its own: run `go mod tidy` in the new directory to resolve the latest.
package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// tmpl holds every template; a malformed one fails at start-up, in the tests.
var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.tmpl"))

// files maps each template to the file it makes.
var files = []struct{ tmpl, out string }{
	{"go.mod.tmpl", "go.mod"},
	{"main.go.tmpl", "main.go"},
	{"main_test.go.tmpl", "main_test.go"},
	{"README.md.tmpl", "README.md"},
}

func main() { os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr)) }

// cli runs the command and returns the process exit code.
func cli(args []string, stdout, stderr io.Writer) int {
	if err := run(args, stdout); err != nil {
		var ee *exitError
		if errors.As(err, &ee) {
			if ee.msg != "" {
				fmt.Fprintln(stderr, "tui:", ee.msg)
			}
			return ee.code
		}
		fmt.Fprintln(stderr, "tui:", strings.TrimPrefix(err.Error(), "tui: "))
		return 1
	}
	return 0
}

const usage = `usage: tui new [-module path] <dir>
       tui theme check [-min ratio] <file.json>
       tui theme list
       tui keys check <file.json>
       tui keys
       tui log [-f] [-n lines] [-grep text] <file>
       tui colors [preset]
       tui borders
       tui easings [name]
       tui prompt input|password|confirm|select|multiselect|spin ...`

var name = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9._-]*$`)

// run is main with its arguments and output made explicit, for tests.
func run(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "new":
		return runNew(args, out)
	case "theme":
		return runTheme(args[1:], out)
	case "keys":
		return runKeys(args[1:], out)
	case "log":
		stop, cancel := interrupted()
		defer cancel()
		return runLog(args[1:], out, stop)
	case "colors":
		return runColors(args[1:], out)
	case "borders":
		if len(args) != 1 {
			return errors.New(usage)
		}
		return runBorders(out)
	case "easings":
		return runEasings(args[1:], out)
	case "prompt":
		return runPrompt(args[1:], out)
	}
	return errors.New(usage)
}

// runNew scaffolds a program: tui new [-module path] <dir>.
func runNew(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	module := fs.String("module", "", "module path (default: the directory's name)")
	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	if fs.NArg() != 1 {
		return errors.New(usage)
	}
	dir := fs.Arg(0)
	base := filepath.Base(filepath.Clean(dir))
	if !name.MatchString(base) {
		return fmt.Errorf("%q is not a usable program name: start with a letter, then letters, digits, '.', '_' or '-'", base)
	}
	mod := *module
	if mod == "" {
		mod = base
	}
	if strings.ContainsAny(mod, " \t\n\"") {
		return fmt.Errorf("%q is not a module path", mod)
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s exists and is not empty", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- a project directory, world-readable like any source tree
		return err
	}
	data := struct{ Name, Module string }{base, mod}
	for _, f := range files {
		if err := render(filepath.Join(dir, f.out), f.tmpl, data); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "created %s\n\n  cd %s\n  go mod tidy\n  go run .\n", dir, dir)
	return nil
}

// render writes the template name, filled from data, to path.
func render(path, name string, data any) error {
	fh, err := os.Create(path) // #nosec G304 -- a path under the directory the user named
	if err != nil {
		return err
	}
	err = tmpl.ExecuteTemplate(fh, name, data)
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	return err
}
