package main

import (
	"strings"
	"testing"
)

func mustParse(t testing.TB, s string) *node {
	t.Helper()
	n, err := parse(strings.NewReader(s))
	if err != nil {
		t.Fatalf("parse(%q): %v", s, err)
	}
	return n
}

func TestParseKeepsOrderAndNumberText(t *testing.T) {
	n := mustParse(t, `{"z":1,"a":1.50,"m":[true,null,"x"],"big":12345678901234567890}`)
	var keys []string
	for _, c := range n.children {
		keys = append(keys, c.key)
	}
	if strings.Join(keys, ",") != "z,a,m,big" {
		t.Fatalf("keys = %v", keys)
	}
	if n.children[1].text != "1.50" || n.children[3].text != "12345678901234567890" {
		t.Fatalf("numbers = %q %q", n.children[1].text, n.children[3].text)
	}
	m := n.children[2]
	if m.kind != array || m.children[0].kind != boolean || m.children[1].kind != null || m.children[2].kind != str {
		t.Fatalf("array kinds wrong: %+v", m.children)
	}
	if n.count() != 8 {
		t.Fatalf("count = %d, want 8", n.count())
	}
}

func TestParseScalarsAndEmpties(t *testing.T) {
	for in, k := range map[string]kind{`"hi"`: str, `42`: num, `true`: boolean, `null`: null, `{}`: object, `[]`: array} {
		if got := mustParse(t, in); got.kind != k {
			t.Errorf("%s parsed as kind %d, want %d", in, got.kind, k)
		}
	}
}

func TestParseErrors(t *testing.T) {
	for _, bad := range []string{``, `{`, `{"a":}`, `[1,]`, `{"a":1} x`, `{"a":1}{"b":2}`, `nope`, `{1:2}`} {
		if _, err := parse(strings.NewReader(bad)); err == nil {
			t.Errorf("parse(%q) succeeded", bad)
		}
	}
	deep := strings.Repeat("[", maxDepth+5) + strings.Repeat("]", maxDepth+5)
	if _, err := parse(strings.NewReader(deep)); err == nil || !strings.Contains(err.Error(), "deep") {
		t.Errorf("a %d-deep document: err = %v", maxDepth+5, err)
	}
}

func TestPaths(t *testing.T) {
	n := mustParse(t, `{"users":[{"name":"a","first name":"b","a.b":1,"q\"":2}],"9":3}`)
	cases := map[string]string{
		"$":                        "$",
		"$.users":                  "users",
		"$.users[0]":               "",
		"$.users[0].name":          "",
		`$.users[0]["first name"]`: "",
		`$.users[0]["a.b"]`:        "",
		`$.users[0]["q\""]`:        "",
		`$["9"]`:                   "",
	}
	got := map[string]bool{}
	var walk func(*node)
	walk = func(x *node) {
		got[x.path()] = true
		for _, c := range x.children {
			walk(c)
		}
	}
	walk(n)
	for p := range cases {
		if !got[p] {
			t.Errorf("no node has the path %s; have %v", p, got)
		}
	}
}

func TestScalarTextAndSummary(t *testing.T) {
	n := mustParse(t, `{"s":"a \"b\" <&>","n":0.10,"o":{"x":1},"e":[],"one":[1]}`)
	if got := n.children[0].scalarText(); got != `"a \"b\" <&>"` {
		t.Errorf("string shown as %s", got)
	}
	if n.children[1].scalarText() != "0.10" {
		t.Error("a number lost its text")
	}
	for i, want := range []string{"", "", "{1 key}", "[0 items]", "[1 item]"} {
		if want != "" && n.children[i].summary() != want {
			t.Errorf("summary %d = %s, want %s", i, n.children[i].summary(), want)
		}
	}
	if n.summary() != "{5 keys}" {
		t.Errorf("root summary = %s", n.summary())
	}
}

func TestPrettyAndCopyText(t *testing.T) {
	n := mustParse(t, `{"a":[1,{"b":null}],"c":"x","d":{}}`)
	want := "{\n  \"a\": [\n    1,\n    {\n      \"b\": null\n    }\n  ],\n  \"c\": \"x\",\n  \"d\": {}\n}"
	if got := n.pretty(1 << 10); got != want {
		t.Errorf("pretty =\n%s\nwant\n%s", got, want)
	}
	if n.children[1].copyText() != "x" {
		t.Errorf("a string copies as %q, want it unquoted", n.children[1].copyText())
	}
	if n.children[0].children[0].copyText() != "1" {
		t.Error("a number copies as something else")
	}
	long := mustParse(t, "["+strings.Repeat(`"abcdefghij",`, 100)+`"z"]`)
	if got := long.pretty(100); len(got) > 110 || !strings.HasSuffix(got, "…") {
		t.Errorf("pretty did not stop at the limit: %d bytes, ends %q", len(got), got[len(got)-3:])
	}
}

func TestMatchesSearchKeysAndValuesInOrder(t *testing.T) {
	n := mustParse(t, `{"Name":"x","list":[{"name":"y"},{"other":"NAMEplate"}],"arr":["name"]}`)
	got := n.matches("name")
	var paths []string
	for _, h := range got {
		paths = append(paths, h.path())
	}
	want := "$.Name,$.list[0].name,$.list[1].other,$.arr[0]"
	if strings.Join(paths, ",") != want {
		t.Fatalf("matches = %v, want %s", paths, want)
	}
	if n.matches("") != nil || len(n.matches("zzz")) != 0 {
		t.Error("empty or unknown query matched")
	}
	// A container's key matches, but its summary text does not.
	if m := mustParse(t, `{"keys":{"a":1}}`).matches("key"); len(m) != 1 || m[0].key != "keys" {
		t.Errorf("container key match = %v", m)
	}
}

func TestIndexKeysAreNotSearched(t *testing.T) {
	if m := mustParse(t, `[10,20]`).matches("1"); len(m) != 1 || m[0].text != "10" {
		t.Fatalf("matches = %v", m)
	}
	if m := mustParse(t, `["a","b","c"]`).matches("2"); len(m) != 0 {
		t.Fatal("an array index matched the query")
	}
}
