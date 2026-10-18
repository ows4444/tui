package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// ---- tui dev: rebuild and restart a program when its source changes ----

// devConfig is what tui dev needs, made explicit for tests.
type devConfig struct {
	dir    string        // where go build runs and the program starts
	pkg    string        // the package to build, as `go build` takes it
	args   []string      // the program's arguments
	watch  []string      // directories whose source is watched
	poll   time.Duration // how often the source is looked at
	goCmd  string        // the go tool ("go")
	stdin  io.Reader     // the program's input; the terminal
	stdout io.Writer     // the program's output
	status io.Writer     // tui dev's own messages and compiler errors
}

// stringList is a repeatable flag.
type stringList []string

func (l *stringList) String() string     { return strings.Join(*l, ",") }
func (l *stringList) Set(s string) error { *l = append(*l, s); return nil }

// runDev is tui dev [-poll d] [-watch dir]... [package] [-- args].
func runDev(args []string, out io.Writer, stop <-chan struct{}) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	poll := fs.Duration("poll", 400*time.Millisecond, "how often to look for changes")
	var watch stringList
	fs.Var(&watch, "watch", "also watch this directory (repeatable)")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%v\n%s", err, usage)
	}
	rest := fs.Args()
	pkg := "."
	if len(rest) > 0 && rest[0] != "--" {
		pkg, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		if rest[0] != "--" {
			return usageError(usage)
		}
		rest = rest[1:]
	}
	if *poll <= 0 {
		return errors.New("tui: -poll must be positive")
	}
	cfg := devConfig{pkg: pkg, args: rest, watch: watch, poll: *poll, goCmd: "go", stdin: os.Stdin, stdout: out, status: os.Stderr}
	if len(cfg.watch) == 0 {
		cfg.watch = []string{moduleDir(cfg)}
	}
	return dev(cfg, stop)
}

// moduleDir is the directory of the module in the working directory, or the
// working directory itself when there is none.
func moduleDir(cfg devConfig) string {
	cmd := exec.Command(cfg.goCmd, "list", "-m", "-f", "{{.Dir}}") // #nosec G204 -- fixed arguments
	cmd.Dir = cfg.dir
	if out, err := cmd.Output(); err == nil {
		if d := strings.TrimSpace(string(out)); d != "" {
			return d
		}
	}
	if cfg.dir != "" {
		return cfg.dir
	}
	return "."
}

// fileState is what a change in a file shows as.
type fileState struct {
	mod  time.Time
	size int64
}

// watched reports whether a change in the file can change the program.
func watched(name string) bool {
	if strings.HasSuffix(name, "_test.go") {
		return false
	}
	switch filepath.Ext(name) {
	case ".go", ".mod", ".sum", ".tmpl":
		return true
	}
	return false
}

// snapshot lists the source files under dirs.
func snapshot(dirs []string) map[string]fileState {
	snap := map[string]fileState{}
	for _, root := range dirs {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // a file that vanished mid-walk is a change, found next time
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", "testdata", "site", "vendor":
					return filepath.SkipDir
				}
				return nil
			}
			if !watched(d.Name()) {
				return nil
			}
			if info, err := d.Info(); err == nil {
				snap[p] = fileState{info.ModTime(), info.Size()}
			}
			return nil
		})
	}
	return snap
}

// changedFiles returns the files that differ between two snapshots.
func changedFiles(a, b map[string]fileState) []string {
	var out []string
	for p, s := range b {
		if o, ok := a[p]; !ok || o != s {
			out = append(out, p)
		}
	}
	for p := range a {
		if _, ok := b[p]; !ok {
			out = append(out, p)
		}
	}
	return out
}

