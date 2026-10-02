package main

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/faces"
)

// oldFaceAt is the hand-computed click arithmetic that click() used before it
// read its regions from the layout. It is kept here only to prove the new
// regions hit exactly the same cells; -1 means no face.
func oldFaceAt(m model, x, y int) int {
	const oldSpriteTop = 2
	w, h := m.cell()
	if !m.wall {
		if x >= 0 && x < w && y >= oldSpriteTop && y < oldSpriteTop+h {
			return m.player.Index()
		}
		return -1
	}
	pitch := h + 2
	col, row := x/(w+wallGap), (y-oldSpriteTop)/pitch
	inCell := x%(w+wallGap) < w && (y-oldSpriteTop)%pitch < h
	if x < 0 || y < oldSpriteTop || !inCell || col >= m.cols() || row >= wallRows {
		return -1
	}
	i := m.page*m.perPage() + row*m.cols() + col
	if i >= faces.Count() {
		return -1
	}
	return i
}

// Every cell of a grid larger than the screen hits the same face as before,
// in the single view and the wall, at both sizes, on the first and a partial
// last page, at several widths.
func TestRegionsMatchTheOldArithmetic(t *testing.T) {
	for _, width := range []int{0, 30, 50, 80, 120, 300} {
		for _, wall := range []bool{false, true} {
			for _, large := range []bool{false, true} {
				m := initialModel()
				m.still = true
				if width > 0 {
					next, _ := m.Update(tui.ResizeMsg{Width: width, Height: 40})
					m = next.(model)
				}
				if large {
					m = m.toggleSize()
				}
				if wall {
					m = m.toggleWall()
				}
				pages := []int{0}
				if wall {
					pages = append(pages, m.pages()-1)
				}
				for _, page := range pages {
					m.page = page
					t.Run(fmt.Sprintf("w%d wall=%v large=%v page=%d", width, wall, large, page), func(t *testing.T) {
						regions := m.regions()
						for y := -3; y < 60; y++ {
							for x := -3; x < 340; x++ {
								want := oldFaceAt(m, x, y)
								got := -1
								if h, ok := regions.At(x, y); ok {
									got = h.ID
								}
								if got != want {
									t.Fatalf("(%d,%d): regions = %d, old arithmetic = %d", x, y, got, want)
								}
							}
						}
					})
				}
			}
		}
	}
}

// Every named region contains the first cell its sprite is drawn in.
func TestEveryRegionContainsItsSpritesFirstCell(t *testing.T) {
	for _, wall := range []bool{false, true} {
		m := initialModel()
		m.still = true
		if wall {
			m = m.toggleWall()
		}
		regions := m.regions()
		want := 1
		if wall {
			want = len(m.visible())
		}
		if regions.Len() != want {
			t.Fatalf("wall=%v: %d regions, want %d", wall, regions.Len(), want)
		}
		for _, i := range m.shown() {
			r, ok := m.regionOf(i)
			if !ok || r.Empty() {
				t.Fatalf("face %d has no region", i)
			}
			if h, ok := regions.At(r.X, r.Y); !ok || h.ID != i {
				t.Errorf("face %d: its region's first cell (%d,%d) hits %v", i, r.X, r.Y, h)
			}
		}
	}
}

// faceAt is the same lookup click() uses.
func TestFaceAtAgreesWithRegions(t *testing.T) {
	m := initialModel().toggleWall()
	x, y := 3, 4
	i, ok := m.faceAt(x, y)
	if h, hok := m.regions().At(x, y); ok != hok || i != h.ID {
		t.Errorf("faceAt = %d,%v; regions = %d,%v", i, ok, h.ID, hok)
	}
}
