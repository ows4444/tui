package widgets

import "testing"

func TestList(t *testing.T) {
	tests := []struct {
		name   string
		items  []string
		marker Marker
		want   string
	}{
		{
			name:   "bullet marker prefixes every item",
			items:  []string{"apple", "banana"},
			marker: MarkerBullet,
			want:   "• apple\n• banana",
		},
		{
			name:   "number marker prefixes ordinal and dot-space",
			items:  []string{"apple", "banana"},
			marker: MarkerNumber,
			want:   "1. apple\n2. banana",
		},
		{
			name:   "none marker has no prefix",
			items:  []string{"apple", "banana"},
			marker: MarkerNone,
			want:   "apple\nbanana",
		},
		{
			name:   "empty items returns empty string",
			items:  nil,
			marker: MarkerBullet,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := List(tt.items, tt.marker)
			if got != tt.want {
				t.Errorf("List(%v, %v) = %q, want %q", tt.items, tt.marker, got, tt.want)
			}
		})
	}
}

func TestListNumberMarkerAlignsAcrossDigitWidths(t *testing.T) {
	items := make([]string, 10)
	for i := range items {
		items[i] = "item"
	}

	got := List(items, MarkerNumber)

	// Every item's text ("item") must start at the same column, so the
	// single-digit ordinals must be padded to match "10. ".
	want := " 1. item\n" +
		" 2. item\n" +
		" 3. item\n" +
		" 4. item\n" +
		" 5. item\n" +
		" 6. item\n" +
		" 7. item\n" +
		" 8. item\n" +
		" 9. item\n" +
		"10. item"
	if got != want {
		t.Errorf("List(10 items, MarkerNumber) = %q, want %q", got, want)
	}
}

func TestListEmptyDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("List panicked on empty items: %v", r)
		}
	}()

	if got := List([]string{}, MarkerNone); got != "" {
		t.Errorf("List([], MarkerNone) = %q, want empty string", got)
	}
	if got := List(nil, MarkerNumber); got != "" {
		t.Errorf("List(nil, MarkerNumber) = %q, want empty string", got)
	}
}
