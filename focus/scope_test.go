package focus

import "testing"

func TestPushScopesTabToItsOwnItems(t *testing.T) {
	root := New(3).Set(1)
	s := root.Push(2)
	if s.Len() != 2 || s.Current() != 0 || s.Depth() != 1 || root.Depth() != 0 {
		t.Fatalf("scope len=%d cur=%d depth=%d rootdepth=%d", s.Len(), s.Current(), s.Depth(), root.Depth())
	}
	var seen []int
	for i := 0; i < 5; i++ {
		s, _ = s.Update(tab())
		seen = append(seen, s.Current())
	}
	for _, c := range seen {
		if c < 0 || c > 1 {
			t.Fatalf("Tab left the scope: %v", seen)
		}
	}
	s, _ = s.Update(shiftTab())
	if s.Current() < 0 || s.Current() > 1 {
		t.Fatalf("Shift+Tab left the scope: %d", s.Current())
	}
}

func TestPopRestoresTheFocusedItemAndDisabledSet(t *testing.T) {
	root := New(4).SetDisabled(2, true).Set(3)
	s := root.Push(2).Next()
	back := s.Pop()
	if back.Current() != 3 || back.Enabled(2) || back.Depth() != 0 || back.Len() != 4 {
		t.Fatalf("Pop: cur=%d enabled(2)=%v depth=%d len=%d", back.Current(), back.Enabled(2), back.Depth(), back.Len())
	}
	if root.Pop().Current() != 3 {
		t.Error("Pop on a Ring with no scope must return it unchanged")
	}
}

func TestNestedScopesUnwindInOrder(t *testing.T) {
	r := New(3).Set(2).Push(2).Next().Push(1)
	if r.Depth() != 2 {
		t.Fatalf("depth = %d", r.Depth())
	}
	r = r.Pop()
	if r.Depth() != 1 || r.Current() != 1 {
		t.Errorf("after first Pop: depth=%d cur=%d", r.Depth(), r.Current())
	}
	r = r.Pop()
	if r.Depth() != 0 || r.Current() != 2 {
		t.Errorf("after second Pop: depth=%d cur=%d", r.Depth(), r.Current())
	}
}

func TestPushEmptyScopeAndValueSemantics(t *testing.T) {
	root := New(2)
	if e := root.Push(0); e.Len() != 0 || e.Current() != -1 || e.Pop().Len() != 2 {
		t.Error("empty scope misbehaves")
	}
	_ = root.Push(5).SetDisabled(0, true)
	if root.Len() != 2 || !root.Enabled(0) {
		t.Error("Push leaked into the receiver")
	}
}
