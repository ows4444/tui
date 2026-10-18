package main

import (
	"errors"
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/dialog"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/loadingbar"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/scrollbar"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/toast"
	"github.com/ows4444/tui/virtuallist"
	"github.com/ows4444/tui/wizard"
)

// The pages, in the order of the tab bar.
const (
	pageMenus = iota
	pageLoading
	pageFields
	pageWizard
	pageCount
)

var pageNames = [pageCount]string{"Menus", "Loading", "Fields", "Wizard"}

// field numbers the four fields of the Fields page, for Tab.
const fieldCount = 4

const listRows = 1000

type model struct {
	th       ui.Theming
	w, h     int
	page     int
	help     bool
	chose    string // the last menu item chosen
	menu     menu.Model
	dialog   dialog.Model
	drawer   drawer.Model
	popover  popover.Model
	toast    toast.Model
	bar      loadingbar.Model
	skel     skeleton.Model
	list     virtuallist.Model
	scroll   scrollbar.Model
	email    emailinput.Model
	secret   maskedinput.Model
	complete autocomplete.Model
	color    colorpicker.Model
	field    int // the focused field on the Fields page
	steps    wizard.Model
	stepErr  string
	started  bool
}

func newModel() model {
	m := model{
		menu: menu.New([]menu.Item{
			{Label: "Open", Value: "open"},
			{Label: "Recent", Value: "recent", Children: []menu.Item{{Label: "report.md", Value: "report"}, {Label: "notes.txt", Value: "notes"}}},
			{Label: "Quit", Value: "quit"},
		}),
		dialog:   dialog.New("Delete file", "This cannot be undone."),
		drawer:   drawer.New("Filters\n  [x] open\n  [ ] closed\n\nEsc closes"),
		popover:  popover.New("Anchored near a point", 6, 4),
		toast:    toast.New("Saved"),
		bar:      loadingbar.New(24),
		skel:     skeleton.New(),
		list:     virtuallist.New(listRows, 5, func(i int) string { return fmt.Sprintf("row %04d", i) }),
		scroll:   scrollbar.New(listRows, 5),
		email:    emailinput.New(),
		secret:   maskedinput.New(),
		complete: autocomplete.New("alpha", "alps", "beta", "gamma"),
		color: colorpicker.New(ansi.RGB{R: 0xe0, G: 0x40, B: 0x40}, ansi.RGB{R: 0x40, G: 0xa0, B: 0x40},
			ansi.RGB{R: 0x40, G: 0x60, B: 0xe0}),
		steps: wizard.New("Account", "Profile", "Done"),
	}
	m.skel.Width, m.skel.Lines = 28, 3
	m.dialog.Width = 30
	m.dialog.Hide() // the overlays start open
	m.drawer.Hide()
	m.popover.Hide()
	return m
}

func (m model) Init() tui.Cmd { return nil }

// start runs the animations the first time the model sees its size.
func (m model) start() (model, tui.Cmd) {
	m.started = true
	return m, tui.Batch(m.bar.Start(), m.skel.Start(), m.focusField())
}

// focusField gives the keys to the field Tab has reached, and takes them from
// the others. It returns the Cmd focusing may need.
func (m *model) focusField() tui.Cmd {
	m.email.Blur()
	m.secret.Blur()
	m.complete.Blur()
	if m.page != pageFields {
		return nil
	}
	switch m.field {
	case 0:
		return m.email.Focus()
	case 1:
		return m.secret.Focus()
	case 2:
		return m.complete.Focus()
	}
	return nil
}

// overlay returns the open overlay's name, or "".
func (m model) overlay() string {
	switch {
	case m.dialog.Open():
		return "dialog"
	case m.drawer.Open():
		return "drawer"
	case m.popover.Open():
		return "popover"
	}
	return ""
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.list.Height = max(1, m.bodyRows()-2)
		m.scroll.Visible, m.scroll.Length = m.list.Height, m.list.Height
		if !m.started {
			return m.start()
		}
		return m, nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	case menu.SelectedMsg:
		m.chose = msg.Item.Label
		if msg.Item.Value == "quit" {
			return m, tui.Quit()
		}
		return m, nil
	case dialog.DismissedMsg, drawer.DismissedMsg, popover.DismissedMsg, toast.DismissedMsg:
		return m.dismissed(), nil
	}
	// Ticks of the animations, and the overlays' own messages.
	m.bar, cmd = m.bar.Update(msg)
	var c2, c3 tui.Cmd
	m.skel, c2 = m.skel.Update(msg)
	m.toast, c3 = m.toast.Update(msg)
	m.drawer, _ = m.drawer.Update(msg)
	return m, tui.Batch(cmd, c2, c3)
}

