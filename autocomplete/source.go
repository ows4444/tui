package autocomplete

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ows4444/tui"
)

// Source supplies suggestions that depend on what was typed: the entries of a
// directory, the symbols of a project, the answer of a server. It is called with
// the text in the input on a goroutine of its own each time that text changes;
// what it returns is listed after the Suggestions that match, in the order
// given. ctx is cancelled when newer text supersedes this run (and when the
// Program ends), so a slow Source should stop early. The Source decides what
// matches, so it should return only suggestions that fit the text; accepting
// one replaces the whole input with it. It must not touch the Model.
type Source func(ctx context.Context, text string) []string

// sourceMsg is what the Source's task returns: the text it answered and what it
// found, kept if the input still holds that text.
type sourceMsg struct {
	name  string
	query string
	found []string
}

func (m Model) taskKey() string { return "autocomplete:" + m.Name }

// restart starts the Source on the new text, cancelling the run for the older
// one, or only cancels it when the text is empty.
func (m Model) restart() (Model, tui.Cmd) {
	if m.Source == nil {
		return m, nil
	}
	if m.Tasks == nil {
		m.Tasks = tui.NewTasks()
	}
	text := m.Input.Value()
	if text == "" {
		m.Tasks.Cancel(m.taskKey())
		m.result = nil
		return m, nil
	}
	src, name := m.Source, m.Name
	return m, m.Tasks.Start(m.taskKey(), func(ctx context.Context) tui.Msg {
		return sourceMsg{name: name, query: text, found: src(ctx, text)}
	})
}

// taskDone keeps the Source's answer when it is for this field and still for
// the text in the input; any other TaskDoneMsg is not ours and is left alone.
func (m Model) taskDone(d tui.TaskDoneMsg) (Model, bool) {
	sm, ok := d.Msg.(sourceMsg)
	if !ok || sm.name != m.Name {
		return m, false
	}
	if d.Canceled || sm.query != m.Input.Value() {
		return m, true
	}
	m.result = &sourceResult{query: sm.query, found: sm.found}
	return m, true
}

type sourceResult struct {
	query string
	found []string
}

// fromSource lists what the Source found for text.
func (m Model) fromSource(text string) []string {
	if m.result != nil && m.result.query == text {
		return m.result.found
	}
	return nil
}

// maxPathEntries bounds what PathSource lists from one directory.
const maxPathEntries = 200

// PathSource completes file system paths: for "src/ma" it lists the entries of
// "src/" whose names start with "ma", directories first and with a trailing
// separator, so accepting one and typing on goes down a level. A leading "~/"
// is the home directory. Case is ignored in the prefix. Hidden entries are
// listed only when the prefix starts with a dot. An unreadable directory gives
// no suggestions.
func PathSource() Source {
	return func(ctx context.Context, text string) []string {
		dir, prefix := filepath.Split(text)
		read := dir
		switch {
		case read == "":
			read = "."
		case strings.HasPrefix(read, "~/") || read == "~":
			if home, err := os.UserHomeDir(); err == nil {
				read = filepath.Join(home, strings.TrimPrefix(read, "~"))
			}
		}
		entries, err := os.ReadDir(read)
		if err != nil {
			return nil
		}
		lp := strings.ToLower(prefix)
		var dirs, files []string
		for _, e := range entries {
			if ctx.Err() != nil {
				return nil
			}
			name := e.Name()
			if strings.HasPrefix(name, ".") && !strings.HasPrefix(prefix, ".") {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(name), lp) {
				continue
			}
			if e.IsDir() {
				dirs = append(dirs, dir+name+string(filepath.Separator))
			} else {
				files = append(files, dir+name)
			}
		}
		sort.Strings(dirs)
		sort.Strings(files)
		out := append(dirs, files...)
		if len(out) > maxPathEntries {
			out = out[:maxPathEntries]
		}
		return out
	}
}
