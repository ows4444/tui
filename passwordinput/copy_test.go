package passwordinput

import "testing"

// A password field never puts its value on the clipboard.
func TestNewDisablesCopy(t *testing.T) {
	if !New().DisableCopy {
		t.Fatal("New().DisableCopy is false")
	}
}
