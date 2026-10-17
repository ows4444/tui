package tuislog_test

import (
	"fmt"
	"log/slog"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuislog"
)

// A Handler gives each record to a function as one styled line. Here the lines
// are printed with their styling removed; a program would pass
// tuislog.ToProgram(p, nil) to slog.New instead.
func Example() {
	h := tuislog.New(func(line string) { fmt.Println(ansi.StripANSI(line)) },
		&tuislog.Options{TimeFormat: "-"})
	log := slog.New(h)
	log.Info("listening", "port", 8080)
	log.Error("copy failed", "file", "a b.txt")
	// Output:
	// INFO  listening port=8080
	// ERROR copy failed file="a b.txt"
}
