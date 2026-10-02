package ansi

import "strconv"

// KittyKeyboardEnableFlags pushes the given kitty keyboard protocol flag
// bitmask (1 disambiguate, 2 report event types, 4 alternate keys, 8 all
// keys as escape codes). KittyKeyboardEnableFlags(1) equals
// KittyKeyboardEnable; KittyKeyboardDisable pops it.
func KittyKeyboardEnableFlags(flags int) string {
	return CSI + ">" + strconv.Itoa(flags) + "u"
}
