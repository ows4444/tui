package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func TestNodeExactSize(t *testing.T) {
	th := theme.Theme{}
	rows := [][]string{{"a", "b"}, {"c", "d"}}
	items := []ProgressItem{{Label: "x", Percent: 0.5}}
	cases := map[string]string{
		"Banner":          Banner("hello", VariantInfo, th, 30),
		"BigText":         BigText("Hi", 0, th),
		"Alert":           Alert("careful", VariantWarning, th, 30),
		"ChatMessage":     ChatMessage(0, "me", "hello there", time.Unix(0, 0), false, th),
		"Breadcrumb":      Breadcrumb([]string{"a", "b", "c"}, th),
		"Badge":           Badge("new", VariantSuccess, th),
		"Box":             Box("t", "content\nmore", th, 20),
		"BarChart":        BarChart([]BarItem{{Label: "a", Value: 3}, {Label: "b", Value: 5}}, 30, th),
		"Center":          Center("mid", 10, 3),
		"DiffView":        DiffView("@@ -1 +1 @@\n-a\n+b\n", 40, th),
		"Gradient":        Gradient("gradient text", nil, true, th),
		"FormField":       FormField("Name", "value", "bad", th),
		"Form":            Form("one", "two"),
		"CodeBlock":       CodeBlock("x := 1\ny := 2", 30, true, th),
		"CodeBlockLang":   CodeBlockLang("x := 1", "go", 30, true, th),
		"Card":            Card("t", "body", th, 20),
		"Checkbox":        Checkbox("ok", true, true, th),
		"HeatMap":         HeatMap([][]float64{{0, 1}, {0.5, 0.2}}, th),
		"ErrorBoundary":   ErrorBoundary(func() string { return "fine" }, "fallback", th),
		"Pagination":      Pagination(2, 9, th),
		"PaginationDots":  PaginationDots(2, 9, th),
		"KeyValue":        KeyValue([]KV{{Key: "k", Value: "v"}}, th),
		"KeyHint":         KeyHint("q", "quit"),
		"KeyHints":        KeyHints(" ", Hint{Key: "q", Action: "quit"}),
		"InfoBox":         InfoBox("info", []TreeRow{{Key: "a", Value: "b"}}, th, 30),
		"LineChart":       LineChart([]float64{1, 3, 2, 5}, 20, 4, th),
		"Divider":         Divider(20),
		"DividerWith":     DividerWith(20, th),
		"DividerLabel":    DividerLabel(20, "lbl"),
		"DividerLabelW":   DividerLabelWith(20, "lbl", th),
		"Link":            Link("text", "http://x", true, th),
		"Gauge":           Gauge(0.4, 20, th),
		"Spacer":          Spacer(4, 2),
		"ProgressBar":     ProgressBar(0.4, 20, th),
		"MultiProgress":   MultiProgress(items, 30, th),
		"ProgressCircle":  ProgressCircle(0.4, 5, th),
		"Stepper":         Stepper([]string{"a", "b", "c"}, 1, th),
		"Sparkline":       Sparkline([]float64{1, 2, 3}),
		"SparklineWith":   SparklineWith([]float64{1, 2, 3}, th),
		"StatusIndicator": StatusIndicator("ok", VariantSuccess, th),
		"Header":          Header("title", th),
		"HeaderAccessory": HeaderWithAccessory("title", "acc", 30, th),
		"Tag":             Tag("tag", 0, VariantInfo, th),
		"TokenCounter":    TokenCounter(10, 100, th),
		"Toggle":          Toggle("on", true, false, th),
		"List":            List([]string{"a", "b"}, 0),
		"ListWith":        ListWith([]string{"a", "b"}, 0, th),
		"Tooltip":         Tooltip("tip", th, 20),
		"TooltipOverlay":  TooltipOverlay("base line\nsecond", "tip", 2, 0, th),
		"Panel":           Panel("t", "body", th, 20),
		"UsageMonitor":    UsageMonitor("use", []UsageStat{{Label: "a"}}, th),
		"Table":           Table([]string{"h1", "h2"}, rows, th),
		"TableRows":       TableRows([]string{"h1", "h2"}, rows, ansi.Style{}, ansi.Style{}, func(int) ansi.Style { return ansi.Style{} }),
		"TableRowsWith":   TableRowsWith([]string{"h1", "h2"}, rows, ansi.Style{}, ansi.Style{}, func(int) ansi.Style { return ansi.Style{} }, th),
		"Empty":           "",
		"Wide":            strings.Repeat("界", 60) + "\n" + strings.Repeat("x", 200),
	}
	sizes := []layout.Size{{W: 1, H: 1}, {W: 20, H: 3}, {W: 80, H: 24}}
	for name, s := range cases {
		n := Node(s)
		for _, sz := range sizes {
			out := n.Render(sz)
			lines := strings.Split(out, "\n")
			if len(lines) != sz.H {
				t.Errorf("%s @%dx%d: %d rows", name, sz.W, sz.H, len(lines))
				continue
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != sz.W {
					t.Errorf("%s @%dx%d row %d: width %d", name, sz.W, sz.H, i, w)
				}
			}
		}
	}
}
