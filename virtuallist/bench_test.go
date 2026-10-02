package virtuallist

import (
	"strconv"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

// BenchmarkVirtualList10k measures View() over a 10,000-item list scrolled
// to the middle. Cost should track the visible window (Height),
// not ItemCount, so a regression to O(n) rendering shows up here.
func BenchmarkVirtualList10k(b *testing.B) {
	m := New(10000, 30, func(i int) string { return "item " + strconv.Itoa(i) })
	m.LineDown(5000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// BenchmarkList10kWheelScroll measures a wheel event plus View over a
// 10,000-item list, alternating direction so it stays mid-list.
func BenchmarkList10kWheelScroll(b *testing.B) {
	m := New(10000, 30, func(i int) string { return "item " + strconv.Itoa(i) })
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 80, H: 30}
	m.LineDown(5000)
	down := tui.MouseEvent{X: 1, Y: 1, Action: tui.MouseActionPress, Button: tui.MouseButtonWheelDown}
	up := down
	up.Button = tui.MouseButtonWheelUp
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ev := down
		if i%2 == 1 {
			ev = up
		}
		m, _ = m.Update(ev)
		_ = m.View()
	}
}
