package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

type kind int

const (
	object kind = iota
	array
	str
	num
	boolean
	null
)

// node is one value of the document. Object members keep the order the file
// gave them, and numbers keep their text ("1.50" stays "1.50").
type node struct {
	key      string // the member name, or the index of an array element; "" for the root
	kind     kind
	text     string // a string's value, or a number, bool or null as written
	children []*node
	parent   *node
}

func (n *node) container() bool { return n.kind == object || n.kind == array }

const maxDepth = 500

// parse reads one JSON document from r. Anything after it is an error.
func parse(r io.Reader) (*node, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	root, err := value(dec, 0)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("unexpected data after the end of the document")
	}
	return root, nil
}

func value(dec *json.Decoder, depth int) (*node, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("nested more than %d levels deep", maxDepth)
	}
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		return container(dec, t, depth)
	case string:
		return &node{kind: str, text: t}, nil
	case json.Number:
		return &node{kind: num, text: t.String()}, nil
	case bool:
		return &node{kind: boolean, text: strconv.FormatBool(t)}, nil
	}
	return &node{kind: null, text: "null"}, nil
}

func container(dec *json.Decoder, open json.Delim, depth int) (*node, error) {
	n := &node{kind: array}
	if open == '{' {
		n.kind = object
	}
	for i := 0; dec.More(); i++ {
		key := strconv.Itoa(i)
		if n.kind == object {
			k, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key = k.(string) // the decoder only yields strings where a key must be
		}
		child, err := value(dec, depth+1)
		if err != nil {
			return nil, err
		}
		child.key, child.parent = key, n
		n.children = append(n.children, child)
	}
	if _, err := dec.Token(); err != nil { // the closing bracket
		return nil, err
	}
	return n, nil
}

// ---- paths -----------------------------------------------------------------

var plainKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// path is the JSONPath-style address of n: $.users[2].name.
func (n *node) path() string {
	if n.parent == nil {
		return "$"
	}
	step := "[" + n.key + "]"
	if n.parent.kind == object {
		if plainKey.MatchString(n.key) {
			step = "." + n.key
		} else {
			step = "[" + quote(n.key) + "]"
		}
	}
	return n.parent.path() + step
}

func quote(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // a string always encodes
	return strings.TrimSuffix(b.String(), "\n")
}

// ---- text ------------------------------------------------------------------

// scalarText is how a leaf is shown: strings quoted, the rest as written.
func (n *node) scalarText() string {
	if n.kind == str {
		return quote(n.text)
	}
	return n.text
}

// summary is the one-line form of a container: {3 keys} or [12 items].
func (n *node) summary() string {
	c := len(n.children)
	switch {
	case n.kind == object && c == 1:
		return "{1 key}"
	case n.kind == object:
		return fmt.Sprintf("{%d keys}", c)
	case c == 1:
		return "[1 item]"
	}
	return fmt.Sprintf("[%d items]", c)
}

// pretty writes n as indented JSON, stopping after limit bytes.
func (n *node) pretty(limit int) string {
	var b strings.Builder
	n.write(&b, 0, limit)
	if b.Len() > limit {
		return b.String()[:limit] + "…"
	}
	return b.String()
}

func (n *node) write(b *strings.Builder, depth, limit int) {
	if b.Len() > limit {
		return
	}
	if !n.container() {
		b.WriteString(n.scalarText())
		return
	}
	open, closeCh := "{", "}"
	if n.kind == array {
		open, closeCh = "[", "]"
	}
	if len(n.children) == 0 {
		b.WriteString(open + closeCh)
		return
	}
	b.WriteString(open + "\n")
	pad := strings.Repeat("  ", depth+1)
	for i, c := range n.children {
		b.WriteString(pad)
		if n.kind == object {
			b.WriteString(quote(c.key) + ": ")
		}
		c.write(b, depth+1, limit)
		if i < len(n.children)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("  ", depth) + closeCh)
}

// copyText is what "copy value" puts on the clipboard.
func (n *node) copyText() string {
	if n.container() {
		return n.pretty(1 << 16)
	}
	return n.text // a string without its quotes, anything else as written
}

// ---- search ----------------------------------------------------------------

// matches lists, in document order, the nodes whose key (for an object member)
// or leaf value contains q, ignoring case.
func (n *node) matches(q string) []*node {
	if q == "" {
		return nil
	}
	q = strings.ToLower(q)
	var out []*node
	var walk func(*node)
	walk = func(x *node) {
		if x.hit(q) {
			out = append(out, x)
		}
		for _, c := range x.children {
			walk(c)
		}
	}
	walk(n)
	return out
}

func (n *node) hit(lowerQ string) bool {
	if n.parent != nil && n.parent.kind == object && strings.Contains(strings.ToLower(n.key), lowerQ) {
		return true
	}
	return !n.container() && strings.Contains(strings.ToLower(n.text), lowerQ)
}

// count is how many values the document holds, containers included.
func (n *node) count() int {
	c := 1
	for _, ch := range n.children {
		c += ch.count()
	}
	return c
}
