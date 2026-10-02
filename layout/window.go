package layout

// Window returns h consecutive lines from lines that contain the line at
// index cursor, centred on it when there is room and clamped to the ends. If
// lines already fit in h (or h <= 0) it returns lines unchanged; the result
// never has more than h lines.
func Window(lines []string, cursor, h int) []string {
	start, end := WindowRange(len(lines), cursor, h)
	return lines[start:end]
}

// WindowRange is Window over indices: the half-open range [start, end) of h
// consecutive items out of total, containing cursor, centred on it when there
// is room and clamped to the ends. If total already fits in h (or h <= 0) it
// is [0, total).
func WindowRange(total, cursor, h int) (start, end int) {
	if h <= 0 || total <= h {
		return 0, total
	}
	start = cursor - h/2
	if start > total-h {
		start = total - h
	}
	if start < 0 {
		start = 0
	}
	return start, start + h
}
