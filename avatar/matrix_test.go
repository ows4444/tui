package avatar

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
)

// outputs are the ways a Model is read. A field that is part of the picture
// must reach every output that draws the picture, and one that is not must
// reach none of them.
type outputs map[string]string

func read(m Model) outputs {
	body, eyes, plate := m.Colors()
	buf := cellbuf.New(max(m.Width, 1), max(m.Height, 1))
	m.DrawCells(buf, buf.Bounds())
	return outputs{
		"View":   m.View(),
		"Cells":  buf.String(),
		"PNG":    string(m.PNG(64)),
		"SVG":    m.SVG(),
		"Colors": fmt.Sprint(body, eyes, plate),
		"Speech": m.Linearize(),
		"Bounds": fmt.Sprint(m.Bounds()),
		"Shape":  m.Shape().String(),
	}
}

func (o outputs) differsFrom(p outputs) []string {
	var out []string
	for k, v := range o {
		if v != p[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func TestEveryFieldReachesEveryOutputItShould(t *testing.T) {
	deg := func(h float64) *float64 { return &h }
	base := New("Ada")
	base.Width, base.Height = 16, 8
	base.Silhouette = SilhouetteRound // a body that can wear an accessory
	base.Accessory = AccessoryNone

	picture := []string{"View", "Cells", "PNG", "SVG"}
	rows := []struct {
		field  string
		change func(*Model)
		want   []string
	}{
		{"Name", func(m *Model) { m.Name = "Grace" }, append([]string{"Colors", "Speech", "Shape?"}, picture...)},
		{"Raw", func(m *Model) { m.Raw = true }, append([]string{"Colors?"}, picture...)},
		{"Width", func(m *Model) { m.Width = 24 }, []string{"View", "Cells", "Bounds"}},
		{"Height", func(m *Model) { m.Height = 12 }, []string{"View", "Cells", "Bounds"}},
		{"Background", func(m *Model) { m.Background = BackgroundCircle }, append([]string{"Colors?"}, picture...)},
		{"Hue", func(m *Model) { m.Hue = deg(0) }, append([]string{"Colors"}, picture...)},
		{"Tone", func(m *Model) { m.Tone = ToneInk }, append([]string{"Colors"}, picture...)},
		{"Silhouette", func(m *Model) { m.Silhouette = SilhouetteSun }, append([]string{"Shape"}, picture...)},
		{"Accessory", func(m *Model) { m.Accessory = AccessoryBunny }, picture},
		{"Eyes", func(m *Model) { m.Eyes = EyesDot }, picture},
		{"Expression", func(m *Model) { m.Expression = ExpressionMad }, append([]string{"Speech", "Colors"}, picture...)},
		{"LookX", func(m *Model) { m.LookX = 1 }, picture},
		{"LookY", func(m *Model) { m.LookY = 1 }, picture},
		{"Motion", func(m *Model) { m.Motion = motion.Reduced }, nil},
		{"Theme(light)", func(m *Model) { m.Theme = theme.LightTheme() }, append([]string{"Colors"}, picture...)},
		{"Decorative", func(m *Model) { m.Decorative = true }, []string{"Speech"}},
		{"X", func(m *Model) { m.X = 5 }, []string{"Bounds"}},
		{"Y", func(m *Model) { m.Y = 5 }, []string{"Bounds"}},
	}
	before := read(base)
	if before["Cells"] != cellsOf(base) {
		t.Fatal("DrawCells differs from View for the base model")
	}
	t.Logf("%-14s %s", "field", "outputs it changes")
	for _, r := range rows {
		m := base
		r.change(&m)
		got := before.differsFrom(read(m))
		want := map[string]bool{}
		for _, w := range r.want {
			want[strings.TrimSuffix(w, "?")] = !strings.HasSuffix(w, "?")
		}
		t.Logf("%-14s %v", r.field, got)
		for _, g := range got {
			if _, ok := want[g]; !ok {
				t.Errorf("%s changed %s, which it should not reach", r.field, g)
			}
		}
		for w, must := range want {
			if must && !contains(got, w) {
				t.Errorf("%s did not reach %s", r.field, w)
			}
		}
		// The cached Model and the uncached one draw the same, and DrawCells
		// equals View, for every change.
		fresh := m
		fresh.cache = nil
		if m.View() != fresh.View() || string(m.PNG(64)) != string(fresh.PNG(64)) {
			t.Errorf("%s: the cache is stale", r.field)
		}
		if read(m)["Cells"] != cellsOf(m) {
			t.Errorf("%s: DrawCells differs from View", r.field)
		}
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// cellsOf is View parsed into a grid, as the renderer would draw it.
func cellsOf(m Model) string {
	buf := cellbuf.New(max(m.Width, 1), max(m.Height, 1))
	tui.DrawView(buf, buf.Bounds(), m.View())
	return buf.String()
}
