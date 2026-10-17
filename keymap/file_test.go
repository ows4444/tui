package keymap

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	got, err := Parse(strings.NewReader(`{"quit": ["q", "ctrl+c", "q"], "down": "j", "help": []}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Overrides{"quit": {"q", "ctrl+c"}, "down": {"j"}, "help": {}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse = %v; want %v", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	for in, want := range map[string]string{
		`{"quit": ["ctrl+"]}`:   `"quit": "ctrl+" is not a key name`,
		`{"quit": ["Enter"]}`:   "is not a key name",
		`{"quit": ["f25"]}`:     "is not a key name",
		`{"quit": 3}`:           "want a key name or a list",
		`{"quit": [1]}`:         "want a key name or a list",
		`["quit"]`:              "keymap:",
		`{"a": "q"} {"b": "q"}`: "more than one",
		``:                      "keymap:",
	} {
		_, err := Parse(strings.NewReader(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Parse(%q) error = %v; want it to contain %q", in, err, want)
		}
	}
}

func TestValidKey(t *testing.T) {
	for _, k := range []string{"j", "?", "+", "é", "enter", "ctrl+c", "ctrl+alt+x", "shift+tab", "alt+enter", "ctrl++", "f1", "f24", "pgdown", "media-next", "ctrl+space"} {
		if !ValidKey(k) {
			t.Errorf("ValidKey(%q) = false", k)
		}
	}
	for _, k := range []string{"", " ", "ctrl+", "Enter", "f0", "f25", "f01", "ctrl+foo", "two words", "jj", "ctrl+ "} {
		if ValidKey(k) {
			t.Errorf("ValidKey(%q) = true", k)
		}
	}
}

func TestApply(t *testing.T) {
	off := NewBinding("help", "?")
	off.SetEnabled(false)
	in := []Binding{NewBinding("quit", "q"), {Keys: []string{"j"}, Desc: "down", Scope: "list"}, off}
	out, unknown := Apply(in, Overrides{"quit": {"x", "ctrl+c"}, "down": {}, "help": {"h"}, "nope": {"z"}, "also nope": {"y"}})
	if !reflect.DeepEqual(unknown, []string{"also nope", "nope"}) {
		t.Fatalf("unknown = %v", unknown)
	}
	if !reflect.DeepEqual(out[0].Keys, []string{"x", "ctrl+c"}) {
		t.Fatalf("quit keys = %v", out[0].Keys)
	}
	if len(out[1].Keys) != 0 || out[1].Scope != "list" {
		t.Fatalf("down = %+v; want unbound, scope kept", out[1])
	}
	if out[2].Enabled() || !reflect.DeepEqual(out[2].Keys, []string{"h"}) {
		t.Fatalf("help = %+v; want rebound and still disabled", out[2])
	}
	if !reflect.DeepEqual(in[0].Keys, []string{"q"}) {
		t.Fatal("Apply changed its input")
	}
}

func TestApplyReportsConflicts(t *testing.T) {
	out, _ := Apply([]Binding{NewBinding("quit", "q"), NewBinding("help", "?")}, Overrides{"help": {"q"}})
	var r Registry
	var failed bool
	for _, b := range out {
		if _, err := r.Add(b); err != nil {
			failed = true
		}
	}
	if !failed {
		t.Fatal("rebinding help to q should conflict with quit")
	}
}
