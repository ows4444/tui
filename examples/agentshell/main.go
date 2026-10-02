// Command agentshell is a template for an agent CLI. It runs inline (no
// alternate screen): finished messages are committed to real terminal
// scrollback with tui.Println, while the live region at the bottom shows only
// what is still changing - the reply being streamed, a pending tool-call
// approval, and the prompt.
//
// The pieces:
//
//   - streamtext.Model reveals the assistant reply token by token;
//   - widgets.ChatMessage + markdown.Render format each committed message;
//   - toolapproval.Model gates a tool call behind an explicit key press;
//   - textarea.Model is the multi-line prompt; a bracketed paste
//     (tui.PasteEvent) lands there and nowhere else, so pasted text can never
//     approve or deny the tool call;
//   - tui.Go runs the (fake) model call with the Program's context.
//
// The model here is scripted and offline; see fakeModel and docs/agent-shell.md
// for where to plug in a real one.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/widgets"
)

// defaultWidth is the wrap width until the first tui.ResizeMsg arrives.
const defaultWidth = 60

// reply is one scripted model answer: markdown text delivered as tokens, then
// optionally a tool call that needs the user's approval.
type reply struct {
	tokens []string
	tool   *toolCall
}

type toolCall struct {
	name, description string
	risk              toolapproval.Risk
	output            string // what the tool prints if approved
}

// fakeModel is the stand-in for a real model client. Replace call (and the
// replyMsg it returns) with a request to your provider; everything else in
// this file is provider-independent.
type fakeModel struct {
	latency time.Duration // simulated time to first token
}

const scriptedAnswer = "I can run the tests for you. Here is the plan:\n\n" +
	"- run `go test ./...`\n- report any failures\n\n" +
	"This needs your **approval** first."

// call is what tui.Go runs: it must honour ctx so quitting cancels it.
func (f fakeModel) call(prompt string) func(ctx context.Context) tui.Msg {
	return func(ctx context.Context) tui.Msg {
		select {
		case <-time.After(f.latency):
		case <-ctx.Done():
			return replyMsg{err: ctx.Err()}
		}
		return replyMsg{reply: reply{
			tokens: strings.SplitAfter(scriptedAnswer, " "),
			tool: &toolCall{
				name:        "run_tests",
				description: "go test ./... (asked by: " + firstLine(prompt) + ")",
				risk:        toolapproval.RiskMedium,
				output:      "ok  example/pkg  0.012s",
			},
		}}
	}
}

func firstLine(s string) string {
	s, _, _ = strings.Cut(strings.TrimSpace(s), "\n")
	if r := []rune(s); len(r) > 30 {
		return string(r[:30])
	}
	return s
}

// replyMsg is the model call's result.
type replyMsg struct {
	reply reply
	err   error
}

// tokenMsg asks for the next scripted token. The model schedules it itself
// every tokenEvery; tests send it by hand.
type tokenMsg struct{}

type model struct {
	fake       fakeModel
	tokenEvery time.Duration // 0: tokens only advance on a tokenMsg sent by hand
	now        func() time.Time
	theme      theme.Theme
	width      int

	prompt textarea.Model

	busy    bool // a model call or stream is in progress
	script  []string
	next    int
	stream  streamtext.Model
	tool    *toolCall
	pending bool // a tool approval is waiting for a key
	approve toolapproval.Model
}

func initialModel() model {
	m := model{
		fake:       fakeModel{latency: 400 * time.Millisecond},
		tokenEvery: 35 * time.Millisecond,
		now:        time.Now,
		theme:      theme.DarkTheme(),
		width:      defaultWidth,
		stream:     streamtext.New(),
	}
	m.prompt = textarea.New()
	m.prompt.Placeholder = "Ask the agent (paste is fine, Enter sends)"
	m.prompt.Focus()
	m.layout()
	return m
}

// layout applies the current width to the widgets that wrap.
func (m *model) layout() {
	m.prompt.Width = m.width - 2
	m.prompt.SoftWrap = true
	m.prompt.Height = 4
	m.stream.Width = m.width - 2
}

func (m model) Init() tui.Cmd { return nil }

// message renders one chat message, markdown formatted, for the scrollback.
func (m model) message(s widgets.Sender, text string, streaming bool) string {
	body := markdown.Render(text, m.width-2, m.theme)
	return widgets.ChatMessage(s, "", body, m.now(), streaming, m.theme)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		if msg.Width > 10 {
			m.width = msg.Width
		}
		m.layout()
		return m, nil

	case tui.PasteEvent:
		// Paste safety: always the prompt, never the approval widget, so a
		// paste containing "y", "yes" or newlines cannot answer the call.
		var cmd tui.Cmd
		m.prompt, cmd = m.prompt.Update(msg)
		return m, cmd

	case tui.Key:
		return m.key(msg)

	case replyMsg:
		if msg.err != nil {
			m.busy = false
			return m, tui.Println(m.message(widgets.SenderError, msg.err.Error(), false))
		}
		m.script, m.next, m.tool = msg.reply.tokens, 0, msg.reply.tool
		m.stream = streamtext.New()
		m.stream.Motion = m.motionPref()
		m.stream.Start() // nothing to reveal yet, so no Cmd; Append schedules the ticks
		m.layout()
		return m, m.scheduleToken()

	case tokenMsg:
		return m.token()

	case toolapproval.ResolvedMsg:
		return m.resolved(msg)
	}
	var cmd tui.Cmd
	m.stream, cmd = m.stream.Update(msg)
	return m, cmd
}

