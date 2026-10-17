package main

import (
	"bytes"
	"fmt"

	"github.com/ows4444/tui/tuitest"
)

// Example overwrites the first byte of a file with 0xff and saves it.
func Example() {
	st := memStore{"a.bin": {0x00, 0x01, 0x02}}
	m, _ := newModel(st, "a.bin")
	s := tuitest.New(m, 80, 24)
	defer s.Close()
	s.Keys("f", "f", "ctrl+s")
	s.WaitForText("Saved a.bin", 2e9)
	fmt.Println(bytes.Equal(st["a.bin"], []byte{0xff, 0x01, 0x02}))
	// Output: true
}
