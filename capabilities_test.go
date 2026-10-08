package tui

import (
	"strings"
	"testing"
	"time"
)

func capMsgs(rec recorderModel) (n int, last CapabilitiesMsg) {
	for _, m := range rec.received {
		if c, ok := m.(CapabilitiesMsg); ok {
			n++
			last = c
		}
	}
	return
}

// AC1: DA1 ends the probe with one message even if others went unanswered.
func TestCapabilityProbeDA1SentinelDeliversOnce(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) }) // the probe sets the global cluster width
	write, read, wait := bgRun(t, recorderModel{quitAfter: 3}, WithCapabilityProbe(time.Hour))
	write("\x1b[?2026;1$y\x1b[?62;22c\x1b[?62;22c")
	write("a") // a key after the replies: replies must not leak as keys
	rec := wait().model.(recorderModel)
	n, c := capMsgs(rec)
	if n != 1 {
		t.Fatalf("CapabilitiesMsg count = %d in %#v", n, rec.received)
	}
	if !c.Capabilities.SyncOutput || c.Capabilities.KittyKeyboard || c.Capabilities.GraphemeClusters {
		t.Errorf("caps = %#v", c.Capabilities)
	}
	for _, m := range rec.received {
		if k, ok := m.(Key); ok && k.Type == KeyUnknown {
			t.Errorf("reply leaked as %#v", m)
		}
	}
	out := string(read())
	for _, q := range []string{queryDECRQM2026, queryDECRQM2027, queryKittyKeyboard, queryKittyGraphics, queryXTVersion, queryDA1} {
		if !strings.Contains(out, q) {
			t.Errorf("query %q not sent", q)
		}
	}
	if strings.Index(out, queryDA1) < strings.Index(out, queryXTVersion) {
		t.Error("DA1 must be the last query")
	}
}

// AC2: no answer within the timeout yields an all-false message.
func TestCapabilityProbeTimeoutAllFalse(t *testing.T) {
	t.Cleanup(func() { setClusterWidth(true) }) // the probe sets the global cluster width
	_, _, wait := bgRun(t, recorderModel{quitAfter: 2}, WithCapabilityProbe(50*time.Millisecond))
	rec := wait().model.(recorderModel)
	n, c := capMsgs(rec)
	if n != 1 || c.Capabilities != (Capabilities{}) {
		t.Fatalf("got n=%d %#v", n, c)
	}
}

func TestCapabilityProbeOffSendsNothing(t *testing.T) {
	write, read, wait := bgRun(t, recorderModel{quitAfter: 2})
	write("a")
	wait()
	if strings.Contains(string(read()), queryDA1) {
		t.Error("probe queries sent without WithCapabilityProbe")
	}
}

func TestCapabilityReplyParsing(t *testing.T) {
	var c capProbe
	c.on = true
	feed := func(kind byte, data string) (Msg, bool) { return c.observe(ReplyEvent{Kind: kind, Data: data}) }
	feed('[', "?2027;2$y")
	feed('[', "?2026;0$y")
	feed('[', "?7u")
	feed('P', ">|kitty(0.35)")
	feed('_', "Gi=31;OK")
	msg, consumed := feed('[', "?62;c")
	got, ok := msg.(CapabilitiesMsg)
	if !ok || !consumed {
		t.Fatalf("no CapabilitiesMsg: %#v", msg)
	}
	want := Capabilities{GraphemeClusters: true, KittyKeyboard: true, KittyGraphics: true, StyledUnderline: true, XTVersion: "kitty(0.35)"}
	if got.Capabilities != want {
		t.Errorf("got %#v want %#v", got.Capabilities, want)
	}
	if m, _ := feed('[', "?62;c"); m != nil {
		t.Error("second DA1 delivered again")
	}
	if _, consumed := feed('[', "5n"); consumed {
		t.Error("unrelated CSI reply consumed")
	}
	if _, consumed := feed(']', "52;c;x"); consumed {
		t.Error("unrelated OSC consumed")
	}
}

// AC3: mode 2026 unsupported means no synchronized-output sequences.
func TestSyncOutputGatedByProbe(t *testing.T) {
	off := NewProgram(staticModel{})
	if off.syncEnable() == "" || off.syncDisable() == "" {
		t.Error("sync must be emitted when probing is off")
	}
	p := NewProgram(staticModel{}, WithCapabilityProbe(time.Hour))
	if p.syncEnable() == "" {
		t.Error("sync suppressed before the probe finished")
	}
	p.capProbe.resolve(Capabilities{})
	if p.syncEnable() != "" || p.syncDisable() != "" {
		t.Error("sync emitted although 2026 unsupported")
	}
	p2 := NewProgram(staticModel{}, WithCapabilityProbe(time.Hour))
	p2.capProbe.resolve(Capabilities{SyncOutput: true})
	if p2.syncEnable() == "" {
		t.Error("sync suppressed although supported")
	}
}

func TestRenderOmitsSyncWhenUnsupported(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{}, WithOutput(out), WithAltScreen(false), WithCapabilityProbe(time.Hour))
	p.capProbe.resolve(Capabilities{})
	p.render()
	if s := string(read()); strings.Contains(s, "?2026") {
		t.Errorf("frame contains 2026: %q", s)
	}
}

func TestStyledUnderlineGate(t *testing.T) {
	in := "\x1b[4:3;38:2::1:2:3mx\x1b[4:0my"
	p := NewProgram(staticModel{}, WithCapabilityProbe(time.Hour))
	if p.gateUnderlines(in) != in {
		t.Error("changed before probe finished")
	}
	p.capProbe.resolve(Capabilities{})
	if got, want := p.gateUnderlines(in), "\x1b[4;38:2::1:2:3mx\x1b[24my"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if p.Capabilities() != (Capabilities{}) || !p.CapabilitiesKnown() {
		t.Error("Capabilities accessors")
	}
}
