package main

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

type table struct {
	name  string
	cols  []string
	types []string
	rows  [][]value
}

type database struct {
	tables []*table
}

func (d *database) find(name string) *table {
	for _, t := range d.tables {
		if strings.EqualFold(t.name, name) {
			return t
		}
	}
	return nil
}

// result is what a query gives back.
type result struct {
	cols []string
	rows [][]value
}

type colRef struct{ table, name string }

// env is one row of the (joined) source: its columns and their values.
type env struct {
	cols []colRef
	vals []value
}

func (e env) lookup(c colExpr) (value, *sqlError) {
	found := -1
	for i, ref := range e.cols {
		if !strings.EqualFold(ref.name, c.name) || (c.table != "" && !strings.EqualFold(ref.table, c.table)) {
			continue
		}
		if found >= 0 {
			return nil, errAt(c.pos, "the column %s is in both tables: write table.%s", c.name, c.name)
		}
		found = i
	}
	if found < 0 {
		name := c.name
		if c.table != "" {
			name = c.table + "." + c.name
		}
		return nil, errAt(c.pos, "there is no column called %s", name)
	}
	return e.vals[found], nil
}

// query runs one SELECT.
func (d *database) query(q string) (*result, *sqlError) {
	s, err := parse(q)
	if err != nil {
		return nil, err
	}
	envs, err := d.source(s)
	if err != nil {
		return nil, err
	}
	if s.where != nil {
		var kept []env
		for _, e := range envs {
			v, err := eval(s.where, e, nil)
			if err != nil {
				return nil, err
			}
			if v == true {
				kept = append(kept, e)
			}
		}
		envs = kept
	}
	out, err := d.project(s, envs)
	if err != nil {
		return nil, err
	}
	return finish(s, out), nil
}

func (d *database) refTable(r tableRef) (*table, *sqlError) {
	t := d.find(r.name)
	if t == nil {
		return nil, errAt(r.pos, "there is no table called %s", r.name)
	}
	return t, nil
}

func (r tableRef) label() string {
	if r.alias != "" {
		return r.alias
	}
	return r.name
}

// source is every row of the FROM table, joined to the JOIN table if any.
func (d *database) source(s *selectStmt) ([]env, *sqlError) {
	t, err := d.refTable(s.from)
	if err != nil {
		return nil, err
	}
	cols := make([]colRef, len(t.cols))
	for i, c := range t.cols {
		cols[i] = colRef{s.from.label(), c}
	}
	envs := make([]env, len(t.rows))
	for i, r := range t.rows {
		envs[i] = env{cols, r}
	}
	if s.join == nil {
		return envs, nil
	}
	jt, err := d.refTable(s.join.ref)
	if err != nil {
		return nil, err
	}
	jcols := append([]colRef(nil), cols...)
	for _, c := range jt.cols {
		jcols = append(jcols, colRef{s.join.ref.label(), c})
	}
	var out []env
	for _, l := range envs {
		for _, r := range jt.rows {
			e := env{jcols, append(append([]value(nil), l.vals...), r...)}
			v, err := eval(s.join.on, e, nil)
			if err != nil {
				return nil, err
			}
			if v == true {
				out = append(out, e)
			}
		}
	}
	return out, nil
}

// outRow is one result row with what it was made from, for ORDER BY.
type outRow struct {
	vals  []value
	src   env
	group []env
}

type output struct {
	cols []string
	rows []outRow
	ord  []orderItem
}

func (d *database) project(s *selectStmt, envs []env) (*output, *sqlError) {
	o := &output{ord: s.order}
	var items []selItem
	if s.star {
		if len(envs) == 0 {
			items = d.starItems(s)
		} else {
			for _, c := range envs[0].cols {
				items = append(items, selItem{e: colExpr{table: c.table, name: c.name}, text: c.name})
			}
		}
		if s.join != nil { // two tables: say which table a repeated name is from
			seen := map[string]int{}
			for _, it := range items {
				seen[strings.ToLower(it.text)]++
			}
			for i, it := range items {
				if seen[strings.ToLower(it.text)] > 1 {
					items[i].text = it.e.(colExpr).table + "." + it.text
				}
			}
		}
	} else {
		items = s.items
	}
	for _, it := range items {
		h := it.text
		if it.alias != "" {
			h = it.alias
		} else if c, ok := it.e.(colExpr); ok && !s.star {
			h = c.name
		}
		o.cols = append(o.cols, h)
	}
	grouped := len(s.group) > 0 || s.having != nil || anyAggregate(items, s.order)
	if !grouped {
		for _, e := range envs {
			vals, err := evalAll(items, e, nil)
			if err != nil {
				return nil, err
			}
			o.rows = append(o.rows, outRow{vals: vals, src: e})
		}
		return o, nil
	}
	return o, groupRows(s, items, envs, o)
}

