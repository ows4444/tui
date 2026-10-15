// Package clusterwidth holds the process-wide default for grapheme-cluster
// width: whether package ansi's Width, Truncate and TrimLeftWidth, and a zero
// ansi.Measurer, treat an extended grapheme cluster as one unit.
//
// The default is on unless the TUI_NO_CLUSTERS environment variable is set to
// a non-empty value the first time a width is measured. There is no exported
// way to change it after that: a program that must measure differently for one
// terminal carries an ansi.Measurer instead. This package is internal so that
// only the module's own tests can flip it, to check that nothing else does.
package clusterwidth

import (
	"os"
	"sync/atomic"
)

// Setting is an atomic bool whose default is resolved from the environment on
// first use. The zero value is unset.
type Setting struct {
	v atomic.Int32 // unset until decided
}

const (
	unset int32 = iota
	on
	off
)

// Default is the process-wide setting.
var Default Setting

// Load reports whether cluster handling is on.
func (c *Setting) Load() bool {
	switch c.v.Load() {
	case on:
		return true
	case off:
		return false
	}
	def := on
	if os.Getenv("TUI_NO_CLUSTERS") != "" {
		def = off
	}
	// An explicit Store that raced this read wins.
	c.v.CompareAndSwap(unset, def)
	return c.v.Load() == on
}

// Store sets cluster handling, overriding the environment.
func (c *Setting) Store(enabled bool) {
	if enabled {
		c.v.Store(on)
	} else {
		c.v.Store(off)
	}
}
