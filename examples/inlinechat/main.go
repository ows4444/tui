// Command inlinechat is an inline-mode (no alternate screen) chat prompt.
// Each sent message and its canned reply are committed to real scrollback
// with tui.Println; only the one-line prompt is the live region.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

type replyMsg struct{ text string }

type model struct {
	input   string
	waiting bool
}

func (m model) Init() tui.Cmd { return nil }

func reply(to string) tui.Cmd {
	return tui.FromCtx(motion.After(300*time.Millisecond, func(time.Time) tui.Msg {
		return replyMsg{text: "bot: you said " + fmt.Sprintf("%q", to)}
	}))
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc:
			return m, tui.Quit()
		case msg.Type == tui.KeyEnter:
			text := strings.TrimSpace(m.input)
			if text == "" || m.waiting {
				return m, nil
			}
			m.input, m.waiting = "", true
			return m, tui.Batch(tui.Println("you: "+text), reply(text))
		case msg.Type == tui.KeyBackspace:
			if r := []rune(m.input); len(r) > 0 {
				m.input = string(r[:len(r)-1])
			}
		case msg.Type == tui.KeyRunes:
			m.input += msg.Text
		}
	case replyMsg:
		m.waiting = false
		return m, tui.Println(msg.text)
	}
	return m, nil
}

func (m model) View() string {
	if m.waiting {
		return "bot is typing..."
	}
	return "> " + m.input + "_"
}

func main() {
	if _, err := tui.NewProgram(model{}, tui.WithAltScreen(false)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
