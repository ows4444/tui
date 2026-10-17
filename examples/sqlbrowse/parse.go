package main

import "strings"

// value is a cell: nil (NULL), float64, string or bool.
type value = any

type expr interface{}

type (
	litExpr struct{ v value }
	colExpr struct {
		table, name string
		pos         int
	}
	unaryExpr struct {
		op string // "-" or "NOT"
		x  expr
	}
	binExpr struct {
		op   string // + - * / % = <> < <= > >= AND OR LIKE
		l, r expr
	}
	inExpr struct {
		x    expr
		list []expr
		not  bool
	}
	betweenExpr struct {
		x, lo, hi expr
		not       bool
	}
	isNullExpr struct {
		x   expr
		not bool
	}
	callExpr struct {
		name string // capitals
		args []expr
		star bool // COUNT(*)
		pos  int
	}
)

type selItem struct {
	e     expr
	alias string
	text  string // the item as written, for the column heading
}

type orderItem struct {
	e    expr
	desc bool
}

type tableRef struct {
	name, alias string
	pos         int
}

type joinClause struct {
	ref tableRef
	on  expr
}

type selectStmt struct {
	distinct bool
	star     bool
	items    []selItem
	from     tableRef
	join     *joinClause
	where    expr
	group    []expr
	having   expr
	order    []orderItem
	limit    int // -1 for none
	offset   int
}

var reserved = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "GROUP": true, "BY": true, "HAVING": true, "ORDER": true, "LIMIT": true,
	"OFFSET": true, "JOIN": true, "INNER": true, "ON": true, "AS": true, "AND": true, "OR": true, "NOT": true, "IN": true,
	"LIKE": true, "IS": true, "NULL": true, "BETWEEN": true, "DISTINCT": true, "ASC": true, "DESC": true, "TRUE": true, "FALSE": true,
}

type parser struct {
	q    string
	toks []token
	i    int

	implicit *token // the last name taken as an alias without AS
}

func (p *parser) peek() token { return p.toks[p.i] }

func (p *parser) next() token {
	t := p.toks[p.i]
	if t.kind != tEOF {
		p.i++
	}
	return t
}

func (p *parser) isKw(kw string) bool {
	t := p.peek()
	return t.kind == tIdent && t.upper() == kw
}

func (p *parser) acceptKw(kw string) bool {
	if p.isKw(kw) {
		p.i++
		return true
	}
	return false
}

func (p *parser) isSym(s string) bool {
	t := p.peek()
	return t.kind == tSym && t.text == s
}

func (p *parser) acceptSym(s string) bool {
	if p.isSym(s) {
		p.i++
		return true
	}
	return false
}

func (p *parser) near() string {
	t := p.peek()
	if t.kind == tEOF {
		return "the end of the query"
	}
	return "“" + t.text + "”"
}

func (p *parser) fail(f string) *sqlError {
	return errAt(p.peek().pos, f+" near %s", p.near())
}

func (p *parser) expectKw(kw string) *sqlError {
	if !p.acceptKw(kw) {
		return p.fail("expected " + kw)
	}
	return nil
}

// parse reads one SELECT statement.
func parse(q string) (*selectStmt, *sqlError) {
	toks, err := lex(q)
	if err != nil {
		return nil, err
	}
	p := &parser{q: q, toks: toks}
	if p.peek().kind == tEOF {
		return nil, errAt(0, "type a query")
	}
	if !p.isKw("SELECT") {
		return nil, errAt(p.peek().pos, "this browser only runs SELECT queries (it starts with %s)", p.near())
	}
	p.next()
	s, err := p.selectBody()
	if err != nil {
		return nil, err
	}
	p.acceptSym(";")
	if p.peek().kind != tEOF {
		return nil, p.fail("unexpected")
	}
	return s, nil
}

