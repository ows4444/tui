package avatar_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

// A list makes a Model for each row on every frame. New is cheap to call again:
// the pictures drawn lately are kept for all Models, so a row is drawn once.
// Decorative keeps a screen reader from hearing the name twice.
func Example_list() {
	for _, name := range []string{"ada", "grace", "ada"} {
		m := avatar.New(name)
		m.Width, m.Height, m.Decorative = 4, 2, true
		first := strings.Split(m.View(), "\n")[0]
		fmt.Printf("%-5s %d cells wide, %q spoken\n", name, ansi.Width(first), m.Linearize())
	}
	// Output:
	// ada   4 cells wide, "" spoken
	// grace 4 cells wide, "" spoken
	// ada   4 cells wide, "" spoken
}

// An app with a house style pins the hue and the silhouette; the name still
// chooses the size, the eyes and the rest.
func Example_houseStyle() {
	blue := 250.0
	for _, name := range []string{"ada", "grace"} {
		m := avatar.New(name)
		m.Hue, m.Tone, m.Silhouette = &blue, avatar.ToneMid, avatar.SilhouetteRound
		fmt.Println(name, m.Shape())
	}
	// Output:
	// ada round
	// grace round
}

// SetExpression eases into a pose and returns the Cmd to hand back from Update;
// assigning Expression changes it at once.
func Example_expression() {
	m := avatar.New("ada")
	cmd := m.SetExpression(avatar.ExpressionHappy)
	fmt.Println(m.Expression, cmd != nil, m.Linearize())
	// Output:
	// happy true Avatar: ada, happy
}

// SVG and PNG are for use outside the terminal. Both draw the pose now shown.
func Example_outsideTheTerminal() {
	m := avatar.New("ada")
	svg, img := m.SVG(), m.PNG(64)
	fmt.Println(strings.HasPrefix(svg, "<svg"), string(img[1:4]))
	// Output:
	// true PNG
}
