package main

import (
	"math"
	"regexp"
	"strings"
)

var aggregates = map[string]bool{"COUNT": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true}

func hasAggregate(e expr) bool {
	switch x := e.(type) {
	case callExpr:
		if aggregates[x.name] {
			return true
		}
		for _, a := range x.args {
			if hasAggregate(a) {
				return true
			}
		}
	case unaryExpr:
		return hasAggregate(x.x)
	case binExpr:
		return hasAggregate(x.l) || hasAggregate(x.r)
	case inExpr:
		return hasAggregate(x.x)
	case betweenExpr:
		return hasAggregate(x.x) || hasAggregate(x.lo) || hasAggregate(x.hi)
	case isNullExpr:
		return hasAggregate(x.x)
	}
	return false
}

func anyAggregate(items []selItem, order []orderItem) bool {
	for _, it := range items {
		if hasAggregate(it.e) {
			return true
		}
	}
	for _, o := range order {
		if hasAggregate(o.e) {
			return true
		}
	}
	return false
}

// eval computes e for one row; g is the rows of the group when aggregating.
func eval(e expr, row env, g []env) (value, *sqlError) {
	switch x := e.(type) {
	case litExpr:
		return x.v, nil
	case colExpr:
		return row.lookup(x)
	case unaryExpr:
		return evalUnary(x, row, g)
	case binExpr:
		return evalBinary(x, row, g)
	case isNullExpr:
		v, err := eval(x.x, row, g)
		return (v == nil) != x.not, err
	case inExpr:
		return evalIn(x, row, g)
	case betweenExpr:
		return evalBetween(x, row, g)
	case callExpr:
		return evalCall(x, row, g)
	}
	return nil, errAt(-1, "cannot evaluate this")
}

func evalUnary(x unaryExpr, row env, g []env) (value, *sqlError) {
	v, err := eval(x.x, row, g)
	if err != nil || v == nil {
		return nil, err
	}
	if x.op == "NOT" {
		return v != true, nil
	}
	f, ok := asNumber(v)
	if !ok {
		return nil, errAt(-1, "cannot negate %q", format(v))
	}
	return -f, nil
}

func evalBinary(x binExpr, row env, g []env) (value, *sqlError) {
	l, err := eval(x.l, row, g)
	if err != nil {
		return nil, err
	}
	switch x.op { // AND and OR do not always need the right side
	case "AND":
		if l == false {
			return false, nil
		}
	case "OR":
		if l == true {
			return true, nil
		}
	}
	r, err := eval(x.r, row, g)
	if err != nil {
		return nil, err
	}
	switch x.op {
	case "AND":
		return l == true && r == true, nil
	case "OR":
		return l == true || r == true, nil
	}
	if l == nil || r == nil {
		return nil, nil // anything with NULL is unknown
	}
	switch x.op {
	case "=":
		return compare(l, r) == 0, nil
	case "<>":
		return compare(l, r) != 0, nil
	case "<":
		return compare(l, r) < 0, nil
	case "<=":
		return compare(l, r) <= 0, nil
	case ">":
		return compare(l, r) > 0, nil
	case ">=":
		return compare(l, r) >= 0, nil
	case "LIKE":
		return like(format(l), format(r)), nil
	}
	a, aok := asNumber(l)
	b, bok := asNumber(r)
	if !aok || !bok {
		return nil, errAt(-1, "%s needs numbers, not %q and %q", x.op, format(l), format(r))
	}
	switch x.op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	}
	if b == 0 {
		return nil, errAt(-1, "division by zero")
	}
	if x.op == "%" {
		return math.Mod(a, b), nil
	}
	return a / b, nil
}

func evalIn(x inExpr, row env, g []env) (value, *sqlError) {
	v, err := eval(x.x, row, g)
	if err != nil || v == nil {
		return nil, err
	}
	for _, it := range x.list {
		c, err := eval(it, row, g)
		if err != nil {
			return nil, err
		}
		if c != nil && compare(v, c) == 0 {
			return !x.not, nil
		}
	}
	return x.not, nil
}

