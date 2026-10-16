package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

const doc = `{"name":"tui","version":"1.0.0","deps":[],"authors":[{"name":"Ada","city":"London"},{"name":"Grace","city":"New York"}],"stars":0}`

func screen(s *tuitest.Session) string {
	return ansi.StripANSI(strings.Join(s.Screen(), "\n"))
}

func TestShowsTheTopLevelAndSearches(t *testing.T) {
	m, err := newModel([]byte(doc), "doc.json")
	if err != nil {
		t.Fatal(err)
	}
	s := tuitest.New(m, 70, 16)
	defer s.Close()
	got := screen(s)
	for _, want := range []string{"doc.json", "name: \"tui\"", "version", "authors []"} {
		if !strings.Contains(got, want) {
			t.Errorf("first screen lacks %q:\n%s", want, got)
		}
	}
	s.Keys("g", "r", "a", "c", "e")
	got = screen(s)
	if !strings.Contains(got, "Grace") || strings.Contains(got, "\"Ada\"") || strings.Contains(got, "version") {
		t.Errorf("search did not narrow the tree:\n%s", got)
	}
	if !strings.Contains(got, "match \"grace\"") {
		t.Errorf("status does not name the query:\n%s", got)
	}
	s.Keys("ctrl+c")
}

func TestBadJSONIsAnError(t *testing.T) {
	if _, err := newModel([]byte(`{"a":`), "bad.json"); err == nil || !strings.Contains(err.Error(), "bad.json") {
		t.Errorf("err = %v", err)
	}
}

func TestReadFromStdinOrFile(t *testing.T) {
	data, src, err := read(nil, strings.NewReader(`[1]`))
	if err != nil || string(data) != `[1]` || src != "stdin" {
		t.Errorf("stdin: %q %q %v", data, src, err)
	}
	if _, _, err := read([]string{"/no/such/file.json"}, nil); err == nil {
		t.Error("a missing file did not fail")
	}
}