func (p *parser) selectBody() (*selectStmt, *sqlError) {
	s := &selectStmt{limit: -1}
	s.distinct = p.acceptKw("DISTINCT")
	if err := p.items(s); err != nil {
		return nil, err
	}
	if !p.isKw("FROM") && p.implicit != nil {
		// "SELECT id FORM t": FORM was taken for an alias, so blame it, not what follows.
		return nil, errAt(p.implicit.pos, "expected FROM near “%s”", p.implicit.text)
	}
	if err := p.expectKw("FROM"); err != nil {
		return nil, err
	}
	var err *sqlError
	if s.from, err = p.tableRef(); err != nil {
		return nil, err
	}
	if p.isKw("INNER") || p.isKw("JOIN") {
		p.acceptKw("INNER")
		if err := p.expectKw("JOIN"); err != nil {
			return nil, err
		}
		ref, err := p.tableRef()
		if err != nil {
			return nil, err
		}
		if err := p.expectKw("ON"); err != nil {
			return nil, err
		}
		on, err := p.expr()
		if err != nil {
			return nil, err
		}
		s.join = &joinClause{ref, on}
	}
	if p.acceptKw("WHERE") {
		if s.where, err = p.expr(); err != nil {
			return nil, err
		}
	}
	if p.acceptKw("GROUP") {
		if err := p.expectKw("BY"); err != nil {
			return nil, err
		}
		if s.group, err = p.exprList(); err != nil {
			return nil, err
		}
	}
	if p.acceptKw("HAVING") {
		if s.having, err = p.expr(); err != nil {
			return nil, err
		}
	}
	if p.acceptKw("ORDER") {
		if err := p.expectKw("BY"); err != nil {
			return nil, err
		}
		if err := p.orderList(s); err != nil {
			return nil, err
		}
	}
	return s, p.limits(s)
}

func (p *parser) limits(s *selectStmt) *sqlError {
	for p.isKw("LIMIT") || p.isKw("OFFSET") {
		kw := p.next().upper()
		t := p.peek()
		if t.kind != tNumber || t.num < 0 || t.num != float64(int(t.num)) {
			return p.fail("expected a whole number after " + kw)
		}
		p.next()
		if kw == "LIMIT" {
			s.limit = int(t.num)
		} else {
			s.offset = int(t.num)
		}
	}
	return nil
}

func (p *parser) items(s *selectStmt) *sqlError {
	if p.acceptSym("*") {
		s.star = true
		return nil
	}
	for {
		start := p.peek().pos
		e, err := p.expr()
		if err != nil {
			return err
		}
		it := selItem{e: e, text: strings.TrimSpace(p.q[start:p.peek().pos])}
		p.implicit = nil
		if p.acceptKw("AS") {
			t := p.next()
			if t.kind != tIdent || reserved[t.upper()] {
				return errAt(t.pos, "expected a name after AS")
			}
			it.alias = t.text
		} else if t := p.peek(); t.kind == tIdent && !reserved[t.upper()] {
			p.implicit = &t
			it.alias = p.next().text
		}
		s.items = append(s.items, it)
		if !p.acceptSym(",") {
			return nil
		}
	}
}

func (p *parser) tableRef() (tableRef, *sqlError) {
	t := p.peek()
	if t.kind != tIdent || reserved[t.upper()] {
		return tableRef{}, p.fail("expected a table name")
	}
	p.next()
	ref := tableRef{name: t.text, pos: t.pos}
	if p.acceptKw("AS") {
		a := p.next()
		if a.kind != tIdent || reserved[a.upper()] {
			return ref, errAt(a.pos, "expected a name after AS")
		}
		ref.alias = a.text
	} else if a := p.peek(); a.kind == tIdent && !reserved[a.upper()] {
		ref.alias = p.next().text
	}
	return ref, nil
}

func (p *parser) exprList() ([]expr, *sqlError) {
	var out []expr
	for {
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		if !p.acceptSym(",") {
			return out, nil
		}
	}
}

func (p *parser) orderList(s *selectStmt) *sqlError {
	for {
		e, err := p.expr()
		if err != nil {
			return err
		}
		it := orderItem{e: e}
		if p.acceptKw("DESC") {
			it.desc = true
		} else {
			p.acceptKw("ASC")
		}
		s.order = append(s.order, it)
		if !p.acceptSym(",") {
			return nil
		}
	}
}

// ---- expressions, loosest binding first ------------------------------------

func (p *parser) expr() (expr, *sqlError) { return p.or() }

func (p *parser) or() (expr, *sqlError) {
	l, err := p.and()
	for err == nil && p.acceptKw("OR") {
		var r expr
		if r, err = p.and(); err == nil {
			l = binExpr{"OR", l, r}
		}
	}
	return l, err
}

func (p *parser) and() (expr, *sqlError) {
	l, err := p.not()
	for err == nil && p.acceptKw("AND") {
		var r expr
		if r, err = p.not(); err == nil {
			l = binExpr{"AND", l, r}
		}
	}
	return l, err
}

func (p *parser) not() (expr, *sqlError) {
	if p.acceptKw("NOT") {
		x, err := p.not()
		return unaryExpr{"NOT", x}, err
	}
	return p.cmp()
}

