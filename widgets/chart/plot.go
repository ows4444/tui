package chart

import (
	"math"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// SeriesKind is how a Series is drawn.
type SeriesKind int

const (
	// Line joins the points with straight lines. It is the zero value.
	Line SeriesKind = iota
	// Scatter draws each point alone.
	Scatter
	// Bars draws a column from the bottom of the plot (or from zero, when the
	// y range includes it) up to each value.
	Bars
)

// Series is one line, set of points or set of bars of a Plot.
type Series struct {
	// Kind is how the values are drawn; the zero value is Line.
	Kind SeriesKind
	// Name labels the series in the legend and in Linearize.
	Name string
	// Values are plotted left to right, evenly spaced. NaN and infinite
	// values leave a gap.
	Values []float64
	// Color draws the line; nil takes the next colour of the theme's chart
	// palette (Primary, Secondary, Success, Warning, Error, Info).
	Color ansi.Color
}

// PlotOptions sizes and labels a Plot.
type PlotOptions struct {
	// Width and Height are the plot area in cells, not counting the axes,
	// their labels or the legend. Both must be positive.
	Width, Height int
	// Min and Max fix the y range. Both zero take the range of the data.
	Min, Max float64
	// YTicks is the number of labelled values down the y axis; 0 means 4. It is
	// at least 2 and at most Height.
	YTicks int
	// XLabels are placed along the x axis, evenly spaced from its left end to
	// its right; one that would run into the one before it is left out.
	XLabels []string
	// YFormat formats a y value for the axis; nil uses 4 significant digits.
	YFormat func(float64) string
	// Legend adds a line naming each series that has a Name.
	Legend bool
}

// Plot draws one or more series as lines on a shared y axis with labelled
// ticks, an x axis with optional labels and an optional legend. Lines are
// plotted in braille dots as LineChart does; where two series share a cell the
// later one is drawn. Like LineChart it is a pure function of its data; the
// result is the plot area plus a label column on the left and one or two
// lines below, so it is Width plus the widest label columns wide.
//
// Series are told apart by colour, so give each series a Name and set Legend,
// and keep the Linearize text beside it for readers who cannot see colour.
func Plot(series []Series, o PlotOptions, t theme.Theme) string {
	if o.Width <= 0 || o.Height <= 0 {
		return ""
	}
	lo, hi := plotRange(series, o)
	if lo == hi {
		lo, hi = lo-1, hi+1
	}
	n := 0
	for _, s := range series {
		n = max(n, len(s.Values))
	}

	dotCols, dotRows := o.Width*2, o.Height*4
	palette := []ansi.Color{t.Primary, t.Secondary, t.Success, t.Warning, t.Error, t.Info}
	color := func(si int) ansi.Color {
		if c := series[si].Color; c != nil {
			return c
		}
		return palette[si%len(palette)]
	}

	canvas := NewCanvas(o.Width, o.Height)
	// Bars rise from the bottom of the plot, or from zero when the range
	// includes it.
	base := dotRows - 1
	if lo < 0 && hi > 0 {
		base = int((1-(0-lo)/(hi-lo))*float64(dotRows-1) + 0.5)
	}
	for si, s := range series {
		prevX, prevY, have := 0, 0, false
		for i, v := range s.Values {
			if !finite(v) {
				have = false
				continue
			}
			x := 0
			if n > 1 {
				x = i * (dotCols - 1) / (n - 1)
			}
			frac := math.Min(math.Max((v-lo)/(hi-lo), 0), 1)
			y := int((1-frac)*float64(dotRows-1) + 0.5)
			switch s.Kind {
			case Scatter:
				canvas.Set(x, y, color(si))
			case Bars:
				canvas.Line(x, base, x, y, color(si))
				if x+1 < dotCols {
					canvas.Line(x+1, base, x+1, y, color(si))
				}
			default:
				if have {
					canvas.Line(prevX, prevY, x, y, color(si))
				} else {
					canvas.Set(x, y, color(si))
				}
			}
			prevX, prevY, have = x, y, true
		}
	}
	plotRows := canvas.Rows(t)

	// y axis labels.
	k := o.YTicks
	if k <= 0 {
		k = 4
	}
	k = min(max(k, 2), o.Height)
	format := o.YFormat
	if format == nil {
		format = func(v float64) string { return strconv.FormatFloat(v, 'g', 4, 64) }
	}
	tickRow := make([]string, o.Height)
	labelW := 0
	for j := 0; j < k; j++ {
		row := j * (o.Height - 1) / (k - 1)
		tickRow[row] = format(hi - float64(j)*(hi-lo)/float64(k-1))
		labelW = max(labelW, ansi.Width(tickRow[row]))
	}

	g := t.GlyphSet()
	// The axis, its ticks and its corner come from the glyph set's two rules:
	// a tick is a short horizontal rule that meets the vertical axis.
	axisV, axisH := g.RuleV, g.RuleH
	tick, corner := axisH, axisH
	muted := ansi.NewStyle().Foreground(t.Muted)

	var b strings.Builder
	for r := 0; r < o.Height; r++ {
		label := tickRow[r]
		b.WriteString(strings.Repeat(" ", labelW-ansi.Width(label)) + muted.Render(label))
		if label != "" {
			b.WriteString(muted.Render(tick))
		} else {
			b.WriteString(muted.Render(axisV))
		}
		b.WriteString(plotRows[r])
		b.WriteByte('\n')
	}
	b.WriteString(strings.Repeat(" ", labelW) + muted.Render(corner+strings.Repeat(axisH, o.Width)))

	if len(o.XLabels) > 0 {
		line := make([]rune, o.Width)
		for i := range line {
			line[i] = ' '
		}
		end := -1 // last column used
		for j, lab := range o.XLabels {
			pos := 0
			if len(o.XLabels) > 1 {
				pos = j * (o.Width - 1) / (len(o.XLabels) - 1)
			}
			w := ansi.Width(lab)
			start := min(max(pos-w/2, 0), max(o.Width-w, 0))
			if start <= end || w == 0 || start+w > o.Width {
				continue
			}
			copy(line[start:], []rune(lab))
			end = start + w
		}
		b.WriteString("\n" + strings.Repeat(" ", labelW+1) + strings.TrimRight(string(line), " "))
	}

	if o.Legend {
		var parts []string
		for si, s := range series {
			if s.Name != "" {
				parts = append(parts, ansi.NewStyle().Foreground(color(si)).Render(g.Dot)+" "+s.Name)
			}
		}
		if len(parts) > 0 {
			b.WriteString("\n" + strings.Repeat(" ", labelW+1) + strings.Join(parts, "  "))
		}
	}
	return b.String()
}

// plotRange is the y range: the options' when either is set, else the range
// of the finite values (0, 0 when there are none).
func plotRange(series []Series, o PlotOptions) (lo, hi float64) {
	if o.Min != 0 || o.Max != 0 {
		return o.Min, o.Max
	}
	first := true
	for _, s := range series {
		for _, v := range s.Values {
			if !finite(v) {
				continue
			}
			if first || v < lo {
				lo = v
			}
			if first || v > hi {
				hi = v
			}
			first = false
		}
	}
	return lo, hi
}

// LinearizePlot summarises a Plot as text for accessible output: the y range,
// then one line per series with its name (or "Series n"), count, min, max,
// average and last value, as LinearizeLineChart gives for one.
func LinearizePlot(series []Series, o PlotOptions) string {
	if len(series) == 0 {
		return "Plot: no data"
	}
	lo, hi := plotRange(series, o)
	lines := []string{"Plot: " + strconv.Itoa(len(series)) + " series, y from " + num(lo) + " to " + num(hi)}
	for i, s := range series {
		name := s.Name
		if name == "" {
			name = "Series " + strconv.Itoa(i+1)
		}
		lines = append(lines, summary(name, s.Values))
	}
	return strings.Join(lines, "\n")
}
