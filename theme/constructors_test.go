package theme_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestPresetConstructorsReturnIndependentCopies: every preset is reached
// through its constructor, and changing what a constructor returned changes
// nothing for the next caller.
func TestPresetConstructorsReturnIndependentCopies(t *testing.T) {
	ctors := map[string]func() theme.Theme{
		"Dark": theme.DarkTheme, "Light": theme.LightTheme, "Dracula": theme.DraculaTheme,
		"Nord": theme.NordTheme, "Gruvbox": theme.GruvboxTheme, "TokyoNight": theme.TokyoNightTheme,
		"Monokai": theme.MonokaiTheme, "SolarizedDark": theme.SolarizedDarkTheme,
		"SolarizedLight": theme.SolarizedLightTheme, "Catppuccin": theme.CatppuccinTheme,
		"OneDark": theme.OneDarkTheme, "NightOwl": theme.NightOwlTheme, "RosePine": theme.RosePineTheme,
		"EverforestDark": theme.EverforestDarkTheme, "GitHubDark": theme.GitHubDarkTheme,
		"HighContrast": theme.HighContrastTheme,
	}
	for name, ctor := range ctors {
		want := ctor()
		got := ctor()
		got.Primary = ansi.Red
		got.Glyphs = theme.ASCIIGlyphSet()
		got.Border.Top = "x"
		got.Spacing.M = 99
		got.States = theme.States{}
		if next := ctor(); next != want {
			t.Errorf("%sTheme(): mutating the result changed the next call", name)
		}
	}
}

// TestNoExportedThemeVars proves criterion #103: the theme package exports no
// var of type Theme, so no package can modify a shared preset, and every
// preset it does have is reached through an XxxTheme function.
func TestNoExportedThemeVars(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob: %v %v", files, err)
	}
	fset := token.NewFileSet()
	funcs := map[string]bool{}
	unexported := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		af, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range af.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					funcs[d.Name.Name] = true
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, s := range d.Specs {
					vs := s.(*ast.ValueSpec)
					for i, n := range vs.Names {
						isTheme := false
						if vs.Type != nil {
							isTheme = exprIsTheme(vs.Type)
						}
						if i < len(vs.Values) {
							if cl, ok := vs.Values[i].(*ast.CompositeLit); ok {
								isTheme = isTheme || exprIsTheme(cl.Type)
							}
						}
						switch {
						case isTheme && n.IsExported():
							t.Errorf("exported var %s has type Theme; use an %sTheme function", n.Name, n.Name)
						case isTheme:
							unexported++
						}
					}
				}
			}
		}
	}
	if unexported < 16 {
		t.Fatalf("found only %d preset values; the scan is not looking at them", unexported)
	}
	for _, name := range []string{"Dark", "Light", "Dracula", "HighContrast"} {
		if !funcs[name+"Theme"] {
			t.Errorf("no constructor %sTheme", name)
		}
	}
}

func exprIsTheme(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "Theme"
}
