package datatable

import (
	"sync"

	"github.com/ows4444/tui/ansi"
)

// widthCache memoises natural column widths so a frame does not rescan every
// row. It is keyed on the identity (first-element address and length) of the
// Headers and Rows slices, so replacing or resizing either invalidates it;
// editing a cell in place does not, which is what SetRows is for. A Model
// built as a struct literal has no cache and rescans each time; use New.
type widthCache struct {
	mu      sync.Mutex
	rows    *[]string
	nrows   int
	hdr     *string
	nhdr    int
	widths  []int
	clean   bool // no scanned string needed ansi.Clean
	present bool
}

// scanHook, when non-nil, is called with the number of rows scanned by each
// natural-width computation. Tests use it to prove the cache; it is nil in
// production.
var scanHook func(rows int)

func rowsKey(rows [][]string) *[]string {
	if len(rows) == 0 {
		return nil
	}
	return &rows[0]
}

func hdrKey(h []string) *string {
	if len(h) == 0 {
		return nil
	}
	return &h[0]
}

// rekey re-points the cache at a reordered copy of the same rows.
func (c *widthCache) rekey(rows [][]string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rows, c.nrows = rowsKey(rows), len(rows)
}

// measure returns the natural width of every column, scanning all rows only
// when the cache is absent or stale. The result must not be modified.
func (m Model) measure() []int {
	w, _ := m.measureClean()
	return w
}

// measureClean is measure that also reports whether every header and cell is
// clean, so that its raw width is the width ansi.Clean text has: DrawCells
// serves a full-table window from the cache only then.
func (m Model) measureClean() ([]int, bool) {
	c := m.widths
	if c == nil {
		return scanWidths(m.Headers, m.Rows)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.present && c.rows == rowsKey(m.Rows) && c.nrows == len(m.Rows) &&
		c.hdr == hdrKey(m.Headers) && c.nhdr == len(m.Headers) {
		return c.widths, c.clean
	}
	c.widths, c.clean = scanWidths(m.Headers, m.Rows)
	c.rows, c.nrows = rowsKey(m.Rows), len(m.Rows)
	c.hdr, c.nhdr = hdrKey(m.Headers), len(m.Headers)
	c.present = true
	return c.widths, c.clean
}

func scanWidths(headers []string, rows [][]string) ([]int, bool) {
	if scanHook != nil {
		scanHook(len(rows))
	}
	clean := true
	w := make([]int, len(headers))
	for i, h := range headers {
		w[i] = ansi.Width(h)
		clean = clean && ansi.Clean(false, h) == h
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(w) {
				if cw := ansi.Width(cell); cw > w[i] {
					w[i] = cw
				}
				clean = clean && ansi.Clean(false, cell) == cell
			}
		}
	}
	return w, clean
}