func (m model) motionPref() motion.Preference {
	if m.tokenEvery == 0 {
		return motion.Reduced // deterministic: no reveal animation in tests
	}
	return motion.Normal
}

func (m model) scheduleToken() tui.Cmd {
	if m.tokenEvery <= 0 {
		return nil
	}
	return tui.FromCtx(motion.After(m.tokenEvery, func(time.Time) tui.Msg { return tokenMsg{} }))
}

// token appends the next scripted token. When the script is exhausted the
// finished message is committed to scrollback and the tool gate (if any) opens.
func (m model) token() (tui.Model, tui.Cmd) {
	if !m.busy || m.script == nil {
		return m, nil
	}
	if m.next < len(m.script) {
		cmd := m.stream.Append(m.script[m.next])
		m.next++
		return m, tui.Batch(cmd, m.scheduleToken())
	}
	m.stream.Skip()
	text := m.stream.Text()
	m.script = nil
	m.stream = streamtext.New()
	m.layout()
	commit := tui.Println(m.message(widgets.SenderAssistant, text, false))
	if m.tool != nil {
		m.pending = true
		m.approve = m.newApproval()
	} else {
		m.busy = false
	}
	return m, commit
}

func (m model) newApproval() toolapproval.Model {
	a := toolapproval.New(m.tool.name, m.tool.description, m.tool.risk)
	a.Theme = m.theme
	return a
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.pending {
		// Only a real key press answers the call: y / Enter allow, n / Esc
		// deny, Left/Right/Tab move the highlight. Other keys edit the prompt.
		switch {
		case k.Type == tui.KeyRunes && k.Text == "y":
			return m.answer(false)
		case k.Type == tui.KeyRunes && k.Text == "n", k.Type == tui.KeyEsc:
			return m.answer(true)
		case k.Type == tui.KeyEnter, k.Type == tui.KeyTab, k.Type == tui.KeyLeft, k.Type == tui.KeyRight:
			var cmd tui.Cmd
			m.approve, cmd = m.approve.Update(k)
			return m, cmd
		}
		var cmd tui.Cmd
		m.prompt, cmd = m.prompt.Update(k)
		return m, cmd
	}
	switch {
	case k.Type == tui.KeyEsc:
		return m, tui.Quit()
	case k.Type == tui.KeyEnter:
		text := strings.TrimSpace(m.prompt.Value())
		if text == "" || m.busy {
			return m, nil
		}
		m.prompt.Reset()
		m.busy = true
		return m, tui.Batch(
			tui.Println(m.message(widgets.SenderUser, text, false)),
			tui.Go(m.fake.call(text)),
		)
	}
	var cmd tui.Cmd
	m.prompt, cmd = m.prompt.Update(k)
	return m, cmd
}

// answer resolves the approval with a synthetic Enter on the widget; deny
// first moves the highlight from Approve to Deny.
func (m model) answer(deny bool) (tui.Model, tui.Cmd) {
	if deny {
		m.approve, _ = m.approve.Update(tui.Key{Type: tui.KeyTab})
	} else {
		m.approve = m.newApproval() // highlight back on Approve
	}
	var cmd tui.Cmd
	m.approve, cmd = m.approve.Update(tui.Key{Type: tui.KeyEnter})
	return m, cmd
}

// resolved commits the outcome to scrollback and finishes the turn.
func (m model) resolved(r toolapproval.ResolvedMsg) (tui.Model, tui.Cmd) {
	tool := m.tool
	m.pending, m.busy, m.tool = false, false, nil
	if tool == nil {
		return m, nil
	}
	if r.Choice == toolapproval.ChoiceDeny {
		return m, tui.Println(m.message(widgets.SenderSystem, "Denied `"+tool.name+"`.", false))
	}
	return m, tui.Println(m.message(widgets.SenderSystem, "Ran `"+tool.name+"`:\n\n```\n"+tool.output+"\n```", false))
}

// View is the live region: only what has not been committed yet.
func (m model) View() string {
	var parts []string
	switch {
	case m.pending:
		parts = append(parts, m.approve.View(), "Enter/y allow, n/Esc deny, Tab cycles")
	case m.busy && m.script != nil:
		parts = append(parts, widgets.ChatMessage(widgets.SenderAssistant, "", m.stream.View(), m.now(), true, m.theme))
	case m.busy:
		parts = append(parts, "thinking...")
	}
	parts = append(parts, "> "+strings.ReplaceAll(m.prompt.View(), "\n", "\n  "))
	return strings.Join(parts, "\n")
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
