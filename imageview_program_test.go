package tui

import (
	"bytes"
	"strings"
	"testing"
)

// kittyView is what imageview.Model.View produces with Kitty set (the
// imageview package cannot be imported here: it imports tui). imageview's own
// tests feed the real View to the cell renderer.
func kittyView(id string) string {
	apc := "\x1b_Ga=T,f=100,C=1,c=6,r=3,i=" + id + ",p=" + id + ",q=2,m=0;AAAA\x1b\\"
	return strings.Join([]string{apc + "      ", "      ", "      "}, "\n")
}

func deleteSeq(id string) string { return "\x1b_Ga=d,d=I,i=" + id + ",q=2\x1b\\" }

func imageProgram(out, log *bytes.Buffer) *Program {
	p := NewProgram(staticModel{}, WithOutput(out), WithFrameLog(log), WithCellRenderer(true))
	p.width, p.height = 40, 100
	return p
}

func imageView(id string, above string) string {
	return above + "\n" + kittyView(id) + "\nfooter"
}

// While an imageview is on screen, the frame log shall show no frame with
// reason non_CSI_escape (the cell renderer takes the kitty sequences as opaque
// segments, so no frame falls back at all).
func TestImageviewFramesNeverFallBackOnEscape(t *testing.T) {
	var out, log bytes.Buffer
	p := imageProgram(&out, &log)
	for _, v := range []string{imageView("5", "a"), imageView("5", "b"), imageView("5", "b")} {
		p.model = staticModel{view: v}
		p.render()
		p.logFrame(out.Len(), p.frameClock())
	}
	if log.Len() == 0 {
		t.Fatal("no frame log")
	}
	if strings.Contains(log.String(), "non_CSI_escape") || strings.Contains(log.String(), "kind=fallback") || strings.Contains(log.String(), "fallback_rows") {
		t.Errorf("frame log names a fallback while an imageview is shown:\n%s", log.String())
	}
	if !p.cells.Valid() {
		t.Error("cell grid not held")
	}
}

// When an imageview is removed from the View, the system shall delete its
// kitty placement; when it moves, the old placement is deleted and the image
// placed at the new row.
func TestImageviewRemovalAndMoveDeletePlacement(t *testing.T) {
	var out, log bytes.Buffer
	p := imageProgram(&out, &log)
	del := deleteSeq("5")
	frame := func(view string) string {
		out.Reset()
		p.model = staticModel{view: view}
		p.render()
		return out.String()
	}
	if got := frame(imageView("5", "a")); strings.Contains(got, del) || !strings.Contains(got, "\x1b_Ga=T,") {
		t.Fatalf("first frame: %q", got)
	}
	// Unchanged image rows are neither re-placed nor deleted.
	if got := frame(imageView("5", "b")); strings.Contains(got, del) || strings.Contains(got, "\x1b_G") {
		t.Errorf("unchanged placement touched: %q", got)
	}
	// Moved down one row.
	moved := "b\nb\n" + strings.SplitN(imageView("5", "x"), "\n", 2)[1]
	if got := frame(moved); !strings.Contains(got, del) || !strings.Contains(got, "\x1b_Ga=T,") || strings.Index(got, del) > strings.Index(got, "\x1b_Ga=T,") {
		t.Errorf("moved: %q, want delete then place", got)
	}
	// Removed.
	if got := frame("b\nb\nfooter"); !strings.Contains(got, del) || strings.Contains(got, "\x1b_Ga=T,") {
		t.Errorf("removed: %q, want delete and no placement", got)
	}
}

// inlineView is what imageview.Model.View produces with Inline set: one
// OSC 1337 File sequence in front of the first of its rows of blank cells.
func inlineView(payload string) string {
	osc := "\x1b]1337;File=inline=1;size=3;width=6;height=3;preserveAspectRatio=0:" + payload + "\a"
	return strings.Join([]string{osc + "      ", "      ", "      "}, "\n")
}

// An inline image is written at its cell with the cursor saved and restored
// around it, written again only when its row changes, and never makes a
// frame fall back.
func TestInlineImageIsWrittenOnceAtItsCell(t *testing.T) {
	var out, log bytes.Buffer
	p := imageProgram(&out, &log)
	frame := func(view string) string {
		out.Reset()
		p.model = staticModel{view: view}
		p.render()
		p.logFrame(out.Len(), p.frameClock())
		return out.String()
	}
	view := func(above, payload string) string { return above + "\n" + inlineView(payload) + "\nfooter" }
	const open = "\x1b]1337;File="
	wrapped := "\x1b7" + strings.SplitN(inlineView("AAAA"), "      ", 2)[0] + "\x1b8"

	first := frame(view("a", "AAAA"))
	if strings.Count(first, open) != 1 || !strings.Contains(first, wrapped) {
		t.Fatalf("first frame does not hold the image once, between a cursor save and restore: %q", first)
	}
	// The same frame again, and a change on another row, leave it alone.
	if got := frame(view("a", "AAAA")); strings.Contains(got, open) {
		t.Errorf("an unchanged frame wrote the image again: %q", got)
	}
	if got := frame(view("b", "AAAA")); strings.Contains(got, open) || !strings.Contains(got, "b") {
		t.Errorf("a change on another row wrote the image again, or was not drawn: %q", got)
	}
	// Another picture in the same cells is written, once.
	if got := frame(view("b", "BBBB")); strings.Count(got, open) != 1 || !strings.Contains(got, "BBBB") {
		t.Errorf("a new picture was not written once: %q", got)
	}
	// Removed: nothing is written for it.
	if got := frame("b\n      \n      \n      \nfooter"); strings.Contains(got, open) {
		t.Errorf("a removed image was written: %q", got)
	}
	if strings.Contains(log.String(), "kind=fallback") || strings.Contains(log.String(), "fallback_rows") {
		t.Errorf("a frame with an inline image fell back:\n%s", log.String())
	}
	if !p.cells.Valid() {
		t.Error("cell grid not held")
	}
}

// A row with no image renders as it did before inline images were kept: the
// same View with and without an image elsewhere writes the same bytes for
// its other rows, a hyperlink is still a hyperlink, and any other OSC is
// still not taken for an image.
func TestRowsWithoutAnImageAreUntouched(t *testing.T) {
	render := func(view string) string {
		var out, log bytes.Buffer
		p := imageProgram(&out, &log)
		p.model = staticModel{view: view}
		p.render()
		return out.String()
	}
	plain := "title\n\x1b[31mred\x1b[0m text\n\x1b]8;;https://example.com\x07link\x1b]8;;\x07\nfooter"
	without := render(plain + "\n      \n      \n      ")
	with := render(plain + "\n" + inlineView("AAAA"))
	const open = "\x1b]1337;File="
	i := strings.Index(with, "\x1b7"+open)
	if i < 0 {
		t.Fatalf("the image was not written: %q", with)
	}
	j := strings.Index(with[i:], "\x1b8") + i + 2
	if got := with[:i] + with[j:]; got != without {
		t.Errorf("the rows around an image differ from the same rows without one:\n with    %q\n without %q", got, without)
	}
	if !strings.Contains(without, "\x1b]8;;https://example.com") {
		t.Errorf("the hyperlink was lost: %q", without)
	}
	// OSC 1337 without a File argument, and other OSCs, are not images.
	for _, osc := range []string{"\x1b]1337;SetMark\x07", "\x1b]0;title\x07", "\x1b]52;c;AAAA\x07"} {
		if got := render("a" + osc + "b"); strings.Contains(got, "\x1b7"+osc) {
			t.Errorf("%q was kept as an image", osc)
		}
	}
}
