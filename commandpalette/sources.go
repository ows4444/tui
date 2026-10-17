package commandpalette

import (
	"context"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
)

// Source supplies commands that depend on what was typed: open any file, jump
// to any symbol, search a server. It is called with the text in the input on
// a goroutine of its own, each time that text changes, and the commands it
// returns are listed after the ones from Model.Commands that match.
//
// ctx is cancelled when a newer query supersedes this one (and when the
// Program ends), so a slow Source should stop early when ctx.Done() fires. It
// need not filter by fuzzy match itself the way Commands are: it is trusted to
// return what matches the query, best first. It must not touch the Model.
type Source func(ctx context.Context, query string) []Command

// sourceMsg is what a source's task returns: the query it answered and what it
// found, for the Model to keep if the query is still the input's.
type sourceMsg struct {
	name  string
	index int
	query string
	found []Command
}

type sourceResult struct {
	query string
	found []Command
}

func (m Model) taskKey(i int) string { return "commandpalette:" + m.Name + ":" + strconv.Itoa(i) }

// restart starts every Source on the new query, cancelling the runs for the
// older one, or only cancels them when the query is blank. It returns the
// Model (which may have made its Tasks) and the Cmd that runs them.
func (m Model) restart() (Model, tui.Cmd) {
	if len(m.Sources) == 0 {
		return m, nil
	}
	if m.Tasks == nil {
		m.Tasks = tui.NewTasks()
	}
	query := m.Input.Value()
	if strings.TrimSpace(query) == "" {
		for i := range m.Sources {
			m.Tasks.Cancel(m.taskKey(i))
		}
		m.results = nil
		return m, nil
	}
	cmds := make([]tui.Cmd, 0, len(m.Sources))
	for i, src := range m.Sources {
		if src == nil {
			continue
		}
		name := m.Name
		cmds = append(cmds, m.Tasks.Start(m.taskKey(i), func(ctx context.Context) tui.Msg {
			return sourceMsg{name: name, index: i, query: query, found: src(ctx, query)}
		}))
	}
	return m, tui.Batch(cmds...)
}

// taskDone keeps a source's answer when it is for this palette and still for
// the text in the input; any other TaskDoneMsg is not the palette's and is
// left alone.
func (m Model) taskDone(d tui.TaskDoneMsg) (Model, bool) {
	sm, ok := d.Msg.(sourceMsg)
	if !ok || sm.name != m.Name {
		return m, false
	}
	if d.Canceled || sm.query != m.Input.Value() {
		return m, true
	}
	next := make(map[int]sourceResult, len(m.results)+1)
	for k, v := range m.results {
		next[k] = v
	}
	next[sm.index] = sourceResult{query: sm.query, found: sm.found}
	m.results = next
	return m, true
}

// fromSources lists what the sources found for query, in source order.
func (m Model) fromSources(query string) []Command {
	var out []Command
	for i := range m.Sources {
		if r, ok := m.results[i]; ok && r.query == query {
			out = append(out, r.found...)
		}
	}
	return out
}
