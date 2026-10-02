//covercheck:helper PTY harness for tests; no production logic

// Package ptytest opens pseudo-terminals for tests, using /dev/ptmx and the
// pty ioctls directly (no dependencies). It exists so tests in different
// packages — term, and the root tui package — can drive a real terminal
// without each carrying its own copy. It is internal: not public API, and
// only available on darwin and linux.
package ptytest
