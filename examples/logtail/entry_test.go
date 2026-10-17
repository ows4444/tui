package main

import (
	"strings"
	"testing"
)

func TestParseLevelWordsAndAliases(t *testing.T) {
	for in, want := range map[string]level{
		"error": lError, "ERR": lError, "Fatal": lError, "panic": lError, "[WARN]": lWarn, "warning": lWarn, "WARN:": lWarn,
		"info": lInfo, "notice": lInfo, "debug": lDebug, "DBG": lDebug, "trace": lTrace, " INFO ": lInfo,
	} {
		if got, ok := parseLevel(in); !ok || got != want {
			t.Errorf("parseLevel(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "the", "errors", "level", "x"} {
		if _, ok := parseLevel(bad); ok {
			t.Errorf("parseLevel(%q) accepted", bad)
		}
	}
}

func TestParseTextLines(t *testing.T) {
	for _, c := range []struct {
		line, ts, msg string
		lvl           level
		has           bool
	}{
		{"2026-10-18T09:30:00.123Z ERROR disk full", "09:30:00.123", "disk full", lError, true},
		{"2026-10-18 09:30:00,456 [WARN] slow request", "09:30:00,456", "slow request", lWarn, true},
		{"09:30:00 INFO started", "09:30:00", "started", lInfo, true},
		{"2026-10-18T09:30:00+02:00 debug x=1", "09:30:00", "x=1", lDebug, true},
		{"just some text", "", "just some text", lInfo, false},
		{"09:30:00 something odd happened", "09:30:00", "something odd happened", lInfo, false},
		{"ERROR: no time here", "", "no time here", lError, true},
		{"", "", "", lInfo, false},
	} {
		e := parseLine(1, c.line)
		if e.ts != c.ts || e.msg != c.msg || e.lvl != c.lvl || e.hasLvl != c.has || e.json || e.raw != c.line {
			t.Errorf("parseLine(%q) = %+v", c.line, e)
		}
	}
}

func TestParseJSONLines(t *testing.T) {
	e := parseLine(7, `{"time":"2026-10-18T09:30:00.250Z","level":"warn","msg":"retrying","path":"/api","status":503,"nested":{"a":[1,2]},"ok":true,"none":null}`)
	if !e.json || e.lvl != lWarn || !e.hasLvl || e.msg != "retrying" || e.ts != "09:30:00.250" || e.id != 7 {
		t.Fatalf("e = %+v", e)
	}
	var keys []string
	for _, f := range e.fields {
		keys = append(keys, f.k)
	}
	if strings.Join(keys, ",") != "time,level,msg,path,status,nested,ok,none" {
		t.Fatalf("fields out of order: %v", keys)
	}
	got := map[string]string{}
	for _, f := range e.fields {
		got[f.k] = f.v
	}
	if got["status"] != "503" || got["nested"] != `{"a":[1,2]}` || got["ok"] != "true" || got["none"] != "null" || got["path"] != "/api" {
		t.Fatalf("values = %v", got)
	}
	if s := e.summary(); !strings.HasPrefix(s, "retrying  path=/api  status=503") || strings.Contains(s, "level=") || strings.Contains(s, "time=") {
		t.Fatalf("summary = %q", s)
	}
}

func TestJSONFieldAliases(t *testing.T) {
	for line, want := range map[string]level{
		`{"severity":"ERROR","message":"x"}`: lError, `{"lvl":"debug","msg":"x"}`: lDebug, `{"loglevel":"warning"}`: lWarn,
		`{"msg":"no level"}`: lInfo, `{"level":"loud","msg":"x"}`: lInfo,
	} {
		if e := parseLine(1, line); e.lvl != want || !e.json {
			t.Errorf("%s: lvl = %v json = %v", line, e.lvl, e.json)
		}
	}
	if e := parseLine(1, `{"@timestamp":"2026-10-18 09:30:00Z","event":"boot"}`); e.ts != "09:30:00" || e.msg != "boot" {
		t.Errorf("e = %+v", e)
	}
}

func TestBrokenJSONIsJustText(t *testing.T) {
	for _, line := range []string{`{"a":`, `{"a":1`, `{not json}`, `{"a":1} trailing`, `{"a" 1}`, `{}x`} {
		e := parseLine(1, line)
		if e.json {
			t.Errorf("%q parsed as JSON", line)
		}
		if e.msg != strings.TrimSpace(line) {
			t.Errorf("%q: msg = %q", line, e.msg)
		}
	}
	if e := parseLine(1, `{}`); !e.json || len(e.fields) != 0 {
		t.Errorf("an empty object: %+v", e)
	}
}

func TestDetailListsEveryFieldOrWrapsText(t *testing.T) {
	e := parseLine(1, `{"time":"09:30:00","level":"info","msg":"request","path":"/api/orders"}`)
	d := strings.Join(e.detail(60), "\n")
	for _, want := range []string{"time   09:30:00", "level  info", "msg    request", "path   /api/orders"} {
		if !strings.Contains(d, want) {
			t.Errorf("detail lacks %q:\n%s", want, d)
		}
	}
	long := strings.Repeat("word ", 30)
	for _, l := range parseLine(1, long).detail(40) {
		if len([]rune(l)) > 40 {
			t.Errorf("line of %d runes", len([]rune(l)))
		}
	}
	if got := wrap(strings.Repeat("x", 25), 10); len(got) != 3 || got[0] != strings.Repeat("x", 10) {
		t.Errorf("wrap = %q", got)
	}
	if got := wrap("", 10); len(got) != 1 {
		t.Errorf("wrap of nothing = %q", got)
	}
}

func TestClock(t *testing.T) {
	for in, want := range map[string]string{"2026-10-18T09:30:00.123Z": "09:30:00.123", "2026-10-18 09:30:00+02:00": "09:30:00", "09:30:00": "09:30:00", "": ""} {
		if got := clock(in); got != want {
			t.Errorf("clock(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGeneratorIsDeterministicAndMixed(t *testing.T) {
	a, b := newGenerator(base), newGenerator(base)
	la, lb := a.lines(500), b.lines(500)
	var json, text, errs int
	for i := range la {
		if la[i] != lb[i] {
			t.Fatalf("line %d differs", i)
		}
		e := parseLine(i, la[i])
		if e.json {
			json++
		} else {
			text++
		}
		if e.lvl == lError {
			errs++
		}
		if e.msg == "" {
			t.Fatalf("line %d has no message: %q", i, la[i])
		}
	}
	if json < 100 || text < 50 || errs < 5 || errs > 150 {
		t.Fatalf("json=%d text=%d errors=%d", json, text, errs)
	}
}
