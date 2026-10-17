package main

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/markdown"
)

const (
	maxEntries = 500       // entries listed from one directory
	maxBytes   = 512 << 10 // bytes read from one file
)

// entry is one item of a directory.
type entry struct {
	name string
	path string
	dir  bool
	size int64
}

// readDir lists dir: directories first, then files, each by name ignoring
// case. Names that start with a dot are left out unless hidden is set. A
// directory with more than maxEntries entries is cut, and cut says so.
func readDir(fsys fs.FS, dir string, hidden bool) (out []entry, cut bool, err error) {
	ents, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, false, err
	}
	for _, e := range ents {
		if !hidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if len(out) == maxEntries {
			cut = true
			break
		}
		en := entry{name: e.Name(), path: path.Join(dir, e.Name()), dir: e.IsDir()}
		if info, err := e.Info(); err == nil {
			en.size = info.Size()
		}
		out = append(out, en)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].dir != out[j].dir {
			return out[i].dir
		}
		return strings.ToLower(out[i].name) < strings.ToLower(out[j].name)
	})
	return out, cut, nil
}

// file is a loaded source file, ready to draw.
type file struct {
	path      string
	lang      string
	size      int64
	lines     []string
	spans     [][]markdown.Span // one row per line, or nil for plain text
	truncated bool
	problem   string // why there is nothing to show, or ""
}

var langByExt = map[string]string{
	".go": "go", ".js": "js", ".mjs": "js", ".cjs": "js", ".jsx": "js", ".ts": "ts", ".tsx": "ts",
	".py": "python", ".rs": "rust", ".c": "c", ".h": "c", ".cc": "cpp", ".cpp": "cpp", ".hpp": "cpp",
	".java": "java", ".kt": "kotlin", ".swift": "swift", ".rb": "ruby", ".php": "php", ".lua": "lua",
	".json": "json", ".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".ini": "ini", ".md": "markdown",
	".sh": "sh", ".bash": "sh", ".zsh": "sh", ".sql": "sql", ".html": "html", ".htm": "html",
	".css": "css", ".xml": "xml", ".svg": "xml", ".diff": "diff", ".patch": "diff", ".mk": "makefile",
}

// language is the name the highlighter knows for the file at p, or "".
func language(p string) string {
	base := path.Base(p)
	switch strings.ToLower(base) {
	case "makefile", "gnumakefile":
		return "makefile"
	case "dockerfile":
		return "dockerfile"
	}
	return langByExt[strings.ToLower(path.Ext(base))]
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit && exp < 4; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTP"[exp])
}

// load reads the start of the file at p, splits it into lines and, for a known
// language, colours them. A binary or unreadable file gives a file whose
// problem says so.
func load(fsys fs.FS, p string, size int64) file {
	f := file{path: p, size: size, lang: language(p)}
	h, err := fsys.Open(p)
	if err != nil {
		f.problem = "Cannot open: " + err.Error()
		return f
	}
	defer func() { _ = h.Close() }()
	b, err := io.ReadAll(io.LimitReader(h, maxBytes))
	if err != nil {
		f.problem = "Cannot read: " + err.Error()
		return f
	}
	f.truncated = size > maxBytes
	for len(b) > 0 && !utf8.Valid(b) && f.truncated && !utf8.FullRune(b[max(0, len(b)-3):]) {
		b = b[:len(b)-1] // a character cut by the limit
	}
	if !utf8.Valid(b) || strings.IndexByte(string(b), 0) >= 0 {
		f.problem = fmt.Sprintf("Binary file, %s.", humanSize(size))
		return f
	}
	text := strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
	if text == "" && len(b) == 0 {
		f.problem = "Empty file."
		return f
	}
	for _, l := range strings.Split(text, "\n") {
		f.lines = append(f.lines, ansi.Sanitize(strings.ReplaceAll(l, "\t", "    ")))
	}
	if f.lang != "" {
		if rows, ok := markdown.DefaultLexer().Lex(strings.Join(f.lines, "\n"), f.lang); ok && len(rows) == len(f.lines) {
			f.spans = rows
		}
	}
	return f
}

// find lists the lines that contain q, ignoring case.
func (f file) find(q string) []int {
	if q == "" {
		return nil
	}
	q = strings.ToLower(q)
	var out []int
	for i, l := range f.lines {
		if strings.Contains(strings.ToLower(l), q) {
			out = append(out, i)
		}
	}
	return out
}
