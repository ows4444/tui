package form_test

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
)

// A signup form: type into the focused field, Tab to move on, and Submit (or
// press Enter) to check everything at once. A failed Submit shows the errors
// and focuses the first invalid field; a valid one yields a SubmittedMsg.
func Example() {
	m := form.New(
		form.Field{Name: "email", Label: "Email", Validators: []form.Validator{form.Required(), form.Email()}},
		form.Field{Name: "password", Label: "Password", Secret: true, Validators: []form.Validator{form.Required(), form.MinLen(8)}},
	)
	m.Focus()

	// Submitting an empty form fails and shows the first error of each field.
	m, _ = m.Submit()
	for _, line := range strings.Split(ansi.StripANSI(m.View()), "\n") {
		fmt.Println(strings.TrimRight(line, " ")) // rows are padded for the cursor
	}
	fmt.Println("focused:", m.Current())

	// Fill it in: type, Tab, type.
	for _, r := range "ada@example.com" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}
	m, _ = m.Update(tui.Key{Type: tui.KeyTab})
	for _, r := range "correct horse" {
		m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})})
	}

	_, cmd := m.Submit()
	msg := cmd().(form.SubmittedMsg)
	fmt.Println("email:", msg.Values["email"])
	fmt.Println("password entered:", len(msg.Values["password"]) > 0)

	// Output:
	// Email:
	//   required
	// Password:
	//   required
	// focused: email
	// email: ada@example.com
	// password entered: true
}
