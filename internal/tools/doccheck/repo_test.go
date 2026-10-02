package main

import (
	"fmt"
	"os"
	"testing"
)

// The docs and README checks read the repository by root-relative path, so
// the tests run from the module root.
func TestMain(m *testing.M) {
	if err := os.Chdir("../../.."); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}
