package filepicker_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ows4444/tui/filepicker"
)

// A filepicker lists a directory's entries.
func Example() {
	dir, err := os.MkdirTemp("", "filepicker")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			fmt.Println(err)
			return
		}
	}
	m := filepicker.New(dir)
	for _, e := range m.Entries() {
		fmt.Println(e.Name)
	}
	// Output:
	// a.txt
	// b.txt
}

// Extensions and DirsOnly take effect at the next listing, so set them and
// call Reload.
func ExampleModel_Reload() {
	dir, err := os.MkdirTemp("", "filepicker")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)
	for _, name := range []string{"main.go", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o600); err != nil {
			fmt.Println(err)
			return
		}
	}
	m := filepicker.New(dir)
	m.Extensions = []string{".go"}
	m = m.Reload()
	for _, e := range m.Entries() {
		fmt.Println(e.Name)
	}
	// Output:
	// main.go
}
