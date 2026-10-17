package main

import "testing"

func TestEval(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"1+2", "3"},
		{"2+3*4", "14"},
		{"(2+3)*4", "20"},
		{"10/4", "2.5"},
		{"7%3", "1"},
		{"2^3^2", "512"},
		{"-3+5", "2"},
		{"--4", "4"},
		{"2*-3", "-6"},
		{"1.5+1.5", "3"},
		{"0.1+0.2", "0.3"},
		{"8÷2×3−1", "11"},
		{" 4 + 4 ", "8"},
		{"((1))", "1"},
	} {
		got, err := eval(tc.in)
		if err != nil || format(got) != tc.want {
			t.Errorf("eval(%q) = %s, %v; want %s", tc.in, format(got), err, tc.want)
		}
	}
}

func TestEvalErrors(t *testing.T) {
	for in, want := range map[string]error{
		"":       errSyntax,
		"1+":     errSyntax,
		"(1+2":   errSyntax,
		"1 2":    errSyntax,
		"*3":     errSyntax,
		"1/0":    errDivZero,
		"5%0":    errDivZero,
		"1..2":   errNotValid,
		"10^999": errNotValid,
	} {
		if _, err := eval(in); err != want {
			t.Errorf("eval(%q) error = %v, want %v", in, err, want)
		}
	}
}

func TestFormat(t *testing.T) {
	for v, want := range map[float64]string{0: "0", 1: "1", -2.5: "-2.5", 1e21: "1e+21", 1.0 / 3: "0.333333333333"} {
		if got := format(v); got != want {
			t.Errorf("format(%v) = %q, want %q", v, got, want)
		}
	}
}
