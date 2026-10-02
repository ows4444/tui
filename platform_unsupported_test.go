package tui

import (
	"os/exec"
	"strings"
	"testing"
)

// TestUnsupportedGOOSFailsClearly builds the root package for GOOS values
// the module does not support (illumos and solaris) and expects the first
// line of compiler output to be the guard message naming the platform as
// unsupported, not an unrelated "undefined: term.X" error.
func TestUnsupportedGOOSFailsClearly(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-compilation is slow; skipped with -short")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH")
	}
	for _, goos := range []string{"illumos", "solaris"} {
		t.Run(goos, func(t *testing.T) {
			cmd := exec.Command(goBin, "build", ".")
			cmd.Env = append(cmd.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("GOOS=%s build unexpectedly succeeded", goos)
			}
			var first string
			for _, l := range strings.Split(string(out), "\n") {
				if strings.TrimSpace(l) != "" && !strings.HasPrefix(l, "#") {
					first = l
					break
				}
			}
			if !strings.Contains(first, "tui_unsupported_platform_this_GOOS_is_not_supported_see_README") {
				t.Fatalf("first error is not the guard message:\n%s", out)
			}
		})
	}
}
