package main

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxPages    = 500
	maxPageSize = 1 << 20
)

// page is one Markdown file of the docs.
type page struct {
	path  string
	title string
	src   string
	lines []string
}

// loadPages reads every .md file under fsys: README.md first, then by path.
// Unreadable and oversized files are skipped, and at most maxPages are kept.
func loadPages(fsys fs.FS) []page {
	var out []page
	_ = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil
		case d.IsDir():
			if p != "." && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		case !strings.EqualFold(path.Ext(p), ".md") || len(out) >= maxPages:
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil || len(b) > maxPageSize || !utf8.Valid(b) {
			return nil
		}
		src := strings.ReplaceAll(string(b), "\r\n", "\n")
		out = append(out, page{path: p, title: titleOf(p, src), src: src, lines: strings.Split(src, "\n")})
		return nil
	})
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := strings.EqualFold(out[i].path, "README.md"), strings.EqualFold(out[j].path, "README.md")
		if ri != rj {
			return ri
		}
		return out[i].path < out[j].path
	})
	return out
}

// titleOf is the text of the first "# " heading, or the file's name.
func titleOf(p, src string) string {
	fence := false
	for _, l := range strings.Split(src, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			fence = !fence
			continue
		}
		if !fence && strings.HasPrefix(l, "# ") {
			return strings.TrimSpace(strings.Trim(l[2:], "# "))
		}
	}
	return strings.TrimSuffix(path.Base(p), path.Ext(p))
}

// target is where a link points.
type target struct {
	external bool   // a web or mail address, not a page
	page     string // path of the page, or "" for the current one
	anchor   string
}

// resolve reads link relative to the page at from.
func resolve(from, link string) target {
	link = strings.TrimSpace(link)
	if i := strings.Index(link, ":"); i > 0 && !strings.ContainsAny(link[:i], "/#?.") || strings.HasPrefix(link, "//") {
		return target{external: true}
	}
	p, anchor, _ := strings.Cut(link, "#")
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return target{anchor: anchor}
	}
	if path.Ext(p) == "" {
		p += ".md"
	}
	return target{page: path.Clean(path.Join(path.Dir(from), p)), anchor: anchor}
}

// hit is one search result: a line of a page.
type hit struct {
	page    int
	line    int
	heading string // text of the nearest heading above, or ""
	text    string
	rank    int
}

const maxHits = 100

// search finds the lines that contain q (case-insensitive) in every page, pages
// whose title contains it first, then by the number of matching lines.
func search(pages []page, q string) []hit {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}
	var out []hit
	for pi, pg := range pages {
		count := 0
		heading, fence := "", false
		var local []hit
		for li, l := range pg.lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
				fence = !fence
			}
			if !fence && strings.HasPrefix(l, "#") {
				if h := strings.TrimSpace(strings.TrimLeft(l, "#")); h != "" && strings.HasPrefix(strings.TrimLeft(l, "#"), " ") {
					heading = h
				}
			}
			if strings.Contains(strings.ToLower(l), q) {
				count++
				local = append(local, hit{page: pi, line: li, heading: heading, text: strings.TrimSpace(l)})
			}
		}
		rank := count
		if strings.Contains(strings.ToLower(pg.title), q) {
			rank += 1000
		}
		for i := range local {
			local[i].rank = rank
		}
		out = append(out, local...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].rank > out[j].rank })
	if len(out) > maxHits {
		out = out[:maxHits]
	}
	return out
}

// plain strips the inline markup a heading may carry.
func plain(s string) string {
	return strings.NewReplacer("**", "", "__", "", "*", "", "_", "", "`", "").Replace(strings.TrimSpace(s))
}
