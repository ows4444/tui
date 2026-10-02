package input

// SetReportCSIReplies chooses how a CSI reply to a terminal query (DA1
// "ESC[?62;22c", DECRPM "ESC[?2026;1$y", a kitty keyboard flags answer
// "ESC[?0u", ...) is decoded. Off (the default) it is consumed as
// KeyUnknown. On it is a ReplyEvent with Kind '[' and Data the bytes between
// "ESC[" and the final byte inclusive (e.g. "?2026;1$y"). The capability
// probe turns it on; ordinary keys are unaffected either way.
func (rd *Reader) SetReportCSIReplies(on bool) { rd.reportCSI = on }
