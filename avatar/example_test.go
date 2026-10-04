package avatar_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

// A name always draws the same avatar. Shape and Colors report what it drew;
// View is the picture, here its four middle rows with the colours stripped,
// so the body shows as ink and the eyes as holes.
func Example() {
	m := avatar.New("alain00")
	m.Width, m.Height = 12, 6
	body, eyes, _ := m.Colors()
	fmt.Println(m.Shape(), body, eyes)
	for _, row := range strings.Split(ansi.StripANSI(m.View()), "\n")[1:5] {
		fmt.Println(strings.TrimRight(row, " "))
	}
	fmt.Println(m.Linearize())
	// Output:
	// round {165 221 255} {7 17 23}
	// ▗▟████████▙▖
	// ▐██▙ ██▙ ██▌
	// ▐███▄███▄▟█▌
	// ▝▜████████▛▘
	// Avatar: alain00
}
