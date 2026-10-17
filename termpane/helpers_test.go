//go:build darwin || linux

package termpane

import "github.com/ows4444/tui/layout"

func sizeOf(w, h int) layout.Size { return layout.Size{W: w, H: h} }

func unconstrained() layout.Constraints { return layout.Unconstrained() }
