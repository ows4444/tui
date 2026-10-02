//covercheck:helper terminal-emulator stand-in for tests; no production logic

// Package vtscreen is a small virtual terminal: it applies the escape
// sequences a Program writes to a grid of cells, so a test can assert on what
// a user would see. tuitest and the renderer tests build on it. It is
// internal: not public API, and it needs no pty, so it works on every GOOS.
package vtscreen
