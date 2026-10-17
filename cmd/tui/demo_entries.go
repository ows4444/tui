package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"

	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/buttongroup"
	"github.com/ows4444/tui/checkbox"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/dropdown"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdownview"
	"github.com/ows4444/tui/multiselect"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/otpinput"
	"github.com/ows4444/tui/pagination"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/radiogroup"
	"github.com/ows4444/tui/rating"
	"github.com/ows4444/tui/slider"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/toggle"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

// demoEntries is the catalog, in the order the list shows it.
// demoEntries is the catalog, in the order the list shows it.
func demoEntries() []entry {
	var all []entry
	all = append(all, actionEntries()...)
	all = append(all, choiceEntries()...)
	all = append(all, inputEntries()...)
	all = append(all, navigationEntries()...)
	all = append(all, dataEntries()...)
	all = append(all, feedbackEntries()...)
	all = append(all, layoutEntries()...)
	return all
}

// actionEntries are the action widgets of the catalog.
func actionEntries() []entry {
	return []entry{
		{
			name: "Button", group: "action", summary: "A pressable control; a label like &Save gets an Alt+S hot key.",
			code:  []string{`b := button.New("&Save")`, `b.Focus()`, `// Enter, Space or Alt+S press it:`, `// the Cmd it returns delivers button.PressedMsg.`},
			build: func() demoWidget { return newLive(button.New("&Save")) },
		},
		{
			name: "Button group", group: "action", summary: "A row of buttons that take one place in the focus order; here a single choice.",
			code: []string{`g := buttongroup.New(buttongroup.ModeSingle,`, `	"&Small", "&Medium", "&Large")`, `// Left and Right move, Enter chooses.`},
			build: func() demoWidget {
				return newLive(buttongroup.New(buttongroup.ModeSingle, "&Small", "&Medium", "&Large"))
			},
		},
	}
}

// choiceEntries are the choice widgets of the catalog.
func choiceEntries() []entry {
	return []entry{
		{
			name: "Checkbox", group: "choice", summary: "A box that is checked or not; Space flips it.",
			code:  []string{`c := checkbox.New("Remember me")`, `c.Focus()`},
			build: func() demoWidget { return newLive(checkbox.New("Remember me")) },
		},
		{
			name: "Toggle", group: "choice", summary: "An on/off switch; Space or Enter flips it, Left and Right set it.",
			code:  []string{`t := toggle.New("Notifications")`, `t.Focus()`},
			build: func() demoWidget { return newLive(toggle.New("Notifications")) },
		},
		{
			name: "Radio group", group: "choice", summary: "Exactly one of several options.",
			code:  []string{`r := radiogroup.New("Email", "SMS", "Push")`},
			build: func() demoWidget { return newLive(radiogroup.New("Email", "SMS", "Push")) },
		},
		{
			name: "Dropdown", group: "choice", summary: "A closed control that opens a list; type letters to jump.",
			code:  []string{`d := dropdown.New("Plan", "Free", "Pro", "Team")`, `// Enter opens, arrows move, Enter chooses.`},
			build: func() demoWidget { return newLive(dropdown.New("Plan", "Free", "Pro", "Team", "Enterprise")) },
		},
		{
			name: "Picker", group: "choice", summary: "A single-choice list; TypeAhead jumps to the next label that starts with what you type.",
			code: []string{`p := picker.NewStrings("Apple", "Banana", ...)`, `p.TypeAhead = true`},
			build: func() demoWidget {
				p := picker.NewStrings("Apple", "Apricot", "Banana", "Blueberry", "Cherry", "Cranberry", "Grape", "Lemon")
				p.TypeAhead = true
				return newLive(p)
			},
		},
		{
			name: "Multi-select", group: "choice", summary: "Any number of options; Space marks one, Enter confirms.",
			code:  []string{`m := multiselect.NewStrings("Go", "Rust", "Zig")`},
			build: func() demoWidget { return newLive(multiselect.NewStrings("Go", "Rust", "Zig", "Python", "C")) },
		},
	}
}

