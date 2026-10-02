package tui

import (
	"strings"
	"testing"
	"time"
)

type fakeTimers struct {
	now   time.Time
	fires []func()
}

func (f *fakeTimers) install(p *Program) {
	p.announce.Now = func() time.Time { return f.now }
	p.announce.After = func(_ time.Duration, fn func()) { f.fires = append(f.fires, fn) }
}

func TestAnnounceRegionShowsForTTL(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "body"}, WithOutput(out), WithAnnounceRegion(1))
	ft := &fakeTimers{now: time.Unix(1000, 0)}
	ft.install(p)
	p.render()
	p.handleAnnounce(Announce("saved ok")())
	if !strings.Contains(string(read()), "saved ok") {
		t.Fatalf("announcement not drawn: %q", read())
	}
	if len(ft.fires) != 1 {
		t.Fatalf("expiry timers = %d, want 1", len(ft.fires))
	}
	// Before the TTL a stray expiry keeps the text.
	ft.now = ft.now.Add(AnnounceTTL - time.Millisecond)
	before := len(read())
	p.handleAnnounce(announceExpireMsg{})
	if p.announce.Len() != 1 || len(read()) != before {
		t.Error("announcement removed before its 3 s were up")
	}
	ft.now = ft.now.Add(2 * time.Millisecond)
	p.handleAnnounce(announceExpireMsg{})
	if p.announce.Len() != 0 {
		t.Error("announcement not cleared after the TTL")
	}
	if len(read()) == before {
		t.Error("no frame redrawn after expiry")
	}
	if p.lastFrame[len(p.lastFrame)-1] != "" {
		t.Errorf("region not blank: %q", p.lastFrame)
	}
}

func TestAnnounceRegionKeepsLatestRows(t *testing.T) {
	out, _ := captureOutput(t)
	p := NewProgram(staticModel{view: "v"}, WithOutput(out), WithAnnounceRegion(2))
	(&fakeTimers{now: time.Unix(1, 0)}).install(p)
	for _, s := range []string{"one", "two", "three"} {
		p.handleAnnounce(AnnounceWith(s, Polite)())
	}
	got := p.withAnnounceRegion("v")
	if got != "v\ntwo\nthree" {
		t.Errorf("view = %q", got)
	}
}

func TestAnnounceRegionOffUnchanged(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "body"}, WithOutput(out))
	p.render()
	n := len(read())
	p.handleAnnounce(Announce("x")())
	if len(read()) != n {
		t.Error("announce wrote without the option")
	}
	if p.withAnnounceRegion("a") != "a" {
		t.Error("view changed without option")
	}
}

func TestAssertiveNotificationWhenProbeSupports(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "b"}, WithOutput(out), WithCapabilityProbe(time.Hour))
	p.handleAnnounce(AnnounceWith("no probe yet", Assertive)())
	if strings.Contains(string(read()), "\x1b]99;") {
		t.Fatal("notification sent before the probe finished")
	}
	p.capProbe.resolve(Capabilities{Notifications: true})
	p.handleAnnounce(AnnounceWith("disk \x1b[31mfull\x1b[0m\nnow", Assertive)())
	p.handleAnnounce(AnnounceWith("polite", Polite)())
	got := string(read())
	want := "\x1b]99;i=1:u=2;disk full now\x1b\\"
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want it to contain %q", got, want)
	}
	if strings.Count(got, "\x1b]99;") != 1 {
		t.Errorf("polite or unsupported announcement notified: %q", got)
	}
}

func TestAssertiveNoNotificationWhenUnsupported(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "b"}, WithOutput(out), WithCapabilityProbe(time.Hour))
	p.capProbe.resolve(Capabilities{})
	p.handleAnnounce(AnnounceWith("x", Assertive)())
	if len(read()) != 0 {
		t.Errorf("wrote %q", read())
	}
}

func TestNotificationProbeReply(t *testing.T) {
	var c capProbe
	c.on = true
	if _, consumed := c.observe(ReplyEvent{Kind: ']', Data: "99;i=32:p=?;p=title,body:a=focus"}); !consumed {
		t.Error("OSC 99 reply not consumed")
	}
	msg, _ := c.observe(ReplyEvent{Kind: '[', Data: "?62;c"})
	if got := msg.(CapabilitiesMsg); !got.Capabilities.Notifications {
		t.Errorf("Notifications not set: %#v", got)
	}
	if !strings.Contains(capabilityProbeSeqs, queryNotify) {
		t.Error("probe does not send the notification query")
	}
}

type otherMsg struct{}

func TestAssertivePrefixAndDropsPending(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a"}, WithOutput(out), WithAccessible(true))
	// A message is waiting, so polite lines stay queued (pending).
	p.msgs <- otherMsg{}
	p.handleAnnounce(AnnounceWith("p1", Polite)())
	p.handleAnnounce(AnnounceWith("p2", Polite)())
	if len(read()) != 0 {
		t.Fatalf("polite written while more messages waited: %q", read())
	}
	p.handleAnnounce(AnnounceWith("fire", Assertive)())
	if got := string(read()); got != "Alert: fire\r\n" {
		t.Errorf("got %q, want only the alert", got)
	}
	// Queued polite lines are written when a non-announcement message is next.
	p.handleAnnounce(AnnounceWith("p3", Polite)())
	p.handleAnnounce(otherMsg{})
	if got := string(read()); got != "Alert: fire\r\np3\r\n" {
		t.Errorf("got %q", got)
	}
}

func TestLinearizeFullLine(t *testing.T) {
	run := func(opts ...ProgramOption) string {
		out, read := captureOutput(t)
		p := NewProgram(staticModel{view: "hello"}, append(opts, WithOutput(out), WithAccessible(true))...)
		p.render()
		p.model = staticModel{view: "hello world"}
		p.render()
		return string(read())
	}
	if got := run(); got != "hello\r\n world\r\n" {
		t.Errorf("default = %q", got)
	}
	if got := run(WithLinearizeFullLine()); got != "hello\r\nhello world\r\n" {
		t.Errorf("full line = %q", got)
	}
}
