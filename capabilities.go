package tui

import (
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/capprobe"
)

// Capabilities is what the terminal reported to the startup probe enabled by
// WithCapabilityProbe. The zero value means "nothing was confirmed", which is
// also what a terminal that never answers yields.
type Capabilities struct {
	// SyncOutput: DECRQM reported mode 2026 (synchronized output) as
	// supported.
	SyncOutput bool
	// GraphemeClusters: DECRQM reported mode 2027 as supported.
	GraphemeClusters bool
	// KittyKeyboard: the terminal answered the kitty keyboard flags query.
	KittyKeyboard bool
	// KittyGraphics: the terminal answered the kitty graphics query with OK.
	KittyGraphics bool
	// Sixel: the terminal's DA1 reply listed feature 4, Sixel graphics. Prefer
	// KittyGraphics when both are set.
	Sixel bool
	// InlineImages: the terminal is believed to draw iTerm2's inline images
	// (OSC 1337 File), which take a PNG as it is. It is inferred, not
	// queried: true when its XTVERSION names iTerm2 or WezTerm. Prefer
	// KittyGraphics when both are set, and this over Sixel.
	InlineImages bool
	// StyledUnderline: extended underlines (SGR 4:n) are believed to render.
	// It is inferred, not queried: true when the terminal spoke a kitty
	// protocol or its XTVERSION names a terminal known to support them.
	StyledUnderline bool
	// Notifications: the terminal answered the kitty desktop-notification
	// query (OSC 99 with p=?), so Assertive announcements may also be sent as
	// notifications (see WithAnnounceRegion).
	Notifications bool
	// XTVersion is the XTVERSION reply ("kitty(0.35.2)"), or "".
	XTVersion string
}

// CapabilitiesMsg is delivered to Update exactly once when the probe
// finishes: on the DA1 sentinel reply, or on timeout with every capability
// false.
type CapabilitiesMsg struct {
	Capabilities Capabilities
}

// DefaultCapabilityProbeTimeout is used when WithCapabilityProbe is given a
// non-positive timeout.
const DefaultCapabilityProbeTimeout = 500 * time.Millisecond

// Probe queries, defined with the parser that reads their replies.
const (
	queryDECRQM2026     = capprobe.QueryDECRQM2026
	queryDECRQM2027     = capprobe.QueryDECRQM2027
	queryKittyKeyboard  = capprobe.QueryKittyKeyboard
	queryKittyGraphics  = capprobe.QueryKittyGraphics
	queryXTVersion      = capprobe.QueryXTVersion
	queryNotify         = capprobe.QueryNotify
	queryDA1            = capprobe.QueryDA1
	capabilityProbeSeqs = capprobe.Queries
)

// WithCapabilityProbe makes Run send, at startup, DECRQM for modes 2026 and
// 2027, the kitty keyboard and kitty graphics queries, the OSC 99 desktop
// notification query and XTVERSION, followed
// by DA1 as a sentinel. The first DA1 reply ends the probe: Update receives
// one CapabilitiesMsg and Program.Capabilities reports the result. If DA1 is
// not answered within timeout (non-positive: DefaultCapabilityProbeTimeout)
// the message carries every capability false. While probing is on, mode 2026
// is only used, and styled underlines are only emitted, when the terminal
// reported support. When the terminal reports mode 2027 as settable the probe
// also sends ESC[?2027h (reset on exit) and turns cluster widths on; otherwise
// widths are per codepoint. Either way the answer becomes Program.Measurer
// and is delivered to Update as a second ResizeMsg carrying it; the
// process-wide ansi.SetClusterWidth setting is left alone. Off by default;
// ignored under WithAccessible.
func WithCapabilityProbe(timeout time.Duration) ProgramOption {
	return func(p *Program) {
		if timeout <= 0 {
			timeout = DefaultCapabilityProbeTimeout
		}
		p.capProbe.on, p.capProbe.timeout = true, timeout
	}
}

// Capabilities returns the probe result. Before the probe has finished (or
// when WithCapabilityProbe is off) it is the zero value; use ok to tell
// "nothing supported" from "not known yet".
func (p *Program) Capabilities() Capabilities {
	c, _ := p.capProbe.result()
	return c
}

// CapabilitiesKnown reports whether the probe has finished.
func (p *Program) CapabilitiesKnown() bool {
	_, ok := p.capProbe.result()
	return ok
}

type capProbe struct {
	on      bool
	timeout time.Duration

	mu       sync.Mutex
	parser   capprobe.Parser // gathers the replies; guarded by mu
	resolved bool
	final    Capabilities
}

func (c *capProbe) active() bool { return c.on }

func (c *capProbe) result() (Capabilities, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.final, c.resolved
}

