package theme_test

import (
	"testing"
	"time"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/appshell"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/commandpalette"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/contextmenu"
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
	"github.com/ows4444/tui/imageview"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/menubar"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/notificationcenter"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/radiogroup"
	"github.com/ows4444/tui/rating"
	"github.com/ows4444/tui/scrollbar"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/slider"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/splitpane"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/toolapproval"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/wizard"
)

// allRoles overrides every token with a colour no preset uses.
func allRoles() theme.Tokens {
	c := func(n uint8) ansi.Color { return ansi.RGB{R: 250, G: n, B: 7} }
	return theme.Tokens{
		Text: c(1), Muted: c(2), Accent: c(3), Focus: c(4), Selection: c(5),
		Border: c(6), Background: c(7), Surface: c(8), Success: c(9),
		Warning: c(10), Error: c(11), Info: c(12), Overlay: c(13), TextInverse: c(14),
	}
}

// tokenCase renders one widget. build gets the theme to apply (through the
// widget's SetTheme) and an optional per-instance override (WithTokens).
type tokenCase struct {
	component string
	build     func(th theme.Theme, inst *theme.Tokens) string
}

func tokenCases() []tokenCase {
	base := "line one\nline two\nline three"
	return []tokenCase{
		{theme.ComponentAccordion, func(th theme.Theme, i *theme.Tokens) string {
			m := accordion.New(accordion.Section{Title: "A", Content: "x"}, accordion.Section{Title: "B", Content: "y"}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentAppShell, func(th theme.Theme, i *theme.Tokens) string {
			m := appshell.New("Title", 30, 3).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentAutocomplete, func(th theme.Theme, i *theme.Tokens) string {
			m := autocomplete.New("alpha", "beta").SetTheme(th)
			m.Input.SetValue("a")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentColorPicker, func(th theme.Theme, i *theme.Tokens) string {
			m := colorpicker.New(ansi.Red, ansi.Green).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentCommandPalette, func(th theme.Theme, i *theme.Tokens) string {
			m := commandpalette.New(commandpalette.Command{Name: "open", Description: "d"}).SetTheme(th)
			m.Input.SetValue("o")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentButton, func(th theme.Theme, i *theme.Tokens) string {
			m := button.New("Save").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentConfirm, func(th theme.Theme, i *theme.Tokens) string {
			m := confirm.New("Sure?").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentContextMenu, func(th theme.Theme, i *theme.Tokens) string {
			m := contextmenu.New(contextmenu.Item{Label: "Copy"}, contextmenu.Item{Label: "Paste"}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show(0, 0)
			return m.Render(base)
		}},
		{theme.ComponentDataTable, func(th theme.Theme, i *theme.Tokens) string {
			m := datatable.New([]string{"a", "b"}, [][]string{{"1", "2"}, {"3", "4"}}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentDatePicker, func(th theme.Theme, i *theme.Tokens) string {
			m := datepicker.New(time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentDialog, func(th theme.Theme, i *theme.Tokens) string {
			m := dialog.New("Title", "Message").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show()
			return m.Render(base)
		}},
		{theme.ComponentDrawer, func(th theme.Theme, i *theme.Tokens) string {
			m := drawer.New("content").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show()
			return m.Render(base)
		}},
		{theme.ComponentErrorRetry, func(th theme.Theme, i *theme.Tokens) string {
			m := errorretry.New("boom", 3).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentFaces, func(th theme.Theme, i *theme.Tokens) string {
			m := faces.New().SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentFilePicker, func(th theme.Theme, i *theme.Tokens) string {
			m := filepicker.New(".").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentForm, func(th theme.Theme, i *theme.Tokens) string {
			m := form.New(form.Field{Name: "a", Label: "A", Value: "hi"}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentHelpScreen, func(th theme.Theme, i *theme.Tokens) string {
			m := helpscreen.New(widgets.Hint{Key: "q", Action: "quit"}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show()
			return m.Render(base)
		}},
		{theme.ComponentImageView, func(th theme.Theme, i *theme.Tokens) string {
			m := imageview.New(nil, 10, 2, "alt").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentLoadingBar, func(th theme.Theme, i *theme.Tokens) string {
			m := loadingbar.New(20).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentMarkdown, func(th theme.Theme, i *theme.Tokens) string {
			m := markdown.New("# Title\n\nSome *text* with `code`.\n\n- item")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View(30, th)
		}},
		{theme.ComponentMaskedInput, func(th theme.Theme, i *theme.Tokens) string {
			m := maskedinput.New().SetTheme(th)
			m.SetValue("abc")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentMenu, func(th theme.Theme, i *theme.Tokens) string {
			m := menu.New([]menu.Item{{Label: "One"}, {Label: "Two"}}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentMenuBar, func(th theme.Theme, i *theme.Tokens) string {
			m := menubar.New(menubar.Menu{Title: "File", Items: []menubar.Item{{Label: "Open"}}}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show(0)
			return m.Render(base)
		}},
		{theme.ComponentMultiSelect, func(th theme.Theme, i *theme.Tokens) string {
			m := multiselect.NewStrings("a", "b").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentNotificationCenter, func(th theme.Theme, i *theme.Tokens) string {
			m := notificationcenter.New(3, 30).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Push(notificationcenter.Notification{Message: "hi", Variant: widgets.VariantSuccess})
			return m.Render(base)
		}},
		{theme.ComponentPasswordInput, func(th theme.Theme, i *theme.Tokens) string {
			m := passwordinput.New().SetTheme(th)
			m.SetValue("abc")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentPicker, func(th theme.Theme, i *theme.Tokens) string {
			m := picker.NewStrings("a", "b").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentPopover, func(th theme.Theme, i *theme.Tokens) string {
			m := popover.New("tip", 2, 1).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show()
			return m.Render(base)
		}},
		{theme.ComponentRadioGroup, func(th theme.Theme, i *theme.Tokens) string {
			m := radiogroup.New("A", "B").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentRating, func(th theme.Theme, i *theme.Tokens) string {
			m := rating.New(5).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentScrollbar, func(th theme.Theme, i *theme.Tokens) string {
			m := scrollbar.New(100, 10).SetTheme(th)
			m.Length = 10
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentSkeleton, func(th theme.Theme, i *theme.Tokens) string {
			m := skeleton.New().SetTheme(th)
			m.Width, m.Lines = 10, 2
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentSlider, func(th theme.Theme, i *theme.Tokens) string {
			m := slider.New(0, 10).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentSpinner, func(th theme.Theme, i *theme.Tokens) string {
			m := spinner.New().SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentSplitPane, func(th theme.Theme, i *theme.Tokens) string {
			m := splitpane.New(layout.Text("left"), layout.Text("right")).SetTheme(th)
			m.Total = 20
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentStreamText, func(th theme.Theme, i *theme.Tokens) string {
			m := streamtext.New().SetTheme(th)
			m.SetText("hello world")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentTabs, func(th theme.Theme, i *theme.Tokens) string {
			m := tabs.New("one", "two").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentTagInput, func(th theme.Theme, i *theme.Tokens) string {
			m := taginput.New().SetTheme(th)
			m.Tags = []string{"go", "tui"}
			m.Input.SetValue("x")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentTextArea, func(th theme.Theme, i *theme.Tokens) string {
			m := textarea.New().SetTheme(th)
			m.SetValue("hello")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentTextInput, func(th theme.Theme, i *theme.Tokens) string {
			m := textinput.New().SetTheme(th)
			m.SetValue("hello")
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentToast, func(th theme.Theme, i *theme.Tokens) string {
			m := toast.New("saved").SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			m.Show()
			return m.Render(base)
		}},
		{theme.ComponentToolApproval, func(th theme.Theme, i *theme.Tokens) string {
			m := toolapproval.New("rm", "remove", toolapproval.RiskHigh).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentTreeView, func(th theme.Theme, i *theme.Tokens) string {
			m := treeview.New(treeview.Node{Label: "root", Children: []treeview.Node{{Label: "kid"}}}).SetTheme(th)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View()
		}},
		{theme.ComponentWizard, func(th theme.Theme, i *theme.Tokens) string {
			m := wizard.New("One", "Two", "Three")
			_ = m.Next(nil)
			if i != nil {
				m = m.WithTokens(*i)
			}
			return m.View(th)
		}},
	}
}

// tokenExclusions are widget packages that cannot show a colour override, with
// the reason; they have no component constant. The test below enumerates them
// so "every themed widget" is exact. Stateless functions in package widgets
// and markdown.Render take a theme argument, so callers pass
// th.ForComponent(name); markdown.Model and wizard.Model resolve their own
// component when they draw.
var tokenExclusions = map[string]string{
	"clipboard":   "renders plain text; its Theme field is not read when drawing",
	"virtuallist": "draws only the caller's renderItem strings; its Theme field is not read when drawing",
}

func TestEveryThemedWidgetOverridesColoursThroughTokens(t *testing.T) {
	dark := theme.DarkTheme()
	tok := allRoles()
	cases := tokenCases()
	other := func(c string) string {
		if c == theme.ComponentToast {
			return theme.ComponentTabs
		}
		return theme.ComponentToast
	}

	for _, c := range cases {
		t.Run(c.component, func(t *testing.T) {
			base := c.build(dark, nil)
			if again := c.build(dark, nil); again != base {
				t.Fatal("render is not deterministic, comparison is meaningless")
			}
			if c.build(dark, &theme.Tokens{}) != base {
				t.Error("an empty Tokens changed the output")
			}
			if got := c.build(dark, &tok); got == base {
				t.Error("a per-instance override did not change the output")
			}
			themeWide := c.build(dark.WithTokens(c.component, tok), nil)
			if themeWide == base {
				t.Error("a theme-level override for the component did not change the output")
			}
			if themeWide != c.build(dark, &tok) {
				t.Error("theme-level and per-instance overrides render differently")
			}
			if got := c.build(dark.WithTokens(other(c.component), tok), nil); got != base {
				t.Errorf("an override for %q leaked into %q", other(c.component), c.component)
			}
			if dark.Components != nil {
				t.Error("the base theme was modified")
			}
		})
	}

	t.Run("coverage", func(t *testing.T) {
		covered := map[string]bool{}
		for _, c := range cases {
			covered[c.component] = true
		}
		// emailinput and numberinput share the textinput component.
		if len(tokenExclusions) != 2 {
			t.Errorf("exclusions changed: %v", tokenExclusions)
		}
		for _, name := range theme.Components() {
			if !covered[name] {
				t.Errorf("component %q is neither tested nor listed in tokenExclusions", name)
			}
		}
	})
}

// emailinput and numberinput wrap textinput and share its component.
func TestTextInputWrappersTakeTokens(t *testing.T) {
	dark := theme.DarkTheme()
	tok := theme.Tokens{Text: ansi.RGB{R: 250, G: 1, B: 7}, Muted: ansi.RGB{R: 250, G: 2, B: 7}}
	e := emailinput.New().SetTheme(dark)
	e.SetValue("a@b")
	n := numberinput.New().SetTheme(dark)
	n.SetValue("12")
	if e.WithTokens(tok).View() == e.View() {
		t.Error("emailinput ignored tokens")
	}
	if n.WithTokens(tok).View() == n.View() {
		t.Error("numberinput ignored tokens")
	}
	if got := e.SetTheme(dark.WithTokens(theme.ComponentTextInput, tok)).View(); got != e.WithTokens(tok).View() {
		t.Error("theme-level textinput tokens did not reach emailinput")
	}
}

// Semantic roles must be overridable and must reach widgets that read them.
func TestTokensCoverSemanticRoles(t *testing.T) {
	dark := theme.DarkTheme()
	red := ansi.RGB{R: 9, G: 9, B: 9}
	roles := []struct {
		name string
		tok  theme.Tokens
		get  func(theme.Tokens) ansi.Color
		role func(theme.Theme) ansi.Color
	}{
		{"Success", theme.Tokens{Success: red}, func(k theme.Tokens) ansi.Color { return k.Success }, func(h theme.Theme) ansi.Color { return h.Success }},
		{"Warning", theme.Tokens{Warning: red}, func(k theme.Tokens) ansi.Color { return k.Warning }, func(h theme.Theme) ansi.Color { return h.Warning }},
		{"Error", theme.Tokens{Error: red}, func(k theme.Tokens) ansi.Color { return k.Error }, func(h theme.Theme) ansi.Color { return h.Error }},
		{"Info", theme.Tokens{Info: red}, func(k theme.Tokens) ansi.Color { return k.Info }, func(h theme.Theme) ansi.Color { return h.Info }},
		{"Overlay", theme.Tokens{Overlay: red}, func(k theme.Tokens) ansi.Color { return k.Overlay }, func(h theme.Theme) ansi.Color { return h.Overlay }},
		{"TextInverse", theme.Tokens{TextInverse: red}, func(k theme.Tokens) ansi.Color { return k.TextInverse }, func(h theme.Theme) ansi.Color { return h.TextInverse }},
	}
	for _, r := range roles {
		th := dark.WithTokens(theme.ComponentToast, r.tok)
		if r.get(th.TokensFor(theme.ComponentToast)) != red {
			t.Errorf("%s: TokensFor lost the override", r.name)
		}
		if r.role(th.ForComponent(theme.ComponentToast)) != red {
			t.Errorf("%s: ForComponent ignored the override", r.name)
		}
		if r.role(th.Resolve(theme.ComponentToast, theme.Tokens{})) != red {
			t.Errorf("%s: Resolve ignored the override", r.name)
		}
		if r.get(th.TokensFor(theme.ComponentTabs)) != r.role(dark) {
			t.Errorf("%s: leaked to another component", r.name)
		}
		if r.role(dark) == red {
			t.Errorf("%s: the base theme was modified", r.name)
		}
	}
	// Error reaches a widget that draws it, Success another.
	e := errorretry.New("boom", 3).SetTheme(dark)
	if e.WithTokens(theme.Tokens{Error: red}).View() == e.View() {
		t.Error("Error token did not reach errorretry")
	}
	m := multiselect.NewStrings("a").SetTheme(dark)
	m.Toggle(0)
	if m.WithTokens(theme.Tokens{Success: red}).View() == m.View() {
		t.Error("Success token did not reach multiselect")
	}
}