// inputEntries are the input widgets of the catalog.
func inputEntries() []entry {
	return []entry{
		{
			name: "Text input", group: "input", summary: "One line of text with a cursor, undo, history and a kill ring.",
			code: []string{`in := textinput.New()`, `in.Prompt, in.Placeholder = "Name: ", "Ada"`, `in.KillRing = true // Ctrl+K, Ctrl+W, Ctrl+Y`},
			build: func() demoWidget {
				in := textinput.New()
				in.Prompt, in.Placeholder, in.KillRing = "Name: ", "Ada Lovelace", true
				in.History = []string{"Grace Hopper", "Alan Turing"}
				return newLive(in)
			},
		},
		{
			name: "Text area", group: "input", summary: "Several lines, with editor aids: cursor line, matching brackets, a ruler.",
			code: []string{`ta := textarea.New()`, `ta.ShowLineNumbers = true`, `ta.HighlightCursorLine, ta.MatchBrackets = true, true`, `ta.Ruler = []int{40}`},
			build: func() demoWidget {
				ta := textarea.New()
				ta.Width, ta.Height = 44, 6
				ta.ShowLineNumbers, ta.HighlightCursorLine, ta.MatchBrackets = true, true, true
				ta.Ruler = []int{40}
				ta.SetValue("func add(a, b int) int {\n    return (a + b)\n}\n")
				return newLive(ta)
			},
		},
		{
			name: "Number input", group: "input", summary: "A number that arrow keys step.",
			code: []string{`n := numberinput.New()`, `n.Prompt, n.Placeholder = "Quantity: ", "digits only"`},
			build: func() demoWidget {
				n := numberinput.New()
				n.Prompt, n.Placeholder = "Quantity: ", "digits only"
				return newLive(n)
			},
		},
		{
			name: "Tag input", group: "input", summary: "Type a word and press Enter to make it a tag.",
			code: []string{`t := taginput.New()`, `t.Input.Prompt = "Tags: "`},
			build: func() demoWidget {
				t := taginput.New()
				t.Input.Prompt, t.Input.Placeholder = "Tags: ", "type, Enter"
				return newLive(t)
			},
		},
		{
			name: "One-time code", group: "input", summary: "A row of boxes for a verification code.",
			code:  []string{`o := otpinput.New(6)`},
			build: func() demoWidget { return newLive(otpinput.New(6)) },
		},
		{
			name: "Date picker", group: "input", summary: "Pick a date with the arrow keys.",
			code: []string{`d := datepicker.New(time.Now())`},
			build: func() demoWidget {
				return newLive(datepicker.New(time.Date(2026, time.October, 18, 0, 0, 0, 0, time.UTC)))
			},
		},
		{
			name: "Slider", group: "input", summary: "A value between two ends; Left and Right move it.",
			code:  []string{`s := slider.New(0, 100)`},
			build: func() demoWidget { return newLive(slider.New(0, 100)) },
		},
		{
			name: "Rating", group: "input", summary: "Stars from one to five.",
			code:  []string{`r := rating.New(5)`},
			build: func() demoWidget { return newLive(rating.New(5)) },
		},
	}
}

// navigationEntries are the navigation widgets of the catalog.
func navigationEntries() []entry {
	return []entry{
		{
			name: "Tabs", group: "navigation", summary: "A tab bar; Left and Right move, Alt+letter jumps to a tab with an &.",
			code:  []string{`t := tabs.New("&Files", "&Edit", "&Help")`},
			build: func() demoWidget { return newLive(tabs.New("&Files", "&Edit", "&Help")) },
		},
		{
			name: "Pagination", group: "navigation", summary: "Page numbers with the current one marked.",
			code:  []string{`p := pagination.New(12)`},
			build: func() demoWidget { return newLive(pagination.New(12)) },
		},
		{
			name: "Accordion", group: "navigation", summary: "Sections that open and close.",
			code: []string{`a := accordion.New(`, `	accordion.Section{Title: "General", Content: "..."})`},
			build: func() demoWidget {
				return newLive(accordion.New(
					accordion.Section{Title: "General", Content: "Language, theme and start-up."},
					accordion.Section{Title: "Editor", Content: "Tabs, wrapping and rulers."},
					accordion.Section{Title: "Keys", Content: "Every binding can be changed."}))
			},
		},
	}
}

