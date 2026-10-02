// Package toolapproval is a gate-before-execution prompt for an
// agent/CLI tool call — InkUI's "ToolApproval". It generalizes
// confirm.Model's two-option (Yes/No) highlight-cycling to three options
// (Approve/Deny/AlwaysAllow), adds a risk badge, and layers on toast.Model's
// motion-driven, id-guarded auto-timeout to auto-deny if the caller never
// answers.
//
// Stability: experimental. Its API may change in any minor release.
package toolapproval

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Risk is how risky the tool call being approved is judged to be — it
// drives View's badge color.
type Risk int

const (
	// RiskLow is the zero value: a LOW badge in the Success colour.
	RiskLow Risk = iota
	// RiskMedium shows a MEDIUM badge in the Warning colour.
	RiskMedium
	// RiskHigh shows a HIGH badge in the Error colour.
	RiskHigh
)

// variant maps a Risk to the widgets.Variant (and so the Theme color) its
// badge renders with, keeping the three levels visually distinct.
func (r Risk) variant() widgets.Variant {
	switch r {
	case RiskMedium:
		return widgets.VariantWarning
	case RiskHigh:
		return widgets.VariantError
	default: // RiskLow
		return widgets.VariantSuccess
	}
}

// label is the risk badge's text.
func (r Risk) label() string {
	switch r {
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	default: // RiskLow
		return "LOW"
	}
}

// Choice is which of the three options a ToolApproval prompt resolves to.
type Choice int

const (
	// ChoiceApprove ("Approve") lets this one tool call run. It is the zero
	// value.
	ChoiceApprove Choice = iota
	// ChoiceDeny ("Deny") refuses the call; it is also what an unanswered
	// prompt resolves to when Timeout elapses.
	ChoiceDeny
	// ChoiceAlwaysAllow ("Always Allow") is the cue to approve this call and
	// stop prompting for it in future. The widget only reports it in
	// ResolvedMsg; it keeps no allow-list, so remembering the choice is the
	// caller's job.
	ChoiceAlwaysAllow
)

// label is a Choice's display text in View.
func (c Choice) label() string {
	switch c {
	case ChoiceDeny:
		return "Deny"
	case ChoiceAlwaysAllow:
		return "Always Allow"
	default: // ChoiceApprove
		return "Approve"
	}
}

// Model is a three-option Approve/Deny/AlwaysAllow gate shown before a
// tool call runs: Left/Right/Tab move which option is highlighted, Enter
// confirms the highlighted one, and — if Timeout elapses first — it
// auto-resolves as Deny, the safe default for an unanswered prompt.
type Model struct {
	ToolName    string
	Description string
	Risk        Risk
	// Timeout, if non-zero, is how long Start waits before auto-resolving
	// as ChoiceDeny if no manual choice has been made yet.
	Timeout time.Duration
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on an
	// option label resolves the prompt with that choice. Off (the default)
	// ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the prompt (its first
	// row is the tool name line); clicks outside it are ignored.
	Bounds hittest.Rect

	highlighted Choice // which option is currently highlighted
	// id disambiguates Start calls the same way toast.Model's id
	// disambiguates Show calls: each Start bumps id, and a timeoutMsg
	// carrying a stale id (from a Start a later Start has since replaced)
	// is ignored by Update instead of resolving a prompt it no longer
	// belongs to.
	id int
}

