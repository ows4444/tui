package tui_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/appshell"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/clipboard"
	"github.com/ows4444/tui/clockview"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/faces"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/helpscreen"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/loadingbar"
	applog "github.com/ows4444/tui/logview"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/virtuallist"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
	"github.com/ows4444/tui/wizard"
)

// rendered is one widget's output and, when it has a layout node, the node.
type rendered struct {
	name string
	out  string
	node layout.Node
}

// themed returns a copy of m with its Theme field, if it has one, set to th.
func themed[M any](m M, th theme.Theme) M {
	v := reflect.ValueOf(&m).Elem()
	if v.Kind() == reflect.Struct {
		if f := v.FieldByName("Theme"); f.IsValid() && f.CanSet() && f.Type() == reflect.TypeOf(th) {
			f.Set(reflect.ValueOf(th))
		}
	}
	return m
}

// model records a stateful widget: its View, when it has one, and its layout
// node, when it has one.
func model(name string, m any) []rendered {
	var out []rendered
	if v, ok := m.(interface{ View() string }); ok {
		out = append(out, rendered{name: name + ".View", out: v.View()})
	}
	if l, ok := m.(interface{ LayoutNode() layout.Node }); ok {
		n := l.LayoutNode()
		out = append(out, rendered{name: name + ".LayoutNode", out: layout.Draw(n, layout.Unconstrained()), node: n})
	}
	return out
}

