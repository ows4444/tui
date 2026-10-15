// Package fuzzy ranks text against a short query typed by a person: every
// rune of the query must appear in the text in order, not necessarily next to
// each other, so "cp" finds "CommandPalette". It has no state and no
// dependencies; the filtering widgets (commandpalette, picker, multiselect,
// treeview) use it so that they rank and highlight the same way.
//
// Matching ignores case. A match scores higher when its runes are
// consecutive, when they start a word (after a space, a punctuation mark, or
// at a lower-to-upper case change) and when it begins near the front of the
// text; it scores lower for each rune skipped between two matched runes.
//
// Stability: experimental. Its API may change in any minor release.
package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

// Scoring weights. They are not exported: callers get an ordering, and the
// absolute numbers carry no meaning outside one query.
const (
	bonusBoundary    = 10 // matched rune starts a word
	bonusConsecutive = 6  // matched rune directly follows the previous one
	bonusFirst       = 4  // matched rune is the first of the text
	penaltyGap       = 1  // per rune skipped between two matched runes
	penaltyLeading   = 1  // per rune before the first match
	leadingCap       = 6  // leading penalty stops growing here
	maxDP            = 512
)

// Result is one text that matched.
type Result struct {
	// Index is the position of the text in the input of Find.
	Index int
	// Score orders results: higher is a better match. Only comparable
	// between results of one query.
	Score int
	// Positions are the rune offsets (not byte offsets) in the text of the
	// matched runes, ascending, one per rune of the query.
	Positions []int
}

// Score matches query against text. ok is false when query is not a
// subsequence of text; an empty query matches everything with score 0 and no
// positions. positions are rune offsets into text, one per query rune.
func Score(query, text string) (score int, positions []int, ok bool) {
	q := fold([]rune(query))
	if len(q) == 0 {
		return 0, nil, true
	}
	t := []rune(text)
	if len(q) > len(t) {
		return 0, nil, false
	}
	tf := fold(t)
	if !subsequence(q, tf) {
		return 0, nil, false
	}
	if len(t) > maxDP {
		// Too long for the table: take the leftmost match. It is still a
		// valid match, only not the best-scoring one.
		return greedy(q, tf)
	}
	return best(q, t, tf)
}

// Find returns the entries of 0..n-1 whose text(i) matches query, best first.
// Equal scores keep the shorter text first, then the original order. An empty
// query returns every index in order with score 0.
func Find(query string, n int, text func(i int) string) []Result {
	out := make([]Result, 0, n)
	lens := make([]int, 0, n)
	for i := 0; i < n; i++ {
		s := text(i)
		sc, pos, ok := Score(query, s)
		if !ok {
			continue
		}
		out = append(out, Result{Index: i, Score: sc, Positions: pos})
		lens = append(lens, len([]rune(s)))
	}
	if strings.TrimSpace(query) == "" {
		return out
	}
	idx := make([]int, len(out))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		x, y := idx[a], idx[b]
		if out[x].Score != out[y].Score {
			return out[x].Score > out[y].Score
		}
		return lens[x] < lens[y]
	})
	sorted := make([]Result, len(out))
	for i, k := range idx {
		sorted[i] = out[k]
	}
	return sorted
}

// Run is a stretch of text that is all matched or all unmatched.
type Run struct {
	Text  string
	Match bool
}

// Split cuts text into alternating runs of unmatched and matched runes, in
// order, so a widget can style each kind itself (a cursor row styles the
// unmatched runs too, where Highlight leaves them alone). positions are rune
// offsets, as Result.Positions has them; out-of-range offsets are ignored.
// Joining the runs' Text gives text back.
func Split(text string, positions []int) []Run {
	r := []rune(text)
	if len(r) == 0 {
		return nil
	}
	on := make([]bool, len(r))
	for _, p := range positions {
		if p >= 0 && p < len(r) {
			on[p] = true
		}
	}
	var runs []Run
	for i := 0; i < len(r); {
		j := i
		for j < len(r) && on[j] == on[i] {
			j++
		}
		runs = append(runs, Run{Text: string(r[i:j]), Match: on[i]})
		i = j
	}
	return runs
}

