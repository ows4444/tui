// Package a11y holds small helpers shared by the widgets' Linearize methods.
package a11y

import (
	"strings"
	"unicode"
)

// PromptLabel turns a text input's prompt into words a screen reader can
// speak: "Name:  " -> "Name", "> " -> "" (nothing to say), "Full name
// (required): " -> "Full name (required)". It trims spaces and decorative
// punctuation from both ends but keeps brackets and quotes, so a prompt never
// comes back with an unbalanced parenthesis.
func PromptLabel(prompt string) string {
	keepStart := "([{\"'"
	keepEnd := ")]}\"'"
	s := strings.TrimLeftFunc(prompt, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(keepStart, r)
	})
	return strings.TrimRightFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune(keepEnd, r)
	})
}
