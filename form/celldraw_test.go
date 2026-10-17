package form

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
)

// sameScreen fails t unless DrawCells draws what View shows, cell by cell,
// cluster, width and style, over a w x h grid.
func sameScreen(t *testing.T, m Model, w, h int) {
	t.Helper()
	want := cellbuf.New(w, h)
	tui.DrawView(want, want.Bounds(), m.View())
	got := cellbuf.New(w, h)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g, e := got.At(x, y), want.At(x, y)
			if g.Cluster != e.Cluster || g.Width != e.Width || got.Style(g.Style) != want.Style(e.Style) {
				t.Fatalf("cell (%d,%d): DrawCells %q w%d %+v, View %q w%d %+v\nView:\n%s", x, y,
					g.Cluster, g.Width, got.Style(g.Style), e.Cluster, e.Width, want.Style(e.Style), m.View())
			}
		}
	}
}

func TestDrawCellsMatchesView(t *testing.T) {
	text := Field{Name: "email", Label: "Email", Value: "ada@example.com", Validators: []Validator{Required(), Email()}}
	cases := map[string]func() Model{
		"text":        func() Model { return New(text) },
		"empty":       func() Model { return New(Field{Name: "a", Label: "A", Placeholder: "type here"}) },
		"secret":      func() Model { return New(Field{Name: "p", Label: "Password", Secret: true, Value: "hunter2"}) },
		"description": func() Model { return New(Field{Name: "a", Label: "A", Description: "Used to sign in"}) },
		"wide":        func() Model { return New(Field{Name: "a", Label: "名前", Value: "日本語 😀"}) },
		"every kind": func() Model {
			return New(text,
				Field{Name: "pw", Label: "Password", Secret: true, Value: "x"},
				Field{Name: "plan", Label: "Plan", Kind: FieldSelect, Options: []string{"Free", "Pro"}},
				Field{Name: "ok", Label: "Agree", Kind: FieldCheckbox},
				Field{Name: "go", Label: "Go?", Kind: FieldConfirm},
				Field{Name: "n", Label: "Read this", Kind: FieldNote},
				Field{Name: "tags", Label: "Tags", Kind: FieldMultiSelect, Options: []string{"a", "b"}},
				Field{Name: "bio", Label: "Bio", Kind: FieldTextArea, Height: 3},
				Field{Name: "f", Label: "File", Kind: FieldFile})
		},
		"groups": func() Model {
			return NewGroups(
				Group{Title: "Account", Description: "Who you are", Fields: []Field{text}},
				Group{Title: "Done", Fields: []Field{{Name: "t", Label: "Terms", Kind: FieldCheckbox}}})
		},
		"control chars": func() Model {
			return New(Field{Name: "a", Label: "A", Description: "x\x1b[31my"})
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			m := build()
			sameScreen(t, m, 60, 24)
			m.Focus()
			sameScreen(t, m, 60, 24)
			m, _ = m.Submit()
			sameScreen(t, m, 60, 24)
			sameScreen(t, m, 20, 3) // clipped
		})
	}
}

func TestDrawCellsEmptyRegion(t *testing.T) {
	New(Field{Name: "a", Label: "A"}).DrawCells(cellbuf.New(10, 1), cellbuf.Rect{})
}

func benchForm() Model {
	var fs []Field
	for i := 0; i < 8; i++ {
		fs = append(fs, Field{Name: string(rune('a' + i)), Label: "Field", Value: "some value", Validators: []Validator{Required()}})
	}
	m := New(fs...)
	m.Focus()
	return m
}

func BenchmarkFormViewDrawView(b *testing.B) {
	m, buf := benchForm(), cellbuf.New(80, 24)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tui.DrawView(buf, buf.Bounds(), m.View())
	}
}

func BenchmarkFormDrawCells(b *testing.B) {
	m, buf := benchForm(), cellbuf.New(80, 24)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.DrawCells(buf, buf.Bounds())
	}
}
