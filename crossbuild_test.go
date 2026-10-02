package tui

import (
	"os/exec"
	"testing"
)

// TestBuildsOnSupportedPlatforms cross-compiles the whole module for every
// GOOS the library claims to support, so a build-tag mistake (like a file
// gated on darwin || linux that the root package depends on) fails here
// instead of on a user's machine. It only compiles; runtime behaviour on
// these platforms is not exercised.
func TestBuildsOnSupportedPlatforms(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-compilation is slow; skipped with -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	for _, goos := range []string{"linux", "darwin", "windows", "freebsd", "openbsd", "netbsd", "dragonfly"} {
		t.Run(goos, func(t *testing.T) {
			cmd := exec.Command(goBin, "build", "./...")
			cmd.Env = append(cmd.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("GOOS=%s go build ./... failed: %v\n%s", goos, err, out)
			}
		})
	}
}
