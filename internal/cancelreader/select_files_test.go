package cancelreader

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// goFilesFor returns the Go files go list selects for this package on goos.
func goFilesFor(t *testing.T, goos string) []string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", `{{join .GoFiles " "}}`, ".")
	cmd.Env = append(cmd.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
	out, err := cmd.Output()
	if err != nil {
		t.Skipf("go list for %s unavailable: %v", goos, err)
	}
	return strings.Fields(string(out))
}

// Criterion #48: on FreeBSD, OpenBSD and NetBSD (and DragonFly, which has the
// same select) the build uses the select-based reader, not the read-deadline
// fallback.
func TestBSDBuildsUseTheSelectReader(t *testing.T) {
	for _, goos := range []string{"freebsd", "openbsd", "netbsd", "dragonfly", "darwin", "linux"} {
		files := goFilesFor(t, goos)
		if !slices.Contains(files, "cancelreader_unix.go") {
			t.Errorf("%s: cancelreader_unix.go (select) not selected: %v", goos, files)
		}
		if slices.Contains(files, "cancelreader_other.go") {
			t.Errorf("%s: the read-deadline fallback is still selected: %v", goos, files)
		}
		if !slices.Contains(files, "select_bsd.go") && goos != "linux" {
			t.Errorf("%s: select_bsd.go not selected: %v", goos, files)
		}
	}
	// Elsewhere the fallback remains.
	if files := goFilesFor(t, "solaris"); !slices.Contains(files, "cancelreader_other.go") || slices.Contains(files, "cancelreader_unix.go") {
		t.Errorf("solaris: want only the fallback, got %v", files)
	}
}
