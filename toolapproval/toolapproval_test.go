package toolapproval

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/widgets"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// TestHighlightCyclesAmongThreeChoices proves criterion #550: Left/Right/Tab
// retarget the highlighted option among the three choices, the same
// highlight-cycling shape as confirm.Model generalized from 2 to 3 options,
// including wrapping at both ends.
func TestHighlightCyclesAmongThreeChoices(t *testing.T) {
	tests := []struct {
		name  string
		start Choice
		keys  []tui.KeyType
		want  Choice
	}{
		{"Right from Approve goes to Deny", ChoiceApprove, []tui.KeyType{tui.KeyRight}, ChoiceDeny},
		{"Right from Deny goes to AlwaysAllow", ChoiceDeny, []tui.KeyType{tui.KeyRight}, ChoiceAlwaysAllow},
		{"Right wraps from AlwaysAllow to Approve", ChoiceAlwaysAllow, []tui.KeyType{tui.KeyRight}, ChoiceApprove},
		{"Tab behaves like Right", ChoiceApprove, []tui.KeyType{tui.KeyTab}, ChoiceDeny},
		{"Left from Approve wraps to AlwaysAllow", ChoiceApprove, []tui.KeyType{tui.KeyLeft}, ChoiceAlwaysAllow},
		{"Left from AlwaysAllow goes to Deny", ChoiceAlwaysAllow, []tui.KeyType{tui.KeyLeft}, ChoiceDeny},
		{"Left from Deny goes to Approve", ChoiceDeny, []tui.KeyType{tui.KeyLeft}, ChoiceApprove},
		{"full cycle of three Rights returns to start", ChoiceApprove, []tui.KeyType{tui.KeyRight, tui.KeyRight, tui.KeyRight}, ChoiceApprove},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New("rm", "delete a file", RiskLow)
			m.highlighted = tt.start
			for _, kt := range tt.keys {
				m, _ = m.Update(key(kt))
			}
			if m.Highlighted() != tt.want {
				t.Errorf("Highlighted() = %v, want %v", m.Highlighted(), tt.want)
			}
		})
	}
}

// TestEnterResolvesHighlightedChoice proves criterion #551: Enter confirms
// the currently highlighted option, delivering a ResolvedMsg via a Cmd
// identifying which of the three choices was picked.
func TestEnterResolvesHighlightedChoice(t *testing.T) {
	for _, choice := range []Choice{ChoiceApprove, ChoiceDeny, ChoiceAlwaysAllow} {
		m := New("rm", "delete a file", RiskLow)
		m.highlighted = choice

		next, cmd := m.Update(key(tui.KeyEnter))
		if cmd == nil {
			t.Fatalf("Enter with highlighted=%v: Update returned a nil Cmd", choice)
		}
		msg, ok := tui.RunCmd(context.Background(), cmd).(ResolvedMsg)
		if !ok {
			t.Fatalf("Enter with highlighted=%v: Cmd produced %T, want ResolvedMsg", choice, tui.RunCmd(context.Background(), cmd))
		}
		if msg.Choice != choice {
			t.Errorf("ResolvedMsg.Choice = %v, want %v", msg.Choice, choice)
		}
		_ = next
	}
}

// TestViewShowsRiskBadgeDistinctlyColoredPerLevel proves criterion #552:
// View shows a risk badge reflecting the given Risk level, distinctly
// colored per level.
func TestViewShowsRiskBadgeDistinctlyColoredPerLevel(t *testing.T) {
	tests := []struct {
		risk    Risk
		label   string
		variant widgets.Variant
	}{
		{RiskLow, "LOW", widgets.VariantSuccess},
		{RiskMedium, "MEDIUM", widgets.VariantWarning},
		{RiskHigh, "HIGH", widgets.VariantError},
	}

	renders := map[Risk]string{}
	for _, tt := range tests {
		m := New("rm", "delete a file", tt.risk)
		got := m.View()
		renders[tt.risk] = got

		if !strings.Contains(ansi.StripANSI(got), tt.label) {
			t.Errorf("Risk=%v: View() missing label %q, got %q", tt.risk, tt.label, ansi.StripANSI(got))
		}

		wantBadge := widgets.Badge(tt.label, tt.variant, m.Theme)
		if !strings.Contains(got, wantBadge) {
			t.Errorf("Risk=%v: View() missing badge rendered with variant %v", tt.risk, tt.variant)
		}
	}

	// The three risk levels must actually be visually distinct from one
	// another, not just textually different.
	if renders[RiskLow] == renders[RiskMedium] || renders[RiskLow] == renders[RiskHigh] || renders[RiskMedium] == renders[RiskHigh] {
		t.Error("risk badges for different levels rendered identically, want distinct colors")
	}
}