// everyWidget renders every widget in the library under th, using only ASCII
// data, so any non-ASCII rune in the output is a glyph the widget chose. It
// covers each stateless function in package widgets and the default state of
// each stateful model that draws.
func everyWidget(t *testing.T, th theme.Theme) []rendered {
	t.Helper()
	var out []rendered
	add := func(name, s string) { out = append(out, rendered{name: name, out: s}) }
	addAll := func(rs []rendered) { out = append(out, rs...) }

	// package widgets
	rows := []widgets.TreeRow{{Key: "region", Value: "eu", Children: []widgets.TreeRow{{Key: "az", Value: "1"}}}, {Key: "tier", Value: "gold"}}
	flat := []widgets.TreeRow{{Key: "region", Value: "eu"}, {Key: "tier", Value: "gold"}}
	tv := treeview.New(treeview.Node{Label: "root", Children: []treeview.Node{{Label: "a"}, {Label: "b", Children: []treeview.Node{{Label: "c"}}}}})
	long := strings.Repeat("a fairly long line of code that will not fit ", 4)
	table := [][]string{{"api", "up"}, {"worker", "down"}}
	for _, v := range []widgets.Variant{widgets.VariantSuccess, widgets.VariantWarning, widgets.VariantError, widgets.VariantInfo, widgets.VariantNeutral} {
		add(fmt.Sprintf("widgets.Alert(%v)", v), widgets.Alert("disk almost full", v, th, 30))
		add(fmt.Sprintf("widgets.Banner(%v)", v), widgets.Banner("maintenance tonight", v, th, 30))
		add(fmt.Sprintf("widgets.Badge(%v)", v), widgets.Badge("new", v, th))
		add(fmt.Sprintf("widgets.StatusIndicator(%v)", v), widgets.StatusIndicator("api", v, th))
	}
	add("chart.BarChart", chart.BarChart([]chart.BarItem{{Label: "a", Value: 3}, {Label: "bb", Value: 9}}, 30, th))
	add("widgets.BigText", widgets.BigText("AB 12", widgets.FontBlock, th))
	add("widgets.Breadcrumb", widgets.Breadcrumb([]string{"home", "docs", "api"}, th))
	add("widgets.Card", widgets.Card("Title", "body text", th, 24))
	add("widgets.Box", widgets.Box("Title", "body text", th, 24))
	add("widgets.Panel", widgets.Panel("Title", "body text", th, 24))
	add("widgets.Panel(untitled)", widgets.Panel("", "body text", th, 24))
	add("widgets.Center", widgets.Center("mid", 11, 3))
	add("widgets.Checkbox(on)", widgets.Checkbox("agree", true, true, th))
	add("widgets.Checkbox(off)", widgets.Checkbox("agree", false, false, th))
	for _, streaming := range []bool{true, false} {
		add(fmt.Sprintf("widgets.ChatMessage(streaming=%v)", streaming),
			widgets.ChatMessage(widgets.SenderAssistant, "", "hello there", time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), streaming, th))
	}
	add("widgets.CodeBlock(numbers)", widgets.CodeBlock("x := 1\n"+long+"\nreturn x", 30, true, th))
	add("widgets.CodeBlock(plain)", widgets.CodeBlock("x := 1\n"+long, 30, false, th))
	add("widgets.CodeBlockLang", widgets.CodeBlockLang("func main() {\n\t"+long+"\n}", "go", 30, true, th))
	add("widgets.DiffView", widgets.DiffView("--- a\n+++ b\n@@ -1 +1 @@\n-old\n+new", 30, th))
	add("widgets.Divider", widgets.DividerWith(20, th))
	add("widgets.DividerLabel", widgets.DividerLabelWith(20, "section", th))
	add("widgets.ErrorBoundary", widgets.ErrorBoundary(func() string { panic("boom") }, "fallback", th))
	add("widgets.Form", widgets.Form("one", "two"))
	add("widgets.FormField", widgets.FormField("Name", "field", "required", th))
	add("chart.Gauge", chart.Gauge(40, 20, th))
	add("widgets.Gradient", widgets.Gradient("gradient", []ansi.RGB{{R: 255}, {B: 255}}, false, th))
	add("widgets.Header", widgets.Header("Title", th))
	add("widgets.HeaderWithAccessory", widgets.HeaderWithAccessory("Title", "v1", 30, th))
	add("chart.HeatMap", chart.HeatMap([][]float64{{0, 1, 2}, {3, 4, 5}}, th))
	add("widgets.InfoBox(flat)", widgets.InfoBox("Info", flat, th, 26))
	add("widgets.InfoBox(nested)", widgets.InfoBox("Info", rows, th, 26))
	add("widgets.KeyHint", widgets.KeyHint("q", "quit"))
	add("widgets.KeyHints", widgets.KeyHints("  ", widgets.Hint{Key: "q", Action: "quit"}, widgets.Hint{Key: "?", Action: "help"}))
	add("widgets.KeyValue", widgets.KeyValue([]widgets.KV{{Key: "a", Value: "1"}, {Key: "bb", Value: "2"}}, th))
	add("chart.LineChart", chart.LineChart([]float64{1, 4, 2, 8, 5}, 20, 5, th))
	add("widgets.Link", widgets.Link("docs", "https://example.com", true, th))
	add("widgets.List(bullet)", widgets.ListWith([]string{"one", "two"}, widgets.MarkerBullet, th))
	add("widgets.List(number)", widgets.ListWith([]string{"one", "two"}, widgets.MarkerNumber, th))
	add("widgets.MultiProgress", widgets.MultiProgress([]widgets.ProgressItem{{Label: "a", Percent: 30}, {Label: "b", Percent: 80}}, 30, th))
	add("widgets.Pagination", widgets.Pagination(2, 5, th))
	add("widgets.PaginationDots", widgets.PaginationDots(2, 5, th))
	add("widgets.ProgressBar", widgets.ProgressBar(0.4, 20, th))
	add("widgets.ProgressCircle", widgets.ProgressCircle(40, 6, th))
	add("widgets.ProgressCircle(wide)", widgets.ProgressCircle(40, 16, th))
	add("treeview.Sidebar", treeview.Sidebar("Nav", tv, "", map[string]string{"0": "*"}, map[string]string{"0": "3"}, th, 26))
	add("widgets.Spacer", widgets.Spacer(3, 2))
	add("chart.Sparkline", chart.SparklineWith([]float64{1, 3, 2, 8, 5, 9, 4}, th))
	add("widgets.Stepper", widgets.Stepper([]string{"Build", "Test", "Deploy"}, 1, th))
	add("widgets.Table", widgets.Table([]string{"Service", "State"}, table, th))
	add("widgets.TableRows", widgets.TableRowsWith([]string{"Service", "State"}, table, ansi.Style{}, ansi.Style{}, func(int) ansi.Style { return ansi.Style{} }, th))
	add("widgets.Tag", widgets.Tag("beta", widgets.TagSolid, widgets.VariantInfo, th))
	add("widgets.TokenCounter", widgets.TokenCounter(300, 1000, th))
	add("widgets.Toggle(on)", widgets.Toggle("dark mode", true, true, th))
	add("widgets.Toggle(off)", widgets.Toggle("dark mode", false, false, th))
	add("widgets.Tooltip", widgets.Tooltip("hint text", th, 20))
	add("widgets.TooltipOverlay", widgets.TooltipOverlay("base line\nsecond line", "tip", 2, 0, th))
	add("widgets.UsageMonitor", widgets.UsageMonitor("Usage", []widgets.UsageStat{{Label: "cpu", Value: 40, Limit: 100}, {Label: "mem", Value: 70, Limit: 100}}, th))
	add("accordion.ToolCall", accordion.ToolCall("read", "file.txt", accordion.ToolCallSuccess, "ok").Content)
	add("accordion.ThinkingSection", accordion.ThinkingSectionWith("thinking", "reasoning text", 12, time.Second, true, th).Content)

	// other packages
	add("markdown.Render", markdown.Render("# Title\n\n- one\n  - nested\n- two\n\n> quoted\n\n---\n\n1. first\n2. second\n\n`code` and **bold**", 40, th))
	acc := themed(accordion.New(accordion.Section{Title: "A", Content: "one"}, accordion.Section{Title: "B", Content: "two"}), th)
	addAll(model("accordion", acc))
	addAll(model("appshell", themed(appshell.New("App", 40, 6), th)))
	addAll(model("autocomplete", themed(autocomplete.New("alpha", "beta"), th)))
	addAll(model("clipboard", themed(clipboard.New("text", "label"), th)))
	addAll(model("clock", themed(clockview.New(clockview.ModeClock), th)))
	addAll(model("colorpicker", themed(colorpicker.New(ansi.Red, ansi.Green), th)))
	addAll(model("commandpalette", themed(commandpalette.New(commandpalette.Command{Name: "open", Description: "open a file"}), th)))
	addAll(model("confirm", themed(confirm.New("Sure?"), th)))
	dt := themed(datatable.New([]string{"Service", "Description", "State"}, [][]string{{"api", strings.Repeat("long text ", 5), "up"}, {"worker", "queue", "down"}}), th)
	addAll(model("datatable", dt))
	addAll(model("datepicker", themed(datepicker.New(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)), th)))
	addAll(model("dialog", themed(dialog.New("Title", "message"), th)))
	addAll(model("drawer", themed(drawer.New("content"), th)))
	addAll(model("emailinput", themed(emailinput.New(), th)))
	addAll(model("errorretry", themed(errorretry.New("failed", 0), th)))
	addAll(model("errorretry(retryable)", themed(errorretry.New("failed", 3), th)))
	addAll(model("faces", themed(faces.New(), th)))
	addAll(model("filepicker", themed(filepicker.New(t.TempDir()), th)))
	fm := form.New(
		form.Field{Name: "n", Label: "Name", Validators: []form.Validator{form.Required()}},
		form.Field{Name: "p", Label: "Pass", Secret: true},
		form.Field{Name: "plan", Label: "Plan", Kind: form.FieldSelect, Options: []string{"Free", "Pro"}},
		form.Field{Name: "t", Label: "Terms", Kind: form.FieldCheckbox, Validators: []form.Validator{func(v string) error {
			if v != "true" {
				return fmt.Errorf("accept the terms")
			}
			return nil
		}}},
	)
	fm = themed(fm, th)
	fm.Focus()
	fm, _ = fm.Submit()
	addAll(model("form", fm))
	addAll(model("helpscreen", themed(helpscreen.New(widgets.Hint{Key: "q", Action: "quit"}), th)))
	if js, err := treeview.FromJSON([]byte(`{"a":[1,2],"b":{"c":"d"}}`)); err == nil {
		addAll(model("jsonviewer", themed(js, th)))
	}
	addAll(model("loadingbar", themed(loadingbar.New(20), th)))
	lg := themed(applog.New(30, 4), th)
	lg.Append("a log line")
	addAll(model("log", lg))
	mi := themed(maskedinput.New(), th)
	mi.SetValue("secret")
	addAll(model("maskedinput", mi))
	addAll(model("menu", themed(menu.New([]menu.Item{{Label: "One", Children: []menu.Item{{Label: "Sub"}}}, {Label: "Two"}}), th)))
	addAll(model("multiselect", themed(multiselect.NewStrings("a", "b", "c"), th)))
	nc := themed(notificationcenter.New(3, 30), th)
	nc.Push(notificationcenter.Notification{Message: "saved"})
	addAll(model("notificationcenter", nc))
	addAll(model("numberinput", themed(numberinput.New(), th)))
	pw := themed(passwordinput.New(), th)
	pw.SetValue("hunter2")
	addAll(model("passwordinput", pw))
	addAll(model("pathinput", themed(textinput.NewPath(), th)))
	addAll(model("picker", themed(picker.NewStrings("a", "b", "c"), th)))
	addAll(model("popover", themed(popover.New("content", 2, 1), th)))
	addAll(model("searchinput", themed(textinput.NewSearch(), th)))
	addAll(model("skeleton", themed(func() skeleton.Model { m := skeleton.New(); m.Width, m.Lines = 12, 2; return m }(), th)))
	addAll(model("spinner", themed(spinner.New(), th)))
	st := themed(streamtext.New(), th)
	st.Width = 30
	addAll(model("streamtext", st))
	addAll(model("streamtext(typewriter)", themed(streamtext.NewTypewriter(), th)))
	addAll(model("tabs", themed(tabs.New("One", "Two"), th)))
	addAll(model("taginput", themed(taginput.New(), th)))
	ta := themed(textarea.New(), th)
	ta.SetValue("line one\nline two")
	addAll(model("textarea", ta))
	addAll(model("textinput", themed(textinput.New(), th)))
	addAll(model("toast", themed(toast.New("saved"), th)))
	addAll(model("toolapproval", themed(toolapproval.New("run", "runs a command", toolapproval.RiskHigh), th)))
	addAll(model("treeview", themed(tv, th)))
	vp := themed(viewport.New(20, 3), th)
	vp.SetContent("one\ntwo\nthree\nfour")
	addAll(model("viewport", vp))
	addAll(model("virtuallist", themed(virtuallist.New(50, 4, func(i int) string { return fmt.Sprint("item ", i) }), th)))
	addAll(model("wizard", themed(wizard.New("One", "Two", "Three"), th)))
	return out
}

