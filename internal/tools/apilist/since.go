package main

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// entries returns the declaration lines of a listing, each keyed by the
// package section it sits in, so the same line in two packages is two entries.
func entries(listing string) map[string]bool {
	out := map[string]bool{}
	pkg := ""
	for _, l := range strings.Split(listing, "\n") {
		switch {
		case strings.HasPrefix(l, "## "):
			pkg = l
		case l == "" || strings.HasPrefix(l, "#"):
		default:
			out[pkg+"\n"+l] = true
		}
	}
	return out
}

// breaking returns the lines of the old listing that the new one no longer has:
// an exported symbol that was removed, or whose declaration changed. Lines only
// present in the new listing (new symbols, new struct fields) are additions and
// are not reported. It cannot see a change that only adds lines but still
// breaks callers, such as a new method on an exported interface.
func breaking(oldListing, newListing string) []string { return missing(oldListing, newListing) }

// added returns the lines of the new listing that the old one does not have:
// new symbols and new struct fields, in the same form as breaking.
func added(oldListing, newListing string) []string { return missing(newListing, oldListing) }

// missing returns the entries of from that to lacks, as "package: line", sorted.
func missing(from, to string) []string {
	have := entries(to)
	var out []string
	for k := range entries(from) {
		if have[k] {
			continue
		}
		pkg, line, _ := strings.Cut(k, "\n")
		out = append(out, fmt.Sprintf("%s: %s", strings.TrimPrefix(pkg, "## "), line))
	}
	sort.Strings(out)
	return out
}

// gitRunner runs git with args and returns its standard output.
type gitRunner func(args ...string) (string, error)

func runGit(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output() // #nosec G204 -- fixed git subcommands, dev tool
	return string(out), err
}

// latestTag returns the most recent tag reachable from HEAD, and false if
// there is none.
func latestTag(git gitRunner) (string, bool) {
	out, err := git("describe", "--tags", "--abbrev=0")
	tag := strings.TrimSpace(out)
	return tag, err == nil && tag != ""
}

// sinceTag compares the current listing with the api.txt committed at ref, or
// at the latest tag when ref is empty. It returns the revision used ("" when
// ref is empty and there is no tag, in which case nothing is compared and the
// caller keeps only the plain check), the breaking changes found and the
// additions.
func sinceTag(git gitRunner, current, ref string) (tag string, changes, adds []string, err error) {
	tag = ref
	if tag == "" {
		var ok bool
		if tag, ok = latestTag(git); !ok {
			return "", nil, nil, nil
		}
	}
	old, err := git("show", tag+":"+listingFile)
	if err != nil {
		return tag, nil, nil, fmt.Errorf("reading %s at %s: %w", listingFile, tag, err)
	}
	return tag, breaking(old, current), added(old, current), nil
}
