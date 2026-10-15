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

// Fields in groups: the form shows one group at a time, Enter checks it and
// moves on, and a group can be hidden by what is already entered.
func ExampleNewGroups() {
	m := form.NewGroups(
		form.Group{Title: "Account", Fields: []form.Field{
			{Name: "email", Label: "Email", Value: "ada@example.com", Validators: []form.Validator{form.Required()}},
		}},
		form.Group{Title: "Company", Fields: []form.Field{
			{Name: "company", Label: "Company", Validators: []form.Validator{form.Required()}},
		}, Hide: func(v map[string]string) bool { return v["email"] != "" }},
		form.Group{Title: "Done", Fields: []form.Field{
			{Name: "terms", Label: "Accept terms", Kind: form.FieldCheckbox},
		}},
	)
	m.Focus()
	fmt.Println(m.GroupCount(), "groups, showing", m.CurrentGroup())

	m, _ = m.Update(tui.Key{Type: tui.KeyEnter}) // Account is valid; Company is hidden
	fmt.Println("now showing", m.CurrentGroup(), m.Current())

	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter}) // the last group submits
	fmt.Println(cmd().(form.SubmittedMsg).Values)
	// Output:
	// 2 groups, showing 0
	// now showing 2 terms
	// map[email:ada@example.com terms:false]
}

// A select whose choices carry a typed value, and a select whose options come
// from another field.
func ExampleSelectOf() {
	m := form.New(
		form.SelectOf("size", "Size",
			form.Option[int]{Label: "Small", Value: 1},
			form.Option[int]{Label: "Large", Value: 10}),
		form.Field{Name: "city", Label: "City", Kind: form.FieldSelect,
			OptionsFunc: func(v map[string]string) []string {
				if v["size"] == "Large" {
					return []string{"Tokyo", "Lahore"}
				}
				return []string{"Kyoto"}
			}},
	)
	m.Focus()
	m, _ = m.Update(tui.Key{Type: tui.KeyRight}) // Small -> Large

	size, _ := form.Value[int](m, "size")
	fmt.Println(size, m.Values()["city"])
	// Output: 10 Tokyo
}
