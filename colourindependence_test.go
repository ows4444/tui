package tui_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/errorretry"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

// statefulWidget is one status-bearing widget and every state that must be
// told apart without colour.
type statefulWidget struct {
	name   string
	states []state
}

type state struct{ name, out string }

var allVariants = []widgets.Variant{widgets.VariantNeutral, widgets.VariantInfo, widgets.VariantSuccess, widgets.VariantWarning, widgets.VariantError}

func variantStates(render func(widgets.Variant) string) []state {
	names := map[widgets.Variant]string{widgets.VariantNeutral: "neutral", widgets.VariantInfo: "info", widgets.VariantSuccess: "success", widgets.VariantWarning: "warning", widgets.VariantError: "error"}
	var out []state
	for _, v := range allVariants {
		out = append(out, state{names[v], render(v)})
	}
	return out
}

// colourIndependentWidgets is the explicit list of widgets whose meaning is a
// state, not just a value. A new status-bearing widget must be added here by
// hand; TestColourIndependence then proves its states differ as plain text.
func colourIndependentWidgets(th theme.Theme) []statefulWidget {
	var ws []statefulWidget
	add := func(name string, states ...state) { ws = append(ws, statefulWidget{name, states}) }

	// Variant-coloured widgets.
	add("widgets.Alert", variantStates(func(v widgets.Variant) string { return widgets.Alert("disk almost full", v, th, 30) })...)
	add("widgets.Banner", variantStates(func(v widgets.Variant) string { return widgets.Banner("maintenance tonight", v, th, 30) })...)
	add("widgets.Badge", variantStates(func(v widgets.Variant) string { return widgets.Badge("new", v, th) })...)
	add("widgets.StatusIndicator", variantStates(func(v widgets.Variant) string { return widgets.StatusIndicator("api", v, th) })...)
	add("widgets.Tag(solid)", variantStates(func(v widgets.Variant) string { return widgets.Tag("beta", widgets.TagSolid, v, th) })...)
	add("toast levels", variantStates(func(v widgets.Variant) string {
		m := toast.New("saved")
		m.Theme, m.Variant = th, v
		m.Show()
		return m.Render("base line one\nbase line two\nbase line three\nbase line four")
	})...)

	// Toggles.
	add("widgets.Checkbox", state{"on", widgets.Checkbox("agree", true, false, th)}, state{"off", widgets.Checkbox("agree", false, false, th)})
	add("widgets.Toggle", state{"on", widgets.Toggle("dark", true, false, th)}, state{"off", widgets.Toggle("dark", false, false, th)})

	// Progress and steps.
	add("widgets.Stepper", state{"step 0", widgets.Stepper([]string{"A", "B", "C"}, 0, th)}, state{"step 1", widgets.Stepper([]string{"A", "B", "C"}, 1, th)}, state{"step 2", widgets.Stepper([]string{"A", "B", "C"}, 2, th)}, state{"done", widgets.Stepper([]string{"A", "B", "C"}, 3, th)})
	add("widgets.ProgressBar", state{"0", widgets.ProgressBar(0, 20, th)}, state{"0.5", widgets.ProgressBar(0.5, 20, th)}, state{"1", widgets.ProgressBar(1, 20, th)})
	add("chart.Gauge", state{"low", chart.Gauge(0.1, 20, th)}, state{"mid", chart.Gauge(0.6, 20, th)}, state{"high", chart.Gauge(0.95, 20, th)})
	add("widgets.TokenCounter", state{"low", widgets.TokenCounter(100, 1000, th)}, state{"warn", widgets.TokenCounter(850, 1000, th)}, state{"over", widgets.TokenCounter(1200, 1000, th)})
	add("widgets.Pagination", state{"page 1", widgets.Pagination(1, 5, th)}, state{"page 2", widgets.Pagination(2, 5, th)}, state{"page 5", widgets.Pagination(5, 5, th)})
	add("widgets.PaginationDots", state{"page 1", widgets.PaginationDots(1, 5, th)}, state{"page 2", widgets.PaginationDots(2, 5, th)}, state{"page 5", widgets.PaginationDots(5, 5, th)})
	add("accordion.ToolCall", state{"pending", accordion.ToolCall("read", "f", accordion.ToolCallPending, "").Title}, state{"success", accordion.ToolCall("read", "f", accordion.ToolCallSuccess, "").Title}, state{"error", accordion.ToolCall("read", "f", accordion.ToolCallError, "").Title})

	// Animated bar: the segment position is its state.
	lb0 := loadingbar.New(12)
	lb0.Theme = th
	lb1 := lb0
	lb1.Start()
	lb1, _ = lb1.Update(loadingbarTick(lb1))
	add("loadingbar", state{"start", lb0.View()}, state{"moved", lb1.View()})

	// Confirm: focus on yes / no.
	cy, cn := confirm.New("Sure?"), confirm.New("Sure?")
	cy.Theme, cn.Theme = th, th
	cn, _ = cn.Update(tui.Key{Type: tui.KeyRight})
	add("confirm", state{"yes", cy.View()}, state{"no", cn.View()})

	// Multiselect: checked / unchecked, cursor on / off.
	ms := multiselect.NewStrings("a", "b")
	ms.Theme = th
	msChecked := ms
	msChecked.Toggle(0)
	msCursor := ms
	msCursor.SetCursor(1)
	add("multiselect", state{"unchecked", ms.View()}, state{"checked", msChecked.View()}, state{"cursor moved", msCursor.View()})

	// Toolapproval: risk levels, and which option is highlighted.
	var risks, choices []state
	for _, r := range []toolapproval.Risk{toolapproval.RiskLow, toolapproval.RiskMedium, toolapproval.RiskHigh} {
		m := toolapproval.New("run", "", r)
		m.Theme = th
		risks = append(risks, state{fmt.Sprint("risk ", r), m.View()})
	}
	m := toolapproval.New("run", "", toolapproval.RiskLow)
	m.Theme = th
	for i := 0; i < 3; i++ {
		choices = append(choices, state{fmt.Sprint("highlight ", i), m.View()})
		m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	}
	add("toolapproval risk", risks...)
	add("toolapproval choice", choices...)

	// Errorretry: retries remain / used / exhausted.
	er := errorretry.New("failed", 1)
	er.Theme = th
	erUsed, _ := er.Update(tui.Key{Type: tui.KeyEnter})
	add("errorretry", state{"fresh", er.View()}, state{"exhausted", erUsed.View()})

	// Form: untouched / validation error / valid; field focus.
	mkForm := func(val string, submit bool) form.Model {
		f := form.New(form.Field{Name: "n", Label: "Name", Value: val, Validators: []form.Validator{form.Required()}})
		f.Theme = th
		if submit {
			f, _ = f.Submit()
		}
		return f
	}
	add("form validation", state{"untouched", mkForm("", false).View()}, state{"invalid", mkForm("", true).View()}, state{"valid", mkForm("ok", true).View()})

	// Tabs, tree, accordion, datatable.
	t0 := tabs.New("One", "Two")
	t0.Theme = th
	t1 := t0
	t1.SetActive(1)
	add("tabs", state{"first active", t0.View()}, state{"second active", t1.View()})

	tv := treeview.New(treeview.Node{Label: "root", Children: []treeview.Node{{Label: "a"}}})
	tv.Theme = th
	tvOpen, _ := tv.Update(tui.Key{Type: tui.KeyRight})
	add("treeview", state{"collapsed", tv.View()}, state{"expanded", tvOpen.View()})

	ac := accordion.New(accordion.Section{Title: "A", Content: "one"}, accordion.Section{Title: "B", Content: "two"})
	ac.Theme = th
	acOpen := ac
	acOpen.Toggle(0)
	add("accordion", state{"closed", ac.View()}, state{"open", acOpen.View()})

	dt := datatable.New([]string{"Name", "State"}, [][]string{{"api", "up"}, {"db", "up"}})
	dt.Theme = th
	dt2 := dt
	dt2.SetCursor(1)
	add("datatable selected row", state{"row 0", dt.View()}, state{"row 1", dt2.View()})
	return ws
}

func loadingbarTick(m loadingbar.Model) tui.Msg {
	// The tick message is unexported; obtain it from the Cmd Start returns.
	c := m
	cmd := c.Start()
	if cmd == nil {
		return nil
	}
	return tui.RunCmd(context.Background(), cmd)
}

// With NO_COLOR=1 every status-bearing widget's states must render as
// pairwise distinct text once escape sequences are removed: colour (and
// bold/reverse) alone must never carry a state.
func TestColourIndependence(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	ws := colourIndependentWidgets(theme.DarkTheme())
	if len(ws) < 26 {
		t.Fatalf("only %d widget state sets enumerated", len(ws))
	}
	for _, w := range ws {
		seen := map[string]string{}
		for _, s := range w.states {
			plain := ansi.StripANSI(s.out)
			if plain == "" {
				t.Errorf("%s/%s renders nothing", w.name, s.name)
			}
			if prev, dup := seen[plain]; dup {
				t.Errorf("%s: states %q and %q render identically without colour:\n%s", w.name, prev, s.name, plain)
			}
			seen[plain] = s.name
		}
		if len(w.states) < 2 {
			t.Errorf("%s has fewer than two states", w.name)
		}
	}
}
