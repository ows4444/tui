package imageview

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/render"
	"github.com/ows4444/tui/layout"
)

func noisyPNG(t *testing.T) []byte {
	t.Helper()
	rng := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{uint8(rng.Intn(256)), uint8(rng.Intn(256)), uint8(rng.Intn(256)), 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestKittyChunks(t *testing.T) {
	data := noisyPNG(t)
	if base64.StdEncoding.EncodedLen(len(data)) <= MaxChunk {
		t.Fatal("test PNG too small to need chunking")
	}
	m := New(data, 10, 4, "alt")
	m.Kitty = true
	view := m.View()

	var chunks []string
	rest := view
	for {
		i := strings.Index(rest, "\x1b_G")
		if i < 0 {
			break
		}
		j := strings.Index(rest[i:], "\x1b\\")
		if j < 0 {
			t.Fatal("unterminated APC")
		}
		chunks = append(chunks, rest[i+3:i+j])
		rest = rest[i+j+2:]
	}
	if len(chunks) < 2 {
		t.Fatalf("got %d chunks, want several", len(chunks))
	}
	var payload strings.Builder
	for i, c := range chunks {
		ctl, body, ok := strings.Cut(c, ";")
		if !ok {
			t.Fatalf("chunk %d has no payload", i)
		}
		if len(body) > MaxChunk {
			t.Errorf("chunk %d payload %d bytes > %d", i, len(body), MaxChunk)
		}
		switch {
		case i == 0:
			for _, k := range []string{"a=T", "f=100", "c=10", "r=4", "m=1"} {
				if !strings.Contains(ctl, k) {
					t.Errorf("first chunk %q lacks %s", ctl, k)
				}
			}
		case i == len(chunks)-1:
			if ctl != "m=0" {
				t.Errorf("last chunk control %q, want m=0", ctl)
			}
		default:
			if ctl != "m=1" {
				t.Errorf("middle chunk %d control %q, want m=1", i, ctl)
			}
		}
		payload.WriteString(body)
	}
	got, err := base64.StdEncoding.DecodeString(payload.String())
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("payload does not decode to the PNG (err=%v)", err)
	}
}

func TestSmallPNGIsOneChunk(t *testing.T) {
	s := Transmit([]byte("tiny"), 3, 2)
	if strings.Count(s, "\x1b_G") != 1 || !strings.Contains(s, "m=0;") || strings.Contains(s, "m=1") {
		t.Errorf("Transmit = %q", s)
	}
}

func TestPlaceholderWithoutKitty(t *testing.T) {
	m := New(noisyPNG(t), 20, 5, "a cat")
	v := m.View()
	if strings.Contains(v, "\x1b_") {
		t.Fatal("graphics sequence without kitty support")
	}
	plain := ansi.StripANSI(v)
	if !strings.Contains(plain, "a cat") {
		t.Errorf("placeholder lacks alt text: %q", v)
	}
	if strings.Contains(plain, "\x1b") {
		t.Errorf("escape byte left after stripping styling: %q", plain)
	}
	for i := 0; i < len(plain); i++ {
		if plain[i] >= 0x80 {
			t.Fatalf("non-ASCII byte in placeholder: %q", plain)
		}
	}
}

func TestPlaceholderHasNoEscapeBytesUnstyled(t *testing.T) {
	m := New(nil, 12, 3, "x\x1b]52;c;ZXZpbA==\x07y")
	v := ansi.StripANSI(m.View())
	if strings.Contains(v, "\x1b") || strings.Contains(v, "\x07") {
		t.Errorf("alt text escaped: %q", v)
	}
}

func TestSizeInvariants(t *testing.T) {
	data := noisyPNG(t)
	for _, kitty := range []bool{false, true} {
		for w := 0; w <= 12; w++ {
			for h := 0; h <= 5; h++ {
				m := New(data, w, h, "alternative text")
				m.Kitty = kitty
				v := m.View()
				if w == 0 || h == 0 {
					if v != "" {
						t.Fatalf("%dx%d kitty=%v = %q, want empty", w, h, kitty, v)
					}
					continue
				}
				rows := strings.Split(v, "\n")
				if len(rows) != h {
					t.Fatalf("%dx%d kitty=%v: %d rows", w, h, kitty, len(rows))
				}
				for i, r := range rows {
					if got := ansi.Width(r); got != w {
						t.Fatalf("%dx%d kitty=%v row %d width %d", w, h, kitty, i, got)
					}
				}
			}
		}
	}
}

func TestWidthSkipsAPC(t *testing.T) {
	if got := ansi.Width("\x1b_Ga=T,m=0;AAAA\x1b\\abc"); got != 3 {
		t.Errorf("Width with APC = %d, want 3", got)
	}
}

func TestLayoutNodeAndLinearize(t *testing.T) {
	m := New(nil, 8, 3, "logo")
	if got := m.Linearize(); got != "logo" {
		t.Errorf("Linearize = %q", got)
	}
	m.Alt = ""
	if got := m.Linearize(); got != "Image" {
		t.Errorf("Linearize = %q", got)
	}
	out := m.LayoutNode().Render(layout.Size{W: 6, H: 2})
	if rows := strings.Split(out, "\n"); len(rows) != 2 || ansi.Width(rows[0]) != 6 {
		t.Errorf("Render = %q", out)
	}
}

// Images carry kitty ids: explicit, or derived and stable, and the view's
// transmit sequence names the id in both i= and p= so showing it again
// replaces its placement.
func TestImageIDs(t *testing.T) {
	data := noisyPNG(t)
	m := New(data, 10, 4, "alt")
	m.Kitty = true
	m.ID = 42
	if v := m.View(); !strings.Contains(v, "i=42,p=42,q=2,") {
		t.Errorf("View lacks the explicit id: %q", v[:80])
	}
	m.ID = 0
	id := m.imageID()
	if id == 0 || id != m.imageID() {
		t.Fatalf("derived id %d not stable and nonzero", id)
	}
	if !strings.Contains(m.View(), "i="+strconv.FormatUint(uint64(id), 10)+",") {
		t.Error("View lacks the derived id")
	}
	if other := New(data, 11, 4, "alt").imageID(); other == id {
		t.Error("different sizes share an id")
	}
	if got := Delete(42); got != "\x1b_Ga=d,d=I,i=42,q=2\x1b\\" {
		t.Errorf("Delete = %q", got)
	}
	if s := Transmit([]byte("x"), 1, 1); strings.Contains(s, "i=") {
		t.Errorf("Transmit has an id: %q", s)
	}
}

// The real View is drawn by the cell renderer without falling back, its
// placement is deleted when the image leaves the view or moves, and it is not
// re-sent while its rows are unchanged.
func TestViewThroughCellRendererLifecycle(t *testing.T) {
	m := New(noisyPNG(t), 10, 3, "alt")
	m.Kitty, m.ID = true, 9
	c := render.New()
	prev := 0
	draw := func(lines ...string) string {
		t.Helper()
		out, st, ok := c.Frame(render.Frame{
			Lines: lines, Max: max(len(lines), prev), PrevRows: prev, Width: 40, Height: 100,
			RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
		})
		if !ok || c.Reason() != "" || len(st.Fallbacks) != 0 {
			t.Fatalf("fell back: ok=%v reason=%q rows=%v", ok, c.Reason(), st.Fallbacks)
		}
		prev = max(len(lines), prev)
		return out
	}
	img := strings.Split(m.View(), "\n")
	del := Delete(9)
	if out := draw(append([]string{"title"}, img...)...); !strings.Contains(out, "\x1b_Ga=T,") || strings.Contains(out, del) {
		t.Fatalf("first frame: placement missing or deleted: %.60q", out)
	}
	if out := draw(append([]string{"TITLE"}, img...)...); strings.Contains(out, "\x1b_G") {
		t.Errorf("unchanged image rows re-sent or deleted: %.60q", out)
	}
	if out := draw(append([]string{"TITLE", "pad"}, img...)...); !strings.Contains(out, del) || strings.Index(out, del) > strings.Index(out, "\x1b_Ga=T,") {
		t.Errorf("moved image: want delete before re-place: %.80q", out)
	}
	if out := draw("TITLE", "pad", "gone"); !strings.Contains(out, del) || strings.Contains(out, "\x1b_Ga=T,") {
		t.Errorf("removed image: want delete only: %.80q", out)
	}
}

func envStub(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

// When $TMUX is set the graphics sequences are one DCS tmux; string with every
// ESC doubled; unwrapping it gives back the bare sequences.
func TestViewWrapsKittyInTmuxPassthrough(t *testing.T) {
	m := New(noisyPNG(t), 10, 3, "alt")
	m.Kitty, m.ID = true, 9
	bare := m.View()
	m.Getenv = envStub(map[string]string{"TMUX": "/tmp/tmux,1,0"})
	got := m.View()
	row0, rest, _ := strings.Cut(got, "\n")
	if !strings.HasPrefix(row0, "\x1bPtmux;\x1b\x1b_Ga=T,") || !strings.Contains(row0, "\x1b\x1b\\\x1b\\") {
		t.Fatalf("row 0 is not a tmux passthrough: %.80q", row0)
	}
	if strings.Contains(row0, "\x1b_G") && !strings.Contains(row0, "\x1b\x1b_G") {
		t.Error("an ESC of the wrapped sequence was not doubled")
	}
	bareRow0, bareRest, _ := strings.Cut(bare, "\n")
	if rest != bareRest {
		t.Error("only row 0 may change")
	}
	end := strings.LastIndex(row0, "\x1b\\")
	inner := strings.ReplaceAll(strings.TrimPrefix(row0[:end+2], "\x1bPtmux;"), "\x1b\x1b", "\x1b")
	if inner = strings.TrimSuffix(inner, "\x1b\\"); !strings.HasPrefix(bareRow0, inner) {
		t.Error("unwrapping the passthrough does not give the bare sequences")
	}
}

func TestViewEmitsKittyUnchangedWithoutAMultiplexer(t *testing.T) {
	m := New(noisyPNG(t), 10, 3, "alt")
	m.Kitty, m.ID = true, 9
	bare := m.View()
	for _, vars := range []map[string]string{{}, {"TMUX": "", "STY": ""}, {"TERM": "xterm"}} {
		m.Getenv = envStub(vars)
		if got := m.View(); got != bare {
			t.Errorf("env %v changed the output", vars)
		}
	}
	if strings.Contains(bare, "Ptmux") {
		t.Error("bare output mentions tmux")
	}
	m.Getenv = envStub(map[string]string{"STY": "1.pts"})
	if got := m.View(); !strings.HasPrefix(got, "\x1bP\x1b_Ga=T,") {
		t.Errorf("screen: %.40q", got)
	}
}

// The cell renderer keeps a tmux-wrapped image as one opaque segment: no
// fallback, and the rows are not re-sent while they are unchanged.
func TestWrappedViewThroughCellRenderer(t *testing.T) {
	m := New(noisyPNG(t), 10, 3, "alt")
	m.Kitty, m.ID = true, 9
	m.Getenv = envStub(map[string]string{"TMUX": "x"})
	c := render.New()
	prev := 0
	draw := func(lines ...string) string {
		t.Helper()
		out, st, ok := c.Frame(render.Frame{
			Lines: lines, Max: max(len(lines), prev), PrevRows: prev, Width: 40, Height: 100,
			RegionTop: func() string { return "" }, FitLine: func(s string) string { return s },
		})
		if !ok || c.Reason() != "" || len(st.Fallbacks) != 0 {
			t.Fatalf("fell back: ok=%v reason=%q rows=%v", ok, c.Reason(), st.Fallbacks)
		}
		prev = max(len(lines), prev)
		return out
	}
	img := strings.Split(m.View(), "\n")
	if out := draw(append([]string{"title"}, img...)...); !strings.Contains(out, "\x1bPtmux;") {
		t.Fatalf("first frame lost the wrapped placement: %.60q", out)
	}
	if out := draw(append([]string{"TITLE"}, img...)...); strings.Contains(out, "Ptmux;") {
		t.Errorf("unchanged wrapped rows re-sent: %.60q", out)
	}
	// The placement id is recovered through the wrapper, so a removed image is
	// deleted.
	if out := draw("TITLE", "pad", "gone"); !strings.Contains(out, Delete(9)) {
		t.Errorf("removed wrapped image not deleted: %.80q", out)
	}
}
