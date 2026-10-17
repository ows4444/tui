package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/prompt"
)

// keys feeds question i its own input, so each prompt's Program reads only its
// own keys.
func keys(per ...string) func(int) prompt.Config {
	return func(i int) prompt.Config {
		in := ""
		if i < len(per) {
			in = per[i]
		}
		return prompt.Config{Program: []tui.ProgramOption{
			tui.WithInput(strings.NewReader(in)),
			tui.WithOutput(io.Discard),
			tui.WithReducedMotion(true),
		}}
	}
}

const (
	down  = "\x1b[B"
	enter = "\r"
)

func TestAskCollectsEveryAnswer(t *testing.T) {
	a, err := ask(keys(
		"Ada Lovelace"+enter,
		"ada@example.com"+enter,
		"12"+enter,
		down+down+down+enter,    // Student
		" "+down+down+" "+enter, // Editor plugins, Test frameworks
		"9"+enter,
		"  Great tool  "+enter,
		enter, // default: yes
	))
	if err != nil {
		t.Fatal(err)
	}
	want := answers{Name: "Ada Lovelace", Email: "ada@example.com", Years: 12, Role: "student", Score: 9, Comment: "Great tool", Send: true}
	if a.Name != want.Name || a.Email != want.Email || a.Years != 12 || a.Role != "student" || a.Score != 9 || a.Comment != "Great tool" || !a.Send {
		t.Errorf("answers = %+v", a)
	}
	if strings.Join(a.Tools, "|") != "Editor plugins|Test frameworks" {
		t.Errorf("tools = %v", a.Tools)
	}
}

func TestOptionalCommentAndNoTools(t *testing.T) {
	a, err := ask(keys("Bo"+enter, "bo@example.com"+enter, "0"+enter, enter, enter, "0"+enter, enter, "n"+enter))
	if err != nil {
		t.Fatal(err)
	}
	if a.Comment != "" || len(a.Tools) != 0 || a.Role != "backend" || a.Years != 0 || a.Score != 0 {
		t.Errorf("answers = %+v", a)
	}
}

// Bad answers are never accepted: with the keys ending while the prompt still
// holds one, the answer is ErrCanceled, not the bad value.
func TestBadAnswersAreNeverAccepted(t *testing.T) {
	cases := []struct {
		name  string
		input []string
	}{
		{"empty name", []string{"   " + enter}},
		{"email", []string{"A" + enter, "not-an-email" + enter}},
		{"years", []string{"A" + enter, "a@b.co" + enter, "61" + enter}},
		{"years text", []string{"A" + enter, "a@b.co" + enter, "many" + enter}},
		{"score", []string{"A" + enter, "a@b.co" + enter, "3" + enter, enter, enter, "11" + enter}},
	}
	for _, c := range cases {
		a, err := ask(keys(c.input...))
		if !errors.Is(err, prompt.ErrCanceled) {
			t.Errorf("%s: err = %v, answers %+v", c.name, err, a)
		}
	}
}

func TestCancelReturnsErrCanceled(t *testing.T) {
	_, err := ask(keys("\x03"))
	if !errors.Is(err, prompt.ErrCanceled) {
		t.Errorf("err = %v", err)
	}
}

func TestParsers(t *testing.T) {
	for _, ok := range []string{"a@b.co", "ada@example.com"} {
		if _, err := parseEmail(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "ada", "Ada <ada@example.com>", "a@", "@b.co"} {
		if _, err := parseEmail(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	for s, ok := range map[string]bool{"0": true, "60": true, "61": false, "-1": false, "x": false, "": false} {
		if _, err := parseYears(s); (err == nil) != ok {
			t.Errorf("parseYears(%q) err=%v", s, err)
		}
	}
	for s, ok := range map[string]bool{"0": true, "10": true, "11": false, "-1": false, "5.5": false} {
		if _, err := parseScore(s); (err == nil) != ok {
			t.Errorf("parseScore(%q) err=%v", s, err)
		}
	}
}

func TestSummaryText(t *testing.T) {
	var b bytes.Buffer
	a := answers{Name: "Ada", Email: "ada@example.com", Years: 12, Role: "backend", Tools: []string{"Linters", "Profilers"}, Score: 9, Comment: "Nice"}
	if err := summary(&b, a, false); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Thank you, Ada.", "ada@example.com", "Linters, Profilers", "9 / 10", "Nice"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("summary lacks %q:\n%s", want, b.String())
		}
	}
	b.Reset()
	a.Tools, a.Comment = nil, ""
	_ = summary(&b, a, false)
	if !strings.Contains(b.String(), "none") || strings.Contains(b.String(), "Comment") {
		t.Errorf("empty tools and comment:\n%s", b.String())
	}
}

func TestSummaryJSON(t *testing.T) {
	var b bytes.Buffer
	a := answers{Name: "Ada", Years: 3, Tools: []string{"Linters"}, Send: true}
	if err := summary(&b, a, true); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, b.String())
	}
	if got["name"] != "Ada" || got["years"] != float64(3) {
		t.Errorf("json = %v", got)
	}
	if _, ok := got["Send"]; ok {
		t.Error("the internal Send flag leaked into the JSON")
	}
	if _, ok := got["comment"]; ok {
		t.Error("an empty comment was written")
	}
}

func TestSendFinishes(t *testing.T) {
	pr, pw := io.Pipe() // an input that stays open and silent, so only the work ends the prompt
	defer pw.Close()
	cfg := prompt.Config{Program: []tui.ProgramOption{tui.WithInput(pr), tui.WithOutput(io.Discard), tui.WithReducedMotion(true)}}
	if err := send(cfg, time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestSendStopsOnCtrlC(t *testing.T) {
	cfg := keys("\x03")(0)
	err := send(cfg, 200*time.Millisecond)
	if err == nil {
		t.Fatal("Ctrl+C did not stop the upload")
	}
}