func (d *database) starItems(s *selectStmt) []selItem {
	var items []selItem
	add := func(r tableRef) {
		if t := d.find(r.name); t != nil {
			for _, c := range t.cols {
				items = append(items, selItem{e: colExpr{table: r.label(), name: c}, text: c})
			}
		}
	}
	add(s.from)
	if s.join != nil {
		add(s.join.ref)
	}
	return items
}

func groupRows(s *selectStmt, items []selItem, envs []env, o *output) *sqlError {
	var order []string
	groups := map[string][]env{}
	for _, e := range envs {
		key := ""
		for _, g := range s.group {
			v, err := eval(g, e, nil)
			if err != nil {
				return err
			}
			key += "\x00" + keyOf(v)
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], e)
	}
	if len(s.group) == 0 && len(envs) == 0 {
		order, groups = []string{""}, map[string][]env{"": {}} // a total over no rows is still one row
	}
	for _, k := range order {
		g := groups[k]
		var first env
		if len(g) > 0 {
			first = g[0]
		}
		if s.having != nil {
			v, err := eval(s.having, first, g)
			if err != nil {
				return err
			}
			if v != true {
				continue
			}
		}
		vals, err := evalAll(items, first, g)
		if err != nil {
			return err
		}
		o.rows = append(o.rows, outRow{vals: vals, src: first, group: g})
	}
	return nil
}

func evalAll(items []selItem, e env, g []env) ([]value, *sqlError) {
	vals := make([]value, len(items))
	for i, it := range items {
		v, err := eval(it.e, e, g)
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}
	return vals, nil
}

// finish applies DISTINCT, ORDER BY, OFFSET and LIMIT.
func finish(s *selectStmt, o *output) *result {
	rows := o.rows
	if s.distinct {
		seen := map[string]bool{}
		var uniq []outRow
		for _, r := range rows {
			k := ""
			for _, v := range r.vals {
				k += "\x00" + keyOf(v)
			}
			if !seen[k] {
				seen[k] = true
				uniq = append(uniq, r)
			}
		}
		rows = uniq
	}
	if len(s.order) > 0 {
		keys := make([][]value, len(rows))
		for i, r := range rows {
			keys[i] = orderKeys(s, o, r)
		}
		idx := make([]int, len(rows))
		for i := range idx {
			idx[i] = i
		}
		sort.SliceStable(idx, func(a, b int) bool {
			for k, it := range s.order {
				c := compare(keys[idx[a]][k], keys[idx[b]][k])
				if c != 0 {
					return (c < 0) != it.desc
				}
			}
			return false
		})
		sorted := make([]outRow, len(rows))
		for i, j := range idx {
			sorted[i] = rows[j]
		}
		rows = sorted
	}
	if s.offset > 0 {
		rows = rows[min(s.offset, len(rows)):]
	}
	if s.limit >= 0 && s.limit < len(rows) {
		rows = rows[:s.limit]
	}
	res := &result{cols: o.cols}
	for _, r := range rows {
		res.rows = append(res.rows, r.vals)
	}
	return res
}

// orderKeys are the values a row is sorted by: an output column named by the
// ORDER BY (a heading, or its number), otherwise the expression on the source.
func orderKeys(s *selectStmt, o *output, r outRow) []value {
	keys := make([]value, len(s.order))
	for k, it := range s.order {
		switch e := it.e.(type) {
		case litExpr:
			if n, ok := e.v.(float64); ok && n >= 1 && int(n) <= len(r.vals) {
				keys[k] = r.vals[int(n)-1]
				continue
			}
		case colExpr:
			if e.table == "" {
				for i, c := range o.cols {
					if strings.EqualFold(c, e.name) {
						keys[k] = r.vals[i]
						goto done
					}
				}
			}
		}
		keys[k], _ = eval(it.e, r.src, r.group) // a problem here sorts as NULL; the columns were checked already
	done:
	}
	return keys
}

// ---- values ----------------------------------------------------------------

func keyOf(v value) string { return fmt.Sprintf("%T:%v", v, v) }

// compare orders values: NULL first, then numbers and text. A number and text
// that reads as a number compare as numbers.
func compare(a, b value) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	}
	if x, ok := a.(float64); ok {
		if y, ok := asNumber(b); ok {
			return cmpFloat(x, y)
		}
	}
	if y, ok := b.(float64); ok {
		if x, ok := asNumber(a); ok {
			return cmpFloat(x, y)
		}
	}
	if x, ok := a.(bool); ok {
		if y, ok := b.(bool); ok {
			switch {
			case x == y:
				return 0
			case !x:
				return -1
			}
			return 1
		}
	}
	return strings.Compare(format(a), format(b))
}

func cmpFloat(x, y float64) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

func asNumber(v value) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

// format shows a value as text.
func format(v value) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case float64:
		x = math.Round(x*1e6) / 1e6
		if x == math.Trunc(x) && math.Abs(x) < 1e15 {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	return fmt.Sprint(v)
}
