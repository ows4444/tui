package main

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

var (
	errSyntax   = errors.New("incomplete expression")
	errDivZero  = errors.New("cannot divide by zero")
	errNotValid = errors.New("not a number")
)

// eval computes an arithmetic expression: + - * / % ^, unary minus,
// parentheses and decimal numbers. It is a recursive-descent parser, so
// precedence and nesting follow from the grammar:
//
//	expr  = term { ("+" | "-") term }
//	term  = unary { ("*" | "/" | "%") unary }
//	unary = "-" unary | power
//	power = primary [ "^" unary ]
func eval(s string) (float64, error) {
	p := parser{s: strings.NewReplacer("÷", "/", "×", "*", "−", "-").Replace(s)}
	v, err := p.expr()
	if err != nil {
		return 0, err
	}
	if p.peek() != 0 {
		return 0, errSyntax
	}
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, errNotValid
	}
	return v, nil
}

type parser struct {
	s string
	i int
}

// peek skips spaces and returns the next byte, or 0 at the end.
func (p *parser) peek() byte {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
	if p.i < len(p.s) {
		return p.s[p.i]
	}
	return 0
}

func (p *parser) expr() (float64, error) {
	v, err := p.term()
	for err == nil && (p.peek() == '+' || p.peek() == '-') {
		op := p.s[p.i]
		p.i++
		var r float64
		if r, err = p.term(); err != nil {
			break
		}
		if op == '+' {
			v += r
		} else {
			v -= r
		}
	}
	return v, err
}

func (p *parser) term() (float64, error) {
	v, err := p.unary()
	for err == nil && (p.peek() == '*' || p.peek() == '/' || p.peek() == '%') {
		op := p.s[p.i]
		p.i++
		var r float64
		if r, err = p.unary(); err != nil {
			break
		}
		switch {
		case op == '*':
			v *= r
		case r == 0:
			return 0, errDivZero
		case op == '/':
			v /= r
		default:
			v = math.Mod(v, r)
		}
	}
	return v, err
}

func (p *parser) unary() (float64, error) {
	if p.peek() == '-' {
		p.i++
		v, err := p.unary()
		return -v, err
	}
	return p.power()
}

func (p *parser) power() (float64, error) {
	v, err := p.primary()
	if err == nil && p.peek() == '^' {
		p.i++
		var e float64
		if e, err = p.unary(); err == nil {
			v = math.Pow(v, e)
		}
	}
	return v, err
}

func (p *parser) primary() (float64, error) {
	if p.peek() == '(' {
		p.i++
		v, err := p.expr()
		if err != nil {
			return 0, err
		}
		if p.peek() != ')' {
			return 0, errSyntax
		}
		p.i++
		return v, nil
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.') {
		p.i++
	}
	if start == p.i {
		return 0, errSyntax
	}
	v, err := strconv.ParseFloat(p.s[start:p.i], 64)
	if err != nil {
		return 0, errNotValid
	}
	return v, nil
}

// format prints v with at most 12 significant digits and no trailing zeros.
func format(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'g', 12, 64)
}
