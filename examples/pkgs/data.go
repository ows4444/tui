package main

import (
	"fmt"
	"sort"
	"strings"
)

// pkg is one package of the made-up registry. installed is the version on the
// machine, or "" when it is not installed.
type pkg struct {
	name      string
	latest    string
	installed string
	sizeKB    int
	desc      string
	deps      []string
}

type status int

const (
	available status = iota
	installedOK
	updatable
)

func (p pkg) status() status {
	switch {
	case p.installed == "":
		return available
	case p.installed != p.latest:
		return updatable
	}
	return installedOK
}

func (s status) String() string { return [...]string{"available", "installed", "update"}[s] }

func size(kb int) string {
	if kb >= 1024 {
		return fmt.Sprintf("%.1f MB", float64(kb)/1024)
	}
	return fmt.Sprintf("%d KB", kb)
}

// registry is the list the browser starts with.
func registry() []pkg {
	p := func(name, latest, installed string, kb int, desc string, deps ...string) pkg {
		return pkg{name, latest, installed, kb, desc, deps}
	}
	return []pkg{
		p("argon", "2.3.0", "2.3.0", 180, "Command-line flag parsing with sub-commands and shell completion.", "textwrap"),
		p("bytesize", "1.0.2", "1.0.2", 24, "Parse and print sizes such as 1.5 GB."),
		p("cachet", "0.9.4", "", 310, "An in-memory cache with expiry and a size limit.", "clockwork"),
		p("clockwork", "1.4.1", "1.3.0", 42, "A clock you can set in tests."),
		p("colorize", "3.1.0", "3.1.0", 96, "Terminal colours that follow NO_COLOR and the terminal's depth."),
		p("configure", "2.0.0", "", 120, "Read settings from files, the environment and flags, in that order.", "tomlish", "textwrap"),
		p("crontab", "1.2.5", "", 65, "Parse and evaluate cron schedules.", "clockwork"),
		p("csvkit", "4.0.1", "4.0.1", 140, "Stream large CSV files without loading them."),
		p("difflib", "0.7.0", "", 88, "Line and word diffs with a unified output."),
		p("dotenv", "1.1.0", "1.0.0", 18, "Load a .env file into the environment."),
		p("fmtx", "5.2.0", "5.2.0", 210, "Extra formatting verbs for tables, durations and bytes.", "bytesize"),
		p("gitlite", "0.4.2", "", 1480, "Read a git repository's objects and history without the git binary.", "zlibx"),
		p("glob", "1.0.0", "", 30, "Match paths with ** patterns."),
		p("httpmock", "2.1.3", "", 175, "Record and replay HTTP calls in tests.", "clockwork"),
		p("jsonpath", "1.6.0", "1.5.2", 72, "Query JSON documents with JSONPath expressions."),
		p("logr", "3.0.1", "", 150, "Structured logging with levels and fields.", "colorize"),
		p("markdowner", "2.8.0", "", 460, "A CommonMark parser and renderer.", "textwrap"),
		p("mimetype", "1.3.0", "", 54, "Detect a file's type from its first bytes."),
		p("pathx", "0.12.0", "0.12.0", 36, "Path helpers that behave the same on every system."),
		p("progressbar", "2.2.2", "", 58, "Progress bars with rates and time left.", "colorize"),
		p("retry", "1.0.4", "", 22, "Retry a call with backoff and jitter.", "clockwork"),
		p("semverx", "3.4.0", "3.3.1", 40, "Parse, compare and bump semantic versions."),
		p("shellwords", "1.0.1", "", 14, "Split a command line the way a shell does."),
		p("sqlmini", "0.6.0", "", 920, "A tiny SQL parser for filters and queries.", "tomlish"),
		p("syncx", "1.9.0", "1.9.0", 48, "Worker pools, rate limiters and error groups."),
		p("textwrap", "1.2.0", "1.2.0", 28, "Wrap text to a width, with indentation."),
		p("tomlish", "2.5.0", "", 135, "Read and write TOML."),
		p("uuidx", "2.0.2", "", 33, "Create and parse UUIDs of every version."),
		p("watchfs", "0.8.1", "", 190, "Watch files and directories for changes.", "syncx"),
		p("yamlkit", "3.0.0", "", 540, "Read and write YAML, keeping comments.", "textwrap"),
		p("zlibx", "1.5.0", "", 260, "Compression helpers: gzip, zstd and streaming readers."),
	}
}

func index(ps []pkg, name string) int {
	for i, p := range ps {
		if p.name == name {
			return i
		}
	}
	return -1
}

// dependents are the installed packages that need ps[i].
func dependents(ps []pkg, i int) []string {
	var out []string
	for _, p := range ps {
		if p.installed == "" {
			continue
		}
		for _, d := range p.deps {
			if d == ps[i].name {
				out = append(out, p.name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// installOrder lists what must be installed to get ps[i], dependencies first,
// leaving out what is installed already or is in skip (already queued).
func installOrder(ps []pkg, i int, skip map[int]bool) []int {
	var out []int
	seen := map[int]bool{}
	var visit func(int)
	visit = func(k int) {
		if seen[k] || skip[k] || ps[k].installed != "" {
			return
		}
		seen[k] = true
		for _, d := range ps[k].deps {
			if j := index(ps, d); j >= 0 {
				visit(j)
			}
		}
		out = append(out, k)
	}
	visit(i)
	return out
}

func joinNames(ps []pkg, idx []int) string {
	var names []string
	for _, i := range idx {
		names = append(names, ps[i].name)
	}
	return strings.Join(names, ", ")
}
