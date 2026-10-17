# Tables

`datatable` is a table with a cursor, a sort, a filter, cell editing and a
window onto the rows that costs the rows shown, not the rows held.

```go
m := datatable.New([]string{"Name", "Size"}, rows) // rows is [][]string
m.Height = 10
```

Forward messages to `m.Update`; handle `datatable.SelectedMsg{Row, SourceRow,
Cells, Col}` when Enter confirms the row under the cursor. `m.Cursor()` is the
shown row, `m.SourceRow(i)` maps a shown row back to the slice you gave in
(they differ while sorted or filtered).

## Columns

`Model.Columns[i]` shapes column `i`:

```go
m.Columns = []datatable.Column{
	{Max: 24},                          // cut with an ellipsis past 24 cells
	{Align: layout.AlignEnd, Min: 8,    // numbers flush right
		Format: humanBytes,             // shown as "1.2 MB"; the cell stays "1234567"
		Less:   byteCount},             // sorted by the number
}
```

`Format` changes what is drawn, spoken and clipped, not the cell that sorting,
filtering and `SelectedMsg` use. `ColumnWidths` fixes widths for the layout node.

## Sort, filter, edit

- `s` sorts by the current column (numbers as numbers, or `Column.Less`); `S`
  restores the order. `SortBy(col, desc)` does it from code.
- With `Filter` on, `/` opens a filter line: typed text keeps the rows whose
  cells fuzzy-match; Enter keeps the filter, Esc clears it.
- `SetCell`, `InsertRow` and `RemoveRow` edit the source rows on a copy and apply
  the sort and filter again.

## Marking rows

With `Multi` on, Space marks the row under the cursor and Ctrl+A marks all shown
(or unmarks them). A marked row shows a check in the gutter. `Marked()` returns
the marked source rows, ascending; marks stay with their rows through sort,
filter and row edits.

## Cell cursor, styles, frozen columns

`CellMode` makes the cursor one cell; `CellStyle(row, col, value)` styles cells;
`FrozenCols` keeps leading columns in view while the others scroll when `Width`
is smaller than the table.

## Big tables: Source

For a table over a database cursor, a log or a million rows, give it a source
instead of a slice:

```go
type rows struct{ db *DB }
func (r rows) Rows() int           { return r.db.Count() }
func (r rows) Row(i int) []string  { return r.db.Fetch(i) }

m := datatable.NewSource([]string{"ID", "Name"}, rows{db})
m.Height = 20
```

Only the rows near the window are asked for. Sort, filter and editing need all
the rows in hand, so they are off for a source. `SetSource` swaps it; `SetRows`
goes back to in-memory rows. See `examples/table` and `examples/asyncload`.
