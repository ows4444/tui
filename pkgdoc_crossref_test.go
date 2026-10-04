package tui_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// imageview's package comment names its three protocols in the order they
// are tried, and says which tui.Capabilities field goes with each.
func TestImageviewDocNamesItsProtocolsInOrder(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "imageview/imageview.go", nil, parser.ParseComments|parser.PackageClauseOnly)
	if err != nil {
		t.Fatal(err)
	}
	if f.Doc == nil {
		t.Fatal("imageview has no package comment")
	}
	doc := strings.Join(strings.Fields(f.Doc.Text()), " ")
	at := -1
	for _, name := range []string{"Kitty graphics sends", "Inline images (OSC 1337)", "Sixel is decoded"} {
		i := strings.Index(doc, name)
		if i < 0 {
			t.Fatalf("the package comment does not describe %q", name)
		}
		if i < at {
			t.Errorf("%q is described out of order", name)
		}
		at = i
	}
	for _, field := range []string{"KittyGraphics", "InlineImages", "Sixel", "tried in that order"} {
		if !strings.Contains(doc, field) {
			t.Errorf("the package comment does not mention %q", field)
		}
	}
}

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
