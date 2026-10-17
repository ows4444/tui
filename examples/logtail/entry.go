package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type level int

const (
	lTrace level = iota
	lDebug
	lInfo
	lWarn
	lError
	numLevels
)

var levelNames = [numLevels]string{"TRACE", "DEBUG", "INFO", "WARN", "ERROR"}

// parseLevel reads a level word in any case, with the usual aliases. ok is
// false for anything else.
func parseLevel(s string) (level, bool) {
	switch strings.ToLower(strings.Trim(strings.TrimSpace(s), "[]():")) {
	case "trace", "trc":
		return lTrace, true
	case "debug", "dbg":
		return lDebug, true
	case "info", "information", "inf", "notice":
		return lInfo, true
	case "warn", "warning", "wrn":
		return lWarn, true
	case "error", "err", "fatal", "panic", "critical", "crit", "severe":
		return lError, true
	}
	return lInfo, false
}

type kv struct{ k, v string }

// entry is one log line, read apart.
type entry struct {
	id     int
	raw    string
	ts     string
	lvl    level
	hasLvl bool
	msg    string
	fields []kv
	json   bool
}

var (
	// 2026-10-18T09:30:00.123Z, 2026-10-18 09:30:00,123 or 09:30:00
	stamp = regexp.MustCompile(`^\[?(\d{4}-\d{2}-\d{2}[T ])?(\d{2}:\d{2}:\d{2}(?:[.,]\d+)?)(?:Z|[+-]\d{2}:?\d{2})?\]?\s*`)
	word  = regexp.MustCompile(`^(\[?[A-Za-z]{3,8}\]?:?)\s+`)
)

// parseLine reads a log line: a JSON object with level, message and time
// fields, or text that may start with a time and a level word.
func parseLine(id int, line string) entry {
	e := entry{id: id, raw: line, lvl: lInfo}
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
		if parseJSON(&e, t) {
			return e
		}
	}
	rest := t
	if m := stamp.FindStringSubmatch(rest); m != nil {
		e.ts = m[2]
		rest = rest[len(m[0]):]
	}
	if m := word.FindStringSubmatch(rest); m != nil {
		if l, ok := parseLevel(m[1]); ok {
			e.lvl, e.hasLvl = l, true
			rest = rest[len(m[0]):]
		}
	}
	e.msg = rest
	return e
}

// parseJSON fills e from a JSON object, keeping the members in file order. It
// reports false if t is not an object.
func parseJSON(e *entry, t string) bool {
	dec := json.NewDecoder(strings.NewReader(t))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return false
	}
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return false
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return false
		}
		e.fields = append(e.fields, kv{k.(string), scalar(raw)})
	}
	if _, err := dec.Token(); err != nil {
		return false
	}
	e.json = true
	for _, f := range e.fields {
		switch strings.ToLower(f.k) {
		case "level", "lvl", "severity", "loglevel":
			if l, ok := parseLevel(f.v); ok && !e.hasLvl {
				e.lvl, e.hasLvl = l, true
			}
		case "msg", "message", "event":
			if e.msg == "" {
				e.msg = f.v
			}
		case "time", "ts", "timestamp", "@timestamp":
			if e.ts == "" {
				e.ts = clock(f.v)
			}
		}
	}
	return true
}

// scalar shows a JSON value as text: a string without its quotes, anything
// else as written.
func scalar(raw json.RawMessage) string {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "null" // unmarshalling null into a string would give an empty one
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var b bytes.Buffer
	if json.Compact(&b, raw) == nil {
		return b.String()
	}
	return string(raw)
}

// clock cuts a timestamp down to its time of day.
func clock(s string) string {
	if i := strings.IndexAny(s, "T "); i >= 0 && i+1 < len(s) {
		s = s[i+1:]
	}
	if i := strings.IndexAny(s, "Z+"); i > 0 {
		s = s[:i]
	}
	return s
}

// extras are the fields shown after the message on one line: all of them
// except the ones read into the time, level and message.
func (e entry) extras() []kv {
	var out []kv
	usedMsg := false
	for _, f := range e.fields {
		switch strings.ToLower(f.k) {
		case "level", "lvl", "severity", "loglevel", "time", "ts", "timestamp", "@timestamp":
			continue
		case "msg", "message", "event":
			if !usedMsg && f.v == e.msg {
				usedMsg = true
				continue
			}
		}
		out = append(out, f)
	}
	return out
}

// summary is the one line the entry takes: the message and its fields.
func (e entry) summary() string {
	var b strings.Builder
	b.WriteString(e.msg)
	for _, f := range e.extras() {
		fmt.Fprintf(&b, "  %s=%s", f.k, f.v)
	}
	return b.String()
}

// detail is the lines shown when the entry is expanded: every field of a JSON
// entry, one a line, or the raw text of any other.
func (e entry) detail(width int) []string {
	if !e.json {
		return wrap(e.raw, width)
	}
	w := 0
	for _, f := range e.fields {
		w = max(w, len([]rune(f.k)))
	}
	var out []string
	for _, f := range e.fields {
		lines := wrap(f.v, max(10, width-w-2))
		for i, l := range lines {
			key := ""
			if i == 0 {
				key = f.k
			}
			out = append(out, fmt.Sprintf("%-*s  %s", w, key, l))
		}
		if len(lines) == 0 {
			out = append(out, f.k)
		}
	}
	return out
}

// wrap breaks s into lines of at most width runes, at spaces where it can.
func wrap(s string, width int) []string {
	width = max(1, width)
	var out []string
	for _, para := range strings.Split(strings.ReplaceAll(s, "\r", ""), "\n") {
		r := []rune(para)
		for len(r) > width {
			cut := width
			for i := width; i > width/2; i-- {
				if r[i] == ' ' {
					cut = i
					break
				}
			}
			out = append(out, strings.TrimRight(string(r[:cut]), " "))
			r = []rune(strings.TrimLeft(string(r[cut:]), " "))
		}
		out = append(out, string(r))
	}
	return out
}
