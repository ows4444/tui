package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/prompt"
)

// exitError ends the command with a given exit code. A message is printed, an
// empty one is not: `confirm` answering no, or a canceled prompt, is not an
// error to report.
type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

const (
	exitNo       = 1   // confirm answered no
	exitCanceled = 130 // Esc or Ctrl+C, as a shell reports an interrupt
)

// ui returns the program options that draw a prompt on standard error, so the
// answer is the only thing on standard output and `x=$(tui prompt input Name)`
// captures just that. Tests replace it.
var ui = func() []tui.ProgramOption { return []tui.ProgramOption{tui.WithOutput(os.Stderr)} }

const promptUsage = `usage: tui prompt input [-placeholder s] [-default s] <label>
       tui prompt password <label>
       tui prompt confirm [-yes] <label>
       tui prompt select <label> <choice>...
       tui prompt multiselect <label> <choice>...
       tui prompt spin <label> -- <command> [args]...`

// runPrompt runs one prompt and prints the answer to out. The prompt itself is
// drawn on standard error and reads the terminal.
func runPrompt(args []string, out io.Writer) error {
	if len(args) == 0 {
		return usageError(promptUsage)
	}
	kind, rest := args[0], args[1:]
	fs := flag.NewFlagSet("prompt "+kind, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	placeholder := fs.String("placeholder", "", "hint shown while empty (input, password)")
	def := fs.String("default", "", "text to start with (input, password)")
	yes := fs.Bool("yes", false, "Enter answers yes (confirm)")
	switch kind {
	case "input", "password", "confirm", "select", "multiselect":
	case "spin":
		return runSpin(rest)
	default:
		return usageError(promptUsage)
	}
	if err := fs.Parse(rest); err != nil {
		return fmt.Errorf("%v\n%s", err, promptUsage)
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return usageError(promptUsage)
	}
	label, choices := pos[0], pos[1:]
	cfg := prompt.Config{Program: ui(), Placeholder: *placeholder, Default: *def}

	var err error
	switch kind {
	case "input":
		var s string
		if s, err = prompt.Input(label, cfg); err == nil {
			fmt.Fprintln(out, s)
		}
	case "password":
		var s string
		if s, err = prompt.Password(label, cfg); err == nil {
			fmt.Fprintln(out, s)
		}
	case "confirm":
		var ok bool
		if ok, err = prompt.Confirm(label, *yes, cfg); err == nil && !ok {
			return &exitError{code: exitNo}
		}
	case "select":
		if len(choices) == 0 {
			return usageError(promptUsage)
		}
		var i int
		if i, err = prompt.Select(label, choices, cfg); err == nil {
			fmt.Fprintln(out, choices[i])
		}
	case "multiselect":
		if len(choices) == 0 {
			return usageError(promptUsage)
		}
		var is []int
		if is, err = prompt.MultiSelect(label, choices, cfg); err == nil {
			for _, i := range is {
				fmt.Fprintln(out, choices[i])
			}
		}
	}
	if errors.Is(err, prompt.ErrCanceled) {
		return &exitError{code: exitCanceled}
	}
	return err
}

// runSpin runs a command under a spinner. The command's own output is held and
// shown only if it fails, and its exit code becomes ours.
func runSpin(args []string) error {
	label, command, ok := splitDash(args)
	if !ok || len(command) == 0 {
		return usageError(promptUsage)
	}
	var captured strings.Builder
	code := 0
	err := prompt.Spin(label, func(ctx context.Context) error {
		c := exec.CommandContext(ctx, command[0], command[1:]...) // #nosec G204 G702 -- the command the user asked to run
		c.Stdout, c.Stderr = &captured, &captured
		runErr := c.Run()
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			code = ee.ExitCode()
		} else if runErr != nil {
			return runErr
		}
		return nil
	}, prompt.Config{Program: ui()})
	if errors.Is(err, prompt.ErrCanceled) {
		return &exitError{code: exitCanceled}
	}
	if err != nil {
		return err
	}
	if code != 0 {
		fmt.Fprint(os.Stderr, captured.String())
		return &exitError{code: code}
	}
	return nil
}

// splitDash splits `label -- command args` at the first "--".
func splitDash(args []string) (label string, command []string, ok bool) {
	for i, a := range args {
		if a == "--" {
			return strings.Join(args[:i], " "), args[i+1:], i > 0
		}
	}
	return "", nil, false
}
