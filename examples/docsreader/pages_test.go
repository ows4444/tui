package main

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadPagesOrdersAndSkips(t *testing.T) {
	fsys := fstest.MapFS{
		"b.md": {Data: []byte("# Bee\n")}, "README.md": {Data: []byte("# Home\n")}, "a/z.md": {Data: []byte("no heading here")},
		"notes.txt": {Data: []byte("x")}, ".hidden/h.md": {Data: []byte("# H")}, "bad.md": {Data: []byte{0xff, 0xfe}},
		"big.md": {Data: []byte(strings.Repeat("x", maxPageSize+1))}, "UP.MD": {Data: []byte("# Up\n")},
	}
	ps := loadPages(fsys)
	var got []string
	for _, p := range ps {
		got = append(got, p.path)
	}
	if strings.Join(got, ",") != "README.md,UP.MD,a/z.md,b.md" {
		t.Fatalf("pages = %v", got)
	}
	if ps[2].title != "z" || ps[0].title != "Home" {
		t.Fatalf("titles = %q %q", ps[2].title, ps[0].title)
	}
}

func TestLoadPagesIsBounded(t *testing.T) {
	fsys := fstest.MapFS{}
	for i := 0; i < maxPages+20; i++ {
		fsys[strings.Repeat("a", 1)+string(rune('a'+i%26))+strings.Repeat("b", i/26)+".md"] = &fstest.MapFile{Data: []byte("# T\n")}
	}
	if n := len(loadPages(fsys)); n > maxPages {
		t.Fatalf("%d pages", n)
	}
}

func TestTitleOfIgnoresHeadingsInCode(t *testing.T) {
	src := "intro\n```\n# not a title\n```\n# Real ##\n"
	if got := titleOf("x/y.md", src); got != "Real" {
		t.Fatalf("title = %q", got)
	}
	if titleOf("dir/file.md", "no headings") != "file" {
		t.Fatal("fallback title")
	}
}

func TestResolveLinks(t *testing.T) {
	for _, c := range []struct {
		from, link string
		want       target
	}{
		{"README.md", "quickstart.md", target{page: "quickstart.md"}},
		{"README.md", "quickstart", target{page: "quickstart.md"}},
		{"widgets/table.md", "forms.md", target{page: "widgets/forms.md"}},
		{"widgets/table.md", "../README.md#top", target{page: "README.md", anchor: "top"}},
		{"a.md", "#section", target{anchor: "section"}},
		{"a.md", "b.md#x", target{page: "b.md", anchor: "x"}},
		{"a.md", "b.md?raw=1#x", target{page: "b.md", anchor: "x"}},
		{"a.md", "https://example.com/p.md", target{external: true}},
		{"a.md", "mailto:me@example.com", target{external: true}},
		{"a.md", "//cdn.example.com/x", target{external: true}},
		{"a/b.md", "../../outside.md", target{page: "../outside.md"}},
	} {
		if got := resolve(c.from, c.link); got != c.want {
			t.Errorf("resolve(%q, %q) = %+v, want %+v", c.from, c.link, got, c.want)
		}
	}
}

func TestSearchRanksTitlesThenCounts(t *testing.T) {
	ps := loadPages(sampleDocs())
	hits := search(ps, "table")
	if len(hits) == 0 || ps[hits[0].page].path != "widgets/table.md" {
		t.Fatalf("first hit = %+v", hits[:1])
	}
	for _, h := range hits {
		if !strings.Contains(strings.ToLower(ps[h.page].lines[h.line]), "table") {
			t.Fatalf("hit %+v does not contain the query", h)
		}
	}
	if search(ps, "  ") != nil || len(search(ps, "zzzqq")) != 0 {
		t.Fatal("empty or missing query matched")
	}
	if got := search(ps, "TABLE"); len(got) != len(hits) {
		t.Fatalf("search is case sensitive: %d vs %d", len(got), len(hits))
	}
}

func TestSearchReportsTheHeadingAbove(t *testing.T) {
	ps := loadPages(sampleDocs())
	var found bool
	for _, h := range search(ps, "firewall") {
		if ps[h.page].path == "faq.md" && h.heading == "Troubleshooting" {
			found = true
		}
	}
	if !found {
		t.Fatal("the firewall hit does not name its heading")
	}
	// A "#" inside a code fence is not a heading.
	ps = []page{{path: "x.md", title: "x", lines: strings.Split("# Real\n```\n# fake\nneedle\n```\n", "\n")}}
	if h := search(ps, "needle"); len(h) != 1 || h[0].heading != "Real" {
		t.Fatalf("hit = %+v", h)
	}
}

func TestSearchIsBounded(t *testing.T) {
	ps := []page{{path: "x.md", title: "x", lines: strings.Split(strings.Repeat("hit\n", 1000), "\n")}}
	if n := len(search(ps, "hit")); n != maxHits {
		t.Fatalf("%d hits", n)
	}
}

func TestSampleDocsLinkOnlyToRealPages(t *testing.T) {
	ps := loadPages(sampleDocs())
	known := map[string]bool{}
	for _, p := range ps {
		known[p.path] = true
	}
	for _, p := range ps {
		rest := p.src
		for {
			i := strings.Index(rest, "](")
			if i < 0 {
				break
			}
			end := strings.IndexByte(rest[i:], ')')
			link := rest[i+2 : i+end]
			rest = rest[i+end:]
			if t2 := resolve(p.path, link); !t2.external && t2.page != "" && !known[t2.page] {
				t.Errorf("%s links to %s, which does not exist", p.path, link)
			}
		}
	}
}
