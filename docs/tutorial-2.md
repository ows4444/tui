# Tutorial, part 2: search and count

[Part 1](tutorial.md) built a task list you move through with the arrow keys.
This part replaces the hand-written list with a `picker`, so that typing
narrows it, and adds a line that counts what is left. It takes about ten
minutes, and starts from the finished program of part 1.

The finished program is [examples/tutorial2](../examples/tutorial2), with the
test this page ends on.

## 1. A list that filters itself

Replace the `tasks` loop in `View` with a `picker.Model`, and turn its filter
on:

```go
list := picker.NewStrings("[ ] write docs", "[ ] fix bug")
list.Filter = true
```

With `Filter` on, typed keys that the picker does not use as bindings go to a
query line above the list. The rows that fuzzy-match stay, best first, and the
letters that matched are underlined. Esc clears the query. The picker still
reports a choice with `picker.SelectedMsg{Index, Item}`, and `Index` is the
row's place in the list you built, not its place on the screen, so it points
at your own `tasks` slice whatever is shown.

## 2. Two places for the keys

Two widgets want the keyboard: the text field that adds a task and the list.
Keep a `inList bool`; Tab flips it and moves focus with it:

```go
case tui.KeyTab:
	m.inList = !m.inList
	if m.inList {
		m.input.Blur()
		return m, nil
	}
	return m, m.input.Focus()
```

Every other message goes to whichever one has focus. Enter in the field adds
a task; Enter in the list arrives as a `picker.SelectedMsg`.

## 3. Marking a task, and keeping the search

A picker holds strings, so a task's state lives in its label: `[ ]` or `[x]`.
When a task changes, build the list again from the tasks and give it the old
query and cursor back:

```go
m.list = picker.NewStrings(labels...)
m.list.Filter = true
m.list.SetQuery(query)
m.list.SetCursor(cursor)
```

`SetQuery` puts the cursor on the best match, so set the cursor after it.

## 4. A count

`pending` is the number of tasks without a mark, kept while the labels are
built, and the view prints `fmt.Sprintf("%d of %d left", ...)` under the list.

## 5. Test it

`tuitest` drives the whole program through a virtual screen:

```go
s := tuitest.New(newModel(), 64, 14)
defer s.Close()
for _, title := range []string{"write docs", "fix bug", "write tests"} {
	s.Keys(title, "enter")
}
s.Keys("tab", "w", "r", "i") // to the list; narrow to the "write" tasks
s.Keys("down", "enter")      // second match: "write tests"
// the screen now holds "[x] write tests" and "2 of 3 left"
```

## Where next

- [The cookbook](cookbook.md) has short recipes for forms in steps, sorted
  tables, charts and one-line prompts.
- [Testing](testing.md) covers `tuitest` in full.
