package taginput

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New()
	m.Input.Prompt = "Add tag: "
	if got := m.Linearize(); got != "No tags\nAdd tag, text field, empty" {
		t.Errorf("empty = %q", got)
	}
	m.Tags = []string{"go", "tui"}
	if got := m.Linearize(); got != "Tags: go, tui (2 tags)\nAdd tag, text field, empty" {
		t.Errorf("two tags = %q", got)
	}
	m.Tags = []string{"go"}
	m.MaxTags = 5
	if got := m.Linearize(); got != "Tags: go (1 of 5 tags)\nAdd tag, text field, empty" {
		t.Errorf("with max = %q", got)
	}
	m.MaxTags = 0
	if got := m.Linearize(); got != "Tags: go (1 tag)\nAdd tag, text field, empty" {
		t.Errorf("one tag = %q", got)
	}
}
