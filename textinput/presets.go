package textinput

// NewSearch returns a Model configured for search: a "/ " prompt and a
// "Search..." placeholder, with New's default styling. Everything else
// (Update, View, scrolling) is a plain Model.
func NewSearch() Model {
	m := New()
	m.Prompt = "/ "
	m.Placeholder = "Search..."
	return m
}

// NewPath returns a Model configured for file-path entry: a "/path/to/file"
// placeholder, with New's default styling. Everything else is a plain Model.
func NewPath() Model {
	m := New()
	m.Placeholder = "/path/to/file"
	return m
}
