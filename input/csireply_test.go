package input

import (
	"strings"
	"testing"
)

func TestCSIRepliesReported(t *testing.T) {
	rd := NewReader(strings.NewReader("\x1b[?2026;1$y\x1b[?62;22c\x1b[?0ux"))
	rd.SetReportCSIReplies(true)
	for _, want := range []string{"?2026;1$y", "?62;22c", "?0u"} {
		ev, err := rd.ReadEvent()
		r, ok := ev.(ReplyEvent)
		if err != nil || !ok || r.Kind != '[' || r.Data != want {
			t.Fatalf("got %#v %v, want CSI reply %q", ev, err, want)
		}
	}
	if ev, _ := rd.ReadEvent(); ev.(Key).Type != KeyRunes {
		t.Errorf("trailing key = %#v", ev)
	}
	// Off by default: consumed as KeyUnknown.
	rd = NewReader(strings.NewReader("\x1b[?62;22c"))
	if ev, _ := rd.ReadEvent(); ev.(Key).Type != KeyUnknown {
		t.Errorf("default = %#v", ev)
	}
}
