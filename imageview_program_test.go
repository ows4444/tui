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
