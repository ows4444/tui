//covercheck:helper copy-isolation and size-matrix helpers for tests; no production logic

// Package testutil holds test helpers shared by the module's packages. It is
// imported only from _test.go files.
package testutil

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

// Dump renders v deeply and deterministically, including unexported fields:
// structs, arrays, slices (contents, len and cap ignored beyond contents),
// maps (sorted) and interfaces are walked; pointers, channels and funcs are
// reduced to their nil-ness, because sharing those is by design. Two values
// with equal dumps hold equal reachable value-typed state.
func Dump(v any) string {
	var b strings.Builder
	dump(&b, reflect.ValueOf(v), 0)
	return b.String()
}

func dump(b *strings.Builder, v reflect.Value, depth int) {
	if depth > 40 {
		b.WriteString("<deep>")
		return
	}
	if !v.IsValid() {
		b.WriteString("<invalid>")
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		fmt.Fprint(b, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fmt.Fprint(b, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		fmt.Fprint(b, v.Uint())
	case reflect.Float32, reflect.Float64:
		fmt.Fprint(b, v.Float())
	case reflect.Complex64, reflect.Complex128:
		fmt.Fprint(b, v.Complex())
	case reflect.String:
		fmt.Fprintf(b, "%q", v.String())
	case reflect.Ptr, reflect.Chan, reflect.Func, reflect.UnsafePointer:
		fmt.Fprintf(b, "%s(nil=%t)", v.Kind(), v.IsNil())
	case reflect.Interface:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		dump(b, v.Elem(), depth+1)
	case reflect.Slice:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		fallthrough
	case reflect.Array:
		b.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			dump(b, v.Index(i), depth+1)
		}
		b.WriteByte(']')
	case reflect.Map:
		if v.IsNil() {
			b.WriteString("nil")
			return
		}
		var parts []string
		it := v.MapRange()
		for it.Next() {
			var kb, vb strings.Builder
			dump(&kb, it.Key(), depth+1)
			dump(&vb, it.Value(), depth+1)
			parts = append(parts, kb.String()+":"+vb.String())
		}
		sort.Strings(parts)
		b.WriteString("map{" + strings.Join(parts, ",") + "}")
	case reflect.Struct:
		b.WriteByte('{')
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(t.Field(i).Name + "=")
			dump(b, v.Field(i), depth+1)
		}
		b.WriteByte('}')
	default:
		fmt.Fprintf(b, "<%s>", v.Kind())
	}
}

// StdMsgs is a broad set of input messages: navigation, editing, toggling,
// paste and resize. Widgets ignore the ones that do not apply to them.
func StdMsgs() map[string]tui.Msg {
	k := func(t tui.KeyType) tui.Msg { return tui.Key{Type: t} }
	r := func(rs ...rune) tui.Msg { return tui.Key{Type: tui.KeyRunes, Text: string(rs), Code: rs[0]} }
	c := func(x rune) tui.Msg { return tui.Key{Type: tui.KeyCtrl, Code: x} }
	return map[string]tui.Msg{
		"up": k(tui.KeyUp), "down": k(tui.KeyDown), "left": k(tui.KeyLeft), "right": k(tui.KeyRight),
		"enter": k(tui.KeyEnter), "esc": k(tui.KeyEsc), "tab": k(tui.KeyTab), "space": k(tui.KeySpace),
		"home": k(tui.KeyHome), "end": k(tui.KeyEnd), "pgup": k(tui.KeyPgUp), "pgdn": k(tui.KeyPgDown),
		"backspace": k(tui.KeyBackspace), "delete": k(tui.KeyDelete),
		"rune-a": r('a'), "rune-1": r('1'), "rune-slash": r('/'), "rune-y": r('y'), "rune-n": r('n'),
		"ctrl-u": c('u'), "ctrl-k": c('k'), "ctrl-w": c('w'), "ctrl-a": c('a'), "ctrl-e": c('e'),
		"paste":  tui.PasteEvent{Text: "pasted\ntext"},
		"resize": tui.ResizeMsg{Width: 20, Height: 5},
	}
}

// StdSeq is a message sequence that exercises drill-in/back-out and
// edit-then-move patterns, where a later step could write into storage shared
// with an earlier snapshot.
func StdSeq() []tui.Msg {
	m := StdMsgs()
	var out []tui.Msg
	for _, n := range []string{"down", "enter", "down", "space", "enter", "esc", "esc", "enter", "rune-a",
		"backspace", "left", "delete", "paste", "ctrl-u", "up", "right", "enter", "esc", "tab", "space", "end", "home"} {
		out = append(out, m[n])
	}
	return out
}

// CopyIsolation asserts that a value copy of a Model is unaffected by Update
// on the original. build returns a fresh, ready-to-use Model (focused, open,
// populated) for each case; update is the Model's Update. For every message
// in msgs (and for StdMsgs, unless msgs is non-nil) it snapshots a copy,
// updates the original, and fails if the snapshot's reachable state changed.
// It then feeds StdSeq step by step and checks every earlier snapshot after
// each step, so that a later write into storage shared with any earlier copy
// is caught too.
func CopyIsolation[M any](t *testing.T, build func() M, update func(M, tui.Msg) M, msgs map[string]tui.Msg) {
	t.Helper()
	if msgs == nil {
		msgs = StdMsgs()
	}
	for name, msg := range msgs {
		m := build()
		snap := m
		before := Dump(snap)
		_ = update(m, msg)
		if after := Dump(snap); after != before {
			t.Errorf("%s: copy changed after Update on original\nbefore: %s\nafter:  %s", name, before, after)
		}
	}

	m := build()
	var snaps []M
	var want []string
	for i, msg := range StdSeq() {
		snaps = append(snaps, m)
		want = append(want, Dump(m))
		m = update(m, msg)
		for j := range snaps {
			if got := Dump(snaps[j]); got != want[j] {
				t.Fatalf("sequence step %d (%T %v): snapshot %d changed\nbefore: %s\nafter:  %s", i, msg, msg, j, want[j], got)
			}
		}
	}
}
