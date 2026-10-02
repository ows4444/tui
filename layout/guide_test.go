package layout_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// The layout guide cites runnable examples by name; each one must exist, or the
// guide points at code that is not there.
func TestGuideCitesExamplesThatExist(t *testing.T) {
	guide, err := os.ReadFile("../docs/layout.md")
	if err != nil {
		t.Fatal(err)
	}
	// The examples live next to the code they show: mostly here, and the hit
	// example in package hittest.
	var src []byte
	for _, f := range []string{"example_test.go", "../hittest/layout_test.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src = append(src, b...)
	}
	cited := regexp.MustCompile("`(Example[A-Za-z0-9_]*)`").FindAllStringSubmatch(string(guide), -1)
	if len(cited) < 5 {
		t.Fatalf("the guide cites only %d examples; expected the pitfall examples", len(cited))
	}
	for _, m := range cited {
		if !strings.Contains(string(src), "func "+m[1]+"(") {
			t.Errorf("docs/layout.md cites %s, which no example file defines", m[1])
		}
	}
}
