package main

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/avatar"
	"github.com/ows4444/tui/imageview"
	"github.com/ows4444/tui/theme"
)

// avatarNames are the names the Avatar entry cycles through.
var avatarNames = []string{"ada", "grace", "linus", "ken", "margaret", "dennis", "barbara", "alan"}

// placeable is a demo widget that takes the mouse: the catalog tells it the
// screen cell where its first row is drawn, then sends it the events.
type placeable interface {
	place(x, y int) demoWidget
}

// avatarDemo is the Avatar entry: an avatar that follows the pointer, reacts
// to a click, and takes Left and Right for the name and E for the expression.
type avatarDemo struct {
	m    avatar.Model
	name int
}

func newAvatarDemo() avatarDemo {
	a := avatarDemo{}
	a.m = avatar.New(avatarNames[0])
	a.m.Width, a.m.Height, a.m.Background = 16, 8, avatar.BackgroundCircle
	return a
}

func (a avatarDemo) Init() tui.Cmd { return nil }

func (a avatarDemo) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if k, ok := msg.(tui.Key); ok {
		switch {
		case k.Type == tui.KeyRight, k.Type == tui.KeyLeft:
			step := len(avatarNames) - 1 // one back
			if k.Type == tui.KeyRight {
				step = 1
			}
			a.name = (a.name + step) % len(avatarNames)
			a.m = a.renamed(avatarNames[a.name])
		case k.Type == tui.KeyRunes && strings.EqualFold(k.Text, "e"):
			next := (a.m.Expression + 1) % (avatar.ExpressionSick + 1)
			cmd = a.m.SetExpression(next)
		}
		return a, cmd
	}
	a.m, cmd = a.m.Update(msg) // mouse events, and the ticks of its animation
	return a, cmd
}

// renamed is the avatar for name, in the pose and place of the current one.
func (a avatarDemo) renamed(name string) avatar.Model {
	n := avatar.New(name)
	n.Width, n.Height, n.Background = a.m.Width, a.m.Height, a.m.Background
	n.X, n.Y, n.Theme, n.Expression = a.m.X, a.m.Y, a.m.Theme, a.m.Expression
	return n
}

func (a avatarDemo) View() string {
	m := a.m
	picture := m.View()
	if c := demoCaps; c.KittyGraphics || c.InlineImages || c.Sixel {
		img := imageview.New(m.PNG(128), m.Width, m.Height, m.Linearize())
		img.Kitty, img.Inline, img.Sixel = c.KittyGraphics, !c.KittyGraphics && c.InlineImages, !c.KittyGraphics && !c.InlineImages && c.Sixel
		picture = img.View()
	}
	return picture + "\n" + m.Name + ": " + m.Shape().String() + ", " + m.Expression.String()
}

func (a avatarDemo) focus() (demoWidget, tui.Cmd) { return a, nil }
func (a avatarDemo) blur() demoWidget             { return a }
func (a avatarDemo) started() (demoWidget, tui.Cmd) {
	return a, nil
}

func (a avatarDemo) withTheme(t theme.Theme) demoWidget {
	a.m = a.m.SetTheme(t)
	return a
}

func (a avatarDemo) place(x, y int) demoWidget {
	a.m.X, a.m.Y = x, y
	return a
}
