package main

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

// collect runs c with r and gathers its lines until it ends.
func collect(t *testing.T, r runner, c command, cancelAfter time.Duration) (lines []string, d doneMsg) {
	t.Helper()
	ch, cancel := start(context.Background(), r, c)
	defer cancel()
	if cancelAfter > 0 {
		time.AfterFunc(cancelAfter, cancel)
	}
	began := time.Now()
	for {
		switch msg := next(1, ch, began)().(type) {
		case outMsg:
			lines = append(lines, msg.line)
		case doneMsg:
			return lines, msg
		}
	}
}

func TestScriptRunnerPlaysTheScriptAndReturnsItsExit(t *testing.T) {
	c := command{name: "x", script: []string{"a", "b"}, exit: 4}
	lines, d := collect(t, scriptRunner(0), c, 0)
	if strings.Join(lines, "|") != "a|b" || d.code != 4 || d.err != nil {
		t.Fatalf("lines=%v done=%+v", lines, d)
	}
}

func TestScriptRunnerStopsWhenCancelled(t *testing.T) {
	c := command{script: []string{"a", "b", "c"}}
	lines, d := collect(t, scriptRunner(80*time.Millisecond), c, 100*time.Millisecond)
	if d.err == nil || len(lines) >= 3 {
		t.Fatalf("lines=%v err=%v", lines, d.err)
	}
	if describe(d.code, d.err) != "cancelled" {
		t.Fatalf("describe = %q", describe(d.code, d.err))
	}
}

func skipWithoutShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs sh")
	}
}

func TestShellRunnerMergesOutputAndReportsTheExit(t *testing.T) {
	skipWithoutShell(t)
	lines, d := collect(t, shellRunner, command{shell: "echo out; echo err 1>&2; exit 3"}, 0)
	if strings.Join(lines, "|") != "out|err" || d.code != 3 || d.err != nil {
		t.Fatalf("lines=%v done=%+v", lines, d)
	}
	if describe(d.code, d.err) != "exit 3" {
		t.Fatalf("describe = %q", describe(d.code, d.err))
	}
}

func TestShellRunnerSuccess(t *testing.T) {
	skipWithoutShell(t)
	_, d := collect(t, shellRunner, command{shell: "true"}, 0)
	if d.code != 0 || d.err != nil || describe(d.code, d.err) != "exit 0" {
		t.Fatalf("done = %+v", d)
	}
}

func TestShellRunnerCancelKillsTheProcess(t *testing.T) {
	skipWithoutShell(t)
	began := time.Now()
	_, d := collect(t, shellRunner, command{shell: "echo started; sleep 30"}, 150*time.Millisecond)
	if time.Since(began) > 5*time.Second || describe(d.code, d.err) != "cancelled" {
		t.Fatalf("took %v, done=%+v", time.Since(began), d)
	}
}

func TestShellRunnerStripsEscapeSequences(t *testing.T) {
	skipWithoutShell(t)
	lines, _ := collect(t, shellRunner, command{shell: `printf 'a\033[2Jb\033]52;c;eHl6\007c\n'`}, 0)
	if len(lines) != 1 || strings.ContainsAny(lines[0], "\x1b\x07") {
		t.Fatalf("output reached the screen with escapes: %q", lines)
	}
}

func TestShellRunnerHandlesAVeryLongLine(t *testing.T) {
	skipWithoutShell(t)
	lines, d := collect(t, shellRunner, command{shell: "head -c 2000000 /dev/zero | tr '\\0' x; echo; echo after"}, 0)
	if d.err == nil && len(lines) == 0 {
		t.Fatalf("nothing came back: %+v", d)
	}
	for _, l := range lines {
		if len(l) > maxLineBytes {
			t.Fatalf("a line of %d bytes got through", len(l))
		}
	}
}

func TestShellRunnerMissingCommand(t *testing.T) {
	skipWithoutShell(t)
	_, d := collect(t, shellRunner, command{shell: "definitely-not-a-command-xyz"}, 0)
	if d.code == 0 {
		t.Fatalf("a missing command exited 0: %+v", d)
	}
}
