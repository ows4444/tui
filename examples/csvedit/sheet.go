package main

import (
	"encoding/csv"
	"errors"
	"sort"
	"strconv"
	"strings"
)

// sheet is the table being edited: a header row and the rows under it. Every
// row has exactly len(headers) cells; the methods keep it so. Methods return a
// new sheet and never change the receiver, which is what makes undo a stack of
// sheets.
type sheet struct {
	headers []string
	rows    [][]string
}

func (s sheet) cols() int { return len(s.headers) }

func (s sheet) clone() sheet {
	out := sheet{headers: append([]string(nil), s.headers...), rows: make([][]string, len(s.rows))}
	for i, r := range s.rows {
		out.rows[i] = append([]string(nil), r...)
	}
	return out
}

// parseCSV reads text as CSV. The first record is the header; short records
// are padded and long ones cut to its width, so a ragged file still loads.
func parseCSV(text string) (sheet, error) {
	r := csv.NewReader(strings.NewReader(text))
	r.FieldsPerRecord = -1
	recs, err := r.ReadAll()
	if err != nil {
		return sheet{}, err
	}
	if len(recs) == 0 {
		return sheet{}, errors.New("the file is empty")
	}
	s := sheet{headers: recs[0]}
	if len(s.headers) == 0 {
		return sheet{}, errors.New("the first row has no columns")
	}
	for _, rec := range recs[1:] {
		row := make([]string, s.cols())
		copy(row, rec)
		s.rows = append(s.rows, row)
	}
	return s, nil
}

func (s sheet) encode() string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	_ = w.Write(s.headers)
	_ = w.WriteAll(s.rows) // WriteAll flushes; a strings.Builder cannot fail
	return b.String()
}

func (s sheet) set(row, col int, text string) sheet {
	s = s.clone()
	if row < 0 {
		s.headers[col] = text
	} else {
		s.rows[row][col] = text
	}
	return s
}

func (s sheet) cell(row, col int) string {
	if row < 0 {
		return s.headers[col]
	}
	return s.rows[row][col]
}

// insertRow puts an empty row at index i (0 to len(rows)).
func (s sheet) insertRow(i int) sheet {
	s = s.clone()
	s.rows = append(s.rows, nil)
	copy(s.rows[i+1:], s.rows[i:])
	s.rows[i] = make([]string, s.cols())
	return s
}

func (s sheet) deleteRow(i int) sheet {
	s = s.clone()
	s.rows = append(s.rows[:i], s.rows[i+1:]...)
	return s
}

// insertCol puts an empty column called name at index i (0 to cols).
func (s sheet) insertCol(i int, name string) sheet {
	s = s.clone()
	ins := func(r []string, v string) []string {
		r = append(r, "")
		copy(r[i+1:], r[i:])
		r[i] = v
		return r
	}
	s.headers = ins(s.headers, name)
	for k := range s.rows {
		s.rows[k] = ins(s.rows[k], "")
	}
	return s
}

// deleteCol removes column i; the last column cannot be removed.
func (s sheet) deleteCol(i int) (sheet, bool) {
	if s.cols() <= 1 {
		return s, false
	}
	s = s.clone()
	cut := func(r []string) []string { return append(r[:i], r[i+1:]...) }
	s.headers = cut(s.headers)
	for k := range s.rows {
		s.rows[k] = cut(s.rows[k])
	}
	return s, true
}

// numeric reports whether every non-empty cell of column c is a number.
func (s sheet) numeric(c int) bool {
	seen := false
	for _, r := range s.rows {
		if strings.TrimSpace(r[c]) == "" {
			continue
		}
		if _, err := strconv.ParseFloat(strings.TrimSpace(r[c]), 64); err != nil {
			return false
		}
		seen = true
	}
	return seen
}

// sorted orders the rows by column c, as numbers when the column is all
// numbers and as text (ignoring case) otherwise. Empty cells go last either
// way, and rows that compare equal keep their order.
func (s sheet) sorted(c int, desc bool) sheet {
	s = s.clone()
	num := s.numeric(c)
	less := func(a, b string) bool {
		a, b = strings.TrimSpace(a), strings.TrimSpace(b)
		if num {
			x, _ := strconv.ParseFloat(a, 64)
			y, _ := strconv.ParseFloat(b, 64)
			return x < y
		}
		return strings.ToLower(a) < strings.ToLower(b)
	}
	sort.SliceStable(s.rows, func(i, j int) bool {
		a, b := s.rows[i][c], s.rows[j][c]
		switch {
		case strings.TrimSpace(a) == "" || strings.TrimSpace(b) == "":
			return strings.TrimSpace(a) != "" && strings.TrimSpace(b) == ""
		case desc:
			return less(b, a)
		}
		return less(a, b)
	})
	return s
}

// colName is the spreadsheet letter of column c: A to Z, then AA and on.
func colName(c int) string {
	s := ""
	for c++; c > 0; c = (c - 1) / 26 {
		s = string(rune('A'+(c-1)%26)) + s
	}
	return s
}