// New returns a Model with Approve highlighted by default — the common
// case where the caller expects most prompts to be approved.
func New(toolName, description string, risk Risk) Model {
	return Model{
		ToolName:    toolName,
		Description: description,
		Risk:        risk,
		Theme:       theme.DarkTheme(),
		KeyMap:      DefaultKeyMap(),
		highlighted: ChoiceApprove,
	}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Next   keymap.Binding // highlight the next option, wrapping
	Prev   keymap.Binding // highlight the previous option, wrapping
	Accept keymap.Binding // resolve with the highlighted option
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Next:   keymap.NewBinding("next option", "right", "tab"),
		Prev:   keymap.NewBinding("previous option", "left"),
		Accept: keymap.NewBinding("confirm", "enter"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Next.Keys)+len(km.Prev.Keys)+len(km.Accept.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the prompt currently honours, with
// descriptions, for help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Next, km.Prev, km.Accept}
}

// Highlighted reports which option the cursor is currently on — not which
// was chosen; that only happens via ResolvedMsg.
func (m Model) Highlighted() Choice { return m.highlighted }

// ResolvedMsg is delivered (via the Cmd Update returns) once the prompt is
// answered, by Enter or by Timeout elapsing.
type ResolvedMsg struct {
	Choice Choice
}

// timeoutMsg is Start's auto-deny timer firing, id-guarded the same way
// toast's dismissMsg is: a stale timeoutMsg (from a Start a later Start
// has since replaced, or after a manual choice was already made) is
// ignored instead of resolving the prompt again.
type timeoutMsg struct{ id int }

// Start begins (or restarts) the auto-deny timer, returning a Cmd that
// delivers a timeoutMsg guarded by the generation current at the time —
// return this from your own Init/Update so the timeout actually fires.
func (m *Model) Start() tui.Cmd {
	m.id++
	id := m.id
	return tui.FromCtx(motion.After(m.Timeout, func(time.Time) tui.Msg { return timeoutMsg{id: id} }))
}

// resolved reports whether this Model has already been answered (manually
// or by timeout) and so should ignore any further timeoutMsg.
func (m Model) resolved() bool { return m.id == 0 }

// Update moves the highlight on Left/Right/Tab (wrapping across the three
// choices), confirms the highlighted option on Enter, and auto-resolves as
// ChoiceDeny when a timeoutMsg whose id matches the current generation
// arrives (and, with Mouse on, resolves with the option a left click lands on);
// a stale timeoutMsg (superseded by a later Start, or arriving
// after a manual choice already resolved this generation) is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.MouseEvent:
		return m.updateMouse(msg)
	case tui.Key:
		km := m.keys()
		switch {
		case keymap.Matches(msg, km.Next):
			m.highlighted = (m.highlighted + 1) % 3
		case keymap.Matches(msg, km.Prev):
			m.highlighted = (m.highlighted + 3 - 1) % 3
		case keymap.Matches(msg, km.Accept):
			choice := m.highlighted
			m.id = 0 // mark resolved so a later stale timeoutMsg is ignored
			return m, func() tui.Msg { return ResolvedMsg{Choice: choice} }
		}
	case timeoutMsg:
		if m.resolved() || msg.id != m.id {
			return m, nil
		}
		m.id = 0 // mark resolved
		return m, func() tui.Msg { return ResolvedMsg{Choice: ChoiceDeny} }
	}
	return m, nil
}

// updateMouse resolves the prompt when a left press lands on an option label
// (its padding or brackets included) in the options row, the last line of View.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft ||
		!m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	rows := 1 + strings.Count(m.ToolName, "\n")
	if m.Description != "" {
		rows += 1 + strings.Count(m.Description, "\n")
	}
	if ly != rows {
		return m, nil
	}
	start := 0
	for _, c := range []Choice{ChoiceApprove, ChoiceDeny, ChoiceAlwaysAllow} {
		end := start + ansi.Width(c.label()) + 2
		if lx >= start && lx < end {
			m.highlighted = c
			m.id = 0 // mark resolved so a later stale timeoutMsg is ignored
			return m, func() tui.Msg { return ResolvedMsg{Choice: c} }
		}
		start = end + 2
	}
	return m, nil
}

// View renders the tool name/description, a risk badge colored per Risk,
// and the three options with the highlighted one shown in reverse video.
func (m Model) View() string {
	badge := widgets.Badge(m.Risk.label(), m.Risk.variant(), m.themed())

	highlight := m.themed().ResolvedStates().Selected.Bold()
	options := make([]string, 3)
	for i, c := range []Choice{ChoiceApprove, ChoiceDeny, ChoiceAlwaysAllow} {
		label := " " + c.label() + " "
		if c == m.highlighted {
			// Brackets replace the padding: visible without colour, same width.
			label = highlight.Render("[" + c.label() + "]")
		}
		options[i] = label
	}

	header := m.ToolName + "  " + badge
	if m.Description != "" {
		header += "\n" + m.Description
	}
	return header + "\n" + options[0] + "  " + options[1] + "  " + options[2]
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the prompt as plain text for accessible output (see
// tui.Linearizer): "Approval needed: <tool>, <low|medium|high> risk", the
// description if there is one, then one line per option with its position and
// ", selected" on the highlighted one, and, if Timeout is set, a final line
// saying it denies automatically when unanswered.
func (m Model) Linearize() string {
	risk := map[Risk]string{RiskLow: "low", RiskMedium: "medium", RiskHigh: "high"}[m.Risk]
	out := []string{"Approval needed: " + m.ToolName + ", " + risk + " risk"}
	if m.Description != "" {
		out = append(out, m.Description)
	}
	for i, c := range []Choice{ChoiceApprove, ChoiceDeny, ChoiceAlwaysAllow} {
		line := c.label() + ", option " + strconv.Itoa(i+1) + " of 3"
		if c == m.highlighted {
			line += ", selected"
		}
		out = append(out, line)
	}
	if m.Timeout > 0 {
		out = append(out, "Denies automatically after "+m.Timeout.String()+" if unanswered")
	}
	return strings.Join(out, "\n")
}
