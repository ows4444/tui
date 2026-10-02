package chart

import "testing"

func TestSparklineEmpty(t *testing.T) {
	if got := Sparkline(nil); got != "" {
		t.Errorf("Sparkline(nil) = %q, want empty", got)
	}
}

func TestSparklineFlatUsesMiddleBar(t *testing.T) {
	tests := [][]float64{
		{5},
		{1, 1, 1},
		{0, 0},
	}
	for _, values := range tests {
		got := Sparkline(values)
		want := string(sparkBars[len(sparkBars)/2])
		for range values[1:] {
			want += string(sparkBars[len(sparkBars)/2])
		}
		if got != want {
			t.Errorf("Sparkline(%v) = %q, want %q (flat -> uniform middle bar, not full)", values, got, want)
		}
	}
}

func TestSparklineFullRange(t *testing.T) {
	// Values 0..7 against an 8-level scale land exactly on each bar in
	// order, the strongest test of the rounding math across the range.
	got := Sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7})
	want := string(sparkBars)
	if got != want {
		t.Errorf("Sparkline(0..7) = %q, want %q", got, want)
	}
}

func TestSparklineUnsortedInput(t *testing.T) {
	got := Sparkline([]float64{5, 1, 9, 3})
	want := string([]rune{sparkBars[4], sparkBars[0], sparkBars[7], sparkBars[2]})
	if got != want {
		t.Errorf("Sparkline([5,1,9,3]) = %q, want %q", got, want)
	}
}

func TestSparklineNegativeValues(t *testing.T) {
	got := Sparkline([]float64{-10, 0, 10})
	want := string([]rune{sparkBars[0], sparkBars[4], sparkBars[7]})
	if got != want {
		t.Errorf("Sparkline([-10,0,10]) = %q, want %q", got, want)
	}
}
