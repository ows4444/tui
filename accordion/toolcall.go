package accordion

import (
	"github.com/ows4444/tui/ansi"
)

// ToolCallStatus is the lifecycle state of a tool invocation shown by
// ToolCall.
type ToolCallStatus int

// ToolCall status values.
const (
	ToolCallPending ToolCallStatus = iota
	ToolCallRunning
	ToolCallSuccess
	ToolCallError
)

// statusMarker returns the plain-text status indicator for a status. It is
// deliberately plain text with no embedded ANSI codes: Model's
// View wraps a cursor row's whole Title in its own cursorStyle.Render(title)
// call, and ansi.Style.Render always appends a trailing SGR reset. Any
// separately-Render-ed sub-style embedded in Title would fire that reset
// partway through the title, clobbering the cursor highlight for the rest
// of the string (the bug round 11 found for picker.Item/color-picker).
// Encoding status as plain text instead means accordion's own styling is
// the only styling ever applied to the title.
func statusMarker(status ToolCallStatus) string {
	switch status {
	case ToolCallPending:
		return "[pending]"
	case ToolCallRunning:
		return "[running]"
	case ToolCallSuccess:
		return "[success]"
	case ToolCallError:
		return "[error]"
	default:
		return "[unknown]"
	}
}

// ToolCall builds an Section for a single tool invocation, rather
// than a new Model type: like ThinkingSection, Model with one
// Section already covers "title always shown, content shown on toggle".
//
// Title is name plus a plain-text status marker (e.g. "read_file
// [running]"). Content is the formatted args and result, e.g.
// "Args: <args>\nResult: <result>". Empty name, args, or result are all
// valid and never cause a panic.
//
// name, args and result are untrusted (tool and model output): tabs are
// expanded and every escape sequence and control character except SGR
// styling is removed. Use ToolCallRaw to render them unchanged.
func ToolCall(name, args string, status ToolCallStatus, result string) Section {
	clean := func(s string) string { return ansi.SanitizeKeepSGR(ansi.ExpandTabs(s)) }
	return ToolCallRaw(clean(name), clean(args), status, clean(result))
}

// ToolCallRaw is ToolCall without sanitising: name, args and result are
// used unchanged. Only use it for trusted text.
func ToolCallRaw(name, args string, status ToolCallStatus, result string) Section {
	title := name + " " + statusMarker(status)

	content := "Args: " + args + "\nResult: " + result

	return Section{Title: title, Content: content}
}
