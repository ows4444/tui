package render

import "testing"

// parseRow alone expands tabs (direct callers bypass the runtime's fit).
func TestParseRowExpandsTabs(t *testing.T) {
	cr := New()
	row, ok := cr.parseRow("a\tb", nil)
	if !ok || len(row) != 9 || cr.text(row[8]) != "b" {
		t.Errorf("parseRow tab: ok=%v len=%d row=%v", ok, len(row), row)
	}
}
