package ansi

// Clean makes a caller-supplied string safe to draw: unless raw, tabs become
// spaces and every escape sequence and control character except '\n' is
// removed (Sanitize), so a label, file name or message cannot move the
// cursor, clear the screen, set the clipboard or restyle the widgets around
// it. With raw set, s is returned unchanged.
func Clean(raw bool, s string) string {
	if raw {
		return s
	}
	return Sanitize(ExpandTabs(s))
}

// CleanAll is Clean applied to every element of in, returning a new slice.
func CleanAll(raw bool, in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = Clean(raw, s)
	}
	return out
}