// TestStartReturnsTimeoutCmdGuardedByID proves the id-guarded shape of
// criterion #553's Start half: Start bumps the generation and returns a
// Cmd whose eventual timeoutMsg carries that generation's id.
func TestStartReturnsTimeoutCmdGuardedByID(t *testing.T) {
	m := New("rm", "delete a file", RiskLow)
	m.Timeout = time.Millisecond

	cmd := m.Start()
	if cmd == nil {
		t.Fatal("Start() should return a non-nil Cmd")
	}
	msg, ok := tui.RunCmd(context.Background(), cmd).(timeoutMsg)
	if !ok {
		t.Fatalf("Start()'s Cmd produced %T, want timeoutMsg", tui.RunCmd(context.Background(), cmd))
	}
	if msg.id != m.id {
		t.Errorf("timeoutMsg.id = %d, want %d (current generation)", msg.id, m.id)
	}
}

// TestTimeoutAutoResolvesAsDeny proves the auto-deny half of criterion
// #553: a timeoutMsg whose id matches the current generation resolves the
// prompt as ChoiceDeny.
func TestTimeoutAutoResolvesAsDeny(t *testing.T) {
	m := New("rm", "delete a file", RiskLow)
	m.Timeout = time.Millisecond
	m.highlighted = ChoiceApprove // prove the timeout ignores what's highlighted

	cmd := m.Start()
	msg := tui.RunCmd(context.Background(), cmd).(timeoutMsg)

	next, resolveCmd := m.Update(msg)
	if resolveCmd == nil {
		t.Fatal("a matching timeoutMsg should return a non-nil Cmd")
	}
	resolved, ok := tui.RunCmd(context.Background(), resolveCmd).(ResolvedMsg)
	if !ok {
		t.Fatalf("Cmd produced %T, want ResolvedMsg", tui.RunCmd(context.Background(), resolveCmd))
	}
	if resolved.Choice != ChoiceDeny {
		t.Errorf("ResolvedMsg.Choice = %v, want ChoiceDeny", resolved.Choice)
	}
	_ = next
}

// TestStaleTimeoutIgnoredAfterRestart proves criterion #553's stale-id
// guard: if Start is called again before a previous timer fires, that
// earlier timer's timeoutMsg must not resolve the prompt Start restarted.
func TestStaleTimeoutIgnoredAfterRestart(t *testing.T) {
	m := New("rm", "delete a file", RiskLow)
	m.Timeout = time.Millisecond

	firstCmd := m.Start()
	firstMsg := tui.RunCmd(context.Background(), firstCmd).(timeoutMsg) // id from generation 1

	secondCmd := m.Start() // restarted before the first timer fired; generation 2

	next, cmd := m.Update(firstMsg) // the stale, generation-1 timeout arrives
	if cmd != nil {
		t.Error("a stale timeoutMsg from a previous Start() should not resolve the prompt")
	}

	secondMsg := tui.RunCmd(context.Background(), secondCmd).(timeoutMsg) // id from generation 2
	next, cmd = next.Update(secondMsg)
	if cmd == nil {
		t.Fatal("the current generation's timeoutMsg should resolve the prompt")
	}
	if resolved := tui.RunCmd(context.Background(), cmd).(ResolvedMsg); resolved.Choice != ChoiceDeny {
		t.Errorf("ResolvedMsg.Choice = %v, want ChoiceDeny", resolved.Choice)
	}
	_ = next
}

// TestManualChoicePreventsLaterStaleTimeout proves criterion #553's other
// half explicitly called out in the task: a manual choice made before the
// timeout prevents a later-arriving stale timeoutMsg from also resolving.
func TestManualChoicePreventsLaterStaleTimeout(t *testing.T) {
	m := New("rm", "delete a file", RiskLow)
	m.Timeout = time.Millisecond
	m.highlighted = ChoiceAlwaysAllow

	startCmd := m.Start()
	timeoutMsgFromStart := tui.RunCmd(context.Background(), startCmd).(timeoutMsg) // the timer that would auto-deny

	// The user answers manually (Enter) before that timer's message is
	// ever delivered to Update.
	afterEnter, enterCmd := m.Update(key(tui.KeyEnter))
	if enterCmd == nil {
		t.Fatal("Enter should resolve the prompt")
	}
	resolved := tui.RunCmd(context.Background(), enterCmd).(ResolvedMsg)
	if resolved.Choice != ChoiceAlwaysAllow {
		t.Fatalf("manual Enter resolved as %v, want ChoiceAlwaysAllow", resolved.Choice)
	}

	// Now the stale timer's message finally arrives.
	final, timeoutCmd := afterEnter.Update(timeoutMsgFromStart)
	if timeoutCmd != nil {
		t.Errorf("stale timeoutMsg arriving after a manual choice should not resolve again, got Cmd producing %v", tui.RunCmd(context.Background(), timeoutCmd))
	}
	_ = final
}

// TestNonMatchingMsgIsNoOp proves Update ignores unrelated Msg types.
func TestNonMatchingMsgIsNoOp(t *testing.T) {
	m := New("rm", "delete a file", RiskLow)
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil {
		t.Error("an unrelated Msg should not produce a Cmd")
	}
	if next.Highlighted() != ChoiceApprove {
		t.Error("an unrelated Msg should not change the highlight")
	}
}
