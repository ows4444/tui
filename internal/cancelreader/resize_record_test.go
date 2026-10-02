package cancelreader

import "testing"

func coordRec(typ uint16, x, y int16) inputRecord {
	return inputRecord{eventType: typ, event: [4]uint32{uint32(uint16(x)) | uint32(uint16(y))<<16}}
}

func TestDecodeRecord(t *testing.T) {
	cases := []struct {
		name string
		in   inputRecord
		want consoleRecord
	}{
		{"key down", inputRecord{eventType: eventKey, event: [4]uint32{1}}, consoleRecord{key: true}},
		{"key up", inputRecord{eventType: eventKey}, consoleRecord{}},
		{"resize", coordRec(eventWindowBufferSize, 120, 30), consoleRecord{resize: true, w: 120, h: 30}},
		{"resize large", coordRec(eventWindowBufferSize, 9001, 32767), consoleRecord{resize: true, w: 9001, h: 32767}},
		{"mouse", inputRecord{eventType: 0x0002}, consoleRecord{}},
		{"focus", inputRecord{eventType: 0x0010, event: [4]uint32{1}}, consoleRecord{}},
		{"menu", inputRecord{eventType: 0x0008}, consoleRecord{}},
	}
	for _, c := range cases {
		if got := decodeRecord(c.in); got != c.want {
			t.Errorf("%s: decodeRecord = %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestDecodeRecordsAndHasResize(t *testing.T) {
	buf := []inputRecord{
		{eventType: eventKey, event: [4]uint32{1}},
		coordRec(eventWindowBufferSize, 80, 24),
		coordRec(eventWindowBufferSize, 100, 40),
	}
	recs := decodeRecords(buf, 3)
	if len(recs) != 3 || !recs[0].key {
		t.Fatalf("recs = %+v", recs)
	}
	if w, h, ok := hasResize(recs); !ok || w != 100 || h != 40 {
		t.Fatalf("hasResize = %d,%d,%v, want last size 100x40", w, h, ok)
	}
	if _, _, ok := hasResize(recs[:1]); ok {
		t.Fatal("hasResize true with no resize record")
	}
	if got := decodeRecords(buf, 99); len(got) != 3 {
		t.Fatalf("n beyond buffer: len = %d, want 3", len(got))
	}
	if got := decodeRecords(buf, 0); len(got) != 0 {
		t.Fatalf("n = 0: len = %d", len(got))
	}
}

func TestWaitForKeyNotifiesResize(t *testing.T) {
	c := &fakeConsole{}
	c.push(consoleRecord{resize: true, w: 80, h: 24}, keyRec)
	calls := 0
	if err := waitForKeyNotify(c, func() bool { return false }, func() { calls++ }); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || c.discarded != 1 {
		t.Fatalf("notified %d times, discarded %d; want 1 and 1", calls, c.discarded)
	}
}

func TestWaitForKeyNoResizeNoNotify(t *testing.T) {
	c := &fakeConsole{}
	c.push(otherRec, keyRec)
	if err := waitForKeyNotify(c, func() bool { return false }, func() { t.Error("unexpected notify") }); err != nil {
		t.Fatal(err)
	}
}