// Highlight returns text with each matched run passed through mark and the
// rest left as it is. Adjacent matched runes are given to mark together, so a
// style opens and closes once per run. A nil mark returns text unchanged.
func Highlight(text string, positions []int, mark func(string) string) string {
	if len(positions) == 0 || mark == nil {
		return text
	}
	var b strings.Builder
	for _, run := range Split(text, positions) {
		if run.Match {
			b.WriteString(mark(run.Text))
		} else {
			b.WriteString(run.Text)
		}
	}
	return b.String()
}

// fold lower-cases runes one for one, so offsets into the folded slice are
// offsets into the original.
func fold(r []rune) []rune {
	out := make([]rune, len(r))
	for i, c := range r {
		out[i] = unicode.ToLower(c)
	}
	return out
}

func subsequence(q, t []rune) bool {
	qi := 0
	for _, c := range t {
		if c == q[qi] {
			qi++
			if qi == len(q) {
				return true
			}
		}
	}
	return false
}

func greedy(q, t []rune) (int, []int, bool) {
	pos := make([]int, 0, len(q))
	qi := 0
	for i, c := range t {
		if c == q[qi] {
			pos = append(pos, i)
			qi++
			if qi == len(q) {
				return 0, pos, true
			}
		}
	}
	return 0, nil, false
}

// bonus is the extra score for matching the rune at i of the original text.
func bonus(t []rune, i int) int {
	if i == 0 {
		return bonusBoundary + bonusFirst
	}
	prev, cur := t[i-1], t[i]
	switch {
	case !unicode.IsLetter(prev) && !unicode.IsDigit(prev):
		return bonusBoundary
	case unicode.IsLower(prev) && unicode.IsUpper(cur):
		return bonusBoundary
	case unicode.IsLetter(prev) && unicode.IsDigit(cur):
		return bonusBoundary / 2
	}
	return 0
}

const unreachable = -1 << 30

// best finds the highest-scoring placement of q in t: dp[i][j] is the best
// score with q[i] matched at t[j]. The gap penalty is linear, so the best
// earlier match for each j is a running maximum, which keeps this O(len(q) *
// len(t)).
func best(q, t, tf []rune) (int, []int, bool) {
	m, n := len(q), len(t)
	dp := make([][]int, m)
	from := make([][]int, m)
	for i := range dp {
		dp[i] = make([]int, n)
		from[i] = make([]int, n)
		for j := range dp[i] {
			dp[i][j] = unreachable
			from[i][j] = -1
		}
	}
	for j := 0; j < n; j++ {
		if tf[j] == q[0] {
			lead := j * penaltyLeading
			if lead > leadingCap {
				lead = leadingCap
			}
			dp[0][j] = bonus(t, j) - lead
		}
	}
	for i := 1; i < m; i++ {
		// run is max over k < j-1 of dp[i-1][k] + k*penaltyGap, with its k.
		run, runK := unreachable, -1
		for j := i; j < n; j++ {
			if k := j - 2; k >= 0 && dp[i-1][k] != unreachable {
				if v := dp[i-1][k] + k*penaltyGap; v > run {
					run, runK = v, k
				}
			}
			if tf[j] != q[i] {
				continue
			}
			bestV, bestK := unreachable, -1
			if p := dp[i-1][j-1]; p != unreachable {
				bestV, bestK = p+bonusConsecutive, j-1
			}
			if runK >= 0 {
				if v := run - (j-1)*penaltyGap; v > bestV {
					bestV, bestK = v, runK
				}
			}
			if bestK >= 0 {
				dp[i][j] = bestV + bonus(t, j)
				from[i][j] = bestK
			}
		}
	}
	bj, bv := -1, unreachable
	for j := 0; j < n; j++ {
		if dp[m-1][j] > bv {
			bv, bj = dp[m-1][j], j
		}
	}
	if bj < 0 {
		return 0, nil, false
	}
	pos := make([]int, m)
	for i, j := m-1, bj; i >= 0; i-- {
		pos[i] = j
		j = from[i][j]
	}
	return bv, pos, true
}
