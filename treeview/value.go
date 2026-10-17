package treeview

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/ows4444/tui/ansi"
)

// ValueOptions bounds how much of a Go value FromValue and ValueNode show.
type ValueOptions struct {
	// MaxDepth is how many levels below the root are expanded into nodes; deeper
	// values are shown as "...". 0 means 8.
	MaxDepth int
	// MaxItems is how many fields, entries or elements of one struct, map, slice
	// or array are shown; the rest are summed in a "... N more" leaf. 0 means 200.
	MaxItems int
}

func (o ValueOptions) depth() int {
	if o.MaxDepth > 0 {
		return o.MaxDepth
	}
	return 8
}

func (o ValueOptions) items() int {
	if o.MaxItems > 0 {
		return o.MaxItems
	}
	return 200
}

// FromValue returns a Model whose tree shows the Go value v: the exported
// fields of a struct, the entries of a map (sorted by key), the elements of a
// slice or array, each with its own children, and leaves for everything else.
// Pointers and interfaces are followed, and a pointer that leads back to a value
// already being drawn is marked "<cycle>" instead of looping. time.Time and
// types with an Error or String method are shown by that text. It is for looking
// at a value, such as a config or a response, in a program that already has a
// tree; the text of strings is cleaned like any label, so a value from outside
// cannot write escape sequences.
func FromValue(v any, opts ...ValueOptions) Model {
	return New(ValueNode("value", v, opts...))
}

// ValueNode is the root Node FromValue builds, labelled label, for placing the
// value among other nodes.
func ValueNode(label string, v any, opts ...ValueOptions) Node {
	var o ValueOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	w := valueWalker{o: o, seen: map[uintptr]bool{}}
	return w.node(label, reflect.ValueOf(v), 0)
}

type valueWalker struct {
	o    ValueOptions
	seen map[uintptr]bool // pointers, maps and slices on the path from the root
}

var (
	timeType   = reflect.TypeOf(time.Time{})
	errorType  = reflect.TypeOf((*error)(nil)).Elem()
	stringType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
)

func leaf(label, text string) Node { return Node{Label: ansi.Sanitize(label + ": " + text)} }

func (w *valueWalker) node(label string, v reflect.Value, depth int) Node {
	if !v.IsValid() {
		return leaf(label, "nil")
	}
	if v.Type() == timeType {
		return leaf(label, v.Interface().(time.Time).Format(time.RFC3339))
	}
	if v.Type().Implements(errorType) || v.Type().Implements(stringType) {
		if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil() {
			return leaf(label, "nil")
		}
		if t, ok := textOf(v); ok {
			return leaf(label, strconv.Quote(t))
		}
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return leaf(label, "nil")
		}
		if v.Kind() == reflect.Pointer {
			p := v.Pointer()
			if w.seen[p] {
				return leaf(label, "<cycle>")
			}
			w.seen[p] = true
			defer delete(w.seen, p)
		}
		return w.node(label, v.Elem(), depth)
	case reflect.Struct:
		if depth >= w.o.depth() {
			return leaf(label, "...")
		}
		t := v.Type()
		var kids []Node
		n := 0
		for i := 0; i < t.NumField(); i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if n++; n > w.o.items() {
				kids = append(kids, leaf("...", strconv.Itoa(exportedFields(t)-w.o.items())+" more"))
				break
			}
			kids = append(kids, w.node(t.Field(i).Name, v.Field(i), depth+1))
		}
		return Node{Label: ansi.Sanitize(label) + " " + t.Name() + " {}", Children: kids}
	case reflect.Map:
		if v.IsNil() {
			return leaf(label, "nil")
		}
		if depth >= w.o.depth() {
			return leaf(label, "...")
		}
		if p := v.Pointer(); w.seen[p] {
			return leaf(label, "<cycle>")
		} else {
			w.seen[p] = true
			defer delete(w.seen, p)
		}
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface()) })
		var kids []Node
		for i, k := range keys {
			if i >= w.o.items() {
				kids = append(kids, leaf("...", strconv.Itoa(len(keys)-i)+" more"))
				break
			}
			kids = append(kids, w.node(fmt.Sprint(k.Interface()), v.MapIndex(k), depth+1))
		}
		return Node{Label: ansi.Sanitize(label) + " {}", Children: kids}
	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice {
			if v.IsNil() {
				return leaf(label, "nil")
			}
		}
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return leaf(label, strconv.Itoa(v.Len())+" bytes")
		}
		if depth >= w.o.depth() {
			return leaf(label, "...")
		}
		if v.Kind() == reflect.Slice && v.Len() > 0 {
			if p := v.Pointer(); w.seen[p] {
				return leaf(label, "<cycle>")
			} else {
				w.seen[p] = true
				defer delete(w.seen, p)
			}
		}
		var kids []Node
		for i := 0; i < v.Len(); i++ {
			if i >= w.o.items() {
				kids = append(kids, leaf("...", strconv.Itoa(v.Len()-i)+" more"))
				break
			}
			kids = append(kids, w.node(strconv.Itoa(i), v.Index(i), depth+1))
		}
		return Node{Label: ansi.Sanitize(label) + " []", Children: kids}
	case reflect.String:
		return leaf(label, strconv.Quote(v.String()))
	case reflect.Func, reflect.Chan, reflect.UnsafePointer:
		if v.IsNil() {
			return leaf(label, "nil")
		}
		return leaf(label, v.Type().String())
	}
	return leaf(label, fmt.Sprint(v.Interface()))
}

// textOf calls Error or String on v, and reports whether it could (a field
// reached through an unexported path cannot be called).
func textOf(v reflect.Value) (s string, ok bool) {
	if !v.CanInterface() {
		return "", false
	}
	defer func() {
		if recover() != nil {
			s, ok = "", false // a method that panics on a zero value
		}
	}()
	switch x := v.Interface().(type) {
	case error:
		return x.Error(), true
	case fmt.Stringer:
		return x.String(), true
	}
	return "", false
}

func exportedFields(t reflect.Type) int {
	n := 0
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).IsExported() {
			n++
		}
	}
	return n
}
