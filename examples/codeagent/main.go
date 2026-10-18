// Command codeagent is the shell of a coding agent, driven by a script so it
// runs offline: a transcript of what you asked and what the agent said, tool
// calls shown as cards, and a prompt at the bottom. Replies stream in; a tool
// that changes things stops and waits for your approval, and an edit shows its
// diff first. Plug a real model in where the script is (planFor).
//
// Try: "fix the failing test" (reads a file, shows a diff to approve, runs the
// tests), "delete the build directory" (a high-risk command) or "explain the
// layout package". Starred-risk calls always ask; tools you choose "always
// allow" run without asking for the rest of the session.
//
// Mouse: click Approve, Deny, Always or Diff on a waiting call, click a card's
// title to open or close its output, scroll the transcript with the wheel,
// click the prompt to focus it. Keyboard: Enter sends, Up and Down walk
// through earlier prompts, PgUp and PgDn scroll, on a waiting call y or Enter
// approves, n or Esc denies, a always allows the tool and d shows the whole
// diff, Esc stops a reply, Ctrl+L clears, ? or /help shows help, Ctrl+C stops
// a running reply or quits, Ctrl+D or /quit quits. Pasted text goes to the
// prompt only and can never approve a call. Because letters belong to the
// prompt, q is not a quit key here.
//
//	go run ./examples/codeagent
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
)

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithBracketedPaste(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