// resolve records the final result once; it reports whether this call won.
func (c *capProbe) resolve(caps Capabilities) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resolved {
		return false
	}
	c.resolved, c.final = true, caps
	return true
}

// observe feeds one input event to the probe. consumed means the event was a
// probe reply and must not reach Update; extra, when non-nil, is the
// CapabilitiesMsg to deliver instead.
func (c *capProbe) observe(ev any) (extra Msg, consumed bool) {
	if !c.on {
		return nil, false
	}
	r, ok := ev.(ReplyEvent)
	if !ok {
		return nil, false
	}
	c.mu.Lock()
	consumed, done, res := c.parser.Observe(r.Kind, r.Data)
	c.mu.Unlock()
	if done {
		caps := Capabilities(res)
		if c.resolve(caps) {
			return CapabilitiesMsg{Capabilities: caps}, true
		}
	}
	return nil, consumed
}

// startCapabilityProbe writes the queries and arms the timeout. It runs on
// the loop goroutine, after the input reader is started.
func (p *Program) startCapabilityProbe(done <-chan struct{}) {
	if !p.capProbe.on || p.accessible {
		return
	}
	p.write(capabilityProbeSeqs)
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		t := time.NewTimer(p.capProbe.timeout)
		defer t.Stop()
		select {
		case <-t.C:
			if p.capProbe.resolve(Capabilities{}) {
				p.applyCapabilities(Capabilities{})
				select {
				case p.msgs <- CapabilitiesMsg{}:
				case <-done:
				}
			}
		case <-done:
		}
	}()
}

// Mode 2027 (grapheme cluster width) set and reset.
const (
	clusterModeEnable  = capprobe.ClusterModeEnable
	clusterModeDisable = capprobe.ClusterModeDisable
)

// applyCapabilities acts on the finished probe: when the terminal reported
// mode 2027 as settable it is enabled and cluster widths are used; otherwise
// widths are measured per codepoint. The answer is this Program's Measurer, not
// a process-wide setting, so other Programs are unaffected; the loop tells the
// model with a second ResizeMsg. The mode is reset on restore.
func (p *Program) applyCapabilities(c Capabilities) {
	m := ansi.ClusterMeasurer(c.GraphemeClusters)
	p.measurer.Store(&m)
	if !c.GraphemeClusters {
		return
	}
	p.outMu.Lock()
	defer p.outMu.Unlock()
	if p.outClosed {
		return
	}
	p.clusterMode = true
	_, _ = io.WriteString(p.output, clusterModeEnable)
}

// probeSaysNo reports whether probing is on, finished, and given ok false.
func (p *Program) probeSaysNo(ok func(Capabilities) bool) bool {
	if !p.capProbe.on {
		return false
	}
	c, known := p.capProbe.result()
	return known && !ok(c)
}

// syncEnable and syncDisable are the mode 2026 frame brackets, or "" once a
// finished probe found no support.
func (p *Program) syncEnable() string {
	if p.probeSaysNo(func(c Capabilities) bool { return c.SyncOutput }) {
		return ""
	}
	return ansi.SyncOutputEnable
}

func (p *Program) syncDisable() string {
	if p.probeSaysNo(func(c Capabilities) bool { return c.SyncOutput }) {
		return ""
	}
	return ansi.SyncOutputDisable
}

// gateUnderlines flattens extended underlines (4:n) to plain underline when a
// finished probe found no styled-underline support.
func (p *Program) gateUnderlines(s string) string {
	if !strings.Contains(s, ":") || !p.probeSaysNo(func(c Capabilities) bool { return c.StyledUnderline }) {
		return s
	}
	return flattenStyledUnderlines(s)
}

// flattenStyledUnderlines rewrites SGR sub-parameters "4:0" to 24 and
// "4:1".."4:5" to 4, leaving everything else (including 38:/48:/58: colour
// groups) untouched.
func flattenStyledUnderlines(s string) string {
	if !strings.Contains(s, "\x1b[") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != 0x1b || i+1 >= len(s) || s[i+1] != '[' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i + 2
		for j < len(s) && (s[j] < 0x40 || s[j] > 0x7E) {
			j++
		}
		if j >= len(s) {
			b.WriteString(s[i:])
			break
		}
		if s[j] == 'm' && strings.Contains(s[i+2:j], "4:") {
			toks := strings.Split(s[i+2:j], ";")
			for k, t := range toks {
				switch t {
				case "4:0":
					toks[k] = "24"
				case "4:1", "4:2", "4:3", "4:4", "4:5":
					toks[k] = "4"
				}
			}
			b.WriteString("\x1b[" + strings.Join(toks, ";") + "m")
		} else {
			b.WriteString(s[i : j+1])
		}
		i = j + 1
	}
	return b.String()
}

func (p *Program) clusterModeOn() bool {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	return p.clusterMode
}