func (m model) dismissed() model {
	m.dialog.Hide()
	m.drawer.Hide()
	m.popover.Hide()
	m.toast.Hide()
	return m
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case m.help:
		m.help = false
		return m, nil
	}
	// An open overlay keeps the keys until it is dismissed.
	switch m.overlay() {
	case "dialog":
		m.dialog, cmd = m.dialog.Update(k)
		return m, cmd
	case "drawer":
		m.drawer, cmd = m.drawer.Update(k)
		return m, cmd
	case "popover":
		m.popover, cmd = m.popover.Update(k)
		return m, cmd
	}
	switch {
	case k.Type >= tui.KeyF1 && k.Type <= tui.KeyF4:
		return m.goTo(int(k.Type - tui.KeyF1))
	case k.Type == tui.KeyCtrl && k.Code == 'n':
		return m.goTo((m.page + 1) % pageCount)
	case k.Type == tui.KeyCtrl && k.Code == 'p':
		return m.goTo((m.page + pageCount - 1) % pageCount)
	}
	switch m.page {
	case pageMenus:
		return m.menusKey(k)
	case pageLoading:
		return m.loadingKey(k)
	case pageFields:
		return m.fieldsKey(k)
	}
	return m.wizardKey(k)
}

func (m model) goTo(page int) (tui.Model, tui.Cmd) {
	m.page = page
	return m, m.focusField()
}

func (m model) menusKey(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if k.Type == tui.KeyRunes {
		switch k.Text {
		case "d":
			m.dialog.Show()
			return m, nil
		case "w":
			m.drawer.Show()
			return m, nil
		case "p":
			m.popover.Show()
			return m, nil
		case "t":
			return m, m.toast.Show()
		case "?":
			m.help = true
			return m, nil
		case "q":
			return m, tui.Quit()
		}
	}
	m.menu, cmd = m.menu.Update(k)
	return m, cmd
}

func (m model) loadingKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && k.Text == "?":
		m.help = true
		return m, nil
	case k.Type == tui.KeyRunes && k.Text == "q":
		return m, tui.Quit()
	}
	m.list, _ = m.list.Update(k)
	m.scroll.SetOffset(m.list.Offset())
	return m, nil
}

func (m model) fieldsKey(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if k.Type == tui.KeyTab {
		m.field = (m.field + 1) % fieldCount
		return m, m.focusField()
	}
	switch m.field {
	case 0:
		m.email, cmd = m.email.Update(k)
	case 1:
		m.secret, cmd = m.secret.Update(k)
	case 2:
		m.complete, cmd = m.complete.Update(k)
	default:
		m.color, cmd = m.color.Update(k)
	}
	return m, cmd
}

func (m model) wizardKey(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type != tui.KeyRunes {
		return m, nil
	}
	switch k.Text {
	case "n":
		m.stepErr = ""
		err := m.steps.Next(func() error {
			if m.steps.Current() == 0 && !m.email.Valid() {
				return errors.New("give a valid email on the Fields page first")
			}
			return nil
		})
		if err != nil {
			m.stepErr = err.Error()
		}
	case "b":
		m.stepErr = ""
		m.steps.Back()
	case "?":
		m.help = true
	case "q":
		return m, tui.Quit()
	}
	return m, nil
}

// tabAt is the page whose name is at column x of the tab bar, or -1.
func tabAt(x int) int {
	col := 1
	for i, name := range pageNames {
		w := len(name) + 2
		if x >= col && x < col+w {
			return i
		}
		col += w + 1
	}
	return -1
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress || m.overlay() != "" {
		return m, nil
	}
	if ev.Y == 1 {
		if p := tabAt(ev.X); p >= 0 {
			return m.goTo(p)
		}
	}
	return m, nil
}

// bodyRows is the rows between the tab bar and the hint line.
func (m model) bodyRows() int { return max(1, m.h-4) }
