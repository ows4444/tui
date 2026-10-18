package main

import "strings"

type risk int

const (
	low risk = iota
	medium
	high
)

var riskNames = [...]string{"low risk", "medium risk", "high risk"}

// world is the fake project the agent works on: what its tools have changed.
type world struct {
	edited  bool // layout/wrap.go was edited
	cleaned bool // the build directory was removed
}

// tool is a call the agent wants to make.
type tool struct {
	name, arg string
	risk      risk
	icon      string
	diff      []string
	result    func(w world) []string
	fails     func(w world) bool
	effect    func(w world) world
}

// stepKind says what a plan step is.
type stepKind int

const (
	sSay stepKind = iota
	sTool
)

type planStep struct {
	kind stepKind
	say  func(w world) string
	tool tool
}

func say(s string) planStep { return planStep{kind: sSay, say: func(world) string { return s }} }

func sayFn(f func(w world) string) planStep { return planStep{kind: sSay, say: f} }

func call(t tool) planStep { return planStep{kind: sTool, tool: t} }

var readTest = tool{
	name: "read_file", arg: "layout/wrap_test.go", risk: low, icon: "▤",
	result: func(world) []string {
		return []string{"41 lines read", "func TestWrapRowGap(t *testing.T) {", "\tgot := Wrap(1, 1, a, b, c).Render(Size{W: 12, H: 8})", "\t// want two rows of chips with one blank row between", "\tif rows := strings.Count(got, \"\\n\") + 1; rows != 5 {", "\t\tt.Fatalf(\"got %d rows, want 5\", rows)"}
	},
}

var editWrap = tool{
	name: "edit_file", arg: "layout/wrap.go", risk: medium, icon: "✎",
	diff: []string{
		"@@ -38,7 +38,7 @@ func (w wrapNode) Render(s Size) string {",
		" \tfor i, row := range rows {",
		" \t\tout = append(out, row...)",
		"-\t\tif i < len(rows) {",
		"-\t\t\tout = append(out, blank(w.rowGap)...)",
		"+\t\tif i < len(rows)-1 {",
		"+\t\t\tout = append(out, blank(w.rowGap)...)",
		" \t\t}",
		" \t}",
	},
	result: func(world) []string { return []string{"1 file changed, 2 insertions(+), 2 deletions(-)"} },
	effect: func(w world) world { w.edited = true; return w },
}

var runTests = tool{
	name: "run_tests", arg: "go test ./layout", risk: medium, icon: "▶",
	result: func(w world) []string {
		if w.edited {
			return []string{"ok  \tgithub.com/ows4444/tui/layout\t0.412s"}
		}
		return []string{"--- FAIL: TestWrapRowGap (0.00s)", "    wrap_test.go:41: got 6 rows, want 5", "FAIL\tgithub.com/ows4444/tui/layout\t0.398s"}
	},
	fails: func(w world) bool { return !w.edited },
}

var removeBuild = tool{
	name: "run_command", arg: "rm -rf build/ dist/", risk: high, icon: "⚠",
	result: func(world) []string { return []string{"removed build/ (212 files)", "removed dist/ (4 files)"} },
	effect: func(w world) world { w.cleaned = true; return w },
}

// planFor is the scripted agent: the same prompt always gets the same plan.
func planFor(prompt string) []planStep {
	p := strings.ToLower(prompt)
	switch {
	case strings.Contains(p, "test") || strings.Contains(p, "fix") || strings.Contains(p, "bug"):
		return []planStep{
			say("I'll start by reading the failing test."), call(readTest),
			say("The test expects five rows but Wrap produces six: the row gap is added after the last row as well as between rows. I'll stop it adding one after the last."),
			call(editWrap), say("Now I'll run the layout tests to check."), call(runTests),
			sayFn(func(w world) string {
				if w.edited {
					return "The layout tests pass now. The change is two lines in layout/wrap.go; nothing else was touched."
				}
				return "The test still fails because the edit was not applied. Ask me again and approve the change to fix it."
			}),
		}
	case strings.Contains(p, "delete") || strings.Contains(p, "remove") || strings.Contains(p, "clean"):
		return []planStep{say("I can clear out the build output. This deletes files, so I need your approval first."), call(removeBuild),
			sayFn(func(w world) string {
				if w.cleaned {
					return "Done. Run the build again to recreate them."
				}
				return "The build directory is untouched."
			})}
	case strings.Contains(p, "explain") || strings.Contains(p, "what") || strings.Contains(p, "how"):
		return []planStep{say("The layout package arranges nodes in a tree. A node reports the size it wants under some constraints, then draws itself at the size it is finally given. Row and Column share space by grow and shrink weights, GridNode lines cells up in columns, Wrap flows children like words, and Stack puts layers on top of each other. Named nodes let a program ask where anything ended up, which is how mouse clicks find their targets.")}
	}
	return []planStep{say("I am a scripted demo, so I only know a few requests. Try: \"fix the failing test\", \"delete the build directory\" or \"explain the layout package\".")}
}
