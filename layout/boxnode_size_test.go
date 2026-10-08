package layout

import "testing"

// A Width, a Height or a title set on the box is the size the box asks for.
// Unconstrained, BoxNode then draws what Box.Render draws for the same box,
// so moving a call from the deprecated method to BoxNode changes nothing.
func TestBoxNodeAsksForTheBoxesOwnSize(t *testing.T) {
	unbounded := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	border := NewBox().Border(NormalBorder())
	for _, tc := range []struct {
		name    string
		box     Box
		content string
		size    Size
	}{
		{"height", border.Height(3), "x", Size{W: 3, H: 5}},
		{"height shorter than the content", border.Height(1), "a\nb\nc", Size{W: 3, H: 3}},
		{"width", border.Width(5), "x", Size{W: 7, H: 3}},
		{"width narrower than the content", border.Width(2), "abcdef", Size{W: 4, H: 3}},
		{"width and height, padded", border.Width(4).Height(2).PaddingAll(1), "x", Size{W: 8, H: 6}},
		{"title wider than the content", border.Title("Settings", AlignStart), "x", Size{W: 12, H: 3}},
		{"title narrower than the content", border.Title("ab", AlignStart), "abcdefgh", Size{W: 10, H: 3}},
		{"title with the top side off", border.Title("Settings", AlignStart).BorderSides(false, true, true, true), "x", Size{W: 3, H: 2}},
		{"title on a box of fixed width", border.Title("Settings", AlignStart).Width(3), "x", Size{W: 5, H: 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := BoxNode(tc.box, Block(tc.content))
			s := n.Measure(unbounded)
			if s != tc.size {
				t.Errorf("Measure = %+v, want %+v", s, tc.size)
			}
			// Box.Render is the reference BoxNode must match.
			want := tc.box.Render(tc.content)
			if got := n.Render(tc.size); got != want {
				t.Fatalf("Render(%+v) =\n%s\nwant, as Box.Render draws it,\n%s", tc.size, got, want)
			}
		})
	}
}

// The layout's constraints still bound the box: given less room than it asks
// for, it keeps its frame and the child is cut inside it.
func TestBoxNodeGivenLessRoomThanItAsksForKeepsItsFrame(t *testing.T) {
	border := NewBox().Border(NormalBorder())
	n := BoxNode(border.Width(8).Height(4), Block("abcdefgh\n2\n3\n4"))
	tight := Constraints{MaxW: 5, MaxH: 4}
	if s := n.Measure(tight); s != (Size{W: 5, H: 4}) {
		t.Errorf("Measure = %+v, want {W:5 H:4}", s)
	}
	if got, want := n.Render(Size{W: 5, H: 4}), "┌───┐\n│abc│\n│2  │\n└───┘"; got != want {
		t.Fatalf("Render =\n%s\nwant\n%s", got, want)
	}
	// A title wider than the room is clipped to the border, as before.
	n = BoxNode(border.Title("Settings", AlignStart), Block("x"))
	if got, want := n.Render(Size{W: 6, H: 3}), "┌ Set┐\n│x   │\n└────┘"; got != want {
		t.Fatalf("Render =\n%s\nwant\n%s", got, want)
	}
}
