package main

import (
	"errors"
	"strings"
	"testing"
)

const oldAPI = `# Public API of m. Generated.

## m/a (package a)

func Keep()
func Remove()
func Change(x int) string
type T struct {
	A int
}

## m/b (package b)

func Keep()
`

func TestBreakingReportsRemovedAndChanged(t *testing.T) {
	newAPI := strings.Replace(oldAPI, "func Remove()\n", "", 1)
	newAPI = strings.Replace(newAPI, "func Change(x int) string", "func Change(x, y int) string", 1)
	got := breaking(oldAPI, newAPI)
	want := []string{"m/a (package a): func Change(x int) string", "m/a (package a): func Remove()"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBreakingIgnoresAdditions(t *testing.T) {
	newAPI := strings.Replace(oldAPI, "\tA int\n", "\tA int\n\tB int\n", 1) + "func Brand() new\n"
	newAPI = strings.Replace(newAPI, "## m/b (package b)\n", "## m/b (package b)\n\nfunc Extra()\n", 1)
	if got := breaking(oldAPI, newAPI); len(got) != 0 {
		t.Errorf("additions reported as breaking: %q", got)
	}
}

func TestBreakingKeysLinesByPackage(t *testing.T) {
	// The same line existing in another package must not hide a removal.
	newAPI := strings.Replace(oldAPI, "## m/b (package b)\n\nfunc Keep()\n", "## m/b (package b)\n\n", 1)
	got := breaking(oldAPI, newAPI)
	if len(got) != 1 || !strings.HasPrefix(got[0], "m/b") {
		t.Errorf("got %q, want the removal in m/b only", got)
	}
}

func fakeGit(tag string, files map[string]string) gitRunner {
	return func(args ...string) (string, error) {
		switch args[0] {
		case "describe":
			if tag == "" {
				return "", errors.New("fatal: No names found")
			}
			return tag + "\n", nil
		case "show":
			if body, ok := files[args[1]]; ok {
				return body, nil
			}
			return "", errors.New("not found")
		}
		return "", errors.New("unexpected git call")
	}
}

func TestSinceTagFallsBackWhenThereIsNoTag(t *testing.T) {
	tag, changes, adds, err := sinceTag(fakeGit("", nil), oldAPI, "")
	if tag != "" || changes != nil || adds != nil || err != nil {
		t.Errorf("got %q, %v, %v, %v; want the plain check", tag, changes, adds, err)
	}
}

func TestSinceTagComparesWithTheTaggedListing(t *testing.T) {
	git := fakeGit("v0.1.0", map[string]string{"v0.1.0:" + listingFile: oldAPI})
	tag, changes, _, err := sinceTag(git, strings.Replace(oldAPI, "func Remove()\n", "", 1), "")
	if err != nil || tag != "v0.1.0" || len(changes) != 1 {
		t.Fatalf("got %q, %v, %v", tag, changes, err)
	}
	if _, changes, _, _ := sinceTag(git, oldAPI, ""); len(changes) != 0 {
		t.Errorf("unchanged API reported %v", changes)
	}
}

func TestSinceTagReportsAMissingListingAtTheTag(t *testing.T) {
	if _, _, _, err := sinceTag(fakeGit("v0.1.0", nil), oldAPI, ""); err == nil {
		t.Error("want an error when the tag has no api.txt")
	}
}

// Criterion #38: only additions: nothing breaking, and the added lines listed.
func TestSinceTagListsAdditionsAndReportsNothingBreaking(t *testing.T) {
	git := fakeGit("v0.1.0", map[string]string{"v0.1.0:" + listingFile: oldAPI})
	cur := strings.Replace(oldAPI, "\tA int\n", "\tA int\n\tB int\n", 1) + "\n## m/c (package c)\n\nfunc New()\n"
	_, changes, adds, err := sinceTag(git, cur, "")
	if err != nil || len(changes) != 0 {
		t.Fatalf("got changes %v, err %v; want none", changes, err)
	}
	want := "m/a (package a): \tB int|m/c (package c): func New()"
	if strings.Join(adds, "|") != want {
		t.Errorf("added = %q, want %q", adds, want)
	}
}

// Criterion #39: with no tag, a given ref is the base.
func TestSinceTagUsesAGivenRefWhenThereIsNoTag(t *testing.T) {
	git := fakeGit("", map[string]string{"main:" + listingFile: oldAPI})
	tag, changes, _, err := sinceTag(git, strings.Replace(oldAPI, "func Remove()\n", "", 1), "main")
	if err != nil || tag != "main" || len(changes) != 1 {
		t.Fatalf("got %q, %v, %v; want the removal against main", tag, changes, err)
	}
	// A ref wins over an existing tag.
	git = fakeGit("v0.1.0", map[string]string{"v0.1.0:" + listingFile: oldAPI, "dev:" + listingFile: "# empty\n"})
	if tag, _, _, _ := sinceTag(git, oldAPI, "dev"); tag != "dev" {
		t.Errorf("tag = %q, want the explicit ref", tag)
	}
	if _, _, _, err := sinceTag(fakeGit("", nil), oldAPI, "nope"); err == nil {
		t.Error("an unknown ref must be an error")
	}
}
