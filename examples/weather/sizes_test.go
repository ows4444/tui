package main

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/internal/testutil"
)

// fastFixture answers at once and always says it is the same moment.
func fastFixture() *fixture {
	f := newFixture()
	f.delay = func(place) time.Duration { return 0 }
	f.now = func() time.Time { return time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC) }
	return f
}

// loadedModel is a model that has already got Berlin's forecast.
func loadedModel() model {
	f := fastFixture()
	m := newModel(f, 0, "")
	m.now = f.now
	m.state, m.target = loading, cities[0]
	m = m.onLoaded(loadedMsg{p: cities[0], fc: f.build(cities[0])})
	return m
}

// TestSizeMatrix renders the example at 40x10, 80x24, 120x40 and 300x80 and
// checks each against its golden file with no line wider than the terminal.
func TestSizeMatrix(t *testing.T) {
	testutil.SizeMatrix(t, func() tui.Model { return loadedModel() })
}
