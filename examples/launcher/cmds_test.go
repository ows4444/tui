package main

import (
	"strings"
	"testing"
)

func TestParseCommands(t *testing.T) {
	cs, err := parseCommands("# project\n\nbuild | Compile | go build ./...\n  test|Run tests|go test -race ./... | tee out.txt\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 2 || cs[0].name != "build" || cs[0].shell != "go build ./..." || cs[1].desc != "Run tests" {
		t.Fatalf("commands = %+v", cs)
	}
	if cs[1].shell != "go test -race ./... | tee out.txt" {
		t.Fatalf("a pipe inside the command was cut: %q", cs[1].shell)
	}
}

func TestParseCommandsErrors(t *testing.T) {
	for in, want := range map[string]string{
		"":                     "no commands",
		"# only a comment\n":   "no commands",
		"build | missing":      "line 1",
		"a | b |  \n":          "cannot be empty",
		" | b | echo\n":        "cannot be empty",
		"a | b | x\na | c | y": "line 2: a appears twice",
	} {
		_, err := parseCommands(in)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("parseCommands(%q) err = %v, want %q", in, err, want)
		}
	}
}

func TestSampleCommandsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range sampleCommands() {
		if c.name == "" || c.desc == "" || len(c.script) == 0 || c.shell != "" || seen[c.name] {
			t.Errorf("bad sample command %+v", c)
		}
		seen[c.name] = true
	}
	if len(seen) < 8 {
		t.Errorf("only %d sample commands", len(seen))
	}
}