func evalBetween(x betweenExpr, row env, g []env) (value, *sqlError) {
	v, err := eval(x.x, row, g)
	if err != nil {
		return nil, err
	}
	lo, err := eval(x.lo, row, g)
	if err != nil {
		return nil, err
	}
	hi, err := eval(x.hi, row, g)
	if err != nil {
		return nil, err
	}
	if v == nil || lo == nil || hi == nil {
		return nil, nil
	}
	return (compare(v, lo) >= 0 && compare(v, hi) <= 0) != x.not, nil
}

// like matches s against a pattern with % (any run) and _ (any one character),
// ignoring case.
func like(s, pattern string) bool {
	var b strings.Builder
	b.WriteString("(?is)^")
	for _, r := range pattern {
		switch r {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	return err == nil && re.MatchString(s)
}

func evalCall(c callExpr, row env, g []env) (value, *sqlError) {
	if aggregates[c.name] {
		if g == nil { // not grouping: a group of no rows is an empty slice, not nil
			return nil, errAt(c.pos, "%s cannot be used here", c.name)
		}
		return aggregate(c, g)
	}
	args := make([]value, len(c.args))
	for i, a := range c.args {
		v, err := eval(a, row, g)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	need := func(n int) *sqlError {
		if len(args) != n {
			return errAt(c.pos, "%s takes %d argument(s)", c.name, n)
		}
		return nil
	}
	switch c.name {
	case "UPPER", "LOWER", "LENGTH", "ABS":
		if err := need(1); err != nil || args[0] == nil {
			return nil, err
		}
		return unaryFn(c, args[0])
	case "ROUND":
		return roundFn(c, args)
	case "COALESCE":
		for _, a := range args {
			if a != nil {
				return a, nil
			}
		}
		return nil, nil
	}
	return nil, errAt(c.pos, "there is no function called %s", c.name)
}

func unaryFn(c callExpr, v value) (value, *sqlError) {
	switch c.name {
	case "UPPER":
		return strings.ToUpper(format(v)), nil
	case "LOWER":
		return strings.ToLower(format(v)), nil
	case "LENGTH":
		return float64(len([]rune(format(v)))), nil
	}
	f, ok := asNumber(v)
	if !ok {
		return nil, errAt(c.pos, "ABS needs a number")
	}
	return math.Abs(f), nil
}

func roundFn(c callExpr, args []value) (value, *sqlError) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errAt(c.pos, "ROUND takes a number and optionally the decimals")
	}
	if args[0] == nil {
		return nil, nil
	}
	f, ok := asNumber(args[0])
	if !ok {
		return nil, errAt(c.pos, "ROUND needs a number")
	}
	places := 0.0
	if len(args) == 2 {
		if places, ok = asNumber(args[1]); !ok {
			return nil, errAt(c.pos, "ROUND needs the decimals as a number")
		}
	}
	p := math.Pow(10, math.Max(0, math.Min(places, 9)))
	return math.Round(f*p) / p, nil
}

func aggregate(c callExpr, g []env) (value, *sqlError) {
	if c.star {
		if c.name != "COUNT" {
			return nil, errAt(c.pos, "%s(*) is not allowed", c.name)
		}
		return float64(len(g)), nil
	}
	if len(c.args) != 1 {
		return nil, errAt(c.pos, "%s takes one argument", c.name)
	}
	var vals []value
	for _, row := range g {
		v, err := eval(c.args[0], row, nil)
		if err != nil {
			return nil, err
		}
		if v != nil {
			vals = append(vals, v)
		}
	}
	switch c.name {
	case "COUNT":
		return float64(len(vals)), nil
	case "MIN", "MAX":
		var best value
		for i, v := range vals {
			switch cmp := compare(v, best); {
			case i == 0, c.name == "MIN" && cmp < 0, c.name == "MAX" && cmp > 0:
				best = v
			}
		}
		return best, nil
	}
	sum, n := 0.0, 0
	for _, v := range vals {
		f, ok := asNumber(v)
		if !ok {
			return nil, errAt(c.pos, "%s needs numbers, not %q", c.name, format(v))
		}
		sum += f
		n++
	}
	switch {
	case n == 0:
		return nil, nil
	case c.name == "AVG":
		return sum / float64(n), nil
	}
	return sum, nil
}
