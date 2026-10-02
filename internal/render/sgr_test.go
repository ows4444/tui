package render

import "testing"

func TestCellRendererSGRResetKeepsLink(t *testing.T) {
	var st cellStyle
	st.link = 3
	if !applySGR(&st, "1") || !applySGR(&st, "0") || st.link != 3 || st.attrs != 0 {
		t.Errorf("state after reset = %+v", st)
	}
	if applySGR(&st, "4:9") || applySGR(&st, "38:2:1:2") || applySGR(&st, "58:5:1") {
		t.Error("malformed or unmodelled colon SGR accepted")
	}
}