func (p *parser) cmp() (expr, *sqlError) {
	l, err := p.add()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		switch {
		case t.kind == tSym && strings.Contains("= <> != < <= > >=", t.text) && len(t.text) <= 2 && t.text != "":
			p.next()
			r, err := p.add()
			if err != nil {
				return nil, err
			}
			op := t.text
			if op == "!=" {
				op = "<>"
			}
			l = binExpr{op, l, r}
		case p.isKw("LIKE"), p.isKw("NOT") && p.peekKw(1, "LIKE"):
			not := p.acceptKw("NOT")
			p.next()
			r, err := p.add()
			if err != nil {
				return nil, err
			}
			l = binExpr{"LIKE", l, r}
			if not {
				l = unaryExpr{"NOT", l}
			}
		case p.isKw("IN"), p.isKw("NOT") && p.peekKw(1, "IN"):
			not := p.acceptKw("NOT")
			p.next()
			list, err := p.parenList()
			if err != nil {
				return nil, err
			}
			l = inExpr{l, list, not}
		case p.isKw("BETWEEN"), p.isKw("NOT") && p.peekKw(1, "BETWEEN"):
			not := p.acceptKw("NOT")
			p.next()
			lo, err := p.add()
			if err != nil {
				return nil, err
			}
			if err := p.expectKw("AND"); err != nil {
				return nil, err
			}
			hi, err := p.add()
			if err != nil {
				return nil, err
			}
			l = betweenExpr{l, lo, hi, not}
		case p.isKw("IS"):
			p.next()
			not := p.acceptKw("NOT")
			if err := p.expectKw("NULL"); err != nil {
				return nil, err
			}
			l = isNullExpr{l, not}
		default:
			return l, nil
		}
	}
}

func (p *parser) peekKw(n int, kw string) bool {
	if p.i+n >= len(p.toks) {
		return false
	}
	t := p.toks[p.i+n]
	return t.kind == tIdent && t.upper() == kw
}

func (p *parser) parenList() ([]expr, *sqlError) {
	if !p.acceptSym("(") {
		return nil, p.fail("expected (")
	}
	list, err := p.exprList()
	if err != nil {
		return nil, err
	}
	if !p.acceptSym(")") {
		return nil, p.fail("expected )")
	}
	return list, nil
}

func (p *parser) binary(next func() (expr, *sqlError), ops string) (expr, *sqlError) {
	l, err := next()
	for err == nil && p.peek().kind == tSym && len(p.peek().text) == 1 && strings.Contains(ops, p.peek().text) {
		op := p.next().text
		var r expr
		if r, err = next(); err == nil {
			l = binExpr{op, l, r}
		}
	}
	return l, err
}

func (p *parser) add() (expr, *sqlError) { return p.binary(p.mul, "+-") }
func (p *parser) mul() (expr, *sqlError) { return p.binary(p.unary, "*/%") }

func (p *parser) unary() (expr, *sqlError) {
	if p.acceptSym("-") {
		x, err := p.unary()
		return unaryExpr{"-", x}, err
	}
	return p.primary()
}

func (p *parser) primary() (expr, *sqlError) {
	t := p.peek()
	switch {
	case t.kind == tNumber:
		p.next()
		return litExpr{t.num}, nil
	case t.kind == tString:
		p.next()
		return litExpr{t.text}, nil
	case t.kind == tSym && t.text == "(":
		p.next()
		e, err := p.expr()
		if err == nil && !p.acceptSym(")") {
			err = p.fail("expected )")
		}
		return e, err
	case t.kind == tIdent:
		return p.identExpr()
	}
	return nil, p.fail("expected a value")
}

func (p *parser) identExpr() (expr, *sqlError) {
	t := p.next()
	switch t.upper() {
	case "NULL":
		return litExpr{nil}, nil
	case "TRUE":
		return litExpr{true}, nil
	case "FALSE":
		return litExpr{false}, nil
	}
	if reserved[t.upper()] {
		p.i--
		return nil, p.fail("expected a value")
	}
	if p.acceptSym("(") {
		return p.call(t)
	}
	if p.acceptSym(".") {
		c := p.next()
		if c.kind != tIdent {
			return nil, errAt(c.pos, "expected a column name after the dot")
		}
		return colExpr{table: t.text, name: c.text, pos: t.pos}, nil
	}
	return colExpr{name: t.text, pos: t.pos}, nil
}

func (p *parser) call(name token) (expr, *sqlError) {
	c := callExpr{name: name.upper(), pos: name.pos}
	if p.acceptSym("*") {
		c.star = true
	} else if !p.isSym(")") {
		args, err := p.exprList()
		if err != nil {
			return nil, err
		}
		c.args = args
	}
	if !p.acceptSym(")") {
		return nil, p.fail("expected )")
	}
	return c, nil
}