// dataEntries are the data widgets of the catalog.
func dataEntries() []entry {
	return []entry{
		{
			name: "Data table", group: "data", summary: "Rows with a cursor; / filters, s sorts, and TypeAhead jumps by the first column.",
			code: []string{`t := datatable.New(headers, rows)`, `t.Sort, t.Filter, t.TypeAhead = true, true, true`},
			build: func() demoWidget {
				t := datatable.New([]string{"Name", "Role", "City"}, [][]string{
					{"Ada", "Engineer", "London"}, {"Grace", "Admiral", "New York"}, {"Alan", "Cryptanalyst", "Manchester"},
					{"Linus", "Maintainer", "Portland"}, {"Margaret", "Director", "Cambridge"}, {"Dennis", "Engineer", "Murray Hill"},
				})
				t.Height, t.Filter, t.TypeAhead = 6, true, true
				return newLive(t)
			},
		},
		{
			name: "Tree view", group: "data", summary: "A tree you expand with Right and collapse with Left.",
			code: []string{`t := treeview.New(treeview.Node{Label: "src"}, ...)`, `t.TypeAhead = true`},
			build: func() demoWidget {
				t := treeview.New(
					treeview.Node{Label: "cmd", Children: []treeview.Node{{Label: "tui"}}},
					treeview.Node{Label: "layout", Children: []treeview.Node{{Label: "box.go"}, {Label: "merge.go"}}},
					treeview.Node{Label: "widgets", Children: []treeview.Node{{Label: "table.go"}, {Label: "tree.go"}}},
					treeview.Node{Label: "README.md"})
				t.TypeAhead = true
				return newLive(t)
			},
		},
		{
			name: "Line chart", group: "data", summary: "Braille dots, or chunkier glyphs for fonts without them; Left and Right change the marker.",
			code: []string{`chart.Plot(series, chart.PlotOptions{`, `	Width: 44, Height: 7, Marker: chart.Quadrant,`, `}, theme)`},
			build: func() demoWidget {
				markers := []chart.Marker{chart.Braille, chart.Sextant, chart.Quadrant, chart.HalfBlock, chart.Block}
				data := []chart.Series{{Name: "load", Values: []float64{3, 5, 4, 8, 6, 9, 7, 12, 10, 14, 11, 9, 13, 16, 12}}}
				return custom{press: stepWith(0, len(markers)-1, 1), draw: func(th theme.Theme, i int) string {
					return chart.Plot(data, chart.PlotOptions{Width: 44, Height: 7, Marker: markers[i]}, th) + "\nmarker: " + markers[i].String()
				}}
			},
		},
		{
			name: "Markdown", group: "data", summary: "Rendered Markdown in a scrollable view.",
			code: []string{`m := markdownview.New(md, 56, 8)`},
			build: func() demoWidget {
				return newLive(markdownview.New("# Release notes\n\n- **Faster** scrolling\n- A `dropdown` widget\n- Tasks:\n  - [x] merged borders\n  - [ ] sleep\n\nSee [the docs](https://example.com).", 56, 8))
			},
		},
	}
}

// feedbackEntries are the feedback widgets of the catalog.
func feedbackEntries() []entry {
	return []entry{
		{
			name: "Progress bar", group: "feedback", summary: "A bar for a percentage; Left and Right change it.",
			code: []string{`widgets.ProgressBar(0.42, 40, theme)`, `// stateless: you pass the number each frame`},
			build: func() demoWidget {
				return custom{state: 42, press: stepWith(0, 100, 5), draw: func(th theme.Theme, pct int) string {
					return widgets.ProgressBar(float64(pct)/100, 40, th) + fmt.Sprintf("  %d%%", pct)
				}}
			},
		},
		{
			name: "Spinner", group: "feedback", summary: "An animated indicator; 18 named animations to choose from.",
			code: []string{`p, _ := spinner.Lookup("dots2")`, `s := spinner.New().WithPreset(p)`, `cmd := s.Start() // return it from Init`},
			build: func() demoWidget {
				p, _ := spinner.Lookup("dots2")
				s := spinner.New().WithPreset(p)
				s.Label = "Loading"
				l := newLive(s)
				l.start = func(m *spinner.Model) tui.Cmd { return m.Start() }
				return l
			},
		},
	}
}

// layoutEntries are the layout widgets of the catalog.
func layoutEntries() []entry {
	return []entry{
		{
			name: "Merged boxes", group: "layout", summary: "Bordered boxes that share their borders: MergedRow and MergedColumn.",
			code: []string{`layout.MergedRow(`, `	layout.Fill(box("one")),`, `	layout.Fill(layout.MergedColumn(`, `		layout.Fill(box("two")), layout.Fill(box("three")))))`},
			build: func() demoWidget {
				return custom{draw: func(th theme.Theme, _ int) string {
					box := func(s string) layout.Node {
						return layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).BorderColor(th.BorderColor), layout.Block(s))
					}
					return layout.MergedRow(layout.Fill(box("one")), layout.Fill(layout.MergedColumn(layout.Fill(box("two")), layout.Fill(box("three"))))).Render(layout.Size{W: 31, H: 7})
				}}
			},
		},
		{
			name: "Auto grid", group: "layout", summary: "As many equal columns as fit; Left and Right change the width.",
			code: []string{`layout.AutoGrid(12, 1, cards...)`, `// 12 wide at least, one gap between`},
			build: func() demoWidget {
				return custom{state: 40, press: stepWith(14, 56, 6), draw: func(th theme.Theme, w int) string {
					var cards []layout.Node
					for i := 1; i <= 6; i++ {
						cards = append(cards, layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()), layout.Block(fmt.Sprintf("card %d", i))))
					}
					grid := layout.AutoGrid(12, 1, cards...)
					size := grid.Measure(layout.Constraints{MaxW: w, MaxH: layout.Unbounded})
					return strings.TrimRight(grid.Render(size), "\n") + fmt.Sprintf("\nwidth %d", w)
				}}
			},
		},
	}
}
