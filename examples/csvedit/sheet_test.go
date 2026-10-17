package main

import (
	"reflect"
	"testing"
)

func two() sheet {
	s, _ := parseCSV("name,n\nb,10\na,9\nc,\n")
	return s
}

func TestParseAndEncodeRoundTrip(t *testing.T) {
	text := "a,b\n\"x, y\",\"line\nbreak\"\n\"q\"\"uote\",z\n"
	s, err := parseCSV(text)
	if err != nil {
		t.Fatal(err)
	}
	if s.rows[0][0] != "x, y" || s.rows[0][1] != "line\nbreak" || s.rows[1][0] != `q"uote` {
		t.Fatalf("rows = %q", s.rows)
	}
	again, _ := parseCSV(s.encode())
	if !reflect.DeepEqual(s, again) {
		t.Fatalf("round trip changed the sheet:\n%v\n%v", s, again)
	}
}

func TestRaggedRowsAreSquared(t *testing.T) {
	s, err := parseCSV("a,b,c\n1\n1,2,3,4\n")
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range s.rows {
		if len(r) != 3 {
			t.Errorf("row %d has %d cells, want 3", i, len(r))
		}
	}
	if s.rows[0][1] != "" || s.rows[1][2] != "3" {
		t.Fatalf("rows = %q", s.rows)
	}
}

func TestParseErrors(t *testing.T) {
	if _, err := parseCSV(""); err == nil {
		t.Error("an empty file loaded")
	}
	if _, err := parseCSV("a,\"b\n"); err == nil {
		t.Error("an unterminated quote loaded")
	}
}

func TestMethodsDoNotChangeTheReceiver(t *testing.T) {
	s := two()
	before := s.encode()
	_ = s.set(0, 0, "x")
	_ = s.insertRow(1)
	_ = s.deleteRow(0)
	_ = s.insertCol(1, "new")
	_, _ = s.deleteCol(0)
	_ = s.sorted(0, true)
	if s.encode() != before {
		t.Fatalf("the sheet changed:\n%s", s.encode())
	}
}

func TestRowAndColumnEditing(t *testing.T) {
	s := two().insertRow(1)
	if len(s.rows) != 4 || s.rows[1][0] != "" || s.rows[2][0] != "a" {
		t.Fatalf("insertRow: %q", s.rows)
	}
	s = s.deleteRow(1)
	if len(s.rows) != 3 || s.rows[1][0] != "a" {
		t.Fatalf("deleteRow: %q", s.rows)
	}
	s = s.insertCol(1, "mid")
	if !reflect.DeepEqual(s.headers, []string{"name", "mid", "n"}) || len(s.rows[0]) != 3 || s.rows[0][2] != "10" {
		t.Fatalf("insertCol: %v %q", s.headers, s.rows)
	}
	s, ok := s.deleteCol(1)
	if !ok || s.cols() != 2 {
		t.Fatalf("deleteCol: ok=%v cols=%d", ok, s.cols())
	}
	one := sheet{headers: []string{"x"}, rows: [][]string{{"1"}}}
	if _, ok := one.deleteCol(0); ok {
		t.Fatal("the last column was deleted")
	}
	if got := s.insertRow(len(s.rows)); len(got.rows) != 4 {
		t.Fatal("insertRow at the end failed")
	}
}

func TestSetHeaderAndCell(t *testing.T) {
	s := two().set(-1, 1, "count").set(0, 0, "z")
	if s.headers[1] != "count" || s.rows[0][0] != "z" || s.cell(-1, 1) != "count" || s.cell(0, 0) != "z" {
		t.Fatalf("%v %q", s.headers, s.rows)
	}
}

func TestSortNumbersAsNumbersAndEmptyLast(t *testing.T) {
	s := two().sorted(1, false)
	if got := []string{s.rows[0][1], s.rows[1][1], s.rows[2][1]}; !reflect.DeepEqual(got, []string{"9", "10", ""}) {
		t.Fatalf("ascending: %q", got)
	}
	s = two().sorted(1, true)
	if got := []string{s.rows[0][1], s.rows[1][1], s.rows[2][1]}; !reflect.DeepEqual(got, []string{"10", "9", ""}) {
		t.Fatalf("descending: %q", got)
	}
}

func TestSortTextIgnoresCaseAndIsStable(t *testing.T) {
	s, _ := parseCSV("k,v\nb,1\nA,2\na,3\nB,4\n")
	s = s.sorted(0, false)
	got := ""
	for _, r := range s.rows {
		got += r[1]
	}
	if got != "2314" {
		t.Fatalf("order %s, want 2314 (A and a equal, keeping their order)", got)
	}
}

func TestNumericDetection(t *testing.T) {
	s, _ := parseCSV("a,b,c\n1,x,\n2.5,2,\n,3,\n")
	if !s.numeric(0) || s.numeric(1) || s.numeric(2) {
		t.Fatalf("numeric = %v %v %v", s.numeric(0), s.numeric(1), s.numeric(2))
	}
}

func TestColName(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA"} {
		if got := colName(i); got != want {
			t.Errorf("colName(%d) = %q, want %q", i, got, want)
		}
	}
}
