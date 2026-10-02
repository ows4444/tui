package tui

import (
	"log"
	"os"
)

// LogToFile redirects the standard library logger to the file at path
// (created if missing, appended to otherwise) and sets its prefix. Nothing is
// written to the terminal, so logging is safe while a Program owns the screen.
// The caller should Close the returned file when done.
func LogToFile(path, prefix string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304 -- the caller chooses the log file
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	log.SetPrefix(prefix)
	return f, nil
}
