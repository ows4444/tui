//go:build !(linux || darwin || dragonfly || freebsd || netbsd || openbsd || windows)

package tui

// This file only compiles on a GOOS the module does not support (solaris,
// illumos, aix, plan9, js, wasip1, ...). It fails the build with a message
// naming the platform as unsupported instead of the unrelated
// "undefined: term.GetSize" the missing terminal code gives.
// Supported: linux, darwin, dragonfly, freebsd, netbsd, openbsd, windows.
//
// The failure is a bad import, which the go command reports before any
// type-check error, so the guard is always the first line of output whatever
// the file is called.

import _ "tui_unsupported_platform_this_GOOS_is_not_supported_see_README"
