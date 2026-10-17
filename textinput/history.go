package textinput

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/keymap"
)

// historyState is where the person is in History. The zero value is "not
// browsing".
type historyState struct {
	browsing bool
	idx      int    // index into History while browsing
	draft    string // what was typed before the first recall
	// search is the text a search matches, set by the first HistorySearch press
	// and kept until another key ends it.
	searching bool
	needle    string
	from      int // the History index the last search stopped at
}

// historyKeys are the bindings History and suggestions add to KeyMap, defaulted
// when a custom KeyMap leaves them empty.
func (km KeyMap) withHistoryDefaults() KeyMap {
	d := DefaultKeyMap()
	if len(km.HistoryPrev.Keys) == 0 {
		km.HistoryPrev = d.HistoryPrev
	}
	if len(km.HistoryNext.Keys) == 0 {
		km.HistoryNext = d.HistoryNext
	}
	if len(km.HistorySearch.Keys) == 0 {
		km.HistorySearch = d.HistorySearch
	}
	if len(km.AcceptSuggestion.Keys) == 0 {
		km.AcceptSuggestion = d.AcceptSuggestion
	}
	return km
}

// historyBindings are the history and suggestion bindings in force, for help
// text: only those that do something now.
func (m Model) historyBindings() []keymap.Binding {
	km := m.keys()
	var out []keymap.Binding
	if len(m.History) > 0 {
		out = append(out, km.HistoryPrev, km.HistoryNext, km.HistorySearch)
	}
	if m.suggestionRest() != "" {
		out = append(out, km.AcceptSuggestion)
	}
	return out
}

// handleHistoryKey applies the history and suggestion keys, and reports whether
// k was one. Any other key ends a search and, for an edit, leaves history
// browsing (the recalled text is the value now).
func (m *Model) handleHistoryKey(k tui.Key, km KeyMap) bool {
	switch {
	case len(m.History) > 0 && keyHit(k, km.HistoryPrev):
		m.recall(-1)
		return true
	case len(m.History) > 0 && keyHit(k, km.HistoryNext):
		m.recall(+1)
		return true
	case len(m.History) > 0 && keyHit(k, km.HistorySearch):
		m.searchHistory()
		return true
	case m.ed.Cursor() == m.ed.Len() && m.suggestionRest() != "" && keyHit(k, km.AcceptSuggestion):
		m.insert(m.suggestionRest())
		m.hist = historyState{}
		return true
	}
	m.hist.searching = false
	return false
}

// recall moves through History: dir -1 to an older entry, +1 to a newer one or
// back to what was being typed.
func (m *Model) recall(dir int) {
	h := &m.hist
	h.searching = false
	n := len(m.History)
	if !h.browsing {
		if dir > 0 {
			return
		}
		h.browsing, h.idx, h.draft = true, n, m.Value()
	}
	h.idx = min(max(h.idx+dir, 0), n)
	if h.idx == n {
		m.ed.Set(h.draft, m.CharLimit)
		h.browsing = false
		return
	}
	m.ed.Set(m.History[h.idx], m.CharLimit)
}

// searchHistory finds the newest entry, older than the last one found, that
// contains what had been typed when the search began.
func (m *Model) searchHistory() {
	h := &m.hist
	if !h.searching {
		h.searching, h.needle, h.from = true, m.Value(), len(m.History)
		if !h.browsing {
			h.draft = m.Value()
		}
	}
	for i := h.from - 1; i >= 0; i-- {
		if strings.Contains(m.History[i], h.needle) {
			h.from, h.browsing, h.idx = i, true, i
			m.ed.Set(m.History[i], m.CharLimit)
			return
		}
	}
	// Nothing older: stay on the last match, or on the draft if there was none.
}

// suggestionRest is the part of the suggestion after what is typed, or "" when
// there is none to show: the field is not at the end of a non-empty value, or
// the value is not the start of the suggestion.
func (m Model) suggestionRest() string {
	v := m.Value()
	if !m.focused || v == "" || m.ed.Cursor() != m.ed.Len() {
		return ""
	}
	cand := m.Suggestion
	if m.SuggestFromHistory {
		cand = ""
		for i := len(m.History) - 1; i >= 0; i-- {
			if len(m.History[i]) > len(v) && strings.HasPrefix(m.History[i], v) {
				cand = m.History[i]
				break
			}
		}
	}
	if len(cand) <= len(v) || !strings.HasPrefix(cand, v) {
		return ""
	}
	return singleLine(cand[len(v):])
}

// ghost renders the suggestion after the cursor in the placeholder style,
// cut to the room left when Width is set. used is the columns already drawn
// after the prompt.
func (m Model) ghost(used int) string {
	rest := m.suggestionRest()
	if rest == "" {
		return ""
	}
	if m.Width > 0 {
		rest = ansi.Truncate(rest, max(m.Width-used-1, 0))
	}
	return m.PlaceholderStyle.Render(rest)
}
