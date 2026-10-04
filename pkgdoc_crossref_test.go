package tui_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// The two character packages answer different questions (who is this, and
// how is it feeling), so each package comment points at the other.
func TestAvatarAndFacesPointAtEachOther(t *testing.T) {
	for file, other := range map[string]string{
		"avatar/avatar.go": "package faces",
		"faces/faces.go":   "package avatar",
	} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments|parser.PackageClauseOnly)
		if err != nil {
			t.Fatal(err)
		}
		if f.Doc == nil || !strings.Contains(strings.Join(strings.Fields(f.Doc.Text()), " "), other) {
			t.Errorf("the package comment in %s does not name %q", file, other)
		}
	}
}