func nonASCIIRune(s string) (rune, bool) {
	for _, r := range ansi.StripANSI(s) {
		if r >= 0x80 {
			return r, true
		}
	}
	return 0, false
}

// Under the ASCII theme no widget draws a non-ASCII rune. Every input is ASCII,
// so a non-ASCII rune is a glyph the widget chose without asking theme.Glyphs.
func TestASCIIThemeDrawsOnlySevenBitOutput(t *testing.T) {
	outs := everyWidget(t, theme.DarkTheme().ASCII())
	if len(outs) < 120 {
		t.Fatalf("only %d widget outputs were rendered; the harness lost widgets", len(outs))
	}
	for _, o := range outs {
		if r, bad := nonASCIIRune(o.out); bad {
			t.Errorf("%s draws %q (U+%04X) under the ASCII theme:\n%s", o.name, r, r, ansi.StripANSI(o.out))
		}
	}
}

// The harness is not vacuous: under the default theme many widgets do draw
// non-ASCII glyphs, so an empty result above would mean the test saw nothing.
func TestUnicodeThemeDrawsNonASCIIGlyphs(t *testing.T) {
	var drawing []string
	for _, o := range everyWidget(t, theme.DarkTheme()) {
		if _, ok := nonASCIIRune(o.out); ok {
			drawing = append(drawing, o.name)
		}
	}
	n := len(drawing)
	t.Logf("%d widget outputs draw non-ASCII glyphs under theme.Dark: %s", n, strings.Join(drawing, ", "))
	if n < 25 {
		t.Errorf("only %d widget outputs draw non-ASCII glyphs under theme.Dark; the harness is not exercising the glyph paths", n)
	}
}

// The layout nodes keep their exact size under the ASCII glyphs.
func TestASCIIThemeLayoutNodesKeepTheirExactSize(t *testing.T) {
	sizes := []layout.Size{{W: 40, H: 10}, {W: 80, H: 24}, {W: 300, H: 80}, {W: 12, H: 3}, {W: 5, H: 2}}
	checked := 0
	for _, o := range everyWidget(t, theme.DarkTheme().ASCII()) {
		if o.node == nil {
			continue
		}
		for _, s := range sizes {
			got := o.node.Render(s)
			lines := strings.Split(got, "\n")
			if got == "" || len(lines) != s.H {
				t.Errorf("%s at %v: %d rows, want %d", o.name, s, len(lines), s.H)
				continue
			}
			for i, l := range lines {
				if w := ansi.Width(l); w != s.W {
					t.Errorf("%s at %v: row %d is %d wide, want %d", o.name, s, i, w, s.W)
					break
				}
			}
			if r, bad := nonASCIIRune(got); bad {
				t.Errorf("%s at %v draws %q under the ASCII theme", o.name, s, r)
			}
			checked++
		}
	}
	if checked < 200 {
		t.Errorf("only %d layout renders were checked", checked)
	}
}
