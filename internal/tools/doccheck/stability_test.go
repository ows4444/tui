package main

import (
	"strings"
	"testing"
)

const rootDoc = `// Package tui is the root.
//
// Experimental: alpha, beta,
// gamma.
//
// Other text.
package tui
`

func TestStabilityAgreement(t *testing.T) {
	root := t.TempDir()
	write(t, root, "doc.go", rootDoc)
	write(t, root, "alpha/doc.go", "// Package alpha does a thing. Stability: experimental.\npackage alpha\n")
	write(t, root, "beta/a.go", "// Package beta does a thing. Stability: experimental.\npackage beta\n")
	write(t, root, "gamma/a.go", "// Package gamma does a thing.\npackage gamma\n")                          // listed, not marked
	write(t, root, "delta/a.go", "// Package delta does a thing. Stability: experimental.\npackage delta\n") // marked, not listed
	write(t, root, "stable/a.go", "// Package stable does a thing.\npackage stable\n")
	write(t, root, "internal/x/a.go", "// Package x. Stability: experimental.\npackage x\n")

	got, err := stability(root)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(got, "\n")
	if len(got) != 2 || !strings.Contains(joined, "gamma") || !strings.Contains(joined, "delta") {
		t.Errorf("want findings for gamma and delta only, got:\n%s", joined)
	}
}

func TestStabilityReportsMarkedButUnlisted(t *testing.T) {
	root := t.TempDir()
	write(t, root, "doc.go", "// Package tui is the root.\npackage tui\n")
	write(t, root, "a/a.go", "// Package a. Stability: experimental.\npackage a\n")
	got, err := stability(root)
	if err != nil || len(got) != 1 {
		t.Fatalf("a marked package with no root list must be reported, got %v, %v", got, err)
	}
}
