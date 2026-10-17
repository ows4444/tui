package appshell_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/appshell"
)

// An appshell frames content with a title, a scrolling area and an input line.
func Example() {
	m := appshell.New("my app", 40, 5)
	fmt.Println(strings.Contains(ansi.StripANSI(m.View()), "my app"))
	// Output:
	// true
}

type yesNo struct{}

func (yesNo) Update(msg tui.Msg) (yesNo, tui.Cmd) {
	if k, ok := msg.(tui.Key); ok && k.Text == "y" {
		return yesNo{}, appshell.PopCmd("yes") // hand "yes" to the screen below
	}
	return yesNo{}, nil
}
func (yesNo) View() string { return "Delete? (y/n)" }

type page struct{ answer string }

func (p page) Update(msg tui.Msg) (page, tui.Cmd) {
	switch m := msg.(type) {
	case tui.Key:
		if m.Text == "d" {
			return p, appshell.PushCmd(appshell.Wrap(yesNo{}))
		}
	case string:
		p.answer = m
	}
	return p, nil
}
func (p page) View() string { return "page, answer: " + p.answer }

// A screen pushes another over itself and the pushed one pops with an answer
// that comes back to the screen below.
func ExampleStack() {
	s := appshell.NewStack(appshell.Wrap(page{}))
	run := func(msg tui.Msg) {
		var cmd tui.Cmd
		s, cmd = s.Update(msg)
		for cmd != nil { // the program does this with each Cmd
			s, cmd = s.Update(cmd())
		}
	}
	run(tui.Key{Type: tui.KeyRunes, Text: "d"})
	fmt.Println(s.View())
	run(tui.Key{Type: tui.KeyRunes, Text: "y"})
	fmt.Println(s.View())
	// Output:
	// Delete? (y/n)
	// page, answer: yes
}
