package theme

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// Criterion #104: the theme package does not import the root tui package, so
// the root package can import it.
func TestThemeDoesNotImportRoot(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("no sources: %v", err)
	}
	for _, f := range files {
		if filepath.Ext(f) == ".go" && len(f) > 8 && f[len(f)-8:] == "_test.go" {
			continue
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range af.Imports {
			if path, _ := strconv.Unquote(imp.Path.Value); path == "github.com/ows4444/tui" {
				t.Errorf("%s imports the root package", f)
			}
		}
	}
}

// Criterion #108: every preset carries all four state styles.
func TestPresetsCarryStates(t *testing.T) {
	var zero ansi.Style
	for name, th := range map[string]Theme{
		"Dark": DarkTheme(), "Light": LightTheme(), "Dracula": DraculaTheme(), "Nord": NordTheme(), "Gruvbox": GruvboxTheme(),
		"TokyoNight": TokyoNightTheme(), "Monokai": MonokaiTheme(), "SolarizedDark": SolarizedDarkTheme(),
		"SolarizedLight": SolarizedLightTheme(), "Catppuccin": CatppuccinTheme(), "OneDark": OneDarkTheme(),
		"NightOwl": NightOwlTheme(), "RosePine": RosePineTheme(), "EverforestDark": EverforestDarkTheme(),
		"GitHubDark": GitHubDarkTheme(), "HighContrast": HighContrastTheme(),
	} {
		s := th.States
		for field, st := range map[string]ansi.Style{"Focus": s.Focus, "Hover": s.Hover, "Disabled": s.Disabled, "Selected": s.Selected} {
			if st == zero {
				t.Errorf("%s.States.%s is empty", name, field)
			}
		}
		if s.Focus != ansi.NewStyle().Foreground(th.Focus) {
			t.Errorf("%s: Focus state is not the Focus role, so contrast is not the one already tested", name)
		}
	}
}

// Criterion #109: an empty state falls back to its colour role, an explicit
// one is kept, and a theme without States renders as it always did.
func TestEmptyStatesFallBackToRoles(t *testing.T) {
	th := Theme{Primary: ansi.Red, Muted: ansi.Blue, Focus: ansi.Green, Text: ansi.White, Selection: ansi.Yellow}
	got := th.ResolvedStates()
	if got.Focus.Render("x") != ansi.NewStyle().Foreground(ansi.Green).Render("x") ||
		got.Hover.Render("x") != ansi.NewStyle().Foreground(ansi.Red).Render("x") ||
		got.Disabled.Render("x") != ansi.NewStyle().Foreground(ansi.Blue).Render("x") ||
		got.Selected.Render("x") != ansi.NewStyle().Foreground(ansi.White).Background(ansi.Yellow).Render("x") {
		t.Fatalf("fallbacks are not built from the roles: %+v", got)
	}

	th.States.Hover = ansi.NewStyle().Bold()
	if th.ResolvedStates().Hover != ansi.NewStyle().Bold() {
		t.Fatal("an explicit state was replaced")
	}
	if (Theme{}).States != (States{}) {
		t.Fatal("the zero Theme has states")
	}
}

func TestAutoPicksByBackground(t *testing.T) {
	a := Auto{Dark: NordTheme(), Light: SolarizedLightTheme()}
	if a.For(ansi.RGB{R: 250, G: 250, B: 250}) != SolarizedLightTheme() || a.For(ansi.RGB{}) != NordTheme() {
		t.Fatal("Auto.For picked the wrong theme")
	}
}
