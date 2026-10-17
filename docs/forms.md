# Forms

`form` is a column of fields with validators. Tab and Shift+Tab move between
them, Enter checks them all, and only a form whose every field passes yields a
`SubmittedMsg`.

```go
m := form.New(
	form.Field{Name: "email", Label: "Email", Validators: []form.Validator{form.Required(), form.Email()}},
	form.Field{Name: "password", Label: "Password", Secret: true, Validators: []form.Validator{form.Required(), form.MinLen(8)}},
	form.Field{Name: "plan", Label: "Plan", Kind: form.FieldSelect, Options: []string{"Free", "Pro"}},
	form.Field{Name: "terms", Label: "Accept the terms", Kind: form.FieldCheckbox, Validators: []form.Validator{mustBeTrue}},
)
cmd := m.Focus()
```

Forward every message to `m.Update`; handle `form.SubmittedMsg{Values}`, a map
from field name to value. Every value is a string, so one `Validator`
(`func(string) error`) checks any field. After a failed submit each failing
field shows its first error and focus goes to the first; an error clears once the
field validates.

## Field kinds

| Kind | Value | Notes |
|---|---|---|
| `FieldText` (the default) | the text | `Secret` masks it, `Placeholder`, `Width` |
| `FieldSelect` | the chosen option | Left and Right choose; `OptionsFunc` makes the options depend on other fields |
| `FieldCheckbox` | `"true"` or `"false"` | Space toggles; `form.Bool(m, name)` |
| `FieldConfirm` | `"true"` or `"false"` | a Yes/No choice answered with Y and N; `form.Bool` reads it |
| `FieldMultiSelect` | chosen options joined by `,` | `form.Selected(m, name)` |
| `FieldTextArea` | the text | Enter adds a line, Ctrl+S submits |
| `FieldFile` | the path | Ctrl+O opens a file list under the field; `Options` filters extensions; pair with `form.FileExists()` |
| `FieldNote` | none | a read-only line (`Label`) between fields; takes no focus and is left out of `Values` |

## Typed values

`form.SelectOf[T]` is a select whose options carry values of any type, read back
with `form.Value[T](m, name)`; `form.Int(m, name)` reads a text field as an
integer.

## Groups

`form.NewGroups` shows one titled group of fields at a time. Enter checks the
group and moves on; the last group's Enter submits every field. A group's `Hide`
function can skip it based on the values entered so far, and hidden groups are
left out of `Values`.

## Validators

`Required`, `MinLen`, `MaxLen`, `Email`, `Match(regexp, message)` and
`FileExists` pass an empty value unless it is `Required`, so a field can be
optional but well-formed when filled in. Write your own as a
`func(string) error`.

## Keys

`form.KeyMap` rebinds Submit, the select keys, Toggle, the group keys and the file
browse key. `keymap.Parse` and `keymap.Apply` let a user's key file do it, and
`m.Bindings()` lists what the field with focus honours, for a help line.
