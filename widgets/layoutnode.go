package widgets

import "github.com/ows4444/tui/layout"

// Node adapts any widget's rendered string (Badge, Table, Gauge, ...) to a
// layout.Node. Measure reports the string's natural size; Render pads or
// clips to the allotted size, so the result is always exactly W x H. It
// does not change what the widget functions return.
func Node(rendered string) layout.Node { return layout.Block(rendered) }