// dev builds the package, runs it, and builds and runs it again whenever the
// source changes, until stop is closed. A build that fails prints the compiler's
// errors and waits for the next change.
func dev(cfg devConfig, stop <-chan struct{}) error {
	tmp, err := os.MkdirTemp("", "tui-dev-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	bin := filepath.Join(tmp, "app"+exeSuffix)
	say := func(format string, a ...any) { fmt.Fprintf(cfg.status, "tui dev: "+format+"\n", a...) }
	for {
		base := snapshot(cfg.watch)
		say("building %s", cfg.pkg)
		var child *exec.Cmd
		var exited chan error
		if msg, err := build(cfg, bin); err != nil {
			fmt.Fprint(cfg.status, msg)
			say("build failed; waiting for changes")
		} else {
			if child, exited, err = start(cfg, bin); err != nil {
				say("%v", err)
			} else {
				say("running (Ctrl+C stops)")
			}
		}
		why := wait(cfg, base, stop, exited, say)
		if child != nil && why != "exited" {
			if killed := halt(child, exited); killed {
				resetTerminal(cfg)
			}
		}
		if why == "stop" {
			return nil
		}
		if why == "exited" {
			// Nothing runs now: wait for a change before building again.
			if !idle(cfg, base, stop) {
				return nil
			}
		}
		say("source changed; restarting")
	}
}

// wait blocks until the source changes ("change"), the program exits
// ("exited") or stop closes ("stop").
func wait(cfg devConfig, base map[string]fileState, stop <-chan struct{}, exited <-chan error, say func(string, ...any)) string {
	tick := time.NewTicker(cfg.poll)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return "stop"
		case err := <-exited:
			if err != nil {
				say("exited: %v; waiting for changes", err)
			} else {
				say("exited; waiting for changes")
			}
			return "exited"
		case <-tick.C:
			if len(changedFiles(base, snapshot(cfg.watch))) > 0 {
				return "change"
			}
		}
	}
}

// idle waits for a change and reports false if stop closed first.
func idle(cfg devConfig, base map[string]fileState, stop <-chan struct{}) bool {
	tick := time.NewTicker(cfg.poll)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return false
		case <-tick.C:
			if len(changedFiles(base, snapshot(cfg.watch))) > 0 {
				return true
			}
		}
	}
}

// build compiles the package to bin and returns the compiler's complaints.
func build(cfg devConfig, bin string) (string, error) {
	cmd := exec.Command(cfg.goCmd, "build", "-o", bin, cfg.pkg) // #nosec G204 -- the package is the one the user named
	cmd.Dir = cfg.dir
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	return buf.String(), err
}

// start runs the built program on the terminal and reports its exit on the
// returned channel.
func start(cfg devConfig, bin string) (*exec.Cmd, chan error, error) {
	cmd := exec.Command(bin, cfg.args...) // #nosec G204 -- the program was just built from the user's package
	cmd.Dir = cfg.dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = cfg.stdin, cfg.stdout, cfg.status
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	return cmd, exited, nil
}

// halt asks the program to end (SIGTERM, so it can restore the terminal), waits
// a moment, then kills it. It reports whether it had to kill.
func halt(cmd *exec.Cmd, exited chan error) (killed bool) {
	if cmd.Process == nil {
		return false
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
		<-exited
		return true
	}
	select {
	case <-exited:
		return false
	case <-time.After(2 * time.Second):
		_ = cmd.Process.Kill()
		<-exited
		return true
	}
}

// resetTerminal puts the terminal back after a program was killed before it
// could: it leaves the alternate screen, shows the cursor and turns off the
// mouse, paste and keyboard modes a program may have set.
func resetTerminal(cfg devConfig) {
	fmt.Fprint(cfg.status, "\x1b[?1049l\x1b[?25h\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?2004l\x1b[<u\x1b[0m")
	if f, ok := cfg.stdin.(*os.File); ok {
		cmd := exec.Command("stty", "sane") // #nosec G204 -- fixed command
		cmd.Stdin = f
		_ = cmd.Run() // best effort: there may be no stty, or no terminal
	}
}

// exeSuffix is what a built program is named with: Windows will not run a file
// without .exe.
var exeSuffix = func() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}()
