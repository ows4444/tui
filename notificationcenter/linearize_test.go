package notificationcenter

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/widgets"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(5, 0)
	if got := m.Linearize(); got != "" {
		t.Errorf("empty queue = %q", got)
	}
	m.Push(Notification{Message: "Build passed", Variant: widgets.VariantSuccess})
	m.Push(Notification{Message: "Disk almost full", Variant: widgets.VariantWarning})
	want := "Notifications, 2\nsuccess 1 of 2: Build passed\nwarning 2 of 2: Disk almost full"
	if got := m.Linearize(); got != want {
		t.Errorf("Linearize =\n%s\nwant\n%s", got, want)
	}
}
