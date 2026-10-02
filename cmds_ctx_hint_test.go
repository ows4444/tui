package tui

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/motion"
)

// Criterion #103: a context Cmd called directly prints a hint naming RunCmd,
// in %v, %s and when wrapped in another value.
func TestUnrunContextCmdExplainsItself(t *testing.T) {
	cmds := map[string]Cmd{
		"FromCtx": FromCtx(func(context.Context) Msg { return nil }),
		"Tick":    Tick(time.Hour, func(time.Time) Msg { return nil }),
		"motion":  FromCtx(func(ctx context.Context) Msg { return motion.After(time.Hour, func(time.Time) Msg { return nil })(ctx) }),
	}
	type wrapped struct{ Inner Msg }
	for name, cmd := range cmds {
		msg := cmd()
		for _, s := range []string{fmt.Sprint(msg), fmt.Sprintf("%v", msg), fmt.Sprintf("%+v", wrapped{msg})} {
			if !strings.Contains(s, "tui.RunCmd") {
				t.Errorf("%s: printed %q, want a hint naming tui.RunCmd", name, s)
			}
		}
	}
}

// Criterion #104: the Cmd type's documentation explains the direct-call case.
func TestCmdDocNamesRunCmd(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "model.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.TYPE || g.Specs[0].(*ast.TypeSpec).Name.Name != "Cmd" {
			continue
		}
		doc := g.Doc.Text()
		for _, want := range []string{"FromCtx", "Tick", "RunCmd", "internal message"} {
			if !strings.Contains(doc, want) {
				t.Errorf("Cmd doc does not mention %q", want)
			}
		}
		return
	}
	t.Fatal("type Cmd not found in model.go")
}
