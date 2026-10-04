package faces

import (
	"fmt"
	"testing"
)

// The same name always gets the same face; case and surrounding space do
// not matter, and the face is the one at IndexFor.
func TestForIsStable(t *testing.T) {
	for _, name := range []string{"ada", "linus", "", "Team Rocket 3", "\U0001F98A", "a much longer name with spaces in it"} {
		i := IndexFor(name)
		if i < 0 || i >= Count() {
			t.Fatalf("IndexFor(%q) = %d, outside the catalog", name, i)
		}
		if again := IndexFor(name); again != i {
			t.Errorf("IndexFor(%q) gave %d and then %d", name, i, again)
		}
		if For(name).Name != Get(i).Name {
			t.Errorf("For(%q) is not the face at IndexFor", name)
		}
	}
	if IndexFor("Ada") != IndexFor("  ada ") {
		t.Error("case and surrounding space changed the face")
	}
	// The mapping is fixed: these are the faces these names have in the
	// 50-face catalog, and a change to the hash would move them.
	for name, want := range map[string]string{"ada": "Chip", "linus": "Quasar", "": "Drift", "Team Rocket 3": "Jitter"} {
		if got := For(name).Name; got != want {
			t.Errorf("For(%q) = %s, want %s", name, got, want)
		}
	}
}

// A thousand names reach every face in the catalog, and none is given to far
// more names than its share.
func TestForCoversTheCatalog(t *testing.T) {
	seen := make([]int, Count())
	for i := 0; i < 1000; i++ {
		seen[IndexFor(fmt.Sprintf("user-%d", i))]++
	}
	share := 1000 / Count()
	for i, n := range seen {
		if n == 0 {
			t.Errorf("no name in 1000 was given face %d (%s)", i, Get(i).Name)
		}
		if n > 3*share {
			t.Errorf("face %d was given to %d names, over three times its share of %d", i, n, share)
		}
	}
}

// Names that differ in one letter are not neighbours in the catalog.
func TestForSpreadsSimilarNames(t *testing.T) {
	near := 0
	for i := 0; i < 200; i++ {
		a, b := IndexFor(fmt.Sprintf("user-%d", i)), IndexFor(fmt.Sprintf("user-%d", i+1))
		if d := a - b; d == 0 || d == 1 || d == -1 {
			near++
		}
	}
	if near > 40 {
		t.Errorf("%d of 200 consecutive names landed on the same or a neighbouring face", near)
	}
}

// A Model set to a name's index plays that name's face.
func TestModelPlaysTheFaceForAName(t *testing.T) {
	m := New()
	m.Set(IndexFor("linus"))
	if m.Face().Name != For("linus").Name {
		t.Errorf("the Model plays %s, want %s", m.Face().Name, For("linus").Name)
	}
}
